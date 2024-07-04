package tcp

import (
	"log"
	"net"
	"sync"
	"time"

	"github.com/yetxu/rocommon"
	"github.com/yetxu/rocommon/service"
	"github.com/yetxu/rocommon/socket"
	"github.com/yetxu/rocommon/util"
)

// 连接器实现(启动时可能会有多个连接器)
type tcpConnector struct {
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
	sess *tcpSession
}

func (c *tcpConnector) connect(addr string) {
	c.SetRuneState(true) //true表示正在运行
	for {
		c.connNum++

		if c.connNum > 1 {
			var preDesc *service.ETCDServiceDesc
			c.RawContextData("sid", &preDesc)
			if preDesc != nil {
				//preDesc 目标节点信息
				util.DebugF("[tcpConnector] connect begin=%v sid=%v[%v]", addr, preDesc.ID, c.connNum-1)
			}
		}
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			//log.Println("dail err:", err)
			if c.reconnectTime == 0 || c.GetCloseFlag() {
				//连接出错事件
				util.ErrorF("tcpConnector err=%v", err.Error())
				c.ProcEvent(&rocommon.RecvMsgEvent{Sess: c.sess, Message: &rocommon.SessionConnectError{}, Err: err})
				break
			}

			select {
			case <-time.After(c.reconnectTime):
				continue
			}
		}
		if c.connNum > 1 {
			var preDesc *service.ETCDServiceDesc
			c.RawContextData("sid", &preDesc)
			if preDesc != nil {
				//preDesc 目标节点信息
				util.DebugF("[tcpConnector] connect success:%v sid=%v[%v]", addr, preDesc.ID, c.connNum-1)
			}
		}
		if c.GetCloseFlag() {
			util.DebugF("[tcpConnector] connect success but be stoped by new node:%v [%v]", addr, c.connNum-1)
			break
		}

		c.wg.Add(1)
		//设置socket选项
		c.SocketOpt(conn)
		c.connNum = 0
		c.sess.setConn(conn)
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
	if c.GetCloseFlag() {
		c.StopWg.Done()
	}
	util.InfoF("connector stop...")
}

// interface ServerNode
func (c *tcpConnector) Start() rocommon.ServerNode {
	c.StopWg.Wait()
	if c.GetRuneState() {
		return c
	}

	go c.connect(c.GetAddr())
	return c
}

func (c *tcpConnector) Stop() {
	if !c.GetRuneState() {
		return
	}
	c.SetCloseFlag(true)
	c.StopWg.Add(1)
	c.sess.Close()
	c.StopWg.Wait()
}

func (c *tcpConnector) TypeOfName() string {
	return "tcpConnector"
}

func (c *tcpConnector) Session() rocommon.Session {
	return c.sess
}

func (c *tcpConnector) SetReconnectTime(delta time.Duration) {
	//调试模式下重连不生效
	if service.DebugMode {
		return
	}
	c.reconnectTime = delta
}

func init() {
	log.Println("tcpConnector server node register")
	socket.RegisterServerNode(func() rocommon.ServerNode {
		node := new(tcpConnector)
		node.SessionManager = socket.NewNetSessionManager()
		node.sess = newTcpSession(nil, node, func() {
			node.wg.Done()
		})
		node.NetTCPSocketOption.Init()
		return node
	})
}
