// Copyright 2020 The go-ethereum Authors
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

package lotterybook

import (
	"context"
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/prque"
	"github.com/ethereum/go-ethereum/contracts/lotterybook/contract"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/log"
)

var errChequeManagerClosed = errors.New("cheque manager closed")

// chequeManager the manager of received cheques for all life cycle management.
type chequeManager struct {
	address  common.Address
	chain    Blockchain
	contract *contract.LotteryBook
	cdb      *chequeDB
	queryCh  chan chan []*Cheque
	chequeCh chan *Cheque
	closeCh  chan struct{}
	wg       sync.WaitGroup
	claim    func(context.Context, *Cheque) error
}

// newChequeManager returns an instance of cheque manager and starts
// the underlying routines for status management.
func newChequeManager(address common.Address, chain Blockchain, contract *contract.LotteryBook, cdb *chequeDB, claim func(context.Context, *Cheque) error) *chequeManager {
	mgr := &chequeManager{
		address:  address,
		chain:    chain,
		contract: contract,
		cdb:      cdb,
		queryCh:  make(chan chan []*Cheque),
		chequeCh: make(chan *Cheque),
		closeCh:  make(chan struct{}),
		claim:    claim,
	}
	mgr.wg.Add(1)
	go mgr.run()
	return mgr
}

// run starts a background routine for cheques management.
func (m *chequeManager) run() {
	defer m.wg.Done()

	// Establish subscriptions
	newHeadCh := make(chan core.ChainHeadEvent, 1024)
	sub := m.chain.SubscribeChainHeadEvent(newHeadCh)
	if sub == nil {
		return
	}
	defer sub.Unsubscribe()

	var (
		current = m.chain.CurrentHeader().Number.Uint64()

		// todo if we have lots of cheques maintained in the memory,
		// it will lead to OOM. We need a better mechanism to load
		// cheques by demand.
		//
		// leave the todo here. Seems if there are N valid cheques,
		// then it means there are at least N different lotteries.
		// If the malicious client want to attack us, it has to submit
		// lots of lotteries which in theory is impossible(cost is too
		// high).
		active      = make(map[common.Hash]*Cheque)
		indexes     = make(map[common.Hash]int)
		activeQueue = prque.New(func(data interface{}, index int) {
			cheque := data.(*Cheque)
			indexes[cheque.LotteryId] = index
			active[cheque.LotteryId] = cheque
		})
	)
	// checkAndClaim checks whether the cheque is the winner or not.
	// If so, claim the corresponding lottery via sending on-chain
	// transaction.
	checkAndClaim := func(cheque *Cheque, hash common.Hash) (err error) {
		defer func() {
			// No matter we aren't the lucky winner or we already claim
			// the lottery, delete the record anyway. Keep it if any error
			// occurs.
			if err == nil {
				m.cdb.deleteCheque(m.address, cheque.Signer(), cheque.LotteryId, false)
			}
		}()
		if !cheque.reveal(hash) {
			loseLotteryGauge.Inc(1)
			return nil
		}
		// todo(rjl493456442) if any error occurs(but we are the lucky winner), re-try
		// is necesssary. Most of the failures can be timeout, signing failures, etc.
		winLotteryGauge.Inc(1)
		ctx, cancelFn := context.WithTimeout(context.Background(), txTimeout)
		defer cancelFn()
		return m.claim(ctx, cheque)
	}
	// Read all stored cheques received locally
	cheques, drawers := m.cdb.listCheques(m.address, nil)
	for index, cheque := range cheques {
		// The valid claim block range is [revealNumber+1, revealNumber+256).
		// However the head block can be reorged with very high chance. So
		// a small processing confirms is applied to ensure the reveal hash
		// is stable enough.
		//
		// For receiver, the reasonable claim range [revealNumber+6, revealNumber+256).
		if current < cheque.RevealNumber+lotteryProcessConfirms {
			activeQueue.Push(cheque, -int64(cheque.RevealNumber))
		} else if current < cheque.RevealNumber+lotteryClaimPeriod {
			// Lottery can still be claimed, try it!
			revealHash := m.chain.GetHeaderByNumber(cheque.RevealNumber)

			// Create an independent routine to claim the lottery.
			// This function may takes very long time, don't block
			// the entire thread here. It's ok to spin up routines
			// blindly here, there won't have too many cheques to claim.
			go checkAndClaim(cheque, revealHash.Hash())
		} else {
			// Lottery is already out of claim window, delete it.
			log.Debug("Cheque expired", "lotteryid", cheque.LotteryId)
			m.cdb.deleteCheque(m.address, drawers[index], cheque.LotteryId, false)
		}
	}
	for {
		select {
		case ev := <-newHeadCh:
			current = ev.Block.NumberU64()

		checkExpiration:
			for !activeQueue.Empty() {
				item, priority := activeQueue.Pop()
				height := uint64(-priority)

				// Short circuit if they are still active lotteries.
				if current < height+lotteryProcessConfirms {
					activeQueue.Push(item, priority)
					break checkExpiration
				}
				// Wipe the cheque if it's already stale.
				cheque := item.(*Cheque)
				delete(indexes, cheque.LotteryId)
				delete(active, cheque.LotteryId)

				if current < height+lotteryClaimPeriod {
					// Create an independent routine to claim the lottery.
					// This function may takes very long time, don't block
					// the entire thread here. It's ok to spin up routines
					// blindly here, there won't have too many cheques to claim.
					go checkAndClaim(cheque, m.chain.GetHeaderByNumber(height).Hash())
					continue
				}
				m.cdb.deleteCheque(m.address, cheque.Signer(), cheque.LotteryId, false)
			}

		case cheque := <-m.chequeCh:
			if index, exist := indexes[cheque.LotteryId]; exist {
				activeQueue.Remove(index)
			}
			activeQueue.Push(cheque, -int64(cheque.RevealNumber))

		case retCh := <-m.queryCh:
			var ret []*Cheque
			for _, cheque := range active {
				ret = append(ret, cheque)
			}
			retCh <- ret

		case <-m.closeCh:
			return
		}
	}
}

// trackCheque adds a newly received cheque for life cycle management.
func (m *chequeManager) trackCheque(cheque *Cheque) error {
	select {
	case m.chequeCh <- cheque:
		return nil
	case <-m.closeCh:
		return errChequeManagerClosed
	}
}

// activeCheques returns all active cheques received which is
// waiting for reveal.
func (m *chequeManager) activeCheques() []*Cheque {
	reqCh := make(chan []*Cheque, 1)
	select {
	case m.queryCh <- reqCh:
		return <-reqCh
	case <-m.closeCh:
		return nil
	}
}

func (m *chequeManager) close() {
	close(m.closeCh)
	m.wg.Wait()
}
