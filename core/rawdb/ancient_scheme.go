// Copyright 2022 The go-ethereum Authors
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

package rawdb

import (
	"path/filepath"

	"github.com/ethereum/go-ethereum/ethdb"
)

// The list of table names of chain freezer.
const (
	// ChainFreezerHeaderTable indicates the name of the freezer header table.
	ChainFreezerHeaderTable = "headers"

	// ChainFreezerHashTable indicates the name of the freezer canonical hash table.
	ChainFreezerHashTable = "hashes"

	// ChainFreezerBodiesTable indicates the name of the freezer block body table.
	ChainFreezerBodiesTable = "bodies"

	// ChainFreezerReceiptTable indicates the name of the freezer receipts table.
	ChainFreezerReceiptTable = "receipts"

	// ChainFreezerDifficultyTable indicates the name of the freezer total difficulty table.
	ChainFreezerDifficultyTable = "diffs"
)

// chainFreezerNoSnappy configures whether compression is disabled for the ancient-tables.
// Hashes and difficulties don't compress well.
var chainFreezerNoSnappy = map[string]bool{
	ChainFreezerHeaderTable:     false,
	ChainFreezerHashTable:       true,
	ChainFreezerBodiesTable:     false,
	ChainFreezerReceiptTable:    false,
	ChainFreezerDifficultyTable: true,
}

// chainFreezerSize configures the maximum size for each freezer table data files.
var chainFreezerSize = map[string]uint32{
	// The size of each item's value is roughly 650 bytes, about 2 millions
	// items per data file.
	ChainFreezerHeaderTable: 2 * 1000 * 1000 * 1000,

	// The size of each item’s value is fixed at 32 bytes, 2 millions items
	// per data file.
	ChainFreezerHashTable: 64 * 1000 * 1000,

	// The size of each item’s value is less than 10 bytes, 2 millions items
	// per data file.
	ChainFreezerDifficultyTable: 20 * 1000 * 1000,

	ChainFreezerBodiesTable:  2 * 1000 * 1000 * 1000,
	ChainFreezerReceiptTable: 2 * 1000 * 1000 * 1000,
}

const (
	// stateHistoryAccountIndex indicates the name of the freezer state history table.
	stateHistoryMeta         = "history.meta"
	stateHistoryAccountIndex = "account.index"
	stateHistoryStorageIndex = "storage.index"
	stateHistoryAccountData  = "account.data"
	stateHistoryStorageData  = "storage.data"
)

// stateFreezerNoSnappy configures whether compression is disabled for the state freezer.
var stateFreezerNoSnappy = map[string]bool{
	stateHistoryMeta:         true,
	stateHistoryAccountIndex: false,
	stateHistoryStorageIndex: false,
	stateHistoryAccountData:  false,
	stateHistoryStorageData:  false,
}

// stateFreezerSize configures the maximum size for each freezer table data files.
var stateFreezerSize = map[string]uint32{
	// The size of each item's value is fixed at 73 bytes, about 2 millions
	// items per data file.
	stateHistoryMeta:         128 * 1000 * 1000,
	stateHistoryAccountIndex: 2 * 1000 * 1000 * 1000,
	stateHistoryStorageIndex: 2 * 1000 * 1000 * 1000,
	stateHistoryAccountData:  2 * 1000 * 1000 * 1000,
	stateHistoryStorageData:  2 * 1000 * 1000 * 1000,
}

// The list of identifiers of ancient stores.
var (
	ChainFreezerName       = "chain"        // the folder name of chain segment ancient store.
	MerkleStateFreezerName = "state"        // the folder name of state history ancient store.
	VerkleStateFreezerName = "state_verkle" // the folder name of state history ancient store.
)

// freezers the collections of all builtin freezers.
var freezers = []string{ChainFreezerName, MerkleStateFreezerName, VerkleStateFreezerName}

// NewStateFreezer initializes the ancient store for state history.
//
//   - if the empty directory is given, initializes the pure in-memory
//     state freezer (e.g. dev mode).
//   - if non-empty directory is given, initializes the regular file-based
//     state freezer.
func NewStateFreezer(ancientDir string, verkle bool, readOnly bool) (ethdb.ResettableAncientStore, error) {
	if ancientDir == "" {
		return NewMemoryFreezer(readOnly, stateFreezerNoSnappy), nil
	}
	var name string
	if verkle {
		name = filepath.Join(ancientDir, VerkleStateFreezerName)
	} else {
		name = filepath.Join(ancientDir, MerkleStateFreezerName)
	}
	return newResettableFreezer(name, "eth/db/state", readOnly, stateFreezerSize, stateFreezerNoSnappy)
}
