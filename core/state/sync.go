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
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

// StateSync is the main state synchronisation scheduler, which provides yet the
// unknown state hashes to retrieve, accepts node data associated with said hashes
// and reconstructs the state database step by step until all is done.
type StateSync trie.TrieSync

// NewStateSync create a new state trie download scheduler.
func NewStateSync(root common.Hash, database trie.DatabaseReader) *StateSync {
	var syncer *trie.TrieSync

	callback := func(leaf []byte, parent common.Hash) error {
		var obj struct {
			Nonce    uint64
			Balance  *big.Int
			Root     common.Hash
			CodeHash []byte
		}
		if err := rlp.Decode(bytes.NewReader(leaf), &obj); err != nil {
			return err
		}
		syncer.AddSubTrie(obj.Root, 64, parent, nil)
		syncer.AddRawEntry(common.BytesToHash(obj.CodeHash), 64, parent)

		return nil
	}
	syncer = trie.NewTrieSync(root, database, callback)
	return (*StateSync)(syncer)
}

// Missing retrieves the known missing nodes from the state trie for retrieval.
func (s *StateSync) Missing(max int) []common.Hash {
	return (*trie.TrieSync)(s).Missing(max)
}

// Process injects a batch of retrieved trie nodes data, returning if something
// was committed to the database and also the index of an entry if processing of
// it failed.
func (s *StateSync) Process(list []trie.SyncResult, dbw trie.DatabaseWriter) (bool, int, error) {
	return (*trie.TrieSync)(s).Process(list, dbw)
}

// Pending returns the number of state entries currently pending for download.
func (s *StateSync) Pending() int {
	return (*trie.TrieSync)(s).Pending()
}
