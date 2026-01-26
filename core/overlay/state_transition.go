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

package overlay

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie/bintrie"
)

// TransitionState is a structure that holds the progress markers of the
// translation process.
// TODO gballet:
// * see if I can get rid of the pointer now that this piece
// has been rewritten.
// * the conversion pointers should no longer be necessary,
// remove them when it's been confirmed.
// * we can't keep the preimage offset in the file
type TransitionState struct {
	CurrentAccountAddress *common.Address // addresss of the last translated account
	CurrentSlotHash       common.Hash     // hash of the last translated storage slot
	CurrentPreimageOffset int64           // next byte to read from the preimage file
	Started, Ended        bool

	// Mark whether the storage for an account has been processed. This is useful if the
	// maximum number of leaves of the conversion is reached before the whole storage is
	// processed.
	StorageProcessed bool

	BaseRoot common.Hash // hash of the last read-only MPT base tree
}

// InTransition returns true if the translation process is in progress.
func (ts *TransitionState) InTransition() bool {
	return ts != nil && ts.Started && !ts.Ended
}

// Transitioned returns true if the translation process has been completed.
func (ts *TransitionState) Transitioned() bool {
	return ts != nil && ts.Ended
}

// Copy returns a deep copy of the TransitionState object.
func (ts *TransitionState) Copy() *TransitionState {
	ret := &TransitionState{
		Started:               ts.Started,
		Ended:                 ts.Ended,
		CurrentSlotHash:       ts.CurrentSlotHash,
		CurrentPreimageOffset: ts.CurrentPreimageOffset,
		StorageProcessed:      ts.StorageProcessed,
		BaseRoot:              ts.BaseRoot,
	}
	if ts.CurrentAccountAddress != nil {
		addr := *ts.CurrentAccountAddress
		ret.CurrentAccountAddress = &addr
	}
	return ret
}

var (
	conversionProgressAddressKey       = common.Hash{1}
	conversionProgressSlotKey          = common.Hash{2}
	conversionProgressStorageProcessed = common.Hash{3}
)

// LoadTransitionState retrieves the Verkle transition state associated with
// the given state root hash from the database.
func LoadTransitionState(db ethdb.KeyValueReader, root common.Hash) *TransitionState {
	stem := bintrie.GetBinaryTreeKeyStorageSlot(params.BinaryTransitionRegistryAddress, conversionProgressAddressKey[:])
	leaf := rawdb.ReadStorageTrieNode(db, common.Hash{}, stem[:bintrie.StemSize])
	if len(leaf) == 0 {
		return &TransitionState{}
	}

	node, err := bintrie.DeserializeNode(leaf, 0)
	if err != nil {
		panic("could not deserialize conversion pointers contract storage")
	}
	leafNode := node.(*bintrie.StemNode)
	currentAccount := common.BytesToAddress(leafNode.Values[65][12:])
	currentSlotHash := common.BytesToHash(leafNode.Values[66])
	storageProcessed := len(leafNode.Values[67]) > 0 && leafNode.Values[67][0] != 0
	return &TransitionState{
		CurrentAccountAddress: &currentAccount,
		CurrentSlotHash:       currentSlotHash,
		StorageProcessed:      storageProcessed,
	}
}
