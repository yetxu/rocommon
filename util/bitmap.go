package util

import "fmt"

type BitMap struct {
	bites  []byte
	max    uint32
	curNum int32
}

func NewBitMap(max uint32) *BitMap {
	b := make([]byte, (max>>3)+1)
	return &BitMap{bites: b, max: max}
}

func (a *BitMap) Add(num uint32) {
	if a.max < num {
		return
	}
	index := num >> 3
	pos := num & 0x07
	a.bites[index] |= 1 << pos

	a.curNum++
}

func (a *BitMap) IsExist(num uint32) bool {
	if a.max < num {
		return false
	}
	index := num >> 3
	pos := num & 0x07
	return a.bites[index]&(1<<pos) != 0
}
func (a *BitMap) Remove(num uint32) {
	if a.max < num {
		return
	}
	index := num >> 3
	pos := num & 0x07
	a.bites[index] = a.bites[index] & ^(1 << pos)

	a.curNum--
}
func (a *BitMap) Max() uint32 {
	return a.max
}
func (a *BitMap) Bites() []byte {
	return a.bites
}

// 清空原先数据
func (a *BitMap) SetBites(b []byte) {
	a.bites = b
	a.cntNumProcess()
}
func (a *BitMap) String() string {
	return fmt.Sprint(a.bites)
}
func (a *BitMap) CurNum() int32 {
	return a.curNum
}
func (a *BitMap) cntNumProcess() {
	a.curNum = 0
	for idx := 0; idx < len(a.bites); idx++ {
		tmpVal := a.bites[idx]
		for {
			if tmpVal <= 0 {
				break
			}
			if (tmpVal & 1) > 0 {
				a.curNum++
			}
			tmpVal >>= 1
		}
	}
}
