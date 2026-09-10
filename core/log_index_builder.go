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

package core

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// TableWrite describes a table root that should be written to the index contract.
type TableWrite struct {
	FirstBlock uint64
	TableSize  uint64
	Root       common.Hash
}

// BuildLogIndexForBlock builds the EIP-8304 level-0 log index table for the given
// block. Interim implementation: fixed-size entries and a flat keccak root
func BuildLogIndexForBlock(blockNumber uint64, receipts types.Receipts) []TableWrite {
	b0 := types.NewIndexBuilder()
	b0.AddBlockEntries(blockNumber, receipts)
	return []TableWrite{{FirstBlock: blockNumber, TableSize: 1, Root: b0.Build()}}
}
