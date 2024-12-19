// Copyright 2024 The go-ethereum Authors
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

// Package state provides a caching layer atop the Ethereum state trie.
package state

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"slices"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
)

func TestLinearJournalDirty(t *testing.T) {
	testJournalDirty(t, newLinearJournal())
}

func TestSparseJournalDirty(t *testing.T) {
	testJournalDirty(t, newSparseJournal())
}

// This test verifies some basics around journalling: the ability to
// deliver a dirty-set.
func testJournalDirty(t *testing.T, j journal) {
	acc := &types.StateAccount{
		Nonce:    1,
		Balance:  new(uint256.Int),
		Root:     common.Hash{},
		CodeHash: nil,
	}
	{
		j.nonceChange(common.Address{0x1}, acc, false, false)
		if have, want := len(j.dirtyAccounts()), 1; have != want {
			t.Errorf("wrong size of dirty accounts, have %v want %v", have, want)
		}
	}
	{
		j.storageChange(common.Address{0x2}, common.Hash{0x1}, common.Hash{0x1}, common.Hash{})
		if have, want := len(j.dirtyAccounts()), 2; have != want {
			t.Errorf("wrong size of dirty accounts, have %v want %v", have, want)
		}
	}
	{ // The previous scopes should also be accounted for
		j.snapshot()
		if have, want := len(j.dirtyAccounts()), 2; have != want {
			t.Errorf("wrong size of dirty accounts, have %v want %v", have, want)
		}
	}
}

func TestLinearJournalAccessList(t *testing.T) {
	testJournalAccessList(t, newLinearJournal())
}

func TestSparseJournalAccessList(t *testing.T) {
	testJournalAccessList(t, newSparseJournal())
}

func testJournalAccessList(t *testing.T, j journal) {
	var statedb = &StateDB{}
	statedb.accessList = newAccessList()
	statedb.journal = j

	j.snapshot()
	{
		// If the journal performs the rollback in the wrong order, this
		// will cause a panic.
		statedb.AddSlotToAccessList(common.Address{0x1}, common.Hash{0x4})
		statedb.AddSlotToAccessList(common.Address{0x3}, common.Hash{0x4})
	}
	statedb.RevertSnapshot()
	j.snapshot()
	{
		statedb.AddAddressToAccessList(common.Address{0x2})
		statedb.AddAddressToAccessList(common.Address{0x3})
		statedb.AddAddressToAccessList(common.Address{0x4})
	}
	statedb.RevertSnapshot()
	if statedb.accessList.ContainsAddress(common.Address{0x2}) {
		t.Fatal("should be missing")
	}
}

func TestLinearJournalRefunds(t *testing.T) {
	testJournalRefunds(t, newLinearJournal())
}

func TestSparseJournalRefunds(t *testing.T) {
	testJournalRefunds(t, newSparseJournal())
}

func testJournalRefunds(t *testing.T, j journal) {
	var statedb = &StateDB{}
	statedb.accessList = newAccessList()
	statedb.journal = j
	j.snapshot()
	{
		j.refundChange(0)
		j.refundChange(1)
		j.snapshot()
		{
			j.refundChange(2)
			j.refundChange(3)
		}
		j.revertSnapshot(statedb)
		if have, want := statedb.refund, uint64(2); have != want {
			t.Fatalf("have %d want %d", have, want)
		}
		j.snapshot()
		{
			j.refundChange(2)
			j.refundChange(3)
		}
		j.discardSnapshot()
	}
	j.revertSnapshot(statedb)
	if have, want := statedb.refund, uint64(0); have != want {
		t.Fatalf("have %d want %d", have, want)
	}
}

type fuzzReader struct {
	input     io.Reader
	exhausted bool
}

func (f *fuzzReader) byte() byte {
	return f.bytes(1)[0]
}

func (f *fuzzReader) bytes(n int) []byte {
	r := make([]byte, n)
	if _, err := f.input.Read(r); err != nil {
		f.exhausted = true
	}
	return r
}

func newEmptyState() *StateDB {
	s, _ := New(types.EmptyRootHash, NewDatabaseForTesting())
	return s
}

