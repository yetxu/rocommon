package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/yetxu/rocommon"
	"github.com/yetxu/rocommon/util"
)

// 第一次服务器启动时间
var ServiceStartupTime uint64 = 0

// 注册到服务器发现
func ETCDRegister(node rocommon.ServerNode, opts ...interface{}) *ETCDServiceDesc {
	property := node.(rocommon.ServerNodeProperty)
	sd := &ETCDServiceDesc{
		ID:    GenServiceID(property),
		Name:  property.GetName(),
		Host:  property.GetAddr(),
		Type:  property.ServerType(),
		Zone:  property.GetZone(),
		Index: property.GetIndex(),
	}
	sd.RegTime = util.GetTimeSeconds()
	//服务器节点信息
	node.(rocommon.ContextSet).SetContextData("sid", sd, "ETCDRegister")

	//获取本地IPv4
	addrs, err := net.InterfaceAddrs()
	if err == nil {
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					sd.LocalAddr = ipnet.IP.String()
					break
				}
			}
		}
	}
	if sd.LocalAddr != "" {
		hostList := strings.Split(sd.Host, ":")
		if hostList[0] == "0.0.0.0" {
			sd.Host = sd.LocalAddr + ":" + hostList[1]
		}
	}

	//先查询是否存在相同的该节点，如果存在不做处理(或者通过del操作关闭其他客户端)
	etcdKey := GenServicePrefix(sd.ID, property.GetZone())
	rsp, err := etcdDiscovery.EtcdKV.Get(context.TODO(), etcdKey)
	if err != nil {
		util.PanicF("etcd discovery get err:%v\n", err)
		//log.Fatalf("etcd discovery get err:%v\n", err)
	} else {
		if rsp.Count > 0 {
			util.PanicF("current node has been register to etcd:%v\n", etcdKey)
			//log.Fatalf("current node has been register to etcd", sd.ID)
		} else {
			etcdDiscovery.RegisterWithTimeOut(etcdKey, sd.String())
			etcdDiscovery.WatchSelf(etcdKey, *sd)
		}
	}
	//cross etcd
	if crossEtcdDiscovery != nil {
		//先查询是否存在相同的该节点，如果存在不做处理(或者通过del操作关闭其他客户端)
		etcdKey := GenServicePrefix(sd.ID, property.GetZone())
		rsp, err := crossEtcdDiscovery.EtcdKV.Get(context.TODO(), etcdKey)
		if err != nil {
			util.PanicF("etcd discovery get err:%v\n", err)
		} else {
			if rsp.Count > 0 {
				util.PanicF("current node has been register to etcd:%v\n", etcdKey)
			} else {
				crossEtcdDiscovery.RegisterWithTimeOut(etcdKey, sd.String())
				crossEtcdDiscovery.WatchSelf(etcdKey, *sd)
			}
		}
	}

	//添加服务器开服时间(server/zone)
	InitServiceStartupTime(property.GetZone())

	return sd
}

func InitServiceStartupTime(zone int) {
	//添加服务器开服时间(server/zone)
	startupKey := GenServiceZonePrefix(zone)
	rsp1, err1 := etcdDiscovery.EtcdKV.Get(context.TODO(), startupKey)
	if err1 != nil {
		util.PanicF("etcd discovery get err:%v\n", err1)
	} else {
		if rsp1.Count > 0 {
			//已经注册了服务器启动时间
			tmpTime, _ := strconv.ParseUint(string(rsp1.Kvs[0].Value), 10, 64)
			atomic.StoreUint64(&ServiceStartupTime, tmpTime)
		} else {
			nowTime := util.GetCurrentTime()
			atomic.StoreUint64(&ServiceStartupTime, nowTime)
			val := strconv.FormatUint(nowTime, 10)
			etcdDiscovery.Register(startupKey, val)
		}
		tmpTime := GetServiceStartupTime()
		tmpTime1 := time.Unix(int64(tmpTime/1000), 0).In(util.GetLoc())
		util.InfoF("Service StartupTime %v| %v", tmpTime, tmpTime1)
	}
}

// return ms
func GetServiceStartupTime() uint64 {
	return atomic.LoadUint64(&ServiceStartupTime)
}

