package http

import (
	"errors"
	"net"
	"net/http"
	"rocommon"
	"rocommon/socket"
	"rocommon/util"
	"time"
)

var errNotFound = errors.New("404 Not found")
var ErrUnknownOperation = errors.New("unknown http operation")

type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (this tcpKeepAliveListener) Accept() (net.Conn, error) {
	tc, err := this.AcceptTCP()
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

func (this *httpAcceptor) Port() int {
	if this.listener == nil {
		return 0
	}
	return this.listener.Addr().(*net.TCPAddr).Port
}

func (this *httpAcceptor) Start() rocommon.ServerNode {
	//ServeHTTP
	this.sv = &http.Server{Addr: this.GetAddr(), Handler: this}

	ln, err := net.Listen("tcp", this.GetAddr())
	if err != nil {
		util.ErrorF("http.listen failed=%v", err)
		return this
	}

	this.listener = ln
	util.ErrorF("http.listen success")

	go func() {
		this.sv.Serve(tcpKeepAliveListener{this.listener.(*net.TCPListener)})
		if err != nil && err != http.ErrServerClosed {
			util.ErrorF("http.listen name=%v failed=%v", this.GetName(), err)
		}
	}()
	return this
}

func (this *httpAcceptor) Stop() {

}

func (this *httpAcceptor) TypeOfName() string {
	return "httpAcceptor"
}

func (this *httpAcceptor) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	sess := newHttpSession(this, req, res)

	var msg interface{}
	var err error

	this.ProcEvent(&rocommon.RecvMsgEvent{Sess: sess, Message: msg})

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
