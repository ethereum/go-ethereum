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
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/triedb"
)

var (
	adopter8298 = common.HexToAddress("0xaa")
	source8298  = common.HexToAddress("0xbb")
	other8298   = common.HexToAddress("0xcc")
	rules8298   = params.Rules{IsBerlin: true, IsEIP158: true, IsAmsterdam: true}
)

// TestSetCodeFrom checks that adopting code (EIP-8298) neither rewrites code
// that is already stored nor survives a revert, and that the BAL records it
// as a code hash.
func TestSetCodeFrom(t *testing.T) {
	var (
		old  = []byte{0x60, 0x01}
		code = []byte{0x60, 0x02}
		hash = crypto.Keccak256Hash(code)
	)
	db := NewDatabaseForTesting()
	base, _ := New(types.EmptyRootHash, db)
	base.SetCode(adopter8298, old, tracing.CodeChangeUnspecified)
	base.SetCode(source8298, code, tracing.CodeChangeUnspecified)
	root, err := base.Commit(params.Rules{}, 0)
	if err != nil {
		t.Fatal(err)
	}
	state, _ := New(root, db)
	state.Prepare(rules8298, common.Address{}, common.Address{}, nil, nil, nil)
	state.SetTxContext(common.Hash{}, 0, 1)

	snap := state.Snapshot()
	if prev := state.SetCodeFrom(adopter8298, source8298); !bytes.Equal(prev, old) {
		t.Fatalf("previous code %x, want %x", prev, old)
	}
	obj := state.getStateObject(adopter8298)
	if state.GetCodeHash(adopter8298) != hash || !bytes.Equal(state.GetCode(adopter8298), code) {
		t.Fatal("code not adopted")
	}
	if obj.dirtyCode || !obj.codeAdopted {
		t.Fatalf("stored source: dirty %v, adopted %v, want false, true", obj.dirtyCode, obj.codeAdopted)
	}
	state.RevertToSnapshot(snap)
	if state.GetCodeHash(adopter8298) != crypto.Keccak256Hash(old) || obj.dirtyCode || obj.codeAdopted {
		t.Fatal("revert did not restore the previous code exactly")
	}

	state.SetCodeFrom(adopter8298, source8298)
	acc := state.Finalise(rules8298).Accounts[adopter8298]
	if acc == nil {
		t.Fatal("adopter missing from the access list")
	}
	if got, ok := acc.CodeHashChange[1]; !ok || got != hash {
		t.Fatalf("BAL code hash change: got %x (%v), want %x", got, ok, hash)
	}
	if _, ok := acc.CodeChange[1]; ok {
		t.Fatal("adopted code recorded as bytecode")
	}
}

