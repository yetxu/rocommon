package rocommon

import "fmt"

// hooker中使用 上层消息队列通过该消息类型来确定当前是创建了连接
//连接成功事件
type SessionConnected struct {
	ConnectedSId string
}

func (this *SessionConnected) String() string {
	return fmt.Sprintf("%+v", *this)
}

//连接出错事件
type SessionConnectError struct{}

func (this *SessionConnectError) String() string {
	return fmt.Sprintf("%+v", *this)
}

//接收其他服务器/客户端的连接
type SessionAccepted struct{}

func (this *SessionAccepted) String() string {
	return fmt.Sprintf("%+v", *this)
}

//session关闭
type SessionClosed struct {
	CloseSId string
}

func (this *SessionClosed) String() string {
	return fmt.Sprintf("%+v", *this)
}
