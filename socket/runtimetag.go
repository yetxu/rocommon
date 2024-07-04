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

func (this *NetRuntimeTag) SetCloseFlag(b bool) {
	this.Lock()
	defer this.Unlock()
	this.CloseFlag = b
}

func (this *NetRuntimeTag) GetCloseFlag() bool {
	this.Lock()
	defer this.Unlock()
	return this.CloseFlag
}

func (this *NetRuntimeTag) SetRuneState(b bool) {
	if b {
		atomic.StoreInt64(&this.runState, 1)
	} else {
		atomic.StoreInt64(&this.runState, 0)
	}
}

func (this *NetRuntimeTag) GetRuneState() bool {
	return atomic.LoadInt64(&this.runState) != 0
}
