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
	a := &NetSyncRecv{
		eventCh: make(chan ProcEvent),
	}
	a.cb = func(ev ProcEvent) {
		a.eventCh <- ev
	}
	return a
}

func (a *NetSyncRecv) EventCB() EventCallBack {
	return a.cb
}

func (a *NetSyncRecv) Recv(cb EventCallBack) *NetSyncRecv {
	cb(<-a.eventCh)
	return a
}

func (a *NetSyncRecv) WaitMsg(msg interface{}) {
	var wg sync.WaitGroup

	wg.Add(1)
	a.Recv(func(ev ProcEvent) {
		//msgType := reflect.TypeOf(msg)
		switch ev.Msg().(type) {
		case *SessionConnected:
			msg = ev.Msg()
			a.sess = ev.Session()
			wg.Done()
		}
	})
	wg.Wait()
	return
}

func (a *NetSyncRecv) Session() Session {
	return a.sess
}