// fuzzJournals is pretty similar to `TestSnapshotRandom`/ `newTestAction` in
// statedb_test.go. They both execute a sequence of state-actions, however, they
// test for different aspects.
// This test compares two differing journal-implementations.
// The other test compares every point in time, whether it is identical when going
// forward as when going backwards through the journal entries.
func fuzzJournals(t *testing.T, data []byte) {
	var (
		reader   = fuzzReader{input: bytes.NewReader(data)}
		stateDbs = []*StateDB{
			newEmptyState(),
			newEmptyState(),
		}
	)
	apply := func(action func(stateDbs *StateDB)) {
		for _, sdb := range stateDbs {
			action(sdb)
		}
	}
	stateDbs[0].journal = newLinearJournal()
	stateDbs[1].journal = newSparseJournal()

	for !reader.exhausted {
		op := reader.byte() % 18
		switch op {
		case 0: // Add account to access lists
			addr := common.BytesToAddress(reader.bytes(1))
			t.Logf("Op %d: Add to access list %#x", op, addr)
			apply(func(sdb *StateDB) {
				sdb.accessList.AddAddress(addr)
			})
		case 1: // Add slot to access list
			addr := common.BytesToAddress(reader.bytes(1))
			slot := common.BytesToHash(reader.bytes(1))
			t.Logf("Op %d: Add addr:slot to access list %#x : %#x", op, addr, slot)
			apply(func(sdb *StateDB) {
				sdb.AddSlotToAccessList(addr, slot)
			})
		case 2:
			var (
				addr  = common.BytesToAddress(reader.bytes(1))
				value = uint64(reader.byte())
			)
			t.Logf("Op %d: Add balance %#x %d", op, addr, value)
			apply(func(sdb *StateDB) {
				sdb.AddBalance(addr, uint256.NewInt(value), 0)
			})
		case 3:
			t.Logf("Op %d: Copy journals[0]", op)
			stateDbs[0].journal = stateDbs[0].journal.copy()
		case 4:
			t.Logf("Op %d: Copy journals[1]", op)
			stateDbs[1].journal = stateDbs[1].journal.copy()
		case 5:
			var (
				addr = common.BytesToAddress(reader.bytes(1))
				code = reader.bytes(2)
			)
			t.Logf("Op %d: (Create and) set code 0x%x", op, addr)
			apply(func(s *StateDB) {
				if !s.Exist(addr) {
					s.CreateAccount(addr)
				}
				storageRoot := s.GetStorageRoot(addr)
				emptyStorage := storageRoot == (common.Hash{}) || storageRoot == types.EmptyRootHash

				if obj := s.getStateObject(addr); obj != nil {
					if obj.selfDestructed {
						// If it's selfdestructed, we cannot create into it
						return
					}
				}
				if emptyStorage {
					s.CreateContract(addr)
					// We also set some code here, to prevent the
					// CreateContract action from being performed twice in a row,
					// which would cause a difference in state when unrolling
					// the linearJournal. (CreateContact assumes created was false prior to
					// invocation, and the linearJournal rollback sets it to false).
					s.SetCode(addr, code)
				}
			})
		case 6:
			addr := common.BytesToAddress(reader.bytes(1))
			t.Logf("Op %d: Create 0x%x", op, addr)
			apply(func(sdb *StateDB) {
				if !sdb.Exist(addr) {
					sdb.CreateAccount(addr)
				}
			})
		case 7:
			addr := common.BytesToAddress(reader.bytes(1))
			t.Logf("Op %d: (Create and) destruct 0x%x", op, addr)
			apply(func(s *StateDB) {
				if !s.Exist(addr) {
					s.CreateAccount(addr)
				}
				s.SelfDestruct(addr)
			})
		case 8:
			txHash := common.BytesToHash(reader.bytes(1))
			t.Logf("Op %d: Add log %#x", op, txHash)
			apply(func(sdb *StateDB) {
				sdb.logs[txHash] = append(sdb.logs[txHash], new(types.Log))
				sdb.logSize++
				sdb.journal.logChange(txHash)
			})
		case 9:
			var (
				addr  = common.BytesToAddress(reader.bytes(1))
				nonce = binary.BigEndian.Uint64(reader.bytes(8))
			)
			t.Logf("Op %d: Set nonce %#x %d", op, addr, nonce)
			apply(func(sdb *StateDB) {
				sdb.SetNonce(addr, nonce)
			})
		case 10:
			refund := uint64(reader.byte())
			t.Logf("Op %d: Set refund %d", op, refund)
			apply(func(sdb *StateDB) {
				sdb.journal.refundChange(refund)
			})
		case 11:
			var (
				addr = common.BytesToAddress(reader.bytes(1))
				key  = common.BytesToHash(reader.bytes(1))
				val  = common.BytesToHash(reader.bytes(1))
			)
			t.Logf("Op %d: Set storage %#x [%#x]=%#x", op, addr, key, val)
			apply(func(sdb *StateDB) {
				sdb.SetState(addr, key, val)
			})
		case 12:
			var (
				addr = common.BytesToAddress(reader.bytes(1))
			)
			t.Logf("Op %d: Zero-balance transfer (touch) %#x", op, addr)
			apply(func(sdb *StateDB) {
				sdb.AddBalance(addr, uint256.NewInt(0), 0)
			})
		case 13:
			var (
				addr  = common.BytesToAddress(reader.bytes(1))
				key   = common.BytesToHash(reader.bytes(1))
				value = common.BytesToHash(reader.bytes(1))
			)
			t.Logf("Op %d: Set t-storage %#x [%#x]=%#x", op, addr, key, value)
			apply(func(sdb *StateDB) {
				sdb.SetTransientState(addr, key, value)
			})
		case 14:
			t.Logf("Op %d: Reset journal", op)
			apply(func(sdb *StateDB) {
				sdb.journal.reset()
			})
		case 15:
			t.Logf("Op %d: Snapshot", op)
			apply(func(sdb *StateDB) {
				sdb.Snapshot()
			})
		case 16:
			t.Logf("Op %d: Discard snapshot", op)
			apply(func(sdb *StateDB) {
				sdb.DiscardSnapshot()
			})

		case 17:
			t.Logf("Op %d: Revert snapshot", op)
			apply(func(sdb *StateDB) {
				sdb.RevertSnapshot()
			})
		}
		// Cross-check the dirty-sets
		accs1 := stateDbs[0].journal.dirtyAccounts()
		slices.SortFunc(accs1, func(a, b common.Address) int {
			return bytes.Compare(a.Bytes(), b.Bytes())
		})
		accs2 := stateDbs[1].journal.dirtyAccounts()
		slices.SortFunc(accs2, func(a, b common.Address) int {
			return bytes.Compare(a.Bytes(), b.Bytes())
		})
		if !slices.Equal(accs1, accs2) {
			t.Fatalf("mismatched dirty-sets:\n%v\n%v", accs1, accs2)
		}

		for _, addr := range accs1 {
			if cHash := stateDbs[0].GetCodeHash(addr); cHash != types.EmptyCodeHash && cHash != (common.Hash{}) {
				have := crypto.Keccak256Hash(stateDbs[0].GetCode(addr))
				if have != cHash {
					t.Fatalf("0: mismatched codehash <-> code.\ncodehash:   %x\nhash(code): %x\n", cHash, have)
				}
				have = crypto.Keccak256Hash(stateDbs[1].GetCode(addr))
				if have != cHash {
					t.Fatalf("1: mismatched codehash <-> code.\ncodehash:   %x\nhash(code): %x\n", cHash, have)
				}
			}
		}
	}
	h1, err1 := stateDbs[0].Commit(0, false)
	h2, err2 := stateDbs[1].Commit(0, false)
	if err1 != err2 {
		t.Fatalf("Mismatched errors: %v %v", err1, err2)
	}
	if h1 != h2 {
		t.Fatalf("Mismatched roots: %v %v", h1, h2)
	}
}

