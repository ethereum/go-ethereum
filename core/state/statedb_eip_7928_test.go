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
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
)

// TestApplyBlockAccessListConcurrentPrefetch stresses the concurrent storage
// prefetch scheduling with many contracts (each carrying a non-empty storage
// trie), so that several BAL workers schedule prefetches into the shared,
// non-thread-safe prefetcher at once. Run with -race to detect unguarded
// concurrent access. The resulting root must still match the sequential one.
func TestApplyBlockAccessListConcurrentPrefetch(t *testing.T) {
	const n = 64
	code := []byte{0x60, 0x00, 0x60, 0x00}
	addrOf := func(i int) common.Address {
		return common.BigToAddress(uint256.NewInt(uint64(0x1000 + i)).ToBig())
	}
	slot := common.HexToHash("0x01")

	// Base state: n contracts, each with a pre-existing storage slot so its
	// storage root is non-empty.
	db := NewDatabaseForTesting()
	base, _ := New(types.EmptyRootHash, db)
	for i := range n {
		addr := addrOf(i)
		base.SetBalance(addr, uint256.NewInt(100), tracing.BalanceChangeUnspecified)
		base.SetCode(addr, code, tracing.CodeChangeUnspecified)
		base.SetState(addr, slot, common.HexToHash("0xaa"))
	}
	root0, err := base.Commit(params.Rules{}, 0)
	if err != nil {
		t.Fatalf("commit base: %v", err)
	}

	mutate := func(s *StateDB) {
		for i := range n {
			addr := addrOf(i)
			s.SetBalance(addr, uint256.NewInt(uint64(200+i)), tracing.BalanceChangeUnspecified)
			s.SetState(addr, slot, common.BigToHash(uint256.NewInt(uint64(i+1)).ToBig()))
		}
	}
	seq, _ := New(root0, db)
	mutate(seq)
	wantRoot := seq.IntermediateRoot(params.Rules{IsEIP158: true})

	cb := bal.NewConstructionBlockAccessList()
	for i := range n {
		addr := addrOf(i)
		cb.BalanceChange(0, addr, uint256.NewInt(uint64(200+i)))
		cb.StorageWrite(0, addr, slot, common.BigToHash(uint256.NewInt(uint64(i+1)).ToBig()))
	}
	balState, _ := New(root0, db)
	balState.StartPrefetcher("test", nil)
	if err := balState.ApplyBlockAccessList(cb.ToEncodingObj()); err != nil {
		balState.StopPrefetcher()
		t.Fatalf("apply block access list: %v", err)
	}
	gotRoot := balState.IntermediateRoot(params.Rules{IsEIP158: true})
	balState.StopPrefetcher()

	if gotRoot != wantRoot {
		t.Fatalf("BAL apply root = %x, want %x", gotRoot, wantRoot)
	}
}

// TestApplyBlockAccessListMatchesSequential checks that installing a block's
// post-state through ApplyBlockAccessList yields exactly the same state root as
// applying the same mutations one by one. The BAL path warms both the account
// trie and the storage tries through the prefetcher and pulls them back at
// IntermediateRoot, so this also exercises that machinery. Run with -race to
// catch data races in the concurrent prefetch scheduling.
func TestApplyBlockAccessListMatchesSequential(t *testing.T) {
	var (
		existing = common.HexToAddress("0x1111")
		contract = common.HexToAddress("0x2222")
		fresh    = common.HexToAddress("0x3333")

		slotA = common.HexToHash("0x01")
		slotB = common.HexToHash("0x02")
		code  = []byte{0x60, 0x00, 0x60, 0x00}
	)

	// Build a base state with a plain account and a contract that already has
	// some storage (so its storage root is non-empty and the storage-trie
	// prefetch path is exercised).
	db := NewDatabaseForTesting()
	base, _ := New(types.EmptyRootHash, db)
	base.SetBalance(existing, uint256.NewInt(1000), tracing.BalanceChangeUnspecified)
	base.SetNonce(existing, 1, tracing.NonceChangeUnspecified)
	base.SetBalance(contract, uint256.NewInt(50), tracing.BalanceChangeUnspecified)
	base.SetCode(contract, code, tracing.CodeChangeUnspecified)
	base.SetState(contract, slotA, common.HexToHash("0xaa"))
	base.SetState(contract, slotB, common.HexToHash("0xbb"))
	root0, err := base.Commit(params.Rules{}, 0)
	if err != nil {
		t.Fatalf("commit base: %v", err)
	}

	// mutate applies the block's post-state via the ordinary setters.
	mutate := func(s *StateDB) {
		s.SetBalance(existing, uint256.NewInt(1234), tracing.BalanceChangeUnspecified)
		s.SetNonce(existing, 2, tracing.NonceChangeUnspecified)
		s.SetState(contract, slotA, common.HexToHash("0xcc")) // changed
		s.SetState(contract, slotB, common.HexToHash("0xbb")) // unchanged, must be a no-op
		s.SetBalance(fresh, uint256.NewInt(7), tracing.BalanceChangeUnspecified)
		s.SetNonce(fresh, 1, tracing.NonceChangeUnspecified)
	}

	// Sequential reference root.
	seq, _ := New(root0, db)
	mutate(seq)
	wantRoot := seq.IntermediateRoot(params.Rules{IsEIP158: true})
	if wantRoot == root0 {
		t.Fatal("mutations did not change the state root")
	}

	// Same post-state expressed as a block access list.
	cb := bal.NewConstructionBlockAccessList()
	cb.BalanceChange(0, existing, uint256.NewInt(1234))
	cb.NonceChange(existing, 0, 2)
	cb.StorageWrite(0, contract, slotA, common.HexToHash("0xcc"))
	cb.StorageWrite(0, contract, slotB, common.HexToHash("0xbb")) // no-op write
	cb.BalanceChange(0, fresh, uint256.NewInt(7))
	cb.NonceChange(fresh, 0, 1)
	list := cb.ToEncodingObj()

	balState, _ := New(root0, db)
	balState.StartPrefetcher("test", nil)
	if err := balState.ApplyBlockAccessList(list); err != nil {
		balState.StopPrefetcher()
		t.Fatalf("apply block access list: %v", err)
	}
	gotRoot := balState.IntermediateRoot(params.Rules{IsEIP158: true})
	balState.StopPrefetcher()

	if gotRoot != wantRoot {
		t.Fatalf("BAL apply root = %x, want %x", gotRoot, wantRoot)
	}
}

