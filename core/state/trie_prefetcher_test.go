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
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
)

func filledStateDB() *StateDB {
	state, _ := New(types.EmptyRootHash, NewDatabase(rawdb.NewMemoryDatabase()), nil)

	// Create an account and check if the retrieved balance is correct
	addr := common.HexToAddress("0xaffeaffeaffeaffeaffeaffeaffeaffeaffeaffe")
	skey := common.HexToHash("aaa")
	sval := common.HexToHash("bbb")

	state.SetBalance(addr, uint256.NewInt(42)) // Change the account trie
	state.SetCode(addr, []byte("hello"))       // Change an external metadata
	state.SetState(addr, skey, sval)           // Change the storage trie
	for i := 0; i < 100; i++ {
		sk := common.BigToHash(big.NewInt(int64(i)))
		state.SetState(addr, sk, sk) // Change the storage trie
	}
	return state
}

func TestUseAfterClose(t *testing.T) {
	db := filledStateDB()
	prefetcher := newTriePrefetcher(db.db, db.originalRoot, "")
	skey := common.HexToHash("aaa")
	prefetcher.prefetch(common.Hash{}, db.originalRoot, common.Address{}, [][]byte{skey.Bytes()})
	a := prefetcher.trie(common.Hash{}, db.originalRoot)
	prefetcher.close()
	b := prefetcher.trie(common.Hash{}, db.originalRoot)
	if a == nil {
		t.Fatal("Prefetching before close should not return nil")
	}
	if b == nil {
		t.Fatal("Trie after close should not return nil")
	}
}
