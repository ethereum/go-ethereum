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

package state

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/trie"
)

// TestExhaustedIterator verifies the exhaustedIterator sentinel: Next is false,
// Error is nil, Hash/Key are zero, Slot is nil, and double Release is safe.
func TestExhaustedIterator(t *testing.T) {
	var it exhaustedIterator

	if it.Next() {
		t.Fatal("Next() returned true")
	}
	if err := it.Error(); err != nil {
		t.Fatalf("Error() = %v, want nil", err)
	}
	if hash := it.Hash(); hash != (common.Hash{}) {
		t.Fatalf("Hash() = %x, want zero", hash)
	}
	if key, err := it.Key(); key != (common.Hash{}) || err != nil {
		t.Fatalf("Key() = %x, %v; want zero, nil", key, err)
	}
	if slot := it.Slot(); slot != (common.Hash{}) {
		t.Fatalf("Slot() = %x, want nil", slot)
	}
	it.Release()
	it.Release()
}

// TestAccountIterator tests the account iterator: correct count, ascending
// hash order, valid full-format RLP, data integrity, address preimage
// resolution, and seek behavior.
func TestAccountIterator(t *testing.T) {
	testAccountIterator(t, rawdb.HashScheme)
	testAccountIterator(t, rawdb.PathScheme)
}

func testAccountIterator(t *testing.T, scheme string) {
	_, sdb, ndb, root, accounts := makeTestState(scheme)
	ndb.Commit(root, false)

	iteratee, err := sdb.Iteratee(root)
	if err != nil {
		t.Fatalf("(%s) failed to create iteratee: %v", scheme, err)
	}
	// Build lookups from address hash.
	addrByHash := make(map[common.Hash]*testAccount)
	for _, acc := range accounts {
		addrByHash[crypto.Keccak256Hash(acc.address.Bytes())] = acc
	}

	// --- Full iteration: count, ordering, RLP validity, data integrity, address resolution ---
	acctIt, err := iteratee.NewAccountIterator(common.Hash{})
	if err != nil {
		t.Fatalf("(%s) failed to create account iterator: %v", scheme, err)
	}
	var (
		hashes   []common.Hash
		prevHash common.Hash
	)
	for acctIt.Next() {
		hash := acctIt.Hash()
		if hash == (common.Hash{}) {
			t.Fatalf("(%s) zero hash at position %d", scheme, len(hashes))
		}
		if len(hashes) > 0 && bytes.Compare(prevHash.Bytes(), hash.Bytes()) >= 0 {
			t.Fatalf("(%s) hashes not ascending: %x >= %x", scheme, prevHash, hash)
		}
		prevHash = hash
		hashes = append(hashes, hash)

		// Decode and verify account data.
		got := acctIt.Account()
		if got == nil {
			t.Fatalf("(%s) nil account at %x", scheme, hash)
		}
		acc := addrByHash[hash]
		if got.Nonce != acc.nonce {
			t.Fatalf("(%s) nonce %x: got %d, want %d", scheme, hash, got.Nonce, acc.nonce)
		}
		if got.Balance.Cmp(acc.balance) != 0 {
			t.Fatalf("(%s) balance %x: got %v, want %v", scheme, hash, got.Balance, acc.balance)
		}
		// Verify address preimage resolution.
		addr, err := acctIt.Address()
		if err != nil {
			t.Fatalf("(%s) failed to address: %v", scheme, err)
		}
		if addr != acc.address {
			t.Fatalf("(%s) Address() = %x, want %x", scheme, addr, acc.address)
		}
	}
	acctIt.Release()

	if err := acctIt.Error(); err != nil {
		t.Fatalf("(%s) iteration error: %v", scheme, err)
	}
	if len(hashes) != len(accounts) {
		t.Fatalf("(%s) iterated %d accounts, want %d", scheme, len(hashes), len(accounts))
	}

	// --- Seek: starting from midpoint should skip earlier entries ---
	mid := hashes[len(hashes)/2]
	seekIt, err := iteratee.NewAccountIterator(mid)
	if err != nil {
		t.Fatalf("(%s) failed to create seeked iterator: %v", scheme, err)
	}
	seekCount := 0
	for seekIt.Next() {
		if bytes.Compare(seekIt.Hash().Bytes(), mid.Bytes()) < 0 {
			t.Fatalf("(%s) seeked iterator returned hash before start", scheme)
		}
		seekCount++
	}
	seekIt.Release()

	if seekCount != len(hashes)/2 {
		t.Fatalf("(%s) unexpected seeked count, %d != %d", scheme, seekCount, len(hashes)/2)
	}
}

