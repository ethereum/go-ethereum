package simulations

import (
	"context"
	"fmt"
	"math/rand"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/rpc"
)

type testService struct {
	id discover.NodeID

	// state stores []byte used to test creating and loading snapshots
	state atomic.Value
}

func newTestService(ctx *adapters.ServiceContext) (node.Service, error) {
	svc := &testService{id: ctx.Config.ID}
	svc.state.Store(ctx.Snapshot)
	return svc, nil
}

func (t *testService) Protocols() []p2p.Protocol {
	return []p2p.Protocol{{
		Name:    "test",
		Version: 1,
		Length:  1,
		Run:     t.Run,
	}}
}

func (t *testService) APIs() []rpc.API {
	return []rpc.API{{
		Namespace: "test",
		Version:   "1.0",
		Service:   &TestAPI{state: &t.state},
	}}
}

func (t *testService) Start(server *p2p.Server) error {
	return nil
}

func (t *testService) Stop() error {
	return nil
}

func (t *testService) Run(_ *p2p.Peer, rw p2p.MsgReadWriter) error {
	for {
		_, err := rw.ReadMsg()
		if err != nil {
			return err
		}
	}
}

func (t *testService) Snapshot() ([]byte, error) {
	return t.state.Load().([]byte), nil
}

// TestAPI provides a simple API to get and increment a counter and to
// subscribe to increment events
type TestAPI struct {
	state   *atomic.Value
	counter int64
	feed    event.Feed
}

func (t *TestAPI) Get() int64 {
	return atomic.LoadInt64(&t.counter)
}

func (t *TestAPI) Add(delta int64) {
	atomic.AddInt64(&t.counter, delta)
	t.feed.Send(delta)
}

func (t *TestAPI) GetState() []byte {
	return t.state.Load().([]byte)
}

func (t *TestAPI) SetState(state []byte) {
	t.state.Store(state)
}

func (t *TestAPI) Events(ctx context.Context) (*rpc.Subscription, error) {
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return nil, rpc.ErrNotificationsUnsupported
	}

	rpcSub := notifier.CreateSubscription()

	go func() {
		events := make(chan int64)
		sub := t.feed.Subscribe(events)
		defer sub.Unsubscribe()

		for {
			select {
			case event := <-events:
				notifier.Notify(rpcSub.ID, event)
			case <-sub.Err():
				return
			case <-rpcSub.Err():
				return
			case <-notifier.Closed():
				return
			}
		}
	}()

	return rpcSub, nil
}

var testServices = adapters.Services{
	"test": newTestService,
}

func testHTTPServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(NewServer(&ServerConfig{
		NewAdapter: func() adapters.NodeAdapter { return adapters.NewSimAdapter(testServices) },
	}))
}

