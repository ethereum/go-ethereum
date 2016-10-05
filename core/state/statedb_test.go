// Copyright 2016 The go-ethereum Authors
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
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
)

// Tests that updating a state trie does not leak any database writes prior to
// actually committing the state.
func TestUpdateLeaks(t *testing.T) {
	// Create an empty state database
	db, _ := ethdb.NewMemDatabase()
	state, _ := New(common.Hash{}, db)

	// Update it with some accounts
	for i := byte(0); i < 255; i++ {
		addr := common.BytesToAddress([]byte{i})
		state.AddBalance(addr, big.NewInt(int64(11*i)))
		state.SetNonce(addr, uint64(42*i))
		if i%2 == 0 {
			state.SetState(addr, common.BytesToHash([]byte{i, i, i}), common.BytesToHash([]byte{i, i, i, i}))
		}
		if i%3 == 0 {
			state.SetCode(addr, []byte{i, i, i, i, i})
		}
		state.IntermediateRoot()
	}
	// Ensure that no data was leaked into the database
	for _, key := range db.Keys() {
		value, _ := db.Get(key)
		t.Errorf("State leaked into database: %x -> %x", key, value)
	}
}

type action struct {
	typ string

	address common.Address
	amount  *big.Int
}

type scriptedTest struct {
	actions    []action
	finalState []action

	revisions []int
}

func TestDelete(t *testing.T) {
	db, _ := ethdb.NewMemDatabase()
	state, _ := New(common.Hash{}, db)

	test := scriptedTest{
		actions: []action{
			{typ: "add", address: common.Address{1}, amount: big.NewInt(1)},
			{typ: "snapshot"},
			{typ: "delete", address: common.Address{1}},
			{typ: "revert"},
		},
		finalState: []action{
			{typ: "add", address: common.Address{1}, amount: big.NewInt(1)},
		},
	}
	for _, action := range test.actions {
		applyAction(state, action, &test)
	}

	expectedState, _ := New(common.Hash{}, db)
	for _, action := range test.finalState {
		applyAction(expectedState, action, &test)
	}
	if expectedState.IntermediateRoot() != state.IntermediateRoot() {
		t.Errorf("state does not match expected state")
	}
}

func applyAction(st *StateDB, action action, test *scriptedTest) {
	switch action.typ {
	case "add":
		st.AddBalance(action.address, action.amount)
	case "snapshot":
		test.revisions = append(test.revisions, st.Snapshot())
	case "delete":
		st.Delete(action.address)
	case "revert":
		st.RevertToSnapshot(test.revisions[len(test.revisions)-1])

		test.revisions = test.revisions[:len(test.revisions)-1]
	}
}

// Tests that no intermediate state of an object is stored into the database,
// only the one right before the commit.
// func TestIntermediateLeaks(t *testing.T) {
// 	// Create two state databases, one transitioning to the final state, the other final from the beginning
// 	transDb, _ := ethdb.NewMemDatabase()
// 	finalDb, _ := ethdb.NewMemDatabase()
// 	transState, _ := New(common.Hash{}, transDb)
// 	finalState, _ := New(common.Hash{}, finalDb)
//
// 	// Update the states with some objects
// 	for i := byte(0); i < 255; i++ {
// 		// Create a new state object with some data into the transition database
// 		obj := transState.GetOrNewStateObject(common.BytesToAddress([]byte{i}))
// 		obj.SetBalance(big.NewInt(int64(11 * i)))
// 		obj.SetNonce(uint64(42 * i))
// 		if i%2 == 0 {
// 			obj.SetState(common.BytesToHash([]byte{i, i, i, 0}), common.BytesToHash([]byte{i, i, i, i, 0}))
// 		}
// 		if i%3 == 0 {
// 			obj.SetCode(crypto.Keccak256Hash([]byte{i, i, i, i, i, 0}), []byte{i, i, i, i, i, 0})
// 		}
// 		transState.UpdateStateObject(obj)
//
// 		// Overwrite all the data with new values in the transition database
// 		obj.SetBalance(big.NewInt(int64(11*i + 1)))
// 		obj.SetNonce(uint64(42*i + 1))
// 		if i%2 == 0 {
// 			obj.SetState(common.BytesToHash([]byte{i, i, i, 0}), common.Hash{})
// 			obj.SetState(common.BytesToHash([]byte{i, i, i, 1}), common.BytesToHash([]byte{i, i, i, i, 1}))
// 		}
// 		if i%3 == 0 {
// 			obj.SetCode(crypto.Keccak256Hash([]byte{i, i, i, i, i, 1}), []byte{i, i, i, i, i, 1})
// 		}
// 		transState.UpdateStateObject(obj)
//
// 		// Create the final state object directly in the final database
// 		obj = finalState.GetOrNewStateObject(common.BytesToAddress([]byte{i}))
// 		obj.SetBalance(big.NewInt(int64(11*i + 1)))
// 		obj.SetNonce(uint64(42*i + 1))
// 		if i%2 == 0 {
// 			obj.SetState(common.BytesToHash([]byte{i, i, i, 1}), common.BytesToHash([]byte{i, i, i, i, 1}))
// 		}
// 		if i%3 == 0 {
// 			obj.SetCode(crypto.Keccak256Hash([]byte{i, i, i, i, i, 1}), []byte{i, i, i, i, i, 1})
// 		}
// 		finalState.UpdateStateObject(obj)
// 	}
// 	if _, err := transState.Commit(); err != nil {
// 		t.Fatalf("failed to commit transition state: %v", err)
// 	}
// 	if _, err := finalState.Commit(); err != nil {
// 		t.Fatalf("failed to commit final state: %v", err)
// 	}
// 	// Cross check the databases to ensure they are the same
// 	for _, key := range finalDb.Keys() {
// 		if _, err := transDb.Get(key); err != nil {
// 			val, _ := finalDb.Get(key)
// 			t.Errorf("entry missing from the transition database: %x -> %x", key, val)
// 		}
// 	}
// 	for _, key := range transDb.Keys() {
// 		if _, err := finalDb.Get(key); err != nil {
// 			val, _ := transDb.Get(key)
// 			t.Errorf("extra entry in the transition database: %x -> %x", key, val)
// 		}
// 	}
// }
