package http

import (
	"net/http"

	"github.com/yetxu/rocommon"
)

type ResponseProc interface {
	WriteRespond(*httpSession) error
}
type RequestProc interface {
	Match(method, url string) bool
}

type httpSession struct {
	err     error
	respond bool

	node rocommon.ServerNode

	req  *http.Request
	resp http.ResponseWriter
}

func newHttpSession(node rocommon.ServerNode, req *http.Request, res http.ResponseWriter) *httpSession {
	sess := &httpSession{
		node: node,
		req:  req,
		resp: res,
	}
	return sess
}

func (a *httpSession) Node() rocommon.ServerNode {
	return a.node
}
func (a *httpSession) Raw() interface{} {
	return nil
}
func (a *httpSession) ID() uint64 {
	return 0
}

func (a *httpSession) GetAES() *[]byte {
	return nil
}
func (a *httpSession) SetAES(aes string) {
}
func (a *httpSession) GetHandCode() string {
	return ""
}
func (a *httpSession) IncRecvPingNum(incNum int) {
}
func (a *httpSession) RecvPingNum() int {
	return 0
}

func (a *httpSession) SetHandCode(code string) {
}
func (a *httpSession) GetSessionOpt() interface{} {
	return nil
}
func (a *httpSession) GetSessionOptFlag() bool {
	return true
}
func (a *httpSession) SetSessionOptFlag(flag bool) {
}

func (a *httpSession) Close() {
}
func (a *httpSession) Match(method, url string) bool {
	return a.req.Method == method && a.req.URL.Path == url
}
func (a *httpSession) Request() *http.Request {
	return a.req
}
func (a *httpSession) Response() http.ResponseWriter {
	return a.resp
}

func (a *httpSession) Send(msg interface{}) {
	if proc, ok := msg.(ResponseProc); ok {
		a.err = proc.WriteRespond(a)
		a.respond = true
	} else {
		a.err = ErrUnknownOperation
	}
}

func (a *httpSession) HeartBeat(msg interface{}) {}
