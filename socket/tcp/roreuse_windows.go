package tcp

import (
	"syscall"
)

func Control(network, addr string, c syscall.RawConn) (err error) {
	ret := c.Control(func(fd uintptr) {
		if err = syscall.SetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
			return
		}
	})
	if ret != nil {
		return ret
	}
	return
}
