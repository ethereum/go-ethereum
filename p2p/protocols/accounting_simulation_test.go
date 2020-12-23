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

package protocols

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/mattn/go-colorable"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/simulations"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
)

const (
	content = "123456789"
)

var (
	nodes    = flag.Int("nodes", 30, "number of nodes to create (default 30)")
	msgs     = flag.Int("msgs", 100, "number of messages sent by node (default 100)")
	loglevel = flag.Int("loglevel", 0, "verbosity of logs")
	rawlog   = flag.Bool("rawlog", false, "remove terminal formatting from logs")
)

func init() {
	flag.Parse()
	log.PrintOrigins(true)
	log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(*loglevel), log.StreamHandler(colorable.NewColorableStderr(), log.TerminalFormat(!*rawlog))))
}

//TestAccountingSimulation runs a p2p/simulations simulation
//It creates a *nodes number of nodes, connects each one with each other,
//then sends out a random selection of messages up to *msgs amount of messages
//from the test protocol spec.
//The spec has some accounted messages defined through the Prices interface.
//The test does accounting for all the message exchanged, and then checks
//that every node has the same balance with a peer, but with opposite signs.
//Balance(AwithB) = 0 - Balance(BwithA) or Abs|Balance(AwithB)| == Abs|Balance(BwithA)|
func TestAccountingSimulation(t *testing.T) {
	//setup the balances objects for every node
	bal := newBalances(*nodes)
	//setup the metrics system or tests will fail trying to write metrics
	dir, err := ioutil.TempDir("", "account-sim")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	SetupAccountingMetrics(1*time.Second, filepath.Join(dir, "metrics.db"))
	//define the node.Service for this test
	services := adapters.Services{
		"accounting": func(ctx *adapters.ServiceContext) (node.Service, error) {
			return bal.newNode(), nil
		},
	}
	//setup the simulation
	adapter := adapters.NewSimAdapter(services)
	net := simulations.NewNetwork(adapter, &simulations.NetworkConfig{DefaultService: "accounting"})
	defer net.Shutdown()

	// we send msgs messages per node, wait for all messages to arrive
	bal.wg.Add(*nodes * *msgs)
	trigger := make(chan enode.ID)
	go func() {
		// wait for all of them to arrive
		bal.wg.Wait()
		// then trigger a check
		// the selected node for the trigger is irrelevant,
		// we just want to trigger the end of the simulation
		trigger <- net.Nodes[0].ID()
	}()

	// create nodes and start them
	for i := 0; i < *nodes; i++ {
		conf := adapters.RandomNodeConfig()
		bal.id2n[conf.ID] = i
		if _, err := net.NewNodeWithConfig(conf); err != nil {
			t.Fatal(err)
		}
		if err := net.Start(conf.ID); err != nil {
			t.Fatal(err)
		}
	}
	// fully connect nodes
	for i, n := range net.Nodes {
		for _, m := range net.Nodes[i+1:] {
			if err := net.Connect(n.ID(), m.ID()); err != nil {
				t.Fatal(err)
			}
		}
	}

	// empty action
	action := func(ctx context.Context) error {
		return nil
	}
	// 	check always checks out
	check := func(ctx context.Context, id enode.ID) (bool, error) {
		return true, nil
	}

	// run simulation
	timeout := 30 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	result := simulations.NewSimulation(net).Run(ctx, &simulations.Step{
		Action:  action,
		Trigger: trigger,
		Expect: &simulations.Expectation{
			Nodes: []enode.ID{net.Nodes[0].ID()},
			Check: check,
		},
	})

	if result.Error != nil {
		t.Fatal(result.Error)
	}

	// check if balance matrix is symmetric
	if err := bal.symmetric(); err != nil {
		t.Fatal(err)
	}
}

// matrix is a matrix of nodes and its balances
// matrix is in fact a linear array of size n*n,
// so the balance for any node A with B is at index
// A*n + B, while the balance of node B with A is at
// B*n + A
// (n entries in the array will not be filled -
//  the balance of a node with itself)
type matrix struct {
	n int     //number of nodes
	m []int64 //array of balances
}

// create a new matrix
func newMatrix(n int) *matrix {
	return &matrix{
		n: n,
		m: make([]int64, n*n),
	}
}

