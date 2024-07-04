package websocket

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"rocommon"
	"rocommon/service"
	"rocommon/socket"
	"rocommon/util"
	"sync"
	"time"
)

//连接器实现(启动时可能会有多个连接器)
type wsConnector struct {
	socket.NetRuntimeTag      //运行状态
	socket.NetTCPSocketOption //socket相关设置
	socket.NetProcessorRPC    //事件处理相关
	socket.NetServerNodeProperty
	socket.SessionManager //会话管理
	socket.NetContextSet

	connNum       int //重连次数
	reconnectTime time.Duration
	wg            sync.WaitGroup

	//连接会话
	sess *wsSession
}

func (c *wsConnector) Start() rocommon.ServerNode {
	c.StopWg.Wait()

	if c.GetRuneState() {
		return c
	}

	go c.connect(c.GetAddr())
	return c
}

func (c *wsConnector) Stop() {
	if !c.GetRuneState() {
		return
	}

	c.StopWg.Add(1)
	c.sess.Close()
	c.SetCloseFlag(true)
	c.StopWg.Wait()
}
func (c *wsConnector) TypeOfName() string {
	return "wsConnector"
}

func (c *wsConnector) Session() rocommon.Session {
	return c.sess
}

func (c *wsConnector) SetReconnectTime(delta time.Duration) {
	//调试模式下重连不生效
	if service.DebugMode {
		return
	}
	c.reconnectTime = delta
}

func (c *wsConnector) connect(addr string) {
	c.SetRuneState(true) //true表示正在运行
	for {
		c.connNum++

		dialer := websocket.Dialer{}
		dialer.Proxy = http.ProxyFromEnvironment
		dialer.HandshakeTimeout = 60 * time.Second

		conn, _, err := dialer.Dial(addr, nil)
		if err != nil {
			util.InfoF("dail err=%v", err)
			if c.reconnectTime == 0 || c.GetCloseFlag() {
				//todo... 连接出错事件
				c.ProcEvent(&rocommon.RecvMsgEvent{Sess: c.sess, Message: &rocommon.SessionConnectError{}, Err: err})
				break
			}

			select {
			case <-time.After(c.reconnectTime):
				continue
			}
		}
		c.sess.setConn(conn)

		c.wg.Add(1)
		//设置socket选项
		c.SocketOptWebSocket(conn)
		c.connNum = 0
		//放到session管理器中
		c.sess.Start()
		//连接事件
		c.ProcEvent(&rocommon.RecvMsgEvent{Sess: c.sess, Message: &rocommon.SessionConnected{}})
		c.wg.Wait()

		c.sess.setConn(nil)
		if c.reconnectTime == 0 || c.GetCloseFlag() {
			break
		}

		//sleep reconnectTime
		select {
		case <-time.After(c.reconnectTime):
			continue
		}
	}
	c.SetRuneState(false)
	//todo... 在调用stop后需要处理
	//c.StopWg.Done()
	util.InfoF("connector stop...")
}

func init() {
	log.Println("wsConnector server node register")
	socket.RegisterServerNode(func() rocommon.ServerNode {
		node := new(wsConnector)
		node.SessionManager = socket.NewNetSessionManager()
		node.sess = newWebSocketSession(nil, node, func() {
			node.wg.Done()
		})
		node.NetTCPSocketOption.Init()
		return node
	})
}
