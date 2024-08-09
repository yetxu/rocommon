package util

import (
	"time"
)

// 定时器接口
type ServerTimer interface {
	//测试是否过期
	IsExpired(ms uint64) bool

	//重置
	Reset(ms uint64, duration time.Duration, fireNow bool)

	Cancel()
	Canceled() bool

	Suspend()
	Resume()
	IsStart() bool
}

const DATE_FORMAT = "2006-01-02 15:04:05"
const DATE_FORMAT_T = "2006-01-02T15:04:05"
const DATE_FORMAT1 = "2006-01-02"
const DATE_FORMAT2 = "15:04:05"
const DATE_FORMAT3 = "15:04"

var gameLoc *time.Location = nil

func GetLoc() *time.Location {
	if gameLoc == nil {
		loc, err := time.LoadLocation("Asia/Shanghai")
		if err != nil {
			gameLoc = time.Local
		} else {
			gameLoc = loc
		}
	}
	return gameLoc
}

// return ms
func GetCurrentTime() uint64 {
	t1 := GetCurrentTimeNow()
	return uint64(t1.UnixNano() / 1e6)
}
func GetCurrentTimeNow() time.Time {
	loc := GetLoc()
	t1 := time.Now()
	return t1.In(loc)
}
func GetTimeMilliseconds() uint64 {
	return GetCurrentTime()
}
func GetTimeSeconds() int64 {
	return GetCurrentTimeNow().Unix()
}

func GetTimeByStr(timeStr string) time.Time {
	tempTime, _ := time.ParseInLocation(DATE_FORMAT, timeStr, GetLoc())
	return tempTime
}
func GetTimeByUint64(uTime uint64) time.Time {
	timeS := int64(uTime / 1000)
	timeMS := int64(uTime % 1000)
	return time.Unix(timeS, timeMS).In(GetLoc())
}
func GetTimeByUint32(uTime uint32) time.Time {
	return time.Unix(int64(uTime), 0).In(GetLoc())
}
func GetHourByTime(uTime uint64) time.Time {
	loc := GetLoc()
	if uTime <= 0 {
		tempTimeStr := GetCurrentTimeNow().Format(DATE_FORMAT2)
		tempTime, _ := time.ParseInLocation(DATE_FORMAT2, tempTimeStr, loc)
		return tempTime
	} else {
		tempTimeStr := GetTimeByUint64(uTime).Format(DATE_FORMAT2)
		tempTime, _ := time.ParseInLocation(DATE_FORMAT2, tempTimeStr, loc)
		return tempTime
	}
}

func GetHourByTimeStr(timeStr string) string {
	loc := GetLoc()
	tempTime, _ := time.ParseInLocation(DATE_FORMAT, timeStr, loc)
	tempTime1 := tempTime.Format(DATE_FORMAT2)
	return tempTime1
}
func GetHourByTimeStr1(timeStr string) time.Time {
	loc := GetLoc()
	hTime, _ := time.ParseInLocation(DATE_FORMAT2, timeStr, loc)
	return hTime
}
func GetDayByTimeStr(timeStr string) string {
	loc := GetLoc()
	sTime, _ := time.ParseInLocation(DATE_FORMAT, timeStr, loc)
	return sTime.Format(DATE_FORMAT1)
}
func GetDayByTimeStr1(timeStr string) time.Time {
	loc := GetLoc()
	dTime, _ := time.ParseInLocation(DATE_FORMAT1, timeStr, loc)
	return dTime
}
func GetDayByTimeStr2(timeMs uint64) time.Time {
	loc := GetLoc()
	tempTimeStr := GetTimeByUint64(timeMs).Format(DATE_FORMAT1)
	tempTime, _ := time.ParseInLocation(DATE_FORMAT1, tempTimeStr, loc)
	return tempTime
}
func GetDayAndHourByTimeStr(timeStr string) (string, string) {
	loc := GetLoc()
	tempTime, _ := time.ParseInLocation(DATE_FORMAT, timeStr, loc)
	hourStr := tempTime.Format(DATE_FORMAT2)
	dayStr := tempTime.Format(DATE_FORMAT1)
	return dayStr, hourStr
}
func GetDayAndHourByTime(timeStr string) (time.Time, time.Time) {
	loc := GetLoc()
	tempTime, _ := time.ParseInLocation(DATE_FORMAT, timeStr, loc)
	dayStr := tempTime.Format(DATE_FORMAT1)
	d, _ := time.ParseInLocation(DATE_FORMAT1, dayStr, loc)
	hourStr := tempTime.Format(DATE_FORMAT2)
	h, _ := time.ParseInLocation(DATE_FORMAT2, hourStr, loc)
	return d, h
}

