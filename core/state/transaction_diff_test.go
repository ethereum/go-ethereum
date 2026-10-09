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
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

func TestTransactionDiff(t *testing.T) {
	s, _ := New(types.EmptyRootHash, NewDatabaseForTesting())
	a, b, c := common.HexToAddress("0x01"), common.HexToAddress("0x02"), common.HexToAddress("0x03")
	k1, k2 := common.HexToHash("0x01"), common.HexToHash("0x02")
	s.SetBalance(a, uint256.NewInt(10), tracing.BalanceChangeUnspecified)
	s.SetState(a, k1, k1)
	s.SetState(a, k2, k2)
	s.SetNonce(a, 1, tracing.NonceChangeUnspecified)
	s.Finalise(params.Rules{})
	s.SetTxContext(common.HexToHash("0xff"), 0, 1)
	s.Prepare(params.Rules{IsAmsterdam: true, IsEIP2929: true}, common.Address{}, common.Address{}, nil, nil, nil)
	s.SetBalance(b, uint256.NewInt(30), tracing.BalanceChangeUnspecified)
	s.SetBalance(a, uint256.NewInt(11), tracing.BalanceChangeUnspecified)
	s.SetState(a, k2, k1)
	s.SetState(a, k1, k2)
	s.SetState(a, k1, k1) // Restored slots are omitted.
	s.SetNonce(a, 2, tracing.NonceChangeUnspecified)
	s.SetCode(b, []byte{0x00}, tracing.CodeChangeUnspecified)
	s.SetCode(c, types.AddressToDelegation(a), tracing.CodeChangeUnspecified)
	s.AddLog(&types.Log{Address: b, Topics: []common.Hash{k1, k2, k2}, Data: []byte{1, 2}})
	snapshot := s.Snapshot()
	s.SetBalance(a, uint256.NewInt(50), tracing.BalanceChangeUnspecified)
	s.AddLog(&types.Log{Address: a})
	s.RevertToSnapshot(snapshot)
	s.AddLog(&types.Log{Address: a, Topics: []common.Hash{k2, k1}})
	beforeBAL := s.stateAccessList.ToEncodingObj()
	diff := s.GetTransactionDiff()
	if !reflect.DeepEqual(beforeBAL, s.stateAccessList.ToEncodingObj()) {
		t.Fatal("diff recorded BAL accesses")
	}
	wantBalances := []TransactionBalanceChange{{a, *uint256.NewInt(10), *uint256.NewInt(11)}, {b, uint256.Int{}, *uint256.NewInt(30)}}
	if !reflect.DeepEqual(diff.Balances, wantBalances) {
		t.Fatalf("balances: %#v", diff.Balances)
	}
	wantSlots := []TransactionSlotChange{{a, k2, k2, k1}}
	if !reflect.DeepEqual(diff.Slots, wantSlots) {
		t.Fatalf("slots: %#v", diff.Slots)
	}
	if len(diff.Deployed) != 1 || diff.Deployed[0] != (TransactionDeployment{b, crypto.Keccak256Hash([]byte{0})}) {
		t.Fatalf("deployed: %#v", diff.Deployed)
	}
	if diff.AccountFlags[a] != 7 || diff.AccountFlags[b] != 10 || diff.AccountFlags[c] != 8 {
		t.Fatalf("flags: %v", diff.AccountFlags)
	}
	if len(diff.Events) != 2 || diff.Events[0].Address != b || diff.Events[1].Address != a {
		t.Fatalf("events: %v", diff.Events)
	}
	if !reflect.DeepEqual(diff.TopicIndices[k2], []uint64{0}) || !reflect.DeepEqual(diff.TopicIndices[k1], []uint64{1}) {
		t.Fatalf("topic indices: %v", diff.TopicIndices)
	}
	s.SetBalance(a, uint256.NewInt(10), tracing.BalanceChangeUnspecified)
	s.SetState(a, k2, k2)
	s.SetNonce(a, 1, tracing.NonceChangeUnspecified)
	diff = s.GetTransactionDiff()
	if len(diff.Balances) != 1 || len(diff.Slots) != 0 || diff.AccountFlags[a] != 0 {
		t.Fatalf("restored values remain: %#v", diff)
	}
}

func TestAssertionSlotAccessList(t *testing.T) {
	s, _ := New(types.EmptyRootHash, NewDatabaseForTesting())
	addr, key := common.HexToAddress("0x123"), common.HexToHash("0x45")
	snapshot := s.Snapshot()
	s.AddSlotToAccessListOnly(addr, key)
	for _, db := range []*StateDB{s, s.Copy()} {
		if account, slot := db.SlotInAccessList(addr, key); account || !slot {
			t.Fatalf("independent slot warmth = %v, %v", account, slot)
		}
	}
	nested := s.Snapshot()
	s.AddAddressToAccessList(addr)
	if account, slot := s.SlotInAccessList(addr, key); !account || !slot {
		t.Fatal("warming account lost slot")
	}
	s.RevertToSnapshot(nested)
	if account, slot := s.SlotInAccessList(addr, key); account || !slot {
		t.Fatal("account revert lost slot")
	}
	s.AddSlotToAccessList(addr, key)
	if account, slot := s.SlotInAccessList(addr, key); !account || !slot {
		t.Fatal("ordinary slot warming failed")
	}
	s.RevertToSnapshot(snapshot)
	if account, slot := s.SlotInAccessList(addr, key); account || slot {
		t.Fatal("slot warmth survived revert")
	}
}

func TestTransactionDiffUnloadedCode(t *testing.T) {
	database := NewDatabaseForTesting()
	s, _ := New(types.EmptyRootHash, database)
	addr := common.HexToAddress("0x123")
	code := []byte{0x60, 0x01}
	s.SetCode(addr, code, tracing.CodeChangeUnspecified)
	root, err := s.Commit(params.Rules{}, 0)
	if err != nil {
		t.Fatal(err)
	}
	s, _ = New(root, database)
	s.SetCode(addr, []byte{0x00}, tracing.CodeChangeUnspecified)
	before := s.GetTransactionAccount(addr)
	if before.CodeHash != crypto.Keccak256Hash(code) {
		t.Fatalf("unloaded original code hash = %x", before.CodeHash)
	}
	if diff := s.GetTransactionDiff(); len(diff.Deployed) != 0 || diff.AccountFlags[addr] != 8 {
		t.Fatalf("replaced code: %#v", diff)
	}
	s.SetCode(addr, code, tracing.CodeChangeUnspecified)
	if diff := s.GetTransactionDiff(); len(diff.Deployed) != 0 || diff.AccountFlags[addr] != 0 {
		t.Fatalf("restored code: %#v", diff)
	}
}

func TestAssertionMissingSlotBAL(t *testing.T) {
	s, _ := New(types.EmptyRootHash, NewDatabaseForTesting())
	s.Prepare(params.Rules{IsAmsterdam: true}, common.Address{}, common.Address{}, nil, nil, nil)
	addr, key := common.HexToAddress("0x123"), common.HexToHash("0x45")
	s.GetState(addr, key)
	if s.GetTransactionState(addr, key) != (common.Hash{}) {
		t.Fatal("absent slot prestate is not zero")
	}
	entries := *s.stateAccessList.ToEncodingObj()
	if len(entries) != 1 || entries[0].Address != addr || len(entries[0].StorageReads) != 1 || common.Hash(entries[0].StorageReads[0].Bytes32()) != key {
		t.Fatalf("missing slot BAL: %#v", entries)
	}
}
