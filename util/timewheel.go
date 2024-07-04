package util

import (
	"container/list"
)

//	https://github.com/ouqiang/timewheel/blob/master/timewheel.go
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

//	interval ms
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

func (this *TimeWheel) initSlots() {
	for idx := 0; idx < this.slotNum; idx++ {
		this.slots[idx] = list.New()
	}
}

//func (this *TimeWheel) Start() {
//	//this.tick = time.NewTicker(this.interval)
//	//this.Start()
//}

func (this *TimeWheel) Update(ms uint64) {
	if this.oldMs <= 0 {
		this.oldMs = ms
		return
	}
	for {
		if this.oldMs > ms {
			this.oldMs = ms
		}
		delaTime := ms - this.oldMs
		if delaTime < this.interval {
			return
		}
		this.oldMs += this.interval
		this.update(ms)
	}
}

func (this *TimeWheel) update(ms uint64) {
	slotIdxList := this.slots[this.currentIdx]
	for item := slotIdxList.Front(); item != nil; {
		task := item.Value.(*TWTask)
		if task.circle > 0 {
			task.circle--
			item = item.Next()
			continue
		}
		this.Callback(task, ms)
		next := item.Next()
		if task.Key != nil {
			delete(this.timer, task.Key)
			//添加到新的槽位节点上，继续触发事件
			slotIdxList.Remove(item)
			if task.Repeated {
				this.AddTask(task)
			}
		}
		item = next
	}

	if this.currentIdx >= this.slotNum-1 {
		this.currentIdx = 0
	} else {
		this.currentIdx++
	}
}

func (this *TimeWheel) AddTask(task *TWTask) bool {
	_, ok := this.timer[task.Key]
	if ok {
		return false
	}

	idx, circle := this.getIdxAndCircle(task.Delay)
	task.circle = circle

	this.slots[idx].PushBack(task)
	if task.Key != nil {
		this.timer[task.Key] = idx
	}
	return true
}

func (this *TimeWheel) getIdxAndCircle(taskDuration uint64) (int, int) {
	tmpVal := int(taskDuration / this.interval)
	circle := int(tmpVal / this.slotNum)
	idx := (this.currentIdx + tmpVal) % this.slotNum
	return idx, circle
}

func (this *TimeWheel) RemoveTask(key interface{}) {
	idx, ok := this.timer[key]
	if !ok {
		return
	}
	slotIdxList := this.slots[idx]
	for item := slotIdxList.Front(); item != nil; {
		task := item.Value.(*TWTask)
		if task.Key == key {
			delete(this.timer, task.Key)
			slotIdxList.Remove(item)
		}
		item = item.Next()
	}
}
