package service

import (
	"runtime/debug"
	"sync"
	"time"

	"github.com/yetxu/rocommon"
	"github.com/yetxu/rocommon/util"
)

// // 事件处理队列
// type NetEventQueue interface {
// 	StartQueue() NetEventQueue

// 	StopQueue() NetEventQueue

// 	Wait()

// 	PostCb(callback func())

// 	AttachUpdateModule(update rocommon.UpdateModule)
// }

// 通用UpdateModule处理
type CommonUpdateModule struct {
}

func (a *CommonUpdateModule) Init()            {}
func (a *CommonUpdateModule) Update(ms uint64) {}

func NewEventQueue() rocommon.NetEventQueue {
	que := &eventQueue{
		onError: func(data interface{}) {
			util.InfoF("onError data:%v \n%s\n", data, string(debug.Stack()))
			//打印堆栈信息
			debug.PrintStack()
		},
	}
	//todo...
	//事件列表暂时容量为20000
	que.queList = make(chan interface{}, 20000)
	que.updateModule = &CommonUpdateModule{}
	return que
}

// eventQueue
type eventQueue struct {
	wg           sync.WaitGroup
	queList      chan interface{}  //目前用channel来代替 todo...
	onError      func(interface{}) //打印奔溃处理
	updateModule rocommon.UpdateModule
}

func (a *eventQueue) AttachUpdateModule(update rocommon.UpdateModule) {
	//if a.updateModule != nil {
	//	util.PanicF("update module has been attached !!!")
	//}
	if update != nil {
		update.Init()
		a.updateModule = update
		util.InfoF("update module attached success")
	}
}

var procNum int = 0
var procNumTime time.Time
var callbackNum int = 0
var callbackTime time.Duration

// 处理回调队列主循环
func (a *eventQueue) StartQueue() rocommon.NetEventQueue {
	a.wg.Add(1)
	//游戏服务器只有一个协程，机器人测试时会有DATE RACE
	//procNumTime = util.GetCurrentTimeNow()

	go func() {
		//log.Println("StartQueue goroutine")
		delayTimer := time.NewTimer(5 * time.Millisecond)
		for {
			delayTimer.Reset(5 * time.Millisecond)
			startUpTime := GetServiceStartupTime()
			if startUpTime > 0 {
				break
			}
			select {
			case <-delayTimer.C:
			}
		}
		//默认执行一次更新操作
		a.updateModule.Update(util.GetCurrentTime())

		nowTime1 := util.GetTimeMilliseconds()
		updateDelayTimer := time.NewTicker(5 * time.Millisecond)
	loop:
		for {
			//nowTime1 := util.GetTimeMilliseconds()
			//delayTimer.Reset(5 * time.Millisecond)
			exit := false
			select {
			case msg := <-a.queList:
				switch t := msg.(type) {
				case func():
					//procNum++
					a.queueCall(t)
				case nil:
					exit = true
					break loop //break //退出事件主循环
					//break //退出事件主循环
				}
			//case <-delayTimer.C:
			case <-updateDelayTimer.C:
			}

			//这边添加阶段判断，避免eventqueue中频繁的Update操作
			nowTime2 := util.GetTimeMilliseconds()
			if nowTime1+10 <= nowTime2 { //10ms
				nowTime1 = nowTime2
				a.updateModule.Update(nowTime2)
			}

			//1秒内处理的协议数量
			//a.AddProcNum(time.Now())
			//定时器update操作
			//callbackNum++
			//callbackTime += time.Now().Sub(now) //一个tick执行的消耗时间

			//nowTime := util.GetTimeMilliseconds()
			//delTime1 := nowTime2 - nowTime1
			//delTime2 := nowTime - nowTime1
			//if len(a.queList) > 100 {
			//	util.DebugF("StartQueue deltime1=%v deltime2=%v quelen=%v", delTime1, delTime2, len(a.queList))
			//}

			if exit {
				break
			}
		}

		a.wg.Done()
		//util.InfoF("Exit Queue goroutine")
	}()
	return a
}

func (a *eventQueue) AddProcNum(nowTime time.Time) {
	if nowTime.Sub(procNumTime) > 1*time.Second {
		if callbackNum > 50 && procNum > 0 {
			util.InfoF("[1s] t=%v procNum=%v quelen=%v callbackNum=%v", nowTime.Sub(procNumTime), procNum,
				len(a.queList), callbackNum)
		}
		procNum = 0
		procNumTime = nowTime
		callbackTime = 0
		callbackNum = 0
	}
}

func (a *eventQueue) StopQueue() rocommon.NetEventQueue {
	a.queList <- nil
	return a
}

func (a *eventQueue) Wait() {
	a.wg.Wait()
}

func (a *eventQueue) PostCb(cb func()) {
	if cb != nil {
		a.queList <- cb
	}
}

func (a *eventQueue) queueCall(cb func()) {
	//todo...
	defer func() {
		//打印奔溃信息
		if err := recover(); err != nil {
			a.onError(err)
		}
	}()

	cb()
}
