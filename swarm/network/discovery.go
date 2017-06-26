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

	"github.com/ethereum/go-ethereum/log"
)

// discovery bzz overlay extension doing peer relaying

type discPeer struct {
	*bzzPeer
	overlay   Overlay
	mtx       sync.Mutex
	peers     map[string]bool
	depth     uint8 // the proximity radius advertised by remote to subscribe to peers
	sentPeers bool  // set to true  when the peer is first notifed of peers close to them
}

// NewDiscovery discovery peer contructor
func newDiscovery(p *bzzPeer, o Overlay) *discPeer {
	d := &discPeer{
		overlay: o,
		bzzPeer: p,
		peers:   make(map[string]bool),
	}
	d.seen(d)
	return d
}

func (d *discPeer) HandleMsg(msg interface{}) error {
	switch msg := msg.(type) {

	case *peersMsg:
		return d.handlePeersMsg(msg)

	case *getPeersMsg:
		return d.handleGetPeersMsg(msg)

	case *subPeersMsg:
		return d.handleSubPeersMsg(msg)

	default:
		return fmt.Errorf("unknown message type: %T", msg)
	}
}

// NotifyPeer notifies the receiver remote end of a peer p or PO po.
// callback for overlay driver
func (d *discPeer) NotifyPeer(a OverlayAddr, po uint8) error {
	if po < d.depth || d.seen(a) {
		return nil
	}
	log.Warn(fmt.Sprintf("notification about %x", a.Address()))

	resp := &peersMsg{
		Peers: []*bzzAddr{ToAddr(a)}, // perhaps the PeerAddr interface is unnecessary generalization
	}
	return d.Send(resp)
}

// NotifyDepth sends a subPeers Msg to the receiver notifying them about
// a change in the prox limit (radius of the set including the nearest X peers
// or first empty row)
// callback for overlay driver
func (d *discPeer) NotifyDepth(po uint8) error {
	return d.Send(&subPeersMsg{Depth: po})
}

/*
peersMsg is the message to pass peer information
It is always a response to a peersRequestMsg

The encoding of a peer is identical to that in the devp2p base protocol peers
messages: [IP, Port, NodeID]
note that a node's DPA address is not the NodeID but the hash of the NodeID.

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
	Peers []*bzzAddr
}

func (msg peersMsg) String() string {
	return fmt.Sprintf("%T: %v", msg, msg.Peers)
}

// getPeersMsg is sent to (random) peers to request (Max) peers of a specific order
type getPeersMsg struct {
	Order uint8
	Max   uint8
}

func (msg getPeersMsg) String() string {
	return fmt.Sprintf("%T: accept max %v peers of PO%03d", msg, msg.Max, msg.Order)
}

// subPeers msg is communicating the depth/sharpness/focus  of the overlay table of a peer
type subPeersMsg struct {
	Depth uint8
}

func (msg subPeersMsg) String() string {
	return fmt.Sprintf("%T: request peers > PO%02d. ", msg, msg.Depth)
}

func (d *discPeer) handleSubPeersMsg(msg *subPeersMsg) error {
	d.depth = msg.Depth
	if !d.sentPeers {
		var peers []*bzzAddr
		d.overlay.EachConn(d.Over(), 255, func(p OverlayConn, po int, isproxbin bool) bool {
			if uint8(po) < d.depth {
				return false
			}
			if !d.seen(p) {
				peers = append(peers, ToAddr(p.Off()))
			}
			return true
		})
		log.Warn(fmt.Sprintf("found initial %v peers not farther than %v", len(peers), d.depth))
		if len(peers) > 0 {
			if err := d.Send(&peersMsg{Peers: peers}); err != nil {
				return err
			}
		}
	}
	d.sentPeers = true
	return nil
}

// handlePeersMsg called by the protocol when receiving peerset (for target address)
// list of nodes ([]PeerAddr in peersMsg) is added to the overlay db using the
// Register interface method
func (d *discPeer) handlePeersMsg(msg *peersMsg) error {
	// register all addresses
	if len(msg.Peers) == 0 {
		log.Debug(fmt.Sprintf("whoops, no peers in incoming peersMsg from %v", d))
		return nil
	}

	c := make(chan OverlayAddr)
	go func() {
		defer close(c)
		for _, a := range msg.Peers {
			d.seen(a)
			c <- a
		}
	}()
	log.Info("discovery overlay register")
	return d.overlay.Register(c)
}

// handleGetPeersMsg is called by the protocol when receiving a
// peerset (for target address) request
// peers suggestions are retrieved from the overlay topology driver
// using the EachConn interface iterator method
// peers sent are remembered throughout a session and not sent twice
func (d *discPeer) handleGetPeersMsg(msg *getPeersMsg) error {
	var peers []*bzzAddr
	i := 0
	d.overlay.EachConn(d.Over(), int(msg.Order), func(p OverlayConn, po int, isproxbin bool) bool {
		i++
		// only send peers we have not sent before in this session
		a := ToAddr(p.Off())
		if d.seen(a) {
			peers = append(peers, a)
		}
		return len(peers) < int(msg.Max)
	})
	if len(peers) == 0 {
		log.Debug(fmt.Sprintf("no peers found for %v", d))
		return nil
	}
	log.Debug(fmt.Sprintf("%v peers sent to %v", len(peers), d))
	resp := &peersMsg{
		Peers: peers,
	}
	go d.Send(resp)
	return nil
}

// RequestOrder broadcasts to trageted peers a request for peers of a particular
// proximity order
func RequestOrder(k Overlay, order, broadcastSize, maxPeers uint8) {
	req := &getPeersMsg{
		Order: uint8(order),
		Max:   maxPeers,
	}
	var i uint8
	var peers []Conn
	k.EachConn(nil, 255, func(p OverlayConn, po int, isproxbin bool) bool {
		peers = append(peers, p.(Conn))
		if len(peers) >= int(broadcastSize) {
			return false
		}
		return true
	})
	go func() {
		for _, c := range peers {
			if err := c.Send(req); err != nil {
				break
			}
		}
		log.Info(fmt.Sprintf("requesting bees of PO%03d from %v/%v (each max %v)", order, i, broadcastSize, maxPeers))
	}()
}

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
