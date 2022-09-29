// Copyright 2022 The go-ethereum Authors
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
	"github.com/ethereum/go-ethereum/ethdb"
)

// testEnv is the environment for all test fields.
type testEnv struct {
	db     ethdb.Database
	nodeDb *Database
	roots  []common.Hash
	paths  [][][]byte
	blobs  [][][]byte
}

// fill creates a list of random nodes for simulation.
func fill(n int, prevPaths [][][]byte, prevBlobs [][][]byte) (common.Hash, *NodeSet) {
	var (
		nodes    = NewNodeSet(common.Hash{}, nil)
		checkDup = func(path []byte) bool {
			if len(path) == 0 {
				return true
			}
			if _, ok := nodes.nodes[string(path)]; ok {
				return true
			}
			return false
		}
	)
	for i := 0; i < n; i++ {
		switch rand.Intn(3) {
		case 0:
			// node creation
			path := randomHash().Bytes()
			if checkDup(path) {
				continue
			}
			nodes.markUpdated(path, randomNode())
		case 1:
			// node modification
			if len(prevPaths) == 0 {
				continue
			}
			paths := prevPaths[len(prevPaths)-1]
			if len(paths) == 0 {
				continue
			}
			index := rand.Intn(len(paths))
			path := paths[index]
			if checkDup(path) {
				continue
			}
			nodes.markUpdated(path, randomNode())
		case 2:
			// node deletion
			if len(prevPaths) == 0 {
				continue
			}
			paths, blobs := prevPaths[len(prevPaths)-1], prevBlobs[len(prevBlobs)-1]
			if len(paths) == 0 {
				continue
			}
			index := rand.Intn(len(paths))
			if len(blobs[index]) == 0 {
				continue
			}
			path := paths[index]
			if checkDup(path) {
				continue
			}
			nodes.markDeleted(path)
		}
	}
	// Add the root node
	root := randomNode()
	nodes.markUpdated(nil, root)
	return root.hash, nodes
}

func fillDB(t *testing.T) *testEnv {
	var (
		diskdb, _ = rawdb.NewDatabaseWithFreezer(rawdb.NewMemoryDatabase(), t.TempDir(), "", false)
		nodeDb    = newTestDatabase(diskdb, rawdb.PathScheme)
		roots     []common.Hash
		paths     [][][]byte
		blobs     [][][]byte
		parent    common.Hash
	)
	// Construct a database with enough trie histories stored
	for i := 0; i < 2*128; i++ {
		var (
			pathlist [][]byte
			bloblist [][]byte
		)
		root, set := fill(500, paths, blobs)
		roots = append(roots, root)

		set.forEachWithOrder(false, func(path string, n *memoryNode) {
			pathlist = append(pathlist, []byte(path))
			if n.isDeleted() {
				bloblist = append(bloblist, nil)
			} else {
				bloblist = append(bloblist, set.nodes[path].rlp())
			}
		})
		paths = append(paths, pathlist)
		blobs = append(blobs, bloblist)

		nodeDb.Update(root, parent, NewWithNodeSet(set))
		parent = root
	}
	return &testEnv{
		db:     diskdb,
		nodeDb: nodeDb,
		roots:  roots,
		paths:  paths,
		blobs:  blobs,
	}
}

