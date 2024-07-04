package service

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/yetxu/rocommon/socket"
	"github.com/yetxu/rocommon/socket/mysql"
	"github.com/yetxu/rocommon/util"

	"github.com/olivere/elastic/v7"
	"github.com/olivere/elastic/v7/config"
	"gopkg.in/yaml.v2"
)

var (
	serviceName        string //gate game
	serviceConfig      ConfigServerNode
	etcdDiscovery      *ServiceDiscovery //etcd服务器发现
	crossEtcdDiscovery *ServiceDiscovery //跨服etcd服务器发现
	redisConnector     RedisConnector    //用来db连接redis使用
	elasticConnector   *elastic.Client   //elastic
	mysqlConnector     *mysql.MysqlConnector
	mysqlOrmConnector  *mysql.MysqlOrmConnector

	//调试标记(调试模式下设置成true，不会触发重连)
	//默认为非调试模式
	DebugMode = false
)

func Init(name string) {
	rand.Seed(int64(util.GetTimeMilliseconds()))
	serviceName = name
	err := ServerCmd.Parse(os.Args[1:])
	if err != nil {
		log.Printf("ServerCmd.Parse failed!!!")
		panic(err)
	}

	//命令行解析 / yaml配置文件解析 :start game.exe -config game_config.yaml -server game1
	initServerConfig(*flagServerConfigPath, *flagServerName)
	serviceConfig.ServerStartTime = util.GetTimeMilliseconds()

	//日志初始化
	err = util.InitLog(serviceConfig.Node.LogLevel, serviceConfig.Node.LogMaxSize,
		serviceConfig.Node.Logfile, serviceName+strconv.Itoa(serviceConfig.Node.Id)+"_",
		serviceConfig.Node.UniLogFile)
	if err != nil {
		log.Printf("InitLog failed!!! err=%v", err)
		panic(err)
	}

	//etcd
	if etcdDiscovery == nil && serviceConfig.Node.EtcdAddr != "" {
		sd, err := NewNetServiceDiscovery(serviceConfig.Node.EtcdAddr)
		if err != nil {
			util.PanicF("Service Discovery start err:%v addr:%v", err, serviceConfig.Node.EtcdAddr)
		} else {
			etcdDiscovery = sd
			util.InfoF("service discovery start success")
		}
	}
	//cross etcd
	if crossEtcdDiscovery == nil && serviceConfig.Node.CrossEtcdAddr != "" &&
		serviceConfig.Node.CrossEtcdAddr != serviceConfig.Node.EtcdAddr {
		sd, err := NewNetServiceDiscovery(serviceConfig.Node.CrossEtcdAddr)
		if err != nil {
			util.PanicF("Service CrossDiscovery start err:%v addr:%v", err, serviceConfig.Node.CrossEtcdAddr)
		} else {
			crossEtcdDiscovery = sd
			util.InfoF("service CrossDiscovery start success")
		}
	}

	//是否需要连接redis数据库
	if len(serviceConfig.Redis.RedisAddr) > 0 {
		redisConnector = NewNetRedisConnector(serviceConfig.Redis.RedisAddr,
			serviceConfig.Redis.Password,
			serviceConfig.Redis.DBIndex,
			serviceConfig.Redis.RedisCluster)
		redisConnector.SetName(serviceName)
		_, err := redisConnector.RedisCli().Ping().Result()
		if err != nil {
			util.PanicF("New RedisConnector ping failed er=%v", err)
		}
		util.InfoF("redisconnector success...")
	}
	//mysql
	if serviceConfig.Redis.MysqlAddr != "" {
		mysqlConnector = socket.NewServerNode("mysqlConnector", name,
			serviceConfig.Redis.MysqlAddr, nil).(*mysql.MysqlConnector)
		mysqlConnector.Start()
		if mysqlConnector.IsReady() {
			util.InfoF("mysqlConnector connect success...")
		} else {
			util.PanicF("mysqlConnector connect failed...")
		}

		mysqlOrmConnector = socket.NewServerNode("MysqlOrmConnector", name,
			serviceConfig.Redis.MysqlAddr, nil).(*mysql.MysqlOrmConnector)
		mysqlOrmConnector.Start()
		if mysqlOrmConnector.IsReady() {
			util.InfoF("mysqlOrmConnector connect success...")
		} else {
			util.PanicF("mysqlOrmConnector connect failed...")
		}

		//
		//mysqlConnector.Operate(func(rawClient interface{}) interface{} {
		//	client := rawClient.(*sql.DB)
		//	retQuery := mysql.NewWrapper(client).Query("select User from user")
		//
		//	retQuery.Each(func(wrapper *mysql.Wrapper) bool {
		//		var name string
		//		err := wrapper.Scan(&name)
		//		util.InfoF("scan=%v err=%v", name, err)
		//		return true
		//	})
		//
		//	return nil
		//})
	}

	//是否需要连接elasticsearch
	//http://www.wtgame.cn:9200/_nodes/http?pretty
	if serviceConfig.Elastic.Url != "" {
		var err error = nil
		bSniff := false
		elasticConnector, err = elastic.NewClientFromConfig(&config.Config{
			URL:   serviceConfig.Elastic.Url,
			Index: serviceConfig.Elastic.Index,
			Sniff: &bSniff,
		})
		if err != nil {
			util.ErrorF("New elasticConnector err=%v...", err)
		}
		util.InfoF("New ElasticConnector...")
	}
}

