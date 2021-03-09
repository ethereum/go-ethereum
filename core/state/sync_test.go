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
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/rlp"
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
func makeTestState() (Database, common.Hash, []*testAccount) {
	// Create an empty state
	db := NewDatabase(rawdb.NewMemoryDatabase())
	state, _ := New(common.Hash{}, db, nil)

	// Fill it with some arbitrary data
	var accounts []*testAccount
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
		if i%5 == 0 {
			for j := byte(0); j < 5; j++ {
				hash := crypto.Keccak256Hash([]byte{i, i, i, i, i, j, j})
				obj.SetState(db, hash, hash)
			}
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
	state, err := New(root, NewDatabase(db), nil)
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
		if code := state.GetCode(acc.address); !bytes.Equal(code, acc.code) {
			t.Errorf("account %d: code mismatch: have %x, want %x", i, code, acc.code)
		}
	}
}

// checkTrieConsistency checks that all nodes in a (sub-)trie are indeed present.
func checkTrieConsistency(db ethdb.Database, root common.Hash) error {
	if v, _ := db.Get(root[:]); v == nil {
		return nil // Consider a non existent state consistent.
	}
	trie, err := trie.New(root, trie.NewDatabase(db))
	if err != nil {
		return err
	}
	it := trie.NodeIterator(nil)
	for it.Next(true) {
	}
	return it.Error()
}

// checkStateConsistency checks that all data of a state root is present.
func checkStateConsistency(db ethdb.Database, root common.Hash) error {
	// Create and iterate a state trie rooted in a sub-node
	if _, err := db.Get(root.Bytes()); err != nil {
		return nil // Consider a non existent state consistent.
	}
	state, err := New(root, NewDatabase(db), nil)
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
	sync := NewStateSync(empty, rawdb.NewMemoryDatabase(), trie.NewSyncBloom(1, memorydb.New()))
	if nodes, paths, codes := sync.Missing(1); len(nodes) != 0 || len(paths) != 0 || len(codes) != 0 {
		t.Errorf(" content requested for empty state: %v, %v, %v", nodes, paths, codes)
	}
}

// Tests that given a root hash, a state can sync iteratively on a single thread,
// requesting retrieval tasks and returning all of them in one go.
func TestIterativeStateSyncIndividual(t *testing.T) {
	testIterativeStateSync(t, 1, false, false)
}
func TestIterativeStateSyncBatched(t *testing.T) {
	testIterativeStateSync(t, 100, false, false)
}
func TestIterativeStateSyncIndividualFromDisk(t *testing.T) {
	testIterativeStateSync(t, 1, true, false)
}
func TestIterativeStateSyncBatchedFromDisk(t *testing.T) {
	testIterativeStateSync(t, 100, true, false)
}
func TestIterativeStateSyncIndividualByPath(t *testing.T) {
	testIterativeStateSync(t, 1, false, true)
}
func TestIterativeStateSyncBatchedByPath(t *testing.T) {
	testIterativeStateSync(t, 100, false, true)
}

