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

package trie

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
)

// makeTestTrie create a sample test trie to test node-wise reconstruction.
func makeTestTrie() (*Database, *StateTrie, map[string][]byte) {
	// Create an empty trie
	triedb := NewDatabase(memorydb.New())
	trie, _ := NewStateTrie(common.Hash{}, common.Hash{}, triedb)

	// Fill it with some arbitrary data
	content := make(map[string][]byte)
	for i := byte(0); i < 255; i++ {
		// Map the same data under multiple keys
		key, val := common.LeftPadBytes([]byte{1, i}, 32), []byte{i}
		content[string(key)] = val
		trie.Update(key, val)

		key, val = common.LeftPadBytes([]byte{2, i}, 32), []byte{i}
		content[string(key)] = val
		trie.Update(key, val)

		// Add some other data to inflate the trie
		for j := byte(3); j < 13; j++ {
			key, val = common.LeftPadBytes([]byte{j, i}, 32), []byte{j, i}
			content[string(key)] = val
			trie.Update(key, val)
		}
	}
	root, nodes, err := trie.Commit(false)
	if err != nil {
		panic(fmt.Errorf("failed to commit trie %v", err))
	}
	if err := triedb.Update(NewWithNodeSet(nodes)); err != nil {
		panic(fmt.Errorf("failed to commit db %v", err))
	}
	// Re-create the trie based on the new state
	trie, _ = NewSecure(common.Hash{}, root, triedb)
	return triedb, trie, content
}

// checkTrieContents cross references a reconstructed trie with an expected data
// content map.
func checkTrieContents(t *testing.T, db *Database, root []byte, content map[string][]byte) {
	// Check root availability and trie contents
	trie, err := NewStateTrie(common.Hash{}, common.BytesToHash(root), db)
	if err != nil {
		t.Fatalf("failed to create trie at %x: %v", root, err)
	}
	if err := checkTrieConsistency(db, common.BytesToHash(root)); err != nil {
		t.Fatalf("inconsistent trie at %x: %v", root, err)
	}
	for key, val := range content {
		if have := trie.Get([]byte(key)); !bytes.Equal(have, val) {
			t.Errorf("entry %x: content mismatch: have %x, want %x", key, have, val)
		}
	}
}

// checkTrieConsistency checks that all nodes in a trie are indeed present.
func checkTrieConsistency(db *Database, root common.Hash) error {
	// Create and iterate a trie rooted in a subnode
	trie, err := NewStateTrie(common.Hash{}, root, db)
	if err != nil {
		return nil // Consider a non existent state consistent
	}
	it := trie.NodeIterator(nil)
	for it.Next(true) {
	}
	return it.Error()
}

// trieElement represents the element in the state trie(bytecode or trie node).
type trieElement struct {
	path     string
	hash     common.Hash
	syncPath SyncPath
}

// Tests that an empty trie is not scheduled for syncing.
func TestEmptySync(t *testing.T) {
	dbA := NewDatabase(memorydb.New())
	dbB := NewDatabase(memorydb.New())
	emptyA := NewEmpty(dbA)
	emptyB, _ := New(common.Hash{}, emptyRoot, dbB)

	for i, trie := range []*Trie{emptyA, emptyB} {
		sync := NewSync(trie.Hash(), memorydb.New(), nil)
		if paths, nodes, codes := sync.Missing(1); len(paths) != 0 || len(nodes) != 0 || len(codes) != 0 {
			t.Errorf("test %d: content requested for empty trie: %v, %v, %v", i, paths, nodes, codes)
		}
	}
}

// Tests that given a root hash, a trie can sync iteratively on a single thread,
// requesting retrieval tasks and returning all of them in one go.
func TestIterativeSyncIndividual(t *testing.T)       { testIterativeSync(t, 1, false) }
func TestIterativeSyncBatched(t *testing.T)          { testIterativeSync(t, 100, false) }
func TestIterativeSyncIndividualByPath(t *testing.T) { testIterativeSync(t, 1, true) }
func TestIterativeSyncBatchedByPath(t *testing.T)    { testIterativeSync(t, 100, true) }

