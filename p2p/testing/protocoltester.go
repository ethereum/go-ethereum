package testing

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/adapters"
	"github.com/ethereum/go-ethereum/p2p/simulations"
)

type ProtocolTester struct {
	*ProtocolSession
	network *simulations.Network
	na      adapters.NodeAdapter
}

func NewProtocolTester(t *testing.T, id *adapters.NodeId, n int, run adapters.ProtoCall) *ProtocolTester {

	net := simulations.NewNetwork(&simulations.NetworkConfig{})
	naf := func(conf *simulations.NodeConfig) adapters.NodeAdapter {
		na := adapters.NewSimNode(conf.Id, net)
		if conf.Id.NodeID == id.NodeID {
			log.Trace(fmt.Sprintf("adapter run function set to protocol for node %v (=%v)", conf.Id, id))
			na.Run = run
		}
		return na
	}
	net.SetNaf(naf)
	err := net.NewNode(&simulations.NodeConfig{Id: id})
	if err != nil {
		panic(err.Error())
	}

	//na := net.GetNode(id).Adapter()
	na := net.GetNodeAdapter(id)

	ids := adapters.RandomNodeIds(n)

	ps := NewProtocolSession(na, ids)
	self := &ProtocolTester{
		ProtocolSession: ps,
		network:         net,
		na:              na,
	}

	self.Connect(ids...)

	return self
}

func (self *ProtocolTester) Start(id *adapters.NodeId) error {
	err := self.network.NewNode(&simulations.NodeConfig{Id: id})
	if err != nil {
		return err
	}
	node := self.network.GetNode(id)
	if node == nil {
		log.Trace(fmt.Sprintf("node for peer %v not found", id))
		return nil
	}
	if node.Adapter() == nil {
		log.Trace(fmt.Sprintf("node adapter for peer %v not found", id))
		return nil
	}
	return nil
}

func (self *ProtocolTester) Connect(ids ...*adapters.NodeId) {
	for _, id := range ids {
		log.Trace(fmt.Sprintf("start node %v", id))
		err := self.Start(id)
		if err != nil {
			log.Trace(fmt.Sprintf("error starting peer %v: %v", id, err))
		}
		log.Trace(fmt.Sprintf("connect to %v", id))
		err = self.na.Connect(id.Bytes())
		if err != nil {
			log.Trace(fmt.Sprintf("error connecting to peer %v: %v", id, err))
		}
	}

}
