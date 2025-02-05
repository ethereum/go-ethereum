// Copyright 2021 The go-ethereum Authors
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
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/internal/testrand"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/holiman/uint256"
)

func filledStateDB() *StateDB {
	state, _ := New(types.EmptyRootHash, NewDatabaseForTesting())

	// Create an account and check if the retrieved balance is correct
	addr := common.HexToAddress("0xaffeaffeaffeaffeaffeaffeaffeaffeaffeaffe")
	skey := common.HexToHash("aaa")
	sval := common.HexToHash("bbb")

	state.SetBalance(addr, uint256.NewInt(42), tracing.BalanceChangeUnspecified) // Change the account trie
	state.SetCode(addr, []byte("hello"))                                         // Change an external metadata
	state.SetState(addr, skey, sval)                                             // Change the storage trie
	for i := 0; i < 100; i++ {
		sk := common.BigToHash(big.NewInt(int64(i)))
		state.SetState(addr, sk, sk) // Change the storage trie
	}
	return state
}

func TestUseAfterTerminate(t *testing.T) {
	db := filledStateDB()
	prefetcher := newTriePrefetcher(db.db, db.originalRoot, "", true)
	skey := common.HexToHash("aaa")

	if err := prefetcher.prefetch(common.Hash{}, db.originalRoot, common.Address{}, nil, []common.Hash{skey}, false); err != nil {
		t.Errorf("Prefetch failed before terminate: %v", err)
	}
	prefetcher.terminate(false)

	if err := prefetcher.prefetch(common.Hash{}, db.originalRoot, common.Address{}, nil, []common.Hash{skey}, false); err == nil {
		t.Errorf("Prefetch succeeded after terminate: %v", err)
	}
	if tr := prefetcher.trie(common.Hash{}, db.originalRoot); tr == nil {
		t.Errorf("Prefetcher returned nil trie after terminate")
	}
}

func TestVerklePrefetcher(t *testing.T) {
	disk := rawdb.NewMemoryDatabase()
	db := triedb.NewDatabase(disk, triedb.VerkleDefaults)
	sdb := NewDatabase(db, nil)

	state, err := New(types.EmptyRootHash, sdb)
	if err != nil {
		t.Fatalf("failed to initialize state: %v", err)
	}
	// Create an account and check if the retrieved balance is correct
	addr := testrand.Address()
	skey := testrand.Hash()
	sval := testrand.Hash()

	state.SetBalance(addr, uint256.NewInt(42), tracing.BalanceChangeUnspecified) // Change the account trie
	state.SetCode(addr, []byte("hello"))                                         // Change an external metadata
	state.SetState(addr, skey, sval)                                             // Change the storage trie
	root, _ := state.Commit(0, true, false)

	state, _ = New(root, sdb)
	sRoot := state.GetStorageRoot(addr)
	fetcher := newTriePrefetcher(sdb, root, "", false)

	// Read account
	fetcher.prefetch(common.Hash{}, root, common.Address{}, []common.Address{addr}, nil, false)

	// Read storage slot
	fetcher.prefetch(crypto.Keccak256Hash(addr.Bytes()), sRoot, addr, nil, []common.Hash{skey}, false)

	fetcher.terminate(false)
	accountTrie := fetcher.trie(common.Hash{}, root)
	storageTrie := fetcher.trie(crypto.Keccak256Hash(addr.Bytes()), sRoot)

	rootA := accountTrie.Hash()
	rootB := storageTrie.Hash()
	if rootA != rootB {
		t.Fatal("Two different tries are retrieved")
	}
}
