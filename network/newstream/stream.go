// Copyright 2019 The Swarm Authors
// This file is part of the Swarm library.
//
// The Swarm library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The Swarm library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the Swarm library. If not, see <http://www.gnu.org/licenses/>.

package newstream

import (
	"sync"

	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethersphere/swarm/log"
	"github.com/ethersphere/swarm/network"
	"github.com/ethersphere/swarm/p2p/protocols"
	"github.com/ethersphere/swarm/state"
)

// SlipStream implements node.Service
var _ node.Service = (*SlipStream)(nil)

var SyncerSpec = &protocols.Spec{
	Name:       "bzz-stream",
	Version:    8,
	MaxMsgSize: 10 * 1024 * 1024,
	Messages: []interface{}{
		StreamInfoReq{},
		StreamInfoRes{},
		GetRange{},
		OfferedHashes{},
		ChunkDelivery{},
		WantedHashes{},
	},
}

// SlipStream is the base type that handles all client/server operations on a node
// it is instantiated once per stream protocol instance, that is, it should have
// one instance per node
type SlipStream struct {
	mtx            sync.RWMutex
	intervalsStore state.Store //every protocol would make use of this
	peers          map[enode.ID]*Peer
	kad            *network.Kademlia

	providers map[string]StreamProvider

	spec    *protocols.Spec   //this protocol's spec
	balance protocols.Balance //implements protocols.Balance, for accounting
	prices  protocols.Prices  //implements protocols.Prices, provides prices to accounting

	quit chan struct{} // terminates registry goroutines
}

func NewSlipStream(intervalsStore state.Store, kad *network.Kademlia, providers ...StreamProvider) *SlipStream {
	slipStream := &SlipStream{
		intervalsStore: intervalsStore,
		kad:            kad,
		peers:          make(map[enode.ID]*Peer),
		providers:      make(map[string]StreamProvider),
		quit:           make(chan struct{}),
	}

	for _, p := range providers {
		slipStream.providers[p.StreamName()] = p
	}

	slipStream.spec = SyncerSpec

	return slipStream
}

func (s *SlipStream) getPeer(id enode.ID) *Peer {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	p := s.peers[id]
	return p
}

func (s *SlipStream) addPeer(p *Peer) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	s.peers[p.ID()] = p
}

func (s *SlipStream) removePeer(p *Peer) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	if _, found := s.peers[p.ID()]; found {
		log.Error("removing peer", "id", p.ID())
		delete(s.peers, p.ID())
		p.Left()
	} else {
		log.Warn("peer was marked for removal but not found", "peer", p.ID())
	}
}

// Run is being dispatched when 2 nodes connect
func (s *SlipStream) Run(p *p2p.Peer, rw p2p.MsgReadWriter) error {
	peer := protocols.NewPeer(p, rw, s.spec)
	bp := network.NewBzzPeer(peer)

	np := network.NewPeer(bp, s.kad)
	s.kad.On(np)
	defer s.kad.Off(np)

	sp := NewPeer(bp, s.intervalsStore, s.providers)
	s.addPeer(sp)
	defer s.removePeer(sp)
	return peer.Run(sp.HandleMsg)
}

func (s *SlipStream) Protocols() []p2p.Protocol {
	return []p2p.Protocol{
		{
			Name:    "bzz-stream",
			Version: 1,
			Length:  10 * 1024 * 1024,
			Run:     s.Run,
		},
	}
}

func (s *SlipStream) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "bzz-stream",
			Version:   "1.0",
			Service:   NewAPI(s),
			Public:    false,
		},
	}
}

// Additional public methods accessible through API for pss
type API struct {
	*SlipStream
}

func NewAPI(s *SlipStream) *API {
	return &API{SlipStream: s}
}

func (s *SlipStream) Start(server *p2p.Server) error {
	log.Info("slip stream starting")
	return nil
}

func (s *SlipStream) Stop() error {
	log.Info("slip stream closing")
	s.mtx.Lock()
	defer s.mtx.Unlock()
	close(s.quit)
	return nil
}
