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
	root := buildExpectedRoot(t, accounts)

	for _, scheme := range []string{rawdb.HashScheme, rawdb.PathScheme} {
		t.Run(scheme, func(t *testing.T) {
			db := rawdb.NewMemoryDatabase()
			for _, a := range accounts {
				rawdb.WriteAccountSnapshot(db, a.hash, types.SlimAccountRLP(a.account))
				for _, s := range a.storage {
					rawdb.WriteStorageSnapshot(db, a.hash, s.hash, s.value)
				}
			}
			if _, err := GenerateTrie(db, scheme, root, nil); err != nil {
				t.Fatalf("GenerateTrie failed: %v", err)
			}
			if scheme == rawdb.PathScheme {
				assertCanonicalNodes(t, db, accounts)
			}
		})
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

	for _, scheme := range []string{rawdb.HashScheme, rawdb.PathScheme} {
		t.Run(scheme, func(t *testing.T) {
			db := rawdb.NewMemoryDatabase()

			// Write flat state. Storage-bearing accounts rotate through three
			// on-disk Root states that GenerateTrie's pre-pass must all bring
			// into alignment:
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

			if _, err := GenerateTrie(db, scheme, expectedRoot, nil); err != nil {
				t.Fatalf("GenerateTrie failed: %v", err)
			}
			if scheme == rawdb.PathScheme {
				assertCanonicalNodes(t, db, accounts)
			}
		})
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
		accounts []testAccount
	}{
		{
			// One populated partition whose subtree root is a leaf. The node the
			// partition wrote at [5] is left unreferenced, so GenerateTrie has to
			// delete it.
			name:     "single account, leaf root",
			accounts: []testAccount{mkAccount("0x5a00000000000000000000000000000000000000000000000000000000000000")},
		},
		{
			// One populated partition whose subtree root is an extension. Like the
			// leaf case, the node at [5] is left unreferenced and must be deleted.
			name: "two accounts sharing two nibbles, extension root",
			accounts: []testAccount{
				mkAccount("0x5300000000000000000000000000000000000000000000000000000000000000"),
				mkAccount("0x5320000000000000000000000000000000000000000000000000000000000000"),
			},
		},
		{
			// One populated partition whose subtree root is a branch. Here [5] stays
			// referenced by the new root, so nothing is orphaned.
			name: "two accounts diverging at second nibble, branch root",
			accounts: []testAccount{
				mkAccount("0x5a00000000000000000000000000000000000000000000000000000000000000"),
				mkAccount("0x5f00000000000000000000000000000000000000000000000000000000000000"),
			},
		},
		{
			// Several populated partitions. Every [i] stays referenced by the top
			// branch, so nothing is orphaned.
			name: "accounts across multiple partitions",
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
			assertCanonicalNodes(t, db, tc.accounts)
		})
	}
}

// assertCanonicalNodes checks that the trie nodes persisted under the path
// scheme exactly match the canonical set: the account-trie nodes a StackTrie
// over the accounts emits, plus, per account with slots, the storage-trie nodes
// a StackTrie over those slots emits. accounts must carry their final Root
// values (post storage-root reconciliation).
func assertCanonicalNodes(t *testing.T, db ethdb.Database, accounts []testAccount) {
	t.Helper()

	sorted := make([]testAccount, len(accounts))
	copy(sorted, accounts)
	sort.Slice(sorted, func(i, j int) bool {
		return bytes.Compare(sorted[i].hash[:], sorted[j].hash[:]) < 0
	})

	// Canonical account-trie node paths.
	wantAccount := make(map[string]struct{})
	acct := trie.NewStackTrie(func(path []byte, _ common.Hash, _ []byte) {
		wantAccount[string(path)] = struct{}{}
	})
	for i := range sorted {
		data, err := rlp.EncodeToBytes(&sorted[i].account)
		if err != nil {
			t.Fatal(err)
		}
		if err := acct.Update(sorted[i].hash[:], data); err != nil {
			t.Fatal(err)
		}
	}
	acct.Hash()

	// Canonical storage-trie node keys (accountHash ++ path), one StackTrie per
	// account that has slots.
	wantStorage := make(map[string]struct{})
	for _, a := range accounts {
		if len(a.storage) == 0 {
			continue
		}
		slots := make([]testSlot, len(a.storage))
		copy(slots, a.storage)
		sort.Slice(slots, func(i, j int) bool {
			return bytes.Compare(slots[i].hash[:], slots[j].hash[:]) < 0
		})
		owner := a.hash
		st := trie.NewStackTrie(func(path []byte, _ common.Hash, _ []byte) {
			wantStorage[string(owner[:])+string(path)] = struct{}{}
		})
		for _, s := range slots {
			if err := st.Update(s.hash[:], s.value); err != nil {
				t.Fatal(err)
			}
		}
		st.Hash()
	}

	assertSameNodeSet(t, "account", diskNodeKeys(db, rawdb.TrieNodeAccountPrefix), wantAccount)
	assertSameNodeSet(t, "storage", diskNodeKeys(db, rawdb.TrieNodeStoragePrefix), wantStorage)
}

// diskNodeKeys returns the set of path-scheme node keys with the given prefix
// stripped (account: hexPath; storage: accountHash ++ hexPath).
func diskNodeKeys(db ethdb.Database, prefix []byte) map[string]struct{} {
	keys := make(map[string]struct{})
	it := db.NewIterator(prefix, nil)
	defer it.Release()
	for it.Next() {
		keys[string(it.Key()[len(prefix):])] = struct{}{}
	}
	return keys
}