func WaitExitSignal() {
	log.Printf("wait for exit signal[SIGTERM SIGINT SIGQUIT SIGKILL]")
	util.InfoF("wait for exit signal[SIGTERM SIGINT SIGQUIT SIGKILL]")
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGKILL)

	<-ch

	if GetMysql() != nil {
		GetMysql().Stop()
	}
	if GetMysqlORM() != nil {
		GetMysqlORM().Stop()
	}
}

func GetServiceName() string {
	return serviceName
}

// server/gate#type@zone@index -> gate#1@1@1
func GetLocalServiceID() string {
	//gate#id@zone@type
	return fmt.Sprintf("%s#%d@%d@%d", serviceName,
		serviceConfig.Node.Zone, serviceConfig.Node.Type, serviceConfig.Node.Id)
}

func initServerConfig(configPath, serverName string) {
	if configPath == "" {
		return
	}

	yamlFile, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Fatalf("load config [%v] err:%v\n", configPath, err)
	}
	//err = yaml.Unmarshal(yamlFile, &serviceConfig)

	//tmpaa := ConfigServerNode{}
	//ab := ConfigServerNode{
	//	Node: configServerNode{
	//		NodeName: "gate1",
	//		Addr:     "12312313",
	//		Type:     1,
	//		Id:       1,
	//		Zone:     1,
	//	},
	//	Acceptor: configServerAcceptor{
	//		Addr: "123123",
	//	},
	//	Redis: configServerDB{
	//		RedisAddr: "1231232",
	//		Password:  "123",
	//	},
	//}
	//tmpaa.Server = append(tmpaa.Server, ab)
	//tmpaa.Server = append(tmpaa.Server, ab)
	//tt, _ := yaml.Marshal(tmpaa)
	//log.Printf("aa=%v", string(tt))

	serviceConfigList := ServerNode{}
	err = yaml.Unmarshal(yamlFile, &serviceConfigList)
	if err != nil {
		log.Fatalf("unmarshal [%v] err:%v\n", configPath, err)
	}

	if serverName == "" {
		strList := strings.Split(configPath, "/")
		if len(strList) > 0 {
			strList1 := strings.Split(strList[len(strList)-1], "_")
			if len(strList1) > 0 {
				serverName = strList1[0]
				if serverName == "" {
					log.Panicf("servername invalid path=%v servername=%v", configPath, serverName)
				}
			}
		}

	}
	bFind := false
	for idx := 0; idx < len(serviceConfigList.Server); idx++ {
		if serviceConfigList.Server[idx].Node.NodeName == serverName {
			serviceConfig = serviceConfigList.Server[idx]
			bFind = true
			break
		}
	}
	if !bFind {
		log.Panicf("servername yaml not find path=%v servername=%v", configPath, serverName)
	}
	log.Println("Server yaml config load success:", configPath, serverName)
}

// 返回服务器配置文件
func GetServiceConfig() ConfigServerNode {
	return serviceConfig
}

func GetServiceDiscovery() *ServiceDiscovery {
	return etcdDiscovery
}

func GetRedis() BaseCmdable {
	return redisConnector.RedisCli()
}
func SetRedis(cli RedisConnector) {
	redisConnector = cli
}

func GetElastic() *elastic.Client {
	return elasticConnector
}
func GetMysql() *mysql.MysqlConnector {
	return mysqlConnector
}
func GetMysqlORM() *mysql.MysqlOrmConnector {
	return mysqlOrmConnector
}
