// Copyright 2026 The go-ethereum Authors
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
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
)

// Methods this package implements to satisfy state.Trie, which no test reached.
//
// UpdateAccountBatch and UpdateStorageBatch come from the interface upstream
// introduced in "trie: introduce UpdateBatch". Nothing in this tree calls them
// yet - the batch path is not wired up - so they are implementations of
// somebody else's contract that no caller exercises. That is exactly why they
// are worth pinning: the day the batch path is wired in they run on every
// block, and a divergence from the one-at-a-time methods they stand in for
// would be a consensus bug on the first block that used them.
//
// Copy is the opposite case - it is reached in production, through
// mustCopyTrie from StateDB.Copy, and still had no test.

func testAccount(nonce uint64, balance uint64) *types.StateAccount {
	return &types.StateAccount{
		Nonce:    nonce,
		Balance:  uint256.NewInt(balance),
		Root:     types.EmptyRootHash,
		CodeHash: types.EmptyCodeHash[:],
	}
}

// TestUpdateAccountBatchMatchesSequential pins that batching accounts is the
// same transition as applying them one by one.
func TestUpdateAccountBatchMatchesSequential(t *testing.T) {
	var (
		addrs    []common.Address
		accounts []*types.StateAccount
		codeLens []int
	)
	for i := range 8 {
		addrs = append(addrs, common.Address{byte(i + 1), 0xaa})
		accounts = append(accounts, testAccount(uint64(i+1), uint64((i+1)*1000)))
		codeLens = append(codeLens, i*31) // distinct sizes in the packed basic data
	}

	batched := newTestTrie()
	if err := batched.UpdateAccountBatch(addrs, accounts, codeLens); err != nil {
		t.Fatalf("batch update failed: %v", err)
	}
	sequential := newTestTrie()
	for i, addr := range addrs {
		if err := sequential.UpdateAccount(addr, accounts[i], codeLens[i]); err != nil {
			t.Fatalf("sequential update failed: %v", err)
		}
	}
	if got, want := batched.Hash(), sequential.Hash(); got != want {
		t.Fatalf("batched root %x differs from sequential %x", got, want)
	}

	// A mismatched batch must be refused rather than silently truncated to the
	// shorter of the two, which would drop accounts on the floor.
	if err := newTestTrie().UpdateAccountBatch(addrs, accounts[:2], codeLens); err == nil {
		t.Fatal("a batch with fewer accounts than addresses was accepted")
	}
	if err := newTestTrie().UpdateAccountBatch(addrs, accounts, codeLens[:2]); err == nil {
		t.Fatal("a batch with fewer code lengths than addresses was accepted")
	}
}

// TestUpdateStorageBatchMatchesSequential pins the same for storage, across
// both homes an account's slots live in.
func TestUpdateStorageBatchMatchesSequential(t *testing.T) {
	addr := common.Address{0x5a, 0x10}

	var keys, values [][]byte
	for i := range 6 {
		// Alternate between a slot inside the header stem and one in the
		// overflow bucket, so the batch spans both.
		var slot common.Hash
		if i%2 == 0 {
			slot[31] = byte(i + 1)
		} else {
			slot[30] = byte(i + 4)
		}
		var value common.Hash
		value[31] = byte(i + 1)
		keys = append(keys, slot.Bytes())
		values = append(values, value.Bytes())
	}

	batched := newTestTrie()
	if err := batched.UpdateStorageBatch(addr, keys, values); err != nil {
		t.Fatalf("batch update failed: %v", err)
	}
	sequential := newTestTrie()
	for i, key := range keys {
		if err := sequential.UpdateStorage(addr, key, values[i]); err != nil {
			t.Fatalf("sequential update failed: %v", err)
		}
	}
	if got, want := batched.Hash(), sequential.Hash(); got != want {
		t.Fatalf("batched root %x differs from sequential %x", got, want)
	}

	if err := newTestTrie().UpdateStorageBatch(addr, keys, values[:2]); err == nil {
		t.Fatal("a batch with fewer values than keys was accepted")
	}
}

// TestCopyIsIndependent pins that a copied trie shares no mutable state with
// its original.
//
// StateDB.Copy reaches this through mustCopyTrie, so a copy that aliased the
// tree or the tracers would let work done on one statedb show up in another -
// and the tracers in particular decide the node set emitted at commit, which is
// what gets persisted.
func TestCopyIsIndependent(t *testing.T) {
	original := newTestTrie()
	base := testAccount(1, 100)
	if err := original.UpdateAccount(common.Address{0x01}, base, 0); err != nil {
		t.Fatal(err)
	}
	rootBefore := original.Hash()

	cp := original.Copy()
	if got := cp.Hash(); got != rootBefore {
		t.Fatalf("the copy did not start from the same state: %x vs %x", got, rootBefore)
	}
	// Mutating the copy must leave the original where it was.
	if err := cp.UpdateAccount(common.Address{0x02}, testAccount(2, 200), 0); err != nil {
		t.Fatal(err)
	}
	if got := original.Hash(); got != rootBefore {
		t.Fatalf("writing to the copy changed the original: %x, was %x", got, rootBefore)
	}
	if cp.Hash() == rootBefore {
		t.Fatal("writing to the copy did not change it; the two may be the same tree")
	}
	// And the reverse, since aliasing in either direction is the same bug.
	if err := original.UpdateAccount(common.Address{0x03}, testAccount(3, 300), 0); err != nil {
		t.Fatal(err)
	}
	if cp.Hash() == original.Hash() {
		t.Fatal("writing to the original changed the copy")
	}
}
