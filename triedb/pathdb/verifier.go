// Copyright 2020 The go-ethereum Authors
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

package pathdb

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/triedb/internal"
)

// VerifyState traverses the flat states specified by the given state root and
// ensures they are matched with each other.
func (db *Database) VerifyState(root common.Hash) error {
	acctIt, err := db.AccountIterator(root, common.Hash{})
	if err != nil {
		return err // The required snapshot might not exist.
	}
	defer acctIt.Release()

	got, err := internal.GenerateTrieRoot(nil, "", acctIt, common.Hash{}, stackTrieHasher, func(_ ethdb.KeyValueWriter, accountHash, codeHash common.Hash, stat *internal.GenerateStats) (common.Hash, error) {
		// Migrate the code first, commit the contract code into the tmp db.
		if codeHash != types.EmptyCodeHash {
			code := rawdb.ReadCode(db.diskdb, codeHash)
			if len(code) == 0 {
				return common.Hash{}, errors.New("failed to read contract code")
			}
		}
		// Then migrate all storage trie nodes into the tmp db.
		storageIt, err := db.StorageIterator(root, accountHash, common.Hash{})
		if err != nil {
			return common.Hash{}, err
		}
		defer storageIt.Release()

		hash, err := internal.GenerateTrieRoot(nil, "", storageIt, accountHash, stackTrieHasher, nil, stat, false)
		if err != nil {
			return common.Hash{}, err
		}
		return hash, nil
	}, internal.NewGenerateStats(), true)

	if err != nil {
		return err
	}
	if got != root {
		return fmt.Errorf("state root hash mismatch: got %x, want %x", got, root)
	}
	return nil
}

func stackTrieHasher(_ ethdb.KeyValueWriter, _ string, _ common.Hash, in chan internal.TrieKV, out chan common.Hash) {
	t := trie.NewStackTrie(nil)
	for leaf := range in {
		t.Update(leaf.Key[:], leaf.Value)
	}
	out <- t.Hash()
}