// TestAccessListRecordsAdoptedCode checks that code set through AdoptCode is
// recorded in the block access list by hash, that a reverted adoption leaves a
// plain code change, and that ApplyBlockAccessList resolves adopted code from
// the pre-block state and from code deployed earlier in the block.
func TestAccessListRecordsAdoptedCode(t *testing.T) {
	var (
		source   = common.HexToAddress("0x1111")
		adopter  = common.HexToAddress("0x2222")
		deployer = common.HexToAddress("0x3333")
		late     = common.HexToAddress("0x4444")
		reverted = common.HexToAddress("0x5555")

		preCode  = []byte{0x60, 0x01, 0x00}
		newCode  = []byte{0x60, 0x02, 0x00}
		preHash  = crypto.Keccak256Hash(preCode)
		newHash  = crypto.Keccak256Hash(newCode)
		rules    = params.Rules{IsAmsterdam: true, IsEIP158: true}
		recorded = bal.NewConstructionBlockAccessList()
	)
	db := NewDatabaseForTesting()
	base, _ := New(types.EmptyRootHash, db)
	base.SetCode(source, preCode, tracing.CodeChangeUnspecified)
	base.SetNonce(source, 1, tracing.NonceChangeUnspecified)
	base.SetNonce(adopter, 1, tracing.NonceChangeUnspecified)
	base.SetNonce(late, 1, tracing.NonceChangeUnspecified)
	base.SetNonce(reverted, 1, tracing.NonceChangeUnspecified)
	root0, err := base.Commit(params.Rules{}, 0)
	if err != nil {
		t.Fatalf("commit base: %v", err)
	}
	seq, _ := New(root0, db)

	// Transaction 1: adopt pre-block code, deploy new code, and adopt code
	// whose adoption is reverted before a plain code write.
	seq.SetTxContext(common.Hash{}, 0, 1)
	seq.Prepare(rules, common.Address{}, common.Address{}, nil, nil, nil)
	seq.AdoptCode(adopter, preHash, preCode)
	seq.SetCode(deployer, newCode, tracing.CodeChangeContractCreation)
	snap := seq.Snapshot()
	seq.AdoptCode(reverted, preHash, preCode)
	seq.RevertToSnapshot(snap)
	seq.SetCode(reverted, newCode, tracing.CodeChangeUnspecified)
	recorded.Merge(seq.Finalise(rules))

	// Transaction 2: adopt the code deployed by transaction 1.
	seq.SetTxContext(common.Hash{}, 1, 2)
	seq.Prepare(rules, common.Address{}, common.Address{}, nil, nil, nil)
	seq.AdoptCode(late, newHash, newCode)
	recorded.Merge(seq.Finalise(rules))
	wantRoot := seq.IntermediateRoot(rules)

	list := recorded.ToEncodingObj()
	want := map[common.Address]struct {
		index uint32
		code  []byte
		hash  *common.Hash
	}{
		adopter:  {1, nil, &preHash},
		deployer: {1, newCode, nil},
		reverted: {1, newCode, nil},
		late:     {2, nil, &newHash},
	}
	for _, access := range *list {
		w, ok := want[access.Address]
		if !ok {
			continue
		}
		if len(access.CodeChanges) != 1 {
			t.Fatalf("%x: want 1 code change, got %d", access.Address, len(access.CodeChanges))
		}
		c := access.CodeChanges[0]
		if c.BlockAccessIndex != w.index || !bytes.Equal(c.NewCode, w.code) || (c.NewCodeHash == nil) != (w.hash == nil) || (w.hash != nil && *c.NewCodeHash != *w.hash) {
			t.Fatalf("%x: code change mismatch: have %+v", access.Address, c)
		}
		delete(want, access.Address)
	}
	if len(want) != 0 {
		t.Fatalf("missing code changes: %v", want)
	}
	// The access list must survive an RLP round trip and rebuild the state.
	enc, err := rlp.EncodeToBytes(list)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	var dec bal.BlockAccessList
	if err := rlp.DecodeBytes(enc, &dec); err != nil {
		t.Fatalf("decode: %v", err)
	}
	balState, _ := New(root0, db)
	if err := balState.ApplyBlockAccessList(&dec); err != nil {
		t.Fatalf("apply block access list: %v", err)
	}
	if gotRoot := balState.IntermediateRoot(rules); gotRoot != wantRoot {
		t.Fatalf("BAL apply root = %x, want %x", gotRoot, wantRoot)
	}
	if code := balState.GetCode(adopter); !bytes.Equal(code, preCode) {
		t.Fatalf("adopter code = %x, want %x", code, preCode)
	}
	if code := balState.GetCode(late); !bytes.Equal(code, newCode) {
		t.Fatalf("late adopter code = %x, want %x", code, newCode)
	}
}
