// Copyright 2014 The go-ethereum Authors
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
	"math/big"
	"testing"

	checker "gopkg.in/check.v1"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
)

type StateSuite struct {
	state *StateDB
}

var _ = checker.Suite(&StateSuite{})

var toAddr = common.BytesToAddress

func (s *StateSuite) TestDump(c *checker.C) {
	// generate a few entries
	obj1 := s.state.GetOrNewStateObject(toAddr([]byte{0x01}))
	obj1.AddBalance(big.NewInt(22))
	obj2 := s.state.GetOrNewStateObject(toAddr([]byte{0x01, 0x02}))
	obj2.SetCode([]byte{3, 3, 3, 3, 3, 3, 3})
	obj3 := s.state.GetOrNewStateObject(toAddr([]byte{0x02}))
	obj3.SetBalance(big.NewInt(44))

	// write some of them to the trie
	s.state.UpdateStateObject(obj1)
	s.state.UpdateStateObject(obj2)
	s.state.Commit()

	// check that dump contains the state objects that are in trie
	got := string(s.state.Dump())
	want := `{
    "root": "71edff0130dd2385947095001c73d9e28d862fc286fca2b922ca6f6f3cddfdd2",
    "accounts": {
        "0000000000000000000000000000000000000001": {
            "balance": "22",
            "nonce": 0,
            "root": "56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421",
            "codeHash": "c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470",
            "code": "",
            "storage": {}
        },
        "0000000000000000000000000000000000000002": {
            "balance": "44",
            "nonce": 0,
            "root": "56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421",
            "codeHash": "c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470",
            "code": "",
            "storage": {}
        },
        "0000000000000000000000000000000000000102": {
            "balance": "0",
            "nonce": 0,
            "root": "56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421",
            "codeHash": "87874902497a5bb968da31a2998d8f22e949d1ef6214bcdedd8bae24cca4b9e3",
            "code": "03030303030303",
            "storage": {}
        }
    }
}`
	if got != want {
		c.Errorf("dump mismatch:\ngot: %s\nwant: %s\n", got, want)
	}
}

func (s *StateSuite) SetUpTest(c *checker.C) {
	db, _ := ethdb.NewMemDatabase()
	s.state, _ = New(common.Hash{}, db)
}

func TestNull(t *testing.T) {
	db, _ := ethdb.NewMemDatabase()
	state, _ := New(common.Hash{}, db)

	address := common.HexToAddress("0x823140710bf13990e4500136726d8b55")
	state.CreateAccount(address)
	//value := common.FromHex("0x823140710bf13990e4500136726d8b55")
	var value common.Hash
	state.SetState(address, common.Hash{}, value)
	state.Commit()
	value = state.GetState(address, common.Hash{})
	if !common.EmptyHash(value) {
		t.Errorf("expected empty hash. got %x", value)
	}
}

func (s *StateSuite) TestSnapshot(c *checker.C) {
	stateobjaddr := toAddr([]byte("aa"))
	var storageaddr common.Hash
	data1 := common.BytesToHash([]byte{42})
	data2 := common.BytesToHash([]byte{43})

	// set initial state object value
	s.state.SetState(stateobjaddr, storageaddr, data1)
	// get snapshot of current state
	snapshot := s.state.Copy()

	// set new state object value
	s.state.SetState(stateobjaddr, storageaddr, data2)
	// restore snapshot
	s.state.Set(snapshot)

	// get state storage value
	res := s.state.GetState(stateobjaddr, storageaddr)

	c.Assert(data1, checker.DeepEquals, res)
}

// use testing instead of checker because checker does not support
// printing/logging in tests (-check.vv does not work)
func TestSnapshot2(t *testing.T) {
	db, _ := ethdb.NewMemDatabase()
	state, _ := New(common.Hash{}, db)

	stateobjaddr0 := toAddr([]byte("so0"))
	stateobjaddr1 := toAddr([]byte("so1"))
	var storageaddr common.Hash

	data0 := common.BytesToHash([]byte{17})
	data1 := common.BytesToHash([]byte{18})

	state.SetState(stateobjaddr0, storageaddr, data0)
	state.SetState(stateobjaddr1, storageaddr, data1)

	// db, trie are already non-empty values
	so0 := state.GetStateObject(stateobjaddr0)
	so0.balance = big.NewInt(42)
	so0.nonce = 43
	so0.SetCode([]byte{'c', 'a', 'f', 'e'})
	so0.remove = false
	so0.deleted = false
	so0.dirty = true
	state.SetStateObject(so0)
	state.Commit()

	// and one with deleted == true
	so1 := state.GetStateObject(stateobjaddr1)
	so1.balance = big.NewInt(52)
	so1.nonce = 53
	so1.SetCode([]byte{'c', 'a', 'f', 'e', '2'})
	so1.remove = true
	so1.deleted = true
	so1.dirty = true
	state.SetStateObject(so1)

	so1 = state.GetStateObject(stateobjaddr1)
	if so1 != nil {
		t.Fatalf("deleted object not nil when getting")
	}

	snapshot := state.Copy()
	state.Set(snapshot)

	so0Restored := state.GetStateObject(stateobjaddr0)
	so0Restored.GetState(storageaddr)
	so1Restored := state.GetStateObject(stateobjaddr1)
	// non-deleted is equal (restored)
	compareStateObjects(so0Restored, so0, t)
	// deleted should be nil, both before and after restore of state copy
	if so1Restored != nil {
		t.Fatalf("deleted object not nil after restoring snapshot")
	}
}

func compareStateObjects(so0, so1 *StateObject, t *testing.T) {
	if so0.address != so1.address {
		t.Fatalf("Address mismatch: have %v, want %v", so0.address, so1.address)
	}
	if so0.balance.Cmp(so1.balance) != 0 {
		t.Fatalf("Balance mismatch: have %v, want %v", so0.balance, so1.balance)
	}
	if so0.nonce != so1.nonce {
		t.Fatalf("Nonce mismatch: have %v, want %v", so0.nonce, so1.nonce)
	}
	if !bytes.Equal(so0.codeHash, so1.codeHash) {
		t.Fatalf("CodeHash mismatch: have %v, want %v", so0.codeHash, so1.codeHash)
	}
	if !bytes.Equal(so0.code, so1.code) {
		t.Fatalf("Code mismatch: have %v, want %v", so0.code, so1.code)
	}
	if !bytes.Equal(so0.initCode, so1.initCode) {
		t.Fatalf("InitCode mismatch: have %v, want %v", so0.initCode, so1.initCode)
	}

	for k, v := range so1.storage {
		if so0.storage[k] != v {
			t.Fatalf("Storage key %s mismatch: have %v, want %v", k, so0.storage[k], v)
		}
	}
	for k, v := range so0.storage {
		if so1.storage[k] != v {
			t.Fatalf("Storage key %s mismatch: have %v, want none.", k, v)
		}
	}

	if so0.remove != so1.remove {
		t.Fatalf("Remove mismatch: have %v, want %v", so0.remove, so1.remove)
	}
	if so0.deleted != so1.deleted {
		t.Fatalf("Deleted mismatch: have %v, want %v", so0.deleted, so1.deleted)
	}
	if so0.dirty != so1.dirty {
		t.Fatalf("Dirty mismatch: have %v, want %v", so0.dirty, so1.dirty)
	}
}
