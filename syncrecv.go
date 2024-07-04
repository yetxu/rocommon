package rocommon

import (
	"sync"
)

type NetSyncRecv struct {
	eventCh chan ProcEvent
	cb      func(ev ProcEvent)
	sess    Session
}

func NewNetSyncRecv(node ServerNode) *NetSyncRecv {
	this := &NetSyncRecv{
		eventCh: make(chan ProcEvent),
	}
	this.cb = func(ev ProcEvent) {
		this.eventCh <- ev
	}
	return this
}

func (this *NetSyncRecv) EventCB() EventCallBack {
	return this.cb
}

func (this *NetSyncRecv) Recv(cb EventCallBack) *NetSyncRecv {
	cb(<-this.eventCh)
	return this
}

func (this *NetSyncRecv) WaitMsg(msg interface{}) {
	var wg sync.WaitGroup

	wg.Add(1)
	this.Recv(func(ev ProcEvent) {
		//msgType := reflect.TypeOf(msg)
		switch ev.Msg().(type) {
		case *SessionConnected:
			msg = ev.Msg()
			this.sess = ev.Session()
			wg.Done()
		}
	})
	wg.Wait()
	return
}

func (this *NetSyncRecv) Session() Session {
	return this.sess
}
