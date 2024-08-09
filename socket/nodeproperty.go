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

func (a *NetServerNodeProperty) GetName() string {
	return a.name
}

func (a *NetServerNodeProperty) SetName(s string) {
	a.name = s
}

func (a *NetServerNodeProperty) GetAddr() string {
	return a.addr
}

func (a *NetServerNodeProperty) SetAddr(s string) {
	a.addr = s
}

func (a *NetServerNodeProperty) SetQueue(v rocommon.NetEventQueue) {
	a.queue = v
}

func (a *NetServerNodeProperty) Queue() rocommon.NetEventQueue {
	return a.queue
}

func (a *NetServerNodeProperty) SetServerType(t int) {
	a.serverType = t
}

func (a *NetServerNodeProperty) ServerType() int {
	return a.serverType
}

func (a *NetServerNodeProperty) SetZone(t int) {
	a.zone = t
}

func (a *NetServerNodeProperty) GetZone() int {
	return a.zone
}

func (a *NetServerNodeProperty) SetIndex(t int) {
	a.index = t
}

func (a *NetServerNodeProperty) GetIndex() int {
	return a.index
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

func (a *NetContextSet) SetContextData(key, value interface{}, from string) {
	a.guard.Lock()
	defer a.guard.Unlock()
	if a.dataMap == nil {
		a.dataMap = map[interface{}]keyValueData{}
	}

	if _, ok := a.dataMap[key]; ok {
		// if value != nil {
		// 	util.InfoF("ContextData clean key:%v oldValue:%v newValue:%v [%v]", key, data, value, from)
		// } else {
		// 	util.FatalF("ContextData exist key:%v oldValue:%v newValue:%v [%v]", key, data, value, from)
		// }
		a.dataMap[key] = keyValueData{key, value}
	} else {
		a.dataMap[key] = keyValueData{key, value}
	}
}

func (a *NetContextSet) GetContextData(key interface{}) (interface{}, bool) {
	a.guard.RLock()
	defer a.guard.RUnlock()
	if a.dataMap == nil {
		a.dataMap = map[interface{}]keyValueData{}
	}

	if data, ok := a.dataMap[key]; ok {
		return data.value, true
	}
	return nil, false
}

// 根据给定类型获取数据
func (a *NetContextSet) RawContextData(key interface{}, valuePtr interface{}) bool {
	value, ok := a.GetContextData(key)
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

func (a *NetRedisParam) SetPwd(pwd string) {
	a.Pwd = pwd
}

func (a *NetRedisParam) SetDBIndex(db int) {
	a.DBIndex = db
}
