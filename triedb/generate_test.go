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

package triedb

import (
	"bytes"
	"context"
	"sort"
	"sync/atomic"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/holiman/uint256"
)

// testAccount is a helper for building test state with deterministic ordering.
type testAccount struct {
	hash    common.Hash
	account types.StateAccount
	storage []testSlot // must be sorted by hash
}

type testSlot struct {
	hash  common.Hash
	value []byte
}

// buildExpectedRoot computes the state root from sorted test accounts using
// StackTrie (which requires sorted key insertion).
func buildExpectedRoot(t *testing.T, accounts []testAccount) common.Hash {
	t.Helper()
	// Sort accounts by hash
	sort.Slice(accounts, func(i, j int) bool {
		return bytes.Compare(accounts[i].hash[:], accounts[j].hash[:]) < 0
	})
	acctTrie := trie.NewStackTrie(nil)
	for i := range accounts {
		data, err := rlp.EncodeToBytes(&accounts[i].account)
		if err != nil {
			t.Fatal(err)
		}
		acctTrie.Update(accounts[i].hash[:], data)
	}
	return acctTrie.Hash()
}

// computeStorageRootFromSlots computes the storage trie root from sorted slots.
func computeStorageRootFromSlots(slots []testSlot) common.Hash {
	sort.Slice(slots, func(i, j int) bool {
		return bytes.Compare(slots[i].hash[:], slots[j].hash[:]) < 0
	})
	st := trie.NewStackTrie(nil)
	for _, s := range slots {
		st.Update(s.hash[:], s.value)
	}
	return st.Hash()
}

func TestGenerateTrieEmpty(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	if err := GenerateTrie(db, rawdb.HashScheme, types.EmptyRootHash, nil); err != nil {
		t.Fatalf("GenerateTrie on empty state failed: %v", err)
	}
}

func TestGenerateTrieAccountsOnly(t *testing.T) {
	db := rawdb.NewMemoryDatabase()

	accounts := []testAccount{
		{
			hash: common.HexToHash("0x01"),
			account: types.StateAccount{
				Nonce:    1,
				Balance:  uint256.NewInt(100),
				Root:     types.EmptyRootHash,
				CodeHash: types.EmptyCodeHash.Bytes(),
			},
		},
		{
			hash: common.HexToHash("0x02"),
			account: types.StateAccount{
				Nonce:    2,
				Balance:  uint256.NewInt(200),
				Root:     types.EmptyRootHash,
				CodeHash: types.EmptyCodeHash.Bytes(),
			},
		},
	}
	for _, a := range accounts {
		rawdb.WriteAccountSnapshot(db, a.hash, types.SlimAccountRLP(a.account))
	}
	root := buildExpectedRoot(t, accounts)

	if err := GenerateTrie(db, rawdb.HashScheme, root, nil); err != nil {
		t.Fatalf("GenerateTrie failed: %v", err)
	}
}

func TestGenerateTrieWithStorage(t *testing.T) {
	db := rawdb.NewMemoryDatabase()

	slots := []testSlot{
		{hash: common.HexToHash("0xaa"), value: []byte{0x01, 0x02, 0x03}},
		{hash: common.HexToHash("0xbb"), value: []byte{0x04, 0x05, 0x06}},
	}
	storageRoot := computeStorageRootFromSlots(slots)

	accounts := []testAccount{
		{
			hash: common.HexToHash("0x01"),
			account: types.StateAccount{
				Nonce:    1,
				Balance:  uint256.NewInt(100),
				Root:     storageRoot,
				CodeHash: types.EmptyCodeHash.Bytes(),
			},
			storage: slots,
		},
		{
			hash: common.HexToHash("0x02"),
			account: types.StateAccount{
				Nonce:    0,
				Balance:  uint256.NewInt(50),
				Root:     types.EmptyRootHash,
				CodeHash: types.EmptyCodeHash.Bytes(),
			},
		},
	}
	// Write account snapshots
	for _, a := range accounts {
		rawdb.WriteAccountSnapshot(db, a.hash, types.SlimAccountRLP(a.account))
	}
	// Write storage snapshots
	for _, a := range accounts {
		for _, s := range a.storage {
			rawdb.WriteStorageSnapshot(db, a.hash, s.hash, s.value)
		}
	}
	root := buildExpectedRoot(t, accounts)

	if err := GenerateTrie(db, rawdb.HashScheme, root, nil); err != nil {
		t.Fatalf("GenerateTrie failed: %v", err)
	}
}

