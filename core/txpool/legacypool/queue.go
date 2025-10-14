// Copyright 2025 The go-ethereum Authors
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

package legacypool

import (
	"sort"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// queue manages nonce-gapped transactions that have been validated but are
// not yet processable.
type queue struct {
	config Config
	signer types.Signer
	queued map[common.Address]*list     // Queued but non-processable transactions
	beats  map[common.Address]time.Time // Last heartbeat from each known account
}

func newQueue(config Config, signer types.Signer) *queue {
	return &queue{
		signer: signer,
		config: config,
		queued: make(map[common.Address]*list),
		beats:  make(map[common.Address]time.Time),
	}
}

// evictList returns the hashes of transactions that are old enough to be evicted.
func (q *queue) evictList() []common.Hash {
	var removed []common.Hash
	for addr, list := range q.queued {
		if time.Since(q.beats[addr]) > q.config.Lifetime {
			for _, tx := range list.Flatten() {
				removed = append(removed, tx.Hash())
			}
		}
	}
	queuedEvictionMeter.Mark(int64(len(removed)))
	return removed
}

func (q *queue) stats() int {
	queued := 0
	for _, list := range q.queued {
		queued += list.Len()
	}
	return queued
}

func (q *queue) content() map[common.Address][]*types.Transaction {
	queued := make(map[common.Address][]*types.Transaction, len(q.queued))
	for addr, list := range q.queued {
		queued[addr] = list.Flatten()
	}
	return queued
}

func (q *queue) contentFrom(addr common.Address) []*types.Transaction {
	var queued []*types.Transaction
	if list, ok := q.get(addr); ok {
		queued = list.Flatten()
	}
	return queued
}

func (q *queue) get(addr common.Address) (*list, bool) {
	l, ok := q.queued[addr]
	return l, ok
}

func (q *queue) bump(addr common.Address) {
	q.beats[addr] = time.Now()
}

func (q *queue) addresses() []common.Address {
	addrs := make([]common.Address, 0, len(q.queued))
	for addr := range q.queued {
		addrs = append(addrs, addr)
	}
	return addrs
}

func (q *queue) remove(addr common.Address, tx *types.Transaction) {
	if future := q.queued[addr]; future != nil {
		if txOld := future.txs.Get(tx.Nonce()); txOld != nil && txOld.Hash() != tx.Hash() {
			// Edge case, a different transaction
			// with the same nonce is in the queued, just ignore
			return
		}
		if removed, _ := future.Remove(tx); removed {
			// Reduce the queued counter
			queuedGauge.Dec(1)
		}
		if future.Empty() {
			delete(q.queued, addr)
			delete(q.beats, addr)
		}
	}
}

func (q *queue) add(tx *types.Transaction) (*common.Hash, error) {
	// Try to insert the transaction into the future queue
	from, _ := types.Sender(q.signer, tx) // already validated
	if q.queued[from] == nil {
		q.queued[from] = newList(false)
	}
	inserted, old := q.queued[from].Add(tx, q.config.PriceBump)
	if !inserted {
		// An older transaction was better, discard this
		queuedDiscardMeter.Mark(1)
		return nil, txpool.ErrReplaceUnderpriced
	}
	// If we never record the heartbeat, do it right now.
	if _, exist := q.beats[from]; !exist {
		q.beats[from] = time.Now()
	}
	if old == nil {
		// Nothing was replaced, bump the queued counter
		queuedGauge.Inc(1)
		return nil, nil
	}
	h := old.Hash()
	// Transaction was replaced, bump the replacement counter
	queuedReplaceMeter.Mark(1)
	return &h, nil
}

// promoteExecutables iterates over all accounts with queued transactions, selecting
// for promotion any that are now executable. It also drops any transactions that are
// deemed too old (nonce too low) or too costly (insufficient funds or over gas limit).
//
// Returns three lists:
// - all transactions that were removed from the queue and selected for promotion;
// - all other transactions that were removed from the queue and dropped;
// - the list of addresses removed.
func (q *queue) promoteExecutables(accounts []common.Address, gasLimit uint64, currentState *state.StateDB, nonces *noncer) ([]*types.Transaction, []common.Hash, []common.Address) {
	// Track the promotable transactions to broadcast them at once
	var (
		promotable       []*types.Transaction
		dropped          []common.Hash
		removedAddresses []common.Address
	)
	// Iterate over all accounts and promote any executable transactions
	for _, addr := range accounts {
		list := q.queued[addr]
		if list == nil {
			continue // Just in case someone calls with a non existing account
		}
		// Drop all transactions that are deemed too old (low nonce)
		forwards := list.Forward(currentState.GetNonce(addr))
		for _, tx := range forwards {
			dropped = append(dropped, tx.Hash())
		}
		log.Trace("Removing old queued transactions", "count", len(forwards))

		// Drop all transactions that are too costly (low balance or out of gas)
		drops, _ := list.Filter(currentState.GetBalance(addr), gasLimit)
		for _, tx := range drops {
			dropped = append(dropped, tx.Hash())
		}
		log.Trace("Removing unpayable queued transactions", "count", len(drops))
		queuedNofundsMeter.Mark(int64(len(drops)))

		// Gather all executable transactions and promote them
		readies := list.Ready(nonces.get(addr))
		promotable = append(promotable, readies...)
		log.Trace("Promoting queued transactions", "count", len(promotable))
		queuedGauge.Dec(int64(len(readies)))

		// Drop all transactions over the allowed limit
		var caps = list.Cap(int(q.config.AccountQueue))
		for _, tx := range caps {
			hash := tx.Hash()
			dropped = append(dropped, hash)
			log.Trace("Removing cap-exceeding queued transaction", "hash", hash)
		}
		queuedRateLimitMeter.Mark(int64(len(caps)))

		// Delete the entire queue entry if it became empty.
		if list.Empty() {
			delete(q.queued, addr)
			delete(q.beats, addr)
			removedAddresses = append(removedAddresses, addr)
		}
	}
	queuedGauge.Dec(int64(len(dropped)))
	return promotable, dropped, removedAddresses
}

// truncate drops the oldest transactions from the queue until the total
// number is below the configured limit. Returns the hashes of all dropped
// transactions and the addresses of accounts that became empty due to
// the truncation.
func (q *queue) truncate() ([]common.Hash, []common.Address) {
	queued := uint64(0)
	for _, list := range q.queued {
		queued += uint64(list.Len())
	}
	if queued <= q.config.GlobalQueue {
		return nil, nil
	}

	// Sort all accounts with queued transactions by heartbeat
	addresses := make(addressesByHeartbeat, 0, len(q.queued))
	for addr := range q.queued {
		addresses = append(addresses, addressByHeartbeat{addr, q.beats[addr]})
	}
	sort.Sort(sort.Reverse(addresses))

	// Drop transactions until the total is below the limit
	var (
		removed          = make([]common.Hash, 0)
		removedAddresses = make([]common.Address, 0)
	)
	for drop := queued - q.config.GlobalQueue; drop > 0 && len(addresses) > 0; {
		addr := addresses[len(addresses)-1]
		list := q.queued[addr.address]

		addresses = addresses[:len(addresses)-1]

		// Drop all transactions if they are less than the overflow
		if size := uint64(list.Len()); size <= drop {
			for _, tx := range list.Flatten() {
				q.remove(addr.address, tx)
				removed = append(removed, tx.Hash())
			}
			drop -= size
			queuedRateLimitMeter.Mark(int64(size))
			removedAddresses = append(removedAddresses, addr.address)
			continue
		}
		// Otherwise drop only last few transactions
		txs := list.Flatten()
		for i := len(txs) - 1; i >= 0 && drop > 0; i-- {
			q.remove(addr.address, txs[i])
			removed = append(removed, txs[i].Hash())
			drop--
			queuedRateLimitMeter.Mark(1)
		}
	}
	// No need to clear empty accounts, remove already does that
	return removed, removedAddresses
}

// addressByHeartbeat is an account address tagged with its last activity timestamp.
type addressByHeartbeat struct {
	address   common.Address
	heartbeat time.Time
}

type addressesByHeartbeat []addressByHeartbeat

func (a addressesByHeartbeat) Len() int           { return len(a) }
func (a addressesByHeartbeat) Less(i, j int) bool { return a[i].heartbeat.Before(a[j].heartbeat) }
func (a addressesByHeartbeat) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
