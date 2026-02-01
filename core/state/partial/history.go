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

package partial

import (
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/ethereum/go-ethereum/ethdb"
)

// BALHistory manages storage and retrieval of Block Access Lists for reorg handling.
// It's a thin wrapper over rawdb accessor functions, following go-ethereum patterns.
type BALHistory struct {
	db        ethdb.Database
	retention uint64 // Number of blocks to retain BAL history
}

// NewBALHistory creates a new BAL history manager.
func NewBALHistory(db ethdb.Database, retention uint64) *BALHistory {
	return &BALHistory{
		db:        db,
		retention: retention,
	}
}

// Store saves a BAL for a specific block number.
func (h *BALHistory) Store(blockNum uint64, accessList *bal.BlockAccessList) {
	rawdb.WriteBALHistory(h.db, blockNum, accessList)
}

// Get retrieves the BAL for a specific block number.
// Returns nil, false if not found.
func (h *BALHistory) Get(blockNum uint64) (*bal.BlockAccessList, bool) {
	accessList := rawdb.ReadBALHistory(h.db, blockNum)
	return accessList, accessList != nil
}

// Delete removes the BAL for a specific block number.
func (h *BALHistory) Delete(blockNum uint64) {
	rawdb.DeleteBALHistory(h.db, blockNum)
}

// Prune removes all BALs before the specified block number.
// Uses SafeDeleteRange for interruptible pruning.
func (h *BALHistory) Prune(beforeBlock uint64) error {
	return rawdb.PruneBALHistory(h.db, beforeBlock)
}

// Retention returns the configured retention window in blocks.
func (h *BALHistory) Retention() uint64 {
	return h.retention
}