// called from the testBalance's Add accounting function: register balance change
func (m *matrix) add(i, j int, v int64) error {
	// index for the balance of local node i with remote nodde j is
	// i * number of nodes + remote node
	mi := i*m.n + j
	// register that balance
	m.m[mi] += v
	return nil
}

// check that the balances are symmetric:
// balance of node i with node j is the same as j with i but with inverted signs
func (m *matrix) symmetric() error {
	//iterate all nodes
	for i := 0; i < m.n; i++ {
		//iterate starting +1
		for j := i + 1; j < m.n; j++ {
			log.Debug("bal", "1", i, "2", j, "i,j", m.m[i*m.n+j], "j,i", m.m[j*m.n+i])
			if m.m[i*m.n+j] != -m.m[j*m.n+i] {
				return fmt.Errorf("value mismatch. m[%v, %v] = %v; m[%v, %v] = %v", i, j, m.m[i*m.n+j], j, i, m.m[j*m.n+i])
			}
		}
	}
	return nil
}

// all the balances
type balances struct {
	i int
	*matrix
	id2n map[enode.ID]int
	wg   *sync.WaitGroup
}

func newBalances(n int) *balances {
	return &balances{
		matrix: newMatrix(n),
		id2n:   make(map[enode.ID]int),
		wg:     &sync.WaitGroup{},
	}
}

// create a new testNode for every node created as part of the service
func (b *balances) newNode() *testNode {
	defer func() { b.i++ }()
	return &testNode{
		bal:   b,
		i:     b.i,
		peers: make([]*testPeer, b.n), //a node will be connected to n-1 peers
	}
}

type testNode struct {
	bal       *balances
	i         int
	lock      sync.Mutex
	peers     []*testPeer
	peerCount int
}

// do the accounting for the peer's test protocol
// testNode implements protocols.Balance
func (t *testNode) Add(a int64, p *Peer) error {
	//get the index for the remote peer
	remote := t.bal.id2n[p.ID()]
	log.Debug("add", "local", t.i, "remote", remote, "amount", a)
	return t.bal.add(t.i, remote, a)
}

//run the p2p protocol
//for every node, represented by testNode, create a remote testPeer
func (t *testNode) run(p *p2p.Peer, rw p2p.MsgReadWriter) error {
	spec := createTestSpec()
	//create accounting hook
	spec.Hook = NewAccounting(t, &dummyPrices{})

	//create a peer for this node
	tp := &testPeer{NewPeer(p, rw, spec), t.i, t.bal.id2n[p.ID()], t.bal.wg}
	t.lock.Lock()
	t.peers[t.bal.id2n[p.ID()]] = tp
	t.peerCount++
	if t.peerCount == t.bal.n-1 {
		//when all peer connections are established, start sending messages from this peer
		go t.send()
	}
	t.lock.Unlock()
	return tp.Run(tp.handle)
}

// p2p message receive handler function
func (tp *testPeer) handle(ctx context.Context, msg interface{}) error {
	tp.wg.Done()
	log.Debug("receive", "from", tp.remote, "to", tp.local, "type", reflect.TypeOf(msg), "msg", msg)
	return nil
}

type testPeer struct {
	*Peer
	local, remote int
	wg            *sync.WaitGroup
}

func (t *testNode) send() {
	log.Debug("start sending")
	for i := 0; i < *msgs; i++ {
		//determine randomly to which peer to send
		whom := rand.Intn(t.bal.n - 1)
		if whom >= t.i {
			whom++
		}
		t.lock.Lock()
		p := t.peers[whom]
		t.lock.Unlock()

		//determine a random message from the spec's messages to be sent
		which := rand.Intn(len(p.spec.Messages))
		msg := p.spec.Messages[which]
		switch msg.(type) {
		case *perBytesMsgReceiverPays:
			msg = &perBytesMsgReceiverPays{Content: content[:rand.Intn(len(content))]}
		case *perBytesMsgSenderPays:
			msg = &perBytesMsgSenderPays{Content: content[:rand.Intn(len(content))]}
		}
		log.Debug("send", "from", t.i, "to", whom, "type", reflect.TypeOf(msg), "msg", msg)
		p.Send(context.TODO(), msg)
	}
}

// define the protocol
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