func TestDatabaseRollback(t *testing.T) {
	defer func(origin int) {
		defaultCacheSize = origin
	}(defaultCacheSize)
	defaultCacheSize = 1024 * 256 // Lower the dirty cache size

	var (
		env   = fillDB(t)
		db    = env.nodeDb.backend.(*snapDatabase)
		dl    = db.tree.bottom().(*diskLayer)
		index int
	)
	for index = 0; index < len(env.roots); index++ {
		if env.roots[index] == dl.root {
			break
		}
	}
	// Ensure all the trie histories are stored properly
	var parent = emptyRoot
	for i := uint64(1); i <= dl.id; i++ {
		h, err := loadTrieHistory(db.freezer, i)
		if err != nil {
			t.Errorf("Failed to load trie history, index %d, err %v", i, err)
		}
		if h.Parent != parent {
			t.Error("Trie history is not continuous")
		}
		parent = h.Root
	}
	// Ensure immature trie histories are not persisted
	for i := dl.id + 1; i <= uint64(len(env.roots)); i++ {
		blob := rawdb.ReadTrieHistory(env.nodeDb.diskdb, i)
		if len(blob) != 0 {
			t.Error("Unexpected trie history", "id", i)
		}
	}
	// Revert the db to historical point with reverse state available
	for i := index; i > 0; i-- {
		if err := env.nodeDb.Recover(env.roots[i-1]); err != nil {
			t.Error("Failed to revert db status", "err", err)
		}
		if db.tree.bottom().Root() != env.roots[i-1] {
			t.Error("Unexpected disk layer root")
		}
		// Compare the reverted state with the constructed one, they should be same.
		paths, blobs := env.paths[i-1], env.blobs[i-1]
		for j := 0; j < len(paths); j++ {
			layer := env.nodeDb.GetReader(env.roots[i-1])
			if len(blobs[j]) == 0 {
				// deleted node, expect error
				blob, _ := layer.NodeBlob(common.Hash{}, paths[j], crypto.Keccak256Hash(blobs[j]))
				if len(blob) != 0 {
					t.Error("Unexpected state", "path", paths[j], "got", blob)
				}
			} else {
				// normal node, expect correct value
				blob, err := layer.NodeBlob(common.Hash{}, paths[j], crypto.Keccak256Hash(blobs[j]))
				if err != nil {
					t.Error("Failed to retrieve state", "err", err)
				}
				if !bytes.Equal(blob, blobs[j]) {
					t.Error("Unexpected state", "path", paths[j], "want", blobs[j], "got", blob)
				}
			}
		}
	}
	if db.tree.len() != 1 {
		t.Error("Only disk layer is expected")
	}
}

func TestDatabaseBatchRollback(t *testing.T) {
	defer func(origin int) {
		defaultCacheSize = origin
	}(defaultCacheSize)
	defaultCacheSize = 1024 * 256 // Lower the dirty cache size

	var (
		env = fillDB(t)
		db  = env.nodeDb.backend.(*snapDatabase)
	)
	// Revert the db to historical point with all trie histories available
	if err := env.nodeDb.Recover(common.Hash{}); err != nil {
		t.Error("Failed to revert db", "err", err)
	}
	ndl := db.tree.bottom().(*diskLayer)
	if ndl.Root() != emptyRoot {
		t.Error("Unexpected disk layer root")
	}
	if db.tree.len() != 1 {
		t.Error("Only disk layer is expected")
	}
	// Ensure all the states are deleted by reverting.
	for i, paths := range env.paths {
		blobs := env.blobs[i]
		for j, path := range paths {
			if len(blobs[j]) == 0 {
				continue
			}
			hash := crypto.Keccak256Hash(blobs[j])
			blob, _ := ndl.NodeBlob(common.Hash{}, path, hash)
			if len(blob) != 0 {
				t.Error("Unexpected state")
			}
		}
	}
	// Ensure all lookups and trie histories are cleaned up
	number, err := db.freezer.Ancients()
	if err != nil {
		t.Fatalf("Failed to retrieve ancient items")
	}
	if number != 0 {
		t.Fatalf("Unexpected trie histories")
	}
	for i := 0; i < len(env.roots); i++ {
		_, exist := rawdb.ReadStateLookup(db.diskdb, env.roots[i])
		if exist {
			t.Fatalf("Unexpected lookup")
		}
	}
}

func TestDatabaseRecoverable(t *testing.T) {
	defer func(origin int) {
		defaultCacheSize = origin
	}(defaultCacheSize)
	defaultCacheSize = 1024 * 256 // Lower the dirty cache size

	var (
		env   = fillDB(t)
		db    = env.nodeDb.backend.(*snapDatabase)
		dl    = db.tree.bottom()
		index int
	)
	for index = 0; index < len(env.roots); index++ {
		if env.roots[index] == dl.Root() {
			break
		}
	}
	// Initial state should be recoverable
	result, _ := env.nodeDb.Recoverable(common.Hash{})
	if !result {
		t.Error("Layer unrecoverable")
	}
	// All the states below the disk layer should be recoverable.
	for i := 0; i < index; i++ {
		result, _ := env.nodeDb.Recoverable(env.roots[i])
		if !result {
			t.Error("Layer unrecoverable")
		}
	}
	// All other layers above(including disk layer) shouldn't be
	// recoverable since they are accessible.
	for i := index + 1; i < len(env.roots); i++ {
		result, _ := env.nodeDb.Recoverable(env.roots[i])
		if result {
			t.Error("Layer should be unrecoverable")
		}
	}
}