func testIterativeSync(t *testing.T, count int, bypath bool) {
	// Create a random trie to copy
	srcDb, srcTrie, srcData := makeTestTrie()

	// Create a destination trie and sync with the scheduler
	diskdb := memorydb.New()
	triedb := NewDatabase(diskdb)
	sched := NewSync(srcTrie.Hash(), diskdb, nil)

	// The code requests are ignored here since there is no code
	// at the testing trie.
	paths, nodes, _ := sched.Missing(count)
	var elements []trieElement
	for i := 0; i < len(paths); i++ {
		elements = append(elements, trieElement{
			path:     paths[i],
			hash:     nodes[i],
			syncPath: NewSyncPath([]byte(paths[i])),
		})
	}
	for len(elements) > 0 {
		results := make([]NodeSyncResult, len(elements))
		if !bypath {
			for i, element := range elements {
				data, err := srcDb.Node(element.hash)
				if err != nil {
					t.Fatalf("failed to retrieve node data for hash %x: %v", element.hash, err)
				}
				results[i] = NodeSyncResult{element.path, data}
			}
		} else {
			for i, element := range elements {
				data, _, err := srcTrie.TryGetNode(element.syncPath[len(element.syncPath)-1])
				if err != nil {
					t.Fatalf("failed to retrieve node data for path %x: %v", element.path, err)
				}
				results[i] = NodeSyncResult{element.path, data}
			}
		}
		for _, result := range results {
			if err := sched.ProcessNode(result); err != nil {
				t.Fatalf("failed to process result %v", err)
			}
		}
		batch := diskdb.NewBatch()
		if err := sched.Commit(batch); err != nil {
			t.Fatalf("failed to commit data: %v", err)
		}
		batch.Write()

		paths, nodes, _ = sched.Missing(count)
		elements = elements[:0]
		for i := 0; i < len(paths); i++ {
			elements = append(elements, trieElement{
				path:     paths[i],
				hash:     nodes[i],
				syncPath: NewSyncPath([]byte(paths[i])),
			})
		}
	}
	// Cross check that the two tries are in sync
	checkTrieContents(t, triedb, srcTrie.Hash().Bytes(), srcData)
}

// Tests that the trie scheduler can correctly reconstruct the state even if only
// partial results are returned, and the others sent only later.
func TestIterativeDelayedSync(t *testing.T) {
	// Create a random trie to copy
	srcDb, srcTrie, srcData := makeTestTrie()

	// Create a destination trie and sync with the scheduler
	diskdb := memorydb.New()
	triedb := NewDatabase(diskdb)
	sched := NewSync(srcTrie.Hash(), diskdb, nil)

	// The code requests are ignored here since there is no code
	// at the testing trie.
	paths, nodes, _ := sched.Missing(10000)
	var elements []trieElement
	for i := 0; i < len(paths); i++ {
		elements = append(elements, trieElement{
			path:     paths[i],
			hash:     nodes[i],
			syncPath: NewSyncPath([]byte(paths[i])),
		})
	}
	for len(elements) > 0 {
		// Sync only half of the scheduled nodes
		results := make([]NodeSyncResult, len(elements)/2+1)
		for i, element := range elements[:len(results)] {
			data, err := srcDb.Node(element.hash)
			if err != nil {
				t.Fatalf("failed to retrieve node data for %x: %v", element.hash, err)
			}
			results[i] = NodeSyncResult{element.path, data}
		}
		for _, result := range results {
			if err := sched.ProcessNode(result); err != nil {
				t.Fatalf("failed to process result %v", err)
			}
		}
		batch := diskdb.NewBatch()
		if err := sched.Commit(batch); err != nil {
			t.Fatalf("failed to commit data: %v", err)
		}
		batch.Write()

		paths, nodes, _ = sched.Missing(10000)
		elements = elements[len(results):]
		for i := 0; i < len(paths); i++ {
			elements = append(elements, trieElement{
				path:     paths[i],
				hash:     nodes[i],
				syncPath: NewSyncPath([]byte(paths[i])),
			})
		}
	}
	// Cross check that the two tries are in sync
	checkTrieContents(t, triedb, srcTrie.Hash().Bytes(), srcData)
}

