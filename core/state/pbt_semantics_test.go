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

// TestPBTHasStorage covers the EIP-7610 storage-emptiness predicate across
// both homes of an account's storage (header slots below 64 and the
// overflow bucket) and across the committed/uncommitted boundary. Under the
// binary tree there is no per-account storage root, so a regression here is
// invisible to GetStorageRoot and would silently permit CREATE2 collisions.
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
