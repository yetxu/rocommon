package socket

import (
	"rocommon"
	"rocommon/util"
	"sync"
	"sync/atomic"
	"time"
)

//SessionMagExport interface
type (
	SessionManager interface {
		rocommon.SessionMagExport

		Add(rocommon.Session)
		Remove(rocommon.Session)
	}

	NetSessionManager struct {
		sessionIdGen int64
		count        int64
		sessionMap   sync.Map
		genKey       int
		timeKey      uint32

		lastTimeStamp uint64
		sequence      uint64
	}
)

func NewNetSessionManager() *NetSessionManager {
	mag := &NetSessionManager{
		lastTimeStamp: util.GetTimeMilliseconds(),
	}
	return mag
}

//func (this *NetSessionManager) Add(s rocommon.Session) {
//	id := atomic.AddInt64(&this.sessionIdGen, 1)
//	atomic.AddInt64(&this.count, 1)
//	if id >= math.MaxUint32 {
//		id = 1
//		atomic.StoreInt64(&this.sessionIdGen, 1)
//	}
//	uuid := this.uuidCreate(id)
//	s.(interface{ SetID(uint642 uint64) }).SetID(uuid)
//	this.sessionMap.Store(uuid, s)
//}
func (this *NetSessionManager) Add(s rocommon.Session) {
	uuid := this.genSessionId()
	s.(interface{ SetID(uint642 uint64) }).SetID(uuid)
	//util.InfoF("NetSessionManager add uid=%v", uuid)
	if _, ok := this.sessionMap.Load(uuid); ok {
		util.ErrorF("NetSessionManager add multiple uid=%v", uuid)
		panic(nil)
	}
	this.sessionMap.Store(uuid, s)
}

func (this *NetSessionManager) Remove(s rocommon.Session) {
	this.sessionMap.Delete(s.ID())
	s.(rocommon.ContextSet).SetContextData("user", nil, "MagRemove")
	//if data,ok := s.(rocommon.ContextSet).SetContextData("user");ok {
	//	if data != nil {
	//		data.
	//	}
	//}
	atomic.AddInt64(&this.count, -1)
}

func (this *NetSessionManager) GetSession(id uint64) rocommon.Session {
	if s, ok := this.sessionMap.Load(id); ok {
		return s.(rocommon.Session)
	}
	return nil
}

func (this *NetSessionManager) SessionNum() int {
	return int(atomic.LoadInt64(&this.count))
}

func (this *NetSessionManager) CloseAllSession() {
	this.sessionMap.Range(func(key, value interface{}) bool {
		value.(rocommon.Session).Close()
		return true
	})
}

func (this *NetSessionManager) SetUuidCreateKey(genKey int) {
	this.genKey = genKey
	this.timeKey = uint32(time.Now().UnixNano())
}

func (this *NetSessionManager) uuidCreate(id int64) uint64 {
	var uuid uint64 = 0
	uuid |= uint64(uint64(this.genKey) << 32) //6位
	uuid |= uint64(id) << 20
	uuid |= uint64(this.timeKey) & 0xfffff

	return uuid
}
func (this *NetSessionManager) genSessionId() uint64 {
	currentTimeStamp := uint64(time.Now().Unix())
	if this.lastTimeStamp == 0 {
		this.lastTimeStamp = currentTimeStamp
	}

	if this.genKey > 0xff {
		return 0
	}
	if this.lastTimeStamp == currentTimeStamp {
		this.sequence++
		if this.sequence > 0xffff {
			//一秒内尝试的数量超过上限
			return 0
		}
	} else {
		this.lastTimeStamp = currentTimeStamp
		this.sequence = 1
	}

	var uid uint64 = 0
	//前32位时间戳（秒）
	uid |= this.lastTimeStamp << 24
	//16位序列号
	uid |= uint64(this.sequence&0xffff) << 8
	//左后是逻辑ID（服务器id）
	uid |= uint64(this.genKey & 0xff)
	return uid
}