// Tests that given a root hash, a trie can sync iteratively on a single thread,
// requesting retrieval tasks and returning all of them in one go, however in a
// random order.
func TestIterativeRandomSyncIndividual(t *testing.T) { testIterativeRandomSync(t, 1) }
func TestIterativeRandomSyncBatched(t *testing.T)    { testIterativeRandomSync(t, 100) }

func testIterativeRandomSync(t *testing.T, count int) {
	// Create a random trie to copy
	srcDb, srcTrie, srcData := makeTestTrie()

	// Create a destination trie and sync with the scheduler
	diskdb := memorydb.New()
	triedb := NewDatabase(diskdb)
	sched := NewSync(srcTrie.Hash(), diskdb, nil)

	// The code requests are ignored here since there is no code
	// at the testing trie.
	paths, nodes, _ := sched.Missing(count)
	queue := make(map[string]trieElement)
	for i, path := range paths {
		queue[path] = trieElement{
			path:     paths[i],
			hash:     nodes[i],
			syncPath: NewSyncPath([]byte(paths[i])),
		}
	}
	for len(queue) > 0 {
		// Fetch all the queued nodes in a random order
		results := make([]NodeSyncResult, 0, len(queue))
		for path, element := range queue {
			data, err := srcDb.Node(element.hash)
			if err != nil {
				t.Fatalf("failed to retrieve node data for %x: %v", element.hash, err)
			}
			results = append(results, NodeSyncResult{path, data})
		}
		// Feed the retrieved results back and queue new tasks
		for _, result := range results {
			if err := sched.ProcessNode(result); err != nil {
				t.Fatalf("failed to process result %v", err)
			}
		}
		batch := diskdb.NewBatch()
		if err := sched.Commit(batch); err != nil {
			t.Fatalf("failed to commit data: %v", err)
		}
		batch.Write()

		paths, nodes, _ = sched.Missing(count)
		queue = make(map[string]trieElement)
		for i, path := range paths {
			queue[path] = trieElement{
				path:     path,
				hash:     nodes[i],
				syncPath: NewSyncPath([]byte(path)),
			}
		}
	}
	// Cross check that the two tries are in sync
	checkTrieContents(t, triedb, srcTrie.Hash().Bytes(), srcData)
}

