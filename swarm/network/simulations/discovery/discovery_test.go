package discovery

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/simulations"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/swarm/network"
)

// serviceName is used with the exec adapter so the exec'd binary knows which
// service to execute
const serviceName = "discovery"
const testMinProxBinSize = 2

var services = adapters.Services{
	serviceName: newService,
}

var (
	nodeCount    = flag.Int("nodes", 16, "number of nodes to create (default 10)")
	initCount    = flag.Int("conns", 1, "number of originally connected peers	 (default 1)")
	snapshotFile = flag.String("snapshot", "", "create snapshot")
	loglevel     = flag.Int("loglevel", 3, "verbosity of logs")
)

func init() {
	flag.Parse()
	// register the discovery service which will run as a devp2p
	// protocol when using the exec adapter
	adapters.RegisterServices(services)

	log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(*loglevel), log.StreamHandler(os.Stderr, log.TerminalFormat(false))))
}

// Benchmarks to test the average time it takes for an N-node ring
// to full a healthy kademlia topology
func BenchmarkDiscovery_8_1(b *testing.B)   { benchmarkDiscovery(b, 8, 1) }
func BenchmarkDiscovery_16_1(b *testing.B)  { benchmarkDiscovery(b, 16, 1) }
func BenchmarkDiscovery_32_1(b *testing.B)  { benchmarkDiscovery(b, 32, 1) }
func BenchmarkDiscovery_64_1(b *testing.B)  { benchmarkDiscovery(b, 64, 1) }
func BenchmarkDiscovery_128_1(b *testing.B) { benchmarkDiscovery(b, 128, 1) }
func BenchmarkDiscovery_256_1(b *testing.B) { benchmarkDiscovery(b, 256, 1) }

func BenchmarkDiscovery_8_2(b *testing.B)   { benchmarkDiscovery(b, 8, 2) }
func BenchmarkDiscovery_16_2(b *testing.B)  { benchmarkDiscovery(b, 16, 2) }
func BenchmarkDiscovery_32_2(b *testing.B)  { benchmarkDiscovery(b, 32, 2) }
func BenchmarkDiscovery_64_2(b *testing.B)  { benchmarkDiscovery(b, 64, 2) }
func BenchmarkDiscovery_128_2(b *testing.B) { benchmarkDiscovery(b, 128, 2) }
func BenchmarkDiscovery_256_2(b *testing.B) { benchmarkDiscovery(b, 256, 2) }

func BenchmarkDiscovery_8_4(b *testing.B)   { benchmarkDiscovery(b, 8, 4) }
func BenchmarkDiscovery_16_4(b *testing.B)  { benchmarkDiscovery(b, 16, 4) }
func BenchmarkDiscovery_32_4(b *testing.B)  { benchmarkDiscovery(b, 32, 4) }
func BenchmarkDiscovery_64_4(b *testing.B)  { benchmarkDiscovery(b, 64, 4) }
func BenchmarkDiscovery_128_4(b *testing.B) { benchmarkDiscovery(b, 128, 4) }
func BenchmarkDiscovery_256_4(b *testing.B) { benchmarkDiscovery(b, 256, 4) }

func TestDiscoverySimulationDockerAdapter(t *testing.T) {
	testDiscoverySimulationDockerAdapter(t, *nodeCount, *initCount)
}

func testDiscoverySimulationDockerAdapter(t *testing.T, nodes, conns int) {
	adapter, err := adapters.NewDockerAdapter()
	if err != nil {
		if err == adapters.ErrLinuxOnly {
			t.Skip(err)
		} else {
			t.Fatal(err)
		}
	}
	testDiscoverySimulation(t, nodes, conns, adapter)
}

func TestDiscoverySimulationExecAdapter(t *testing.T) {
	testDiscoverySimulationExecAdapter(t, *nodeCount, *initCount)
}

