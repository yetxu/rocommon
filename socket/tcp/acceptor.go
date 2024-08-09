package tcp

import (
	"context"
	"log"
	"net"
	"time"

	"github.com/yetxu/rocommon"
	"github.com/yetxu/rocommon/socket"
	"github.com/yetxu/rocommon/util"
)

// 监听器实现(启动时可能会有多个连接器)
type tcpAcceptor struct {
	socket.NetRuntimeTag      //运行状态
	socket.NetTCPSocketOption //socket相关设置
	socket.NetProcessorRPC    //事件处理相关
	socket.SessionManager     //会话管理
	socket.NetServerNodeProperty
	socket.NetContextSet

	listener net.Listener
}

// //interface ServerNode
func (a *tcpAcceptor) Start() rocommon.ServerNode {
	//正在停止先等待
	a.StopWg.Wait()
	//防止重入导致错误
	if a.GetRuneState() {
		return a
	}

	//https://github.com/gogf/greuse/blob/master/greuse.go

	var listenCfg = net.ListenConfig{Control: nil}

	ln, err := listenCfg.Listen(context.Background(), "tcp", a.GetAddr())
	//ln, err := net.Listen("tcp", a.GetAddr())
	if err != nil {
		util.PanicF("tcpAcceptor listen failure=%v", err)
	}

	a.listener = ln
	util.InfoF("tcpAcceptor listen success=%v", a.GetAddr())

	go a.tcpAccept()
	return a
}

func (a *tcpAcceptor) tcpAccept() {
	a.SetRuneState(true)
	for {
		conn, err := a.listener.Accept()
		//结束中
		if a.GetCloseFlag() {
			break
		}
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				select {
				case <-time.After(time.Millisecond): //尝试重新获取连接
					continue
				}
			}
			util.InfoF("[tcpAcceptor] accept err:%v", err)
			break
		}
		//util.DebugF("accept ok:%v", conn)
		a.SocketOpt(conn) //option 设置
		func() {
			session := newTcpSession(conn, a, nil)
			//util.InfoF("[tcpAcceptor] accept session:start:%v", session)
			session.Start()
			//通知上层事件(这边的回调要放到队列中，否则会有多线程冲突)
			a.ProcEvent(&rocommon.RecvMsgEvent{Sess: session, Message: &rocommon.SessionAccepted{}})
		}()
	}
	a.SetRuneState(false)
	a.SetCloseFlag(false)
	a.StopWg.Done()
}

func (a *tcpAcceptor) Stop() {
	if !a.GetRuneState() {
		return
	}

	a.StopWg.Add(1)
	a.SetCloseFlag(true)
	a.listener.Close()
	//关闭当前监听服务器的所有连接
	a.CloseAllSession()
	//等待协程结束
	a.StopWg.Wait()
}

func (a *tcpAcceptor) CloseAllSession() {

}

func (a *tcpAcceptor) TypeOfName() string {
	return "tcpAcceptor"
}

func init() {
	log.Println("tcpAcceptor server node register")
	socket.RegisterServerNode(func() rocommon.ServerNode {
		node := &tcpAcceptor{
			SessionManager: socket.NewNetSessionManager(),
		}
		node.NetTCPSocketOption.Init()
		return node
	})
}
