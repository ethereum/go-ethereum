package testing

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/adapters"
	"github.com/ethereum/go-ethereum/p2p/simulations"
	"github.com/ethereum/go-ethereum/rpc"
)

type ProtocolTester struct {
	*ProtocolSession
	network *simulations.Network
}

func NewProtocolTester(t *testing.T, id *adapters.NodeId, n int, run func(*p2p.Peer, p2p.MsgReadWriter) error) *ProtocolTester {
	net := simulations.NewNetwork(&simulations.NetworkConfig{})
	naf := func(conf *simulations.NodeConfig) adapters.NodeAdapter {
		node := &testNode{}
		if conf.Id.NodeID == id.NodeID {
			log.Trace(fmt.Sprintf("adapter run function set to protocol for node %v (=%v)", conf.Id, id))
			node.run = run
		}
		return adapters.NewSimNode(conf.Id, node, net)
	}
	net.SetNaf(naf)

	if err := net.NewNode(&simulations.NodeConfig{Id: id}); err != nil {
		panic(err.Error())
	}
	if err := net.Start(id); err != nil {
		panic(err.Error())
	}

	node := net.GetNodeAdapter(id).(*adapters.SimNode)
	ids := adapters.RandomNodeIds(n)
	ps := NewProtocolSession(node, ids)
	self := &ProtocolTester{
		ProtocolSession: ps,
		network:         net,
	}

	self.Connect(id, ids...)

	return self
}

func (self *ProtocolTester) Connect(selfId *adapters.NodeId, ids ...*adapters.NodeId) {
	for _, id := range ids {
		log.Trace(fmt.Sprintf("start node %v", id))
		if err := self.network.NewNode(&simulations.NodeConfig{Id: id}); err != nil {
			panic(fmt.Sprintf("error starting peer %v: %v", id, err))
		}
		if err := self.network.Start(id); err != nil {
			panic(fmt.Sprintf("error starting peer %v: %v", id, err))
		}
		log.Trace(fmt.Sprintf("connect to %v", id))
		if err := self.network.Connect(selfId, id); err != nil {
			panic(fmt.Sprintf("error connecting to peer %v: %v", id, err))
		}
	}

}

// testNode wraps a protocol run function and implements the node.Service
// interface
type testNode struct {
	run func(*p2p.Peer, p2p.MsgReadWriter) error
}

func (t *testNode) Protocols() []p2p.Protocol {
	return []p2p.Protocol{{Run: t.run}}
}

func (t *testNode) APIs() []rpc.API {
	return nil
}

func (t *testNode) Start(server p2p.Server) error {
	return nil
}

func (t *testNode) Stop() error {
	return nil
}
