// Copyright 2014 The go-ethereum Authors
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
	"container/heap"
	"math/big"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/core/txpool"
	"github.com/XinFinOrg/XDPoSChain/core/types"
)

// txWithMinerFee wraps a transaction with its gas price or effective miner gasTipCap
type txWithMinerFee struct {
	tx   *txpool.LazyTransaction
	from common.Address
	fees *big.Int
}

// newTxWithMinerFee creates a wrapped transaction, calculating the effective
// miner gasTipCap if a base fee is provided.
// Returns error in case of a negative effective miner gasTipCap.
func newTxWithMinerFee(tx *txpool.LazyTransaction, from common.Address, baseFee *big.Int) (*txWithMinerFee, error) {
	tip := new(big.Int).Set(tx.GasTipCap)
	if baseFee != nil {
		if tx.GasFeeCap.Cmp(baseFee) < 0 {
			return nil, types.ErrGasFeeCapTooLow
		}
		effectiveTip := new(big.Int).Sub(tx.GasFeeCap, baseFee)
		if tip.Cmp(effectiveTip) > 0 {
			tip = effectiveTip
		}
	}
	return &txWithMinerFee{
		tx:   tx,
		from: from,
		fees: tip,
	}, nil
}

// TxByPriceAndTime implements both the sort and the heap interface, making it useful
// for all at once sorting as well as individually adding and removing elements.
type txByPriceAndTime struct {
	txs        []*txWithMinerFee
	payersSwap map[common.Address]*big.Int
}

func (s txByPriceAndTime) Len() int {
	return len(s.txs)
}

func (s txByPriceAndTime) Less(i, j int) bool {
	i_price := s.txs[i].fees
	if tx := s.txs[i].tx.Resolve(); tx != nil && tx.To() != nil {
		if _, ok := s.payersSwap[*tx.To()]; ok {
			i_price = common.TRC21GasPrice
		}
	}

	j_price := s.txs[j].fees
	if tx := s.txs[j].tx.Resolve(); tx != nil && tx.To() != nil {
		if _, ok := s.payersSwap[*tx.To()]; ok {
			j_price = common.TRC21GasPrice
		}
	}

	// If the prices are equal, use the time the transaction was first seen for
	// deterministic sorting
	cmp := i_price.Cmp(j_price)
	if cmp == 0 {
		return s.txs[i].tx.Time.Before(s.txs[j].tx.Time)
	}
	return cmp > 0
}

func (s txByPriceAndTime) Swap(i, j int) {
	s.txs[i], s.txs[j] = s.txs[j], s.txs[i]
}

func (s *txByPriceAndTime) Push(x interface{}) {
	s.txs = append(s.txs, x.(*txWithMinerFee))
}

func (s *txByPriceAndTime) Pop() interface{} {
	old := s.txs
	n := len(old)
	x := old[n-1]
	old[n-1] = nil // avoid memory leak
	s.txs = old[0 : n-1]
	return x
}

// transactionsByPriceAndNonce represents a set of transactions that can return
// transactions in a profit-maximizing sorted order, while supporting removing
// entire batches of transactions for non-executable accounts.
type transactionsByPriceAndNonce struct {
	txs     map[common.Address][]*txpool.LazyTransaction // Per account nonce-sorted list of transactions
	heads   txByPriceAndTime                             // Next transaction for each unique account (price heap)
	signer  types.Signer                                 // Signer for the set of transactions
	baseFee *big.Int                                     // Current base fee
}

// newTransactionsByPriceAndNonce creates a transaction set that can retrieve
// price sorted transactions in a nonce-honouring way.
//
// Note, the input map is reowned so the caller should not interact any more with
// if after providing it to the constructor.
func newTransactionsByPriceAndNonce(signer types.Signer, txs map[common.Address][]*txpool.LazyTransaction, payersSwap map[common.Address]*big.Int, baseFee *big.Int) (*transactionsByPriceAndNonce, types.Transactions) {
	// Initialize a price and received time based heap with the head transactions
	heads := txByPriceAndTime{
		txs:        make([]*txWithMinerFee, 0, len(txs)),
		payersSwap: payersSwap,
	}
	specialTxs := types.Transactions{}
	for from, accTxs := range txs {
		var normalTxs []*txpool.LazyTransaction
		for _, lazyTx := range accTxs {
			if tx := lazyTx.Resolve(); tx.IsSpecialTransaction() {
				specialTxs = append(specialTxs, tx)
			} else {
				normalTxs = append(normalTxs, lazyTx)
			}
		}
		if len(normalTxs) > 0 {
			wrapped, err := newTxWithMinerFee(normalTxs[0], from, baseFee)
			if err != nil {
				delete(txs, from)
				continue
			}
			heads.txs = append(heads.txs, wrapped)
			// Remove the first normal transaction for this sender
			txs[from] = normalTxs[1:]
		} else {
			// Remove the account if there are no normal transactions
			delete(txs, from)
		}
	}
	heap.Init(&heads)

	// Assemble and return the transaction set
	return &transactionsByPriceAndNonce{
		txs:     txs,
		heads:   heads,
		signer:  signer,
		baseFee: baseFee,
	}, specialTxs
}

// Peek returns the next transaction by price.
func (t *transactionsByPriceAndNonce) Peek() *txpool.LazyTransaction {
	if len(t.heads.txs) == 0 {
		return nil
	}
	return t.heads.txs[0].tx
}

// Shift replaces the current best head with the next one from the same account.
func (t *transactionsByPriceAndNonce) Shift() {
	acc := t.heads.txs[0].from
	if txs, ok := t.txs[acc]; ok && len(txs) > 0 {
		if wrapped, err := newTxWithMinerFee(txs[0], acc, t.baseFee); err == nil {
			t.heads.txs[0], t.txs[acc] = wrapped, txs[1:]
			heap.Fix(&t.heads, 0)
			return
		}
	}
	heap.Pop(&t.heads)
}

// Pop removes the best transaction, *not* replacing it with the next one from
// the same account. This should be used when a transaction cannot be executed
// and hence all subsequent ones should be discarded from the same account.
func (t *transactionsByPriceAndNonce) Pop() {
	heap.Pop(&t.heads)
}
