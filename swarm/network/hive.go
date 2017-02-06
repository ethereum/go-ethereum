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
	"bytes"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p/discover"
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

	EachLivePeer([]byte, int, func(Peer) bool)
	EachPeer([]byte, int, func(PeerAddr) bool)

	SuggestPeer() (PeerAddr, int, bool)

	String() string
}

// Hive implements the PeerPool interface
type Hive struct {
	*HiveParams // settings
	Overlay     // the overlay topology driver
	peers       map[discover.NodeID]Peer

	lock   sync.Mutex
	quit   chan bool
	toggle chan bool
	more   chan bool
}

const (
	peersBroadcastSetSize = 1
	maxPeersPerRequest    = 5
	callInterval          = 3000000000
)

type HiveParams struct {
	PeersBroadcastSetSize uint
	MaxPeersPerRequest    uint
	CallInterval          uint64
}

func NewHiveParams() *HiveParams {
	return &HiveParams{
		PeersBroadcastSetSize: peersBroadcastSetSize,
		MaxPeersPerRequest:    maxPeersPerRequest,
		CallInterval:          callInterval,
	}
}

// Hive constructor embeds both arguments
// HiveParams config parameters
// Overlay Topology Driver Interface
func NewHive(params *HiveParams, overlay Overlay) *Hive {
	return &Hive{
		HiveParams: params,
		Overlay:    overlay,
		peers:      make(map[discover.NodeID]Peer),
	}
}

// messages that hive regusters handles for
var HiveMsgs = []interface{}{
	&getPeersMsg{},
	&peersMsg{},
}

/*
peersMsg is the message to pass peer information
It is always a response to a peersRequestMsg

The encoding of a peer is identical to that in the devp2p base protocol peers
messages: [IP, Port, NodeID]
note that a node's DPA address is not the NodeID but the hash of the NodeID.

To mitigate against spurious peers messages, requests should be remembered
and correctness of responses should be checked

If the proxBin of peers in the response is incorrect the sender should be
disconnected
*/

// peersMsg encapsulates an array of peer addresses
// used for communicating about known peers
// relevvant for bootstrapping connectivity and updating peersets
type peersMsg struct {
	Peers []*peerAddr
}

func (self peersMsg) String() string {
	return fmt.Sprintf("%T: %v", self, self.Peers)
}

// getPeersMsg is sent to (random) peers to request (Max) peers of a specific order
type getPeersMsg struct {
	Order uint
	Max   uint
}

func (self getPeersMsg) String() string {
	return fmt.Sprintf("%T: accept max %v peers of PO%03d", self, self.Max, self.Order)
}

// Start receives network info only at startup
// listedAddr is a function to retrieve listening address to advertise to peers
// connectPeer is a function to connect to a peer based on its NodeID or enode URL
// af() returns an arbitrary ticker channel
// there are called on the p2p.Server which runs on the node
func (self *Hive) Start(connectPeer func(string) error, af func() <-chan time.Time) (err error) {

	self.toggle = make(chan bool)
	self.more = make(chan bool)
	self.quit = make(chan bool)
	glog.V(logger.Debug).Infof("hive started")
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
			glog.V(logger.Detail).Infof("hive delegate to overlay driver: suggest addr to connect to")
			addr, order, want := self.SuggestPeer()

			if addr != nil {
				glog.V(logger.Detail).Infof("========> connect to bee %v", addr)
				err := connectPeer(NodeId(addr).NodeID.String())
				if err != nil {
					glog.V(logger.Detail).Infof("===X====> connect to bee %v failed: %v", addr, err)
				}
			}
			if want {
				req := &getPeersMsg{
					Order: uint(order),
					Max:   self.MaxPeersPerRequest,
				}
				var i uint
				var err error
				glog.V(logger.Detail).Infof("requesting bees of PO%03d from %v (each max %v)", order, self.PeersBroadcastSetSize, self.MaxPeersPerRequest)
				self.EachLivePeer(nil, order, func(n Peer) bool {
					glog.V(logger.Detail).Infof("%T sent to %v", req, n.ID())
					err = n.Send(req)
					if err == nil {
						i++
						if i >= self.PeersBroadcastSetSize {
							return false
						}
					}
					return true
				})
				glog.V(logger.Detail).Infof("sent %T to %d/%d peers", req, i, self.PeersBroadcastSetSize)
			}
			glog.V(logger.Info).Infof("%v", self)

			select {
			case self.toggle <- want:
				glog.V(logger.Detail).Infof("keep hive alive: %v", want)
			case <-self.quit:
				return
			}
		}
		glog.V(logger.Warn).Infof("%v", self)
	}()
	return
}

