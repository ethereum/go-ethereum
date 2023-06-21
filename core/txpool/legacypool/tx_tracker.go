// Copyright 2023 The go-ethereum Authors
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

// Package legacypool implements the normal EVM execution transaction pool.
package legacypool

import (
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"golang.org/x/exp/slices"
)

var recheckInterval = 10 * time.Second

// TxTracker is a struct used to track priority transactions; it will check from
// time to time if the main pool has forgotten about any of the transaction
// it is tracking, and if so, submit it again.
// This is used to track 'locals'.
// This struct does not care about transaction validity, price-bumps or account limits,
// but optimistically accepts transactions.
type TxTracker struct {
	all    map[common.Hash]*types.Transaction // All tracked transactions
	byAddr map[common.Address]*sortedMap      // Transactions by address

	journal  *journal       // Journal of local transaction to back up to disk
	modified bool           // Modification tracking
	pool     txpool.SubPool // The 'main' subpool to interact with
	signer   types.Signer

	shutdownCh chan struct{}
	mu         sync.Mutex
	wg         sync.WaitGroup
}

func NewTxTracker(journalPath string, chainConfig *params.ChainConfig, next txpool.SubPool) *TxTracker {
	signer := types.LatestSigner(chainConfig)
	pool := &TxTracker{
		all:        make(map[common.Hash]*types.Transaction),
		byAddr:     make(map[common.Address]*sortedMap),
		signer:     signer,
		shutdownCh: make(chan struct{}),
		pool:       next,
	}
	if journalPath != "" {
		pool.journal = newTxJournal(journalPath)
	}
	return pool
}

// Track adds a transaction tx to the tracked set.
func (tracker *TxTracker) Track(tx *types.Transaction) {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	// If we're already tracking it, it's a no-op
	if _, ok := tracker.all[tx.Hash()]; ok {
		return
	}
	tracker.all[tx.Hash()] = tx
	addr, _ := types.Sender(tracker.signer, tx)
	if tracker.byAddr[addr] == nil {
		tracker.byAddr[addr] = newSortedMap()
	}
	tracker.byAddr[addr].Put(tx)
	tracker.modified = true
}

// recheck checks and returns any transactions that needs to be resubmitted.
func (tracker *TxTracker) recheck() []*txpool.Transaction {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()

	if !tracker.modified {
		return nil
	}
	var resubmits []*txpool.Transaction
	for sender, txs := range tracker.byAddr {
		stales := txs.Forward(tracker.pool.Nonce(sender))
		// Wipe the stales
		for _, tx := range stales {
			delete(tracker.all, tx.Hash())
		}
		// Check the non-stale
		for _, tx := range txs.Flatten() {
			if tracker.pool.Has(tx.Hash()) {
				continue
			}
			resubmits = append(resubmits, &txpool.Transaction{
				Tx: tx,
			})
		}
	}

	{ // rejournal
		txs := make(map[common.Address]types.Transactions)
		for _, tx := range tracker.all {
			addr, _ := types.Sender(tracker.signer, tx)
			txs[addr] = append(txs[addr], tx)
		}
		// Sort them
		for _, list := range txs {
			slices.SortFunc(list, func(a, b *types.Transaction) bool {
				return a.Nonce() < b.Nonce()
			})
		}
		if err := tracker.journal.rotate(txs); err != nil {
			log.Warn("Transaction journal rotation failed", "err", err)
		}
	}
	return resubmits
}

// Start implements node.Lifecycle interface
// Start is called after all services have been constructed and the networking
// layer was also initialized to spawn any goroutines required by the service.
func (tracker *TxTracker) Start() error {
	tracker.wg.Add(1)
	go tracker.loop()
	return nil
}

// Start implements node.Lifecycle interface
// Stop terminates all goroutines belonging to the service, blocking until they
// are all terminated.
func (tracker *TxTracker) Stop() error {
	close(tracker.shutdownCh)
	tracker.wg.Wait()
	return nil
}

func (tracker *TxTracker) loop() {
	defer tracker.wg.Done()
	t := time.NewTimer(recheckInterval)
	for {
		select {
		case <-tracker.shutdownCh:
			return
		case <-t.C:
			// resubmit
			if resubmits := tracker.recheck(); len(resubmits) > 0 {
				tracker.pool.Add(resubmits, false, false)
			}
			t.Reset(recheckInterval)
		}
	}
}
