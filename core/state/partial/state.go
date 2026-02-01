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
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/triedb"
)

// PartialState manages state for partial stateful nodes.
// It applies BAL diffs to update state without re-executing transactions.
type PartialState struct {
	db      ethdb.Database
	trieDB  *triedb.Database
	filter  ContractFilter
	history *BALHistory

	// Current state root
	stateRoot common.Hash
}

// NewPartialState creates a new partial state manager.
func NewPartialState(db ethdb.Database, trieDB *triedb.Database, filter ContractFilter, balRetention uint64) *PartialState {
	return &PartialState{
		db:      db,
		trieDB:  trieDB,
		filter:  filter,
		history: NewBALHistory(db, balRetention),
	}
}

// Filter returns the contract filter used by this partial state.
func (s *PartialState) Filter() ContractFilter {
	return s.filter
}

// SetRoot sets the current state root.
func (s *PartialState) SetRoot(root common.Hash) {
	s.stateRoot = root
}

// Root returns the current state root.
func (s *PartialState) Root() common.Hash {
	return s.stateRoot
}

// ApplyBALAndComputeRoot applies BAL diffs and returns the new state root.
// This is the core function for partial state block processing.
//
// TODO: Implement in Phase 3/4 - this will:
// 1. Open trie at current root
// 2. Apply balance/nonce changes from BAL
// 3. Apply storage changes for tracked contracts
// 4. Commit trie changes using existing pathdb compression
// 5. Return new state root
func (s *PartialState) ApplyBALAndComputeRoot(currentRoot common.Hash, accessList *bal.BlockAccessList) (common.Hash, error) {
	// Placeholder - will be implemented in Phase 4
	panic("ApplyBALAndComputeRoot not yet implemented")
}

// History returns the BAL history manager.
func (s *PartialState) History() *BALHistory {
	return s.history
}
