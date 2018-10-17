package protocols

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
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
	nodes    = flag.Int("nodes", 30, "number of nodes to create (default 10)")
	msgs     = flag.Int("msgs", 100, "number of messages sent by node (default 10)")
	loglevel = flag.Int("loglevel", 0, "verbosity of logs")
	rawlog   = flag.Bool("rawlog", false, "remove terminal formatting from logs")
)

func init() {
	flag.Parse()
	log.PrintOrigins(true)
	log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(*loglevel), log.StreamHandler(colorable.NewColorableStderr(), log.TerminalFormat(!*rawlog))))
}

func TestAccountingSimulation(t *testing.T) {
	bal := newBalances(*nodes)
	services := adapters.Services{
		"accounting": func(ctx *adapters.ServiceContext) (node.Service, error) {
			return bal.newNode(), nil
		},
	}
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

	// check if balance matrix is symmetric
	if err := bal.symmetric(); err != nil {
		t.Fatal(err)
	}
	if result.Error != nil {
		t.Fatal(result.Error)
	}

	// check if balance matrix is symmetric
	if err := bal.symmetric(); err != nil {
		t.Fatal(err)
	}
}

type matrix struct {
	n int
	m []int64
}

func newMatrix(n int) *matrix {
	return &matrix{
		n: n,
		m: make([]int64, n*n),
	}
}

func (m *matrix) add(i, j int, v int64) error {
	mi := i*m.n + j
	m.m[mi] += v
	return nil
}

func (m *matrix) symmetric() error {
	for i := 0; i < m.n; i++ {
		for j := i + 1; j < m.n; j++ {
			log.Warn("bal", "1", i, "2", j, "i,j", m.m[i*m.n+j], "j,i", m.m[j*m.n+i])
			if m.m[i*m.n+j] != -m.m[j*m.n+i] {
				return fmt.Errorf("value mismatch. m[%v, %v] = %v; m[%v, %v] = %v", i, j, m.m[i*m.n+j], j, i, m.m[j*m.n+i])
			}
		}
	}
	return nil
}

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

func (b *balances) newNode() *testNode {
	defer func() { b.i++ }()
	return &testNode{
		bal:   b,
		i:     b.i,
		peers: make([]*testPeer, b.n),
	}
}

type testNode struct {
	bal       *balances
	i         int
	lock      sync.Mutex
	peers     []*testPeer
	peerCount int
}

func (t *testNode) Add(a int64, p *Peer) error {
	remote := t.bal.id2n[p.ID()]
	log.Warn("add", "local", t.i, "remote", remote, "amount", a)
	return t.bal.add(t.i, remote, a)
}

func (t *testNode) run(p *p2p.Peer, rw p2p.MsgReadWriter) error {
	spec := createTestSpec()
	spec.Hook = NewAccounting(t, &dummyPrices{})

	tp := &testPeer{NewPeer(p, rw, spec), t.i, t.bal.id2n[p.ID()], t.bal.wg}
	t.lock.Lock()
	t.peers[t.bal.id2n[p.ID()]] = tp
	t.peerCount++
	if t.peerCount == t.bal.n-1 {
		go t.send()
	}
	t.lock.Unlock()
	return tp.Run(tp.handle)
}

func (tp *testPeer) handle(ctx context.Context, msg interface{}) error {
	tp.wg.Done()
	log.Warn("receive", "from", tp.remote, "to", tp.local, "type", reflect.TypeOf(msg), "msg", msg)
	return nil
}

type testPeer struct {
	*Peer
	local, remote int
	wg            *sync.WaitGroup
}

func (t *testNode) send() {
	log.Warn("start sending")
	for i := 0; i < *msgs; i++ {
		whom := rand.Intn(t.bal.n - 1)
		if whom >= t.i {
			whom++
		}
		t.lock.Lock()
		p := t.peers[whom]
		t.lock.Unlock()

		which := rand.Intn(len(p.spec.Messages))
		msg := p.spec.Messages[which]
		switch msg.(type) {
		case *perBytesMsgReceiverPays:
			msg = &perBytesMsgReceiverPays{Content: content[:rand.Intn(len(content))]}
		case *perBytesMsgSenderPays:
			msg = &perBytesMsgSenderPays{Content: content[:rand.Intn(len(content))]}
		}
		log.Warn("send", "from", t.i, "to", whom, "type", reflect.TypeOf(msg), "msg", msg)
		p.Send(context.TODO(), msg)
	}
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
