// Copyright 2021 The go-ethereum Authors
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
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
)

// testEnv is the environment for all test fields.
type testEnv struct {
	db      *Database
	numbers []uint64
	roots   []common.Hash
	keys    [][]string
	vals    [][][]byte
}

// fill randomly creates the nodes for layer and commits
// the constructed layer into the given database.
func fill(db *Database, n int, testkeys [][]string, testvals [][][]byte, parentHash common.Hash, parentBlob []byte, parentNumber uint64) (common.Hash, []byte, uint64, []string, [][]byte) {
	var (
		keys  []string
		vals  [][]byte
		nodes = make(map[string]*nodeWithPreValue)
	)
	for i := 0; i < n; i++ {
		var (
			storage []byte
			node    *nodeWithPreValue
		)
		switch rand.Intn(3) {
		case 0:
			// node creation
			storage = EncodeStorageKey(common.Hash{}, randomHash().Bytes())
			node = &nodeWithPreValue{
				cachedNode: randomNode(),
				pre:        nil,
			}
		case 1:
			// node modification
			if parentNumber == 0 {
				continue
			}
			pkeys := testkeys[parentNumber-1]
			if len(pkeys) == 0 {
				continue
			}
			index := rand.Intn(len(pkeys))
			storage = []byte(pkeys[index])

			node = &nodeWithPreValue{
				cachedNode: randomNode(),
				pre:        testvals[parentNumber-1][index],
			}
		case 2:
			// node deletion
			if parentNumber == 0 {
				continue
			}
			pkeys, pvals := testkeys[parentNumber-1], testvals[parentNumber-1]
			if len(pkeys) == 0 {
				continue
			}
			index := rand.Intn(len(pkeys))
			if len(pvals[index]) == 0 {
				continue
			}
			storage = []byte(pkeys[index])

			node = &nodeWithPreValue{
				cachedNode: randomEmptyNode(),
				pre:        pvals[index],
			}
		}
		// Don't add duplicated updates
		if _, ok := nodes[string(storage)]; ok {
			continue
		}
		nodes[string(storage)] = node
		keys = append(keys, string(storage))

		if node.node == nil {
			vals = append(vals, nil)
		} else {
			vals = append(vals, common.CopyBytes(node.rlp()))
		}
	}
	// Add the root node
	root := randomNode()
	nodes[string(EncodeStorageKey(common.Hash{}, nil))] = &nodeWithPreValue{
		cachedNode: root,
		pre:        parentBlob,
	}
	db.Commit(root.hash, parentHash, &NodeSet{nodes: nodes})
	return root.hash, root.rlp(), parentNumber + 1, keys, vals
}

func fillDB(t *testing.T) *testEnv {
	diskdb, _ := rawdb.NewDatabaseWithFreezer(rawdb.NewMemoryDatabase(), t.TempDir(), "", false)
	var (
		db       = NewDatabase(diskdb, nil)
		numbers  []uint64
		roots    []common.Hash
		testKeys [][]string
		testVals [][][]byte
	)
	// Construct a database with enough reverse diffs stored
	var (
		parent     common.Hash
		parentBlob []byte
		number     uint64
		keys       []string
		vals       [][]byte
	)
	for i := 0; i < 2*128; i++ {
		parent, parentBlob, number, keys, vals = fill(db, 300, testKeys, testVals, parent, parentBlob, uint64(i))
		numbers = append(numbers, number)
		roots = append(roots, parent)
		testKeys = append(testKeys, keys)
		testVals = append(testVals, vals)
	}
	return &testEnv{
		db:      db,
		numbers: numbers,
		roots:   roots,
		keys:    testKeys,
		vals:    testVals,
	}
}

