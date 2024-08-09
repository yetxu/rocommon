package rocommon

// 连接session
type Session interface {
	//获得net.Conn
	Raw() interface{}

	//todo...
	Node() ServerNode //tcpAcceptor / tcpConnector

	Send(msg interface{})
	Close()
	//表示ID
	ID() uint64

	//aes密码(加密解密)
	GetAES() *[]byte
	SetAES(aes string)
	GetHandCode() string
	SetHandCode(code string)
	GetSessionOpt() interface{}
	GetSessionOptFlag() bool
	SetSessionOptFlag(flag bool)
	HeartBeat(msg interface{})
	IncRecvPingNum(incNum int)
	RecvPingNum() int
}

// 事件处理队列
type NetEventQueue interface {
	StartQueue() NetEventQueue

	StopQueue() NetEventQueue

	Wait()

	PostCb(callback func())

	AttachUpdateModule(update UpdateModule)
}

// 处理主逻辑的更新操作
type UpdateModule interface {
	//传入的时间为毫秒
	Update(ms uint64)
	Init()
}
type UpdateLogic interface {
	//传入的时间为毫秒
	Update(ms uint64)
}

// event相关
type ProcEvent interface {
	Session() Session //会话信息
	Msg() interface{} //消息
	SeqId() uint32    //消息序列号
	KVTime() uint64   //接受到消息时的时间
}

// 输入返回输出
type EventHook interface {
	InEvent(in ProcEvent) ProcEvent   //获得接收事件
	OutEvent(out ProcEvent) ProcEvent //获得发送事件
}
type EventCallBack func(e ProcEvent)

// 消息处理
type MessageProcessor interface {
	//recv
	OnRecvMsg(s Session) (interface{}, uint32, error)
	//send
	OnSendMsg(s Session, msg interface{}) error
}

// /////////////////////////////////
// recv send event -> ProcEvent
type RecvMsgEvent struct {
	Sess     Session
	Message  interface{}
	Err      error
	MsgSeqId uint32
	KvTime   uint64
}

func (a *RecvMsgEvent) Session() Session {
	return a.Sess
}
func (a *RecvMsgEvent) Msg() interface{} {
	return a.Message
}
func (a *RecvMsgEvent) SeqId() uint32 {
	return a.MsgSeqId
}
func (a *RecvMsgEvent) KVTime() uint64 {
	return a.KvTime
}

// 接收到消息处理后并回复消息(如果需要回复调用该接口)
func (a *RecvMsgEvent) Replay(msg interface{}) error {
	a.Sess.Send(msg)
	return nil
}

type SendMsgEvent struct {
	Sess    Session
	Message interface{}
}

func (a *SendMsgEvent) Session() Session {
	return a.Sess
}
func (a *SendMsgEvent) Msg() interface{} {
	return a.Message
}
func (a *SendMsgEvent) SeqId() uint32 {
	return 0
}
func (a *SendMsgEvent) KVTime() uint64 {
	return 0
}

type ReplayEvent interface {
	Replay(msg interface{}) error
}

// 直接发送数据，例如game，发给gate，然后gate直接发送
type TransmitPacket struct {
	MsgData []byte
	MsgId   uint32
	SeqId   uint32
}

// /////////////////////////////////
// http处理
type HTTPRequest struct {
	ReqMsg       interface{} //request
	ResMsg       interface{} //response
	ReqCodecName string      //默认为json
}
