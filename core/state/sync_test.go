// Copyright 2015 The go-ethereum Authors
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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/trie"
)

// testAccount is the data associated with an account used by the state tests.
type testAccount struct {
	address common.Address
	balance *big.Int
	nonce   uint64
	code    []byte
}

// makeTestState create a sample test state to test node-wise reconstruction.
func makeTestState() (ethdb.Database, common.Hash, []*testAccount) {
	// Create an empty state
	db, _ := ethdb.NewMemDatabase()
	state, _ := New(common.Hash{}, db)

	// Fill it with some arbitrary data
	accounts := []*testAccount{}
	for i := byte(0); i < 96; i++ {
		obj := state.GetOrNewStateObject(common.BytesToAddress([]byte{i}))
		acc := &testAccount{address: common.BytesToAddress([]byte{i})}

		obj.AddBalance(big.NewInt(int64(11 * i)))
		acc.balance = big.NewInt(int64(11 * i))

		obj.SetNonce(uint64(42 * i))
		acc.nonce = uint64(42 * i)

		if i%3 == 0 {
			obj.SetCode(crypto.Keccak256Hash([]byte{i, i, i, i, i}), []byte{i, i, i, i, i})
			acc.code = []byte{i, i, i, i, i}
		}
		state.updateStateObject(obj)
		accounts = append(accounts, acc)
	}
	root, _ := state.Commit(false)

	// Return the generated state
	return db, root, accounts
}

// checkStateAccounts cross references a reconstructed state with an expected
// account array.
func checkStateAccounts(t *testing.T, db ethdb.Database, root common.Hash, accounts []*testAccount) {
	// Check root availability and state contents
	state, err := New(root, db)
	if err != nil {
		t.Fatalf("failed to create state trie at %x: %v", root, err)
	}
	if err := checkStateConsistency(db, root); err != nil {
		t.Fatalf("inconsistent state trie at %x: %v", root, err)
	}
	for i, acc := range accounts {
		if balance := state.GetBalance(acc.address); balance.Cmp(acc.balance) != 0 {
			t.Errorf("account %d: balance mismatch: have %v, want %v", i, balance, acc.balance)
		}
		if nonce := state.GetNonce(acc.address); nonce != acc.nonce {
			t.Errorf("account %d: nonce mismatch: have %v, want %v", i, nonce, acc.nonce)
		}
		if code := state.GetCode(acc.address); bytes.Compare(code, acc.code) != 0 {
			t.Errorf("account %d: code mismatch: have %x, want %x", i, code, acc.code)
		}
	}
}

// checkStateConsistency checks that all nodes in a state trie are indeed present.
func checkStateConsistency(db ethdb.Database, root common.Hash) error {
	// Create and iterate a state trie rooted in a sub-node
	if _, err := db.Get(root.Bytes()); err != nil {
		return nil // Consider a non existent state consistent
	}
	state, err := New(root, db)
	if err != nil {
		return err
	}
	it := NewNodeIterator(state)
	for it.Next() {
	}
	return it.Error
}

// Tests that an empty state is not scheduled for syncing.
func TestEmptyStateSync(t *testing.T) {
	empty := common.HexToHash("56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")
	db, _ := ethdb.NewMemDatabase()
	if req := NewStateSync(empty, db).Missing(1); len(req) != 0 {
		t.Errorf("content requested for empty state: %v", req)
	}
}

// Tests that given a root hash, a state can sync iteratively on a single thread,
// requesting retrieval tasks and returning all of them in one go.
func TestIterativeStateSyncIndividual(t *testing.T) { testIterativeStateSync(t, 1) }
func TestIterativeStateSyncBatched(t *testing.T)    { testIterativeStateSync(t, 100) }

func testIterativeStateSync(t *testing.T, batch int) {
	// Create a random state to copy
	srcDb, srcRoot, srcAccounts := makeTestState()

	// Create a destination state and sync with the scheduler
	dstDb, _ := ethdb.NewMemDatabase()
	sched := NewStateSync(srcRoot, dstDb)

	queue := append([]common.Hash{}, sched.Missing(batch)...)
	for len(queue) > 0 {
		results := make([]trie.SyncResult, len(queue))
		for i, hash := range queue {
			data, err := srcDb.Get(hash.Bytes())
			if err != nil {
				t.Fatalf("failed to retrieve node data for %x: %v", hash, err)
			}
			results[i] = trie.SyncResult{Hash: hash, Data: data}
		}
		if _, index, err := sched.Process(results, dstDb); err != nil {
			t.Fatalf("failed to process result #%d: %v", index, err)
		}
		queue = append(queue[:0], sched.Missing(batch)...)
	}
	// Cross check that the two states are in sync
	checkStateAccounts(t, dstDb, srcRoot, srcAccounts)
}

// Tests that the trie scheduler can correctly reconstruct the state even if only
// partial results are returned, and the others sent only later.
func TestIterativeDelayedStateSync(t *testing.T) {
	// Create a random state to copy
	srcDb, srcRoot, srcAccounts := makeTestState()

	// Create a destination state and sync with the scheduler
	dstDb, _ := ethdb.NewMemDatabase()
	sched := NewStateSync(srcRoot, dstDb)

	queue := append([]common.Hash{}, sched.Missing(0)...)
	for len(queue) > 0 {
		// Sync only half of the scheduled nodes
		results := make([]trie.SyncResult, len(queue)/2+1)
		for i, hash := range queue[:len(results)] {
			data, err := srcDb.Get(hash.Bytes())
			if err != nil {
				t.Fatalf("failed to retrieve node data for %x: %v", hash, err)
			}
			results[i] = trie.SyncResult{Hash: hash, Data: data}
		}
		if _, index, err := sched.Process(results, dstDb); err != nil {
			t.Fatalf("failed to process result #%d: %v", index, err)
		}
		queue = append(queue[len(results):], sched.Missing(0)...)
	}
	// Cross check that the two states are in sync
	checkStateAccounts(t, dstDb, srcRoot, srcAccounts)
}

