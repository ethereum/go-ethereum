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
package tracking

import (
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/txpool/legacypool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"golang.org/x/exp/slices"
)

var recheckInterval = time.Minute

// TxTracker is a struct used to track priority transactions; it will check from
// time to time if the main pool has forgotten about any of the transaction
// it is tracking, and if so, submit it again.
// This is used to track 'locals'.
// This struct does not care about transaction validity, price-bumps or account limits,
// but optimistically accepts transactions.
type TxTracker struct {
	all    map[common.Hash]*types.Transaction       // All tracked transactions
	byAddr map[common.Address]*legacypool.SortedMap // Transactions by address

	journal   *journal       // Journal of local transaction to back up to disk
	rejournal time.Duration  // How often to rotate journal
	pool      *txpool.TxPool // The tx pool to interact with
	signer    types.Signer

	shutdownCh chan struct{}
	triggerCh  chan struct{}
	mu         sync.Mutex
	wg         sync.WaitGroup
}

func NewTxTracker(journalPath string, journalTime time.Duration, chainConfig *params.ChainConfig, next *txpool.TxPool) *TxTracker {
	signer := types.LatestSigner(chainConfig)
	pool := &TxTracker{
		all:        make(map[common.Hash]*types.Transaction),
		byAddr:     make(map[common.Address]*legacypool.SortedMap),
		signer:     signer,
		shutdownCh: make(chan struct{}),
		triggerCh:  make(chan struct{}),
		pool:       next,
	}
	if journalPath != "" {
		pool.journal = newTxJournal(journalPath)
		pool.rejournal = journalTime
	}
	return pool
}

// Track adds a transaction to the tracked set.
func (tracker *TxTracker) Track(tx *types.Transaction) {
	tracker.TrackAll([]*types.Transaction{tx})
}

// TrackAll adds a list of transactions to the tracked set.
func (tracker *TxTracker) TrackAll(txs []*types.Transaction) {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	for _, tx := range txs {
		// If we're already tracking it, it's a no-op
		if _, ok := tracker.all[tx.Hash()]; ok {
			continue
		}
		tracker.all[tx.Hash()] = tx
		addr, _ := types.Sender(tracker.signer, tx)
		if tracker.byAddr[addr] == nil {
			tracker.byAddr[addr] = legacypool.NewSortedMap()
		}
		tracker.byAddr[addr].Put(tx)
		_ = tracker.journal.insert(tx)
	}
}

// recheck checks and returns any transactions that needs to be resubmitted.
func (tracker *TxTracker) recheck(journalCheck bool) (resubmits []*types.Transaction, rejournal map[common.Address]types.Transactions) {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	var (
		numStales = 0
		numOk     = 0
	)
	for sender, txs := range tracker.byAddr {
		stales := txs.Forward(tracker.pool.Nonce(sender))
		// Wipe the stales
		for _, tx := range stales {
			delete(tracker.all, tx.Hash())
		}
		numStales += len(stales)
		// Check the non-stale
		for _, tx := range txs.Flatten() {
			if tracker.pool.Has(tx.Hash()) {
				numOk++
				continue
			}
			resubmits = append(resubmits, tx)
		}
	}

	if journalCheck { // rejournal
		rejournal = make(map[common.Address]types.Transactions)
		for _, tx := range tracker.all {
			addr, _ := types.Sender(tracker.signer, tx)
			rejournal[addr] = append(rejournal[addr], tx)
		}
		// Sort them
		for _, list := range rejournal {
			// cmp(a, b) should return a negative number when a < b,
			slices.SortFunc(list, func(a, b *types.Transaction) int {
				return int(a.Nonce() - b.Nonce())
			})
		}
	}
	log.Debug("Tx tracker status", "need-resubmit", len(resubmits), "stale", numStales, "ok", numOk)
	return resubmits, rejournal
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

// TriggerRecheck triggers a recheck, whereby the tracker potentially resubmits
// transactions to the tx pool. This method is mainly useful for test purposes,
// in order to speed up the process.
func (tracker *TxTracker) TriggerRecheck() {
	tracker.triggerCh <- struct{}{}
}

func (tracker *TxTracker) loop() {
	defer tracker.wg.Done()
	if tracker.journal != nil {
		tracker.journal.load(func(transactions []*types.Transaction) []error {
			tracker.TrackAll(transactions)
			return nil
		})
		defer tracker.journal.close()
	}
	var lastJournal = time.Now()
	// Do initial check after 10 seconds, do rechecks more seldom.
	t := time.NewTimer(10 * time.Second)
	for {
		select {
		case <-tracker.shutdownCh:
			return
		case <-tracker.triggerCh:
			resubmits, _ := tracker.recheck(false)
			if len(resubmits) > 0 {
				tracker.pool.Add(resubmits, false)
			}
		case <-t.C:
			checkJournal := tracker.journal != nil && time.Since(lastJournal) > tracker.rejournal
			resubmits, rejournal := tracker.recheck(checkJournal)
			if len(resubmits) > 0 {
				tracker.pool.Add(resubmits, false)
			}
			if checkJournal {
				// Lock to prevent journal.rotate <-> journal.insert (via TrackAll) conflicts
				tracker.mu.Lock()
				lastJournal = time.Now()
				if err := tracker.journal.rotate(rejournal); err != nil {
					log.Warn("Transaction journal rotation failed", "err", err)
				}
				tracker.mu.Unlock()
			}
			t.Reset(recheckInterval)
		}
	}
}
