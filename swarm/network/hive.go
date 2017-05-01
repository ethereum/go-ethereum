// Copyright 2016 The go-ethereum Authors
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

package network

import (
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
)

/*
Hive is the logistic manager of the swarm
it uses an Overlay Topology driver (e.g., generic kademlia nodetable)
to find best peer list for any target
this is used by the netstore to search for content in the swarm

It handles the bzz protocol getPeersMsg peersMsg exchange
and relay the peer request process to the Overlay module

peer connections and disconnections are reported and registered
to keep the nodetable uptodate
*/
type Overlay interface {
	Register(...PeerAddr) error

	On(Peer)
	Off(Peer)

	EachLivePeer([]byte, int, func(Peer, int, bool) bool)
	EachPeer([]byte, int, func(PeerAddr, int) bool)

	SuggestPeer() (PeerAddr, int, bool)

	String() string
	GetAddr() PeerAddr
}

// Hive implements the PeerPool interface
type Hive struct {
	*HiveParams // settings
	Overlay     // the overlay topology driver
	lock        sync.Mutex
	quit        chan bool
	toggle      chan bool
	more        chan bool
}

type HiveParams struct {
	Discovery             bool
	PeersBroadcastSetSize uint8
	MaxPeersPerRequest    uint8
	CallInterval          uint
}

func NewHiveParams() *HiveParams {
	return &HiveParams{
		Discovery:             true,
		PeersBroadcastSetSize: 2,
		MaxPeersPerRequest:    5,
		CallInterval:          1000,
	}
}

// Hive constructor embeds both arguments
// HiveParams config parameters
// Overlay Topology Driver Interface
func NewHive(params *HiveParams, overlay Overlay) *Hive {
	return &Hive{
		HiveParams: params,
		Overlay:    overlay,
	}
}

// Start receives network info only at startup
// connectPeer is a function to connect to a peer based on its NodeID or enode URL
// these are called on the p2p.Server which runs on the node
// af() returns an arbitrary ticker channel
func (self *Hive) Start(server p2p.Server, af func() <-chan time.Time) error {

	self.toggle = make(chan bool)
	self.more = make(chan bool, 1)
	self.quit = make(chan bool)
	log.Debug("hive started")
	// this loop is doing bootstrapping and maintains a healthy table
	go self.keepAlive(af)
	go func() {
		// each iteration, ask kademlia about most preferred peer
		for more := range self.more {
			if !more {
				// receiving false closes the loop while allowing parallel routines
				// to attempt to write to more (remove Peer when shutting down)
				return
			}
			log.Trace("hive delegate to overlay driver: suggest addr to connect to")
			addr, order, want := self.SuggestPeer()

			if addr != nil {
				log.Info(fmt.Sprintf("========> connect to bee %v", addr))
				node, err := discover.ParseNode(NodeId(addr).NodeID.String())
				if err == nil {
					server.AddPeer(node)
				} else {
					log.Error(fmt.Sprintf("===X====> connect to bee %v failed: invalid node URL: %v", addr, err))
				}
			} else {
				log.Trace("cannot suggest peers")
			}

			want = want && self.Discovery
			if want {
				go RequestOrder(self.Overlay, uint8(order), self.PeersBroadcastSetSize, self.MaxPeersPerRequest)
			}

			select {
			case self.toggle <- want:
				log.Trace(fmt.Sprintf("keep hive alive: %v", want))
			case <-self.quit:
				return
			}
			log.Info(fmt.Sprintf("%v", self))
		}
	}()
	return nil
}

func (self *Hive) ticker() <-chan time.Time {
	return time.NewTicker(time.Duration(self.CallInterval) * time.Millisecond).C
}

// keepAlive is a forever loop
// in its awake state it periodically triggers connection attempts
// by writing to self.more until Kademlia Table is saturated
// wake state is toggled by writing to self.toggle
// it restarts if the table becomes non-full again due to disconnections
func (self *Hive) keepAlive(af func() <-chan time.Time) {
	log.Trace("keep alive loop started")
	alarm := af()
	for {
		select {
		case <-alarm:
			log.Trace("wake up: make hive alive")
			self.wake()
		case need := <-self.toggle:
			if alarm == nil && need {
				alarm = af()
			}
			// if hive saturated, no more peers asked
			if alarm != nil && !need {
				alarm = nil
			}
		case <-self.quit:
			return
		}
	}
}

// Add is called at the end of a successful protocol handshake
// to register a connected (live) peer
func (self *Hive) Add(p Peer) error {
	defer self.wake()
	dp := NewDiscovery(p, self.Overlay)
	log.Debug(fmt.Sprintf("to add new bee %v", p))
	self.On(dp)
	self.String()
	log.Debug(fmt.Sprintf("%v", self))
	return nil
}

// Remove called after peer is disconnected
func (self *Hive) Remove(p Peer) {
	defer self.wake()
	log.Debug(fmt.Sprintf("remove bee %v", p))
	self.Off(p)
}

// NodeInfo function is used by the p2p.server RPC interface to display
// protocol specific node information
func (self *Hive) NodeInfo() interface{} {
	return interface{}(self.String())
}

// PeerInfo function is used by the p2p.server RPC interface to display
// protocol specific information any connected peer referred to by their NodeID
func (self *Hive) PeerInfo(id discover.NodeID) interface{} {
	self.lock.Lock()
	defer self.lock.Unlock()
	addr := NewPeerAddrFromNodeId(adapters.NewNodeId(id[:]))
	return interface{}(addr)
}

// Stop terminates the updateloop
func (self *Hive) Stop() {
	// closing toggle channel quits the updateloop
	close(self.quit)
}

func (self *Hive) Healthy() bool {
	// TODO: determine if we have enough peers to consider the network
	//       to be healthy
	return true
}

func (self *Hive) wake() {
	select {
	case self.more <- true:
		log.Trace("hive woken up")
	case <-self.quit:
	default:
		log.Trace("hive already awake")
	}
}

func HexToBytes(s string) []byte {
	id := discover.MustHexID(s)
	return id[:]
}