// todo..解除注册
func ETCDUnregister(node rocommon.ServerNode) {
	property := node.(rocommon.ServerNodeProperty)
	// sd := &ETCDServiceDesc{
	// 	ID:    GenServiceID(property),
	// 	Name:  property.GetName(),
	// 	Host:  property.GetAddr(),
	// 	Type:  property.ServerType(),
	// 	Zone:  property.GetZone(),
	// 	Index: property.GetIndex(),
	// }
	// sd.RegTime = util.GetTimeSeconds()
	etcdKey := GenServicePrefix(GenServiceID(property), property.GetZone())

	util.InfoF("ETCDUnregister =%v", etcdKey)
	etcdDiscovery.Del(etcdKey)
	if crossEtcdDiscovery != nil {
		crossEtcdDiscovery.Del(etcdKey)
	}
}

// 发现服务器，服务可能有多个地址，例如需要连接多个game
// todo...返回多个servernode结构体
func DiscoveryService(serviceName string, serviceZone int, nodeCreator func(MultiServerNode, *ETCDServiceDesc)) rocommon.ServerNode {
	//如果已经存在的，就停止之前正在运行的节点(注意不要配置成一样的节点信息，否则会关闭之前的连接)
	multiNode := NewMultiServerNode() //nodereg.go

	//连接同一个zone里的服务器节点
	etcdKey := GenDiscoveryServicePrefix(serviceName, serviceZone)
	/*
		rsp, err := etcdDiscovery.EtcdKV.Get(context.TODO(),etcdKey, clientv3.WithPrefix())
		if err != nil {
			util.FatalF("etcd discovery get err:%v", err)
			//log.Fatalf("etcd discovery get err:%v\n", err)
		}

		logutil.InfoF("service[%v] node find count:%v", etcdKey, rsp.Count)
		//log.Printf("service[%v] node find count:%v\n", serviceName, rsp.Count)
		for _,data := range rsp.Kvs {
			util.InfoF("etcd discovery start connect:%v", string(data.Key))
			//需要判断节点是否已经存在
			var sd ETCDServiceDesc
			err := json.Unmarshal(data.Value, &sd)
			if err != nil {
				util.InfoF("etcd discovery kv[%v][value]err:%v",data.Key, err)
				continue
			}
			//先停止之前的连接，再执行新的连接
			if preNode := multiNode.GetNode(sd.ID); preNode != nil {
				multiNode.RemoveNode(sd.ID)
				preNode.Stop()
			}
			nodeCreator(multiNode, &sd)
		}
	*/

	//会收到key 对应的最近一次变化通知，
	var ch clientv3.WatchChan
	ch = etcdDiscovery.EtcdCli.Watch(context.TODO(), etcdKey, clientv3.WithPrefix())
	//watch操作
	go func() {
		//查找已经存在的节点
		rsp, err := etcdDiscovery.EtcdKV.Get(context.TODO(), etcdKey, clientv3.WithPrefix())
		if err != nil {
			util.FatalF("etcd discovery get err:%v", err)
			//log.Fatalf("etcd discovery get err:%v\n", err)
		}
		util.InfoF("service[%v] node find count:%v", etcdKey, rsp.Count)
		for _, data := range rsp.Kvs {
			util.InfoF("etcd discovery start connect:%v", string(data.Key))
			//需要判断节点是否已经存在
			var sd ETCDServiceDesc
			err := json.Unmarshal(data.Value, &sd)
			if err != nil {
				util.InfoF("etcd discovery kv[%v][value]err:%v", data.Key, err)
				continue
			}
			//先停止之前的连接，再执行新的连接
			if preNode := multiNode.GetNode(sd.ID); preNode != nil {
				multiNode.RemoveNode(sd.ID)
				preNode.Stop()
			}
			nodeCreator(multiNode, &sd)
		}

		for {
			select {
			case c := <-ch:
				//log.Println("etcd discovery watch count:",len(c.Events))
				//todo...处理删除kv操作
				for _, ev := range c.Events {
					switch ev.Type {
					case mvccpb.PUT:
						var sd ETCDServiceDesc
						err := json.Unmarshal(ev.Kv.Value, &sd)
						if err != nil {
							util.InfoF("err:etcd discovery kv[%v][value]err:%v", string(ev.Kv.Key), err)
							continue
						}

						util.InfoF("etcd discovery watch put key=%v", string(ev.Kv.Key))
						//log.Println("etcd discovery watch put key:",string(ev.Kv.Key))
						//先停止之前的连接，再执行新的连接
						if preNode := multiNode.GetNode(sd.ID); preNode != nil {
							//todo...
							//暂时先处理成，如果存在节点则返回(保证节点ip和端口不变的情况下，否则需要启用移除老连接启用新连接)
							util.InfoF("etcd discovery watch put find oldkey:%v %v", string(ev.Kv.Key), sd.ID)
							//continue
							//调试模式下使用已经存在的节点
							if DebugMode {
								util.InfoF("etcd discovery DebugMode=%v", DebugMode)
								continue
							}

							var preDesc *ETCDServiceDesc
							preNode.(rocommon.ContextSet).RawContextData("sid", &preDesc)
							if preDesc.RegTime == sd.RegTime {
								continue
							}
							multiNode.RemoveNode(sd.ID)
							//todo...通过etcd处理，如果相同的键值还存在则服务器启动时会失败，所以这边暂时不做停止处理
							// 后续解决重连时需要注意
							// 重连产生的问题，重连上来后再断开后stop中的wait才能继续，然后再调用nodeCreator函数，导致每次
							// 关闭对端的节点后才进行连接，因为主动调用stop时，重连上了，导致stop会一直在wait状态，导致执行
							// 不到nodeCreator，关闭对端后，stop中的wait被解除（断开连接导致解除），然后执行nodeCreator
							// 但是因为此时对端已经关闭，所以导致开始时想要连接的反而连接不上，处于重连状态
							// 需要context来主动断开所有协程
							preNode.Stop()
							util.InfoF("remove old node:%v time:%v %v", sd.ID, preDesc.RegTime, util.GetTimeByUint32(uint32(preDesc.RegTime)).String())
							//log.Println("remove node:", sd.ID)
						}
						//util.InfoF("etcd discovery watch put k1111ey:%v", string(ev.Kv.Key))
						nodeCreator(multiNode, &sd)

					case mvccpb.DELETE:
						//注意：social关注本区中的其他social节点，所以自己的节点删除这边会通知，其他节点不会
						util.InfoF("etcd discovery watch delete key:%v", string(ev.Kv.Key))
						//log.Println("etcd discovery watch delete key:", string(ev.Kv.Key))

						nodeID := GenService(string(ev.Kv.Key))
						//log.Println("pre delete:", nodeID)
						//先停止之前的连接，再执行新的连接
						if preNode := multiNode.GetNode(nodeID); preNode != nil {
							//不移除可以触发断线重连，否则，这边直接把节点关闭无法触发断线重连
							//避免这边移除后导致etcd无法成功注册的话还能重连成功
							//multiNode.RemoveNode(nodeID)
							//preNode.Stop()
							util.InfoF("delete node:%v", nodeID)
						}
					}
				}
			}
		}
	}()

	return nil
}

