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

func (this *BitMap) Add(num uint32) {
	if this.max < num {
		return
	}
	index := num >> 3
	pos := num & 0x07
	this.bites[index] |= 1 << pos

	this.curNum++
}

func (this *BitMap) IsExist(num uint32) bool {
	if this.max < num {
		return false
	}
	index := num >> 3
	pos := num & 0x07
	return this.bites[index]&(1<<pos) != 0
}
func (this *BitMap) Remove(num uint32) {
	if this.max < num {
		return
	}
	index := num >> 3
	pos := num & 0x07
	this.bites[index] = this.bites[index] & ^(1 << pos)

	this.curNum--
}
func (this *BitMap) Max() uint32 {
	return this.max
}
func (this *BitMap) Bites() []byte {
	return this.bites
}

//清空原先数据
func (this *BitMap) SetBites(b []byte) {
	this.bites = b
	this.cntNumProcess()
}
func (this *BitMap) String() string {
	return fmt.Sprint(this.bites)
}
func (this *BitMap) CurNum() int32 {
	return this.curNum
}
func (this *BitMap) cntNumProcess() {
	this.curNum = 0
	for idx := 0; idx < len(this.bites); idx++ {
		tmpVal := this.bites[idx]
		for {
			if tmpVal <= 0 {
				break
			}
			if (tmpVal & 1) > 0 {
				this.curNum++
			}
			tmpVal >>= 1
		}
	}
}