func (self *Hive) ticker() <-chan time.Time {
	return time.NewTicker(time.Duration(self.CallInterval)).C
}

// keepAlive is a forever loop
// in its awake state it periodically triggers connection attempts
// by writing to self.more until Kademlia Table is saturated
// wake state is toggled by writing to self.toggle
// it restarts if the table becomes non-full again due to disconnections
func (self *Hive) keepAlive(af func() <-chan time.Time) {
	glog.V(logger.Detail).Infof("keep alive loop started")
	alarm := af()
	for {
		select {
		case <-alarm:
			glog.V(logger.Detail).Infof("wake up: make hive alive")
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

func (self *Hive) Stop() {
	// closing toggle channel quits the updateloop
	close(self.quit)
}

func (self *Hive) wake() {
	select {
	case self.more <- true:
		glog.V(logger.Detail).Infof("hive woken up")
	case <-self.quit:
	default:
		glog.V(logger.Detail).Infof("hive already awake")
	}
}

// Add is called at the end of a successful protocol handshake
// to register a connected (live) peer
func (self *Hive) Add(p Peer) error {
	defer self.wake()
	glog.V(logger.Debug).Infof("to add new bee %v", p)
	self.On(p)

	self.lock.Lock()
	self.peers[p.ID()] = p
	self.lock.Unlock()

	p.Register(&peersMsg{}, self.handlePeersMsg(p))
	p.Register(&getPeersMsg{}, self.handleGetPeersMsg(p))

	return nil
}

// Remove called after peer is disconnected
func (self *Hive) Remove(p Peer) {
	defer self.wake()
	glog.V(logger.Debug).Infof("remove bee %v", p)
	self.Off(p)
	self.lock.Lock()
	delete(self.peers, p.ID())
	self.lock.Unlock()
}

// handlePeersMsg called by the protocol when receiving peerset (for target address)
// list of nodes ([]PeerAddr in peersMsg) is added to the overlay db using the
// Register interface method
func (self *Hive) handlePeersMsg(p Peer) func(interface{}) error {
	return func(msg interface{}) error {
		// wake up the hive on news of new arrival
		defer self.wake()
		// register all addresses
		var nas []PeerAddr
		for _, na := range msg.(*peersMsg).Peers {
			nas = append(nas, PeerAddr(na))
		}
		return self.Register(nas...)
	}
}

// handleGetPeersMsg is called by the protocol when receiving a
// peerset (for target address) request
// peers suggestions are retrieved from the overlay topology driver
// using the EachLivePeer interface iterator method
// peers sent are remembered throughout a session and not sent twice
func (self *Hive) handleGetPeersMsg(p Peer) func(interface{}) error {
	return func(msg interface{}) error {
		req := msg.(*getPeersMsg)
		var peers []*peerAddr
		alreadySent := p.Peers()
		self.EachLivePeer(p.OverlayAddr(), int(req.Order), func(n Peer) bool {
			if bytes.Compare(n.OverlayAddr(), p.OverlayAddr()) != 0 &&
				// only send peers we have not sent before in this session
				!alreadySent[n.ID()] {
				alreadySent[n.ID()] = true
				peers = append(peers, &peerAddr{n.OverlayAddr(), n.UnderlayAddr()})
			}
			return len(peers) < int(req.Max)
		})

		resp := &peersMsg{
			Peers: peers,
		}
		err := p.Send(resp)
		if err != nil {
			return err
		}
		return nil
	}
}

func (self *Hive) NodeInfo() interface{} {
	return interface{}(self.String())
}

func (self *Hive) PeerInfo(id discover.NodeID) interface{} {
	self.lock.Lock()
	defer self.lock.Unlock()
	p, ok := self.peers[id]
	if !ok {
		return nil
	}
	return interface{}(&peerAddr{p.OverlayAddr(), p.UnderlayAddr()})
}

func HexToBytes(s string) []byte {
	id := discover.MustHexID(s)
	return id[:]
}
