package socket

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/yetxu/rocommon"
	"github.com/yetxu/rocommon/util"
)

// SessionMagExport interface
type (
	SessionManager interface {
		rocommon.SessionMagExport

		Add(rocommon.Session)
		Remove(rocommon.Session)
	}

	NetSessionManager struct {
		//sessionIdGen int64
		count      int64
		sessionMap sync.Map
		genKey     int
		timeKey    uint32

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

//	func (a *NetSessionManager) Add(s rocommon.Session) {
//		id := atomic.AddInt64(&a.sessionIdGen, 1)
//		atomic.AddInt64(&a.count, 1)
//		if id >= math.MaxUint32 {
//			id = 1
//			atomic.StoreInt64(&a.sessionIdGen, 1)
//		}
//		uuid := a.uuidCreate(id)
//		s.(interface{ SetID(uint642 uint64) }).SetID(uuid)
//		a.sessionMap.Store(uuid, s)
//	}
func (a *NetSessionManager) Add(s rocommon.Session) {
	uuid := a.genSessionId()
	s.(interface{ SetID(uint642 uint64) }).SetID(uuid)
	//util.InfoF("NetSessionManager add uid=%v", uuid)
	if _, ok := a.sessionMap.Load(uuid); ok {
		util.ErrorF("NetSessionManager add multiple uid=%v", uuid)
		panic("")
	}
	a.sessionMap.Store(uuid, s)
}

func (a *NetSessionManager) Remove(s rocommon.Session) {
	a.sessionMap.Delete(s.ID())
	s.(rocommon.ContextSet).SetContextData("user", nil, "MagRemove")
	//if data,ok := s.(rocommon.ContextSet).SetContextData("user");ok {
	//	if data != nil {
	//		data.
	//	}
	//}
	atomic.AddInt64(&a.count, -1)
}

func (a *NetSessionManager) GetSession(id uint64) rocommon.Session {
	if s, ok := a.sessionMap.Load(id); ok {
		return s.(rocommon.Session)
	}
	return nil
}

func (a *NetSessionManager) SessionNum() int {
	return int(atomic.LoadInt64(&a.count))
}

func (a *NetSessionManager) CloseAllSession() {
	a.sessionMap.Range(func(key, value interface{}) bool {
		value.(rocommon.Session).Close()
		return true
	})
}

func (a *NetSessionManager) SetUuidCreateKey(genKey int) {
	a.genKey = genKey
	a.timeKey = uint32(time.Now().UnixNano())
}

// func (a *NetSessionManager) uuidCreate(id int64) uint64 {
// 	var uuid uint64 = 0
// 	uuid |= uint64(uint64(a.genKey) << 32) //6位
// 	uuid |= uint64(id) << 20
// 	uuid |= uint64(a.timeKey) & 0xfffff

//		return uuid
//	}
func (a *NetSessionManager) genSessionId() uint64 {
	currentTimeStamp := uint64(time.Now().Unix())
	if a.lastTimeStamp == 0 {
		a.lastTimeStamp = currentTimeStamp
	}

	if a.genKey > 0xff {
		return 0
	}
	if a.lastTimeStamp == currentTimeStamp {
		a.sequence++
		if a.sequence > 0xffff {
			//一秒内尝试的数量超过上限
			return 0
		}
	} else {
		a.lastTimeStamp = currentTimeStamp
		a.sequence = 1
	}

	var uid uint64 = 0
	//前32位时间戳（秒）
	uid |= a.lastTimeStamp << 24
	//16位序列号
	uid |= uint64(a.sequence&0xffff) << 8
	//左后是逻辑ID（服务器id）
	uid |= uint64(a.genKey & 0xff)
	return uid
}