// Tests that given a root hash, a trie can sync iteratively on a single thread,
// requesting retrieval tasks and returning all of them in one go, however in a
// random order.
func TestIterativeRandomStateSyncIndividual(t *testing.T) { testIterativeRandomStateSync(t, 1) }
func TestIterativeRandomStateSyncBatched(t *testing.T)    { testIterativeRandomStateSync(t, 100) }

func testIterativeRandomStateSync(t *testing.T, batch int) {
	// Create a random state to copy
	srcDb, srcRoot, srcAccounts := makeTestState()

	// Create a destination state and sync with the scheduler
	dstDb, _ := ethdb.NewMemDatabase()
	sched := NewStateSync(srcRoot, dstDb)

	queue := make(map[common.Hash]struct{})
	for _, hash := range sched.Missing(batch) {
		queue[hash] = struct{}{}
	}
	for len(queue) > 0 {
		// Fetch all the queued nodes in a random order
		results := make([]trie.SyncResult, 0, len(queue))
		for hash := range queue {
			data, err := srcDb.Get(hash.Bytes())
			if err != nil {
				t.Fatalf("failed to retrieve node data for %x: %v", hash, err)
			}
			results = append(results, trie.SyncResult{Hash: hash, Data: data})
		}
		// Feed the retrieved results back and queue new tasks
		if _, index, err := sched.Process(results, dstDb); err != nil {
			t.Fatalf("failed to process result #%d: %v", index, err)
		}
		queue = make(map[common.Hash]struct{})
		for _, hash := range sched.Missing(batch) {
			queue[hash] = struct{}{}
		}
	}
	// Cross check that the two states are in sync
	checkStateAccounts(t, dstDb, srcRoot, srcAccounts)
}

// Tests that the trie scheduler can correctly reconstruct the state even if only
// partial results are returned (Even those randomly), others sent only later.
func TestIterativeRandomDelayedStateSync(t *testing.T) {
	// Create a random state to copy
	srcDb, srcRoot, srcAccounts := makeTestState()

	// Create a destination state and sync with the scheduler
	dstDb, _ := ethdb.NewMemDatabase()
	sched := NewStateSync(srcRoot, dstDb)

	queue := make(map[common.Hash]struct{})
	for _, hash := range sched.Missing(0) {
		queue[hash] = struct{}{}
	}
	for len(queue) > 0 {
		// Sync only half of the scheduled nodes, even those in random order
		results := make([]trie.SyncResult, 0, len(queue)/2+1)
		for hash := range queue {
			delete(queue, hash)

			data, err := srcDb.Get(hash.Bytes())
			if err != nil {
				t.Fatalf("failed to retrieve node data for %x: %v", hash, err)
			}
			results = append(results, trie.SyncResult{Hash: hash, Data: data})

			if len(results) >= cap(results) {
				break
			}
		}
		// Feed the retrieved results back and queue new tasks
		if _, index, err := sched.Process(results, dstDb); err != nil {
			t.Fatalf("failed to process result #%d: %v", index, err)
		}
		for _, hash := range sched.Missing(0) {
			queue[hash] = struct{}{}
		}
	}
	// Cross check that the two states are in sync
	checkStateAccounts(t, dstDb, srcRoot, srcAccounts)
}

// Tests that at any point in time during a sync, only complete sub-tries are in
// the database.
func TestIncompleteStateSync(t *testing.T) {
	// Create a random state to copy
	srcDb, srcRoot, srcAccounts := makeTestState()

	// Create a destination state and sync with the scheduler
	dstDb, _ := ethdb.NewMemDatabase()
	sched := NewStateSync(srcRoot, dstDb)

	added := []common.Hash{}
	queue := append([]common.Hash{}, sched.Missing(1)...)
	for len(queue) > 0 {
		// Fetch a batch of state nodes
		results := make([]trie.SyncResult, len(queue))
		for i, hash := range queue {
			data, err := srcDb.Get(hash.Bytes())
			if err != nil {
				t.Fatalf("failed to retrieve node data for %x: %v", hash, err)
			}
			results[i] = trie.SyncResult{Hash: hash, Data: data}
		}
		// Process each of the state nodes
		if _, index, err := sched.Process(results, dstDb); err != nil {
			t.Fatalf("failed to process result #%d: %v", index, err)
		}
		for _, result := range results {
			added = append(added, result.Hash)
		}
		// Check that all known sub-tries in the synced state is complete
		for _, root := range added {
			// Skim through the accounts and make sure the root hash is not a code node
			codeHash := false
			for _, acc := range srcAccounts {
				if bytes.Compare(root.Bytes(), crypto.Sha3(acc.code)) == 0 {
					codeHash = true
					break
				}
			}
			// If the root is a real trie node, check consistency
			if !codeHash {
				if err := checkStateConsistency(dstDb, root); err != nil {
					t.Fatalf("state inconsistent: %v", err)
				}
			}
		}
		// Fetch the next batch to retrieve
		queue = append(queue[:0], sched.Missing(1)...)
	}
	// Sanity check that removing any node from the database is detected
	for _, node := range added[1:] {
		key := node.Bytes()
		value, _ := dstDb.Get(key)

		dstDb.Delete(key)
		if err := checkStateConsistency(dstDb, added[0]); err == nil {
			t.Fatalf("trie inconsistency not caught, missing: %x", key)
		}
		dstDb.Put(key, value)
	}
}
