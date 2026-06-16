// Copyright 2026 The go-ethereum Authors
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

package miner

import (
	"fmt"
	"github.com/ethereum/go-ethereum/core/txpool/txorder"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
)

// resolveableTransaction represents a transaction of which some of the fields
// are available, but the full transaction will be resolved from some source.
type resolveableTransaction interface {
	Hash() common.Hash
	Resolve() *types.Transaction
	Gas() uint64
	BlobGas() uint64
}

// lazyTransaction wraps a txpool.LazyTransaction and implements resolveableTransaction.
type lazyTransaction struct {
	*txpool.LazyTransaction
}

func (l *lazyTransaction) Resolve() *types.Transaction {
	return l.LazyTransaction.Resolve()
}

func (l *lazyTransaction) Gas() uint64 {
	return l.LazyTransaction.Gas
}

func (l *lazyTransaction) Hash() common.Hash {
	return l.LazyTransaction.Hash
}

func (l *lazyTransaction) BlobGas() uint64 {
	return l.LazyTransaction.BlobGas
}

// resolvedTransaction implements resolveableTransaction for a transaction
// that was resolved in the first place.  used for building payloads with
// tx set overrides
type resolvedTransaction struct {
	*types.Transaction
}

func (r *resolvedTransaction) Gas() uint64 {
	return r.Transaction.Gas()
}

func (r *resolvedTransaction) Resolve() *types.Transaction {
	return r.Transaction
}

// transactionQueue represents a list of transactions
// which are queued for inclusion.
type transactionQueue interface {
	// Shift removes all transactions from the sender of the
	// current highest-priority transaction for inclusion
	Shift()
	// Pop removes the highest-priority transaction from
	// the queue.
	Pop()
	// HasBlobTxs returns true if the sender of the current
	// highest-priority transaction has queued blob transactions.
	HasBlobTxs() bool
	// ClearBlobTxs removes all blob txs from the sender of the
	// current highest-priority transaction from the queue.
	ClearBlobTxs()
}

// transactionSource is a source that the miner can
type transactionSource interface {
	// Peek returns the next transaction which should be evaluated for inclusion
	// and a transactionQueue for the sender account.
	Peek() (resolveableTransaction, transactionQueue)
}

// plainTxQueue implements transactionQueue for a set of non-blob transactions
type plainTxQueue struct {
	txs *txorder.TransactionsByPriceAndNonce
}

func (q *plainTxQueue) Shift() { q.txs.Shift() }
func (q *plainTxQueue) Pop()   { q.txs.Pop() }

// HasBlobTxs always returns false: plain queues never carry blob transactions.
func (q *plainTxQueue) HasBlobTxs() bool { return false }

// ClearBlobTxs is a no-op for plain queues and always returns false.
func (q *plainTxQueue) ClearBlobTxs() {}

// blobTxQueue implements transactionQueue for a set of blob transactions
type blobTxQueue struct {
	txs *txorder.TransactionsByPriceAndNonce
}

func (q *blobTxQueue) Shift() {
	if q.txs != nil {
		q.txs.Shift()
	}
}

func (q *blobTxQueue) Pop() {
	if q.txs != nil {
		q.txs.Pop()
	}
}

// HasBlobTxs always returns true: this queue exclusively holds blob transactions.
func (q *blobTxQueue) HasBlobTxs() bool { return true }

func (q *blobTxQueue) ClearBlobTxs() {
	if q.txs != nil {
		q.txs.Clear()
	}
}

// feeOrderedTxSource implements transactionSource for the standard strategy
// of transaction inclusion: prioritize by profitability.
type feeOrderedTxSource struct {
	plainQueue *plainTxQueue
	blobQueue  *blobTxQueue
}

