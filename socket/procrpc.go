package socket

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/yetxu/rocommon"
	"github.com/yetxu/rocommon/rpc"
	"github.com/yetxu/rocommon/util"

	"github.com/gorilla/websocket"
)

type (
	NetProcessorRPC struct {
		//解析消息数据,发送消息数据处理
		MsgRPC rocommon.MessageProcessor
		//def.go 消息解析操作放到队列直接的过滤操作(已经序列化为protobuf消息结构，如果在转换之前就做处理的，可以在MsgRPC中直接处理
		Hooker rocommon.EventHook
		//def.go  注册的具体函数回掉（具体的逻辑实现方法，例如:pbbind_gen.go中的gateHandler），没有回调函数时设置为nil
		Callback rocommon.EventCallBack
	}
)
type ProcessorRPCBinder func(b rocommon.ProcessorRPCBundle, usercb rocommon.EventCallBack, args ...interface{})

var (
	//当前执行的进程名称，和回调相关的函数操作
	procRPCByName = map[string]ProcessorRPCBinder{}
)

func RegisterProcessRPC(procName string, f ProcessorRPCBinder) {
	if _, ok := procRPCByName[procName]; ok {
		panic("procRPC has register:" + procName)
	}
	procRPCByName[procName] = f
}

func SetProcessorRPC(node rocommon.ServerNode, procName string, callback rocommon.EventCallBack, args ...interface{}) {
	if proc, ok := procRPCByName[procName]; ok {
		b := node.(rocommon.ProcessorRPCBundle)
		proc(b, callback, args)
	} else {
		panic("procRPC not register:" + procName)
	}
}

// 加入回调队列或者直接执行回调操作
func QueueEventCall(cb rocommon.EventCallBack) rocommon.EventCallBack {
	return func(e rocommon.ProcEvent) {
		if cb != nil {
			SessionQueueCall(e.Session(), func() {
				//now1 := time.Now()
				cb(e)
				//deltaT := time.Now().Sub(now1)
				//if deltaT > 1*time.Millisecond {
				//	if e.Msg() != nil && reflect.TypeOf(e.Msg()) != nil {
				//		tmpMsg := reflect.TypeOf(e.Msg()).Elem().String()
				//		util.DebugF("t=%v profile=%v", deltaT, tmpMsg)
				//	}
				//}
			})
		}
	}
}

// 在会话上执行事件回调，有队列则加入队列，没有就直接执行回调
func SessionQueueCall(s rocommon.Session, cb func()) {
	if s == nil {
		return
	}
	que := s.Node().(interface{ Queue() rocommon.NetEventQueue }).Queue()
	if que != nil {
		que.PostCb(cb) //加入事件队列中
	} else {
		//todo...
		cb() //不存在直接执行回调函数(注意多线程冲突问题)
	}
}

// 注册和回掉函数相关操作
func init() {
	RegisterProcessRPC("tcp.pb",
		func(b rocommon.ProcessorRPCBundle, usercb rocommon.EventCallBack, arg ...interface{}) {
			b.SetTransmitter(new(TCPMessageProcessor))
			b.SetHooker(new(TCPEventHook))
			b.SetCallback(QueueEventCall(usercb))
		})
}

// ///////////////////////////////////////////
// NetProcessorRPC
func (a *NetProcessorRPC) GetRPC() *NetProcessorRPC {
	return a
}

// 收到消息后调用该函数入队列操作
func (a *NetProcessorRPC) ProcEvent(e rocommon.ProcEvent) {
	//todo... hooker callback
	if a.Hooker != nil {
		e = a.Hooker.InEvent(e) //对不同消息类型进行解析，并进行处理
	}

	if a.Callback != nil && e != nil {
		a.Callback(e)
	}
}

func (a *NetProcessorRPC) ReadMsg(session rocommon.Session) (interface{}, uint32, error) {
	if a.MsgRPC != nil {
		return a.MsgRPC.OnRecvMsg(session)
	}
	return nil, 0, errors.New("msgrpc not set!!!")
}