// FuzzJournals fuzzes the journals.
func FuzzJournals(f *testing.F) {
	f.Fuzz(fuzzJournals)
}

// TestFuzzJournals runs 200 fuzz-tests
func TestFuzzJournals(t *testing.T) {
	input := make([]byte, 200)
	for i := 0; i < 200; i++ {
		rand.Read(input)
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Parallel()
			t.Logf("input: %x", input)
			fuzzJournals(t, input)
		})
	}
}

// TestFuzzJournalsSpecific can be used to test a specific input
func TestFuzzJournalsSpecific(t *testing.T) {
	t.Skip("example")
	input := common.FromHex("71d598d781f65eb7c047fed5d09b1e4e0c1ecad5c447a2149e7d1137fcb1b1d63f4ba6f761918a441a98eb61d69fe011cabfbce00d74bb78539ca9946a602e94d6eabc43c0924ba65ce3e171b476208059d81f33e81d90607e0b6e59d6016840b5c4e9b1a8e9798a5a40be909930658eea351d7a312dba0b1c7199c7e5f62a908a80f7faf29bc0108faae0cf0f497d0f4cd228b7600ef0d88532dfafa6349ea7782f28ad7426eeffc155282a9e58a606d25acd8a730dde61a6e5e887d1ba1fea813bb7f2c6caff25")
	fuzzJournals(t, input)
}
