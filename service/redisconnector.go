package service

import (
	"github.com/go-redis/redis"
	"rocommon/socket"
)

const NIL = redis.Nil

type BaseStore = redis.ZStore
type BaseZ = redis.Z
type BaseCmdable = redis.Cmdable

type RedisConnector interface {
	RedisCli() BaseCmdable
	SetName(s string)
}

type NetRedisConnector struct {
	socket.NetServerNodeProperty
	socket.NetContextSet
	socket.NetRedisParam
	redisCommonCli redis.Cmdable
	//redisCli       *redis.Client
	//redisClusterCli *redis.ClusterClient
	cluster int //>0表示cluster模式(集群版本)
}

func NewNetRedisConnector(addr []string, pwd string, dbIndex, cluster int) RedisConnector {
	rs := &NetRedisConnector{}
	rs.cluster = cluster
	if cluster <= 0 {
		rs.SetAddr(addr[0])
		rs.SetPwd(pwd)
		rs.SetDBIndex(dbIndex)

		rs.redisCommonCli = redis.NewClient(&redis.Options{
			Addr:     addr[0],
			Password: pwd,
			DB:       dbIndex,
		})
	} else {
		rs.SetPwd(pwd)
		rs.redisCommonCli = redis.NewClusterClient(
			&redis.ClusterOptions{
				Addrs:    addr,
				Password: pwd,
			})
	}
	return rs
}

func (this *NetRedisConnector) RedisCli() BaseCmdable {
	return this.redisCommonCli
}
