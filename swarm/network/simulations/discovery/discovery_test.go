package discovery_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/adapters"
	"github.com/ethereum/go-ethereum/p2p/simulations"
	"github.com/ethereum/go-ethereum/swarm/network"
)

func TestDiscoverySimulation(t *testing.T) {
	// create 10 node network
	nodeCount := 10
	trigger := make(chan *adapters.NodeId)
	net := simulations.NewNetwork(&simulations.NetworkConfig{
		Id:      "0",
		Backend: true,
	})
	nodes := make(map[*adapters.NodeId]*node, nodeCount)
	net.SetNaf(func(conf *simulations.NodeConfig) adapters.NodeAdapter {
		node := newNode(conf.Id, net, trigger)
		nodes[conf.Id] = node
		return node
	})
	ids := adapters.RandomNodeIds(nodeCount)
	for _, id := range ids {
		net.NewNode(&simulations.NodeConfig{Id: id})
		if err := net.Start(id); err != nil {
			t.Fatalf("error starting node %s: %s", id.Label(), err)
		}
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

		node, ok := nodes[id]
		if !ok {
			return false, fmt.Errorf("unknown node: %s", id)
		}

		// TODO: check list of peers
		_ = node

		return true, nil
	}

	timeout := 10 * time.Second
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

type node struct {
	*network.Hive
	adapters.NodeAdapter

	id      *adapters.NodeId
	network *simulations.Network
	trigger chan *adapters.NodeId
}

func newNode(id *adapters.NodeId, net *simulations.Network, trigger chan *adapters.NodeId) *node {
	addr := network.NewPeerAddrFromNodeId(id)
	kademlia := newKademlia(addr.OverlayAddr())
	hive := newHive(kademlia)
	codeMap := network.BzzCodeMap(network.DiscoveryMsgs...)
	nodeAdapter := adapters.NewSimNode(id, net)
	node := &node{
		Hive:        hive,
		NodeAdapter: nodeAdapter,
		id:          id,
		network:     net,
		trigger:     trigger,
	}
	services := func(peer network.Peer) error {
		discoveryPeer := network.NewDiscovery(peer, kademlia)
		node.Add(discoveryPeer)
		peer.DisconnectHook(func(err error) {
			node.Remove(discoveryPeer)
		})
		return nil
	}
	nodeAdapter.Run = network.Bzz(addr.OverlayAddr(), nodeAdapter, codeMap, services, nil, nil).Run
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

func (n *node) Start() error {
	return n.Hive.Start(n.connectPeer, n.hiveKeepAlive)
}

func (n *node) Stop() error {
	n.Hive.Stop()
	return nil
}

func (n *node) Add(peer network.Peer) error {
	err := n.Hive.Add(peer)
	n.triggerCheck()
	return err
}

func (n *node) Remove(peer network.Peer) {
	n.Hive.Remove(peer)
	n.triggerCheck()
}

func (n *node) RunProtocol(id *adapters.NodeId, rw, rrw p2p.MsgReadWriter, peer *adapters.Peer) error {
	return n.NodeAdapter.(adapters.ProtocolRunner).RunProtocol(id, rw, rrw, peer)
}

func (n *node) connectPeer(s string) error {
	return n.network.Connect(n.id, adapters.NewNodeIdFromHex(s))
}

func (n *node) hiveKeepAlive() <-chan time.Time {
	return time.Tick(time.Second)
}

func (n *node) triggerCheck() {
	// TODO: rate limit the trigger?
	go func() { n.trigger <- n.id }()
}
