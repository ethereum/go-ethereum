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
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
)

// testEnv is the container for all testing fields.
type testEnv struct {
	db       *Database
	numbers  []uint64
	roots    []common.Hash
	keys     [][]string
	vals     [][][]byte
	teardown func()
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
			val     *nodeWithPreValue
		)
		switch rand.Intn(3) {
		case 0:
			// node creation
			storage = EncodeStorageKey(common.Hash{}, randomHash().Bytes())
			val = &nodeWithPreValue{
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
			pos := rand.Intn(len(pkeys))
			storage = []byte(pkeys[pos])

			val = &nodeWithPreValue{
				cachedNode: randomNode(),
				pre:        testvals[parentNumber-1][pos],
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

			val = &nodeWithPreValue{
				cachedNode: randomEmptyNode(),
				pre:        pvals[index],
			}
		}
		// Don't add duplicated updates
		if _, ok := nodes[string(storage)]; ok {
			continue
		}
		nodes[string(storage)] = val
		keys = append(keys, string(storage))

		if val.node == nil {
			vals = append(vals, nil)
		} else {
			vals = append(vals, common.CopyBytes(val.rlp()))
		}
	}
	// Add the root node
	root := randomNode()
	nodes[string(EncodeStorageKey(common.Hash{}, nil))] = &nodeWithPreValue{
		cachedNode: root,
		pre:        parentBlob,
	}
	db.Update(root.hash, parentHash, nodes)
	db.Cap(root.hash, 128)
	return root.hash, root.rlp(), parentNumber + 1, keys, vals
}

func fillDB() *testEnv {
	dir, err := ioutil.TempDir(os.TempDir(), "testing")
	if err != nil {
		panic("Failed to allocate tempdir")
	}
	diskdb, err := rawdb.NewLevelDBDatabaseWithFreezer(dir, 16, 16, path.Join(dir, "test-frdb"), "", false)
	if err != nil {
		panic(fmt.Sprintf("Failed to create database %v", err))
	}
	var (
		db      = NewDatabase(diskdb, nil)
		numbers []uint64
		roots   []common.Hash

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
		teardown: func() {
			os.RemoveAll(dir)
		},
	}
}

func TestDatabaseRollback(t *testing.T) {
	defer func(origin uint64) {
		cacheSizeLimit = origin
	}(cacheSizeLimit)

	cacheSizeLimit = 1024 * 256 // Lower the dirty cache size

	var (
		env       = fillDB()
		dl        = env.db.disklayer()
		diskIndex int
	)
	defer env.teardown()

	for diskIndex = 0; diskIndex < len(env.roots); diskIndex++ {
		if env.roots[diskIndex] == dl.root {
			break
		}
	}
	// Ensure all the reverse diffs are stored properly
	var parent = emptyRoot
	for i := 0; i <= diskIndex; i++ {
		diff, err := loadReverseDiff(env.db.diskdb, uint64(i+1))
		if err != nil {
			t.Errorf("Failed to load reverse diff, index %d, err %v", i+1, err)
		}
		if diff.Parent != parent {
			t.Error("Reverse diff is not continuous")
		}
		parent = diff.Root
	}
	// Ensure immature reverse diffs are not present
	for i := diskIndex + 1; i < len(env.numbers); i++ {
		blob := rawdb.ReadReverseDiff(env.db.diskdb, uint64(i+1))
		if len(blob) != 0 {
			t.Error("Unexpected reverse diff", "index", i)
		}
	}
	// Revert the db to historical point with reverse state available
	for i := diskIndex; i > 0; i-- {
		if err := env.db.Rollback(env.roots[i-1]); err != nil {
			t.Error("Failed to revert db status", "err", err)
		}
		dl := env.db.disklayer()
		if dl.Root() != env.roots[i-1] {
			t.Error("Unexpected disk layer root")
		}
		// Compare the reverted state with the constructed one, they should be same.
		keys, vals := env.keys[i-1], env.vals[i-1]
		for j := 0; j < len(keys); j++ {
			layer := env.db.Snapshot(env.roots[i-1])

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
	if env.db.tree.len() != 1 {
		t.Error("Only disk layer is expected")
	}
}

func TestDatabaseBatchRollback(t *testing.T) {
	defer func(origin uint64) {
		cacheSizeLimit = origin
	}(cacheSizeLimit)

	cacheSizeLimit = 1024 * 256 // Lower the dirty cache size

	var (
		env       = fillDB()
		dl        = env.db.disklayer()
		diskIndex int
	)
	defer env.teardown()
	for diskIndex = 0; diskIndex < len(env.roots); diskIndex++ {
		if env.roots[diskIndex] == dl.root {
			break
		}
	}
	// Revert the db to historical point with reverse state available
	if err := env.db.Rollback(common.Hash{}); err != nil {
		t.Error("Failed to revert db status", "err", err)
	}
	ndl := env.db.disklayer()
	if ndl.Root() != emptyRoot {
		t.Error("Unexpected disk layer root")
	}
	if env.db.tree.len() != 1 {
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
