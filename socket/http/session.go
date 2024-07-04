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

func (this *httpSession) Node() rocommon.ServerNode {
	return this.node
}
func (this *httpSession) Raw() interface{} {
	return nil
}
func (this *httpSession) ID() uint64 {
	return 0
}

func (this *httpSession) GetAES() *[]byte {
	return nil
}
func (this *httpSession) SetAES(aes string) {
}
func (this *httpSession) GetHandCode() string {
	return ""
}
func (this *httpSession) IncRecvPingNum(incNum int) {
}
func (this *httpSession) RecvPingNum() int {
	return 0
}

func (this *httpSession) SetHandCode(code string) {
}
func (this *httpSession) GetSessionOpt() interface{} {
	return nil
}
func (this *httpSession) GetSessionOptFlag() bool {
	return true
}
func (this *httpSession) SetSessionOptFlag(flag bool) {
}

func (this *httpSession) Close() {
}
func (this *httpSession) Match(method, url string) bool {
	return this.req.Method == method && this.req.URL.Path == url
}
func (this *httpSession) Request() *http.Request {
	return this.req
}
func (this *httpSession) Response() http.ResponseWriter {
	return this.resp
}

func (this *httpSession) Send(msg interface{}) {
	if proc, ok := msg.(ResponseProc); ok {
		this.err = proc.WriteRespond(this)
		this.respond = true
	} else {
		this.err = ErrUnknownOperation
	}
}

func (this *httpSession) HeartBeat(msg interface{}) {}
