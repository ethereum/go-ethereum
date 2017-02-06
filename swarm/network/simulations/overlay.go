// +build none

// You can run this simulation using
//
//    go run ./swarm/network/simulations/overlay.go
package main

import (
	"fmt"
	"reflect"
	"runtime"
	"time"

	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/adapters"
	"github.com/ethereum/go-ethereum/p2p/simulations"
	p2ptest "github.com/ethereum/go-ethereum/p2p/testing"
	"github.com/ethereum/go-ethereum/swarm/network"
)

// Network extends simulations.Network with hives for each node.
type Network struct {
	*simulations.Network
	hives []*network.Hive
}

// SimNode is the adapter used by Swarm simulations.
type SimNode struct {
	hive *network.Hive
	adapters.NodeAdapter
}

// the hive update ticker for hive
func af() <-chan time.Time {
	return time.NewTicker(1 * time.Second).C
}

// Start() starts up the hive
// makes SimNode implement *NodeAdapter
func (self *SimNode) Start() error {
	connect := func(s string) error {
		id := network.HexToBytes(s)
		return self.Connect(id)
	}
	return self.hive.Start(connect, af)
}

// Stop() shuts down the hive
// makes SimNode implement *NodeAdapter
func (self *SimNode) Stop() error {
	self.hive.Stop()
	return nil
}

func (self *SimNode) RunProtocol(id *adapters.NodeId, rw, rrw p2p.MsgReadWriter, runc chan bool) error {
	return self.NodeAdapter.(adapters.ProtocolRunner).RunProtocol(id, rw, rrw, runc)
}

// NewSimNode creates adapters for nodes in the simulation.
func (self *Network) NewSimNode(conf *simulations.NodeConfig) adapters.NodeAdapter {
	id := conf.Id
	na := adapters.NewSimNode(id, self.Network, adapters.NewSimPipe)
	addr := network.NewPeerAddrFromNodeId(id)
	// to := network.NewKademlia(addr.OverlayAddr(), nil)   // overlay topology driver
	to := network.NewTestOverlay(addr.OverlayAddr())   // overlay topology driver
	pp := network.NewHive(network.NewHiveParams(), to) // hive
	self.hives = append(self.hives, pp)                // remember hive
	// bzz protocol Run function. messaging through SimPipe
	ct := network.BzzCodeMap(network.HiveMsgs...) // bzz protocol code map
	na.Run = network.Bzz(addr.OverlayAddr(), pp, na, ct, nil).Run
	return &SimNode{
		hive:        pp,
		NodeAdapter: na,
	}
}

func NewNetwork(network *simulations.Network) *Network {
	n := &Network{
		// hives:
		Network: network,
	}
	n.SetNaf(n.NewSimNode)
	return n
}

// NewSessionController sits as the top-most controller for this simulation
// creates an inprocess simulation of basic node running their own bzz+hive
func NewSessionController() (*simulations.ResourceController, chan bool) {
	quitc := make(chan bool)
	return simulations.NewResourceContoller(
		&simulations.ResourceHandlers{
			// POST /
			Create: &simulations.ResourceHandler{
				Handle: func(msg interface{}, parent *simulations.ResourceController) (interface{}, error) {
					conf := msg.(*simulations.NetworkConfig)
					net := simulations.NewNetwork(nil, &event.TypeMux{})
					ppnet := NewNetwork(net)
					c := simulations.NewNetworkController(conf, net.Events(), simulations.NewJournal())
					if len(conf.Id) == 0 {
						conf.Id = fmt.Sprintf("%d", 0)
					}
					glog.V(logger.Debug).Infof("new network controller on %v", conf.Id)
					if parent != nil {
						parent.SetResource(conf.Id, c)
					}
					ids := p2ptest.RandomNodeIds(5)
					for _, id := range ids {
						ppnet.NewNode(&simulations.NodeConfig{Id: id})
						ppnet.Start(id)
						glog.V(logger.Debug).Infof("node %v starting up", id)
					}
					// the nodes only know about their 2 neighbours (cyclically)
					for i, _ := range ids {
						var peerId *adapters.NodeId
						if i == 0 {
							peerId = ids[len(ids)-1]
						} else {
							peerId = ids[i-1]
						}
						err := ppnet.hives[i].Register(network.NewPeerAddrFromNodeId(peerId))
						if err != nil {
							panic(err.Error())
						}
					}
					return struct{}{}, nil
				},
				Type: reflect.TypeOf(&simulations.NetworkConfig{}),
			},
			// DELETE /
			Destroy: &simulations.ResourceHandler{
				Handle: func(msg interface{}, parent *simulations.ResourceController) (interface{}, error) {
					glog.V(logger.Debug).Infof("destroy handler called")
					// this can quit the entire app (shut down the backend server)
					quitc <- true
					return struct{}{}, nil
				},
			},
		},
	), quitc
}

// var server
func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	glog.SetV(logger.Info)
	glog.SetToStderr(true)

	c, quitc := NewSessionController()

	simulations.StartRestApiServer("8888", c)
	// wait until server shuts down
	<-quitc

}
