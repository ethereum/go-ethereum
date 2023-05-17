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

package pathdb

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/VictoriaMetrics/fastcache"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/trie/testutil"
	"github.com/ethereum/go-ethereum/trie/trienode"
)

// testEnv is the environment for all test fields.
type testEnv struct {
	db     *Database
	hashes []common.Hash
	roots  [][]byte
	paths  [][][]byte
	blobs  [][][]byte
}

func (env *testEnv) lastRoot() []byte {
	if len(env.roots) == 0 {
		return nil
	}
	return env.roots[len(env.roots)-1]
}

func (env *testEnv) lastHash() common.Hash {
	if len(env.hashes) == 0 {
		return common.Hash{}
	}
	return env.hashes[len(env.hashes)-1]
}

func (env *testEnv) rand() (bool, []byte, []byte) {
	if len(env.paths) == 0 {
		return false, nil, nil
	}
	paths, blobs := env.paths[len(env.paths)-1], env.blobs[len(env.blobs)-1]
	index := rand.Intn(len(paths))
	return true, paths[index], blobs[index]
}

func (env *testEnv) bottomIndex() int {
	bottom := env.db.tree.bottom()
	for index := 0; index < len(env.hashes); index++ {
		if env.hashes[index] == bottom.Root() {
			return index
		}
	}
	return -1
}

// fill creates a list of random nodes for simulation.
func fill(n int, env *testEnv) {
	var (
		set      = trienode.NewNodeSet(common.Hash{})
		checkDup = func(path []byte) bool {
			if len(path) == 0 {
				return true
			}
			if _, ok := set.Nodes[string(path)]; ok {
				return true
			}
			return false
		}
	)
	for i := 0; i < n; i++ {
		switch rand.Intn(3) {
		case 0:
			// node creation
			path := testutil.RandBytes(32)
			if checkDup(path) {
				continue
			}
			set.Nodes[string(path)] = testutil.RandomNodeWithPrev(nil)
		case 1:
			// node modification
			valid, path, prev := env.rand()
			if !valid {
				continue
			}
			if checkDup(path) {
				continue
			}
			set.Nodes[string(path)] = testutil.RandomNodeWithPrev(prev)
		case 2:
			// node deletion
			valid, path, prev := env.rand()
			if !valid || len(prev) == 0 {
				continue
			}
			if checkDup(path) {
				continue
			}
			set.Nodes[string(path)] = trienode.NewWithPrev(common.Hash{}, nil, prev)
		}
	}
	// Add the root node
	root := testutil.RandomNodeWithPrev(env.lastRoot())
	set.Nodes[""] = root

	// Update sets into database
	env.db.Update(root.Hash, env.lastHash(), trienode.NewWithNodeSet(set))

	// Append the newly added nodes
	var (
		paths [][]byte
		blobs [][]byte
	)
	for path, n := range set.Nodes {
		paths = append(paths, []byte(path))
		blobs = append(blobs, n.Blob)
	}
	env.paths = append(env.paths, paths)
	env.blobs = append(env.blobs, blobs)
	env.hashes = append(env.hashes, root.Hash)
	env.roots = append(env.roots, root.Blob)
}

func newTestEnv(t *testing.T) *testEnv {
	var (
		disk, _ = rawdb.NewDatabaseWithFreezer(rawdb.NewMemoryDatabase(), t.TempDir(), "", false)
		db      = New(disk, fastcache.New(256*1024), &Config{DirtySize: 256 * 1024})
		env     = &testEnv{db: db}
	)
	for i := 0; i < 2*128; i++ {
		fill(500, env)
	}
	return env
}

func TestDatabaseRollback(t *testing.T) {
	var (
		env   = newTestEnv(t)
		dl    = env.db.tree.bottom()
		index = env.bottomIndex()
	)
	// Ensure all the trie histories are stored properly
	var parent = types.EmptyRootHash
	for i := uint64(1); i <= dl.ID(); i++ {
		h, err := loadTrieHistory(env.db.freezer, i)
		if err != nil {
			t.Errorf("Failed to load trie history, index %d, err %v", i, err)
		}
		if h.Parent != parent {
			t.Error("Trie history is not continuous")
		}
		parent = h.Root
	}
	// Ensure immature trie histories are not persisted
	for i := dl.ID() + 1; i <= uint64(len(env.roots)); i++ {
		blob := rawdb.ReadTrieHistory(env.db.diskdb, i)
		if len(blob) != 0 {
			t.Error("Unexpected trie history", "id", i)
		}
	}
	// Revert the db to historical point with reverse state available
	for i := index; i > 0; i-- {
		if err := env.db.Recover(env.hashes[i-1]); err != nil {
			t.Error("Failed to revert db status", "err", err)
		}
		if env.db.tree.bottom().Root() != env.hashes[i-1] {
			t.Error("Unexpected disk layer root")
		}
		// Compare the reverted state with the constructed one, they should be same.
		paths, blobs := env.paths[i-1], env.blobs[i-1]
		for j := 0; j < len(paths); j++ {
			layer := env.db.Reader(env.hashes[i-1])
			if len(blobs[j]) == 0 {
				// deleted node, expect error
				blob, _ := layer.Node(common.Hash{}, paths[j], crypto.Keccak256Hash(blobs[j]))
				if len(blob) != 0 {
					t.Error("Unexpected state", "path", paths[j], "got", blob)
				}
			} else {
				// normal node, expect correct value
				blob, err := layer.Node(common.Hash{}, paths[j], crypto.Keccak256Hash(blobs[j]))
				if err != nil {
					t.Error("Failed to retrieve state", "err", err)
				}
				if !bytes.Equal(blob, blobs[j]) {
					t.Error("Unexpected state", "path", paths[j], "want", blobs[j], "got", blob)
				}
			}
		}
	}
	if env.db.tree.len() != 1 {
		t.Error("Only disk layer is expected")
	}
}

