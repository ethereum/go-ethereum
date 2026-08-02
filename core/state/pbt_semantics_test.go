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

// State-layer semantics of the EIP-8297 tree: zero-as-absence, storage
// emptiness (the EIP-7610 predicate) and account destruction.

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/trie/bintrie"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/holiman/uint256"
)

// newPBTState returns a state database backed by a binary trie.
func newPBTState(t *testing.T) (*StateDB, *triedb.Database) {
	t.Helper()
	disk := rawdb.NewMemoryDatabase()
	db := triedb.NewDatabase(disk, triedb.PBTDefaults)
	sdb, err := New(types.EmptyBinaryHash, NewDatabase(db, nil))
	if err != nil {
		t.Fatal(err)
	}
	return sdb, db
}

// reopenPBT commits the state and returns a fresh StateDB at the new root.
func reopenPBT(t *testing.T, sdb *StateDB, db *triedb.Database, block uint64) *StateDB {
	t.Helper()
	root, err := sdb.Commit(block, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Commit(root, false); err != nil {
		t.Fatal(err)
	}
	next, err := New(root, NewDatabase(db, nil))
	if err != nil {
		t.Fatal(err)
	}
	return next
}

// TestPBTHasStorage covers the storage-emptiness predicate across both homes of
// an account's storage (header slots below 64 and the overflow bucket) and
// across the committed/uncommitted boundary. Under the binary tree there is no
// per-account storage root, so a regression here is invisible to
// GetStorageRoot.
//
// This pins the predicate, not a consensus rule. EIP-7610 rejects deployments
// from a hardcoded address list, so nothing in block processing consults
// HasStorage today.
func TestPBTHasStorage(t *testing.T) {
	var (
		empty    = common.Address{1}
		headerA  = common.Address{2} // slot below 64: lives in the header stem
		overflow = common.Address{3} // slot above 64: lives in the storage bucket
		both     = common.Address{4}
	)
	sdb, db := newPBTState(t)
	for _, addr := range []common.Address{empty, headerA, overflow, both} {
		sdb.CreateAccount(addr)
		sdb.SetNonce(addr, 1, tracing.NonceChangeUnspecified)
	}
	sdb.SetState(headerA, common.Hash{31: 5}, common.Hash{31: 7})
	sdb.SetState(overflow, common.Hash{30: 4}, common.Hash{31: 8}) // slot 1024
	sdb.SetState(both, common.Hash{31: 5}, common.Hash{31: 9})
	sdb.SetState(both, common.Hash{30: 4}, common.Hash{31: 10})

	// Uncommitted writes already make an account non-empty.
	if !sdb.HasStorage(headerA) {
		t.Fatal("uncommitted header slot not seen")
	}
	if !sdb.HasStorage(overflow) {
		t.Fatal("uncommitted overflow slot not seen")
	}
	if sdb.HasStorage(empty) {
		t.Fatal("empty account reported as having storage")
	}

	sdb = reopenPBT(t, sdb, db, 1)

	for _, tc := range []struct {
		addr common.Address
		want bool
	}{
		{empty, false},
		{headerA, true},
		{overflow, true},
		{both, true},
		{common.Address{0xEE}, false}, // never existed
	} {
		if got := sdb.HasStorage(tc.addr); got != tc.want {
			t.Fatalf("HasStorage(%x) = %v, want %v", tc.addr, got, tc.want)
		}
	}
}

// TestPBTZeroIsAbsence pins that writing zero removes the leaf: the root
// after a write-then-zero must equal the root of a state that never wrote.
func TestPBTZeroIsAbsence(t *testing.T) {
	addr := common.Address{1}
	slot := common.Hash{31: 5}

	// Reference: account with no storage.
	ref, refdb := newPBTState(t)
	ref.CreateAccount(addr)
	ref.SetNonce(addr, 1, tracing.NonceChangeUnspecified)
	ref.SetBalance(addr, uint256.NewInt(100), tracing.BalanceChangeUnspecified)
	refRoot, err := ref.Commit(1, true, false)
	if err != nil {
		t.Fatal(err)
	}
	_ = refdb

	// Same account, writing the slot and zeroing it in one block.
	sdb, _ := newPBTState(t)
	sdb.CreateAccount(addr)
	sdb.SetNonce(addr, 1, tracing.NonceChangeUnspecified)
	sdb.SetBalance(addr, uint256.NewInt(100), tracing.BalanceChangeUnspecified)
	sdb.SetState(addr, slot, common.Hash{31: 7})
	sdb.SetState(addr, slot, common.Hash{})
	got, err := sdb.Commit(1, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if got != refRoot {
		t.Fatalf("in-block zeroing left a trace: %x want %x", got, refRoot)
	}
	if sdb.HasStorage(addr) {
		t.Fatal("zeroed slot still counts as storage")
	}

	// And across blocks: write in block 1, zero in block 2.
	two, twodb := newPBTState(t)
	two.CreateAccount(addr)
	two.SetNonce(addr, 1, tracing.NonceChangeUnspecified)
	two.SetBalance(addr, uint256.NewInt(100), tracing.BalanceChangeUnspecified)
	two.SetState(addr, slot, common.Hash{31: 7})
	two = reopenPBT(t, two, twodb, 1)
	two.SetState(addr, slot, common.Hash{})
	got2, err := two.Commit(2, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if got2 != refRoot {
		t.Fatalf("cross-block zeroing left a trace: %x want %x", got2, refRoot)
	}
}

// TestPBTAccountDeletion pins that deleting an account removes everything it
// owns in the unified tree - header stem and overflow storage bucket alike -
// so the resulting root equals a state where the account never existed. A
// merkle-patricia state root never contains a destroyed account's storage
// (its trie simply becomes unreachable), so leaving those leaves behind
// would diverge from any conversion of the same state.
func TestPBTAccountDeletion(t *testing.T) {
	var (
		doomed   = common.Address{1}
		survivor = common.Address{2}
	)
	// Reference: only the survivor ever exists.
	ref, _ := newPBTState(t)
	ref.CreateAccount(survivor)
	ref.SetNonce(survivor, 3, tracing.NonceChangeUnspecified)
	ref.SetState(survivor, common.Hash{31: 5}, common.Hash{31: 1})
	ref.SetState(survivor, common.Hash{30: 4}, common.Hash{31: 2})
	refRoot, err := ref.Commit(1, true, false)
	if err != nil {
		t.Fatal(err)
	}

	// The doomed account holds storage in both homes, then is deleted.
	sdb, db := newPBTState(t)
	for _, addr := range []common.Address{doomed, survivor} {
		sdb.CreateAccount(addr)
		sdb.SetNonce(addr, 3, tracing.NonceChangeUnspecified)
		sdb.SetState(addr, common.Hash{31: 5}, common.Hash{31: 1})
		sdb.SetState(addr, common.Hash{30: 4}, common.Hash{31: 2})
	}
	sdb = reopenPBT(t, sdb, db, 1)
	if !sdb.HasStorage(doomed) {
		t.Fatal("setup: doomed account has no storage")
	}

	// Delete through the trie directly: post-EIP-6780 statedb only destroys
	// same-transaction creations, but the tree operation must be correct.
	tr, err := sdb.db.OpenTrie(sdb.originalRoot)
	if err != nil {
		t.Fatal(err)
	}
	bt := tr.(*bintrie.BinaryTrie)
	for _, addr := range []common.Address{doomed, survivor} {
		if err := bt.UpdateAccount(addr, &types.StateAccount{
			Nonce: 3, Balance: uint256.NewInt(0), CodeHash: types.EmptyCodeHash[:],
		}, 0); err != nil {
			t.Fatal(err)
		}
		bt.UpdateStorage(addr, common.Hash{31: 5}.Bytes(), common.Hash{31: 1}.Bytes())
		bt.UpdateStorage(addr, common.Hash{30: 4}.Bytes(), common.Hash{31: 2}.Bytes())
	}
	if err := bt.DeleteAccount(doomed); err != nil {
		t.Fatal(err)
	}
	if got := bt.Hash(); got != refRoot {
		t.Fatalf("deletion left a trace: %x want %x", got, refRoot)
	}
}

// TestPBTCodeShrink pins EIP-7702 delegation clearing: code can shrink on a
// live account, and the stale header chunk leaves of the longer code must
// disappear, otherwise a replayed tree diverges from a converted one.
func TestPBTCodeShrink(t *testing.T) {
	addr := common.Address{1}
	delegation := append([]byte{0xef, 0x01, 0x00}, common.Address{9}.Bytes()...)

	// Reference: the account never carried code.
	ref, _ := newPBTState(t)
	ref.CreateAccount(addr)
	ref.SetNonce(addr, 2, tracing.NonceChangeUnspecified)
	refRoot, err := ref.Commit(1, true, false)
	if err != nil {
		t.Fatal(err)
	}

	// Same account: set a delegation designator, then clear it.
	sdb, db := newPBTState(t)
	sdb.CreateAccount(addr)
	sdb.SetNonce(addr, 2, tracing.NonceChangeUnspecified)
	sdb.SetCode(addr, delegation, tracing.CodeChangeUnspecified)
	sdb = reopenPBT(t, sdb, db, 1)
	sdb.SetCode(addr, nil, tracing.CodeChangeAuthorizationClear)
	got, err := sdb.Commit(2, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if got != refRoot {
		t.Fatalf("cleared delegation left chunk leaves: %x want %x", got, refRoot)
	}
}

// TestPBTCodeSizePreserved pins that touching a contract without executing
// it preserves the code size the binary tree packs into basic data. The
// code is loaded lazily, so the naive len(obj.code) reads zero here and the
// account's basic-data leaf would silently lose its code size.
func TestPBTCodeSizePreserved(t *testing.T) {
	addr := common.Address{1}
	code := bytes.Repeat([]byte{0x60, 0x01}, 40) // 80 bytes, spanning chunks

	sdb, db := newPBTState(t)
	sdb.CreateAccount(addr)
	sdb.SetNonce(addr, 1, tracing.NonceChangeUnspecified)
	sdb.SetCode(addr, code, tracing.CodeChangeUnspecified)
	sdb = reopenPBT(t, sdb, db, 1)

	// Touch the account with a plain balance change: the code is never read.
	sdb.AddBalance(addr, uint256.NewInt(1), tracing.BalanceChangeUnspecified)
	sdb = reopenPBT(t, sdb, db, 2)

	tr, err := sdb.db.OpenTrie(sdb.originalRoot)
	if err != nil {
		t.Fatal(err)
	}
	bt := tr.(*bintrie.BinaryTrie)
	basic, err := bt.GetStemValue(bintrie.BasicDataKey(addr))
	if err != nil {
		t.Fatal(err)
	}
	if basic == nil {
		t.Fatal("basic data leaf missing")
	}
	_, codeSize, _, _ := bintrie.DecodeBasicData(basic)
	if codeSize != uint32(len(code)) {
		t.Fatalf("code size %d after a balance-only touch, want %d", codeSize, len(code))
	}
}

// TestPBTPrefetcherWarmsOwners exercises the prefetcher in binary tree mode,
// where a single sub-fetcher serves every account: slot tasks must be
// prefetched under their own owner, and the warm trie must be adopted
// before any writes reach it. Getting either wrong is invisible to the
// state root - the reads simply go to disk - so this asserts the resulting
// root instead, with the prefetcher on and off.
func TestPBTPrefetcherWarmsOwners(t *testing.T) {
	build := func(prefetch bool) common.Hash {
		disk := rawdb.NewMemoryDatabase()
		db := triedb.NewDatabase(disk, triedb.PBTDefaults)
		sdb, err := New(types.EmptyBinaryHash, NewDatabase(db, nil))
		if err != nil {
			t.Fatal(err)
		}
		// Seed several accounts, each with storage in both homes.
		var addrs []common.Address
		for i := byte(1); i <= 6; i++ {
			addr := common.Address{i}
			addrs = append(addrs, addr)
			sdb.CreateAccount(addr)
			sdb.SetNonce(addr, uint64(i), tracing.NonceChangeUnspecified)
			sdb.SetState(addr, common.Hash{31: 5}, common.Hash{31: i})
			sdb.SetState(addr, common.Hash{30: 4}, common.Hash{31: i})
		}
		root, err := sdb.Commit(1, true, false)
		if err != nil {
			t.Fatal(err)
		}
		if err := db.Commit(root, false); err != nil {
			t.Fatal(err)
		}

		// Second block: touch every account's storage, optionally warming
		// the trie through the prefetcher first.
		sdb, err = New(root, NewDatabase(db, nil))
		if err != nil {
			t.Fatal(err)
		}
		if prefetch {
			sdb.StartPrefetcher("test", nil)
			defer sdb.StopPrefetcher()
		}
		for i, addr := range addrs {
			sdb.SetState(addr, common.Hash{31: 5}, common.Hash{31: byte(i + 100)})
			sdb.SetState(addr, common.Hash{30: 4}, common.Hash{31: byte(i + 100)})
		}
		next, err := sdb.Commit(2, true, false)
		if err != nil {
			t.Fatal(err)
		}
		return next
	}
	if warm, cold := build(true), build(false); warm != cold {
		t.Fatalf("prefetched root %x differs from cold root %x", warm, cold)
	}
}

// TestPBTProofsVerify pins that proofs taken from a committed binary tree
// verify against its root: inclusion for present account and storage leaves
// in both storage homes, absence for a missing account and an unwritten
// slot, and rejection once a proof node is tampered with. This is the
// machinery behind eth_getProof in binary tree mode.
func TestPBTProofsVerify(t *testing.T) {
	addr := common.Address{1}
	headerSlot := common.Hash{31: 5} // below 64: in the header stem
	bucketSlot := common.Hash{30: 4} // slot 1024: in the storage bucket
	unwritten := common.Hash{31: 63} // never written

	sdb, db := newPBTState(t)
	for i := byte(1); i <= 5; i++ {
		other := common.Address{i}
		sdb.CreateAccount(other)
		sdb.SetNonce(other, uint64(i), tracing.NonceChangeUnspecified)
	}
	sdb.SetState(addr, headerSlot, common.Hash{31: 7})
	sdb.SetState(addr, bucketSlot, common.Hash{31: 8})
	root, err := sdb.Commit(1, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Commit(root, false); err != nil {
		t.Fatal(err)
	}
	sdb, err = New(root, NewDatabase(db, nil))
	if err != nil {
		t.Fatal(err)
	}
	tr, err := sdb.db.OpenTrie(root)
	if err != nil {
		t.Fatal(err)
	}
	bt := tr.(*bintrie.BinaryTrie)

	present := [][]byte{
		bintrie.BasicDataKey(addr),
		bintrie.CodeHashKey(addr),
		bintrie.StorageSlotKey(addr, headerSlot.Bytes()),
		bintrie.StorageSlotKey(addr, bucketSlot.Bytes()),
	}
	for _, key := range present {
		proof := rawdb.NewMemoryDatabase()
		if err := bt.Prove(key, proof); err != nil {
			t.Fatal(err)
		}
		val, err := bintrie.VerifyProof(root, key, proof)
		if err != nil {
			t.Fatalf("proof for %x failed: %v", key, err)
		}
		if val == nil {
			t.Fatalf("proof for %x proved absence of a present leaf", key)
		}
	}
	absent := [][]byte{
		bintrie.BasicDataKey(common.Address{0xEE}),
		bintrie.StorageSlotKey(addr, unwritten.Bytes()),
	}
	for _, key := range absent {
		proof := rawdb.NewMemoryDatabase()
		if err := bt.Prove(key, proof); err != nil {
			t.Fatal(err)
		}
		val, err := bintrie.VerifyProof(root, key, proof)
		if err != nil {
			t.Fatalf("absence proof for %x errored: %v", key, err)
		}
		if val != nil {
			t.Fatalf("absence proof for %x returned a value: %x", key, val)
		}
	}
	// A tampered proof must not verify.
	key := bintrie.BasicDataKey(addr)
	proof := rawdb.NewMemoryDatabase()
	if err := bt.Prove(key, proof); err != nil {
		t.Fatal(err)
	}
	it := proof.NewIterator(nil, nil)
	defer it.Release()
	if !it.Next() {
		t.Fatal("empty proof")
	}
	corrupted := append([]byte{}, it.Value()...)
	corrupted[len(corrupted)-1] ^= 0xff
	if err := proof.Put(it.Key(), corrupted); err != nil {
		t.Fatal(err)
	}
	if _, err := bintrie.VerifyProof(root, key, proof); err == nil {
		t.Fatal("tampered proof verified")
	}
}

// TestPBTSharedOverflowChunks pins the content-addressed code zone: two
// contracts deployed with identical bytecode longer than the 128 header
// chunks share their overflow chunk leaves, so the second deployment adds
// only its own header stem. This is the dedup EIP-8297 exists to provide,
// and it is invisible to the account-level API.
func TestPBTSharedOverflowChunks(t *testing.T) {
	// 130 chunks of code: 128 land in the header stem, 2 overflow into the
	// content-addressed code zone.
	code := bytes.Repeat([]byte{0x5b}, 130*31) // JUMPDEST repeated: no PUSH spans
	var (
		first  = common.Address{1}
		second = common.Address{2}
	)
	countLeaves := func(sdb *StateDB, db *triedb.Database, root common.Hash) int {
		tr, err := sdb.db.OpenTrie(root)
		if err != nil {
			t.Fatal(err)
		}
		it, err := tr.(*bintrie.BinaryTrie).NodeIterator(nil)
		if err != nil {
			t.Fatal(err)
		}
		n := 0
		for it.Next(true) {
			if it.Leaf() {
				n++
			}
		}
		if err := it.Error(); err != nil {
			t.Fatal(err)
		}
		return n
	}

	// One contract.
	one, onedb := newPBTState(t)
	one.CreateAccount(first)
	one.SetNonce(first, 1, tracing.NonceChangeUnspecified)
	one.SetCode(first, code, tracing.CodeChangeUnspecified)
	one = reopenPBT(t, one, onedb, 1)
	leavesOne := countLeaves(one, onedb, one.originalRoot)

	// The same code deployed twice.
	two, twodb := newPBTState(t)
	for _, addr := range []common.Address{first, second} {
		two.CreateAccount(addr)
		two.SetNonce(addr, 1, tracing.NonceChangeUnspecified)
		two.SetCode(addr, code, tracing.CodeChangeUnspecified)
	}
	two = reopenPBT(t, two, twodb, 1)
	leavesTwo := countLeaves(two, twodb, two.originalRoot)

	// The second contract contributes its own header stem - basic data, code
	// hash and its 128 header chunks - and nothing else: the 2 overflow
	// chunks are shared.
	perAccount := 2 + 128
	if got, want := leavesTwo-leavesOne, perAccount; got != want {
		t.Fatalf("second identical contract added %d leaves, want %d (overflow chunks not shared)", got, want)
	}
	// Sanity: the overflow chunks really exist, and both accounts derive the
	// same key for them.
	codeHash := crypto.Keccak256Hash(code)
	shared := bintrie.CodeChunkKey(first, codeHash, 128)
	if other := bintrie.CodeChunkKey(second, codeHash, 128); !bytes.Equal(shared, other) {
		t.Fatal("overflow chunk keys differ between contracts with identical code")
	}
	if shared[0] != bintrie.CodeZone {
		t.Fatalf("overflow chunk is not in the code zone: zone byte %#x", shared[0])
	}
}
