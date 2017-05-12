// +build none

// You can run this simulation using
//
//    go run ./swarm/network/simulations/overlay.go
package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/simulations"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/network"
)

// SimNode is the adapter used by Swarm simulations.
type SimNode struct {
	hive     *network.Hive
	protocol *p2p.Protocol
}

func (s *SimNode) Protocols() []p2p.Protocol {
	return []p2p.Protocol{*s.protocol}
}

func (s *SimNode) APIs() []rpc.API {
	return nil
}

// the hive update ticker for hive
func af() <-chan time.Time {
	return time.NewTicker(1 * time.Second).C
}

// Start() starts up the hive
// makes SimNode implement node.Service
func (self *SimNode) Start(server p2p.Server) error {
	return self.hive.Start(server, af)
}

// Stop() shuts down the hive
// makes SimNode implement node.Service
func (self *SimNode) Stop() error {
	self.hive.Stop()
	return nil
}

// NewSimNode creates adapters for nodes in the simulation.
func NewSimNode(id *adapters.NodeId, snapshot []byte) node.Service {
	addr := network.NewPeerAddrFromNodeId(id)
	kp := network.NewKadParams()

	kp.MinProxBinSize = 2
	kp.MaxBinSize = 3
	kp.MinBinSize = 1
	kp.MaxRetries = 1000
	kp.RetryExponent = 2
	kp.RetryInterval = 1000000

	to := network.NewKademlia(addr.OverlayAddr(), kp) // overlay topology driver
	hp := network.NewHiveParams()
	hp.CallInterval = 5000
	pp := network.NewHive(hp, to) // hive

	services := func(p network.Peer) error {
		dp := network.NewDiscovery(p, to)
		pp.Add(dp)
		log.Trace(fmt.Sprintf("kademlia on %v", dp))
		p.DisconnectHook(func(err error) {
			pp.Remove(dp)
		})
		return nil
	}

	ct := network.BzzCodeMap(network.DiscoveryMsgs...) // bzz protocol code map
	nodeInfo := func() interface{} { return pp.String() }

	return &SimNode{
		hive:     pp,
		protocol: network.Bzz(addr.OverlayAddr(), addr.UnderlayAddr(), ct, services, nil, nodeInfo),
	}
}

func createMockers() map[string]*simulations.MockerConfig {
	configs := make(map[string]*simulations.MockerConfig)

	defaultCfg := simulations.DefaultMockerConfig()
	defaultCfg.Id = "start-stop"
	defaultCfg.Description = "Starts and Stops nodes in go routines"
	defaultCfg.Mocker = startStopMocker

	bootNetworkCfg := simulations.DefaultMockerConfig()
	bootNetworkCfg.Id = "bootNet"
	bootNetworkCfg.Description = "Only boots up all nodes in the config"
	bootNetworkCfg.Mocker = bootMocker

	randomNodesCfg := simulations.DefaultMockerConfig()
	randomNodesCfg.Id = "randomNodes"
	randomNodesCfg.Description = "Boots nodes and then starts and stops some picking randomly"
	randomNodesCfg.Mocker = randomMocker

	configs[defaultCfg.Id] = defaultCfg
	configs[bootNetworkCfg.Id] = bootNetworkCfg
	configs[randomNodesCfg.Id] = randomNodesCfg

	return configs
}

func setupMocker(net *simulations.Network) []*adapters.NodeId {
	conf := net.Config()
	conf.DefaultService = "overlay"

	ids := make([]*adapters.NodeId, 10)
	for i := 0; i < 10; i++ {
		node, err := net.NewNode()
		if err != nil {
			panic(err.Error())
		}
		ids[i] = node.ID()
	}

	for _, id := range ids {
		n := rand.Intn(1000)
		time.Sleep(time.Duration(n) * time.Millisecond)
		if err := net.Start(id); err != nil {
			panic(err.Error())
		}
		log.Debug(fmt.Sprintf("node %v starting up", id))
	}
	for i, id := range ids {
		var peerId *adapters.NodeId
		if i == 0 {
			peerId = ids[len(ids)-1]
		} else {
			peerId = ids[i-1]
		}
		if err := net.Connect(id, peerId); err != nil {
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
		}
		for i := lowid; i < highid; i++ {
			log.Debug(fmt.Sprintf("node %v shutting down", ids[i]))
			net.Stop(ids[i])
			go func(id *adapters.NodeId) {
				time.Sleep(time.Duration(randWait) * time.Millisecond)
				net.Start(id)
			}(ids[i])
			time.Sleep(time.Duration(randWait) * time.Millisecond)
		}
	}
}

func startStopMocker(net *simulations.Network) {
	ids := setupMocker(net)

	for i, id := range ids {
		n := 3000 + i*1000
		go func(id *adapters.NodeId) {
			for {
				// n := rand.Intn(5000)
				// n := 3000
				time.Sleep(time.Duration(n) * time.Millisecond)
				log.Debug(fmt.Sprintf("node %v shutting down", id))
				net.Stop(id)
				// n = rand.Intn(5000)
				n = 2000
				time.Sleep(time.Duration(n) * time.Millisecond)
				log.Debug(fmt.Sprintf("node %v starting up", id))
				net.Start(id)
				n = 5000
			}
		}(id)
	}
}

// var server
func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	log.Root().SetHandler(log.LvlFilterHandler(log.LvlTrace, log.StreamHandler(os.Stderr, log.TerminalFormat(false))))

	services := adapters.Services{
		"overlay": NewSimNode,
	}
	adapters.RegisterServices(services)

	mockers := createMockers()

	config := &simulations.ServerConfig{
		NewAdapter:      func() adapters.NodeAdapter { return adapters.NewSimAdapter(services) },
		DefaultMockerId: "start-stop",
		Mockers:         mockers,
	}

	log.Info("starting simulation server on 0.0.0.0:8888...")
	http.ListenAndServe(":8888", simulations.NewServer(config))
}
