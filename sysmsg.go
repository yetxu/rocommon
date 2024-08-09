package rocommon

import "fmt"

// hooker中使用 上层消息队列通过该消息类型来确定当前是创建了连接
// 连接成功事件
type SessionConnected struct {
	ConnectedSId string
}

func (a *SessionConnected) String() string {
	return fmt.Sprintf("%+v", *a)
}

// 连接出错事件
type SessionConnectError struct{}

func (a *SessionConnectError) String() string {
	return fmt.Sprintf("%+v", *a)
}

// 接收其他服务器/客户端的连接
type SessionAccepted struct{}

func (a *SessionAccepted) String() string {
	return fmt.Sprintf("%+v", *a)
}

// session关闭
type SessionClosed struct {
	CloseSId string
}

func (a *SessionClosed) String() string {
	return fmt.Sprintf("%+v", *a)
}
