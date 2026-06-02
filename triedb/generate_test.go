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
	"math/big"
	"sort"
	"sync/atomic"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
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
	if _, err := GenerateTrie(db, rawdb.HashScheme, types.EmptyRootHash, nil); err != nil {
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

	if _, err := GenerateTrie(db, rawdb.HashScheme, root, nil); err != nil {
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

	if _, err := GenerateTrie(db, rawdb.HashScheme, root, nil); err != nil {
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
	_, err := GenerateTrie(db, rawdb.HashScheme, wrongRoot, nil)
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

	if _, err := GenerateTrie(db, rawdb.HashScheme, expectedRoot, nil); err != nil {
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
	if _, err := GenerateTrie(db, rawdb.HashScheme, common.Hash{}, cancel); err != ErrCancelled {
		t.Fatalf("expected ErrCancelled, got %v", err)
	}
}

// TestGenerateTrieOrphanStorage exercises dangling-slot cleanup: flat storage
// entries for an accountHash that has no corresponding account snapshot must
// be deleted, regardless of whether they sit before, between, or after the
// live accounts within a partition. The state root must match and the
// Deleted counter must reflect every dangling entry.
func TestGenerateTrieOrphanStorage(t *testing.T) {
	db := rawdb.NewMemoryDatabase()

	// Two legitimate accounts in the same partition (first nibble 0x5) so
	// orphans can be placed before, between, and after them in the shared
	// per-partition storage iterator.
	liveA := common.HexToHash("0x5300000000000000000000000000000000000000000000000000000000000000")
	liveB := common.HexToHash("0x5900000000000000000000000000000000000000000000000000000000000000")
	slotsA := []testSlot{{hash: common.HexToHash("0xaa"), value: []byte{0xa1}}}
	slotsB := []testSlot{{hash: common.HexToHash("0xbb"), value: []byte{0xb1}}}

	accounts := []testAccount{
		{
			hash: liveA,
			account: types.StateAccount{
				Nonce:    1,
				Balance:  uint256.NewInt(1),
				Root:     computeStorageRootFromSlots(slotsA),
				CodeHash: types.EmptyCodeHash.Bytes(),
			},
			storage: slotsA,
		},
		{
			hash: liveB,
			account: types.StateAccount{
				Nonce:    2,
				Balance:  uint256.NewInt(2),
				Root:     computeStorageRootFromSlots(slotsB),
				CodeHash: types.EmptyCodeHash.Bytes(),
			},
			storage: slotsB,
		},
	}
	for _, a := range accounts {
		rawdb.WriteAccountSnapshot(db, a.hash, types.SlimAccountRLP(a.account))
		for _, s := range a.storage {
			rawdb.WriteStorageSnapshot(db, a.hash, s.hash, s.value)
		}
	}

	// Dangling slots at three positions within partition 5:
	//   before liveA, between liveA and liveB, after liveB.
	orphans := []struct {
		account common.Hash
		slots   []testSlot
	}{
		{
			account: common.HexToHash("0x5000000000000000000000000000000000000000000000000000000000000000"),
			slots: []testSlot{
				{hash: common.HexToHash("0x11"), value: []byte{0x01}},
				{hash: common.HexToHash("0x22"), value: []byte{0x02}},
			},
		},
		{
			account: common.HexToHash("0x5600000000000000000000000000000000000000000000000000000000000000"),
			slots:   []testSlot{{hash: common.HexToHash("0x33"), value: []byte{0x03}}},
		},
		{
			account: common.HexToHash("0x5d00000000000000000000000000000000000000000000000000000000000000"),
			slots: []testSlot{
				{hash: common.HexToHash("0x44"), value: []byte{0x04}},
				{hash: common.HexToHash("0x55"), value: []byte{0x05}},
			},
		},
	}
	var totalOrphans int64
	for _, o := range orphans {
		for _, s := range o.slots {
			rawdb.WriteStorageSnapshot(db, o.account, s.hash, s.value)
			totalOrphans++
		}
	}

	expectedRoot := buildExpectedRoot(t, accounts)

	stats, err := GenerateTrie(db, rawdb.HashScheme, expectedRoot, nil)
	if err != nil {
		t.Fatalf("GenerateTrie with orphan storage failed: %v", err)
	}
	if stats.Deleted != totalOrphans {
		t.Errorf("Deleted counter = %d, want %d", stats.Deleted, totalOrphans)
	}
	for _, o := range orphans {
		for _, s := range o.slots {
			if v := rawdb.ReadStorageSnapshot(db, o.account, s.hash); v != nil {
				t.Errorf("dangling slot %x/%x not cleared, got %x", o.account, s.hash, v)
			}
		}
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
	var (
		scanned atomic.Int64
		updated atomic.Int64
		deleted atomic.Int64
	)
	ranges := hashRanges(numPartitions)
	blobs := make([][]byte, numPartitions)
	for i, r := range ranges {
		var pos atomic.Uint64
		blob, err := generatePartition(context.Background(), nil, db, rawdb.HashScheme, byte(i), r[0], r[1], &scanned, &updated, &deleted, &pos)
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
	numDeleted := 0
	for _, a := range accounts {
		if (a.hash[0]>>4)%2 == 0 {
			rawdb.DeleteAccountSnapshot(db, a.hash)
			numDeleted++
		}
	}
	if numDeleted == 0 {
		t.Fatal("test setup failure: no accounts fell in even partitions")
	}

	// Step 5: run GenerateTrie. Success implies resume actually consulted
	// the markers — without it, even partitions would yield nil blobs and
	// the root check inside GenerateTrie would fail.
	if _, err := GenerateTrie(db, rawdb.HashScheme, expectedRoot, nil); err != nil {
		t.Fatalf("partial-resume GenerateTrie failed: %v", err)
	}

	// All markers cleared on success.
	for i := 0; i < numPartitions; i++ {
		if _, ok := rawdb.ReadGenerateTriePartitionDone(db, byte(i)); ok {
			t.Errorf("partition %d marker not cleared after successful resume", i)
		}
	}
}

// TestHashRanges checks that hashRanges fully and contiguously covers the
// 256-bit hash space, with the last range absorbing the rounding remainder.
func TestHashRanges(t *testing.T) {
	for _, total := range []int{1, 2, 16, 256} {
		ranges := hashRanges(total)
		if len(ranges) != total {
			t.Fatalf("total=%d: got %d ranges, want %d", total, len(ranges), total)
		}
		if ranges[0][0] != (common.Hash{}) {
			t.Errorf("total=%d: first range starts at %x, want zero", total, ranges[0][0])
		}
		if ranges[total-1][1] != common.MaxHash {
			t.Errorf("total=%d: last range ends at %x, want MaxHash", total, ranges[total-1][1])
		}
		for i, r := range ranges {
			if r[0].Big().Cmp(r[1].Big()) > 0 {
				t.Errorf("total=%d: range %d malformed: start %x > end %x", total, i, r[0], r[1])
			}
			if i == 0 {
				continue
			}
			gap := new(big.Int).Sub(r[0].Big(), ranges[i-1][1].Big())
			if gap.Cmp(common.Big1) != 0 {
				t.Errorf("total=%d: range %d not contiguous with %d (gap=%s)", total, i, i-1, gap)
			}
		}
	}
}

// TestGenerateTriePathSchemeNodeSet runs GenerateTrie on the path scheme and
// checks the persisted account-trie node set against a canonical StackTrie. A
// root-only check can't see the single-partition orphan, but a node-set diff can.
func TestGenerateTriePathSchemeNodeSet(t *testing.T) {
	mkAccount := func(hashHex string) testAccount {
		// Empty storage and no code, so the account trie is the only trie built
		// and the canonical reference is a plain StackTrie over the accounts.
		return testAccount{
			hash: common.HexToHash(hashHex),
			account: types.StateAccount{
				Nonce:    1,
				Balance:  uint256.NewInt(1),
				Root:     types.EmptyRootHash,
				CodeHash: types.EmptyCodeHash.Bytes(),
			},
		}
	}

	cases := []struct {
		name     string
		desc     string
		accounts []testAccount
	}{
		{
			name:     "single account, leaf root",
			desc:     "single populated partition, leaf subtree root: node at [5] is orphaned and must be deleted",
			accounts: []testAccount{mkAccount("0x5a00000000000000000000000000000000000000000000000000000000000000")},
		},
		{
			name: "two accounts sharing two nibbles, extension root",
			desc: "single populated partition, extension subtree root: node at [5] is orphaned and must be deleted",
			accounts: []testAccount{
				mkAccount("0x5300000000000000000000000000000000000000000000000000000000000000"),
				mkAccount("0x5320000000000000000000000000000000000000000000000000000000000000"),
			},
		},
		{
			name: "two accounts diverging at second nibble, branch root",
			desc: "single populated partition, branch subtree root: [5] stays referenced, no orphan",
			accounts: []testAccount{
				mkAccount("0x5a00000000000000000000000000000000000000000000000000000000000000"),
				mkAccount("0x5f00000000000000000000000000000000000000000000000000000000000000"),
			},
		},
		{
			name: "accounts across multiple partitions",
			desc: "multiple populated partitions: every [i] referenced by the top branch, no orphan",
			accounts: []testAccount{
				mkAccount("0x1000000000000000000000000000000000000000000000000000000000000000"),
				mkAccount("0x5a00000000000000000000000000000000000000000000000000000000000000"),
				mkAccount("0x5f00000000000000000000000000000000000000000000000000000000000000"),
				mkAccount("0xc000000000000000000000000000000000000000000000000000000000000000"),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := rawdb.NewMemoryDatabase()
			for _, a := range tc.accounts {
				rawdb.WriteAccountSnapshot(db, a.hash, types.SlimAccountRLP(a.account))
			}
			root := buildExpectedRoot(t, tc.accounts)

			if _, err := GenerateTrie(db, rawdb.PathScheme, root, nil); err != nil {
				t.Fatalf("GenerateTrie (path scheme) failed: %v", err)
			}

			want := canonicalAccountNodePaths(t, tc.accounts)
			got := diskAccountNodePaths(db)

			for p := range got {
				if _, ok := want[p]; !ok {
					t.Errorf("extra account-trie node on disk at path %x [%s]", p, tc.desc)
				}
			}
			for p := range want {
				if _, ok := got[p]; !ok {
					t.Errorf("missing canonical account-trie node at path %x [%s]", p, tc.desc)
				}
			}
		})
	}
}

// canonicalAccountNodePaths builds a StackTrie from the accounts and returns
// the set of node paths it emits.
func canonicalAccountNodePaths(t *testing.T, accounts []testAccount) map[string]struct{} {
	t.Helper()
	sorted := make([]testAccount, len(accounts))
	copy(sorted, accounts)
	sort.Slice(sorted, func(i, j int) bool {
		return bytes.Compare(sorted[i].hash[:], sorted[j].hash[:]) < 0
	})
	paths := make(map[string]struct{})
	st := trie.NewStackTrie(func(path []byte, hash common.Hash, blob []byte) {
		paths[string(path)] = struct{}{}
	})
	for i := range sorted {
		data, err := rlp.EncodeToBytes(&sorted[i].account)
		if err != nil {
			t.Fatal(err)
		}
		if err := st.Update(sorted[i].hash[:], data); err != nil {
			t.Fatal(err)
		}
	}
	st.Hash() // flush to emit the root node at path nil
	return paths
}

// diskAccountNodePaths returns the set of account-trie node paths persisted
// under the path scheme (keyed TrieNodeAccountPrefix + hexPath).
func diskAccountNodePaths(db ethdb.Database) map[string]struct{} {
	paths := make(map[string]struct{})
	it := db.NewIterator(rawdb.TrieNodeAccountPrefix, nil)
	defer it.Release()
	for it.Next() {
		paths[string(it.Key()[len(rawdb.TrieNodeAccountPrefix):])] = struct{}{}
	}
	return paths
}
