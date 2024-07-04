package socket

import (
	"log"
	"rocommon"
	"rocommon/util"
)

type serverCreate func() rocommon.ServerNode

var serverNodeByName = map[string]serverCreate{}

func RegisterServerNode(f serverCreate) {
	node := f()

	if _, ok := serverNodeByName[node.TypeOfName()]; ok {
		log.Fatalf("serverNode type has register name:[%v]", node.TypeOfName())
	}
	serverNodeByName[node.TypeOfName()] = f
}

func NewServerNode(serverType, serverName, addr string, que rocommon.NetEventQueue) rocommon.ServerNode {
	f := serverNodeByName[serverType]
	if f == nil {
		util.PanicF("serverNoe type not found %v", serverType)
	}
	node := f()
	nodeProperty := node.(rocommon.ServerNodeProperty)
	nodeProperty.SetAddr(addr)
	nodeProperty.SetName(serverName)
	nodeProperty.SetQueue(que)
	return node
}
