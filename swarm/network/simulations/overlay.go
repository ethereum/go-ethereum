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
func NewSimNode(id *adapters.NodeId) node.Service {
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

func mocker(net *simulations.Network) {
	conf := net.Config()
	conf.DefaultService = "overlay"

	ids := make([]*adapters.NodeId, 10)
	for i := 0; i < 10; i++ {
		conf, err := net.NewNode()
		if err != nil {
			panic(err.Error())
		}
		ids[i] = conf.Id
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

	config := &simulations.ServerConfig{
		Adapter: adapters.NewSimAdapter(services),
		Mocker:  mocker,
	}

	log.Info("starting simulation server on 0.0.0.0:8888...")
	http.ListenAndServe(":8888", simulations.NewServer(config))
}
