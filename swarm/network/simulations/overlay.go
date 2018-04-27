// +build none

// You can run this simulation using
//
//    go run ./swarm/network/simulations/overlay.go
package main

import (
	"flag"
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
	kad := network.NewKademlia(addr.Over(), kp)
	hp := network.NewHiveParams()
	hp.Discovery = !*noDiscovery
	hp.KeepAliveInterval = 300 * time.Millisecond

	config := &network.BzzConfig{
		OverlayAddr:  addr.Over(),
		UnderlayAddr: addr.Under(),
		HiveParams:   hp,
	}

	return network.NewBzz(config, kad, store, nil, nil), nil
}

// var server
func main() {
	flag.Parse()

	runtime.GOMAXPROCS(runtime.NumCPU())

	log.Root().SetHandler(log.LvlFilterHandler(log.LvlInfo, log.StreamHandler(os.Stderr, log.TerminalFormat(false))))

	s := NewSimulation()
	services := adapters.Services{
		"overlay": s.NewService,
	}
	adapter := adapters.NewSimAdapter(services)
	network := simulations.NewNetwork(adapter, &simulations.NetworkConfig{
		DefaultService: "overlay",
	})

	log.Info("starting simulation server on 0.0.0.0:8888...")
	http.ListenAndServe(":8888", simulations.NewServer(network))
}