// Tests that the trie scheduler can correctly reconstruct the state even if only
// partial results are returned (Even those randomly), others sent only later.
func TestIterativeRandomDelayedSync(t *testing.T) {
	// Create a random trie to copy
	srcDb, srcTrie, srcData := makeTestTrie()

	// Create a destination trie and sync with the scheduler
	diskdb := memorydb.New()
	triedb := NewDatabase(diskdb)
	sched := NewSync(srcTrie.Hash(), diskdb, nil)

	// The code requests are ignored here since there is no code
	// at the testing trie.
	paths, nodes, _ := sched.Missing(10000)
	queue := make(map[string]trieElement)
	for i, path := range paths {
		queue[path] = trieElement{
			path:     path,
			hash:     nodes[i],
			syncPath: NewSyncPath([]byte(path)),
		}
	}
	for len(queue) > 0 {
		// Sync only half of the scheduled nodes, even those in random order
		results := make([]NodeSyncResult, 0, len(queue)/2+1)
		for path, element := range queue {
			data, err := srcDb.Node(element.hash)
			if err != nil {
				t.Fatalf("failed to retrieve node data for %x: %v", element.hash, err)
			}
			results = append(results, NodeSyncResult{path, data})

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
		batch := diskdb.NewBatch()
		if err := sched.Commit(batch); err != nil {
			t.Fatalf("failed to commit data: %v", err)
		}
		batch.Write()
		for _, result := range results {
			delete(queue, result.Path)
		}
		paths, nodes, _ = sched.Missing(10000)
		for i, path := range paths {
			queue[path] = trieElement{
				path:     path,
				hash:     nodes[i],
				syncPath: NewSyncPath([]byte(path)),
			}
		}
	}
	// Cross check that the two tries are in sync
	checkTrieContents(t, triedb, srcTrie.Hash().Bytes(), srcData)
}

// Tests that a trie sync will not request nodes multiple times, even if they
// have such references.
func TestDuplicateAvoidanceSync(t *testing.T) {
	// Create a random trie to copy
	srcDb, srcTrie, srcData := makeTestTrie()

	// Create a destination trie and sync with the scheduler
	diskdb := memorydb.New()
	triedb := NewDatabase(diskdb)
	sched := NewSync(srcTrie.Hash(), diskdb, nil)

	// The code requests are ignored here since there is no code
	// at the testing trie.
	paths, nodes, _ := sched.Missing(0)
	var elements []trieElement
	for i := 0; i < len(paths); i++ {
		elements = append(elements, trieElement{
			path:     paths[i],
			hash:     nodes[i],
			syncPath: NewSyncPath([]byte(paths[i])),
		})
	}
	requested := make(map[common.Hash]struct{})

	for len(elements) > 0 {
		results := make([]NodeSyncResult, len(elements))
		for i, element := range elements {
			data, err := srcDb.Node(element.hash)
			if err != nil {
				t.Fatalf("failed to retrieve node data for %x: %v", element.hash, err)
			}
			if _, ok := requested[element.hash]; ok {
				t.Errorf("hash %x already requested once", element.hash)
			}
			requested[element.hash] = struct{}{}

			results[i] = NodeSyncResult{element.path, data}
		}
		for _, result := range results {
			if err := sched.ProcessNode(result); err != nil {
				t.Fatalf("failed to process result %v", err)
			}
		}
		batch := diskdb.NewBatch()
		if err := sched.Commit(batch); err != nil {
			t.Fatalf("failed to commit data: %v", err)
		}
		batch.Write()

		paths, nodes, _ = sched.Missing(0)
		elements = elements[:0]
		for i := 0; i < len(paths); i++ {
			elements = append(elements, trieElement{
				path:     paths[i],
				hash:     nodes[i],
				syncPath: NewSyncPath([]byte(paths[i])),
			})
		}
	}
	// Cross check that the two tries are in sync
	checkTrieContents(t, triedb, srcTrie.Hash().Bytes(), srcData)
}

// Tests that at any point in time during a sync, only complete sub-tries are in
// the database.
func TestIncompleteSync(t *testing.T) {
	// Create a random trie to copy
	srcDb, srcTrie, _ := makeTestTrie()

	// Create a destination trie and sync with the scheduler
	diskdb := memorydb.New()
	triedb := NewDatabase(diskdb)
	sched := NewSync(srcTrie.Hash(), diskdb, nil)

	// The code requests are ignored here since there is no code
	// at the testing trie.
	var (
		added    []common.Hash
		elements []trieElement
		root     = srcTrie.Hash()
	)
	paths, nodes, _ := sched.Missing(1)
	for i := 0; i < len(paths); i++ {
		elements = append(elements, trieElement{
			path:     paths[i],
			hash:     nodes[i],
			syncPath: NewSyncPath([]byte(paths[i])),
		})
	}
	for len(elements) > 0 {
		// Fetch a batch of trie nodes
		results := make([]NodeSyncResult, len(elements))
		for i, element := range elements {
			data, err := srcDb.Node(element.hash)
			if err != nil {
				t.Fatalf("failed to retrieve node data for %x: %v", element.hash, err)
			}
			results[i] = NodeSyncResult{element.path, data}
		}
		// Process each of the trie nodes
		for _, result := range results {
			if err := sched.ProcessNode(result); err != nil {
				t.Fatalf("failed to process result %v", err)
			}
		}
		batch := diskdb.NewBatch()
		if err := sched.Commit(batch); err != nil {
			t.Fatalf("failed to commit data: %v", err)
		}
		batch.Write()

		for _, result := range results {
			hash := crypto.Keccak256Hash(result.Data)
			if hash != root {
				added = append(added, hash)
			}
			// Check that all known sub-tries in the synced trie are complete
			if err := checkTrieConsistency(triedb, hash); err != nil {
				t.Fatalf("trie inconsistent: %v", err)
			}
		}
		// Fetch the next batch to retrieve
		paths, nodes, _ = sched.Missing(1)
		elements = elements[:0]
		for i := 0; i < len(paths); i++ {
			elements = append(elements, trieElement{
				path:     paths[i],
				hash:     nodes[i],
				syncPath: NewSyncPath([]byte(paths[i])),
			})
		}
	}
	// Sanity check that removing any node from the database is detected
	for _, hash := range added {
		value, _ := diskdb.Get(hash.Bytes())
		diskdb.Delete(hash.Bytes())
		if err := checkTrieConsistency(triedb, root); err == nil {
			t.Fatalf("trie inconsistency not caught, missing: %x", hash)
		}
		diskdb.Put(hash.Bytes(), value)
	}
}

// Tests that trie nodes get scheduled lexicographically when having the same
// depth.
func TestSyncOrdering(t *testing.T) {
	// Create a random trie to copy
	srcDb, srcTrie, srcData := makeTestTrie()

	// Create a destination trie and sync with the scheduler, tracking the requests
	diskdb := memorydb.New()
	triedb := NewDatabase(diskdb)
	sched := NewSync(srcTrie.Hash(), diskdb, nil)

	// The code requests are ignored here since there is no code
	// at the testing trie.
	var (
		reqs     []SyncPath
		elements []trieElement
	)
	paths, nodes, _ := sched.Missing(1)
	for i := 0; i < len(paths); i++ {
		elements = append(elements, trieElement{
			path:     paths[i],
			hash:     nodes[i],
			syncPath: NewSyncPath([]byte(paths[i])),
		})
		reqs = append(reqs, NewSyncPath([]byte(paths[i])))
	}

	for len(elements) > 0 {
		results := make([]NodeSyncResult, len(elements))
		for i, element := range elements {
			data, err := srcDb.Node(element.hash)
			if err != nil {
				t.Fatalf("failed to retrieve node data for %x: %v", element.hash, err)
			}
			results[i] = NodeSyncResult{element.path, data}
		}
		for _, result := range results {
			if err := sched.ProcessNode(result); err != nil {
				t.Fatalf("failed to process result %v", err)
			}
		}
		batch := diskdb.NewBatch()
		if err := sched.Commit(batch); err != nil {
			t.Fatalf("failed to commit data: %v", err)
		}
		batch.Write()

		paths, nodes, _ = sched.Missing(1)
		elements = elements[:0]
		for i := 0; i < len(paths); i++ {
			elements = append(elements, trieElement{
				path:     paths[i],
				hash:     nodes[i],
				syncPath: NewSyncPath([]byte(paths[i])),
			})
			reqs = append(reqs, NewSyncPath([]byte(paths[i])))
		}
	}
	// Cross check that the two tries are in sync
	checkTrieContents(t, triedb, srcTrie.Hash().Bytes(), srcData)

	// Check that the trie nodes have been requested path-ordered
	for i := 0; i < len(reqs)-1; i++ {
		if len(reqs[i]) > 1 || len(reqs[i+1]) > 1 {
			// In the case of the trie tests, there's no storage so the tuples
			// must always be single items. 2-tuples should be tested in state.
			t.Errorf("Invalid request tuples: len(%v) or len(%v) > 1", reqs[i], reqs[i+1])
		}
		if bytes.Compare(compactToHex(reqs[i][0]), compactToHex(reqs[i+1][0])) > 0 {
			t.Errorf("Invalid request order: %v before %v", compactToHex(reqs[i][0]), compactToHex(reqs[i+1][0]))
		}
	}
}
