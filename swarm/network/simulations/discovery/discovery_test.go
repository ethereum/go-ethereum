package discovery_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	p2pnode "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/simulations"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/network"
)

// serviceName is used with the exec adapter so the exec'd binary knows which
// service to execute
const serviceName = "discovery"

var services = adapters.Services{
	serviceName: func(id *adapters.NodeId) p2pnode.Service {
		return newNode(id)
	},
}

func init() {
	// register the discovery service which will run as a devp2p
	// protocol when using the exec adapter
	adapters.RegisterServices(services)

	// log.Root().SetHandler(log.LvlFilterHandler(log.LvlError, log.StreamHandler(os.Stderr, log.TerminalFormat(false))))
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlTrace, log.StreamHandler(os.Stderr, log.TerminalFormat(false))))
}

func TestDiscoverySimulationDockerAdapter(t *testing.T) {
	adapter, err := adapters.NewDockerAdapter()
	if err != nil {
		t.Fatal(err)
	}
	testDiscoverySimulation(t, adapter)
}

func TestDiscoverySimulationExecAdapter(t *testing.T) {
	baseDir, err := ioutil.TempDir("", "swarm-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(baseDir)
	testDiscoverySimulation(t, adapters.NewExecAdapter(baseDir))
}

func TestDiscoverySimulationSimAdapter(t *testing.T) {
	testDiscoverySimulation(t, adapters.NewSimAdapter(services))
}

func testDiscoverySimulation(t *testing.T, adapter adapters.NodeAdapter) {
	// create 10 node network
	nodeCount := 10
	net := simulations.NewNetwork(adapter, &simulations.NetworkConfig{
		Id:             "0",
		Backend:        true,
		DefaultService: serviceName,
	})
	defer net.Shutdown()
	trigger := make(chan *adapters.NodeId)
	ids := make([]*adapters.NodeId, nodeCount)
	for i := 0; i < nodeCount; i++ {
		conf, err := net.NewNode()
		if err != nil {
			t.Fatalf("error starting node %s: %s", conf.Id.Label(), err)
		}
		if err := net.Start(conf.Id); err != nil {
			t.Fatalf("error starting node %s: %s", conf.Id.Label(), err)
		}
		if err := triggerChecks(trigger, net, conf.Id); err != nil {
			t.Fatal("error triggering checks for node %s: %s", conf.Id.Label(), err)
		}
		ids[i] = conf.Id
	}

	// run a simulation which connects the 10 nodes in a ring and waits
	// for full peer discovery
	action := func(ctx context.Context) error {
		for i, id := range ids {
			var peerId *adapters.NodeId
			if i == 0 {
				peerId = ids[len(ids)-1]
			} else {
				peerId = ids[i-1]
			}
			if err := net.Connect(id, peerId); err != nil {
				return err
			}
		}
		return nil
	}
	check := func(ctx context.Context, id *adapters.NodeId) (bool, error) {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
		}

		node := net.GetNode(id)
		if node == nil {
			return false, fmt.Errorf("unknown node: %s", id)
		}
		client, err := node.Client()
		if err != nil {
			return false, fmt.Errorf("error getting node client: %s", err)
		}
		var healthy bool
		if err := client.Call(&healthy, "hive_healthy", nil); err != nil {
			return false, fmt.Errorf("error getting node health: %s", err)
		}
		return healthy, nil
	}

	timeout := 30 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	result := simulations.NewSimulation(net).Run(ctx, &simulations.Step{
		Action:  action,
		Trigger: trigger,
		Expect: &simulations.Expectation{
			Nodes: ids,
			Check: check,
		},
	})
	if result.Error != nil {
		t.Fatalf("simulation failed: %s", result.Error)
	}

	t.Log("Simulation Passed:")
	t.Logf("Duration: %s", result.FinishedAt.Sub(result.StartedAt))
	for _, id := range ids {
		t.Logf("Node %s passed in %s", id.Label(), result.Passes[id].Sub(result.StartedAt))
	}
	t.Logf("Events:")
	for _, event := range result.NetworkEvents {
		t.Log(event)
	}
}

// triggerChecks triggers a simulation step check whenever a peer is added or
// removed from the given node
func triggerChecks(trigger chan *adapters.NodeId, net *simulations.Network, id *adapters.NodeId) error {
	node := net.GetNode(id)
	if node == nil {
		return fmt.Errorf("unknown node: %s", id)
	}
	client, err := node.Client()
	if err != nil {
		return err
	}
	events := make(chan *p2p.PeerEvent)
	sub, err := client.EthSubscribe(context.Background(), events, "peerEvents")
	if err != nil {
		return fmt.Errorf("error getting peer events for node %v: %s", id, err)
	}
	go func() {
		defer sub.Unsubscribe()
		for {
			select {
			case <-events:
				trigger <- id
			case err := <-sub.Err():
				if err != nil {
					log.Error(fmt.Sprintf("error getting peer events for node %v", id), "err", err)
				}
				return
			}
		}
	}()
	return nil
}

type node struct {
	*network.Hive

	protocol *p2p.Protocol
}

func newNode(id *adapters.NodeId) *node {
	addr := network.NewPeerAddrFromNodeId(id)
	kademlia := newKademlia(addr.OverlayAddr())
	hive := newHive(kademlia)
	codeMap := network.BzzCodeMap(network.DiscoveryMsgs...)
	node := &node{Hive: hive}
	services := func(peer network.Peer) error {
		discoveryPeer := network.NewDiscovery(peer, kademlia)
		node.Add(discoveryPeer)
		peer.DisconnectHook(func(err error) {
			node.Remove(discoveryPeer)
		})
		return nil
	}
	node.protocol = network.Bzz(addr.OverlayAddr(), addr.UnderlayAddr(), codeMap, services, nil, nil)
	return node
}

func newKademlia(overlayAddr []byte) *network.Kademlia {
	params := network.NewKadParams()
	params.MinProxBinSize = 2
	params.MaxBinSize = 3
	params.MinBinSize = 1
	params.MaxRetries = 1000
	params.RetryExponent = 2
	params.RetryInterval = 1000000

	return network.NewKademlia(overlayAddr, params)
}

func newHive(kademlia *network.Kademlia) *network.Hive {
	params := network.NewHiveParams()
	params.CallInterval = 5000

	return network.NewHive(params, kademlia)
}

func (n *node) Protocols() []p2p.Protocol {
	return []p2p.Protocol{*n.protocol}
}

func (n *node) APIs() []rpc.API {
	return []rpc.API{{
		Namespace: "hive",
		Version:   "1.0",
		Service:   n.Hive,
	}}
}

func (n *node) Start(server p2p.Server) error {
	return n.Hive.Start(server, n.hiveKeepAlive)
}

func (n *node) Stop() error {
	n.Hive.Stop()
	return nil
}

func (n *node) hiveKeepAlive() <-chan time.Time {
	return time.Tick(time.Second)
}
