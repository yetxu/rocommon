package service

import (
	"flag"
	"os"
)

var (
	ServerCmd = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	//服务器启动配置文件
	flagServerConfigPath = ServerCmd.String("config", "config.yaml", "server config")
	flagServerName       = ServerCmd.String("server", "", "server name")
	FlagServerList       = ServerCmd.String("serverlist", "serverlist.yaml", "serverlist.yaml")

	//临时参数使用
	TempParam         = ServerCmd.String("diff", "abc", "Temp param")
	DBIndexParam      = ServerCmd.Int("db", 0, "DBIndexParam")
	ZoneParam         = ServerCmd.String("zone", "8", "ZoneParam")
	MaxOnlineNumParam = ServerCmd.Int("num", 0, "max online num")
	TestTypeParam     = ServerCmd.Int("t", 1, "test type 1登录压测 2功能压测")
	IPParam           = ServerCmd.String("ip", "127.0.0.1:21001", "test type 1登录压测 2功能压测")
	TypeParam         = ServerCmd.String("type", "", "操作类型")
)
