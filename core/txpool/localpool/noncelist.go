package localpool

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

type nonceOrderedList map[uint64]*types.Transaction

func (n nonceOrderedList) enqueue(tx *types.Transaction) {
	if old, ok := n[tx.Nonce()]; ok {
		log.Warn("Replacing old local transaction with new", "old", old.Hash(), "new", tx.Hash())
	}
	n[tx.Nonce()] = tx
}

func (n nonceOrderedList) forAllPending(initialNonce uint64, call func(tx *types.Transaction)) {
	currentNonce := initialNonce
	for {
		currentNonce++
		tx, ok := n[currentNonce]
		if !ok {
			break
		}
		call(tx)
	}
}

func (n nonceOrderedList) content(initialNonce uint64) ([]*types.Transaction, []*types.Transaction) {
	var (
		pending   []*types.Transaction
		isPending = make(map[common.Hash]struct{})
		queued    []*types.Transaction
	)
	// collect all pending transactions
	n.forAllPending(initialNonce, func(tx *types.Transaction) {
		pending = append(pending, tx)
		isPending[tx.Hash()] = struct{}{}
	})
	// mark all transactions as queued that are not pending
	for _, tx := range n {
		if _, ok := isPending[tx.Hash()]; !ok {
			queued = append(queued, tx)
		}
	}
	return pending, queued
}

func (n nonceOrderedList) reorg(currentNonce uint64) []common.Hash {
	var (
		toDelete []uint64
		hashes   []common.Hash
	)
	for nonce := range n {
		if nonce <= currentNonce {
			toDelete = append(toDelete, nonce)
		}
	}
	for _, nonce := range toDelete {
		hashes = append(hashes, n[nonce].Hash())
		delete(n, nonce)
	}
	return hashes
}
