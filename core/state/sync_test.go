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
	trie, err := trie.New(root, trie.NewDatabase(db, nil))
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
	sync := NewStateSync(empty, rawdb.NewMemoryDatabase(), trie.NewSyncBloom(1, memorydb.New()), nil)
	if keys, nodes, paths, codes := sync.Missing(1); len(keys) != 0 || len(nodes) != 0 || len(paths) != 0 || len(codes) != 0 {
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

// stateElement represents the element in the state trie(bytecode or trie node).
type stateElement struct {
	key  string
	hash common.Hash
	path trie.NodePath
	code common.Hash
}

func testIterativeStateSync(t *testing.T, count int, commit bool, bypath bool) {
	// Create a random state to copy
	srcDb, srcRoot, srcAccounts := makeTestState()
	if commit {
		srcDb.TrieDB().Cap(srcRoot, 0)
	}
	srcTrie, _ := trie.New(srcRoot, srcDb.TrieDB())

	// Create a destination state and sync with the scheduler
	dstDb := rawdb.NewMemoryDatabase()
	sched := NewStateSync(srcRoot, dstDb, trie.NewSyncBloom(1, dstDb), nil)

	keys, nodes, paths, codes := sched.Missing(count)
	var (
		nodeElements []stateElement
		codeElements []stateElement
	)
	for i := 0; i < len(keys); i++ {
		nodeElements = append(nodeElements, stateElement{
			key:  keys[i],
			hash: nodes[i],
			path: paths[i],
		})
	}
	for i := 0; i < len(codes); i++ {
		codeElements = append(codeElements, stateElement{
			code: codes[i],
		})
	}
	for len(nodeElements)+len(codeElements) > 0 {
		var (
			nodeResults = make([]trie.NodeSyncResult, len(nodeElements))
			codeResults = make([]trie.CodeSyncResult, len(codeElements))
		)
		for i, element := range codeElements {
			data, err := srcDb.ContractCode(common.Hash{}, element.code)
			if err != nil {
				t.Fatalf("failed to retrieve contract bytecode for hash %x", element.code)
			}
			codeResults[i] = trie.CodeSyncResult{Hash: element.code, Data: data}
		}
		for i, node := range nodeElements {
			if bypath {
				if len(node.path) == 1 {
					data, _, err := srcTrie.TryGetNode(node.path[0])
					if err != nil {
						t.Fatalf("failed to retrieve node data for path %x: %v", node.path[0], err)
					}
					nodeResults[i] = trie.NodeSyncResult{Key: node.key, Data: data}
				} else {
					var acc Account
					if err := rlp.DecodeBytes(srcTrie.Get(node.path[0]), &acc); err != nil {
						t.Fatalf("failed to decode account on path %x: %v", node.path[0], err)
					}
					stTrie, err := trie.NewWithOwner(srcRoot, common.BytesToHash(node.path[0]), acc.Root, srcDb.TrieDB())
					if err != nil {
						t.Fatalf("failed to retriev storage trie for path %x: %v", node.path[1], err)
					}
					data, _, err := stTrie.TryGetNode(node.path[1])
					if err != nil {
						t.Fatalf("failed to retrieve node data for path %x: %v", node.path[1], err)
					}
					nodeResults[i] = trie.NodeSyncResult{Key: node.key, Data: data}
				}
			} else {
				data, err := srcDb.TrieDB().Snapshot(srcRoot).NodeBlob([]byte(node.key))
				if err != nil {
					t.Fatalf("failed to retrieve node data for key %v", []byte(node.key))
				}
				nodeResults[i] = trie.NodeSyncResult{Key: node.key, Data: data}
			}
		}
		for _, result := range codeResults {
			if err := sched.ProcessCode(result); err != nil {
				t.Errorf("failed to process result %v", err)
			}
		}
		for _, result := range nodeResults {
			if err := sched.ProcessNode(result); err != nil {
				t.Errorf("failed to process result %v", err)
			}
		}
		batch := dstDb.NewBatch()
		if err := sched.Commit(batch); err != nil {
			t.Fatalf("failed to commit data: %v", err)
		}
		batch.Write()

		keys, nodes, paths, codes = sched.Missing(count)
		nodeElements = nodeElements[:0]
		for i := 0; i < len(keys); i++ {
			nodeElements = append(nodeElements, stateElement{
				key:  keys[i],
				hash: nodes[i],
				path: paths[i],
			})
		}
		codeElements = codeElements[:0]
		for i := 0; i < len(codes); i++ {
			codeElements = append(codeElements, stateElement{
				code: codes[i],
			})
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
	sched := NewStateSync(srcRoot, dstDb, trie.NewSyncBloom(1, dstDb), nil)

	var (
		nodeElements []stateElement
		codeElements []stateElement
	)
	keys, nodes, paths, codes := sched.Missing(0)
	for i := 0; i < len(keys); i++ {
		nodeElements = append(nodeElements, stateElement{
			key:  keys[i],
			hash: nodes[i],
			path: paths[i],
		})
	}
	for i := 0; i < len(codes); i++ {
		codeElements = append(codeElements, stateElement{
			code: codes[i],
		})
	}

	for len(nodeElements)+len(codeElements) > 0 {
		// Sync only half of the scheduled nodes
		var nodeProcessd int
		var codeProcessd int
		if len(codeElements) > 0 {
			codeResults := make([]trie.CodeSyncResult, len(codeElements)/2+1)
			for i, element := range codeElements[:len(codeResults)] {
				data, err := srcDb.ContractCode(common.Hash{}, element.code)
				if err != nil {
					t.Fatalf("failed to retrieve contract bytecode for %x", element.code)
				}
				codeResults[i] = trie.CodeSyncResult{Hash: element.code, Data: data}
			}
			for _, result := range codeResults {
				if err := sched.ProcessCode(result); err != nil {
					t.Fatalf("failed to process result %v", err)
				}
			}
			codeProcessd = len(codeResults)
		}
		if len(nodeElements) > 0 {
			nodeResults := make([]trie.NodeSyncResult, len(nodeElements)/2+1)
			for i, element := range nodeElements[:len(nodeResults)] {
				data, err := srcDb.TrieDB().Snapshot(srcRoot).NodeBlob([]byte(element.key))
				if err != nil {
					t.Fatalf("failed to retrieve contract bytecode for %x", element.code)
				}
				nodeResults[i] = trie.NodeSyncResult{Key: element.key, Data: data}
			}
			for _, result := range nodeResults {
				if err := sched.ProcessNode(result); err != nil {
					t.Fatalf("failed to process result %v", err)
				}
			}
			nodeProcessd = len(nodeResults)
		}
		batch := dstDb.NewBatch()
		if err := sched.Commit(batch); err != nil {
			t.Fatalf("failed to commit data: %v", err)
		}
		batch.Write()

		keys, nodes, paths, codes = sched.Missing(0)
		nodeElements = nodeElements[nodeProcessd:]
		for i := 0; i < len(keys); i++ {
			nodeElements = append(nodeElements, stateElement{
				key:  keys[i],
				hash: nodes[i],
				path: paths[i],
			})
		}
		codeElements = codeElements[codeProcessd:]
		for i := 0; i < len(codes); i++ {
			codeElements = append(codeElements, stateElement{
				code: codes[i],
			})
		}
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
	sched := NewStateSync(srcRoot, dstDb, trie.NewSyncBloom(1, dstDb), nil)

	nodeQueue := make(map[string]stateElement)
	codeQueue := make(map[common.Hash]struct{})
	keys, nodes, paths, codes := sched.Missing(count)
	for i, key := range keys {
		nodeQueue[key] = stateElement{
			key:  key,
			hash: nodes[i],
			path: paths[i],
		}
	}
	for _, hash := range codes {
		codeQueue[hash] = struct{}{}
	}
	for len(nodeQueue)+len(codeQueue) > 0 {
		// Fetch all the queued nodes in a random order
		if len(codeQueue) > 0 {
			results := make([]trie.CodeSyncResult, 0, len(codeQueue))
			for hash := range codeQueue {
				data, err := srcDb.ContractCode(common.Hash{}, hash)
				if err != nil {
					t.Fatalf("failed to retrieve node data for %x", hash)
				}
				results = append(results, trie.CodeSyncResult{Hash: hash, Data: data})
			}
			for _, result := range results {
				if err := sched.ProcessCode(result); err != nil {
					t.Fatalf("failed to process result %v", err)
				}
			}
		}
		if len(nodeQueue) > 0 {
			results := make([]trie.NodeSyncResult, 0, len(nodeQueue))
			for key, element := range nodeQueue {
				data, err := srcDb.TrieDB().Snapshot(srcRoot).NodeBlob([]byte(element.key))
				if err != nil {
					t.Fatalf("failed to retrieve node data for %x %v %v", element.hash, []byte(element.key), element.path)
				}
				results = append(results, trie.NodeSyncResult{Key: key, Data: data})
			}
			for _, result := range results {
				if err := sched.ProcessNode(result); err != nil {
					t.Fatalf("failed to process result %v", err)
				}
			}
		}
		// Feed the retrieved results back and queue new tasks
		batch := dstDb.NewBatch()
		if err := sched.Commit(batch); err != nil {
			t.Fatalf("failed to commit data: %v", err)
		}
		batch.Write()

		nodeQueue = make(map[string]stateElement)
		codeQueue = make(map[common.Hash]struct{})
		keys, nodes, paths, codes := sched.Missing(count)
		for i, key := range keys {
			nodeQueue[key] = stateElement{
				key:  key,
				hash: nodes[i],
				path: paths[i],
			}
		}
		for _, hash := range codes {
			codeQueue[hash] = struct{}{}
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
	sched := NewStateSync(srcRoot, dstDb, trie.NewSyncBloom(1, dstDb), nil)

	nodeQueue := make(map[string]stateElement)
	codeQueue := make(map[common.Hash]struct{})
	keys, nodes, paths, codes := sched.Missing(0)
	for i, key := range keys {
		nodeQueue[key] = stateElement{
			key:  key,
			hash: nodes[i],
			path: paths[i],
		}
	}
	for _, hash := range codes {
		codeQueue[hash] = struct{}{}
	}
	for len(nodeQueue)+len(codeQueue) > 0 {
		// Sync only half of the scheduled nodes, even those in random order
		if len(codeQueue) > 0 {
			results := make([]trie.CodeSyncResult, 0, len(codeQueue)/2+1)
			for hash := range codeQueue {
				delete(codeQueue, hash)

				data, err := srcDb.ContractCode(common.Hash{}, hash)
				if err != nil {
					t.Fatalf("failed to retrieve node data for %x", hash)
				}
				results = append(results, trie.CodeSyncResult{Hash: hash, Data: data})

				if len(results) >= cap(results) {
					break
				}
			}
			for _, result := range results {
				if err := sched.ProcessCode(result); err != nil {
					t.Fatalf("failed to process result %v", err)
				}
			}
		}
		if len(nodeQueue) > 0 {
			results := make([]trie.NodeSyncResult, 0, len(nodeQueue)/2+1)
			for key, element := range nodeQueue {
				delete(nodeQueue, key)

				data, err := srcDb.TrieDB().Snapshot(srcRoot).NodeBlob([]byte(element.key))
				if err != nil {
					t.Fatalf("failed to retrieve node data for %x", element.hash)
				}
				results = append(results, trie.NodeSyncResult{Key: key, Data: data})

				if len(results) >= cap(results) {
					break
				}
			}
			// Feed the retrieved results back and queue new tasks
			for _, result := range results {
				if err := sched.ProcessNode(result); err != nil {
					t.Fatalf("failed to process result %v", err)
				}
			}
		}
		batch := dstDb.NewBatch()
		if err := sched.Commit(batch); err != nil {
			t.Fatalf("failed to commit data: %v", err)
		}
		batch.Write()

		keys, nodes, paths, codes := sched.Missing(0)
		for i, key := range keys {
			nodeQueue[key] = stateElement{
				key:  key,
				hash: nodes[i],
				path: paths[i],
			}
		}
		for _, hash := range codes {
			codeQueue[hash] = struct{}{}
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
	sched := NewStateSync(srcRoot, dstDb, trie.NewSyncBloom(1, dstDb), nil)

	var (
		addedCodes []common.Hash
		addedNodes []string
	)
	nodeQueue := make(map[string]stateElement)
	codeQueue := make(map[common.Hash]struct{})
	keys, nodes, paths, codes := sched.Missing(1)
	for i, key := range keys {
		nodeQueue[key] = stateElement{
			key:  key,
			hash: nodes[i],
			path: paths[i],
		}
	}
	for _, hash := range codes {
		codeQueue[hash] = struct{}{}
	}
	for len(nodeQueue)+len(codeQueue) > 0 {
		// Fetch a batch of state nodes
		if len(codeQueue) > 0 {
			results := make([]trie.CodeSyncResult, 0, len(codeQueue))
			for hash := range codeQueue {
				data, err := srcDb.ContractCode(common.Hash{}, hash)
				if err != nil {
					t.Fatalf("failed to retrieve node data for %x", hash)
				}
				results = append(results, trie.CodeSyncResult{Hash: hash, Data: data})
				addedCodes = append(addedCodes, hash)
			}
			// Process each of the state nodes
			for _, result := range results {
				if err := sched.ProcessCode(result); err != nil {
					t.Fatalf("failed to process result %v", err)
				}
			}
		}
		var nodehashes []common.Hash
		if len(nodeQueue) > 0 {
			results := make([]trie.NodeSyncResult, 0, len(nodeQueue))
			for key, element := range nodeQueue {
				data, err := srcDb.TrieDB().Snapshot(srcRoot).NodeBlob([]byte(element.key))
				if err != nil {
					t.Fatalf("failed to retrieve node data for %x", element.hash)
				}
				results = append(results, trie.NodeSyncResult{Key: key, Data: data})

				if element.hash != srcRoot {
					addedNodes = append(addedNodes, element.key)
				}
				nodehashes = append(nodehashes, element.hash)
			}
			// Process each of the state nodes
			for _, result := range results {
				if err := sched.ProcessNode(result); err != nil {
					t.Fatalf("failed to process result %v", err)
				}
			}
		}
		batch := dstDb.NewBatch()
		if err := sched.Commit(batch); err != nil {
			t.Fatalf("failed to commit data: %v", err)
		}
		batch.Write()

		for _, root := range nodehashes {
			// Can't use checkStateConsistency here because subtrie keys may have odd
			// length and crash in LeafKey.
			if err := checkTrieConsistency(dstDb, root); err != nil {
				t.Fatalf("state inconsistent: %v", err)
			}
		}
		// Fetch the next batch to retrieve
		nodeQueue = make(map[string]stateElement)
		codeQueue = make(map[common.Hash]struct{})
		keys, nodes, paths, codes := sched.Missing(1)
		for i, key := range keys {
			nodeQueue[key] = stateElement{
				key:  key,
				hash: nodes[i],
				path: paths[i],
			}
		}
		for _, hash := range codes {
			codeQueue[hash] = struct{}{}
		}
	}
	// Sanity check that removing any node from the database is detected
	for _, node := range addedCodes {
		val := rawdb.ReadCode(dstDb, node)
		rawdb.DeleteCode(dstDb, node)
		if err := checkStateConsistency(dstDb, srcRoot); err == nil {
			t.Errorf("trie inconsistency not caught, missing: %x", node)
		}
		rawdb.WriteCode(dstDb, node, val)
	}
	for _, key := range addedNodes {
		storage, hash := trie.DecodeInternalKey([]byte(key))
		val, h := rawdb.ReadTrieNode(dstDb, storage)
		if h != hash {
			t.Errorf("Unexpected trie node want %x got %x", hash, h)
		}
		rawdb.DeleteTrieNode(dstDb, storage)
		if err := checkStateConsistency(dstDb, srcRoot); err == nil {
			t.Errorf("trie inconsistency not caught, missing: %v", storage)
		}
		rawdb.WriteTrieNode(dstDb, storage, val)
	}
}
