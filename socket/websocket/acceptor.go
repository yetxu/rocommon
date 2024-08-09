package websocket

import (
	"context"
	"log"
	"net"
	"net/http"

	"github.com/yetxu/rocommon"
	"github.com/yetxu/rocommon/socket"
	"github.com/yetxu/rocommon/util"

	"github.com/gorilla/websocket"
)

// 监听器实现(启动时可能会有多个连接器)
type tcpWebSocketAcceptor struct {
	socket.NetRuntimeTag      //运行状态
	socket.NetTCPSocketOption //socket相关设置
	socket.NetProcessorRPC    //事件处理相关
	socket.SessionManager     //会话管理
	socket.NetServerNodeProperty
	socket.NetContextSet

	// 保存端口
	listener net.Listener

	certfile string
	keyfile  string
	upgrader *websocket.Upgrader
	sv       *http.Server
}

func (a *tcpWebSocketAcceptor) TypeOfName() string {
	return "wsAcceptor"
}

func (a *tcpWebSocketAcceptor) SetHttps(certfile, keyfile string) {
	a.certfile = certfile
	a.keyfile = keyfile
}

func (a *tcpWebSocketAcceptor) Start() rocommon.ServerNode {
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
		util.PanicF("webSocketAcceptor listen failure=%v", err)
	}

	a.listener = ln
	util.InfoF("webSocketAcceptor listen success=%v", a.GetAddr())

	//process
	//结束中
	if a.GetCloseFlag() {
		return a
	}
	a.SetRuneState(true)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		conn, err := a.upgrader.Upgrade(w, r, nil)
		if err != nil {
			util.InfoF("[webSocketAcceptor] accept err=%v", err)
			return
		}

		a.SocketOptWebSocket(conn) //option 设置
		session := newWebSocketSession(conn, a, nil)
		session.SetContextData("request", r, "newWebSocketSession") //获取request相关信息
		//util.InfoF("[tcpAcceptor] accept session:start:%v", session)
		session.Start()
		//通知上层事件(这边的回调要放到队列中，否则会有多线程冲突)
		a.ProcEvent(&rocommon.RecvMsgEvent{Sess: session, Message: &rocommon.SessionAccepted{}})
	})

	a.sv = &http.Server{Addr: a.GetAddr(), Handler: mux}
	go func() {
		util.InfoF("ws.listen(%s) %s", a.GetName(), a.GetAddr())

		if a.certfile != "" && a.keyfile != "" {
			err = a.sv.ServeTLS(a.listener, a.certfile, a.keyfile)
		} else {
			err = a.sv.Serve(a.listener)
		}

		//服务关闭时会打印
		if err != nil {
			util.ErrorF("ws.listen. failed(%s) %v", a.GetName(), err.Error())
		}

		a.SetRuneState(false)
		a.SetCloseFlag(false)
		a.StopWg.Done()
	}()

	return a
}

func (a *tcpWebSocketAcceptor) Stop() {
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

func init() {
	log.Println("webSocketAcceptor server node register")
	socket.RegisterServerNode(func() rocommon.ServerNode {
		node := &tcpWebSocketAcceptor{
			SessionManager: socket.NewNetSessionManager(),
			upgrader: &websocket.Upgrader{
				CheckOrigin: func(r *http.Request) bool {
					return true
				},
			},
		}
		node.NetTCPSocketOption.Init()
		return node
	})
}
