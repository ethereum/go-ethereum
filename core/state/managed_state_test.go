// Copyright 2015 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.

package state

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
)

var addr = common.BytesToAddress([]byte("test"))

func create() (*ManagedState, *account) {
	db, _ := ethdb.NewMemDatabase()
	statedb := New(common.Hash{}, db)
	ms := ManageState(statedb)
	so := &StateObject{address: addr, nonce: 100}
	ms.StateDB.stateObjects[addr.Str()] = so
	ms.accounts[addr.Str()] = newAccount(so)

	return ms, ms.accounts[addr.Str()]
}

func TestNewNonce(t *testing.T) {
	ms, _ := create()

	nonce := ms.NewNonce(addr)
	if nonce != 100 {
		t.Error("expected nonce 100. got", nonce)
	}

	nonce = ms.NewNonce(addr)
	if nonce != 101 {
		t.Error("expected nonce 101. got", nonce)
	}
}

func TestRemove(t *testing.T) {
	ms, account := create()

	nn := make([]bool, 10)
	for i, _ := range nn {
		nn[i] = true
	}
	account.nonces = append(account.nonces, nn...)

	i := uint64(5)
	ms.RemoveNonce(addr, account.nstart+i)
	if len(account.nonces) != 5 {
		t.Error("expected", i, "'th index to be false")
	}
}

func TestReuse(t *testing.T) {
	ms, account := create()

	nn := make([]bool, 10)
	for i, _ := range nn {
		nn[i] = true
	}
	account.nonces = append(account.nonces, nn...)

	i := uint64(5)
	ms.RemoveNonce(addr, account.nstart+i)
	nonce := ms.NewNonce(addr)
	if nonce != 105 {
		t.Error("expected nonce to be 105. got", nonce)
	}
}

func TestRemoteNonceChange(t *testing.T) {
	ms, account := create()
	nn := make([]bool, 10)
	for i, _ := range nn {
		nn[i] = true
	}
	account.nonces = append(account.nonces, nn...)
	nonce := ms.NewNonce(addr)

	ms.StateDB.stateObjects[addr.Str()].nonce = 200
	nonce = ms.NewNonce(addr)
	if nonce != 200 {
		t.Error("expected nonce after remote update to be", 201, "got", nonce)
	}
	ms.NewNonce(addr)
	ms.NewNonce(addr)
	ms.NewNonce(addr)
	ms.StateDB.stateObjects[addr.Str()].nonce = 200
	nonce = ms.NewNonce(addr)
	if nonce != 204 {
		t.Error("expected nonce after remote update to be", 201, "got", nonce)
	}
}

func TestSetNonce(t *testing.T) {
	ms, _ := create()

	var addr common.Address
	ms.SetNonce(addr, 10)

	if ms.GetNonce(addr) != 10 {
		t.Error("Expected nonce of 10, got", ms.GetNonce(addr))
	}

	addr[0] = 1
	ms.StateDB.SetNonce(addr, 1)

	if ms.GetNonce(addr) != 1 {
		t.Error("Expected nonce of 1, got", ms.GetNonce(addr))
	}
}
