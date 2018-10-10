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
	"errors"
	"fmt"
	"strconv"
	"sync"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	"github.com/ethereum/go-ethereum/swarm/log"
	"github.com/ethereum/go-ethereum/swarm/state"
)

// SwAP Swarm Accounting Protocol
// a peer to peer micropayment system
// A node maintains an individual balance with every peer
// Only messages which have a price will be accounted for
type Swap struct {
	stateStore state.Store        //stateStore is needed in order to keep balances across sessions
	lock       sync.RWMutex       //lock the balances
	balances   map[enode.ID]int64 //map of balances for each peer
}

//Credit us and debit remote
func (s *Swap) Credit(peer *protocols.Peer, amount uint64) (err error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.loadState(peer)

	s.balances[peer.ID()] += int64(amount)
	peerBalance := s.balances[peer.ID()]
	s.stateStore.Put(peer.ID().String(), &peerBalance)

	log.Debug(fmt.Sprintf("balance for peer %s: %s", peer.ID().String(), strconv.FormatInt(peerBalance, 10)))
	return err
}

//Debit us and credit remote
func (s *Swap) Debit(peer *protocols.Peer, amount uint64) (err error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.loadState(peer)

	//local node is being debited (in favor of remote peer), so its balance decreases
	s.balances[peer.ID()] -= int64(amount)
	peerBalance := s.balances[peer.ID()]
	s.stateStore.Put(peer.ID().String(), &peerBalance)

	log.Debug(fmt.Sprintf("balance for peer %s: %s", peer.ID().String(), strconv.FormatInt(peerBalance, 10)))
	return nil
}

//get a peer's balance
func (swap *Swap) GetPeerBalance(peer enode.ID) (int64, error) {
	swap.lock.RLock()
	defer swap.lock.RUnlock()
	if p, ok := swap.balances[peer]; ok {
		return p, nil
	}
	return 0, errors.New("Peer not found")
}

func (s *Swap) loadState(peer *protocols.Peer) {
	var peerBalance int64
	peerID := peer.ID()
	if _, ok := s.balances[peerID]; !ok {
		s.stateStore.Get(peerID.String(), &peerBalance)
		s.balances[peerID] = peerBalance
	}
}

// New - swap constructor
func New(stateStore state.Store) (swap *Swap) {

	swap = &Swap{
		stateStore: stateStore,
		balances:   make(map[enode.ID]int64),
	}
	return
}
