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
	adapters.NetAdapter
	TestMessenger
	TestNetAdapter
}

type ExchangeSession struct {
	network *simulations.Network
	na      adapters.NetAdapter
	*ExchangeTestSession
}

// NewProtocolTester returns an exchange test session
// this is a resource driver for protocol message exchange
// scenarios expressed as expects and triggers
// see p2p/protocols/exhange_test.go for an example
// this is used primarily to unit test protocols or protocol modules
// correct message exchange, forwarding, and broadcast
// higher level or network behaviour should be tested with network simulators
func NewProtocolTester(t *testing.T, id *discover.NodeID, n int, run func(adapters.NetAdapter, adapters.Messenger) adapters.ProtoCall) *ExchangeSession {
	ids := RandomNodeIDs(n)
	network := simulations.NewNetwork(&adapters.SimPipe{})

	// setup a simulated network of n nodes
	// Startup pivot node
	err := network.StartNode(&simulations.NodeConfig{ID: id, Run: run})
	if err != nil {
		panic(err.Error())
	}
	na := network.GetNode(id).NetAdapter
	s := NewExchangeTestSession(t, na.(TestNetAdapter), network.Messenger.(TestMessenger), nil)
	self := &ExchangeSession{
		network:             network,
		na:                  na,
		ExchangeTestSession: s,
	}
	self.Connect(ids...)
	// Start up connections to virual nodes serving as endpoints for sending/receiving messages for peers
	return self
}

func (self *ExchangeTestSession) Flush(code int, ids ...*discover.NodeID) {
	self.TestConnected(false, ids...)
	glog.V(6).Infof("flushing peers %v (code %v)", ids, code)
	self.TestExchanges(flushExchange(code, ids...))
	self.TestConnected(true, ids...)
}

func (self *ExchangeSession) StartNode(id *discover.NodeID) error {
	err := self.network.StartNode(&simulations.NodeConfig{ID: id, Run: nil})
	if err != nil {
		return err
	}
	self.IDs = append(self.IDs, id)
	return nil
}

func (self *ExchangeSession) Connect(ids ...*discover.NodeID) {
	for _, id := range ids {
		glog.V(6).Infof("start node %v", id)
		self.StartNode(id)
		glog.V(6).Infof("connect to %v", id)
		self.na.Connect(id[:])
	}

}

func RandomNodeID() *discover.NodeID {
	key, err := crypto.GenerateKey()
	if err != nil {
		panic("unable to generate key")
	}
	var id discover.NodeID
	pubkey := crypto.FromECDSAPub(&key.PublicKey)
	copy(id[:], pubkey[1:])
	return &id
}

func RandomNodeIDs(n int) []*discover.NodeID {
	var ids []*discover.NodeID
	for i := 0; i < n; i++ {
		ids = append(ids, RandomNodeID())
	}
	return ids
}
