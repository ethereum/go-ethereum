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

func (q *queue) evict(force bool) []common.Hash {
	removed := make([]common.Hash, 0)
	for addr, list := range q.queued {
		// Any transactions old enough should be removed
		if force || time.Since(q.beats[addr]) > q.config.Lifetime {
			list := list.Flatten()
			for _, tx := range list {
				q.removeTx(addr, tx)
				removed = append(removed, tx.Hash())
			}
			queuedEvictionMeter.Mark(int64(len(list)))
		}
	}
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

func (q queue) removeTx(addr common.Address, tx *types.Transaction) {
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

func (q *queue) add(hash common.Hash, tx *types.Transaction) (*common.Hash, error) {
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

func (q *queue) promoteExecutables(accounts []common.Address, gasLimit uint64, currentState *state.StateDB, nonces *noncer) ([]*types.Transaction, []common.Hash) {
	// Track the promoteable transactions to broadcast them at once
	var promoteable []*types.Transaction
	var removeable []common.Hash

	// Iterate over all accounts and promote any executable transactions
	for _, addr := range accounts {
		list := q.queued[addr]
		if list == nil {
			continue // Just in case someone calls with a non existing account
		}
		// Drop all transactions that are deemed too old (low nonce)
		forwards := list.Forward(currentState.GetNonce(addr))
		for _, tx := range forwards {
			removeable = append(removeable, tx.Hash())
		}
		log.Trace("Removing old queued transactions", "count", len(forwards))
		// Drop all transactions that are too costly (low balance or out of gas)
		drops, _ := list.Filter(currentState.GetBalance(addr), gasLimit)
		for _, tx := range drops {
			removeable = append(removeable, tx.Hash())
		}
		log.Trace("Removing unpayable queued transactions", "count", len(drops))
		queuedNofundsMeter.Mark(int64(len(drops)))

		// Gather all executable transactions and promote them
		readies := list.Ready(nonces.get(addr))
		promoteable = append(promoteable, readies...)
		log.Trace("Promoting queued transactions", "count", len(promoteable))
		queuedGauge.Dec(int64(len(readies)))

		// Drop all transactions over the allowed limit
		var caps = list.Cap(int(q.config.AccountQueue))
		for _, tx := range caps {
			hash := tx.Hash()
			removeable = append(removeable, hash)
			log.Trace("Removing cap-exceeding queued transaction", "hash", hash)
		}
		queuedRateLimitMeter.Mark(int64(len(caps)))
		queuedGauge.Dec(int64(len(removeable)))

		// Delete the entire queue entry if it became empty.
		if list.Empty() {
			delete(q.queued, addr)
			delete(q.beats, addr)
		}
	}
	return promoteable, removeable
}

func (q *queue) truncate() []common.Hash {
	queued := uint64(0)
	for _, list := range q.queued {
		queued += uint64(list.Len())
	}
	if queued <= q.config.GlobalQueue {
		return nil
	}

	// Sort all accounts with queued transactions by heartbeat
	addresses := make(addressesByHeartbeat, 0, len(q.queued))
	for addr := range q.queued {
		addresses = append(addresses, addressByHeartbeat{addr, q.beats[addr]})
	}
	sort.Sort(sort.Reverse(addresses))
	removed := make([]common.Hash, 0)

	// Drop transactions until the total is below the limit
	for drop := queued - q.config.GlobalQueue; drop > 0 && len(addresses) > 0; {
		addr := addresses[len(addresses)-1]
		list := q.queued[addr.address]

		addresses = addresses[:len(addresses)-1]

		// Drop all transactions if they are less than the overflow
		if size := uint64(list.Len()); size <= drop {
			for _, tx := range list.Flatten() {
				q.removeTx(addr.address, tx)
				removed = append(removed, tx.Hash())
			}
			drop -= size
			queuedRateLimitMeter.Mark(int64(size))
			continue
		}
		// Otherwise drop only last few transactions
		txs := list.Flatten()
		for i := len(txs) - 1; i >= 0 && drop > 0; i-- {
			q.removeTx(addr.address, txs[i])
			removed = append(removed, txs[i].Hash())
			drop--
			queuedRateLimitMeter.Mark(1)
		}
	}
	return removed
}