func TestDatabaseRollback(t *testing.T) {
	defer func(origin uint64) {
		cacheSizeLimit = origin
	}(cacheSizeLimit)
	cacheSizeLimit = 1024 * 256 // Lower the dirty cache size

	var (
		env    = fillDB(t)
		snapdb = env.db.backend.(*snapDatabase)
		dl     = snapdb.tree.bottom().(*diskLayer)
		index  int
	)
	for index = 0; index < len(env.roots); index++ {
		if env.roots[index] == dl.root {
			break
		}
	}
	// Ensure all the reverse diffs are stored properly
	var parent = emptyRoot
	for i := 0; i <= index; i++ {
		diff, err := loadReverseDiff(snapdb.freezer, uint64(i+1))
		if err != nil {
			t.Errorf("Failed to load reverse diff, index %d, err %v", i+1, err)
		}
		if diff.Parent != parent {
			t.Error("Reverse diff is not continuous")
		}
		parent = diff.Root
	}
	// Ensure immature reverse diffs are not persisted
	for i := index + 1; i < len(env.numbers); i++ {
		blob := rawdb.ReadReverseDiff(env.db.diskdb, uint64(i+1))
		if len(blob) != 0 {
			t.Error("Unexpected reverse diff", "index", i)
		}
	}
	// Revert the db to historical point with reverse state available
	for i := index; i > 0; i-- {
		if err := env.db.Recover(env.roots[i-1]); err != nil {
			t.Error("Failed to revert db status", "err", err)
		}
		dl := snapdb.tree.bottom().(*diskLayer)
		if dl.Root() != env.roots[i-1] {
			t.Error("Unexpected disk layer root")
		}
		// Compare the reverted state with the constructed one, they should be same.
		keys, vals := env.keys[i-1], env.vals[i-1]
		for j := 0; j < len(keys); j++ {
			layer := env.db.GetReader(env.roots[i-1])
			if len(vals[j]) == 0 {
				// deleted node, expect error
				blob, _ := layer.NodeBlob([]byte(keys[j]), crypto.Keccak256Hash(vals[j])) // error can occur
				if len(blob) != 0 {
					t.Error("Unexpected state", "key", []byte(keys[j]), "got", blob)
				}
			} else {
				// normal node, expect correct value
				blob, err := layer.NodeBlob([]byte(keys[j]), crypto.Keccak256Hash(vals[j]))
				if err != nil {
					t.Error("Failed to retrieve state", "err", err)
				}
				if !bytes.Equal(blob, vals[j]) {
					t.Error("Unexpected state", "key", []byte(keys[j]), "want", vals[j], "got", blob)
				}
			}
		}
	}
	if snapdb.tree.len() != 1 {
		t.Error("Only disk layer is expected")
	}
}

func TestDatabaseBatchRollback(t *testing.T) {
	defer func(origin uint64) {
		cacheSizeLimit = origin
	}(cacheSizeLimit)
	cacheSizeLimit = 1024 * 256 // Lower the dirty cache size

	var (
		env    = fillDB(t)
		snapdb = env.db.backend.(*snapDatabase)
		dl     = snapdb.tree.bottom().(*diskLayer)
		index  int
	)
	for index = 0; index < len(env.roots); index++ {
		if env.roots[index] == dl.root {
			break
		}
	}
	// Revert the db to historical point with reverse state available
	if err := env.db.Recover(common.Hash{}); err != nil {
		t.Error("Failed to revert db status", "err", err)
	}
	ndl := snapdb.tree.bottom().(*diskLayer)
	if ndl.Root() != emptyRoot {
		t.Error("Unexpected disk layer root")
	}
	if snapdb.tree.len() != 1 {
		t.Error("Only disk layer is expected")
	}
	// Ensure all the states are deleted by reverting.
	for i, keys := range env.keys {
		vals := env.vals[i]
		for j, key := range keys {
			if len(vals[j]) == 0 {
				continue
			}
			hash := crypto.Keccak256Hash(vals[j])
			blob, _ := ndl.NodeBlob([]byte(key), hash)
			if len(blob) != 0 {
				t.Error("Unexpected state")
			}
		}
	}
}

func TestDatabaseRecoverable(t *testing.T) {
	defer func(origin uint64) {
		cacheSizeLimit = origin
	}(cacheSizeLimit)
	cacheSizeLimit = 1024 * 256 // Lower the dirty cache size

	var (
		env    = fillDB(t)
		snapdb = env.db.backend.(*snapDatabase)
		dl     = snapdb.tree.bottom().(*diskLayer)
		index  int
	)
	for index = 0; index < len(env.roots); index++ {
		if env.roots[index] == dl.root {
			break
		}
	}
	// Empty state should be recoverable
	if !env.db.Recoverable(common.Hash{}) {
		t.Error("Layer unrecoverable")
	}
	// All the states below the disk layer should be recoverable.
	for i := 0; i < index; i++ {
		if !env.db.Recoverable(env.roots[i]) {
			t.Error("Layer unrecoverable")
		}
	}
	// All other layers above(including disk layer) shouldn't be
	// recoverable since they are accessible.
	for i := index + 1; i < len(env.numbers); i++ {
		if env.db.Recoverable(env.roots[i]) {
			t.Error("Layer should be unrecoverable")
		}
	}
}