func (a *NetProcessorRPC) SendMsg(ev rocommon.ProcEvent) error {
	//执行hook
	if a.Hooker != nil {
		ev = a.Hooker.OutEvent(ev)
	}
	if a.MsgRPC != nil {
		return a.MsgRPC.OnSendMsg(ev.Session(), ev.Msg())
	}
	return nil
}

func (self *NetProcessorRPC) SetTransmitter(mp rocommon.MessageProcessor) {
	self.MsgRPC = mp
}

func (self *NetProcessorRPC) SetHooker(ev rocommon.EventHook) {
	self.Hooker = ev
}

func (self *NetProcessorRPC) SetCallback(ecb rocommon.EventCallBack) {
	self.Callback = ecb
}

// ///////////////////////////////////////////
// EventHook interface def.go
type TCPEventHook struct {
}

func (a *TCPEventHook) InEvent(e rocommon.ProcEvent) rocommon.ProcEvent {
	//todo... important
	//根据收到的消息类型进行过滤处理，例如如果是RecvMsgEvent事件，那么说明进过了protobuf解析，直接返回
	//例如远程过程调用的方式
	inEvent, handled, err := RPCResolveInEvent(e)
	if err != nil {
		util.InfoF("rpc ResolveInEvent err:%v", err)
		return nil
	}
	if !handled {
		//todo... delay resolve event
	}

	return inEvent
}

// 获得发送事件
func (a *TCPEventHook) OutEvent(out rocommon.ProcEvent) rocommon.ProcEvent {
	//todo...
	handled, err := RPCResloveOutEvent(out)
	if err != nil {
		util.InfoF("rpc RPCResolveOutEvent err:%v", err)
		return nil
	}

	if !handled {
		//todo... delay reslove event
	}
	return out
}

// multiHook 例如game server有多个处理操作
type MultiTCPEventHook []rocommon.EventHook

func (a MultiTCPEventHook) InEvent(in rocommon.ProcEvent) rocommon.ProcEvent {
	for _, ev := range a {
		in = ev.InEvent(in)
		if in == nil {
			break
		}
	}
	return in
}

// 获得发送事件
func (a MultiTCPEventHook) OutEvent(out rocommon.ProcEvent) rocommon.ProcEvent {
	for _, ev := range a {
		out = ev.OutEvent(out)
		if out == nil {
			break
		}
	}
	return out
}

func NewMultiTCPEventHook(args ...rocommon.EventHook) rocommon.EventHook {
	return MultiTCPEventHook(args)
}

// 根据收到的消息类型进行过滤处理，例如如果是RecvMsgEvent事件，那么说明经过了protobuf解析，直接返回
// 例如远程过程调用的方式 / RPC消息解析
func RPCResolveInEvent(inEvent rocommon.ProcEvent) (rocommon.ProcEvent, bool, error) {
	//是接收处理消息
	if _, ok := inEvent.(*rocommon.RecvMsgEvent); ok {
		return inEvent, false, nil
	}

	//todo...其他消息类型处理 important
	return inEvent, false, nil
}

func RPCResloveOutEvent(outEvent rocommon.ProcEvent) (bool, error) {
	//todo... RemoteCallMsg
	return true, nil
}

// ///////////////////////////////////////////
// MessageProcessor interface def.go
type TCPMessageProcessor struct {
}

// recv
func (a *TCPMessageProcessor) OnRecvMsg(s rocommon.Session) (msg interface{}, msgSeqId uint32, err error) {
	//todo...
	reader, ok := s.Raw().(io.Reader)
	if !ok || reader == nil {
		util.InfoF("[TCPMessageProcessor] OnRecvMsg err")
		return nil, 0, nil
	}

	opt := s.Node().(SocketOption)
	opt.SocketReadTimeout(reader.(net.Conn), func() {
		msg, msgSeqId, err = rpc.ReadMessage(reader, opt.MaxMsgLen(), s.GetAES())
	})
	return
}

