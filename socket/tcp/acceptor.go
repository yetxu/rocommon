package tcp

import (
	"context"
	"log"
	"net"
	"rocommon"
	"rocommon/socket"
	"rocommon/util"
	"time"
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
func (this *tcpAcceptor) Start() rocommon.ServerNode {
	//正在停止先等待
	this.StopWg.Wait()
	//防止重入导致错误
	if this.GetRuneState() {
		return this
	}

	//https://github.com/gogf/greuse/blob/master/greuse.go

	var listenCfg = net.ListenConfig{Control: nil}

	ln, err := listenCfg.Listen(context.Background(), "tcp", this.GetAddr())
	//ln, err := net.Listen("tcp", this.GetAddr())
	if err != nil {
		util.PanicF("tcpAcceptor listen failure=%v", err)
	}

	this.listener = ln
	util.InfoF("tcpAcceptor listen success=%v", this.GetAddr())

	go this.tcpAccept()
	return this
}

func (this *tcpAcceptor) tcpAccept() {
	this.SetRuneState(true)
	for {
		conn, err := this.listener.Accept()
		//结束中
		if this.GetCloseFlag() {
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
		this.SocketOpt(conn) //option 设置
		func() {
			session := newTcpSession(conn, this, nil)
			//util.InfoF("[tcpAcceptor] accept session:start:%v", session)
			session.Start()
			//通知上层事件(这边的回调要放到队列中，否则会有多线程冲突)
			this.ProcEvent(&rocommon.RecvMsgEvent{Sess: session, Message: &rocommon.SessionAccepted{}})
		}()
	}
	this.SetRuneState(false)
	this.SetCloseFlag(false)
	this.StopWg.Done()
}

func (this *tcpAcceptor) Stop() {
	if !this.GetRuneState() {
		return
	}

	this.StopWg.Add(1)
	this.SetCloseFlag(true)
	this.listener.Close()
	//关闭当前监听服务器的所有连接
	this.CloseAllSession()
	//等待协程结束
	this.StopWg.Wait()
}

func (this *tcpAcceptor) TypeOfName() string {
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