// TestHTTPNetwork tests creating and interacting with a simulation
// network using the HTTP API
func TestHTTPNetwork(t *testing.T) {
	// start the server
	s := testHTTPServer(t)
	defer s.Close()

	// create a network
	client := NewClient(s.URL)
	config := &NetworkConfig{
		DefaultService: "test",
	}
	network, err := client.CreateNetwork(config)
	if err != nil {
		t.Fatalf("error creating network: %s", err)
	}

	// subscribe to events so we can check them later
	events := make(chan *Event, 100)
	sub, err := client.SubscribeNetwork(network.ID, events)
	if err != nil {
		t.Fatalf("error subscribing to network events: %s", err)
	}
	defer sub.Unsubscribe()

	// check the network has an ID
	if network.ID == "" {
		t.Fatal("expected network.ID to be set")
	}

	// check the network exists
	networks, err := client.GetNetworks()
	if err != nil {
		t.Fatalf("error getting networks: %s", err)
	}
	if len(networks) != 1 {
		t.Fatalf("expected 1 network, got %d", len(networks))
	}
	if networks[0].ID != network.ID {
		t.Fatalf("expected network to have ID %q, got %q", network.ID, networks[0].ID)
	}
	gotNetwork, err := client.GetNetwork(network.ID)
	if err != nil {
		t.Fatalf("error getting network: %s", err)
	}
	if gotNetwork.ID != network.ID {
		t.Fatalf("expected network to have ID %q, got %q", network.ID, gotNetwork.ID)
	}

	// create 2 nodes
	nodeIDs := make([]string, 2)
	for i := 0; i < 2; i++ {
		node, err := client.CreateNode(network.ID, nil)
		if err != nil {
			t.Fatalf("error creating node: %s", err)
		}
		nodeIDs[i] = node.ID
	}

	// check both nodes exist
	nodes, err := client.GetNodes(network.ID)
	if err != nil {
		t.Fatalf("error getting nodes: %s", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}
	for i, nodeID := range nodeIDs {
		if nodes[i].ID != nodeID {
			t.Fatalf("expected node %d to have ID %q, got %q", i, nodeID, nodes[i].ID)
		}
		node, err := client.GetNode(network.ID, nodeID)
		if err != nil {
			t.Fatalf("error getting node %d: %s", i, err)
		}
		if node.ID != nodeID {
			t.Fatalf("expected node %d to have ID %q, got %q", i, nodeID, node.ID)
		}
	}

	// start both nodes
	for _, nodeID := range nodeIDs {
		if err := client.StartNode(network.ID, nodeID); err != nil {
			t.Fatalf("error starting node %q: %s", nodeID, err)
		}
	}

	// connect the nodes
	if err := client.ConnectNode(network.ID, nodeIDs[0], nodeIDs[1]); err != nil {
		t.Fatalf("error connecting nodes: %s", err)
	}

	// check we got all the events
	x := &expectEvents{t, events, sub}
	x.expect(
		x.nodeEvent(nodeIDs[0], false),
		x.nodeEvent(nodeIDs[1], false),
		x.nodeEvent(nodeIDs[0], true),
		x.nodeEvent(nodeIDs[1], true),
		x.connEvent(nodeIDs[0], nodeIDs[1], false),
		x.connEvent(nodeIDs[0], nodeIDs[1], true),
	)
}

type expectEvents struct {
	*testing.T

	events chan *Event
	sub    event.Subscription
}

func (t *expectEvents) nodeEvent(id string, up bool) *Event {
	return &Event{
		Type: EventTypeNode,
		Node: &Node{
			Config: &adapters.NodeConfig{
				ID: discover.MustHexID(id),
			},
			Up: up,
		},
	}
}

func (t *expectEvents) connEvent(one, other string, up bool) *Event {
	return &Event{
		Type: EventTypeConn,
		Conn: &Conn{
			One:   discover.MustHexID(one),
			Other: discover.MustHexID(other),
			Up:    up,
		},
	}
}

func (t *expectEvents) expect(events ...*Event) {
	timeout := time.After(10 * time.Second)
	for i := 0; i < len(events); i++ {
		select {
		case event := <-t.events:
			t.Logf("received %s event: %s", event.Type, event)

			expected := events[i]
			if event.Type != expected.Type {
				t.Fatalf("expected event %d to have type %q, got %q", i, expected.Type, event.Type)
			}

			switch expected.Type {

			case EventTypeNode:
				if event.Node == nil {
					t.Fatal("expected event.Node to be set")
				}
				if event.Node.ID() != expected.Node.ID() {
					t.Fatalf("expected node event %d to have id %q, got %q", i, expected.Node.ID().TerminalString(), event.Node.ID().TerminalString())
				}
				if event.Node.Up != expected.Node.Up {
					t.Fatalf("expected node event %d to have up=%t, got up=%t", i, expected.Node.Up, event.Node.Up)
				}

			case EventTypeConn:
				if event.Conn == nil {
					t.Fatal("expected event.Conn to be set")
				}
				if event.Conn.One != expected.Conn.One {
					t.Fatalf("expected conn event %d to have one=%q, got one=%q", i, expected.Conn.One.TerminalString(), event.Conn.One.TerminalString())
				}
				if event.Conn.Other != expected.Conn.Other {
					t.Fatalf("expected conn event %d to have other=%q, got other=%q", i, expected.Conn.Other.TerminalString(), event.Conn.Other.TerminalString())
				}
				if event.Conn.Up != expected.Conn.Up {
					t.Fatalf("expected conn event %d to have up=%t, got up=%t", i, expected.Conn.Up, event.Conn.Up)
				}
			}

		case err := <-t.sub.Err():
			t.Fatalf("network stream closed unexpectedly: %s", err)

		case <-timeout:
			t.Fatal("timed out waiting for expected events")
		}
	}
}

// TestHTTPNodeRPC tests calling RPC methods on nodes via the HTTP API
func TestHTTPNodeRPC(t *testing.T) {
	// start the server
	s := testHTTPServer(t)
	defer s.Close()

	// start a node in a network
	client := NewClient(s.URL)
	network, err := client.CreateNetwork(&NetworkConfig{DefaultService: "test"})
	if err != nil {
		t.Fatalf("error creating network: %s", err)
	}
	node, err := client.CreateNode(network.ID, nil)
	if err != nil {
		t.Fatalf("error creating node: %s", err)
	}
	if err := client.StartNode(network.ID, node.ID); err != nil {
		t.Fatalf("error starting node: %s", err)
	}

	// create two RPC clients
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	rpcClient1, err := client.RPCClient(ctx, network.ID, node.ID)
	if err != nil {
		t.Fatalf("error getting node RPC client: %s", err)
	}
	rpcClient2, err := client.RPCClient(ctx, network.ID, node.ID)
	if err != nil {
		t.Fatalf("error getting node RPC client: %s", err)
	}

	// subscribe to events using client 1
	events := make(chan int64, 1)
	sub, err := rpcClient1.Subscribe(ctx, "test", events, "events")
	if err != nil {
		t.Fatalf("error subscribing to events: %s", err)
	}
	defer sub.Unsubscribe()

	// call some RPC methods using client 2
	if err := rpcClient2.CallContext(ctx, nil, "test_add", 10); err != nil {
		t.Fatalf("error calling RPC method: %s", err)
	}
	var result int64
	if err := rpcClient2.CallContext(ctx, &result, "test_get"); err != nil {
		t.Fatalf("error calling RPC method: %s", err)
	}
	if result != 10 {
		t.Fatalf("expected result to be 10, got %d", result)
	}

	// check we got an event from client 1
	select {
	case event := <-events:
		if event != 10 {
			t.Fatalf("expected event to be 10, got %d", event)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}

// TestHTTPSnapshot tests creating and loading network snapshots
func TestHTTPSnapshot(t *testing.T) {
	// start the server
	s := testHTTPServer(t)
	defer s.Close()

	// create a two-node network
	client := NewClient(s.URL)
	network, err := client.CreateNetwork(&NetworkConfig{DefaultService: "test"})
	if err != nil {
		t.Fatalf("error creating network: %s", err)
	}
	nodeCount := 2
	nodes := make([]*p2p.NodeInfo, nodeCount)
	for i := 0; i < nodeCount; i++ {
		node, err := client.CreateNode(network.ID, nil)
		if err != nil {
			t.Fatalf("error creating node: %s", err)
		}
		if err := client.StartNode(network.ID, node.ID); err != nil {
			t.Fatalf("error starting node: %s", err)
		}
		nodes[i] = node
	}
	if err := client.ConnectNode(network.ID, nodes[0].ID, nodes[1].ID); err != nil {
		t.Fatalf("error connecting nodes: %s", err)
	}

	// store some state in the test services
	states := make([]string, nodeCount)
	for i, node := range nodes {
		rpc, err := client.RPCClient(context.Background(), network.ID, node.ID)
		if err != nil {
			t.Fatalf("error getting RPC client: %s", err)
		}
		defer rpc.Close()
		state := fmt.Sprintf("%x", rand.Int())
		if err := rpc.Call(nil, "test_setState", []byte(state)); err != nil {
			t.Fatalf("error setting service state: %s", err)
		}
		states[i] = state
	}

	// create a snapshot
	snap, err := client.CreateSnapshot(network.ID)
	if err != nil {
		t.Fatalf("error creating snapshot: %s", err)
	}
	for i, state := range states {
		gotState := snap.Nodes[i].Snapshots["test"]
		if string(gotState) != state {
			t.Fatalf("expected snapshot state %q, got %q", state, gotState)
		}
	}

	// create another network
	network, err = client.CreateNetwork(&NetworkConfig{DefaultService: "test"})
	if err != nil {
		t.Fatalf("error creating network: %s", err)
	}

	// subscribe to events so we can check them later
	events := make(chan *Event, 100)
	sub, err := client.SubscribeNetwork(network.ID, events)
	if err != nil {
		t.Fatalf("error subscribing to network events: %s", err)
	}
	defer sub.Unsubscribe()

	// load the snapshot
	if err := client.LoadSnapshot(network.ID, snap); err != nil {
		t.Fatalf("error loading snapshot: %s", err)
	}

	// check the nodes and connection exists
	net, err := client.GetNetwork(network.ID)
	if err != nil {
		t.Fatalf("error getting network: %s", err)
	}
	if len(net.Nodes) != nodeCount {
		t.Fatalf("expected network to have %d nodes, got %d", nodeCount, len(net.Nodes))
	}
	for i, node := range nodes {
		id := net.Nodes[i].ID().String()
		if id != node.ID {
			t.Fatalf("expected node %d to have ID %s, got %s", i, node.ID, id)
		}
	}
	if len(net.Conns) != 1 {
		t.Fatalf("expected network to have 1 connection, got %d", len(net.Conns))
	}
	conn := net.Conns[0]
	if conn.One.String() != nodes[0].ID {
		t.Fatalf("expected connection to have one=%q, got one=%q", nodes[0].ID, conn.One)
	}
	if conn.Other.String() != nodes[1].ID {
		t.Fatalf("expected connection to have other=%q, got other=%q", nodes[1].ID, conn.Other)
	}

	// check the node states were restored
	for i, node := range nodes {
		rpc, err := client.RPCClient(context.Background(), network.ID, node.ID)
		if err != nil {
			t.Fatalf("error getting RPC client: %s", err)
		}
		defer rpc.Close()
		var state []byte
		if err := rpc.Call(&state, "test_getState"); err != nil {
			t.Fatalf("error getting service state: %s", err)
		}
		if string(state) != states[i] {
			t.Fatalf("expected snapshot state %q, got %q", states[i], state)
		}
	}

	// check we got all the events
	x := &expectEvents{t, events, sub}
	x.expect(
		x.nodeEvent(nodes[0].ID, false),
		x.nodeEvent(nodes[0].ID, true),
		x.nodeEvent(nodes[1].ID, false),
		x.nodeEvent(nodes[1].ID, true),
		x.connEvent(nodes[0].ID, nodes[1].ID, false),
		x.connEvent(nodes[0].ID, nodes[1].ID, true),
	)
}