// send
var tmpClient = []byte("client")

func (a *TCPMessageProcessor) OnSendMsg(s rocommon.Session, msg interface{}) (err error) {
	util.InfoF("[TCPMessageProcessor] OnSendMsg session=%v msg=%v", s, msg)
	//todo...
	writer, ok := s.Raw().(io.Writer)
	if !ok || writer == nil {
		util.InfoF("[TCPMessageProcessor] OnSendMsg err")
		return nil
	}

	opt := s.Node().(SocketOption)
	opt.SocketWriteTimeout(writer.(net.Conn), func() {
		nodeName := s.Node().(rocommon.ServerNodeProperty).GetName()
		if nodeName == "client" {
			err = rpc.SendMessage(writer, msg, s.GetAES(), opt.MaxMsgLen(), nodeName)
		} else {
			err = rpc.SendMessage(writer, msg, s.GetAES(), opt.MaxMsgLen(), nodeName)

		}
	})
	return
}

// ///////////////////////////////////////////
// MessageProcessor interface def.go
type WSMessageProcessor struct {
}

const (
	lenMaxLen  = 2 //包体大小2个字节 uint16
	msgIdLen   = 2 //包ID大小2个字节  uint16
	msgSeqlen  = 4 //发送序列号2个字节大小，用来断线重连
	msgFlaglen = 2 //暂定标记，加解密 1表示RSA，2表示AES
)

// recv
func (a *WSMessageProcessor) OnRecvMsg(s rocommon.Session) (msg interface{}, msgSeqId uint32, err error) {
	conn, ok := s.Raw().(*websocket.Conn)
	if !ok || conn == nil {
		util.InfoF("[WSMessageProcessor] OnRecvMsg err")
		return nil, 0, nil
	}

	//reader, ok := s.Raw().(io.Reader)
	//if !ok || reader == nil {
	//	util.InfoF("[TCPMessageProcessor] OnRecvMsg err")
	//	return nil, 0, nil
	//}

	messageType, raw, err := conn.ReadMessage()
	if err != nil {
		util.InfoF("[WSMessageProcessor] OnRecvMsg err=%v", err)
		return nil, 0, nil
	}
	if messageType != websocket.BinaryMessage {
		util.InfoF("[WSMessageProcessor] OnRecvMsg err messageType=%v", messageType)
		return nil, 0, nil
	}

	var msgId uint16
	//var seqId uint32  //包序列号，客户端发送时的序列从1开始
	var flagId uint16 //加密方式
	var msgData []byte

	binary.BigEndian.Uint16(raw) //msgDataLen
	msgId = binary.BigEndian.Uint16(raw[lenMaxLen:])
	msgSeqId = binary.BigEndian.Uint32(raw[lenMaxLen+msgIdLen:])
	flagId = binary.BigEndian.Uint16(raw[lenMaxLen+msgIdLen+msgSeqlen:])
	msgData = raw[msgIdLen+msgSeqlen+msgFlaglen+lenMaxLen:]

	aesKey := s.GetAES()
	switch flagId {
	case 1:
		if int(msgId) == rpc.SC_HAND_SHAKE_NTFMsgId { //SC_HAND_SHAKE_NTF
			msgData, err = rpc.RSADecrypt(msgData, rpc.PrivateClientKey)
			if err != nil {
				return nil, 0, err
			}
		} else if int(msgId) == rpc.CS_HAND_SHAKE_REQMsgId { //CS_HAND_SHAKE_REQ
			msgData, err = rpc.RSADecrypt(msgData, rpc.PrivateServerKey)
			if err != nil {
				return nil, 0, err
			}
		} else if int(msgId) == rpc.SC_HAND_SHAKE_ACKMsgId { //SC_HAND_SHAKE_ACK
			msgData, err = rpc.RSADecrypt(msgData, rpc.PrivateClientKey)
			if err != nil {
				return nil, 0, err
			}
		} else {
			msgData, err = rpc.RSADecrypt(msgData, rpc.PrivateKey)
			if err != nil {
				return nil, 0, err
			}
		}
	case 2:
		msgData, err = rpc.AESCtrDecrypt(msgData, *aesKey, *aesKey...)
		//msgData, err = AESCtrDecrypt(msgData, *aesKey)
		if err != nil {
			return nil, 0, err
		}
	}

	//服务器内部不做加密处理
	msg, _, err = rpc.DecodeMessage(int(msgId), msgData)
	if err != nil {
		//log.Println("[DecodeMessage] err:", err)
		return nil, 0, errors.New(fmt.Sprintf("msg decodeMessage failed:%v %v", msgId, err))
	}

	return
}

