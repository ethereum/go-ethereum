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

// You can run this simulation using
//
//    go run ./swarm/network/simulations/overlay.go
package main

import (
	"flag"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/simulations"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/state"
	colorable "github.com/mattn/go-colorable"
)

var (
	noDiscovery = flag.Bool("no-discovery", false, "disable discovery (useful if you want to load a snapshot)")
	vmodule     = flag.String("vmodule", "", "log filters for logger via Vmodule")
	verbosity   = flag.Int("verbosity", 0, "log filters for logger via Vmodule")
	httpSimPort = 8888
)

func init() {
	flag.Parse()
	//initialize the logger
	//this is a demonstration on how to use Vmodule for filtering logs
	//provide -vmodule as param, and comma-separated values, e.g.:
	//-vmodule overlay_test.go=4,simulations=3
	//above examples sets overlay_test.go logs to level 4, while packages ending with "simulations" to 3
	if *vmodule != "" {
		//only enable the pattern matching handler if the flag has been provided
		glogger := log.NewGlogHandler(log.StreamHandler(colorable.NewColorableStderr(), log.TerminalFormat(true)))
		if *verbosity > 0 {
			glogger.Verbosity(log.Lvl(*verbosity))
		}
		glogger.Vmodule(*vmodule)
		log.Root().SetHandler(glogger)
	}
}

type Simulation struct {
	mtx    sync.Mutex
	stores map[discover.NodeID]*state.InmemoryStore
}

func NewSimulation() *Simulation {
	return &Simulation{
		stores: make(map[discover.NodeID]*state.InmemoryStore),
	}
}

func (s *Simulation) NewService(ctx *adapters.ServiceContext) (node.Service, error) {
	id := ctx.Config.ID
	s.mtx.Lock()
	store, ok := s.stores[id]
	if !ok {
		store = state.NewInmemoryStore()
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

//create the simulation network
func newSimulationNetwork() *simulations.Network {

	s := NewSimulation()
	services := adapters.Services{
		"overlay": s.NewService,
	}
	adapter := adapters.NewSimAdapter(services)
	simNetwork := simulations.NewNetwork(adapter, &simulations.NetworkConfig{
		DefaultService: "overlay",
	})
	return simNetwork
}

//return a new http server
func newOverlaySim(sim *simulations.Network) *simulations.Server {
	return simulations.NewServer(sim)
}

// var server
func main() {
	//cpu optimization
	runtime.GOMAXPROCS(runtime.NumCPU())
	//run the sim
	runOverlaySim()
}

func runOverlaySim() {
	//create the simulation network
	net := newSimulationNetwork()
	//create a http server with it
	sim := newOverlaySim(net)
	log.Info(fmt.Sprintf("starting simulation server on 0.0.0.0:%d...", httpSimPort))
	//start the HTTP server
	http.ListenAndServe(fmt.Sprintf(":%d", httpSimPort), sim)
}
