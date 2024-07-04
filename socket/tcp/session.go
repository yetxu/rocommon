package tcp

import (
	"net"
	"rocommon"
	"rocommon/socket"
	"rocommon/util"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

type SessionIdentify struct {
	id uint64
}

func (this *SessionIdentify) ID() uint64 {
	return this.id
}

func (this *SessionIdentify) SetID(id uint64) {
	this.id = id
}

//Session interface def.go
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

func (this *tcpSession) GetSessionOpt() interface{} {
	return &this.sessionOpt
}
func (this *tcpSession) GetSessionOptFlag() bool {
	this.optMutex.RLock()
	defer this.optMutex.RUnlock()

	return this.sessionOptFlag
}
func (this *tcpSession) SetSessionOptFlag(flag bool) {
	this.optMutex.Lock()
	defer this.optMutex.Unlock()

	this.sessionOptFlag = flag
}

func (this *tcpSession) setConn(c net.Conn) {
	this.Lock()
	defer this.Unlock()
	this.conn = c
}

func (this *tcpSession) GetConn() net.Conn {
	this.Lock()
	defer this.Unlock()
	return this.conn
}

func (this *tcpSession) Raw() interface{} {
	return this.GetConn()
}

func (this *tcpSession) Node() rocommon.ServerNode {
	return this.node
}

func (this *tcpSession) GetAES() *[]byte {
	this.aesMutex.RLock()
	defer this.aesMutex.RUnlock()
	return &this.aesStr
}

func (this *tcpSession) SetAES(aes string) {
	this.aesMutex.Lock()
	defer this.aesMutex.Unlock()
	this.aesStr = []byte(aes)
	//log.Println("SetAES:", aes)
}

func (this *tcpSession) GetHandCode() string {
	this.handCodeMutex.RLock()
	defer this.handCodeMutex.RUnlock()
	return this.handCodeStr
}

func (this *tcpSession) SetHandCode(code string) {
	this.handCodeMutex.Lock()
	defer this.handCodeMutex.Unlock()
	this.handCodeStr = code
	//log.Println("SetAES:", aes)
}

func (this *tcpSession) IncRecvPingNum(incNum int) {
	if incNum <= 0 {
		this.recvPingNum = incNum
	} else {
		this.recvPingNum += incNum
	}
}
func (this *tcpSession) RecvPingNum() int {
	return this.recvPingNum
}

var sendQueueMaxLen = 2000
var sendQueuePool = sync.Pool{
	New: func() interface{} {
		return make(chan interface{}, sendQueueMaxLen+1)
	},
}

func (this *tcpSession) Start() {
	atomic.StoreInt64(&this.closeInt, 0)

	//重置发送队列
	this.sendQueueMaxLen = sendQueueMaxLen
	if this.node.(rocommon.ServerNodeProperty).GetName() == "gate" {
		this.sendQueueMaxLen = 200
	}
	this.sendQueue = make(chan interface{}, this.sendQueueMaxLen+1)
	//this.sendQueue = make(chan interface{}, 32) //todo..暂时默认发送队列长度2000
	//this.sendQueue = make(chan interface{}, sendQueueMaxLen+1) //todo..暂时默认发送队列长度2000
	//this.sendQueue = sendQueuePool.Get().(chan interface{})

	this.exitWg.Add(2)
	//this.node tcpAcceptor
	this.node.(socket.SessionManager).Add(this) //添加到session管理器中
	if this.node.TypeOfName() == "tcpAcceptor" {

		//log.Println("sessionMagNum:", this.node.(socket.SessionManager).SessionNum())
	}
	go func() {
		this.exitWg.Wait()
		//结束操作处理
		close(this.sendQueue)
		//sendQueuePool.Put(this.sendQueue)

		this.node.(socket.SessionManager).Remove(this)
		if this.endCallback != nil {
			this.endCallback()
		}
		//debug.FreeOSMemory()
	}()

	go this.RunRecv()
	go this.RunSend()
}

func (this *tcpSession) Close() {
	//已经关闭
	if ok := atomic.SwapInt64(&this.closeInt, 1); ok != 0 {
		return
	}

	conn := this.GetConn()
	if conn != nil {
		conn.Close()
		//关闭读
		//conn.(*net.TCPConn).CloseRead()
	}
	//util.InfoF("close session")
}

func (this *tcpSession) Send(msg interface{}) {
	//已经关闭
	if atomic.LoadInt64(&this.closeInt) != 0 {
		util.ErrorF("SendLen-sendQueue closeInt connId=%v", this.id)
		return
	}

	//this.sendQueue <- msg

	sendLen := len(this.sendQueue)
	if sendLen < sendQueueMaxLen {
		this.sendQueue <- msg
		return
	}
	util.ErrorF("SendLen-sendQueue=%v addr=%v", sendLen, this.conn.LocalAddr())
}

//服务器进程之前启用ping操作
func (this *tcpSession) HeartBeat(msg interface{}) {
	//已经关闭
	if atomic.LoadInt64(&this.closeInt) != 0 {
		return
	}

	go func() {
		tmpMsg := msg
		delayTimer := time.NewTimer(15 * time.Second)
		for {
			delayTimer.Reset(5 * time.Second)
			select {
			case <-delayTimer.C:
				if atomic.LoadInt64(&this.closeInt) != 0 {
					break
				}

				this.Send(tmpMsg)
				//util.InfoF("Send PingReq id=%v", this.ID())
			}
		}
	}()
}

func (this *tcpSession) RunRecv() {
	//util.DebugF("start RunRecv goroutine")
	defer func() {
		//打印奔溃信息
		//if err := recover(); err != nil {
		//	this.onError(err)
		//}
		//util.InfoF("Stack---:\n%s\n", string(debug.Stack()))
		//打印堆栈信息
		if err := recover(); err != nil {
			debug.PrintStack()
		}
	}()

	recvCount := 0
	for {
		msg, seqId, err := this.ReadMsg(this) //procrpc.go
		if err != nil {
			util.ErrorF("Readmsg-RunRecv error=%v sessionId=%v", err, this.ID())

			//这边需要加锁，避免主线程继续在closInt还未设置成断开时还继续往session写数据，导致多线程冲突
			//this.Lock()
			//做关闭处理，发送数据时已经无法进行发送
			atomic.StoreInt64(&this.closeInt, 1)
			//close(this.sendQueue) //用来退出写协程
			this.sendQueue <- nil //用来退出写协程
			//this.Unlock()

			//抛出错误事件
			this.ProcEvent(&rocommon.RecvMsgEvent{Sess: this, Message: &rocommon.SessionClosed{}, Err: err})

			//todo...或者通过关闭sendQueue来实现关闭
			break
		}
		//接收数据事件放到队列中(需要放到队列中，否则会有线程冲突)
		this.ProcEvent(&rocommon.RecvMsgEvent{Sess: this, Message: msg, Err: nil, MsgSeqId: seqId, KvTime: util.GetTimeMilliseconds()})
		//this.ProcEvent(&rocommon.RecvMsgEvent{Sess: this, Message: msg, Err: nil, MsgSeqId: seqId})

		recvCount++
		if recvCount >= 100 {
			recvCount = 0
			//util.InfoF("RunRecv recCount sessionId=%v", this.ID())
		}
	}

	util.DebugF("exit RunRecv goroutine addr=%v remoteAddr=%v id=%v", this.conn.LocalAddr(), this.conn.RemoteAddr(), this.id)
	this.exitWg.Done()
}

func (this *tcpSession) RunSend() {
	//util.DebugF("start RunSend goroutine")
	defer func() {
		//打印奔溃信息
		//if err := recover(); err != nil {
		//	this.onError(err)
		//}
		//util.InfoF("Stack---:\n%s\n", string(debug.Stack()))
		//打印堆栈信息
		if err := recover(); err != nil {
			debug.PrintStack()
		}
	}()

	sendCount := 0
	//放到另外的队列中c
	for data := range this.sendQueue {
		if data == nil {
			break
		}
		err := this.SendMsg(&rocommon.SendMsgEvent{Sess: this, Message: data})
		//err := this.SendMsg(this, data) //procrpc.go
		if err != nil {
			util.ErrorF("SendMsg RunSend error %v", err)
			break
		}
		sendCount++
		if sendCount >= 100 {
			sendCount = 0
			//util.InfoF("RunSend sendCount sessionId=%v", this.ID())
		}
	}

	util.DebugF("exit RunSend goroutine addr=%v remoteAddr=%v id=%v", this.conn.LocalAddr(), this.conn.RemoteAddr(), this.id)
	c := this.GetConn()
	if c != nil {
		c.Close()
	}

	this.exitWg.Done()
}

///////////////////////
//acceptor中获取到连接后创建session使用
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
