package util

import (
	"container/list"
)

// https://github.com/ouqiang/timewheel/blob/master/timewheel.go
type TimeWheel struct {
	interval uint64
	//tick     *time.Ticker
	slots []*list.List //后续优化跳表实现
	//[key定时器唯一标识,slotnum定时器所在槽位]
	timer      map[interface{}]int
	currentIdx int //当前指针所在槽位
	slotNum    int
	Callback   func(twTask *TWTask, ms uint64)

	oldMs uint64
}

const (
	TWTASK_TYPE_Save = 1 //role save操作
)

type TWTask struct {
	Delay    uint64
	circle   int
	Key      interface{}
	Data     interface{}
	Callback func(interface{})

	Uid          uint64
	CallbackType int32
	Repeated     bool
}

// interval ms
func NewTimeWheel(interval uint64, slotNum int) *TimeWheel {
	if interval <= 0 {
		return nil
	}

	tw := &TimeWheel{
		interval:   interval,
		slots:      make([]*list.List, slotNum),
		currentIdx: 0,
		slotNum:    slotNum,
		timer:      map[interface{}]int{},
	}
	tw.initSlots()
	return tw
}

func (a *TimeWheel) initSlots() {
	for idx := 0; idx < a.slotNum; idx++ {
		a.slots[idx] = list.New()
	}
}

//func (a *TimeWheel) Start() {
//	//a.tick = time.NewTicker(a.interval)
//	//a.Start()
//}

func (a *TimeWheel) Update(ms uint64) {
	if a.oldMs <= 0 {
		a.oldMs = ms
		return
	}
	for {
		if a.oldMs > ms {
			a.oldMs = ms
		}
		delaTime := ms - a.oldMs
		if delaTime < a.interval {
			return
		}
		a.oldMs += a.interval
		a.update(ms)
	}
}

func (a *TimeWheel) update(ms uint64) {
	slotIdxList := a.slots[a.currentIdx]
	for item := slotIdxList.Front(); item != nil; {
		task := item.Value.(*TWTask)
		if task.circle > 0 {
			task.circle--
			item = item.Next()
			continue
		}
		a.Callback(task, ms)
		next := item.Next()
		if task.Key != nil {
			delete(a.timer, task.Key)
			//添加到新的槽位节点上，继续触发事件
			slotIdxList.Remove(item)
			if task.Repeated {
				a.AddTask(task)
			}
		}
		item = next
	}

	if a.currentIdx >= a.slotNum-1 {
		a.currentIdx = 0
	} else {
		a.currentIdx++
	}
}

func (a *TimeWheel) AddTask(task *TWTask) bool {
	_, ok := a.timer[task.Key]
	if ok {
		return false
	}

	idx, circle := a.getIdxAndCircle(task.Delay)
	task.circle = circle

	a.slots[idx].PushBack(task)
	if task.Key != nil {
		a.timer[task.Key] = idx
	}
	return true
}

func (a *TimeWheel) getIdxAndCircle(taskDuration uint64) (int, int) {
	tmpVal := int(taskDuration / a.interval)
	circle := int(tmpVal / a.slotNum)
	idx := (a.currentIdx + tmpVal) % a.slotNum
	return idx, circle
}

func (a *TimeWheel) RemoveTask(key interface{}) {
	idx, ok := a.timer[key]
	if !ok {
		return
	}
	slotIdxList := a.slots[idx]
	for item := slotIdxList.Front(); item != nil; {
		task := item.Value.(*TWTask)
		if task.Key == key {
			delete(a.timer, task.Key)
			slotIdxList.Remove(item)
		}
		item = item.Next()
	}
}