func TestJournal(t *testing.T) {
	defer func(origin int) {
		defaultCacheSize = origin
	}(defaultCacheSize)
	defaultCacheSize = 1024 * 256 // Lower the dirty cache size

	var (
		env   = fillDB(t)
		db    = env.nodeDb.backend.(*snapDatabase)
		dl    = db.tree.bottom()
		index int
	)
	if err := env.nodeDb.Journal(env.roots[len(env.roots)-1]); err != nil {
		t.Error("Failed to journal triedb", "err", err)
	}
	env.nodeDb.Close()

	newdb := newTestDatabase(env.nodeDb.diskdb, rawdb.PathScheme)
	for index = 0; index < len(env.roots); index++ {
		if env.roots[index] == dl.Root() {
			break
		}
	}
	for i := index; i < len(env.roots); i++ {
		paths, blobs := env.paths[i], env.blobs[i]
		for j := 0; j < len(paths); j++ {
			if blobs[j] == nil {
				continue
			}
			layer := newdb.GetReader(env.roots[i])
			blob, err := layer.NodeBlob(common.Hash{}, paths[j], crypto.Keccak256Hash(blobs[j]))
			if err != nil {
				t.Error("Failed to retrieve state", "err", err)
			}
			if !bytes.Equal(blob, blobs[j]) {
				t.Error("Unexpected state", "path", paths[j], "want", blobs[j], "got", blob)
			}
		}
	}
}

func TestReset(t *testing.T) {
	defer func(origin int) {
		defaultCacheSize = origin
	}(defaultCacheSize)
	defaultCacheSize = 1024 * 256 // Lower the dirty cache size

	var (
		env   = fillDB(t)
		db    = env.nodeDb.backend.(*snapDatabase)
		dl    = db.tree.bottom().(*diskLayer)
		index int
	)
	for index = 0; index < len(env.roots); index++ {
		if env.roots[index] == dl.root {
			break
		}
	}
	// Reset database to non-existent target, should reject it
	if err := env.nodeDb.Reset(randomHash()); err == nil {
		t.Fatal("Failed to reject invalid reset")
	}
	// Reset database to state persisted in the disk
	_, hash := rawdb.ReadAccountTrieNode(env.db, nil)
	if err := env.nodeDb.Reset(hash); err != nil {
		t.Fatalf("Failed to reset database %v", err)
	}
	// Ensure journal is deleted from disk
	if blob := rawdb.ReadTrieJournal(env.db); len(blob) != 0 {
		t.Fatal("Failed to clean journal")
	}
	// Ensure all trie histories are nuked
	for i := 0; i <= index; i++ {
		_, err := loadTrieHistory(db.freezer, uint64(i+1))
		if err == nil {
			t.Fatalf("Failed to clean trie history, index %d", i+1)
		}
	}
	// Ensure there is only a single disk layer kept, hash should
	// be matched as well.
	if db.tree.len() != 1 {
		t.Fatalf("Extra layer kept %d", db.tree.len())
	}
	if db.tree.bottom().Root() != hash {
		t.Fatalf("Root hash is not matched exp %x got %x", hash, db.tree.bottom().Root())
	}
}

func TestCommit(t *testing.T) {
	defer func(origin int) {
		defaultCacheSize = origin
	}(defaultCacheSize)
	defaultCacheSize = 1024 * 256 // Lower the dirty cache size

	var (
		env = fillDB(t)
		db  = env.nodeDb.backend.(*snapDatabase)
	)
	if err := db.Commit(env.roots[len(env.roots)-1], false); err != nil {
		t.Fatalf("Failed to cap database %v", err)
	}
	// Ensure there is only a single layer kept
	if db.tree.len() != 1 {
		t.Fatalf("Extra layer kept %d", db.tree.len())
	}
	if db.tree.bottom().Root() != env.roots[len(env.roots)-1] {
		t.Fatalf("Root hash is not matched exp %x got %x", env.roots[len(env.roots)-1], db.tree.bottom().Root())
	}
	_, hash := rawdb.ReadAccountTrieNode(env.db, nil)
	if hash != env.roots[len(env.roots)-1] {
		t.Fatalf("Root hash is not matched exp %x got %x", env.roots[len(env.roots)-1], hash)
	}
}
