package service

import (
	"sync"

	"github.com/yetxu/rocommon"
)

type MultiServerNode interface {
	GetNode(serviceId string) rocommon.ServerNode
	//添加服务器节点 tcpConnector成功后的节点
	AddNode(desc *ETCDServiceDesc, node rocommon.ServerNode)
	RemoveNode(serviceId string)
}

type netServerNode struct {
	sync.RWMutex
	nodeList map[string]rocommon.ServerNode
}

func NewMultiServerNode() *netServerNode {
	m := &netServerNode{}
	m.nodeList = map[string]rocommon.ServerNode{}
	return m
}

func (a *netServerNode) GetNode(serviceId string) rocommon.ServerNode {
	a.RLock()
	defer a.RUnlock()
	if node, ok := a.nodeList[serviceId]; ok {
		return node
	}
	return nil
}

func (a *netServerNode) RemoveNode(serviceId string) {
	a.Lock()
	defer a.Unlock()
	delete(a.nodeList, serviceId)
}

func (a *netServerNode) AddNode(desc *ETCDServiceDesc, node rocommon.ServerNode) {
	a.Lock()
	a.nodeList[desc.ID] = node
	a.Unlock()
}
