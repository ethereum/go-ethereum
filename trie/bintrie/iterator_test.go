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

package bintrie

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-ethereum/triedb/hashdb"
	"github.com/ethereum/go-ethereum/triedb/pathdb"
	"github.com/holiman/uint256"
)

func newTestDatabase(diskdb ethdb.Database, scheme string) *triedb.Database {
	config := &triedb.Config{Preimages: true}
	if scheme == rawdb.HashScheme {
		config.HashDB = &hashdb.Config{CleanCacheSize: 0}
	} else {
		config.PathDB = &pathdb.Config{TrieCleanSize: 0, StateCleanSize: 0}
	}
	return triedb.NewDatabase(diskdb, config)
}

func TestBinaryIterator(t *testing.T) {
	trie, err := NewBinaryTrie(types.EmptyVerkleHash, newTestDatabase(rawdb.NewMemoryDatabase(), rawdb.PathScheme))
	if err != nil {
		t.Fatal(err)
	}
	account0 := &types.StateAccount{
		Nonce:    1,
		Balance:  uint256.NewInt(2),
		Root:     types.EmptyRootHash,
		CodeHash: nil,
	}
	// NOTE: the code size isn't written to the trie via TryUpdateAccount
	// so it will be missing from the test nodes.
	trie.UpdateAccount(common.Address{}, account0, 0)
	account1 := &types.StateAccount{
		Nonce:    1337,
		Balance:  uint256.NewInt(2000),
		Root:     types.EmptyRootHash,
		CodeHash: nil,
	}
	// This address is meant to hash to a value that has the same first byte as 0xbf
	var clash = common.HexToAddress("69fd8034cdb20934dedffa7dccb4fb3b8062a8be")
	trie.UpdateAccount(clash, account1, 0)

	// Manually go over every node to check that we get all
	// the correct nodes.
	it, err := trie.NodeIterator(nil)
	if err != nil {
		t.Fatal(err)
	}
	var leafcount int
	for it.Next(true) {
		t.Logf("Node: %x", it.Path())
		if it.Leaf() {
			leafcount++
			t.Logf("\tLeaf: %x", it.LeafKey())
		}
	}
	if leafcount != 2 {
		t.Fatalf("invalid leaf count: %d != 6", leafcount)
	}
}
