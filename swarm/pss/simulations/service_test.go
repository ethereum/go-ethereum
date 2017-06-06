package pss_simulations

import (
	"context"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	"github.com/ethereum/go-ethereum/p2p/simulations"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/pot"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/pss"
	"github.com/ethereum/go-ethereum/swarm/pss/client"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

func init() {
	h := log.CallerFileHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(true)))
	log.Root().SetHandler(h)
}

// TestPssProtocol starts a pss network along with two test nodes which run
// protocols via the pss network, connects those two test nodes and then
// waits for them to handshake
func TestPssProtocol(t *testing.T) {
	// define the services
	w := &testWrapper{}
	services := adapters.Services{
		"pss":  w.newPssService,
		"test": w.newTestService,
	}

	// create the network
	adapter := adapters.NewSimAdapter(services)
	net := simulations.NewNetwork(adapter, &simulations.NetworkConfig{
		ID: "pss",
	})
	defer net.Shutdown()
	startNode := func(service string) *simulations.Node {
		config := adapters.RandomNodeConfig()
		config.Services = []string{service}
		node, err := net.NewNodeWithConfig(config)
		if err != nil {
			t.Fatalf("error starting node: %s", err)
		}
		if err := net.Start(node.ID()); err != nil {
			t.Fatalf("error starting node %s: %s", node.ID().TerminalString(), err)
		}
		return node
	}

	// start 20 pss nodes
	nodeCount := 20
	for i := 0; i < nodeCount; i++ {
		startNode("pss")
	}

	// start two test nodes (they will use the first two pss nodes)
	node1 := startNode("test")
	node2 := startNode("test")

	// subscribe to handshake events from both nodes
	handshakes := make(chan *testHandshake, 2)
	subscribe := func(client *rpc.Client) *rpc.ClientSubscription {
		sub, err := client.Subscribe(context.Background(), "test", handshakes, "handshake")
		if err != nil {
			t.Fatal(err)
		}
		return sub
	}
	client1, err := node1.Client()
	if err != nil {
		t.Fatal(err)
	}
	sub1 := subscribe(client1)
	defer sub1.Unsubscribe()
	client2, err := node2.Client()
	if err != nil {
		t.Fatal(err)
	}
	sub2 := subscribe(client2)
	defer sub2.Unsubscribe()

	// call AddPeer on node1 with node2's pss address
	if err := client1.Call(nil, "test_addPeer", network.ToOverlayAddr(w.pssNodes[1].Bytes())); err != nil {
		t.Fatal(err)
	}

	// wait for both handshakes
	timeout := time.After(10 * time.Second)
	for i := 0; i < 2; i++ {
		select {
		case hs := <-handshakes:
			t.Logf("got handshake: %+v", hs)
		case <-timeout:
			t.Fatal("timed out waiting for handshakes")
		}
	}
}

// testWrapper creates pss and test nodes, assigning pss nodes to test
// nodes as they are started
type testWrapper struct {
	pssNodes []discover.NodeID
	index    int
}

func (t *testWrapper) newPssService(ctx *adapters.ServiceContext) (node.Service, error) {
	// track the pss node's id so we can use it for the test nodes
	t.pssNodes = append(t.pssNodes, ctx.Config.ID)

	dir, err := ioutil.TempDir("", "pss-test")
	if err != nil {
		panic(err)
	}
	dpa, err := storage.NewLocalDPA(dir)
	if err != nil {
		panic(err)
	}
	addr := network.NewAddrFromNodeID(ctx.Config.ID)
	kad := network.NewKademlia(addr.Over(), network.NewKadParams())
	return pss.NewPss(kad, dpa, pss.NewPssParams(false)), nil
}

func (t *testWrapper) newTestService(ctx *adapters.ServiceContext) (node.Service, error) {
	// connect to the next pss node
	pssNode := t.pssNodes[t.index]
	t.index++
	rpcClient, err := ctx.DialRPC(pssNode)
	if err != nil {
		panic(err)
	}
	return &testService{
		id:         ctx.Config.ID,
		pss:        client.NewPssClientWithRPC(context.Background(), rpcClient),
		handshakes: make(chan *testHandshake),
	}, nil
}

type testHandshake struct {
	ID discover.NodeID
}

// testService runs a simple handshake protocol over pss and exposes an API
// so that clients can wait for handshakes to complete
type testService struct {
	id         discover.NodeID
	pss        *client.PssClient
	handshakes chan *testHandshake
}

func (t *testService) Protocols() []p2p.Protocol {
	return nil
}

func (t *testService) APIs() []rpc.API {
	return []rpc.API{{
		Namespace: "test",
		Version:   "1.0",
		Service:   &TestAPI{t.pss, t.handshakes},
	}}
}

func (t *testService) Start(*p2p.Server) error {
	return t.pss.RunProtocol(&p2p.Protocol{
		Name:    "test",
		Version: 1,
		Run:     t.run,
	})
}

func (t *testService) Stop() error {
	return nil
}

func (t *testService) run(_ *p2p.Peer, rw p2p.MsgReadWriter) error {
	// send a handshake and wait for one back
	go p2p.Send(rw, 0, &testHandshake{t.id})
	msg, err := rw.ReadMsg()
	if err != nil {
		return err
	}
	defer msg.Discard()
	var hs testHandshake
	if err := msg.Decode(&hs); err != nil {
		return err
	}
	t.handshakes <- &hs
	return nil
}

type TestAPI struct {
	pss        *client.PssClient
	handshakes chan *testHandshake
}

func (t *TestAPI) AddPeer(addr []byte) {
	t.pss.AddPssPeer(pot.Address(common.BytesToHash(addr)), &protocols.Spec{
		Name:    "test",
		Version: 1,
	})
}

func (t *TestAPI) Handshake(ctx context.Context) (*rpc.Subscription, error) {
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return nil, rpc.ErrNotificationsUnsupported
	}
	sub := notifier.CreateSubscription()
	go func() {
		for {
			select {
			case hs := <-t.handshakes:
				notifier.Notify(sub.ID, hs)
			case <-sub.Err():
				return
			case <-notifier.Closed():
				return
			}
		}
	}()
	return sub, nil
}
