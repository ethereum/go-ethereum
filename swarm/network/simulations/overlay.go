// +build none

// You can run this simulation using
//
//    go run ./swarm/network/simulations/overlay.go
package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/simulations"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/swarm/network"
)

var noDiscovery = flag.Bool("no-discovery", false, "disable discovery (useful if you want to load a snapshot)")

type Simulation struct {
	mtx    sync.Mutex
	stores map[discover.NodeID]*adapters.SimStateStore
}

func NewSimulation() *Simulation {
	return &Simulation{
		stores: make(map[discover.NodeID]*adapters.SimStateStore),
	}
}

func (s *Simulation) NewService(ctx *adapters.ServiceContext) (node.Service, error) {
	id := ctx.Config.ID
	s.mtx.Lock()
	store, ok := s.stores[id]
	if !ok {
		store = adapters.NewSimStateStore()
		s.stores[id] = store
	}
	s.mtx.Unlock()

	addr := network.NewAddrFromNodeID(id)

	kp := network.NewKadParams()
	kp.MinProxBinSize = 2
	kp.MaxBinSize = 4
	kp.MinBinSize = 1
	kp.MaxRetries = 1000
	kp.RetryExponent = 2
	kp.RetryInterval = 1000000
	kp.PruneInterval = 2000
	kad := network.NewKademlia(addr.Over(), kp)
	ticker := time.NewTicker(time.Duration(kad.PruneInterval) * time.Millisecond)
	kad.Prune(ticker.C)
	hp := network.NewHiveParams()
	hp.Discovery = !*noDiscovery
	hp.KeepAliveInterval = 3 * time.Second

	config := &network.BzzConfig{
		OverlayAddr:  addr.Over(),
		UnderlayAddr: addr.Under(),
		HiveParams:   hp,
	}

	return network.NewBzz(config, kad, store), nil
}

func createMockers() map[string]*simulations.MockerConfig {
	configs := make(map[string]*simulations.MockerConfig)

	defaultCfg := simulations.DefaultMockerConfig()
	defaultCfg.ID = "start-stop"
	defaultCfg.Description = "Starts and Stops nodes in go routines"
	defaultCfg.Mocker = startStopMocker

	bootNetworkCfg := simulations.DefaultMockerConfig()
	bootNetworkCfg.ID = "bootNet"
	bootNetworkCfg.Description = "Only boots up all nodes in the config"
	bootNetworkCfg.Mocker = bootMocker

	randomNodesCfg := simulations.DefaultMockerConfig()
	randomNodesCfg.ID = "randomNodes"
	randomNodesCfg.Description = "Boots nodes and then starts and stops some picking randomly"
	randomNodesCfg.Mocker = randomMocker

	configs[defaultCfg.ID] = defaultCfg
	configs[bootNetworkCfg.ID] = bootNetworkCfg
	configs[randomNodesCfg.ID] = randomNodesCfg

	return configs
}

func setupMocker(net *simulations.Network) []discover.NodeID {
	nodeCount := 30
	ids := make([]discover.NodeID, nodeCount)
	for i := 0; i < nodeCount; i++ {
		node, err := net.NewNode()
		if err != nil {
			panic(err.Error())
		}
		ids[i] = node.ID()
	}

	for _, id := range ids {
		if err := net.Start(id); err != nil {
			panic(err.Error())
		}
		log.Debug(fmt.Sprintf("node %v starting up", id))
	}
	for i, id := range ids {
		var peerID discover.NodeID
		if i == 0 {
			peerID = ids[len(ids)-1]
		} else {
			peerID = ids[i-1]
		}
		ch := make(chan network.OverlayAddr)
		go func() {
			defer close(ch)
			ch <- network.NewAddrFromNodeID(peerID)
		}()
		if err := net.GetNode(id).Node.(*adapters.SimNode).Services()[0].(*network.Bzz).Hive.Register(ch); err != nil {
			panic(err.Error())
		}
	}

	return ids
}

func bootMocker(net *simulations.Network) {
	setupMocker(net)
}

func randomMocker(net *simulations.Network) {
	ids := setupMocker(net)

	for {
		var lowid, highid int
    var wg sync.WaitGroup
		randWait := rand.Intn(5000) + 1000
		rand1 := rand.Intn(9)
		rand2 := rand.Intn(9)
		if rand1 < rand2 {
			lowid = rand1
			highid = rand2
		} else if rand1 > rand2 {
			highid = rand1
			lowid = rand2
		} else {
			if rand1 == 0 {
				rand2 = 9
			} else if rand1 == 9 {
				rand1 = 0
			}
      lowid = rand1
      highid = rand2
		}
    var steps = highid - lowid
    wg.Add(steps)
		for i := lowid; i < highid; i++ {
			log.Info(fmt.Sprintf("node %v shutting down", ids[i]))
			net.Stop(ids[i])
			go func(id discover.NodeID) {
				time.Sleep(time.Duration(randWait) * time.Millisecond)
				net.Start(id)
        wg.Done()
			}(ids[i])
			time.Sleep(time.Duration(randWait) * time.Millisecond)
		}
    wg.Wait()
	}
}

func startStopMocker(net *simulations.Network) {
	ids := setupMocker(net)

	for range time.Tick(10 * time.Second) {
		id := ids[rand.Intn(len(ids))]
		go func() {
			log.Error("stopping node", "id", id)
			if err := net.Stop(id); err != nil {
				log.Error("error stopping node", "id", id, "err", err)
				return
			}

			time.Sleep(3 * time.Second)

			log.Error("starting node", "id", id)
			if err := net.Start(id); err != nil {
				log.Error("error starting node", "id", id, "err", err)
				return
			}
		}()
	}
}

// var server
func main() {
	flag.Parse()

	runtime.GOMAXPROCS(runtime.NumCPU())

	log.Root().SetHandler(log.LvlFilterHandler(log.LvlInfo, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))

	s := NewSimulation()
	services := adapters.Services{
		"overlay": s.NewService,
	}
	adapter := adapters.NewSimAdapter(services)

	network := simulations.NewNetwork(adapter, &simulations.NetworkConfig{
		DefaultService: "overlay",
	})

	mockers := createMockers()

	config := simulations.ServerConfig{
		DefaultMockerID: "randomNodes",
		// DefaultMockerID: "bootNet",
		Mockers: mockers,
	}

	log.Info("starting simulation server on 0.0.0.0:8888...")
	http.ListenAndServe(":8888", simulations.NewServer(network, config))
}