func testIterativeStateSync(t *testing.T, count int, commit bool, bypath bool) {
	// Create a random state to copy
	srcDb, srcRoot, srcAccounts := makeTestState()
	if commit {
		srcDb.TrieDB().Commit(srcRoot, false, nil)
	}
	srcTrie, _ := trie.New(srcRoot, srcDb.TrieDB())

	// Create a destination state and sync with the scheduler
	dstDb := rawdb.NewMemoryDatabase()
	sched := NewStateSync(srcRoot, dstDb, trie.NewSyncBloom(1, dstDb))

	nodes, paths, codes := sched.Missing(count)
	var (
		hashQueue []common.Hash
		pathQueue []trie.SyncPath
	)
	if !bypath {
		hashQueue = append(append(hashQueue[:0], nodes...), codes...)
	} else {
		hashQueue = append(hashQueue[:0], codes...)
		pathQueue = append(pathQueue[:0], paths...)
	}
	for len(hashQueue)+len(pathQueue) > 0 {
		results := make([]trie.SyncResult, len(hashQueue)+len(pathQueue))
		for i, hash := range hashQueue {
			data, err := srcDb.TrieDB().Node(hash)
			if err != nil {
				data, err = srcDb.ContractCode(common.Hash{}, hash)
			}
			if err != nil {
				t.Fatalf("failed to retrieve node data for hash %x", hash)
			}
			results[i] = trie.SyncResult{Hash: hash, Data: data}
		}
		for i, path := range pathQueue {
			if len(path) == 1 {
				data, _, err := srcTrie.TryGetNode(path[0])
				if err != nil {
					t.Fatalf("failed to retrieve node data for path %x: %v", path, err)
				}
				results[len(hashQueue)+i] = trie.SyncResult{Hash: crypto.Keccak256Hash(data), Data: data}
			} else {
				var acc Account
				if err := rlp.DecodeBytes(srcTrie.Get(path[0]), &acc); err != nil {
					t.Fatalf("failed to decode account on path %x: %v", path, err)
				}
				stTrie, err := trie.New(acc.Root, srcDb.TrieDB())
				if err != nil {
					t.Fatalf("failed to retriev storage trie for path %x: %v", path, err)
				}
				data, _, err := stTrie.TryGetNode(path[1])
				if err != nil {
					t.Fatalf("failed to retrieve node data for path %x: %v", path, err)
				}
				results[len(hashQueue)+i] = trie.SyncResult{Hash: crypto.Keccak256Hash(data), Data: data}
			}
		}
		for _, result := range results {
			if err := sched.Process(result); err != nil {
				t.Errorf("failed to process result %v", err)
			}
		}
		batch := dstDb.NewBatch()
		if err := sched.Commit(batch); err != nil {
			t.Fatalf("failed to commit data: %v", err)
		}
		batch.Write()

		nodes, paths, codes = sched.Missing(count)
		if !bypath {
			hashQueue = append(append(hashQueue[:0], nodes...), codes...)
		} else {
			hashQueue = append(hashQueue[:0], codes...)
			pathQueue = append(pathQueue[:0], paths...)
		}
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
	dstDb := rawdb.NewMemoryDatabase()
	sched := NewStateSync(srcRoot, dstDb, trie.NewSyncBloom(1, dstDb))

	nodes, _, codes := sched.Missing(0)
	queue := append(append([]common.Hash{}, nodes...), codes...)

	for len(queue) > 0 {
		// Sync only half of the scheduled nodes
		results := make([]trie.SyncResult, len(queue)/2+1)
		for i, hash := range queue[:len(results)] {
			data, err := srcDb.TrieDB().Node(hash)
			if err != nil {
				data, err = srcDb.ContractCode(common.Hash{}, hash)
			}
			if err != nil {
				t.Fatalf("failed to retrieve node data for %x", hash)
			}
			results[i] = trie.SyncResult{Hash: hash, Data: data}
		}
		for _, result := range results {
			if err := sched.Process(result); err != nil {
				t.Fatalf("failed to process result %v", err)
			}
		}
		batch := dstDb.NewBatch()
		if err := sched.Commit(batch); err != nil {
			t.Fatalf("failed to commit data: %v", err)
		}
		batch.Write()

		nodes, _, codes = sched.Missing(0)
		queue = append(append(queue[len(results):], nodes...), codes...)
	}
	// Cross check that the two states are in sync
	checkStateAccounts(t, dstDb, srcRoot, srcAccounts)
}

// Tests that given a root hash, a trie can sync iteratively on a single thread,
// requesting retrieval tasks and returning all of them in one go, however in a
// random order.
func TestIterativeRandomStateSyncIndividual(t *testing.T) { testIterativeRandomStateSync(t, 1) }
func TestIterativeRandomStateSyncBatched(t *testing.T)    { testIterativeRandomStateSync(t, 100) }

func testIterativeRandomStateSync(t *testing.T, count int) {
	// Create a random state to copy
	srcDb, srcRoot, srcAccounts := makeTestState()

	// Create a destination state and sync with the scheduler
	dstDb := rawdb.NewMemoryDatabase()
	sched := NewStateSync(srcRoot, dstDb, trie.NewSyncBloom(1, dstDb))

	queue := make(map[common.Hash]struct{})
	nodes, _, codes := sched.Missing(count)
	for _, hash := range append(nodes, codes...) {
		queue[hash] = struct{}{}
	}
	for len(queue) > 0 {
		// Fetch all the queued nodes in a random order
		results := make([]trie.SyncResult, 0, len(queue))
		for hash := range queue {
			data, err := srcDb.TrieDB().Node(hash)
			if err != nil {
				data, err = srcDb.ContractCode(common.Hash{}, hash)
			}
			if err != nil {
				t.Fatalf("failed to retrieve node data for %x", hash)
			}
			results = append(results, trie.SyncResult{Hash: hash, Data: data})
		}
		// Feed the retrieved results back and queue new tasks
		for _, result := range results {
			if err := sched.Process(result); err != nil {
				t.Fatalf("failed to process result %v", err)
			}
		}
		batch := dstDb.NewBatch()
		if err := sched.Commit(batch); err != nil {
			t.Fatalf("failed to commit data: %v", err)
		}
		batch.Write()

		queue = make(map[common.Hash]struct{})
		nodes, _, codes = sched.Missing(count)
		for _, hash := range append(nodes, codes...) {
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
	dstDb := rawdb.NewMemoryDatabase()
	sched := NewStateSync(srcRoot, dstDb, trie.NewSyncBloom(1, dstDb))

	queue := make(map[common.Hash]struct{})
	nodes, _, codes := sched.Missing(0)
	for _, hash := range append(nodes, codes...) {
		queue[hash] = struct{}{}
	}
	for len(queue) > 0 {
		// Sync only half of the scheduled nodes, even those in random order
		results := make([]trie.SyncResult, 0, len(queue)/2+1)
		for hash := range queue {
			delete(queue, hash)

			data, err := srcDb.TrieDB().Node(hash)
			if err != nil {
				data, err = srcDb.ContractCode(common.Hash{}, hash)
			}
			if err != nil {
				t.Fatalf("failed to retrieve node data for %x", hash)
			}
			results = append(results, trie.SyncResult{Hash: hash, Data: data})

			if len(results) >= cap(results) {
				break
			}
		}
		// Feed the retrieved results back and queue new tasks
		for _, result := range results {
			if err := sched.Process(result); err != nil {
				t.Fatalf("failed to process result %v", err)
			}
		}
		batch := dstDb.NewBatch()
		if err := sched.Commit(batch); err != nil {
			t.Fatalf("failed to commit data: %v", err)
		}
		batch.Write()
		for _, result := range results {
			delete(queue, result.Hash)
		}
		nodes, _, codes = sched.Missing(0)
		for _, hash := range append(nodes, codes...) {
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

	// isCodeLookup to save some hashing
	var isCode = make(map[common.Hash]struct{})
	for _, acc := range srcAccounts {
		if len(acc.code) > 0 {
			isCode[crypto.Keccak256Hash(acc.code)] = struct{}{}
		}
	}
	isCode[common.BytesToHash(emptyCodeHash)] = struct{}{}
	checkTrieConsistency(srcDb.TrieDB().DiskDB().(ethdb.Database), srcRoot)

	// Create a destination state and sync with the scheduler
	dstDb := rawdb.NewMemoryDatabase()
	sched := NewStateSync(srcRoot, dstDb, trie.NewSyncBloom(1, dstDb))

	var added []common.Hash

	nodes, _, codes := sched.Missing(1)
	queue := append(append([]common.Hash{}, nodes...), codes...)

	for len(queue) > 0 {
		// Fetch a batch of state nodes
		results := make([]trie.SyncResult, len(queue))
		for i, hash := range queue {
			data, err := srcDb.TrieDB().Node(hash)
			if err != nil {
				data, err = srcDb.ContractCode(common.Hash{}, hash)
			}
			if err != nil {
				t.Fatalf("failed to retrieve node data for %x", hash)
			}
			results[i] = trie.SyncResult{Hash: hash, Data: data}
		}
		// Process each of the state nodes
		for _, result := range results {
			if err := sched.Process(result); err != nil {
				t.Fatalf("failed to process result %v", err)
			}
		}
		batch := dstDb.NewBatch()
		if err := sched.Commit(batch); err != nil {
			t.Fatalf("failed to commit data: %v", err)
		}
		batch.Write()
		for _, result := range results {
			added = append(added, result.Hash)
			// Check that all known sub-tries added so far are complete or missing entirely.
			if _, ok := isCode[result.Hash]; ok {
				continue
			}
			// Can't use checkStateConsistency here because subtrie keys may have odd
			// length and crash in LeafKey.
			if err := checkTrieConsistency(dstDb, result.Hash); err != nil {
				t.Fatalf("state inconsistent: %v", err)
			}
		}
		// Fetch the next batch to retrieve
		nodes, _, codes = sched.Missing(1)
		queue = append(append(queue[:0], nodes...), codes...)
	}
	// Sanity check that removing any node from the database is detected
	for _, node := range added[1:] {
		var (
			key     = node.Bytes()
			_, code = isCode[node]
			val     []byte
		)
		if code {
			val = rawdb.ReadCode(dstDb, node)
			rawdb.DeleteCode(dstDb, node)
		} else {
			val = rawdb.ReadTrieNode(dstDb, node)
			rawdb.DeleteTrieNode(dstDb, node)
		}
		if err := checkStateConsistency(dstDb, added[0]); err == nil {
			t.Fatalf("trie inconsistency not caught, missing: %x", key)
		}
		if code {
			rawdb.WriteCode(dstDb, node, val)
		} else {
			rawdb.WriteTrieNode(dstDb, node, val)
		}
	}
}
