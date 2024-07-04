package rocommon

import (
	"time"
)

//代表多种类型
type ServerNode interface {
	//开启服务器
	Start() ServerNode
	Stop()
	//tcpConnector / tcpAcceptor
	TypeOfName() string
}

type ServerNodeProperty interface {
	GetName() string //ServerName gate/game/db
	SetName(s string)
	GetAddr() string
	SetAddr(s string)
	SetQueue(v NetEventQueue)
	Queue() NetEventQueue
	SetServerType(t int)
	ServerType() int
	SetZone(t int)
	GetZone() int
	SetIndex(t int)
	GetIndex() int
}

//session管理接口
type SessionMagExport interface {
	GetSession(uint64) Session
	SessionNum() int
	CloseAllSession()
	SetUuidCreateKey(genKey int)
}

//socketOption socketOption.go
type TCPSocketOption interface {
	SetSocketBuff(read, write int, noDelay bool)
	SetMaxMsgLen(size int)
	SetSocketDeadline(read, write time.Duration)
}

type MySqlOption interface {
	SetConnCount(val int)
}

//NetProcessorRPC procrpc.go
type ProcessorRPCBundle interface {
	SetTransmitter(v MessageProcessor)
	SetHooker(v EventHook)
	SetCallback(v EventCallBack)
}

//tcpConnector暴露的对外接口
type TCPConnector interface {
	TCPSocketOption
	SetReconnectTime(delta time.Duration)
	Session() Session
}

//tcpAcceptor暴露的对外接口
type TCPAcceptor interface {
	TCPSocketOption
	SessionMagExport
}

//NetContextSet nodeproperty.go
type ContextSet interface {
	//绑定自定义属性
	SetContextData(key, value interface{}, from string)
	//获得key对应的属性
	GetContextData(key interface{}) (interface{}, bool)
	//根据给定类型获取数据
	RawContextData(key interface{}, valuePtr interface{}) bool //sid(etcd期间使用) ctx(连接成功后服务器之间使用)
}

type HTTPConnector interface {
	Request(method, path string, param *HTTPRequest) error
}