// /////////////////////////////////////////
type ServiceDiscovery struct {
	etcdConfig clientv3.Config
	EtcdCli    *clientv3.Client //clientv3.New(conf)
	EtcdKV     clientv3.KV
}

func NewNetServiceDiscovery(addr string) (*ServiceDiscovery, error) {
	sd := &ServiceDiscovery{}
	epsStr := fmt.Sprintf("http://%s", addr)
	sd.etcdConfig = clientv3.Config{
		Endpoints:   []string{epsStr},
		DialTimeout: 3 * time.Second,
	}
	cli, err := clientv3.New(sd.etcdConfig)
	if err != nil {
		return nil, err
	} else {
		sd.EtcdCli = cli
		sd.EtcdKV = clientv3.NewKV(sd.EtcdCli)
		return sd, nil
	}
}

func (this *ServiceDiscovery) Close() {
	this.EtcdCli.Close()
}

func (this *ServiceDiscovery) RegisterWithTimeOut(key string, value string) int64 {
	//获得lease数据
	leaseRsp, err := this.EtcdCli.Grant(context.TODO(), 3)
	if err != nil {
		util.PanicF("etcd grant falied=%v", err)
		//log.Fatalf("etcd grant falied:%v\n", err)
		return 0
	}
	ctx, cancel := context.WithTimeout(context.TODO(), 3*time.Second)
	defer cancel()
	rsp, err := this.EtcdKV.Put(ctx, key, value, clientv3.WithLease(leaseRsp.ID))
	if err != nil {
		//util.PanicF("etcd put key failed=%v\n", err)
		util.FatalF("etcd put key failed:%v\n", err)
		return 0
	} else {
		util.InfoF("etcd register ok key=%v clusterid=%v leaseid=%v etcdaddr=%v", key, rsp.Header.ClusterId, leaseRsp.ID, this.etcdConfig.Endpoints)
		//log.Printf("etcd register server:%v clusterid:%v", key, rsp.Header.ClusterId)
	}
	_, err = this.EtcdCli.KeepAlive(context.TODO(), leaseRsp.ID)
	if err != nil {
		util.PanicF("etcd put key failed=%v\n", err)
	}
	return int64(leaseRsp.ID)
}

