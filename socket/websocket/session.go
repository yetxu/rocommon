package websocket

import (
	"rocommon"
	"rocommon/socket"
	tcpBase "rocommon/socket/tcp"
	"rocommon/util"
	"runtime/debug"
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"
)

//Session interface def.go
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

func (this *wsSession) GetSessionOpt() interface{} {
	return &this.sessionOpt
}
func (this *wsSession) GetSessionOptFlag() bool {
	this.optMutex.RLock()
	defer this.optMutex.RUnlock()

	return this.sessionOptFlag
}
func (this *wsSession) SetSessionOptFlag(flag bool) {
	this.optMutex.Lock()
	defer this.optMutex.Unlock()

	this.sessionOptFlag = flag
}

func (this *wsSession) setConn(c *websocket.Conn) {
	this.Lock()
	defer this.Unlock()
	this.conn = c
}

func (this *wsSession) GetConn() *websocket.Conn {
	this.Lock()
	defer this.Unlock()
	return this.conn
}

func (this *wsSession) Raw() interface{} {
	return this.GetConn()
}

func (this *wsSession) Node() rocommon.ServerNode {
	return this.node
}

func (this *wsSession) GetAES() *[]byte {
	this.aesMutex.RLock()
	defer this.aesMutex.RUnlock()
	return &this.aesStr
}

func (this *wsSession) SetAES(aes string) {
	this.aesMutex.Lock()
	defer this.aesMutex.Unlock()
	this.aesStr = []byte(aes)
	//log.Println("SetAES:", aes)
}

func (this *wsSession) GetHandCode() string {
	this.handCodeMutex.RLock()
	defer this.handCodeMutex.RUnlock()
	return this.handCodeStr
}

func (this *wsSession) SetHandCode(code string) {
	this.handCodeMutex.Lock()
	defer this.handCodeMutex.Unlock()
	this.handCodeStr = code
	//log.Println("SetAES:", aes)
}
func (this *wsSession) IncRecvPingNum(incNum int) {
}
func (this *wsSession) RecvPingNum() int {
	return 0
}

var sendQueueMaxLen = 2000
var sendQueuePool = sync.Pool{
	New: func() interface{} {
		return make(chan interface{}, sendQueueMaxLen+1)
	},
}

func (this *wsSession) Start() {
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
	if this.node.TypeOfName() == "wsAcceptor" {

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

func (this *wsSession) Close() {
	//已经关闭
	if ok := atomic.SwapInt64(&this.closeInt, 1); ok != 0 {
		return
	}

	conn := this.GetConn()
	if conn != nil {
		//conn.Close()
		//关闭读
		conn.Close()
		conn.CloseHandler()
	}
	//util.InfoF("close session")
}

func (this *wsSession) Send(msg interface{}) {
	//已经关闭
	if atomic.LoadInt64(&this.closeInt) != 0 {
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
func (this *wsSession) HeartBeat(msg interface{}) {
	//已经关闭
	if atomic.LoadInt64(&this.closeInt) != 0 {
		return
	}
}

func (this *wsSession) RunRecv() {
	// util.DebugF("start RunRecv goroutine")
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

	for {
		msg, seqId, err := this.ReadMsg(this) //procrpc.go
		if err != nil {
			util.ErrorF("Readmsg-RunRecv error=%v", err)

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
	}

	util.DebugF("exit RunRecv goroutine addr=%v", this.conn.LocalAddr())
	this.exitWg.Done()
}

func (this *wsSession) RunSend() {
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

	//放到另外的队列中
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
	}

	util.DebugF("exit RunSend goroutine addr=%v", this.conn.LocalAddr())
	c := this.GetConn()
	if c != nil {
		c.Close()
	}

	this.exitWg.Done()
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