func TestGenerateTrieRootMismatch(t *testing.T) {
	db := rawdb.NewMemoryDatabase()

	acct := types.StateAccount{
		Nonce:    1,
		Balance:  uint256.NewInt(100),
		Root:     types.EmptyRootHash,
		CodeHash: types.EmptyCodeHash.Bytes(),
	}
	rawdb.WriteAccountSnapshot(db, common.HexToHash("0x01"), types.SlimAccountRLP(acct))

	wrongRoot := common.HexToHash("0xdeadbeef")
	err := GenerateTrie(db, rawdb.HashScheme, wrongRoot, nil)
	if err == nil {
		t.Fatal("expected error for root mismatch, got nil")
	}
}

// TestGenerateTrieFixesStaleRoots writes flat state with a mix of stale,
// empty, and correct account roots, then checks that GenerateTrie produces
// the expected state root.
func TestGenerateTrieFixesStaleRoots(t *testing.T) {
	db := rawdb.NewMemoryDatabase()

	const n = 300
	accounts := make([]testAccount, 0, n)
	for i := 0; i < n; i++ {
		addr := common.BytesToAddress([]byte{byte(i >> 8), byte(i)})
		hash := crypto.Keccak256Hash(addr[:])

		acc := testAccount{
			hash: hash,
			account: types.StateAccount{
				Nonce:    uint64(i),
				Balance:  uint256.NewInt(uint64(i + 1)),
				Root:     types.EmptyRootHash,
				CodeHash: types.EmptyCodeHash.Bytes(),
			},
		}
		// Every third account has no storage; the rest get slots.
		if i%3 != 0 {
			acc.storage = []testSlot{
				{hash: common.BytesToHash([]byte{byte(i), 0xaa}), value: []byte{byte(i), 0x01}},
				{hash: common.BytesToHash([]byte{byte(i), 0xbb}), value: []byte{byte(i), 0x02}},
			}
			acc.account.Root = computeStorageRootFromSlots(acc.storage)
		}
		accounts = append(accounts, acc)
	}
	// Expected state root with all Roots correct.
	expectedRoot := buildExpectedRoot(t, accounts)

	// Write flat state. Storage-bearing accounts rotate through three on-disk
	// Root states that GenerateTrie's pre-pass must all bring into alignment:
	//   - stale non-empty Root
	//   - stale empty Root
	//   - correct Root
	for i, a := range accounts {
		for _, s := range a.storage {
			rawdb.WriteStorageSnapshot(db, a.hash, s.hash, s.value)
		}
		onDisk := a.account
		if len(a.storage) > 0 {
			switch i % 3 {
			case 0:
				onDisk.Root = common.BytesToHash([]byte{byte(i), 0xde, 0xad})
			case 1:
				onDisk.Root = types.EmptyRootHash
			}
		}
		rawdb.WriteAccountSnapshot(db, a.hash, types.SlimAccountRLP(onDisk))
	}

	if err := GenerateTrie(db, rawdb.HashScheme, expectedRoot, nil); err != nil {
		t.Fatalf("GenerateTrie failed: %v", err)
	}
}

// TestGenerateTrieCancel verifies GenerateTrie respects the cancel channel.
func TestGenerateTrieCancel(t *testing.T) {
	t.Parallel()
	db := rawdb.NewMemoryDatabase()

	for i := 0; i < 100; i++ {
		addr := common.BytesToAddress([]byte{byte(i)})
		hash := crypto.Keccak256Hash(addr[:])
		rawdb.WriteAccountSnapshot(db, hash, types.SlimAccountRLP(types.StateAccount{
			Balance:  uint256.NewInt(1),
			Root:     types.EmptyRootHash,
			CodeHash: types.EmptyCodeHash[:],
		}))
	}

	cancel := make(chan struct{})
	close(cancel)
	if err := GenerateTrie(db, rawdb.HashScheme, common.Hash{}, cancel); err != ErrCancelled {
		t.Fatalf("expected ErrCancelled, got %v", err)
	}
}

