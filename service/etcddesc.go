package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/yetxu/rocommon"
)

type ETCDServiceDesc struct {
	Name      string
	ID        string
	Host      string
	Port      int
	Tags      []string
	Type      int
	Zone      int
	Index     int
	RegTime   int64 //注册时的时间单位为m
	LocalAddr string
}

func (a *ETCDServiceDesc) String() string {
	data, err := json.Marshal(a)
	if err != nil {
		return ""
	}
	return string(data)
}

// 字符串形式返回服务器ID 例如:server/zone/gate#type@zone@index -> gate#1@1@1
func GenServiceID(prop rocommon.ServerNodeProperty) string {
	return fmt.Sprintf("%s#%d@%d@%d",
		prop.GetName(),
		prop.GetZone(),
		prop.ServerType(),
		prop.GetIndex(),
	)
}
func GenSelfServiceID(prop configServerNode) string {
	return fmt.Sprintf("%s#%d@%d@%d",
		serviceName,
		prop.Zone,
		prop.Type,
		prop.Id,
	)
}

func GenDiscoveryServicePrefix(sName string, zone int) string {
	if zone > 0 {
		return "server/" + sName + "#" + strconv.Itoa(zone)
	} else {
		return "server/" + sName + "#"
	}
}

func GenServicePrefix(sName string, zone int) string {
	return "server/" + sName
	//return "server/" + strconv.Itoa(zone) + "/" + sName
}
func GenServiceZonePrefix(zone int) string {
	return "server/" + strconv.Itoa(zone)
}

func GenService(sKey string) string {
	keyList := strings.Split(sKey, "/")
	if len(keyList) >= 2 {
		return keyList[1]
	}
	return ""
}
func GenServiceStatePrefix(prop configServerNode) string {
	id := fmt.Sprintf("%s#%d@%d@%d",
		serviceName,
		prop.Zone,
		prop.Type,
		prop.Id,
	)
	return "serverstate/" + id
}

func str2Num(str string) (int, error) {
	num, e := strconv.ParseInt(str, 10, 32)
	if e != nil {
		return 0, errors.New("serviceId invalid num convert error:" + str)
	} else {
		return int(num), nil
	}
}

func ParseServiceID(sid string) (serverType, zone, index int, err error) {
	str := strings.Split(sid, "#")
	if len(str) < 2 {
		err = errors.New("serviceId invalid:" + sid)
		return
	} else {
		//serviceName := str[0]
		strProp := strings.Split(str[1], "@")
		if len(strProp) < 3 {
			err = errors.New("serviceId invalid:" + sid)
			return
		} else {
			zone, err = str2Num(strProp[0])
			if err != nil {
				return
			}
			serverType, err = str2Num(strProp[1])
			if err != nil {
				return
			}
			index, err = str2Num(strProp[2])
			if err != nil {
				return
			}
		}
	}
	return
}

type CrossMapServerStateInfoDesc struct {
	LineNum int32 //线路编号
	Num     int32 //线路人数
}
type ETCDServiceState struct {
	ID          int
	SID         string
	MaxLineNum  int32 //每个服务器线路最大数量
	StateList   []CrossMapServerStateInfoDesc
	SpaceMaxNum int32 //每个线路最大在线人数
}

func (a ETCDServiceState) String() string {
	data, err := json.Marshal(a)
	if err != nil {
		return ""
	}
	return string(data)
}
