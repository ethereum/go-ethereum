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

	// ChainFreezerBALTable indicates the name of the freezer block access list
	// table introduced by EIP-7928.
	ChainFreezerBALTable = "bals"
)

// Identifiers of tail groups used by the chain freezer.
const (
	// ChainFreezerBlockDataGroup is the tail group shared by the body and
	// receipt tables. The two tables are pruned together and therefore have
	// the same tail position.
	ChainFreezerBlockDataGroup = "blockdata"

	// ChainFreezerBALGroup is the tail group for the block access list table.
	// BAL is only populated after EIP-7928 activates, so it generally has a
	// higher tail than the block-data group and is pruned independently.
	ChainFreezerBALGroup = "bal"
)

// chainFreezerTableConfigs configures the settings for tables in the chain freezer.
// Compression is disabled for hashes as they don't compress well. Additionally,
// tail truncation is disabled for the header and hash tables, as these are intended
// to be retained long-term.
var chainFreezerTableConfigs = map[string]freezerTableConfig{
	ChainFreezerHeaderTable:  {noSnappy: false},
	ChainFreezerHashTable:    {noSnappy: true},
	ChainFreezerBodiesTable:  {noSnappy: false, tailGroup: ChainFreezerBlockDataGroup},
	ChainFreezerReceiptTable: {noSnappy: false, tailGroup: ChainFreezerBlockDataGroup},
	ChainFreezerBALTable:     {noSnappy: false, tailGroup: ChainFreezerBALGroup},
}

// freezerTableConfig contains the settings for a freezer table.
type freezerTableConfig struct {
	// noSnappy disables item compression when true.
	noSnappy bool

	// tailGroup names a logical group of tables that share the same tail
	// position. Tables in the same group are pruned together and must agree
	// on their tail. An empty value means the table is not prunable; its
	// tail is always 0.
	tailGroup string
}

const (
	// stateHistoryTableSize defines the maximum size of freezer data files.
	stateHistoryTableSize = 2 * 1000 * 1000 * 1000

	// stateHistoryAccountIndex indicates the name of the freezer state history table.
	stateHistoryMeta         = "history.meta"
	stateHistoryAccountIndex = "account.index"
	stateHistoryStorageIndex = "storage.index"
	stateHistoryAccountData  = "account.data"
	stateHistoryStorageData  = "storage.data"
)

// DefaultHistoryGroup is the tail group shared by all state/trienode history
// tables with tail pruning enabled.
const DefaultHistoryGroup = "history"

// stateFreezerTableConfigs configures the settings for tables in the state freezer.
var stateFreezerTableConfigs = map[string]freezerTableConfig{
	stateHistoryMeta:         {noSnappy: true, tailGroup: DefaultHistoryGroup},
	stateHistoryAccountIndex: {noSnappy: false, tailGroup: DefaultHistoryGroup},
	stateHistoryStorageIndex: {noSnappy: false, tailGroup: DefaultHistoryGroup},
	stateHistoryAccountData:  {noSnappy: false, tailGroup: DefaultHistoryGroup},
	stateHistoryStorageData:  {noSnappy: false, tailGroup: DefaultHistoryGroup},
}

const (
	trienodeHistoryHeaderTable       = "trienode.header"
	trienodeHistoryKeySectionTable   = "trienode.key"
	trienodeHistoryValueSectionTable = "trienode.value"
)

// trienodeFreezerTableConfigs configures the settings for tables in the trienode freezer.
var trienodeFreezerTableConfigs = map[string]freezerTableConfig{
	trienodeHistoryHeaderTable: {noSnappy: false, tailGroup: DefaultHistoryGroup},

	// Disable snappy compression to allow efficient partial read.
	trienodeHistoryKeySectionTable: {noSnappy: true, tailGroup: DefaultHistoryGroup},

	// Disable snappy compression to allow efficient partial read.
	trienodeHistoryValueSectionTable: {noSnappy: true, tailGroup: DefaultHistoryGroup},
}

// The list of identifiers of ancient stores.
var (
	ChainFreezerName          = "chain"           // the folder name of chain segment ancient store.
	MerkleStateFreezerName    = "state"           // the folder name of state history ancient store.
	VerkleStateFreezerName    = "state_verkle"    // the folder name of state history ancient store.
	MerkleTrienodeFreezerName = "trienode"        // the folder name of trienode history ancient store.
	VerkleTrienodeFreezerName = "trienode_verkle" // the folder name of trienode history ancient store.
)

// freezers the collections of all builtin freezers.
var freezers = []string{
	ChainFreezerName,
	MerkleStateFreezerName, VerkleStateFreezerName,
	MerkleTrienodeFreezerName, VerkleTrienodeFreezerName,
}

// NewStateFreezer initializes the ancient store for state history.
//
//   - if the empty directory is given, initializes the pure in-memory
//     state freezer (e.g. dev mode).
//   - if non-empty directory is given, initializes the regular file-based
//     state freezer.
func NewStateFreezer(ancientDir string, verkle bool, readOnly bool) (ethdb.ResettableAncientStore, error) {
	if ancientDir == "" {
		return NewMemoryFreezer(readOnly, stateFreezerTableConfigs), nil
	}
	var name string
	if verkle {
		name = filepath.Join(ancientDir, VerkleStateFreezerName)
	} else {
		name = filepath.Join(ancientDir, MerkleStateFreezerName)
	}
	return newResettableFreezer(name, "eth/db/state", readOnly, stateHistoryTableSize, stateFreezerTableConfigs)
}

// NewTrienodeFreezer initializes the ancient store for trienode history.
//
//   - if the empty directory is given, initializes the pure in-memory
//     trienode freezer (e.g. dev mode).
//   - if non-empty directory is given, initializes the regular file-based
//     trienode freezer.
func NewTrienodeFreezer(ancientDir string, verkle bool, readOnly bool) (ethdb.ResettableAncientStore, error) {
	if ancientDir == "" {
		return NewMemoryFreezer(readOnly, trienodeFreezerTableConfigs), nil
	}
	var name string
	if verkle {
		name = filepath.Join(ancientDir, VerkleTrienodeFreezerName)
	} else {
		name = filepath.Join(ancientDir, MerkleTrienodeFreezerName)
	}
	return newResettableFreezer(name, "eth/db/trienode", readOnly, stateHistoryTableSize, trienodeFreezerTableConfigs)
}
