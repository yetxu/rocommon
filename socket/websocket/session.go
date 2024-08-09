package websocket

import (
	"runtime/debug"
	"sync"
	"sync/atomic"

	"github.com/yetxu/rocommon"
	"github.com/yetxu/rocommon/socket"
	tcpBase "github.com/yetxu/rocommon/socket/tcp"
	"github.com/yetxu/rocommon/util"

	"github.com/gorilla/websocket"
)

// Session interface def.go
type wsSession struct {
	sync.Mutex
	tcpBase.SessionIdentify //添加到SessionManager中时会设置tcpSession的ID属性
	*socket.NetProcessorRPC //事件处理相关 procrpc.go
	socket.NetContextSet    //记录session绑定信息 nodeproperty.go
	node                    rocommon.ServerNode

	//net.Conn
	conn      *websocket.Conn
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
}

func (a *wsSession) GetSessionOpt() interface{} {
	return &a.sessionOpt
}
func (a *wsSession) GetSessionOptFlag() bool {
	a.optMutex.RLock()
	defer a.optMutex.RUnlock()

	return a.sessionOptFlag
}
func (a *wsSession) SetSessionOptFlag(flag bool) {
	a.optMutex.Lock()
	defer a.optMutex.Unlock()

	a.sessionOptFlag = flag
}

func (a *wsSession) setConn(c *websocket.Conn) {
	a.Lock()
	defer a.Unlock()
	a.conn = c
}

func (a *wsSession) GetConn() *websocket.Conn {
	a.Lock()
	defer a.Unlock()
	return a.conn
}

func (a *wsSession) Raw() interface{} {
	return a.GetConn()
}

func (a *wsSession) Node() rocommon.ServerNode {
	return a.node
}

func (a *wsSession) GetAES() *[]byte {
	a.aesMutex.RLock()
	defer a.aesMutex.RUnlock()
	return &a.aesStr
}

func (a *wsSession) SetAES(aes string) {
	a.aesMutex.Lock()
	defer a.aesMutex.Unlock()
	a.aesStr = []byte(aes)
	//log.Println("SetAES:", aes)
}

func (a *wsSession) GetHandCode() string {
	a.handCodeMutex.RLock()
	defer a.handCodeMutex.RUnlock()
	return a.handCodeStr
}

func (a *wsSession) SetHandCode(code string) {
	a.handCodeMutex.Lock()
	defer a.handCodeMutex.Unlock()
	a.handCodeStr = code
	//log.Println("SetAES:", aes)
}
func (a *wsSession) IncRecvPingNum(incNum int) {
}
func (a *wsSession) RecvPingNum() int {
	return 0
}

var sendQueueMaxLen = 2000
var sendQueuePool = sync.Pool{
	New: func() interface{} {
		return make(chan interface{}, sendQueueMaxLen+1)
	},
}

func (a *wsSession) Start() {
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
	if a.node.TypeOfName() == "wsAcceptor" {

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

func (a *wsSession) Close() {
	//已经关闭
	if ok := atomic.SwapInt64(&a.closeInt, 1); ok != 0 {
		return
	}

	conn := a.GetConn()
	if conn != nil {
		//conn.Close()
		//关闭读
		conn.Close()
		conn.CloseHandler()
	}
	//util.InfoF("close session")
}

func (a *wsSession) Send(msg interface{}) {
	//已经关闭
	if atomic.LoadInt64(&a.closeInt) != 0 {
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
func (a *wsSession) HeartBeat(msg interface{}) {
	//已经关闭
	if atomic.LoadInt64(&a.closeInt) != 0 {
		return
	}
}

func (a *wsSession) RunRecv() {
	// util.DebugF("start RunRecv goroutine")
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

	for {
		msg, seqId, err := a.ReadMsg(a) //procrpc.go
		if err != nil {
			util.ErrorF("Readmsg-RunRecv error=%v", err)

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
	}

	util.DebugF("exit RunRecv goroutine addr=%v", a.conn.LocalAddr())
	a.exitWg.Done()
}

func (a *wsSession) RunSend() {
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

	//放到另外的队列中
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
	}

	util.DebugF("exit RunSend goroutine addr=%v", a.conn.LocalAddr())
	c := a.GetConn()
	if c != nil {
		c.Close()
	}

	a.exitWg.Done()
}

///////////////////////
//acceptor中获取到连接后创建session使用

func newWebSocketSession(conn *websocket.Conn, node rocommon.ServerNode, endCallback func()) *wsSession {
	session := &wsSession{
		conn:        conn,
		node:        node,
		endCallback: endCallback,
		NetProcessorRPC: node.(interface {
			GetRPC() *socket.NetProcessorRPC
		}).GetRPC(), //使用外层node的RPC处理接口
	}
	node.(socket.SocketOption).CopyOpt(&session.sessionOpt)

	return session
}
