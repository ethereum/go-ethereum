// Copyright 2024 The go-ethereum Authors
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

package txpool

import (
	"container/heap"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type Pending interface {
	// Shift replaces the current best head with the next one from the same account.
	Shift()
	// Peek returns the next transaction by price.
	Peek() (*LazyTransaction, *uint256.Int)

	// Pop removes the best transaction, *not* replacing it with the next one from
	// the same account. This should be used when a transaction cannot be executed
	// and hence all subsequent ones should be discarded from the same account.
	Pop()

	// Empty returns true if the set is empty.
	Empty() bool

	// Clears the set
	Clear()
}

// PendingFilter is a collection of filter rules to allow retrieving a subset
// of transactions for announcement or mining.
//
// Note, the entries here are not arbitrary useful filters, rather each one has
// a very specific call site in mind and each one can be evaluated very cheaply
// by the pool implementations. Only add new ones that satisfy those constraints.
type PendingFilter struct {
	MinTip  *uint256.Int // Minimum miner tip required to include a transaction
	BaseFee *uint256.Int // Minimum 1559 basefee needed to include a transaction
	BlobFee *uint256.Int // Minimum 4844 blobfee needed to include a blob transaction

	OnlyPlainTxs bool // Return only plain EVM transactions (peer-join announces, block space filling)
	OnlyBlobTxs  bool // Return only blob transactions (block blob-space filling)

	OnlyLocals bool // Return only txs from 'local' addresses.
	NoLocals   bool // Remove all txs from 'local' addresses
}

type TxTips struct {
	From common.Address // sender
	Tips uint256.Int    // miner-fees earned by this transaction.
	Time int64          // Time when the transaction was first seen
}

type TipList []*TxTips

func (f TipList) Len() int {
	return len(f)
}

func (f TipList) Less(i, j int) bool {
	cmp := f[i].Tips.Cmp(&f[j].Tips)
	if cmp == 0 {
		return f[i].Time < f[j].Time
	}
	return cmp > 0
}

func (f TipList) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

func (f *TipList) Push(x any) {
	*f = append(*f, x.(*TxTips))
}

func (f *TipList) Pop() any {
	old := *f
	n := len(old)
	x := old[n-1]
	old[n-1] = nil
	*f = old[0 : n-1]
	return x
}

type pendingSet struct {
	Tails map[common.Address][]*LazyTransaction // Per account nonce-sorted list of transactions
	Heads TipList                               // Next transaction for each unique account (price heap)
}

var EmptyPending = NewPendingSet(nil, nil)

func NewPendingSet(heads TipList, tails map[common.Address][]*LazyTransaction) *pendingSet {
	if len(heads) != 0 {
		heap.Init(&heads)
	}
	return &pendingSet{
		Tails: tails,
		Heads: heads,
	}
}

// Shift replaces the current best head with the next one from the same account.
func (ps *pendingSet) Shift() {
	addr := ps.Heads[0].From
	if txs, ok := ps.Tails[addr]; ok && len(txs) > 1 {
		ps.Heads[0].Tips = txs[1].Fees
		ps.Heads[0].Time = txs[1].Time.UnixNano()
		ps.Tails[addr] = txs[1:]
		heap.Fix(&ps.Heads, 0)
		return
	}
	heap.Pop(&ps.Heads)
}

// Peek returns the next transaction by price.
func (ps *pendingSet) Peek() (*LazyTransaction, *uint256.Int) {
	if len(ps.Heads) == 0 {
		return nil, nil
	}
	sender := ps.Heads[0].From
	fees := ps.Heads[0].Tips
	tx := ps.Tails[sender][0]
	return tx, &fees
}

func (ps *pendingSet) Clear() {
	ps.Heads = nil
	ps.Tails = nil
}

func (ps *pendingSet) Empty() bool {
	return len(ps.Heads) == 0
}

// Pop removes the best transaction, *not* replacing it with the next one from
// the same account. This should be used when a transaction cannot be executed
// and hence all subsequent ones should be discarded from the same account.
func (ps *pendingSet) Pop() {
	heap.Pop(&ps.Heads)
}
