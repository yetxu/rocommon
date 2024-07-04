package socket

import (
	"reflect"
	"sync"

	"github.com/yetxu/rocommon"
)

// ///////////////////////////////////////////NetServerNodeProperty
type NetServerNodeProperty struct {
	name       string //服务器名称 game,game,auth
	addr       string //包含了ip和port
	queue      rocommon.NetEventQueue
	serverType int //服务器类型(例如gate，game，auth)
	zone       int //前服务器区号(理解成服务组)
	index      int //服务器区内的编号
}

func (this *NetServerNodeProperty) GetName() string {
	return this.name
}

func (this *NetServerNodeProperty) SetName(s string) {
	this.name = s
}

func (this *NetServerNodeProperty) GetAddr() string {
	return this.addr
}

func (this *NetServerNodeProperty) SetAddr(s string) {
	this.addr = s
}

func (this *NetServerNodeProperty) SetQueue(v rocommon.NetEventQueue) {
	this.queue = v
}

func (this *NetServerNodeProperty) Queue() rocommon.NetEventQueue {
	return this.queue
}

func (this *NetServerNodeProperty) SetServerType(t int) {
	this.serverType = t
}

func (this *NetServerNodeProperty) ServerType() int {
	return this.serverType
}

func (this *NetServerNodeProperty) SetZone(t int) {
	this.zone = t
}

func (this *NetServerNodeProperty) GetZone() int {
	return this.zone
}

func (this *NetServerNodeProperty) SetIndex(t int) {
	this.index = t
}

func (this *NetServerNodeProperty) GetIndex() int {
	return this.index
}

// ///////////////////////////////////////////NetContextSet
// 用来记录session数据
type NetContextSet struct {
	guard   sync.RWMutex //读写锁
	dataMap map[interface{}]keyValueData
	//user 玩家
	//sd 服务器相关数据
}
type keyValueData struct {
	key   interface{}
	value interface{}
}

func (this *NetContextSet) SetContextData(key, value interface{}, from string) {
	this.guard.Lock()
	defer this.guard.Unlock()
	if this.dataMap == nil {
		this.dataMap = map[interface{}]keyValueData{}
	}

	if _, ok := this.dataMap[key]; ok {
		if value == nil {
			//util.InfoF("ContextData clean key:%v oldValue:%v newValue:%v [%v]", key, data, value, from)
		} else {
			//util.FatalF("ContextData exist key:%v oldValue:%v newValue:%v [%v]", key, data, value, from)
		}
		this.dataMap[key] = keyValueData{key, value}
	} else {
		this.dataMap[key] = keyValueData{key, value}
	}
}

func (this *NetContextSet) GetContextData(key interface{}) (interface{}, bool) {
	this.guard.RLock()
	defer this.guard.RUnlock()
	if this.dataMap == nil {
		this.dataMap = map[interface{}]keyValueData{}
	}

	if data, ok := this.dataMap[key]; ok {
		return data.value, true
	}
	return nil, false
}

// 根据给定类型获取数据
func (this *NetContextSet) RawContextData(key interface{}, valuePtr interface{}) bool {
	value, ok := this.GetContextData(key)
	if !ok {
		return false
	}

	switch outValue := valuePtr.(type) {
	case *string:
		*outValue = value.(string)
	default:
		v := reflect.Indirect(reflect.ValueOf(valuePtr))
		if value != nil {

			v.Set(reflect.ValueOf(value))
		}
	}
	return true
}

// ///////////////////////////////////////////NetRedisParam
type NetRedisParam struct {
	Pwd     string
	DBIndex int
}

func (this *NetRedisParam) SetPwd(pwd string) {
	this.Pwd = pwd
}

func (this *NetRedisParam) SetDBIndex(db int) {
	this.DBIndex = db
}
