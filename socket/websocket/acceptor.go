package websocket

import (
	"context"
	"log"
	"net"
	"net/http"
	"rocommon"
	"rocommon/socket"
	"rocommon/util"

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

func (this *tcpWebSocketAcceptor) TypeOfName() string {
	return "wsAcceptor"
}

func (this *tcpWebSocketAcceptor) SetHttps(certfile, keyfile string) {
	this.certfile = certfile
	this.keyfile = keyfile
}

func (this *tcpWebSocketAcceptor) Start() rocommon.ServerNode {
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
		util.PanicF("webSocketAcceptor listen failure=%v", err)
	}

	this.listener = ln
	util.InfoF("webSocketAcceptor listen success=%v", this.GetAddr())

	//process
	//结束中
	if this.GetCloseFlag() {
		return this
	}
	this.SetRuneState(true)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		conn, err := this.upgrader.Upgrade(w, r, nil)
		if err != nil {
			util.InfoF("[webSocketAcceptor] accept err=%v", err)
			return
		}

		this.SocketOptWebSocket(conn) //option 设置
		session := newWebSocketSession(conn, this, nil)
		session.SetContextData("request", r, "newWebSocketSession") //获取request相关信息
		//util.InfoF("[tcpAcceptor] accept session:start:%v", session)
		session.Start()
		//通知上层事件(这边的回调要放到队列中，否则会有多线程冲突)
		this.ProcEvent(&rocommon.RecvMsgEvent{Sess: session, Message: &rocommon.SessionAccepted{}})
	})

	this.sv = &http.Server{Addr: this.GetAddr(), Handler: mux}
	go func() {
		util.InfoF("ws.listen(%s) %s", this.GetName(), this.GetAddr())

		if this.certfile != "" && this.keyfile != "" {
			err = this.sv.ServeTLS(this.listener, this.certfile, this.keyfile)
		} else {
			err = this.sv.Serve(this.listener)
		}

		//服务关闭时会打印
		if err != nil {
			util.ErrorF("ws.listen. failed(%s) %v", this.GetName(), err.Error())
		}

		this.SetRuneState(false)
		this.SetCloseFlag(false)
		this.StopWg.Done()
	}()

	return this
}

func (this *tcpWebSocketAcceptor) Stop() {
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
