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
	"errors"
	"fmt"
	"math"
	"strconv"
	"sync"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	"github.com/ethereum/go-ethereum/swarm/log"
	"github.com/ethereum/go-ethereum/swarm/state"
)

const (
	defaultMaxMsgSize = 1024 * 1024
	OracleID          = "swap"
	swapProtocolName  = "swap"
	swapVersion       = 1
)

var (
	payAt  = int64(-4096 * 10000) // threshold that triggers payment {request} (bytes)
	dropAt = int64(-4096 * 12000) // threshold that triggers disconnect (bytes)

	ErrNotAccountedMsg   = errors.New("Message does not need accounting")
	ErrInsufficientFunds = errors.New("Insufficient funds")
)

// SwAP Swarm Accounting Protocol
// a peer to peer micropayment system
// A node maintains an individual balance with every peer
// Only messages which have a price will be accounted for
type Swap struct {
	chequeManager *ChequeManager     //cheque manager keeps track of issued cheques
	stateStore    state.Store        //stateStore is needed in order to keep balances across sessions
	lock          sync.RWMutex       //lock the balances
	balances      map[enode.ID]int64 //map of balances for each peer
	protocol      *Protocol          //reference to the cheque exchange protocol
}

//Credit us and debit remote
func (s *Swap) Credit(peer *protocols.Peer, amount uint64) (err error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	peerBalance := s.balances[peer.ID()]
	if _, ok := s.balances[peer.ID()]; !ok {
		s.stateStore.Get(peer.ID().String(), &peerBalance)
		s.balances[peer.ID()] = peerBalance
	}
	//local node is being credited(in favor of local node), so the balance increases
	s.balances[peer.ID()] += int64(amount)
	peerBalance = s.balances[peer.ID()]
	s.stateStore.Put(peer.ID().String(), &peerBalance)

	if float64(peerBalance) > math.Abs(float64(payAt)) {
		ctx := context.TODO()
		err := s.wantCheque(ctx, peer.ID())
		if err != nil {
			//TODO: special error handling, as at this point the accounting has been done
			//but the cheque could not be sent?
			log.Warn("Payment threshold exceeded, but error sending cheque!", "err", err)
		}
	}
	if float64(peerBalance) > math.Abs(float64(dropAt)) {
		return ErrInsufficientFunds
	}
	log.Debug(fmt.Sprintf("balance for peer %s: %s", peer.ID().String(), strconv.FormatInt(peerBalance, 10)))
	return err
}

//Debit us and credit remote
func (s *Swap) Debit(peer *protocols.Peer, amount uint64) (err error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	peerBalance := s.balances[peer.ID()]
	if _, ok := s.balances[peer.ID()]; !ok {
		s.stateStore.Get(peer.ID().String(), &peerBalance)
		s.balances[peer.ID()] = peerBalance
	}
	//local node is being debited (in favor of remote peer), so its balance decreases
	s.balances[peer.ID()] -= int64(amount)
	peerBalance = s.balances[peer.ID()]
	s.stateStore.Put(peer.ID().String(), &peerBalance)
	if peerBalance < payAt {
		ctx := context.TODO()
		err := s.issueCheque(ctx, peer.ID())
		if err != nil {
			//TODO: special error handling, as at this point the accounting has been done
			//but the cheque could not be sent?
			log.Warn("Payment threshold exceeded, but error sending cheque!", "err", err)
		}
	}
	if peerBalance < dropAt {
		return ErrInsufficientFunds
	}
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

//Issue a cheque for the remote peer. Happens if we are indebted with the peer
//and crossed the payment threshold
func (s *Swap) issueCheque(ctx context.Context, id enode.ID) error {
	cheque := s.chequeManager.CreateCheque(id, int64(math.Abs(float64(payAt))))
	_ = IssueChequeMsg{
		Cheque: cheque,
	}
	p := s.protocol.getPeer(id)
	if p == nil {
		return fmt.Errorf("wanting to send to non-connected peer!")
	}
	return nil
	//TODO: Don't actually send any cheques yet
	//return p.Send(ctx, msg)
}

//Issue a cheque for the remote peer. Happens if we are indebted with the peer
//and crossed the payment threshold
func (s *Swap) wantCheque(ctx context.Context, id enode.ID) error {
	//TODO: Don't actually initially any real message exchange yet
	//return p.Send(ctx, msg)
	return nil
}

// New - swap constructor
func New(stateStore state.Store) (swap *Swap) {

	swap = &Swap{
		chequeManager: NewChequeManager(stateStore),
		stateStore:    stateStore,
		balances:      make(map[enode.ID]int64),
		protocol:      NewProtocol(),
	}
	return
}