// TestSetCodeFromPendingSource checks that code adopted from a source created
// earlier in the block is still stored after the source replaced its own code
// before the block was committed.
func TestSetCodeFromPendingSource(t *testing.T) {
	var (
		created = []byte{0x60, 0x03}
		stored  = []byte{0x60, 0x04}
		disk    = rawdb.NewMemoryDatabase()
		tdb     = triedb.NewDatabase(disk, nil)
		db      = NewDatabase(tdb, NewCodeDB(disk))
	)
	base, _ := New(types.EmptyRootHash, db)
	base.SetCode(other8298, stored, tracing.CodeChangeUnspecified)
	root, err := base.Commit(params.Rules{}, 0)
	if err != nil {
		t.Fatal(err)
	}
	state, _ := New(root, db)
	state.SetCode(source8298, created, tracing.CodeChangeContractCreation)
	state.Finalise(params.Rules{})
	state.SetCodeFrom(adopter8298, source8298) // pending: the adopter writes it too
	state.Finalise(params.Rules{})
	state.SetCodeFrom(source8298, other8298) // stored: nothing to write
	if state.getStateObject(source8298).dirtyCode {
		t.Fatal("adopting stored code marked it for writing")
	}
	root, err = state.Commit(params.Rules{}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := tdb.Commit(root, false); err != nil {
		t.Fatal(err)
	}
	fresh, _ := New(root, NewDatabase(triedb.NewDatabase(disk, nil), NewCodeDB(disk)))
	if got := fresh.GetCode(adopter8298); !bytes.Equal(got, created) {
		t.Fatalf("adopter code %x, want %x", got, created)
	}
	if err := fresh.Error(); err != nil {
		t.Fatal(err)
	}
}

// TestCodeBatchDedup checks that a code held by several accounts is queued once.
func TestCodeBatchDedup(t *testing.T) {
	code := []byte{0x60, 0x05}
	b := newCodeBatch(NewCodeDB(rawdb.NewMemoryDatabase()))
	b.Put(crypto.Keccak256Hash(code), code)
	b.Put(crypto.Keccak256Hash(code), code)
	if len(b.codes) != 1 {
		t.Fatalf("queued %d codes, want 1", len(b.codes))
	}
	if err := b.Commit(); err != nil {
		t.Fatal(err)
	}
	b.Put(crypto.Keccak256Hash(code), code)
	if len(b.codes) != 1 {
		t.Fatal("committed batch kept the code queued")
	}
}

// TestHookedSetCodeFrom checks the code change event of an adoption: it fires
// once, with the adopted code and the SETCODEFROM reason, and not for an
// adoption that changes nothing.
func TestHookedSetCodeFrom(t *testing.T) {
	code := []byte{0x60, 0x06}
	inner, _ := New(types.EmptyRootHash, NewDatabaseForTesting())
	inner.SetCode(source8298, code, tracing.CodeChangeUnspecified)

	type event struct {
		prevHash, codeHash common.Hash
		prev, code         []byte
		reason             tracing.CodeChangeReason
	}
	var events []event
	hooked := NewHookedState(inner, &tracing.Hooks{
		OnCodeChangeV2: func(_ common.Address, prevHash common.Hash, prev []byte, codeHash common.Hash, code []byte, reason tracing.CodeChangeReason) {
			events = append(events, event{prevHash, codeHash, prev, code, reason})
		},
	})
	hooked.SetCodeFrom(adopter8298, source8298)
	hooked.SetCodeFrom(adopter8298, source8298)       // same code
	hooked.SetCodeFrom(adopter8298, common.Address{}) // missing source
	want := []event{{types.EmptyCodeHash, crypto.Keccak256Hash(code), nil, code, tracing.CodeChangeSetCodeFrom}}
	if !reflect.DeepEqual(events, want) {
		t.Fatalf("events %+v, want %+v", events, want)
	}
}

// TestApplyBlockAccessListAdoptedCode checks that installing a BAL resolves an
// adopted code hash from the code the list deposits, even if its depositor
// replaced that code later in the block, or from the stored code. The result
// matches the sequential state root, and the adopted code is readable once the
// state is committed and reopened with cold caches.
func TestApplyBlockAccessListAdoptedCode(t *testing.T) {
	var (
		created = []byte{0x60, 0x07}
		stored  = []byte{0x60, 0x08}
		disk    = rawdb.NewMemoryDatabase()
		tdb     = triedb.NewDatabase(disk, nil)
		db      = NewDatabase(tdb, NewCodeDB(disk))
	)
	base, _ := New(types.EmptyRootHash, db)
	base.SetCode(other8298, stored, tracing.CodeChangeUnspecified)
	root, err := base.Commit(params.Rules{}, 0)
	if err != nil {
		t.Fatal(err)
	}
	fromList, fromStore := common.HexToAddress("0xdd"), common.HexToAddress("0xee")

	seq, _ := New(root, db)
	seq.SetCode(source8298, created, tracing.CodeChangeContractCreation)
	seq.SetNonce(source8298, 1, tracing.NonceChangeUnspecified)
	seq.SetNonce(fromList, 1, tracing.NonceChangeUnspecified)
	seq.SetCodeFrom(fromList, source8298)
	seq.SetNonce(fromStore, 1, tracing.NonceChangeUnspecified)
	seq.SetCodeFrom(fromStore, other8298)
	seq.SetCodeFrom(source8298, other8298) // the depositor replaces its code
	want := seq.IntermediateRoot(params.Rules{IsEIP158: true})

	cb := bal.NewConstructionBlockAccessList()
	cb.CodeChange(source8298, 1, created)
	cb.NonceChange(source8298, 1, 1)
	cb.CodeHashChange(fromList, 2, crypto.Keccak256Hash(created))
	cb.NonceChange(fromList, 2, 1)
	cb.CodeHashChange(fromStore, 2, crypto.Keccak256Hash(stored))
	cb.NonceChange(fromStore, 2, 1)
	cb.CodeHashChange(source8298, 3, crypto.Keccak256Hash(stored))

	applied, _ := New(root, db)
	if err := applied.ApplyBlockAccessList(cb.ToEncodingObj()); err != nil {
		t.Fatal(err)
	}
	if got := applied.IntermediateRoot(params.Rules{IsEIP158: true}); got != want {
		t.Fatalf("root %x, want %x", got, want)
	}
	if root, err = applied.Commit(params.Rules{IsEIP158: true}, 1); err != nil {
		t.Fatal(err)
	}
	if err := tdb.Commit(root, false); err != nil {
		t.Fatal(err)
	}
	fresh, _ := New(root, NewDatabase(triedb.NewDatabase(disk, nil), NewCodeDB(disk)))
	for addr, code := range map[common.Address][]byte{fromList: created, fromStore: stored, source8298: stored} {
		if got := fresh.GetCode(addr); !bytes.Equal(got, code) {
			t.Errorf("code of %x: %x, want %x", addr, got, code)
		}
	}
	if err := fresh.Error(); err != nil {
		t.Fatal(err)
	}
	// An adopted hash that is neither deposited nor stored is rejected.
	missing := bal.NewConstructionBlockAccessList()
	missing.CodeHashChange(fromList, 1, common.HexToHash("0x01"))
	unresolved, _ := New(root, db)
	if err := unresolved.ApplyBlockAccessList(missing.ToEncodingObj()); err == nil {
		t.Fatal("unresolvable adopted code hash accepted")
	}
}