// newFeeOrderedTxSource creates a feeOrderedTxSource from separate plain and blob
// transaction sets. Either argument may be nil (e.g. when there are no blob
// transactions pending).
func newFeeOrderedTxSource(plain, blob *txorder.TransactionsByPriceAndNonce) *feeOrderedTxSource {
	s := &feeOrderedTxSource{
		plainQueue: &plainTxQueue{txs: plain},
		blobQueue:  &blobTxQueue{txs: blob},
	}

	return s
}

// Peek returns the next most profitable queued transaction
func (s *feeOrderedTxSource) Peek() (resolveableTransaction, transactionQueue) {
	pTx, pTip := s.plainQueue.txs.Peek()
	bTx, bTip := s.blobQueue.txs.Peek()

	switch {
	case pTx == nil && bTx == nil:
		return nil, nil
	case pTx == nil:
		return &lazyTransaction{bTx}, s.blobQueue
	case bTx == nil:
		return &lazyTransaction{pTx}, s.plainQueue
	default:
		if bTip.Gt(pTip) {
			return &lazyTransaction{bTx}, s.blobQueue
		} else {
			return &lazyTransaction{pTx}, s.plainQueue
		}
	}
}

// orderedTxSource implements transactionSource and transactionQueue.
// The transaction set and order is based on a pre-determined list.
type orderedTxSource struct {
	// txs ordered as they are intended to be included in a payload
	txs types.Transactions
	// txs ordered by sender and nonce
	orderedTxs map[common.Address][]*types.Transaction
	signer     types.Signer
}

func newOrderedTxSource(txs types.Transactions, signer types.Signer) (*orderedTxSource, error) {
	orderedTxs := make(map[common.Address][]*types.Transaction)
	for _, tx := range txs {
		from, _ := signer.Sender(tx)
		if _, ok := orderedTxs[from]; !ok {
			orderedTxs[from] = []*types.Transaction{tx}
			continue
		}

		// validate that no transactions from the same sender have conflicting nonces.
		senderTxs := orderedTxs[from]
		for _, senderTx := range senderTxs {
			if tx.Nonce() == senderTx.Nonce() {
				return nil, fmt.Errorf("conflicting transaction nonces")
			}
		}

		var (
			insertionPoint int
			senderTx       *types.Transaction
		)
		for insertionPoint, senderTx = range senderTxs {
			if senderTx.Nonce() > tx.Nonce() {
				break
			}
		}
		orderedTxs[from] = slices.Insert(senderTxs, insertionPoint, tx)
	}
	return &orderedTxSource{txs: txs, signer: signer, orderedTxs: orderedTxs}, nil
}

func (s *orderedTxSource) Pop() {
	from, _ := s.signer.Sender(s.txs[0])
	s.txs = slices.DeleteFunc(s.txs, func(tx *types.Transaction) bool {
		txSender, _ := s.signer.Sender(tx)
		return txSender == from
	})
	delete(s.orderedTxs, from)
}

func (s *orderedTxSource) Shift() {
	delTx := s.txs[0]
	sender, _ := s.signer.Sender(delTx)

	// application order of txs might not be by nonce,
	// so deletion from the nonce-ordered set requires
	// search
	s.orderedTxs[sender] = slices.DeleteFunc(s.orderedTxs[sender], func(tx *types.Transaction) bool {
		return delTx.Hash() == tx.Hash()
	})

	s.txs = s.txs[1:]
}

func (s *orderedTxSource) HasBlobTxs() bool {
	for _, tx := range s.txs {
		if tx.Type() == types.BlobTxType {
			return true
		}
	}
	return false
}

func (s *orderedTxSource) ClearBlobTxs() {
	var remainingTxs types.Transactions
	for _, tx := range s.txs {
		if tx.Type() != types.BlobTxType {
			remainingTxs = append(remainingTxs, tx)
		}
	}
	s.txs = remainingTxs
}

func (s *orderedTxSource) Peek() (resolveableTransaction, transactionQueue) {
	if len(s.txs) == 0 {
		return nil, nil
	}
	return &resolvedTransaction{s.txs[0]}, s
}
