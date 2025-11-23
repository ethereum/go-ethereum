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

package miner

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
)

var (
	ErrBundleTimestampTooEarly = errors.New("bundle timestamp too early")
	ErrBundleTimestampTooLate  = errors.New("bundle timestamp too late")
	ErrBundleReverted          = errors.New("bundle transaction reverted")
)

// Bundle represents a group of transactions that should be executed atomically
// in a specific order within a block.
type Bundle struct {
	Txs           []*types.Transaction
	MinTimestamp  uint64
	MaxTimestamp  uint64
	RevertingTxs  []int // Indices of transactions allowed to revert
	TargetBlock   uint64
}

// ValidateTimestamp checks if the bundle can be included at the given timestamp.
func (b *Bundle) ValidateTimestamp(timestamp uint64) error {
	if b.MinTimestamp > 0 && timestamp < b.MinTimestamp {
		return ErrBundleTimestampTooEarly
	}
	if b.MaxTimestamp > 0 && timestamp > b.MaxTimestamp {
		return ErrBundleTimestampTooLate
	}
	return nil
}

// CanRevert returns true if the transaction at the given index is allowed to revert.
func (b *Bundle) CanRevert(txIndex int) bool {
	for _, idx := range b.RevertingTxs {
		if idx == txIndex {
			return true
		}
	}
	return false
}

// BundleSimulationResult contains the results of simulating a bundle.
type BundleSimulationResult struct {
	Success          bool
	GasUsed          uint64
	Profit           *big.Int
	StateChanges     map[common.Address]*AccountChange
	FailedTxIndex    int // -1 if all succeeded
	FailedTxError    error
	CoinbaseBalance  *big.Int
	TxResults        []*TxSimulationResult
}

// TxSimulationResult contains results for a single transaction in a bundle.
type TxSimulationResult struct {
	Success     bool
	GasUsed     uint64
	Error       error
	Logs        []*types.Log
	ReturnValue []byte
}

// AccountChange represents state changes for an account.
type AccountChange struct {
	BalanceBefore *big.Int
	BalanceAfter  *big.Int
	NonceBefore   uint64
	NonceAfter    uint64
	StorageChanges map[common.Hash]common.Hash
}

// OrderingStrategy defines an interface for custom transaction ordering.
// Implementations can provide arbitrary ordering logic for block building.
type OrderingStrategy interface {
	// OrderTransactions takes pending transactions and bundles, and returns
	// an ordered list of transactions to include in the block.
	OrderTransactions(
		pending map[common.Address][]*txpool.LazyTransaction,
		bundles []*Bundle,
		state *state.StateDB,
		header *types.Header,
	) ([]*types.Transaction, error)
}

// DefaultOrderingStrategy implements the default greedy ordering by gas price.
type DefaultOrderingStrategy struct{}

// OrderTransactions implements the default ordering strategy.
func (s *DefaultOrderingStrategy) OrderTransactions(
	pending map[common.Address][]*txpool.LazyTransaction,
	bundles []*Bundle,
	state *state.StateDB,
	header *types.Header,
) ([]*types.Transaction, error) {
	// Default behavior: just return transactions sorted by price
	// This maintains backward compatibility
	var txs []*types.Transaction
	
	// First, add bundle transactions
	for _, bundle := range bundles {
		if err := bundle.ValidateTimestamp(header.Time); err == nil {
			txs = append(txs, bundle.Txs...)
		}
	}
	
	// Then add pending transactions (would normally use price sorting)
	for _, accountTxs := range pending {
		for _, ltx := range accountTxs {
			if tx := ltx.Resolve(); tx != nil {
				txs = append(txs, tx)
			}
		}
	}
	
	return txs, nil
}

