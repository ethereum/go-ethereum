package testing

import (
	"testing"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p/adapters"
	"github.com/ethereum/go-ethereum/p2p/simulations"
)

type ProtocolTester struct {
	*ProtocolSession
	network *simulations.Network
	na      adapters.NodeAdapter
}

func NewProtocolTester(t *testing.T, id *adapters.NodeId, n int, run func(id adapters.NodeAdapter) adapters.ProtoCall) *ProtocolTester {

	simPipe := adapters.NewSimPipe
	net := simulations.NewNetwork(&simulations.NetworkConfig{})
	naf := func(conf *simulations.NodeConfig) adapters.NodeAdapter {
		na := adapters.NewSimNode(conf.Id, net, simPipe)
		if conf.Id.NodeID == id.NodeID {
			glog.V(logger.Detail).Infof("adapter run function set to protocol for node %v (=%v)", conf.Id, id)
			na.Run = run(na)
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
		glog.V(logger.Detail).Infof("node for peer %v not found", id)
		return nil
	}
	if node.Adapter() == nil {
		glog.V(logger.Detail).Infof("node adapter for peer %v not found", id)
		return nil
	}
	return nil
}

func (self *ProtocolTester) Connect(ids ...*adapters.NodeId) {
	for _, id := range ids {
		glog.V(logger.Detail).Infof("start node %v", id)
		err := self.Start(id)
		if err != nil {
			glog.V(logger.Detail).Infof("error starting peer %v: %v", id, err)
		}
		glog.V(logger.Detail).Infof("connect to %v", id)
		err = self.na.Connect(id.Bytes())
		if err != nil {
			glog.V(logger.Detail).Infof("error connecting to peer %v: %v", id, err)
		}
	}

}