// 获取最近一天5点更新时间(结束时间戳)
func GetLatest5Hour() uint64 {
	loc := GetLoc()
	nowTime := GetCurrentTimeNow()
	if nowTime.Hour() < 5 {
		dayTimeStr := nowTime.Format(DATE_FORMAT1)
		tempDayTime, _ := time.ParseInLocation(DATE_FORMAT1, dayTimeStr, loc)
		tempDayTime = tempDayTime.Add(time.Hour * 5)

		return uint64(tempDayTime.UnixNano() / 1e6)
	} else {
		dayTimeStr := nowTime.Format(DATE_FORMAT1)
		tempDayTime, _ := time.ParseInLocation(DATE_FORMAT1, dayTimeStr, loc)
		tempDayTime = tempDayTime.Add(time.Hour * 24)

		dayTimeStr = tempDayTime.Format(DATE_FORMAT1)
		tempDayTime, _ = time.ParseInLocation(DATE_FORMAT1, dayTimeStr, loc)
		tempDayTime = tempDayTime.Add(time.Hour * 5)
		return uint64(tempDayTime.UnixNano() / 1e6)
	}
}

// 获取最近周一5点更新时间(结束时间戳)
func GetLatestWeek5Hour(ms uint64) uint64 {
	loc := GetLoc()
	nowTime := GetCurrentTimeNow()
	if ms > 0 {
		nowTime = GetTimeByUint64(ms)
	}
	curWeekDay := nowTime.Weekday()
	if curWeekDay == 0 {
		curWeekDay = 7
	}
	delDay := 7 - curWeekDay + 1
	if curWeekDay == 1 {
		if nowTime.Hour() >= 5 {
			delDay = 7
		} else {
			delDay = 0
		}
	}

	nowTime = nowTime.AddDate(0, 0, int(delDay))
	dayTimeStr := nowTime.Format(DATE_FORMAT1)
	tempDayTime, _ := time.ParseInLocation(DATE_FORMAT1, dayTimeStr, loc)
	tempDayTime = tempDayTime.Add(time.Hour * 5)
	return uint64(tempDayTime.UnixNano() / 1e6)
}

func GetLatestMonth5Hour() uint64 {
	loc := GetLoc()
	nowTime := GetCurrentTimeNow()
	year, month, day := nowTime.Date()

	//如果是每个月的1号5点之前
	if day == 1 && nowTime.Hour() < 5 {
		return GetLatest5Hour()
	}

	//如果是一个月的1号5点之后
	aMonth := time.Date(year, month, 1, 0, 0, 0, 0, time.Local)
	nextMonth := aMonth.AddDate(0, 1, 0)
	dayTimeStr := nextMonth.Format(DATE_FORMAT1)

	tempDayTime, _ := time.ParseInLocation(DATE_FORMAT1, dayTimeStr, loc)
	tempDayTime = tempDayTime.Add(time.Hour * 5)
	return uint64(tempDayTime.UnixNano() / 1e6)
}

// 获取两个时间的持续天数
func GetDurationDay(t1, t2 string) int64 {
	loc := GetLoc()
	t1DayStr := GetDayByTimeStr(t1)
	t2DayStr := GetDayByTimeStr(t2)

	sTime1, _ := time.ParseInLocation(DATE_FORMAT1, t1DayStr, loc)
	eTime1, _ := time.ParseInLocation(DATE_FORMAT1, t2DayStr, loc)
	return eTime1.Unix() - sTime1.Unix()
}

// t1, t2 (ms) t2 > t1
func GetDurationDay1(t1, t2 uint64) int32 {
	loc := GetLoc()
	tmpT1 := GetTimeByUint64(t1)
	tmpT2 := GetTimeByUint64(t2)

	dayT1Str := tmpT1.Format(DATE_FORMAT1)
	dayT1, _ := time.ParseInLocation(DATE_FORMAT1, dayT1Str, loc)
	dayT2Str := tmpT2.Format(DATE_FORMAT1)
	dayT2, _ := time.ParseInLocation(DATE_FORMAT1, dayT2Str, loc)
	return int32(dayT2.Sub(dayT1).Hours() / 24)
}