func (a *WSMessageProcessor) OnSendMsg(s rocommon.Session, msg interface{}) (err error) {
	opt := s.Node().(SocketOption)

	conn, ok := s.Raw().(*websocket.Conn)
	if !ok || conn == nil {
		util.InfoF("[WSMessageProcessor] OnRecvMsg err")
		return nil
	}
	nodeName := s.Node().(rocommon.ServerNodeProperty).GetName()
	if nodeName != "wsclient" {
		return
	}
	aesKey := s.GetAES()
	var (
		msgData []byte
		msgId   uint16
		seqId   uint32
		msgInfo *rocommon.MessageInfo
	)
	switch m := msg.(type) {
	case *rocommon.TransmitPacket:
		msgData = m.MsgData
		msgId = uint16(m.MsgId)
		seqId = m.SeqId
	default:
		msgData, msgInfo, err = rpc.EncodeMessage(msg)
		if err != nil {
			return err
		}
		msgId = uint16(msgInfo.ID)
	}
	//todo
	// 注意上层发包不要超过最大值
	msgLen := len(msgData)
	var cryptType uint16 = 0
	//握手阶段
	if msgId == uint16(rpc.SC_HAND_SHAKE_NTFMsgId) {
		cryptType = 1
		msgData, err = rpc.RSAEncrypt(msgData, rpc.PublicClientKey)
		if err != nil {
			return err
		}
		msgLen = len(msgData)
	} else {
		if len(*aesKey) > 0 && msgId != rpc.SC_PING_ACKMsgId {
			cryptType = 2
			msgData, err = rpc.AESCtrEncrypt(msgData, *aesKey, *aesKey...)
			//msgData, err = AESCtrEncrypt(msgData, *aesKey)
			if err != nil {
				return err
			}
			msgLen = len(msgData)
		}
	}
	if msgLen > opt.MaxMsgLen() {
		err = errors.New(fmt.Sprintf("message too big msgId=%v msglen=%v maxlen=%v", msgId, msgLen, opt.MaxMsgLen()))
		util.FatalF("SendMessage err=%v", err)
		err = nil
		return
	}

	//data := make([]byte, lenMaxLen + msgIdLen + msgLen)
	data := make([]byte, lenMaxLen+msgIdLen+msgSeqlen+msgFlaglen+msgLen) //head + body
	//lenMaxLen
	binary.BigEndian.PutUint16(data, uint16(msgLen))
	//msgIdLen
	binary.BigEndian.PutUint16(data[lenMaxLen:], uint16(msgId))
	//seq 返回客户端发送的序列号
	binary.BigEndian.PutUint32(data[lenMaxLen+msgIdLen:], seqId)
	//log.Println("sendSeqId:", seqId)
	//使用的加密方式AES
	binary.BigEndian.PutUint16(data[lenMaxLen+msgIdLen+msgSeqlen:], cryptType)

	//body
	if msgLen > 0 {
		copy(data[lenMaxLen+msgIdLen+msgSeqlen+msgFlaglen:], msgData)
	}
	conn.WriteMessage(websocket.BinaryMessage, data)

	return
}
