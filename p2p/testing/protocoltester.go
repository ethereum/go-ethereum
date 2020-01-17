// Copyright 2018 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

/*
the p2p/testing package provides a unit test scheme to check simple
protocol message exchanges with one pivot node and a number of dummy peers
The pivot test node runs a node.Service, the dummy peers run a mock node
that can be used to send and receive messages
*/

package testing

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/simulations"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
)

// ProtocolTester is the tester environment used for unit testing protocol
// message exchanges. It uses p2p/simulations framework
type ProtocolTester struct {
	*ProtocolSession
	network *simulations.Network
}

// NewProtocolTester constructs a new ProtocolTester
// it takes as argument the pivot node id, the number of dummy peers and the
// protocol run function called on a peer connection by the p2p server
func NewProtocolTester(prvkey *ecdsa.PrivateKey, nodeCount int, run func(*p2p.Peer, p2p.MsgReadWriter) error) *ProtocolTester {
	services := adapters.Services{
		"test": func(ctx *adapters.ServiceContext) (node.Service, error) {
			return &testNode{run}, nil
		},
		"mock": func(ctx *adapters.ServiceContext) (node.Service, error) {
			return newMockNode(), nil
		},
	}
	adapter := adapters.NewSimAdapter(services)
	net := simulations.NewNetwork(adapter, &simulations.NetworkConfig{})
	nodeConfig := &adapters.NodeConfig{
		PrivateKey:      prvkey,
		EnableMsgEvents: true,
		Services:        []string{"test"},
	}
	if _, err := net.NewNodeWithConfig(nodeConfig); err != nil {
		panic(err.Error())
	}
	if err := net.Start(nodeConfig.ID); err != nil {
		panic(err.Error())
	}

	node := net.GetNode(nodeConfig.ID).Node.(*adapters.SimNode)
	peers := make([]*adapters.NodeConfig, nodeCount)
	nodes := make([]*enode.Node, nodeCount)
	for i := 0; i < nodeCount; i++ {
		peers[i] = adapters.RandomNodeConfig()
		peers[i].Services = []string{"mock"}
		if _, err := net.NewNodeWithConfig(peers[i]); err != nil {
			panic(fmt.Sprintf("error initializing peer %v: %v", peers[i].ID, err))
		}
		if err := net.Start(peers[i].ID); err != nil {
			panic(fmt.Sprintf("error starting peer %v: %v", peers[i].ID, err))
		}
		nodes[i] = peers[i].Node()
	}
	events := make(chan *p2p.PeerEvent, 1000)
	node.SubscribeEvents(events)
	ps := &ProtocolSession{
		Server:  node.Server(),
		Nodes:   nodes,
		adapter: adapter,
		events:  events,
	}
	self := &ProtocolTester{
		ProtocolSession: ps,
		network:         net,
	}

	self.Connect(nodeConfig.ID, peers...)

	return self
}

// Stop stops the p2p server
func (t *ProtocolTester) Stop() {
	t.Server.Stop()
	t.network.Shutdown()
}

// Connect brings up the remote peer node and connects it using the
// p2p/simulations network connection with the in memory network adapter
func (t *ProtocolTester) Connect(selfID enode.ID, peers ...*adapters.NodeConfig) {
	for _, peer := range peers {
		log.Trace(fmt.Sprintf("connect to %v", peer.ID))
		if err := t.network.Connect(selfID, peer.ID); err != nil {
			panic(fmt.Sprintf("error connecting to peer %v: %v", peer.ID, err))
		}
	}

}

// testNode wraps a protocol run function and implements the node.Service
// interface
type testNode struct {
	run func(*p2p.Peer, p2p.MsgReadWriter) error
}

func (t *testNode) Protocols() []p2p.Protocol {
	return []p2p.Protocol{{
		Length: 100,
		Run:    t.run,
	}}
}

func (t *testNode) APIs() []rpc.API {
	return nil
}

func (t *testNode) Start(server *p2p.Server) error {
	return nil
}

func (t *testNode) Stop() error {
	return nil
}

// mockNode is a testNode which doesn't actually run a protocol, instead
// exposing channels so that tests can manually trigger and expect certain
// messages
type mockNode struct {
	testNode

	trigger  chan *Trigger
	expect   chan []Expect
	err      chan error
	stop     chan struct{}
	stopOnce sync.Once
}

func newMockNode() *mockNode {
	mock := &mockNode{
		trigger: make(chan *Trigger),
		expect:  make(chan []Expect),
		err:     make(chan error),
		stop:    make(chan struct{}),
	}
	mock.testNode.run = mock.Run
	return mock
}

// Run is a protocol run function which just loops waiting for tests to
// instruct it to either trigger or expect a message from the peer
func (m *mockNode) Run(peer *p2p.Peer, rw p2p.MsgReadWriter) error {
	for {
		select {
		case trig := <-m.trigger:
			wmsg := Wrap(trig.Msg)
			m.err <- p2p.Send(rw, trig.Code, wmsg)
		case exps := <-m.expect:
			m.err <- expectMsgs(rw, exps)
		case <-m.stop:
			return nil
		}
	}
}

func (m *mockNode) Trigger(trig *Trigger) error {
	m.trigger <- trig
	return <-m.err
}

func (m *mockNode) Expect(exp ...Expect) error {
	m.expect <- exp
	return <-m.err
}

func (m *mockNode) Stop() error {
	m.stopOnce.Do(func() { close(m.stop) })
	return nil
}

func expectMsgs(rw p2p.MsgReadWriter, exps []Expect) error {
	matched := make([]bool, len(exps))
	for {
		msg, err := rw.ReadMsg()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		actualContent, err := ioutil.ReadAll(msg.Payload)
		if err != nil {
			return err
		}
		var found bool
		for i, exp := range exps {
			if exp.Code == msg.Code && bytes.Equal(actualContent, mustEncodeMsg(Wrap(exp.Msg))) {
				if matched[i] {
					return fmt.Errorf("message #%d received two times", i)
				}
				matched[i] = true
				found = true
				break
			}
		}
		if !found {
			expected := make([]string, 0)
			for i, exp := range exps {
				if matched[i] {
					continue
				}
				expected = append(expected, fmt.Sprintf("code %d payload %x", exp.Code, mustEncodeMsg(Wrap(exp.Msg))))
			}
			return fmt.Errorf("unexpected message code %d payload %x, expected %s", msg.Code, actualContent, strings.Join(expected, " or "))
		}
		done := true
		for _, m := range matched {
			if !m {
				done = false
				break
			}
		}
		if done {
			return nil
		}
	}
	for i, m := range matched {
		if !m {
			return fmt.Errorf("expected message #%d not received", i)
		}
	}
	return nil
}

// mustEncodeMsg uses rlp to encode a message.
// In case of error it panics.
func mustEncodeMsg(msg interface{}) []byte {
	contentEnc, err := rlp.EncodeToBytes(msg)
	if err != nil {
		panic("content encode error: " + err.Error())
	}
	return contentEnc
}

type WrappedMsg struct {
	Context []byte
	Size    uint32
	Payload []byte
}

func Wrap(msg interface{}) interface{} {
	data, _ := rlp.EncodeToBytes(msg)
	return &WrappedMsg{
		Size:    uint32(len(data)),
		Payload: data,
	}
}