func TestDatabaseBatchRollback(t *testing.T) {
	env := newTestEnv(t)
	if err := env.db.Recover(common.Hash{}); err != nil {
		t.Error("Failed to revert db", "err", err)
	}
	ndl := env.db.tree.bottom()
	if ndl.Root() != types.EmptyRootHash {
		t.Error("Unexpected disk layer root")
	}
	if env.db.tree.len() != 1 {
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
			blob, _ := ndl.Node(common.Hash{}, path, hash)
			if len(blob) != 0 {
				t.Fatal("Unexpected state", blob)
			}
		}
	}
	// Ensure all lookups and trie histories are cleaned up
	number, err := env.db.freezer.Ancients()
	if err != nil {
		t.Fatalf("Failed to retrieve ancient items")
	}
	if number != 0 {
		t.Fatalf("Unexpected trie histories")
	}
	for i := 0; i < len(env.roots); i++ {
		_, exist := rawdb.ReadStateID(env.db.diskdb, env.hashes[i])
		if exist {
			t.Fatalf("Unexpected lookup")
		}
	}
}

func TestDatabaseRecoverable(t *testing.T) {
	var (
		env   = newTestEnv(t)
		index = env.bottomIndex()
	)
	// Initial state should be recoverable
	if !env.db.Recoverable(common.Hash{}) {
		t.Error("Layer unrecoverable")
	}
	// All the states below the disk layer should be recoverable.
	for i := 0; i < index; i++ {
		if !env.db.Recoverable(env.hashes[i]) {
			t.Error("Layer unrecoverable")
		}
	}
	// All other layers above(including disk layer) shouldn't be
	// recoverable since they are accessible.
	for i := index + 1; i < len(env.hashes); i++ {
		if env.db.Recoverable(env.hashes[i]) {
			t.Error("Layer should be unrecoverable")
		}
	}
}

func TestJournal(t *testing.T) {
	var (
		env   = newTestEnv(t)
		index = env.bottomIndex()
	)
	if err := env.db.Journal(env.hashes[len(env.hashes)-1]); err != nil {
		t.Error("Failed to journal triedb", "err", err)
	}
	env.db.Close()

	newdb := New(env.db.diskdb, fastcache.New(2*1024*1024), nil)
	for i := index; i < len(env.hashes); i++ {
		paths, blobs := env.paths[i], env.blobs[i]
		for j := 0; j < len(paths); j++ {
			if blobs[j] == nil {
				continue
			}
			layer := newdb.Reader(env.hashes[i])
			blob, err := layer.Node(common.Hash{}, paths[j], crypto.Keccak256Hash(blobs[j]))
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
	var (
		env   = newTestEnv(t)
		index = env.bottomIndex()
	)
	// Reset database to non-existent target, should reject it
	if err := env.db.Reset(testutil.RandomHash()); err == nil {
		t.Fatal("Failed to reject invalid reset")
	}
	// Reset database to state persisted in the disk
	_, hash := rawdb.ReadAccountTrieNode(env.db.diskdb, nil)
	if err := env.db.Reset(hash); err != nil {
		t.Fatalf("Failed to reset database %v", err)
	}
	// Ensure journal is deleted from disk
	if blob := rawdb.ReadTrieJournal(env.db.diskdb); len(blob) != 0 {
		t.Fatal("Failed to clean journal")
	}
	// Ensure all trie histories are nuked
	for i := 0; i <= index; i++ {
		_, err := loadTrieHistory(env.db.freezer, uint64(i+1))
		if err == nil {
			t.Fatalf("Failed to clean trie history, index %d", i+1)
		}
	}
	// Ensure there is only a single disk layer kept, hash should
	// be matched as well.
	if env.db.tree.len() != 1 {
		t.Fatalf("Extra layer kept %d", env.db.tree.len())
	}
	if env.db.tree.bottom().Root() != hash {
		t.Fatalf("Root hash is not matched exp %x got %x", hash, env.db.tree.bottom().Root())
	}
}

func TestCommit(t *testing.T) {
	env := newTestEnv(t)
	if err := env.db.Commit(env.hashes[len(env.hashes)-1], false); err != nil {
		t.Fatalf("Failed to cap database %v", err)
	}
	// Ensure there is only a single layer kept
	if env.db.tree.len() != 1 {
		t.Fatalf("Extra layer kept %d", env.db.tree.len())
	}
	if env.db.tree.bottom().Root() != env.lastHash() {
		t.Fatalf("Root hash is not matched exp %x got %x", env.lastHash(), env.db.tree.bottom().Root())
	}
	_, hash := rawdb.ReadAccountTrieNode(env.db.diskdb, nil)
	if hash != env.lastHash() {
		t.Fatalf("Root hash is not matched exp %x got %x", env.lastHash(), hash)
	}
}
