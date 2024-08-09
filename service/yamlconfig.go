package service

import "fmt"

/*
#服务器都对内监听处理(处理服务器之间的连接)
#loglevel
#   Debug 1
#	Info 2
#	Warning 3
#	Error 4
#	Fatal 5
# 服务器类型节点Type:[1 gate] [2 game] [3 db] [4 auth] [5 social chat mail] [10 map]

#服务器都对内监听处理(处理服务器之间的连接)
node:
  addr: 0.0.0.0:8101
  type: 1
  id: 1
  zone: 1
  logfile: 。/log
  config: ./config/csv
  loglevel: 1
  etcdaddr: 192.168.56.102:2379
  concern: game

#服务器对外开放端口
acceptor:
  addr: 0.0.0.0:21001

#处理redis连接使用
db:
  redisaddr: 0.0.0.0:6379
*/
//yaml 数据转换为 Go 结构体的在线服务
//https://zhwt.github.io/yaml-to-go/
type ServerNode struct {
	Server []ConfigServerNode
}
type ConfigServerNode struct {
	//服务器都对内监听处理(处理服务器之间的连接)
	Node configServerNode `yaml:"node"`
	//服务器对外开放端口(一般gate使用)
	Acceptor configServerAcceptor `yaml:"acceptor"`
	//DB连接使用
	DB configServerDB `yaml:"db"`
	//elastic
	Elastic configElastic `yaml:"elastic"`
	//SDK
	SDKConfig configSDK `yaml:"sdkhttp"`

	ServerStartTime uint64
}

type configServerNode struct {
	NodeName string `yaml:"nodename"`
	Addr     string `yaml:"addr"`
	//是否是websocket模式
	ISWS       bool   `yaml:"ws"`
	Type       int    `yaml:"type"`
	Id         int    `yaml:"id"`
	Zone       int    `yaml:"zone"`
	Logfile    string `yaml:"logfile"`
	UniLogFile string `yaml:"unilogfile"`
	RecordFile string `yaml:"recordfile"`
	//日志文件等级
	LogLevel int `yaml:"loglevel"`
	//单个日志大小M
	LogMaxSize int    `yaml:"logmaxsize"`
	EtcdAddr   string `yaml:"etcdaddr"`
	//局部跨服etcd注册(跨服远航)
	CrossEtcdAddr string `yaml:"crossetcdaddr"`
	//所有服务器跨服etcd注册
	GCrossEtcdAddr string `yaml:"gcrossetcdaddr"`
	//配置文件路径
	Config string `yaml:"config"`
	//需要连接的服务器类型节点
	Concern []string `yaml:"concern,flow"`
	//创建角色后注册到serverlist列表
	ServerList string `yaml:"serverlist"`
	//php后台gm地址
	PhpServerAddr string `yaml:"phpserveraddr"`
	//是否开启服务器的断线重连
	Reconnect int `yaml:"reconnect"`
	//账号验证模式1:PC模式，不做任何处理 2:激活码模式 3:第三方平台验证模式SDK
	AuthMode int `yaml:"authmode"`
	//1:机器人模式 2:无法注册 3:关闭付费功能
	RobotMode int    `yaml:"robotmode"`
	HttpAddr  string `yaml:"httpaddr"` //gmweb服务器http监听端口
	//gm白名单
	WhiteListGM []string `yaml:"whitelist,flow"`
}

type configServerAcceptor struct {
	Addr string `yaml:"addr"`
}

type configServerConnector struct {
	Concern string `yaml:"concern"`
}

type configServerDB struct {
	RedisAddr    []string `yaml:"redisaddr,flow"`
	Password     string   `yaml:"password"`
	DBIndex      int      `yaml:"dbindex"`
	MysqlAddr    string   `yaml:"mysqladdr"`
	RedisCluster int      `yaml:"cluster"`
}

type configElastic struct {
	Url   string `yaml:"url"`
	Index string `yaml:"index"`
}

type configSDK struct {
	QuickHttpAddr    string `yaml:"quickhttpaddr"`
	QuickHttpAuth    string `yaml:"quickhttpauth"`
	QuickProductCode string `yaml:"quickproductcode"`
	QuickProductKey  string `yaml:"quickproductkey"`
	QuickCallbackKey string `yaml:"quickcallbackkey"`
	QuickMd5key      string `yaml:"quickmd5key"`

	UniHttpAddr     string `yaml:"unihttpaddr"`
	UniSecretKey    string `yaml:"unisecretkey"`
	UniWebTokenAddr string `yaml:"uniwebtokenaddr"`
	UniWebSalt      string `yaml:"uniwebsalt"`
	UniWebPid       string `yaml:"uniwebpid"`

	NbHttpAddr string `yaml:"nbhttpaddr"`
	NbHttpAuth string `yaml:"nbhttpauth"`
	NbGameKey  string `yaml:"nbgamekey"`

	YouYiHttpAddr      string   `yaml:"youyihttpaddr"`
	YouYiPayKey        string   `yaml:"youyipaykey"`
	YouYiGameId        string   `yaml:"youyigameid"`
	YouYiGameIdList    []string `yaml:"youyigameidlist"`
	YouYiGameIdIOS     string   `yaml:"youyigameidios"`
	YouYiGameIdListIOS []string `yaml:"youyigameidlistios"`
}

func (a *ConfigServerNode) Error() string {
	return fmt.Sprintf("%+v", *a)
}