// 以5点位分割线 t2 > t1
func GetDurationDay2(t1, t2 uint64) int32 {
	loc := GetLoc()
	tmpT1 := GetTimeByUint64(t1)
	tmpT2 := GetTimeByUint64(t2)

	dayT1Str := tmpT1.Format(DATE_FORMAT1)
	dayT1, _ := time.ParseInLocation(DATE_FORMAT1, dayT1Str, loc)
	dayT2Str := tmpT2.Format(DATE_FORMAT1)
	dayT2, _ := time.ParseInLocation(DATE_FORMAT1, dayT2Str, loc)
	deltaDay := int32(dayT2.Sub(dayT1).Hours() / 24)

	if tmpT1.Hour() < 5 {
		deltaDay += 1
	}
	if tmpT2.Hour() >= 5 {
		deltaDay += 1
	}
	return deltaDay
}

// t1 < t2
func IsInSameWeek(t1, t2 uint64) bool {
	if t2 < t1 {
		return false
	}

	tmpT1 := GetLatestWeek5Hour(t1)
	if t1 <= tmpT1 && t2 > tmpT1 {
		return true
	}
	return false
}

type baseTimer struct {
	ts        uint64 //上一次时间戳
	duration  uint64 //持续时间
	cancel    bool   //取消标记
	suspend   bool   //是否暂停
	resetTime uint64 //剩余时间 暂停后会用到
}

func (a *baseTimer) Cancel() {
	a.cancel = true
	a.ts = 0
	a.duration = 0
	a.suspend = false
	a.resetTime = 0
}

func (a *baseTimer) Canceled() bool {
	return a.cancel
}

func (a *baseTimer) Suspend() {
	a.suspend = true
	deltaT := a.duration - (GetCurrentTime() - a.ts)
	if deltaT >= a.duration {
		a.resetTime = 0
	} else {
		a.resetTime = a.duration - a.ts
	}
}

func (a *baseTimer) Resume() {
	a.suspend = false
	a.ts = GetCurrentTime()
}

func (a *baseTimer) IsStart() bool {
	return a.ts != 0
}

// 只运行一次的定时器
type OnceTimer struct {
	baseTimer
}

func NewOnceTimer(ms uint64, duration time.Duration) ServerTimer {
	t := &OnceTimer{
		baseTimer: baseTimer{
			ts:        ms,
			duration:  uint64(duration),
			cancel:    false,
			suspend:   false,
			resetTime: 0,
		},
	}
	return t
}

func (a *OnceTimer) IsExpired(ms uint64) bool {
	if a.cancel || a.suspend {
		return false
	}

	if a.resetTime > 0 {
		if (a.ts + a.resetTime) < ms {
			a.resetTime = 0
			return true
		} else {
			return false
		}
	}
	return (a.ts + a.duration) < ms
}

func (a *OnceTimer) Reset(ms uint64, duration time.Duration, fireNow bool) {
	a.ts = ms
	a.duration = uint64(duration)
	a.cancel = false
}

// 运行无限次定时器
type DurationTimer struct {
	baseTimer
	fireNow      bool  //建立后可以立即被触发无论是否过期
	expiredTimes int32 //过期次数
}

func NewDurationTimer(ms uint64, duration time.Duration) ServerTimer {
	t := &DurationTimer{
		fireNow:      false,
		expiredTimes: 0,
		baseTimer: baseTimer{
			ts:        ms,
			duration:  uint64(duration),
			cancel:    false,
			suspend:   false,
			resetTime: 0,
		},
	}
	return t
}

func (a *DurationTimer) IsExpired(ms uint64) bool {
	if a.cancel || a.suspend {
		return false
	}

	if a.resetTime > 0 {
		if (a.ts + a.resetTime) < ms {
			a.resetTime = 0
			a.Reset(ms, time.Duration(a.duration), true)
			return true
		} else {
			return false
		}
	}
	ret := (a.ts + a.duration) < ms
	if ret {
		a.expiredTimes++
		a.ts = ms
	}
	if a.fireNow {
		a.fireNow = false
		return true
	}
	return ret
}

func (a *DurationTimer) Reset(ms uint64, duration time.Duration, fireNow bool) {
	a.ts = ms
	a.duration = uint64(duration)
	a.cancel = false
	a.fireNow = fireNow
}
