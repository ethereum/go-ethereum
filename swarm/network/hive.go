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
	"encoding/json"
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

// Overlay is the interface to Jaak ahd ka)a
type Overlay interface {
	Register(chan OverlayAddr) error

	On(OverlayPeer)
	Off(OverlayConn)

	EachConn([]byte, int, func(OverlayConn, int, bool) bool)
	EachAddr([]byte, int, func(OverlayAddr, int) bool)

	SuggestPeer() (OverlayAddr, int, bool)

	String() string
	BaseAddr() []byte
}

// ReadWriter interface to persist known peers, uses disk for real nodes
type ReadWriter interface {
	ReadAll(string) ([]byte, error)
	WriteAll(string, []byte) error
}

// Hive implements the PeerPool interface
type Hive struct {
	*HiveParams            // settings
	Overlay                // the overlay topology driver
	RW          ReadWriter // ReadWriter

	// bookkeeping
	lock   sync.Mutex
	quit   chan bool
	toggle chan bool
	more   chan bool
}

// HiveParams holds the config options to hive
type HiveParams struct {
	Discovery             bool  // if want discovery of not
	PeersBroadcastSetSize uint8 // how many peers to use when relaying
	MaxPeersPerRequest    uint8 // max size for peer address batches
	CallInterval          uint  // polling interval fir===
}

// NewHiveParams returns hive config with only the
func NewHiveParams() *HiveParams {
	return &HiveParams{
		Discovery:             true,
		PeersBroadcastSetSize: 2,
		MaxPeersPerRequest:    5,
		CallInterval:          1000,
	}
}

// Hive constructor embeds both arguments
// HiveParams: config parameters
// Overlay: Topology Driver Interface
func NewHive(params *HiveParams, overlay Overlay) *Hive {
	return &Hive{
		HiveParams: params,
		Overlay:    overlay,
	}
}

// Start receives network info only at startup
// server is used to connect to a peer based on its NodeID or enode URL
// these are called on the p2p.Server which runs on the node
// af() returns an arbitrary ticker channel
// rw is a read writer for json configs
func (self *Hive) Start(server p2p.Server, af func() <-chan time.Time, rw ReadWriter) error {
	if rw != nil {
		if err := self.loadPeers(); err != nil {
			return err
		}
	}
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
				under, err := discover.ParseNode(string(addr.(Addr).Under()))
				if err == nil {
					server.AddPeer(under)
				} else {
					log.Error(fmt.Sprintf("===X====> connect to bee %v failed: invalid node URL: %v", addr, err))
				}
			} else {
				log.Trace("cannot suggest peers")
			}

			want = want && self.Discovery
			if want {
				RequestOrder(self.Overlay, uint8(order), self.PeersBroadcastSetSize, self.MaxPeersPerRequest)
			}

			select {
			case self.toggle <- want:
				log.Trace(fmt.Sprintf("keep hive alive: %v", want))
			case <-self.quit:
				return
			}
			// log.Info(fmt.Sprintf("%v", self))
		}
	}()
	return nil
}

// Stop terminates the updateloop and saves the peers
func (self *Hive) Stop() {
	if self.RW != nil {
		self.savePeers()
	}
	// closing toggle channel quits the updateloop
	close(self.quit)
}

// default ticker, tickinterval is taken from KadParams.CallInterval
func (self *Hive) ticker() <-chan time.Time {
	return time.NewTicker(time.Duration(self.CallInterval) * time.Millisecond).C
}

// Add is called at the end of a successful protocol handshake
// to register a connected (live) peer
func (self *Hive) Add(p *bzzPeer) error {
	defer self.wake()
	dp := NewDiscovery(p, self.Overlay)
	log.Debug(fmt.Sprintf("to add new bee %v", p))
	self.On(dp)
	self.String()
	log.Debug(fmt.Sprintf("%v", self))
	return nil
}

// Remove called after peer is disconnected
func (self *Hive) Remove(p *bzzPeer) {
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
	addr := NewAddrFromNodeId(adapters.NewNodeId(id[:]))
	return interface{}(addr)
}

// Healthy reports the health state of the kademlia connectivity
//
func (self *Hive) Healthy() bool {
	// TODO: determine if we have enough peers to consider the network
	//       to be healthy
	return true
}

// wake triggers
func (self *Hive) wake() {
	select {
	case self.more <- true:
		log.Trace("hive woken up")
	case <-self.quit:
	default:
		log.Trace("hive already awake")
	}
}

// HexToBytes reads a hex string ontp
func HexToBytes(s string) []byte {
	id := discover.MustHexID(s)
	return id[:]
}

// ToAddr returns the serialisable version of u
func ToAddr(pa OverlayPeer) *bzzAddr {
	if addr, ok := pa.(*bzzAddr); ok {
		return addr
	}
	return pa.(*bzzPeer).bzzAddr
}

// keepAlive is a forever loop
// in its awake state it periodically triggers connection attempts
// by writing to self.more until Kademlia Table is saturated
// wake state is toggled by writing to self.toggle
// it goes to sleep mode if table is saturated
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

// loadPeers, savePeer implement persistence callback/
func (self *Hive) loadPeers() error {
	rw := self.RW
	data, err := rw.ReadAll("peers")
	if err != nil {
		return err
	}
	if data == nil {
		return nil
	}
	var as []*bzzAddr
	if err := json.Unmarshal(data, &as); err != nil {
		return err
	}

	var c chan OverlayAddr
	defer close(c)
	go func() {
		for _, a := range as {
			c <- a
		}
	}()
	return self.Overlay.Register(c)
}

// savePeers, savePeer implement persistence callback/
func (self *Hive) savePeers() error {
	var peers []*bzzAddr
	self.Overlay.EachAddr(nil, 256, func(pa OverlayAddr, i int) bool {
		if pa == nil {
			log.Warn(fmt.Sprintf("empty addr: %v", i))
			return true
		}
		peers = append(peers, ToAddr(pa))
		return true
	})
	data, err := json.Marshal(peers)
	if err != nil {
		return fmt.Errorf("could not encode peers: %v", err)
	}
	if err := self.RW.WriteAll("peers", data); err != nil {
		return fmt.Errorf("could not save peers: %v", err)
	}
	return nil
}
