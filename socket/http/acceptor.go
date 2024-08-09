package http

import (
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/yetxu/rocommon"
	"github.com/yetxu/rocommon/socket"
	"github.com/yetxu/rocommon/util"
)

var errNotFound = errors.New("404 Not found")
var ErrUnknownOperation = errors.New("unknown http operation")

type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (a tcpKeepAliveListener) Accept() (net.Conn, error) {
	tc, err := a.AcceptTCP()
	if err != nil {
		return nil, err
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}

type httpAcceptor struct {
	socket.NetContextSet
	socket.NetServerNodeProperty
	socket.NetProcessorRPC //事件处理相关

	sv *http.Server

	httpDir  string
	httpRoot string

	listener net.Listener
}

func (a *httpAcceptor) Port() int {
	if a.listener == nil {
		return 0
	}
	return a.listener.Addr().(*net.TCPAddr).Port
}

func (a *httpAcceptor) Start() rocommon.ServerNode {
	//ServeHTTP
	a.sv = &http.Server{Addr: a.GetAddr(), Handler: a}

	ln, err := net.Listen("tcp", a.GetAddr())
	if err != nil {
		util.ErrorF("http.listen failed=%v", err)
		return a
	}

	a.listener = ln
	util.ErrorF("http.listen success")

	go func() {
		a.sv.Serve(tcpKeepAliveListener{a.listener.(*net.TCPListener)})
		if err != nil && err != http.ErrServerClosed {
			util.ErrorF("http.listen name=%v failed=%v", a.GetName(), err)
		}
	}()
	return a
}

func (a *httpAcceptor) Stop() {

}

func (a *httpAcceptor) TypeOfName() string {
	return "httpAcceptor"
}

func (a *httpAcceptor) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	sess := newHttpSession(a, req, res)

	var msg interface{}
	var err error

	a.ProcEvent(&rocommon.RecvMsgEvent{Sess: sess, Message: msg})

	if sess.err != nil {
		err = sess.err
		http.Error(sess.resp, err.Error(), http.StatusInternalServerError)
		return
	}
	//todo...
	// .html
}

func init() {
	socket.RegisterServerNode(func() rocommon.ServerNode {
		node := &httpAcceptor{}
		return node
	})
}
