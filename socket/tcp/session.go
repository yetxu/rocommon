package tcp

import (
	"net"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/yetxu/rocommon"
	"github.com/yetxu/rocommon/socket"
	"github.com/yetxu/rocommon/util"
)

type SessionIdentify struct {
	id uint64
}

func (a *SessionIdentify) ID() uint64 {
	return a.id
}

func (a *SessionIdentify) SetID(id uint64) {
	a.id = id
}

// Session interface def.go
type tcpSession struct {
	sync.Mutex
	SessionIdentify         //添加到SessionManager中时会设置tcpSession的ID属性
	*socket.NetProcessorRPC //事件处理相关 procrpc.go
	socket.NetContextSet    //记录session绑定信息 nodeproperty.go
	node                    rocommon.ServerNode

	//net.Conn
	conn      net.Conn
	sendQueue chan interface{}

	exitWg      sync.WaitGroup
	endCallback func()
	closeInt    int64

	aesMutex      sync.RWMutex
	aesStr        []byte
	handCodeMutex sync.RWMutex
	handCodeStr   string

	sessionOpt     socket.NetTCPSocketOption
	optMutex       sync.RWMutex
	sessionOptFlag bool //是否启用无读无写超时处理

	sendQueueMaxLen int

	recvPingNum int
}

func (a *tcpSession) GetSessionOpt() interface{} {
	return &a.sessionOpt
}
func (a *tcpSession) GetSessionOptFlag() bool {
	a.optMutex.RLock()
	defer a.optMutex.RUnlock()

	return a.sessionOptFlag
}
func (a *tcpSession) SetSessionOptFlag(flag bool) {
	a.optMutex.Lock()
	defer a.optMutex.Unlock()

	a.sessionOptFlag = flag
}

func (a *tcpSession) setConn(c net.Conn) {
	a.Lock()
	defer a.Unlock()
	a.conn = c
}

func (a *tcpSession) GetConn() net.Conn {
	a.Lock()
	defer a.Unlock()
	return a.conn
}

func (a *tcpSession) Raw() interface{} {
	return a.GetConn()
}

func (a *tcpSession) Node() rocommon.ServerNode {
	return a.node
}

func (a *tcpSession) GetAES() *[]byte {
	a.aesMutex.RLock()
	defer a.aesMutex.RUnlock()
	return &a.aesStr
}

func (a *tcpSession) SetAES(aes string) {
	a.aesMutex.Lock()
	defer a.aesMutex.Unlock()
	a.aesStr = []byte(aes)
	//log.Println("SetAES:", aes)
}

func (a *tcpSession) GetHandCode() string {
	a.handCodeMutex.RLock()
	defer a.handCodeMutex.RUnlock()
	return a.handCodeStr
}

func (a *tcpSession) SetHandCode(code string) {
	a.handCodeMutex.Lock()
	defer a.handCodeMutex.Unlock()
	a.handCodeStr = code
	//log.Println("SetAES:", aes)
}

func (a *tcpSession) IncRecvPingNum(incNum int) {
	if incNum <= 0 {
		a.recvPingNum = incNum
	} else {
		a.recvPingNum += incNum
	}
}
func (a *tcpSession) RecvPingNum() int {
	return a.recvPingNum
}

var sendQueueMaxLen = 2000
var sendQueuePool = sync.Pool{
	New: func() interface{} {
		return make(chan interface{}, sendQueueMaxLen+1)
	},
}

func (a *tcpSession) Start() {
	atomic.StoreInt64(&a.closeInt, 0)

	//重置发送队列
	a.sendQueueMaxLen = sendQueueMaxLen
	if a.node.(rocommon.ServerNodeProperty).GetName() == "gate" {
		a.sendQueueMaxLen = 200
	}
	a.sendQueue = make(chan interface{}, a.sendQueueMaxLen+1)
	//a.sendQueue = make(chan interface{}, 32) //todo..暂时默认发送队列长度2000
	//a.sendQueue = make(chan interface{}, sendQueueMaxLen+1) //todo..暂时默认发送队列长度2000
	//a.sendQueue = sendQueuePool.Get().(chan interface{})

	a.exitWg.Add(2)
	//a.node tcpAcceptor
	a.node.(socket.SessionManager).Add(a) //添加到session管理器中
	if a.node.TypeOfName() == "tcpAcceptor" {

		//log.Println("sessionMagNum:", a.node.(socket.SessionManager).SessionNum())
	}
	go func() {
		a.exitWg.Wait()
		//结束操作处理
		close(a.sendQueue)
		//sendQueuePool.Put(a.sendQueue)

		a.node.(socket.SessionManager).Remove(a)
		if a.endCallback != nil {
			a.endCallback()
		}
		//debug.FreeOSMemory()
	}()

	go a.RunRecv()
	go a.RunSend()
}

func (a *tcpSession) Close() {
	//已经关闭
	if ok := atomic.SwapInt64(&a.closeInt, 1); ok != 0 {
		return
	}

	conn := a.GetConn()
	if conn != nil {
		conn.Close()
		//关闭读
		//conn.(*net.TCPConn).CloseRead()
	}
	//util.InfoF("close session")
}