// TestStorageIterator tests the storage iterator: correct slot counts against
// the trie, ascending hash order, non-nil slot data, key preimage resolution,
// seek behavior, and empty-storage accounts.
func TestStorageIterator(t *testing.T) {
	testStorageIterator(t, rawdb.HashScheme)
	testStorageIterator(t, rawdb.PathScheme)
}

func testStorageIterator(t *testing.T, scheme string) {
	_, sdb, ndb, root, accounts := makeTestState(scheme)
	ndb.Commit(root, false)

	iteratee, err := sdb.Iteratee(root)
	if err != nil {
		t.Fatalf("(%s) failed to create iteratee: %v", scheme, err)
	}

	// --- Slot count and ordering for every account ---
	var withStorage common.Hash // remember an account that has storage for seek test
	for _, acc := range accounts {
		addrHash := crypto.Keccak256Hash(acc.address.Bytes())
		expected := countStorageSlots(t, scheme, sdb, root, addrHash)

		storageIt, err := iteratee.NewStorageIterator(addrHash, common.Hash{})
		if err != nil {
			t.Fatalf("(%s) failed to create storage iterator for %x: %v", scheme, acc.address, err)
		}
		count := 0
		var prevHash common.Hash
		for storageIt.Next() {
			hash := storageIt.Hash()
			if count > 0 && bytes.Compare(prevHash.Bytes(), hash.Bytes()) >= 0 {
				t.Fatalf("(%s) storage hashes not ascending for %x", scheme, acc.address)
			}
			prevHash = hash
			if storageIt.Slot() == (common.Hash{}) {
				t.Fatalf("(%s) nil slot at %x", scheme, hash)
			}
			// Check key preimage resolution on first slot.
			if _, err := storageIt.Key(); err != nil {
				t.Fatalf("(%s) Key() failed to resolve", scheme)
			}
			count++
		}
		if err := storageIt.Error(); err != nil {
			t.Fatalf("(%s) storage iteration error for %x: %v", scheme, acc.address, err)
		}
		storageIt.Release()

		if count != expected {
			t.Fatalf("(%s) account %x: %d slots, want %d", scheme, acc.address, count, expected)
		}
		if count > 0 {
			withStorage = addrHash
		}
	}

	// --- Seek: starting from second slot should skip the first ---
	if withStorage == (common.Hash{}) {
		t.Fatalf("(%s) no account with storage found", scheme)
	}
	fullIt, err := iteratee.NewStorageIterator(withStorage, common.Hash{})
	if err != nil {
		t.Fatalf("(%s) failed to create full storage iterator: %v", scheme, err)
	}
	var slotHashes []common.Hash
	for fullIt.Next() {
		slotHashes = append(slotHashes, fullIt.Hash())
	}
	fullIt.Release()

	seekIt, err := iteratee.NewStorageIterator(withStorage, slotHashes[1])
	if err != nil {
		t.Fatalf("(%s) failed to create seeked storage iterator: %v", scheme, err)
	}
	seekCount := 0
	for seekIt.Next() {
		if bytes.Compare(seekIt.Hash().Bytes(), slotHashes[1].Bytes()) < 0 {
			t.Fatalf("(%s) seeked storage iterator returned hash before start", scheme)
		}
		seekCount++
	}
	seekIt.Release()

	if seekCount != len(slotHashes)-1 {
		t.Fatalf("(%s) unexpected seeked storage count %d != %d", scheme, seekCount, len(slotHashes)-1)
	}
}

// countStorageSlots counts storage slots for an account by opening the
// storage trie directly.
func countStorageSlots(t *testing.T, scheme string, sdb Database, root common.Hash, addrHash common.Hash) int {
	t.Helper()
	accTrie, err := trie.NewStateTrie(trie.StateTrieID(root), sdb.TrieDB())
	if err != nil {
		t.Fatalf("(%s) failed to open account trie: %v", scheme, err)
	}
	acct, err := accTrie.GetAccountByHash(addrHash)
	if err != nil || acct == nil || acct.Root == types.EmptyRootHash {
		return 0
	}
	storageTrie, err := trie.NewStateTrie(trie.StorageTrieID(root, addrHash, acct.Root), sdb.TrieDB())
	if err != nil {
		t.Fatalf("(%s) failed to open storage trie for %x: %v", scheme, addrHash, err)
	}
	it := trie.NewIterator(storageTrie.MustNodeIterator(nil))
	count := 0
	for it.Next() {
		count++
	}
	return count
}
