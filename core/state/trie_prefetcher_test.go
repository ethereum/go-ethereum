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
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/holiman/uint256"
)

func filledStateDB() *StateDB {
	state, _ := New(types.EmptyRootHash, NewDatabaseForTesting())

	addr := common.HexToAddress("0xaffeaffeaffeaffeaffeaffeaffeaffeaffeaffe")
	skey := common.HexToHash("aaa")
	sval := common.HexToHash("bbb")

	state.SetBalance(addr, uint256.NewInt(42), tracing.BalanceChangeUnspecified)
	state.SetCode(addr, []byte("hello"), tracing.CodeChangeUnspecified)
	state.SetState(addr, skey, sval)
	for i := 0; i < 100; i++ {
		sk := common.BigToHash(big.NewInt(int64(i)))
		state.SetState(addr, sk, sk)
	}
	return state
}

func TestSubfetcherUseAfterTerminate(t *testing.T) {
	db := filledStateDB()

	// Open a trie and create a subfetcher for it.
	id := trie.StateTrieID(db.originalRoot)
	tr, err := trie.NewStateTrie(id, db.db.TrieDB())
	if err != nil {
		t.Fatalf("Failed to open trie: %v", err)
	}
	sf := newPrefetcher(tr, false)
	addr := common.HexToAddress("0xaffeaffeaffeaffeaffeaffeaffeaffeaffeaffe")

	// Scheduling before termination should succeed.
	if err := sf.scheduleAccounts([]common.Address{addr}, false); err != nil {
		t.Fatalf("Schedule failed before terminate: %v", err)
	}
	// Terminate synchronously — waits for pending tasks.
	sf.terminate()

	// Scheduling after termination should fail.
	if err := sf.scheduleAccounts([]common.Address{addr}, false); err == nil {
		t.Fatal("Schedule succeeded after terminate")
	}
}

func TestWrapTriePrefetch(t *testing.T) {
	db := filledStateDB()

	// Create a wrapTrie with prefetching enabled.
	id := trie.StateTrieID(db.originalRoot)
	tr, err := newWrapTrie(id, db.db.TrieDB(), true, true)
	if err != nil {
		t.Fatalf("Failed to create wrapTrie: %v", err)
	}
	addr := common.HexToAddress("0xaffeaffeaffeaffeaffeaffeaffeaffeaffeaffe")

	// Schedule some prefetch work.
	tr.prefetchAccounts([]common.Address{addr}, false)

	// Terminate and verify the trie is usable.
	tr.term()
	if tr.Hash() == (common.Hash{}) {
		t.Fatal("wrapTrie hash is zero after prefetch")
	}
}
