package socket

import (
	"math"
	"net"
	"time"

	"github.com/gorilla/websocket"
)

type SocketOption interface {
	MaxMsgLen() int
	SocketReadTimeout(c net.Conn, callback func())
	SocketWriteTimeout(c net.Conn, callback func())
	CopyOpt(opt *NetTCPSocketOption)
	GetSocketDeadline() (time.Duration, time.Duration)
}

type NetTCPSocketOption struct {
	readBufferSize  int
	writeBufferSize int
	readTimeout     time.Duration
	writeTimeout    time.Duration

	noDelay   bool
	maxMsgLen int
}

func (a *NetTCPSocketOption) Init() {
	//todo...
	a.maxMsgLen = 1024 * 4 * 10 //40k(发送和接受字节最大数量)
}

func (a *NetTCPSocketOption) WriteTimeout() time.Duration {
	return a.writeTimeout
}

func (a *NetTCPSocketOption) ReadTimeout() time.Duration {
	return a.readTimeout
}

func (op *NetTCPSocketOption) SocketOpt(c net.Conn) {
	if conn, ok := c.(*net.TCPConn); ok {
		conn.SetNoDelay(op.noDelay)
		conn.SetReadBuffer(op.readBufferSize)
		conn.SetWriteBuffer(op.writeBufferSize)
		//reuse addr
		//todo...
	}
}
func (op *NetTCPSocketOption) SocketOptWebSocket(c *websocket.Conn) {
	if conn, ok := c.UnderlyingConn().(*net.TCPConn); ok {
		conn.SetNoDelay(op.noDelay)
		conn.SetReadBuffer(op.readBufferSize)
		conn.SetWriteBuffer(op.writeBufferSize)
		//reuse addr
		//todo...
	}
}

func (op *NetTCPSocketOption) MaxMsgLen() int {
	return op.maxMsgLen
}

func (op *NetTCPSocketOption) SetMaxMsgLen(size int) {
	op.maxMsgLen = size
}

// http://blog.sina.com.cn/s/blog_9be3b8f10101lhiq.html
func (op *NetTCPSocketOption) SocketReadTimeout(c net.Conn, callback func()) {
	if op.readTimeout > 0 {
		c.SetReadDeadline(time.Now().Add(op.readTimeout))
		callback()
		c.SetReadDeadline(time.Time{})
	} else {
		callback()
	}
}

func (op *NetTCPSocketOption) SocketWriteTimeout(c net.Conn, callback func()) {
	if op.writeTimeout > 0 {
		c.SetWriteDeadline(time.Now().Add(op.writeTimeout))
		callback()
		c.SetWriteDeadline(time.Time{})
	} else {
		callback()
	}
}

func (op *NetTCPSocketOption) SetSocketBuff(read, write int, noDelay bool) {
	op.readBufferSize = read
	op.writeBufferSize = write
	op.noDelay = noDelay
	if read > 0 {
		op.maxMsgLen = read
	}
	if op.maxMsgLen >= math.MaxUint16 {
		op.maxMsgLen = math.MaxUint16
	}
}

func (op *NetTCPSocketOption) SetSocketDeadline(read, write time.Duration) {
	op.readTimeout = read
	op.writeTimeout = write
}

func (op *NetTCPSocketOption) GetSocketDeadline() (time.Duration, time.Duration) {
	return op.readTimeout, op.writeTimeout
}

func (op *NetTCPSocketOption) CopyOpt(opt *NetTCPSocketOption) {
	//opt.writeTimeout = op.writeTimeout
	//opt.readTimeout = op.readTimeout
	opt.maxMsgLen = op.maxMsgLen
	opt.noDelay = op.noDelay
	opt.readBufferSize = op.readBufferSize
	opt.writeBufferSize = op.writeBufferSize
}
