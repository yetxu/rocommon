package socket

import (
	"sync"
	"sync/atomic"
)

type (
	NetRuntimeTag struct {
		sync.Mutex
		runState  int64
		StopWg    sync.WaitGroup
		CloseFlag bool
	}
)

func (a *NetRuntimeTag) SetCloseFlag(b bool) {
	a.Lock()
	defer a.Unlock()
	a.CloseFlag = b
}

func (a *NetRuntimeTag) GetCloseFlag() bool {
	a.Lock()
	defer a.Unlock()
	return a.CloseFlag
}

func (a *NetRuntimeTag) SetRuneState(b bool) {
	if b {
		atomic.StoreInt64(&a.runState, 1)
	} else {
		atomic.StoreInt64(&a.runState, 0)
	}
}

func (a *NetRuntimeTag) GetRuneState() bool {
	return atomic.LoadInt64(&a.runState) != 0
}