func (a *tcpSession) Send(msg interface{}) {
	//已经关闭
	if atomic.LoadInt64(&a.closeInt) != 0 {
		util.ErrorF("SendLen-sendQueue closeInt connId=%v", a.id)
		return
	}

	//a.sendQueue <- msg

	sendLen := len(a.sendQueue)
	if sendLen < sendQueueMaxLen {
		a.sendQueue <- msg
		return
	}
	util.ErrorF("SendLen-sendQueue=%v addr=%v", sendLen, a.conn.LocalAddr())
}

// 服务器进程之前启用ping操作
func (a *tcpSession) HeartBeat(msg interface{}) {
	//已经关闭
	if atomic.LoadInt64(&a.closeInt) != 0 {
		return
	}

	go func() {
		tmpMsg := msg
		delayTimer := time.NewTimer(15 * time.Second)
		for {
			delayTimer.Reset(5 * time.Second)
			select {
			case <-delayTimer.C:
				if atomic.LoadInt64(&a.closeInt) != 0 {
					break
				}

				a.Send(tmpMsg)
				//util.InfoF("Send PingReq id=%v", a.ID())
			}
		}
	}()
}

func (a *tcpSession) RunRecv() {
	//util.DebugF("start RunRecv goroutine")
	defer func() {
		//打印奔溃信息
		//if err := recover(); err != nil {
		//	a.onError(err)
		//}
		//util.InfoF("Stack---:\n%s\n", string(debug.Stack()))
		//打印堆栈信息
		if err := recover(); err != nil {
			debug.PrintStack()
		}
	}()

	recvCount := 0
	for {
		msg, seqId, err := a.ReadMsg(a) //procrpc.go
		if err != nil {
			util.ErrorF("Readmsg-RunRecv error=%v sessionId=%v", err, a.ID())

			//这边需要加锁，避免主线程继续在closInt还未设置成断开时还继续往session写数据，导致多线程冲突
			//a.Lock()
			//做关闭处理，发送数据时已经无法进行发送
			atomic.StoreInt64(&a.closeInt, 1)
			//close(a.sendQueue) //用来退出写协程
			a.sendQueue <- nil //用来退出写协程
			//a.Unlock()

			//抛出错误事件
			a.ProcEvent(&rocommon.RecvMsgEvent{Sess: a, Message: &rocommon.SessionClosed{}, Err: err})

			//todo...或者通过关闭sendQueue来实现关闭
			break
		}
		//接收数据事件放到队列中(需要放到队列中，否则会有线程冲突)
		a.ProcEvent(&rocommon.RecvMsgEvent{Sess: a, Message: msg, Err: nil, MsgSeqId: seqId, KvTime: util.GetTimeMilliseconds()})
		//a.ProcEvent(&rocommon.RecvMsgEvent{Sess: a, Message: msg, Err: nil, MsgSeqId: seqId})

		recvCount++
		if recvCount >= 100 {
			recvCount = 0
			//util.InfoF("RunRecv recCount sessionId=%v", a.ID())
		}
	}

	util.DebugF("exit RunRecv goroutine addr=%v remoteAddr=%v id=%v", a.conn.LocalAddr(), a.conn.RemoteAddr(), a.id)
	a.exitWg.Done()
}

func (a *tcpSession) RunSend() {
	//util.DebugF("start RunSend goroutine")
	defer func() {
		//打印奔溃信息
		//if err := recover(); err != nil {
		//	a.onError(err)
		//}
		//util.InfoF("Stack---:\n%s\n", string(debug.Stack()))
		//打印堆栈信息
		if err := recover(); err != nil {
			debug.PrintStack()
		}
	}()

	sendCount := 0
	//放到另外的队列中c
	for data := range a.sendQueue {
		if data == nil {
			break
		}
		err := a.SendMsg(&rocommon.SendMsgEvent{Sess: a, Message: data})
		//err := a.SendMsg(a, data) //procrpc.go
		if err != nil {
			util.ErrorF("SendMsg RunSend error %v", err)
			break
		}
		sendCount++
		if sendCount >= 100 {
			sendCount = 0
			//util.InfoF("RunSend sendCount sessionId=%v", a.ID())
		}
	}

	util.DebugF("exit RunSend goroutine addr=%v remoteAddr=%v id=%v", a.conn.LocalAddr(), a.conn.RemoteAddr(), a.id)
	c := a.GetConn()
	if c != nil {
		c.Close()
	}

	a.exitWg.Done()
}

// /////////////////////
// acceptor中获取到连接后创建session使用
func newTcpSession(c net.Conn, node rocommon.ServerNode, endCallback func()) *tcpSession {
	session := &tcpSession{
		conn:        c,
		node:        node,
		endCallback: endCallback,
		NetProcessorRPC: node.(interface {
			GetRPC() *socket.NetProcessorRPC
		}).GetRPC(), //使用外层node的RPC处理接口
	}
	node.(socket.SocketOption).CopyOpt(&session.sessionOpt)

	return session
}
