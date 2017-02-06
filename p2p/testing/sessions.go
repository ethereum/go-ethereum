package testing

import (
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p/adapters"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/simulations"
)

type PeerAdapter interface {
	adapters.NodeAdapter
	TestMessenger
	TestNetAdapter
}

type ExchangeSession struct {
	network *simulations.Network
	na      adapters.NodeAdapter
	*ExchangeTestSession
}

// NewProtocolTester returns an exchange test session
// this is a resource driver for protocol message exchange
// scenarios expressed as expects and triggers
// see p2p/protocols/exhange_test.go for an example
// this is used primarily to unit test protocols or protocol modules
// correct message exchange, forwarding, and broadcast
// higher level or network behaviour should be tested with network simulators
func NewProtocolTester(t *testing.T, id *adapters.NodeId, n int, run func(id adapters.NodeAdapter) adapters.ProtoCall) *ExchangeSession {
	simPipe := adapters.NewSimPipe
	network := simulations.NewNetwork(nil, nil)
	naf := func(conf *simulations.NodeConfig) adapters.NodeAdapter {
		na := adapters.NewSimNode(conf.Id, network, simPipe)
		if conf.Id.NodeID == id.NodeID {
			glog.V(6).Infof("adapter run function set to protocol for node %v (=%v)", conf.Id, id)
			na.Run = run(na)
		}
		return na
	}
	network.SetNaf(naf)
	// setup a simulated network of n nodes
	// Startup pivot node
	err := network.NewNode(&simulations.NodeConfig{Id: id})
	if err != nil {
		panic(err.Error())
	}
	glog.V(6).Infof("network created")
	na := network.GetNode(id).Adapter()
	//s := NewExchangeTestSession(t, na.(TestNetAdapter), na.Messenger().(TestMessenger), nil)
	s := NewExchangeTestSession(t, na.(TestNetAdapter), nil)
	self := &ExchangeSession{
		network:             network,
		na:                  na,
		ExchangeTestSession: s,
	}
	ids := RandomNodeIds(n)
	self.Connect(ids...)
	// Start up connections to virual nodes serving as endpoints for sending/receiving messages for peers
	return self
}

func (self *ExchangeTestSession) Flush(code int, ids ...*adapters.NodeId) {
	self.TestConnected(false, ids...)
	glog.V(6).Infof("flushing peers %v (code %v)", ids, code)
	self.TestExchanges(flushExchange(code, ids...))
	self.TestConnected(true, ids...)
}

func (self *ExchangeSession) Start(id *adapters.NodeId) error {
	err := self.network.NewNode(&simulations.NodeConfig{Id: id})
	if err != nil {
		return err
	}
	node := self.network.GetNode(id)
	if node == nil {
		glog.V(6).Infof("node for peer %v not found", id)
		return nil
	}
	if node.Adapter() == nil {
		glog.V(6).Infof("node adapter for peer %v not found", id)
		return nil
	}
	self.Ids = append(self.Ids, id)
	return nil
}

func (self *ExchangeSession) Connect(ids ...*adapters.NodeId) {
	for _, id := range ids {
		glog.V(6).Infof("start node %v", id)
		err := self.Start(id)
		if err != nil {
			glog.V(6).Infof("error starting peer %v: %v", id, err)
		}
		glog.V(6).Infof("connect to %v", id)
		err = self.na.Connect(id.Bytes())
		if err != nil {
			glog.V(6).Infof("error connecting to peer %v: %v", id, err)
		}
	}

}

func RandomNodeId() *adapters.NodeId {
	key, err := crypto.GenerateKey()
	if err != nil {
		panic("unable to generate key")
	}
	var id discover.NodeID
	pubkey := crypto.FromECDSAPub(&key.PublicKey)
	copy(id[:], pubkey[1:])
	return &adapters.NodeId{id}
}

func RandomNodeIds(n int) []*adapters.NodeId {
	var ids []*adapters.NodeId
	for i := 0; i < n; i++ {
		ids = append(ids, RandomNodeId())
	}
	return ids
}
