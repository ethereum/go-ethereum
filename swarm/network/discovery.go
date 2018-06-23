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

	"github.com/ethereum/go-ethereum/swarm/pot"
)

// discovery bzz extension for requesting and relaying node address records

// discPeer wraps BzzPeer and embeds an Overlay connectivity driver
type discPeer struct {
	*BzzPeer
	overlay   Overlay
	sentPeers bool // whether we already sent peer closer to this address
	mtx       sync.RWMutex
	peers     map[string]bool // tracks node records sent to the peer
	depth     uint8           // the proximity order advertised by remote as depth of saturation
}

// NewDiscovery constructs a discovery peer
func newDiscovery(p *BzzPeer, o Overlay) *discPeer {
	d := &discPeer{
		overlay: o,
		BzzPeer: p,
		peers:   make(map[string]bool),
	}
	// record remote as seen so we never send a peer its own record
	d.seen(d)
	return d
}

// HandleMsg is the message handler that delegates incoming messages
func (d *discPeer) HandleMsg(msg interface{}) error {
	switch msg := msg.(type) {

	case *peersMsg:
		return d.handlePeersMsg(msg)

	case *subPeersMsg:
		return d.handleSubPeersMsg(msg)

	default:
		return fmt.Errorf("unknown message type: %T", msg)
	}
}

// NotifyDepth sends a message to all connections if depth of saturation is changed
func NotifyDepth(depth uint8, h Overlay) {
	f := func(val OverlayConn, po int, _ bool) bool {
		dp, ok := val.(*discPeer)
		if ok {
			dp.NotifyDepth(depth)
		}
		return true
	}
	h.EachConn(nil, 255, f)
}

// NotifyPeer informs all peers about a newly added node
func NotifyPeer(p OverlayAddr, k Overlay) {
	f := func(val OverlayConn, po int, _ bool) bool {
		dp, ok := val.(*discPeer)
		if ok {
			dp.NotifyPeer(p, uint8(po))
		}
		return true
	}
	k.EachConn(p.Address(), 255, f)
}

// NotifyPeer notifies the remote node (recipient) about a peer if
// the peer's PO is within the recipients advertised depth
// OR the peer is closer to the recipient than self
// unless already notified during the connection session
func (d *discPeer) NotifyPeer(a OverlayAddr, po uint8) {
	// immediately return
	if (po < d.getDepth() && pot.ProxCmp(d.localAddr, d, a) != 1) || d.seen(a) {
		return
	}
	// log.Trace(fmt.Sprintf("%08x peer %08x notified of peer %08x", d.localAddr.Over()[:4], d.Address()[:4], a.Address()[:4]))
	resp := &peersMsg{
		Peers: []*BzzAddr{ToAddr(a)},
	}
	go d.Send(resp)
}

// NotifyDepth sends a subPeers Msg to the receiver notifying them about
// a change in the depth of saturation
func (d *discPeer) NotifyDepth(po uint8) {
	// log.Trace(fmt.Sprintf("%08x peer %08x notified of new depth %v", d.localAddr.Over()[:4], d.Address()[:4], po))
	go d.Send(&subPeersMsg{Depth: po})
}

/*
peersMsg is the message to pass peer information
It is always a response to a peersRequestMsg

The encoding of a peer address is identical the devp2p base protocol peers
messages: [IP, Port, NodeID],
Note that a node's FileStore address is not the NodeID but the hash of the NodeID.

TODO:
To mitigate against spurious peers messages, requests should be remembered
and correctness of responses should be checked

If the proxBin of peers in the response is incorrect the sender should be
disconnected
*/

// peersMsg encapsulates an array of peer addresses
// used for communicating about known peers
// relevant for bootstrapping connectivity and updating peersets
type peersMsg struct {
	Peers []*BzzAddr
}

// String pretty prints a peersMsg
func (msg peersMsg) String() string {
	return fmt.Sprintf("%T: %v", msg, msg.Peers)
}

// handlePeersMsg called by the protocol when receiving peerset (for target address)
// list of nodes ([]PeerAddr in peersMsg) is added to the overlay db using the
// Register interface method
func (d *discPeer) handlePeersMsg(msg *peersMsg) error {
	// register all addresses
	if len(msg.Peers) == 0 {
		return nil
	}

	for _, a := range msg.Peers {
		d.seen(a)
		NotifyPeer(a, d.overlay)
	}
	return d.overlay.Register(toOverlayAddrs(msg.Peers...))
}

// subPeers msg is communicating the depth/sharpness/focus of the overlay table of a peer
type subPeersMsg struct {
	Depth uint8
}

// String returns the pretty printer
func (msg subPeersMsg) String() string {
	return fmt.Sprintf("%T: request peers > PO%02d. ", msg, msg.Depth)
}

func (d *discPeer) handleSubPeersMsg(msg *subPeersMsg) error {
	if !d.sentPeers {
		d.setDepth(msg.Depth)
		var peers []*BzzAddr
		d.overlay.EachConn(d.Over(), 255, func(p OverlayConn, po int, isproxbin bool) bool {
			if pob, _ := pof(d, d.localAddr, 0); pob > po {
				return false
			}
			if !d.seen(p) {
				peers = append(peers, ToAddr(p.Off()))
			}
			return true
		})
		if len(peers) > 0 {
			// log.Debug(fmt.Sprintf("%08x: %v peers sent to %v", d.overlay.BaseAddr(), len(peers), d))
			go d.Send(&peersMsg{Peers: peers})
		}
	}
	d.sentPeers = true
	return nil
}

// seen takes an Overlay peer and checks if it was sent to a peer already
// if not, marks the peer as sent
func (d *discPeer) seen(p OverlayPeer) bool {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	k := string(p.Address())
	if d.peers[k] {
		return true
	}
	d.peers[k] = true
	return false
}

func (d *discPeer) getDepth() uint8 {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	return d.depth
}
func (d *discPeer) setDepth(depth uint8) {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	d.depth = depth
}
