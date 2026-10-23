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

package legacypool

import (
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
)

// lazyList is an account's nonce-sorted transactions as Pending serves them,
// with the fee caps and gas limits its filter compares decoded once and packed
// together, so that reading the pool doesn't touch every transaction again.
type lazyList struct {
	txs  []*txpool.LazyTransaction
	fees []lazyFees // of txs[i], which point into it
}

type lazyFees struct {
	feeCap, tipCap uint256.Int
	gas            uint64
}

func newLazyList(pool txpool.LazyResolver, txs types.Transactions) *lazyList {
	l := &lazyList{
		txs:  make([]*txpool.LazyTransaction, len(txs)),
		fees: make([]lazyFees, len(txs)),
	}
	backing := make([]txpool.LazyTransaction, len(txs))
	for i, tx := range txs {
		var feeOverflow, tipOverflow bool
		f := &l.fees[i]
		f.feeCap, feeOverflow = tx.GasFeeCapUint256()
		f.tipCap, tipOverflow = tx.GasTipCapUint256()
		if feeOverflow || tipOverflow {
			panic("fee cap overflows uint256") // as uint256.MustFromBig did
		}
		f.gas = tx.Gas()
		backing[i] = txpool.LazyTransaction{
			Pool:      pool,
			Hash:      tx.Hash(),
			Tx:        tx,
			Time:      tx.Time(),
			GasFeeCap: &f.feeCap,
			GasTipCap: &f.tipCap,
			Gas:       f.gas,
			BlobGas:   tx.BlobGas(),
		}
		l.txs[i] = &backing[i]
	}
	return l
}

// slice returns the transactions at positions [from, to), sharing l's arrays.
func (l *lazyList) slice(from, to int) *lazyList {
	return &lazyList{txs: l.txs[from:to], fees: l.fees[from:to]}
}

// pending returns how many of the leading transactions pass filter's minimum
// tip and gas limit.
func (l *lazyList) pending(filter txpool.PendingFilter) int {
	if filter.MinTip == nil && filter.GasLimitCap == 0 {
		return len(l.fees)
	}
	for i := range l.fees {
		f := &l.fees[i]
		if filter.MinTip != nil && f.tipCmp(filter.MinTip, filter.BaseFee) < 0 {
			return i
		}
		if filter.GasLimitCap != 0 && f.gas > filter.GasLimitCap {
			return i
		}
	}
	return len(l.fees)
}

// tipCmp compares the tip the transaction pays at baseFee, min(tipCap,
// feeCap-baseFee), with tip, as EffectiveGasTipIntCmp does: a fee cap below the
// base fee pays a negative tip, below any tip.
func (f *lazyFees) tipCmp(tip, baseFee *uint256.Int) int {
	if baseFee == nil {
		return f.tipCap.Cmp(tip)
	}
	if f.feeCap.Lt(baseFee) {
		return -1
	}
	var paid uint256.Int
	paid.Sub(&f.feeCap, baseFee)
	if f.tipCap.Lt(&paid) {
		paid = f.tipCap
	}
	return paid.Cmp(tip)
}
