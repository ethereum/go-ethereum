// Copyright 2018 The go-ethereum Authors
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

package swap

import (
	"context"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/log"
)

const (
	IsActiveProtocol = true
)

type SwapProtocol struct {
	peersMu sync.RWMutex
	peers   map[discover.NodeID]*SwapPeer
}

// Peer is the Peer extension for the streaming protocol
type SwapPeer struct {
	*protocols.Peer
	swapProtocol *SwapProtocol
}

func NewSwapProtocol() *SwapProtocol {
	proto := &SwapProtocol{
		peers: make(map[discover.NodeID]*SwapPeer),
	}
	return proto
}

// NewPeer is the constructor for Peer
func NewPeer(peer *protocols.Peer, swap *SwapProtocol) *SwapPeer {
	p := &SwapPeer{
		Peer:         peer,
		swapProtocol: swap,
	}
	return p
}

type IssueChequeMsg struct {
}

type RedeemChequeMsg struct {
}

/////////////////////////////////////////////////////////////////////
// SECTION: node.Service interface
/////////////////////////////////////////////////////////////////////

func (p *SwapProtocol) Start(srv *p2p.Server) error {
	log.Debug("Started swap")
	return nil
}

func (p *SwapProtocol) Stop() error {
	log.Info("swap shutting down")
	return nil
}

var swapSpec = &protocols.Spec{
	Name:       swapProtocolName,
	Version:    swapVersion,
	MaxMsgSize: defaultMaxMsgSize,
	Messages: []interface{}{
		SwapMsg{},
	},
}

func (p *SwapProtocol) Protocols() []p2p.Protocol {
	return []p2p.Protocol{
		{
			Name:    swapSpec.Name,
			Version: swapSpec.Version,
			Length:  swapSpec.Length(),
			Run:     p.Run,
		},
	}
}

func (swap *SwapProtocol) DebitByteCount(peer *SwapPeer, numberOfBytes int) error {
	return nil
}

func (swap *SwapProtocol) Run(peer *p2p.Peer, rw p2p.MsgReadWriter) error {
	p := protocols.NewPeer(peer, rw, swapSpec)
	sp := NewPeer(p, swap)
	swap.setPeer(sp)
	defer swap.deletePeer(sp)
	defer swap.Close()
	return sp.Run(sp.handleSwapMsg)
}

func (p *SwapProtocol) APIs() []rpc.API {
	apis := []rpc.API{
		{
			Namespace: "swap",
			Version:   "1.0",
			Service:   NewAPI(p),
			Public:    true,
		},
	}
	return apis
}

//--------------------

func (swap *SwapProtocol) NodeInfo() interface{} {
	return nil
}

func (swap *SwapProtocol) PeerInfo(id discover.NodeID) interface{} {
	return nil
}

func (swap *SwapProtocol) Close() error {
	return nil
}

func (swap *SwapProtocol) getPeer(peerId discover.NodeID) *SwapPeer {
	swap.peersMu.RLock()
	defer swap.peersMu.RUnlock()

	return swap.peers[peerId]
}

func (swap *SwapProtocol) setPeer(peer *SwapPeer) {
	swap.peersMu.Lock()
	defer swap.peersMu.Unlock()

	swap.peers[peer.ID()] = peer
	metrics.GetOrRegisterGauge("registry.peers", nil).Update(int64(len(swap.peers)))
}

func (swap *SwapProtocol) deletePeer(peer *SwapPeer) {
	swap.peersMu.Lock()
	defer swap.peersMu.Unlock()

	delete(swap.peers, peer.ID())
	metrics.GetOrRegisterGauge("registry.peers", nil).Update(int64(len(swap.peers)))
	swap.peersMu.Unlock()
}

func (swap *SwapProtocol) peersCount() (c int) {
	swap.peersMu.Lock()
	c = len(swap.peers)
	swap.peersMu.Unlock()
	return
}

func (p *SwapPeer) handleSwapMsg(ctx context.Context, msg interface{}) error {
	switch msg := msg.(type) {

	case *IssueChequeMsg:
		return p.handleIssueChequeMsg(ctx, msg)

	case *RedeemChequeMsg:
		return p.handleRedeemChequeMsg(ctx, msg)

	/*
		case *QuitMsg:
			return p.handleQuitMsg(msg)
	*/

	default:
		return fmt.Errorf("unknown message type: %T", msg)
	}
	return nil
}

func (sp *SwapPeer) handleIssueChequeMsg(ctx context.Context, msg interface{}) (err error) {
	return err
}

func (sp *SwapPeer) handleRedeemChequeMsg(ctx context.Context, msg interface{}) (err error) {
	return err
}