func testDiscoverySimulationExecAdapter(t *testing.T, nodes, conns int) {
	baseDir, err := ioutil.TempDir("", "swarm-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(baseDir)
	testDiscoverySimulation(t, nodes, conns, adapters.NewExecAdapter(baseDir))
}

func TestDiscoverySimulationSimAdapter(t *testing.T) {
	testDiscoverySimulationSimAdapter(t, *nodeCount, *initCount)
}

func testDiscoverySimulationSimAdapter(t *testing.T, nodes, conns int) {
	testDiscoverySimulation(t, nodes, conns, adapters.NewSocketAdapter(services))
	// testDiscoverySimulation(t, nodes, conns, adapters.NewSimAdapter(services))
}

func testDiscoverySimulation(t *testing.T, nodes, conns int, adapter adapters.NodeAdapter) {
	startedAt := time.Now()
	result, err := discoverySimulation(nodes, conns, adapter)
	if err != nil {
		t.Fatalf("Setting up simulation failed: %v", err)
	}
	if result.Error != nil {
		t.Fatalf("Simulation failed: %s", result.Error)
	}
	t.Logf("Simulation with %d nodes passed in %s", nodes, result.FinishedAt.Sub(result.StartedAt))
	var min, max time.Duration
	var sum int
	for _, pass := range result.Passes {
		duration := pass.Sub(result.StartedAt)
		if sum == 0 || duration < min {
			min = duration
		}
		if duration > max {
			max = duration
		}
		sum += int(duration.Nanoseconds())
	}
	t.Logf("Min: %s, Max: %s, Average: %s", min, max, time.Duration(sum/len(result.Passes))*time.Nanosecond)
	finishedAt := time.Now()
	t.Logf("Setup: %s, shutdown: %s", result.StartedAt.Sub(startedAt), finishedAt.Sub(result.FinishedAt))
}

func benchmarkDiscovery(b *testing.B, nodes, conns int) {
	for i := 0; i < b.N; i++ {
		result, err := discoverySimulation(nodes, conns, adapters.NewSimAdapter(services))
		if err != nil {
			b.Fatalf("setting up simulation failed", result)
		}
		if result.Error != nil {
			b.Logf("simulation failed: %s", result.Error)
		}
	}
}

func discoverySimulation(nodes, conns int, adapter adapters.NodeAdapter) (*simulations.StepResult, error) {
	// create network
	net := simulations.NewNetwork(adapter, &simulations.NetworkConfig{
		ID:             "0",
		DefaultService: serviceName,
	})
	defer net.Shutdown()
	trigger := make(chan discover.NodeID)
	ids := make([]discover.NodeID, nodes)
	for i := 0; i < nodes; i++ {
		node, err := net.NewNode()
		if err != nil {
			return nil, fmt.Errorf("error starting node: %s", err)
		}
		if err := net.Start(node.ID()); err != nil {
			return nil, fmt.Errorf("error starting node %s: %s", node.ID().TerminalString(), err)
		}
		if err := triggerChecks(trigger, net, node.ID()); err != nil {
			return nil, fmt.Errorf("error triggering checks for node %s: %s", node.ID().TerminalString(), err)
		}
		ids[i] = node.ID()
	}

	// run a simulation which connects the 10 nodes in a ring and waits
	// for full peer discovery
	var addrs [][]byte
	action := func(ctx context.Context) error {
		return nil
	}
	wg := sync.WaitGroup{}
	for i := range ids {
		// collect the overlay addresses, to
		addrs = append(addrs, network.ToOverlayAddr(ids[i].Bytes()))
		for j := 0; j < conns; j++ {
			var k int
			if j == 0 {
				k = (i + 1) % len(ids)
			} else {
				k = rand.Intn(len(ids))
			}
			wg.Add(1)
			go func(i, k int) {
				defer wg.Done()
				net.Connect(ids[i], ids[k])
			}(i, k)
		}
	}
	wg.Wait()
	log.Debug(fmt.Sprintf("nodes: %v", len(addrs)))
	// construct the peer pot, so that kademlia health can be checked
	ppmap := network.NewPeerPot(testMinProxBinSize, ids, addrs)
	check := func(ctx context.Context, id discover.NodeID) (bool, error) {
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
		healthy := &network.Health{}
		if err := client.Call(&healthy, "hive_healthy", ppmap[id]); err != nil {
			return false, fmt.Errorf("error getting node health: %s", err)
		}
		log.Debug(fmt.Sprintf("node %4s healthy: got nearest neighbours: %v, know nearest neighbours: %v, saturated: %v\n%v", id, healthy.GotNN, healthy.KnowNN, healthy.Full, healthy.Hive))
		return healthy.KnowNN && healthy.GotNN && healthy.Full, nil
	}

	// 64 nodes ~ 1min
	// 128 nodes ~
	timeout := 300 * time.Second
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
		return result, nil
	}

	if *snapshotFile != "" {
		snap, err := net.Snapshot()
		if err != nil {
			return nil, errors.New("no shapshot dude")
		}
		jsonsnapshot, err := json.Marshal(snap)
		if err != nil {
			return nil, fmt.Errorf("corrupt json snapshot: %v", err)
		}
		log.Info("writing snapshot", "file", *snapshotFile)
		err = ioutil.WriteFile(*snapshotFile, jsonsnapshot, 0755)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

// triggerChecks triggers a simulation step check whenever a peer is added or
// removed from the given node, and also every second to avoid a race between
// peer events and kademlia becoming healthy
func triggerChecks(trigger chan discover.NodeID, net *simulations.Network, id discover.NodeID) error {
	node := net.GetNode(id)
	if node == nil {
		return fmt.Errorf("unknown node: %s", id)
	}
	client, err := node.Client()
	if err != nil {
		return err
	}
	events := make(chan *p2p.PeerEvent)
	sub, err := client.Subscribe(context.Background(), "admin", events, "peerEvents")
	if err != nil {
		return fmt.Errorf("error getting peer events for node %v: %s", id, err)
	}
	go func() {
		defer sub.Unsubscribe()

		tick := time.NewTicker(time.Second)
		defer tick.Stop()

		for {
			select {
			case <-events:
				trigger <- id
			case <-tick.C:
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

func newService(ctx *adapters.ServiceContext) (node.Service, error) {
	addr := network.NewAddrFromNodeID(ctx.Config.ID)

	kp := network.NewKadParams()
	kp.MinProxBinSize = testMinProxBinSize
	kp.MaxBinSize = 3
	kp.MinBinSize = 1
	kp.MaxRetries = 1000
	kp.RetryExponent = 2
	kp.RetryInterval = 50000000

	if ctx.Config.Reachable != nil {
		kp.Reachable = func(o network.OverlayAddr) bool {
			return ctx.Config.Reachable(o.(*network.BzzAddr).ID())
		}
	}
	kad := network.NewKademlia(addr.Over(), kp)

	hp := network.NewHiveParams()
	hp.KeepAliveInterval = 200 * time.Millisecond

	config := &network.BzzConfig{
		OverlayAddr:  addr.Over(),
		UnderlayAddr: addr.Under(),
		HiveParams:   hp,
	}

	return network.NewBzz(config, kad, nil), nil
}