func TestClose(t *testing.T) {
	defer func(origin uint64) {
		cacheSizeLimit = origin
	}(cacheSizeLimit)
	cacheSizeLimit = 1024 * 256 // Lower the dirty cache size

	var (
		env    = fillDB(t)
		snapdb = env.db.backend.(*snapDatabase)
		dl     = snapdb.tree.bottom().(*diskLayer)
		index  int
	)
	if err := env.db.Close(env.roots[len(env.roots)-1]); err != nil {
		t.Error("Failed to journal triedb", "err", err)
	}
	newdb := NewDatabase(env.db.diskdb, env.db.config)
	for index = 0; index < len(env.roots); index++ {
		if env.roots[index] == dl.root {
			break
		}
	}
	for i := index; i < len(env.numbers); i++ {
		keys, vals := env.keys[i], env.vals[i]
		for j := 0; j < len(keys); j++ {
			if vals[j] == nil {
				continue
			}
			layer := newdb.GetReader(env.roots[i])
			blob, err := layer.NodeBlob([]byte(keys[j]), crypto.Keccak256Hash(vals[j]))
			if err != nil {
				t.Error("Failed to retrieve state", "err", err)
			}
			if !bytes.Equal(blob, vals[j]) {
				t.Error("Unexpected state", "key", []byte(keys[j]), "want", vals[j], "got", blob)
			}
		}
	}
}

func TestReset(t *testing.T) {
	defer func(origin uint64) {
		cacheSizeLimit = origin
	}(cacheSizeLimit)
	cacheSizeLimit = 1024 * 256 // Lower the dirty cache size

	var (
		env    = fillDB(t)
		snapdb = env.db.backend.(*snapDatabase)
		dl     = snapdb.tree.bottom().(*diskLayer)
		index  int
	)
	for index = 0; index < len(env.roots); index++ {
		if env.roots[index] == dl.root {
			break
		}
	}
	// Reset database to non-existent target, should reject it
	if err := env.db.Reset(randomHash()); err == nil {
		t.Fatal("Failed to reject invalid reset")
	}
	// Reset database to state persisted in the disk
	_, hash := rawdb.ReadTrieNode(env.db.DiskDB(), EncodeStorageKey(common.Hash{}, nil))
	if err := env.db.Reset(hash); err != nil {
		t.Fatalf("Failed to reset database %v", err)
	}
	// Ensure journal is deleted from disk
	if blob := rawdb.ReadTrieJournal(env.db.DiskDB()); len(blob) != 0 {
		t.Fatal("Failed to clean journal")
	}
	// Ensure all reverse diffs are nuked
	for i := 0; i <= index; i++ {
		_, err := loadReverseDiff(snapdb.freezer, uint64(i+1))
		if err == nil {
			t.Fatalf("Failed to clean reverse diff, index %d", i+1)
		}
	}
	// Ensure there is only a single disk layer kept, hash should
	// be matched as well.
	if snapdb.tree.len() != 1 {
		t.Fatalf("Extra layer kept %d", snapdb.tree.len())
	}
	if snapdb.tree.bottom().Root() != hash {
		t.Fatalf("Root hash is not matched exp %x got %x", hash, snapdb.tree.bottom().Root())
	}
}

func TestCap(t *testing.T) {
	defer func(origin uint64) {
		cacheSizeLimit = origin
	}(cacheSizeLimit)
	cacheSizeLimit = 1024 * 256 // Lower the dirty cache size

	var (
		env    = fillDB(t)
		snapdb = env.db.backend.(*snapDatabase)
	)
	if err := snapdb.Cap(env.roots[len(env.roots)-1], 0); err != nil {
		t.Fatalf("Failed to cap database %v", err)
	}
	// Ensure there is only a single layer kept
	if snapdb.tree.len() != 1 {
		t.Fatalf("Extra layer kept %d", snapdb.tree.len())
	}
	if snapdb.tree.bottom().Root() != env.roots[len(env.roots)-1] {
		t.Fatalf("Root hash is not matched exp %x got %x", env.roots[len(env.roots)-1], snapdb.tree.bottom().Root())
	}
	_, hash := rawdb.ReadTrieNode(env.db.DiskDB(), EncodeStorageKey(common.Hash{}, nil))
	if hash != env.roots[len(env.roots)-1] {
		t.Fatalf("Root hash is not matched exp %x got %x", env.roots[len(env.roots)-1], hash)
	}
}