// TestGenerateTrieOrphanStorage exercises the orphan-slot skip path: flat
// storage entries for an accountHash that has no corresponding account
// snapshot. updateStorageRoots must skip these without including them in
// any account's storage root.
func TestGenerateTrieOrphanStorage(t *testing.T) {
	db := rawdb.NewMemoryDatabase()

	// One legitimate account with storage.
	liveAccountHash := crypto.Keccak256Hash(common.HexToAddress("0x01").Bytes())
	slots := []testSlot{
		{hash: common.HexToHash("0xaa"), value: []byte{0x01}},
	}
	for _, s := range slots {
		rawdb.WriteStorageSnapshot(db, liveAccountHash, s.hash, s.value)
	}
	acc := testAccount{
		hash: liveAccountHash,
		account: types.StateAccount{
			Nonce:    1,
			Balance:  uint256.NewInt(1),
			Root:     computeStorageRootFromSlots(slots),
			CodeHash: types.EmptyCodeHash.Bytes(),
		},
		storage: slots,
	}
	rawdb.WriteAccountSnapshot(db, acc.hash, types.SlimAccountRLP(acc.account))

	// Orphan storage: entries for an accountHash smaller than liveAccountHash,
	// with no account snapshot behind them. Must be ordered before liveAccountHash
	// so the storage iterator encounters them first.
	var orphanAccountHash common.Hash
	copy(orphanAccountHash[:], liveAccountHash[:])
	orphanAccountHash[0] = 0x00 // guarantees cmp < 0 against liveAccountHash
	rawdb.WriteStorageSnapshot(db, orphanAccountHash, common.HexToHash("0xbb"), []byte{0x02})

	expectedRoot := buildExpectedRoot(t, []testAccount{acc})

	if err := GenerateTrie(db, rawdb.HashScheme, expectedRoot, nil); err != nil {
		t.Fatalf("GenerateTrie with orphan storage failed: %v", err)
	}
}

// TestGenerateTriePartialResume proves that the resume path actually
// fires when a partition's done marker is present.
func TestGenerateTriePartialResume(t *testing.T) {
	db := rawdb.NewMemoryDatabase()

	// Build flat state. Empty storage keeps the test focused on the
	// account-trie resume path.
	const n = 200
	accounts := make([]testAccount, 0, n)
	for i := 0; i < n; i++ {
		addr := common.BytesToAddress([]byte{byte(i >> 8), byte(i)})
		hash := crypto.Keccak256Hash(addr[:])
		acc := testAccount{
			hash: hash,
			account: types.StateAccount{
				Nonce:    uint64(i),
				Balance:  uint256.NewInt(uint64(i + 1)),
				Root:     types.EmptyRootHash,
				CodeHash: types.EmptyCodeHash.Bytes(),
			},
		}
		rawdb.WriteAccountSnapshot(db, acc.hash, types.SlimAccountRLP(acc.account))
		accounts = append(accounts, acc)
	}
	expectedRoot := buildExpectedRoot(t, accounts)

	// Step 2: run every partition once to populate trie nodes on disk
	// and capture each partition's raw root blob.
	var scanned, updated atomic.Int64
	ranges := hashRanges(numPartitions)
	blobs := make([][]byte, numPartitions)
	for i, r := range ranges {
		blob, err := generatePartition(context.Background(), nil, db, rawdb.HashScheme, byte(i), r[0], r[1], &scanned, &updated)
		if err != nil {
			t.Fatalf("pre-run partition %d: %v", i, err)
		}
		blobs[i] = blob
	}

	// Step 3: pre-seed done markers for even partitions only.
	for i := 0; i < numPartitions; i++ {
		if i%2 == 0 {
			rawdb.WriteGenerateTriePartitionDone(db, byte(i), blobs[i])
		}
	}

	// Step 4: delete flat-state account snapshots for every account that
	// lives in an even partition. After this, rerunning generatePartition
	// for an even partition would find no accounts and produce a nil
	// blob — so a correct final root requires the resume path.
	deleted := 0
	for _, a := range accounts {
		if (a.hash[0]>>4)%2 == 0 {
			rawdb.DeleteAccountSnapshot(db, a.hash)
			deleted++
		}
	}
	if deleted == 0 {
		t.Fatal("test setup failure: no accounts fell in even partitions")
	}

	// Step 5: run GenerateTrie. Success implies resume actually consulted
	// the markers — without it, even partitions would yield nil blobs and
	// the root check inside GenerateTrie would fail.
	if err := GenerateTrie(db, rawdb.HashScheme, expectedRoot, nil); err != nil {
		t.Fatalf("partial-resume GenerateTrie failed: %v", err)
	}

	// All markers cleared on success.
	for i := 0; i < numPartitions; i++ {
		if _, ok := rawdb.ReadGenerateTriePartitionDone(db, byte(i)); ok {
			t.Errorf("partition %d marker not cleared after successful resume", i)
		}
	}
}
