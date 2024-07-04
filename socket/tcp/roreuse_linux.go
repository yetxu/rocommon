package tcp

import (
	"golang.org/x/sys/unix"
	"syscall"
)

func Control(network, addr string, c syscall.RawConn) (err error) {
	ret := c.Control(func(fd uintptr) {
		if err = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEADDR, 1); err != nil {
			panic(err)
		}
		if err = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1); err != nil {
			panic(err)
		}
	})
	if ret != nil {
		return ret
	}
	return
}