// watch自己，网络恢复后得到自己被删除的通知，重新设置key租约
// WatchSelf只重新设置lease，不做其他操作(key只是自己)
func (this *ServiceDiscovery) WatchSelf(key string, value ETCDServiceDesc) {
	//调试模式下不生效
	if DebugMode {
		util.InfoF("DebugMode=%v WatchSelf Invalid", DebugMode)
		return
	}
	//watch自己，网络恢复后得到自己被删除的通知，重新设置key租约
	keepaliveWatch := this.EtcdCli.Watch(context.TODO(), key)
	go func() {
		for {
			select {
			case c := <-keepaliveWatch:
				for _, ev := range c.Events {
					switch ev.Type {
					case mvccpb.DELETE:
						util.InfoF("etcd WatchSelf del-self key=%v etcdaddr=%v", key, this.etcdConfig.Endpoints)
						value.RegTime = util.GetTimeSeconds()
						this.RegisterWithTimeOut(key, value.String())
					}
				}
			}
		}
	}()
}

func (this *ServiceDiscovery) WatchKey(key string) {
	keepaliveWatch := this.EtcdCli.Watch(context.TODO(), key)
	go func() {
		for {
			select {
			case c := <-keepaliveWatch:
				for _, ev := range c.Events {
					switch ev.Type {
					case mvccpb.DELETE:
						util.InfoF("etcd WatchKey del key=%v etcdaddr=%v", key, this.etcdConfig.Endpoints)
					}
				}
			}
		}
	}()
}

func (this *ServiceDiscovery) Del(key string) bool {
	_, err := this.EtcdCli.Delete(context.TODO(), key)
	if err != nil {
		util.FatalF("etcd del key failed:%v", key)
		return false
	}
	return true
}

func (this *ServiceDiscovery) Register(key string, value string) {
	rsp, err := this.EtcdKV.Put(context.TODO(), key, value)
	if err != nil {
		util.PanicF("etcd put key failed:%v\n", err)
		//log.Fatalf("etcd put key failed:%v\n", err)
		return
	} else {
		util.InfoF("etcd register server:%v clusterid:%v", key, rsp.Header.ClusterId)
		log.Printf("etcd register server:%v clusterid:%v", key, rsp.Header.ClusterId)
	}
}

// 上报自己服务器当前的状态，供其它进程获取(例如获取当前地图线路情况)
// leaseId < 0 表示不带lease的key更新
func (this *ServiceDiscovery) UpdateStateToETCD(key, val string, leaseId int64) int64 {
	ctx, cancel := context.WithTimeout(context.TODO(), 3*time.Second)
	defer cancel()

	if leaseId >= 0 {
		if clientv3.LeaseID(leaseId) == clientv3.NoLease {
			leaseId = this.RegisterWithTimeOut(key, val)
			util.InfoF("UpdateStateToETCD first key=%v leaseid=%v", key, leaseId)
			return leaseId
		}

		//查看lease是否过期
		_, err := this.EtcdKV.Put(ctx, key, val, clientv3.WithLease(clientv3.LeaseID(leaseId)))
		if err != nil {
			util.FatalF("UpdateStateToETCD etcd update key failed:%v\n", err)
			//重新申请lease并注册
			leaseId = this.RegisterWithTimeOut(key, val)
		} else {
			//util.InfoF("UpdateStateToETCD etcd update ok key=%v clusterid=%v leaseId=%v etcdaddr=%v", key, rsp.Header.ClusterId, leaseId, this.etcdConfig.Endpoints)
			//log.Printf("etcd register server:%v clusterid:%v", key, rsp.Header.ClusterId)
		}
	} else {
		_, err := this.EtcdKV.Put(ctx, key, val)
		if err != nil {
			util.FatalF("UpdateStateToETCD etcd update key failed:%v\n", err)
		} else {
			//util.InfoF("UpdateStateToETCD etcd update ok key=%v clusterid=%v etcdaddr=%v", key, rsp.Header.ClusterId, this.etcdConfig.Endpoints)
			//log.Printf("etcd register server:%v clusterid:%v", key, rsp.Header.ClusterId)
		}
	}

	return leaseId
}
