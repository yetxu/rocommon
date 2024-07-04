package service

import (
	"rocommon"
	"sync"
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

func (this *netServerNode) GetNode(serviceId string) rocommon.ServerNode {
	this.RLock()
	defer this.RUnlock()
	if node, ok := this.nodeList[serviceId]; ok {
		return node
	}
	return nil
}

func (this *netServerNode) RemoveNode(serviceId string) {
	this.Lock()
	defer this.Unlock()
	delete(this.nodeList, serviceId)
}

func (this *netServerNode) AddNode(desc *ETCDServiceDesc, node rocommon.ServerNode) {
	this.Lock()
	this.nodeList[desc.ID] = node
	this.Unlock()
}
