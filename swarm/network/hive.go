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

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/swarm/log"
	"github.com/ethereum/go-ethereum/swarm/state"
)

/*
Hive is the logistic manager of the swarm

When the hive is started, a forever loop is launched that
asks the  kademlia nodetable
to suggest peers to bootstrap connectivity
*/

// HiveParams holds the config options to hive
type HiveParams struct {
	Discovery             bool  // if want discovery of not
	PeersBroadcastSetSize uint8 // how many peers to use when relaying
	MaxPeersPerRequest    uint8 // max size for peer address batches
	KeepAliveInterval     time.Duration
}

// NewHiveParams returns hive config with only the
func NewHiveParams() *HiveParams {
	return &HiveParams{
		Discovery:             true,
		PeersBroadcastSetSize: 3,
		MaxPeersPerRequest:    5,
		KeepAliveInterval:     500 * time.Millisecond,
	}
}

// Hive manages network connections of the swarm node
type Hive struct {
	*HiveParams                      // settings
	*Kademlia                        // the overlay connectiviy driver
	Store       state.Store          // storage interface to save peers across sessions
	addPeer     func(*discover.Node) // server callback to connect to a peer
	// bookkeeping
	lock   sync.Mutex
	ticker *time.Ticker
}

// NewHive constructs a new hive
// HiveParams: config parameters
// Kademlia: connectivity driver using a network topology
// StateStore: to save peers across sessions
func NewHive(params *HiveParams, kad *Kademlia, store state.Store) *Hive {
	return &Hive{
		HiveParams: params,
		Kademlia:   kad,
		Store:      store,
	}
}

// Start stars the hive, receives p2p.Server only at startup
// server is used to connect to a peer based on its NodeID or enode URL
// these are called on the p2p.Server which runs on the node
func (h *Hive) Start(server *p2p.Server) error {
	log.Info("Starting hive", "baseaddr", fmt.Sprintf("%x", h.BaseAddr()[:4]))
	// if state store is specified, load peers to prepopulate the overlay address book
	if h.Store != nil {
		log.Info("Detected an existing store. trying to load peers")
		if err := h.loadPeers(); err != nil {
			log.Error(fmt.Sprintf("%08x hive encoutered an error trying to load peers", h.BaseAddr()[:4]))
			return err
		}
	}
	// assigns the p2p.Server#AddPeer function to connect to peers
	h.addPeer = server.AddPeer
	// ticker to keep the hive alive
	h.ticker = time.NewTicker(h.KeepAliveInterval)
	// this loop is doing bootstrapping and maintains a healthy table
	go h.connect()
	return nil
}

// Stop terminates the updateloop and saves the peers
func (h *Hive) Stop() error {
	log.Info(fmt.Sprintf("%08x hive stopping, saving peers", h.BaseAddr()[:4]))
	h.ticker.Stop()
	if h.Store != nil {
		if err := h.savePeers(); err != nil {
			return fmt.Errorf("could not save peers to persistence store: %v", err)
		}
		if err := h.Store.Close(); err != nil {
			return fmt.Errorf("could not close file handle to persistence store: %v", err)
		}
	}
	log.Info(fmt.Sprintf("%08x hive stopped, dropping peers", h.BaseAddr()[:4]))
	h.EachConn(nil, 255, func(p *Peer, _ int, _ bool) bool {
		log.Info(fmt.Sprintf("%08x dropping peer %08x", h.BaseAddr()[:4], p.Address()[:4]))
		p.Drop(nil)
		return true
	})

	log.Info(fmt.Sprintf("%08x all peers dropped", h.BaseAddr()[:4]))
	return nil
}

// connect is a forever loop
// at each iteration, ask the overlay driver to suggest the most preferred peer to connect to
// as well as advertises saturation depth if needed
func (h *Hive) connect() {
	for range h.ticker.C {

		addr, depth, changed := h.SuggestPeer()
		if h.Discovery && changed {
			NotifyDepth(uint8(depth), h.Kademlia)
		}
		if addr == nil {
			continue
		}

		log.Trace(fmt.Sprintf("%08x hive connect() suggested %08x", h.BaseAddr()[:4], addr.Address()[:4]))
		under, err := discover.ParseNode(string(addr.Under()))
		if err != nil {
			log.Warn(fmt.Sprintf("%08x unable to connect to bee %08x: invalid node URL: %v", h.BaseAddr()[:4], addr.Address()[:4], err))
			continue
		}
		log.Trace(fmt.Sprintf("%08x attempt to connect to bee %08x", h.BaseAddr()[:4], addr.Address()[:4]))
		h.addPeer(under)
	}
}

// Run protocol run function
func (h *Hive) Run(p *BzzPeer) error {
	dp := NewPeer(p, h.Kademlia)
	depth, changed := h.On(dp)
	// if we want discovery, advertise change of depth
	if h.Discovery {
		if changed {
			// if depth changed, send to all peers
			NotifyDepth(depth, h.Kademlia)
		} else {
			// otherwise just send depth to new peer
			dp.NotifyDepth(depth)
		}
	}
	NotifyPeer(p.BzzAddr, h.Kademlia)
	defer h.Off(dp)
	return dp.Run(dp.HandleMsg)
}

// NodeInfo function is used by the p2p.server RPC interface to display
// protocol specific node information
func (h *Hive) NodeInfo() interface{} {
	return h.String()
}

// PeerInfo function is used by the p2p.server RPC interface to display
// protocol specific information any connected peer referred to by their NodeID
func (h *Hive) PeerInfo(id discover.NodeID) interface{} {
	addr := NewAddrFromNodeID(id)
	return struct {
		OAddr hexutil.Bytes
		UAddr hexutil.Bytes
	}{
		OAddr: addr.OAddr,
		UAddr: addr.UAddr,
	}
}

// loadPeers, savePeer implement persistence callback/
func (h *Hive) loadPeers() error {
	var as []*BzzAddr
	err := h.Store.Get("peers", &as)
	if err != nil {
		if err == state.ErrNotFound {
			log.Info(fmt.Sprintf("hive %08x: no persisted peers found", h.BaseAddr()[:4]))
			return nil
		}
		return err
	}
	log.Info(fmt.Sprintf("hive %08x: peers loaded", h.BaseAddr()[:4]))

	return h.Register(as...)
}

// savePeers, savePeer implement persistence callback/
func (h *Hive) savePeers() error {
	var peers []*BzzAddr
	h.Kademlia.EachAddr(nil, 256, func(pa *BzzAddr, i int, _ bool) bool {
		if pa == nil {
			log.Warn(fmt.Sprintf("empty addr: %v", i))
			return true
		}
		log.Trace("saving peer", "peer", pa)
		peers = append(peers, pa)
		return true
	})
	if err := h.Store.Put("peers", peers); err != nil {
		return fmt.Errorf("could not save peers: %v", err)
	}
	return nil
}
