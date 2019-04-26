// Copyright 2015 The go-ethereum Authors
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

// Contains a batch of utility type declarations used by the tests. As the node
// operates on unique types, a lot of them are needed to check various features.

package statediff

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
)

// AccountsMap is a mapping of keccak256(address) => accountWrapper
type AccountsMap map[common.Hash]accountWrapper

// AccountWrapper is used to temporary associate the unpacked account with its raw values
type accountWrapper struct {
	Account  state.Account
	RawKey   []byte
	RawValue []byte
	Proof    [][]byte
	Path     []byte
}

// StateDiff is the final output structure from the builder
type StateDiff struct {
	BlockNumber     int64           `json:"blockNumber"	    gencodec:"required"`
	BlockHash       common.Hash     `json:"blockHash" 	    gencodec:"required"`
	CreatedAccounts AccountDiffsMap `json:"createdAccounts" gencodec:"required"`
	DeletedAccounts AccountDiffsMap `json:"deletedAccounts" gencodec:"required"`
	UpdatedAccounts AccountDiffsMap `json:"updatedAccounts" gencodec:"required"`

	encoded []byte
	err     error
}

func (sd *StateDiff) ensureEncoded() {
	if sd.encoded == nil && sd.err == nil {
		sd.encoded, sd.err = json.Marshal(sd)
	}
}

// Length to implement Encoder interface for StateDiff
func (sd *StateDiff) Length() int {
	sd.ensureEncoded()
	return len(sd.encoded)
}

// Encode to implement Encoder interface for StateDiff
func (sd *StateDiff) Encode() ([]byte, error) {
	sd.ensureEncoded()
	return sd.encoded, sd.err
}

// AccountDiffsMap is a mapping of keccak256(address) => AccountDiff
type AccountDiffsMap map[common.Hash]AccountDiff

// AccountDiff holds the data for a single state diff leaf node
type AccountDiff struct {
	Key     []byte        `json:"key"         gencodec:"required"`
	Value   []byte        `json:"value"       gencodec:"required"`
	Proof   [][]byte      `json:"proof"       gencodec:"required"`
	Storage []StorageDiff `json:"storage"     gencodec:"required"`
	Path    []byte        `json:"path"        gencodec:"required"`
}

// StorageDiff holds the data for a single storage diff leaf node
type StorageDiff struct {
	Key   []byte   `json:"key"         gencodec:"required"`
	Value []byte   `json:"value"       gencodec:"required"`
	Proof [][]byte `json:"proof"       gencodec:"required"`
	Path  []byte   `json:"path"        gencodec:"required"`
}

/*
// State trie leaf is just a short node, below
// that has an rlp encoded account as the value


// SO each account diffs map is reall a map of shortnode keys to values
// Flatten to a slice of short nodes?

// Need to coerce into:

type TrieNode struct {
	// leaf, extension or branch
	nodeKind string

	// If leaf or extension: [0] is key, [1] is val.
	// If branch: [0] - [16] are children.
	elements []interface{}

	// IPLD block information
	cid     *cid.Cid
	rawdata []byte
}
*/
