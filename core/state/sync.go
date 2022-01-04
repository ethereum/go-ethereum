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

package state

import (
	"bytes"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

// NewStateSync create a new state trie download scheduler.
func NewStateSync(root common.Hash, database ethdb.KeyValueReader, bloom *trie.SyncBloom, onLeaf func(paths [][]byte, leaf []byte) error) *trie.Sync {
	// Register the storage slot callback if the external callback is specified.
	var onSlot func(paths [][]byte, hexpath []byte, leaf []byte, parent common.Hash) error
	if onLeaf != nil {
		onSlot = func(paths [][]byte, hexpath []byte, leaf []byte, parent common.Hash) error {
			return onLeaf(paths, leaf)
		}
	}
	// Register the account callback to connect the state trie and the storage
	// trie belongs to the contract.
	var syncer *trie.Sync
	onAccount := func(paths [][]byte, hexpath []byte, leaf []byte, parent common.Hash) error {
		if onLeaf != nil {
			if err := onLeaf(paths, leaf); err != nil {
				return err
			}
		}
		var obj types.StateAccount
		if err := rlp.Decode(bytes.NewReader(leaf), &obj); err != nil {
			return err
		}
		syncer.AddSubTrie(obj.Root, hexpath, parent, onSlot)
		syncer.AddCodeEntry(common.BytesToHash(obj.CodeHash), hexpath, parent)
		return nil
	}
	syncer = trie.NewSync(root, database, onAccount, bloom)
	return syncer
}
