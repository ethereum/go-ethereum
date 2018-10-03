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

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/log"
)

const (
	IsActiveProtocol = true
)

//Here we define the Swap p2p.protocol
//We use it to standardize the interaction between nodes who need clear their accounts.
//Accounting itself is separate (see swarm/swap/swap.go)
//This protocol focuses on sending and receiving cheques and
//cheque management/clearing
//For a better understanding, please read the Swap network paper "Generalized Swap swear and swindle games"
type Protocol struct {
	peersMu sync.RWMutex
	peers   map[enode.ID]*Peer
}

//This is the peer representing a participant in this protocol
type Peer struct {
	*protocols.Peer
}

//Create a new protocol instance
func NewProtocol() *Protocol {
	proto := &Protocol{
		peers: make(map[enode.ID]*Peer),
	}
	return proto
}

// NewPeer is the constructor for the protocol Peer
func NewPeer(peer *protocols.Peer) *Peer {
	p := &Peer{
		Peer: peer,
	}
	return p
}

//In a peer exchange, if node A gets too indebted with node B,
//node A issues a cheque and sends it to B
type IssueChequeMsg struct {
	Cheque *Cheque
}

//In a scenario where B received a cheque from A, node B can
//redeem a cheque, which means it kicks off the process to cash it in.
//In this case, this message is sent to peer A
type RedeemChequeMsg struct {
	Cheque *Cheque
}

/////////////////////////////////////////////////////////////////////
// SECTION: node.Service interface
/////////////////////////////////////////////////////////////////////
func (s *Swap) Start(srv *p2p.Server) error {
	log.Debug("Started swap")
	return nil
}

func (s *Swap) Stop() error {
	log.Info("swap shutting down")
	return nil
}

var swapSpec = &protocols.Spec{
	Name:       swapProtocolName,
	Version:    swapVersion,
	MaxMsgSize: defaultMaxMsgSize,
	Messages: []interface{}{
		IssueChequeMsg{},
		RedeemChequeMsg{},
	},
}

func (s *Swap) Protocols() []p2p.Protocol {
	return []p2p.Protocol{
		{
			Name:    swapSpec.Name,
			Version: swapSpec.Version,
			Length:  swapSpec.Length(),
			Run:     s.run,
		},
	}
}

func (s *Swap) APIs() []rpc.API {
	apis := []rpc.API{
		{
			Namespace: "swap",
			Version:   "1.0",
			Service:   NewAPI(s),
			Public:    true,
		},
	}
	return apis
}

/////////////////////////////////////////////////////////////////////
// SECTION: p2p.protocol interface
/////////////////////////////////////////////////////////////////////
func (s *Swap) run(peer *p2p.Peer, rw p2p.MsgReadWriter) error {
	p := protocols.NewPeer(peer, rw, swapSpec)
	sp := NewPeer(p)
	s.protocol.setPeer(sp)
	defer s.protocol.deletePeer(sp)
	defer s.protocol.Close()
	return sp.Run(sp.handleSwapMsg)
}

func (s *Swap) NodeInfo() interface{} {
	return nil
}

func (s *Swap) PeerInfo(id enode.ID) interface{} {
	return nil
}

//------------------------------------------------------------------------------------------
func (proto *Protocol) Close() {
}

func (proto *Protocol) getPeer(peerId enode.ID) *Peer {
	proto.peersMu.RLock()
	defer proto.peersMu.RUnlock()

	return proto.peers[peerId]
}

func (proto *Protocol) setPeer(peer *Peer) {
	proto.peersMu.Lock()
	defer proto.peersMu.Unlock()

	proto.peers[peer.ID()] = peer
}

func (proto *Protocol) deletePeer(peer *Peer) {
	proto.peersMu.Lock()
	defer proto.peersMu.Unlock()

	delete(proto.peers, peer.ID())
}

func (proto *Protocol) peersCount() (c int) {
	proto.peersMu.Lock()
	c = len(proto.peers)
	proto.peersMu.Unlock()
	return
}

//Protocol message handler for handling cheque messages
func (p *Peer) handleSwapMsg(ctx context.Context, msg interface{}) error {
	switch msg := msg.(type) {

	case *IssueChequeMsg:
		return p.handleIssueChequeMsg(ctx, msg)

	case *RedeemChequeMsg:
		return p.handleRedeemChequeMsg(ctx, msg)

	default:
		return fmt.Errorf("unknown message type: %T", msg)
	}
}

//A IssueChequeMsg has been received
func (p *Peer) handleIssueChequeMsg(ctx context.Context, msg interface{}) (err error) {
	log.Debug("SwapProtocolPeer: handleIssueChequeMsg")
	return err
}

//A RedeemChequeMsg has been received
func (p *Peer) handleRedeemChequeMsg(ctx context.Context, msg interface{}) (err error) {
	log.Debug("SwapProtocolPeer: handleRedeemChequeMsg")
	return err
}