// assertSameNodeSet fails if got and want differ, reporting each offending key.
func assertSameNodeSet(t *testing.T, label string, got, want map[string]struct{}) {
	t.Helper()
	for k := range got {
		if _, ok := want[k]; !ok {
			t.Errorf("%s-trie: extra node on disk at %x", label, k)
		}
	}
	for k := range want {
		if _, ok := got[k]; !ok {
			t.Errorf("%s-trie: missing node on disk at %x", label, k)
		}
	}
}

// peakBatch records the largest ValueSize the batch reaches before any flush.
type peakBatch struct {
	ethdb.Batch
	peak *int
}

func (b *peakBatch) Write() error {
	if s := b.ValueSize(); s > *b.peak {
		*b.peak = s
	}
	return b.Batch.Write()
}

// peakBatchDB hands out peakBatch batches so a test can observe how large the
// write batch grows between flushes.
type peakBatchDB struct {
	ethdb.Database
	peak *int
}

func (d peakBatchDB) NewBatch() ethdb.Batch {
	return &peakBatch{Batch: d.Database.NewBatch(), peak: d.peak}
}

func (d peakBatchDB) NewBatchWithSize(size int) ethdb.Batch {
	return &peakBatch{Batch: d.Database.NewBatchWithSize(size), peak: d.peak}
}

// TestGenerateTrieBatchFlush drives each of generatePartition's batch-flush
// sites past IdealBatchSize and checks the write batch stays bounded (so the
// flush fired) without dropping or skipping any entry.
func TestGenerateTrieBatchFlush(t *testing.T) {
	// h builds a unique partition-0 hash (leading nibble 0) from an int, used
	// for both account hashes and storage slot hashes.
	h := func(i int) common.Hash {
		return common.BytesToHash([]byte{0x00, byte(i >> 16), byte(i >> 8), byte(i)})
	}
	acct := func(root common.Hash) types.StateAccount {
		return types.StateAccount{Nonce: 1, Balance: uint256.NewInt(1), Root: root, CodeHash: types.EmptyCodeHash.Bytes()}
	}
	// Each fixture writes this many entries into partition 0, enough that one
	// flush site overflows IdealBatchSize several times over.
	const n = 5000

	cases := []struct {
		name        string
		build       func(db ethdb.Database)
		wantScanned int64
		wantDeleted int64
	}{
		{
			// Dangling account (no snapshot) sorting before a live account, so its
			// slots are deleted inline (cmp < 0) while the live account is built.
			name: "inline dangling deletes",
			build: func(db ethdb.Database) {
				for i := 0; i < n; i++ {
					rawdb.WriteStorageSnapshot(db, h(1), h(i), []byte{0x01})
				}
				rawdb.WriteAccountSnapshot(db, h(0xffffff), types.SlimAccountRLP(acct(types.EmptyRootHash)))
			},
			wantScanned: 1,
			wantDeleted: n,
		},
		{
			// Dangling account with no live account at all, so every slot is
			// cleared by the tail loop after the account iterator is exhausted.
			name: "tail dangling deletes",
			build: func(db ethdb.Database) {
				for i := 0; i < n; i++ {
					rawdb.WriteStorageSnapshot(db, h(1), h(i), []byte{0x01})
				}
			},
			wantScanned: 0,
			wantDeleted: n,
		},
		{
			// One account whose storage trie alone overflows the batch, so the
			// cmp == 0 storage path flushes mid-build. updated stays 0 only if
			// every slot survived the flush and iterator reopen.
			name: "single account, large storage",
			build: func(db ethdb.Database) {
				slots := make([]testSlot, n)
				for i := range slots {
					slots[i] = testSlot{hash: h(i), value: bytes.Repeat([]byte{byte(i)}, 32)}
				}
				rawdb.WriteAccountSnapshot(db, h(7), types.SlimAccountRLP(acct(computeStorageRootFromSlots(slots))))
				for _, s := range slots {
					rawdb.WriteStorageSnapshot(db, h(7), s.hash, s.value)
				}
			},
			wantScanned: 1,
			wantDeleted: 0,
		},
		{
			// Many empty-storage accounts so the account trie alone overflows the
			// batch, exercising the per-account flush. A skipped account would not
			// be counted in scanned.
			name: "many accounts",
			build: func(db ethdb.Database) {
				for i := 0; i < n; i++ {
					rawdb.WriteAccountSnapshot(db, h(i), types.SlimAccountRLP(acct(types.EmptyRootHash)))
				}
			},
			wantScanned: n,
			wantDeleted: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := rawdb.NewMemoryDatabase()
			tc.build(db)

			peak := 0
			var scanned, updated, deleted atomic.Int64
			var pos atomic.Uint64
			ranges := hashRanges(numPartitions)
			if _, err := generatePartition(context.Background(), nil, peakBatchDB{Database: db, peak: &peak},
				rawdb.HashScheme, 0, ranges[0][0], ranges[0][1], &scanned, &updated, &deleted, &pos); err != nil {
				t.Fatalf("generatePartition: %v", err)
			}

			if scanned.Load() != tc.wantScanned {
				t.Errorf("scanned = %d, want %d (an account was skipped?)", scanned.Load(), tc.wantScanned)
			}
			if deleted.Load() != tc.wantDeleted {
				t.Errorf("deleted = %d, want %d", deleted.Load(), tc.wantDeleted)
			}
			if updated.Load() != 0 {
				t.Errorf("updated = %d, want 0 (a storage slot was dropped across a flush?)", updated.Load())
			}
			// The batch must have stayed bounded. Without this site's flush its
			// full write set (far larger than IdealBatchSize) buffers into one batch.
			if peak > 2*ethdb.IdealBatchSize {
				t.Errorf("peak batch size %d exceeded 2*IdealBatchSize (%d); flush did not fire", peak, 2*ethdb.IdealBatchSize)
			}
		})
	}
}
