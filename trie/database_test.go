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

func fillDB() (*Database, []uint64, []common.Hash, [][]string, [][][]byte, func()) {
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
	// First, we set up 128 diff layers, with 300 items each
	fill := func(parentHash common.Hash, parentNumber uint64) common.Hash {
		var (
			keys  []string
			vals  [][]byte
			nodes = make(map[string]*cachedNode)
		)
		for i := 0; i < 300; i++ {
			var (
				storage []byte
				val     *cachedNode
			)
			r := rand.Intn(3)
			switch r {
			case 0:
				// node creation
				storage, val = EncodeStorageKey(common.Hash{}, randomHash().Bytes()), randomNode()

			case 1:
				// node modification
				if parentNumber == 0 {
					continue
				}
				pkeys := testKeys[parentNumber-1]
				if len(pkeys) == 0 {
					continue
				}
				storage, val = []byte(pkeys[rand.Intn(len(pkeys))]), randomNode()

			case 2:
				// node deletion
				if parentNumber == 0 {
					continue
				}
				pkeys, pvals := testKeys[parentNumber-1], testVals[parentNumber-1]
				if len(pkeys) == 0 {
					continue
				}
				index := rand.Intn(len(pkeys))
				if len(pvals[index]) == 0 {
					continue
				}
				storage, val = []byte(pkeys[index]), randomEmptyNode(crypto.Keccak256Hash(pvals[index]))
			}
			// Don't add duplicate updates
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
		nodes[string(EncodeStorageKey(common.Hash{}, nil))] = root

		db.Update(root.hash, parentHash, nodes)
		db.Cap(root.hash, 128)

		numbers = append(numbers, parentNumber+1)
		roots = append(roots, root.hash)
		testKeys = append(testKeys, keys)
		testVals = append(testVals, vals)
		return root.hash
	}
	// Construct a database with enough reverse diffs stored
	var parent common.Hash
	for i := 0; i < 2*128; i++ {
		parent = fill(parent, uint64(i))
	}
	return db, numbers, roots, testKeys, testVals, func() { os.RemoveAll(dir) }
}

func TestDatabaseRollback(t *testing.T) {
	var (
		db, numbers, roots, testKeys, testVals, teardown = fillDB()
		dl                                               = db.disklayer()
		diskIndex                                        int
	)
	defer teardown()

	for diskIndex = 0; diskIndex < len(roots); diskIndex++ {
		if roots[diskIndex] == dl.root {
			break
		}
	}
	// Ensure all the reverse diffs are stored properly
	var parent = emptyRoot
	for i := 0; i <= diskIndex; i++ {
		diff, err := loadReverseDiff(db.diskdb, uint64(i+1))
		if err != nil {
			t.Error("Failed to load reverse diff", "err", err)
		}
		if diff.Parent != parent {
			t.Error("Reverse diff is not continuous")
		}
		parent = diff.Root
	}
	// Ensure immature reverse diffs are not present
	for i := diskIndex + 1; i < len(numbers); i++ {
		blob := rawdb.ReadReverseDiff(db.diskdb, uint64(i+1))
		if len(blob) != 0 {
			t.Error("Unexpected reverse diff", "index", i)
		}
	}
	// Revert the db to historical point with reverse state available
	for i := diskIndex; i > 0; i-- {
		if err := db.Rollback(roots[i-1]); err != nil {
			t.Error("Failed to revert db status", "err", err)
		}
		dl := db.disklayer()
		if dl.Root() != roots[i-1] {
			t.Error("Unexpected disk layer root")
		}
		// Compare the reverted state with the constructed one, they should be same.
		keys, vals := testKeys[i-1], testVals[i-1]
		for j := 0; j < len(keys); j++ {
			layer := db.Snapshot(roots[i-1])
			blob, err := layer.NodeBlob([]byte(keys[j]), crypto.Keccak256Hash(vals[j]))
			if err != nil {
				t.Error("Failed to retrieve state", "err", err)
			}
			if !bytes.Equal(blob, vals[j]) {
				t.Error("Unexpected state", "key", []byte(keys[j]), "want", vals[j], "got", blob)
			}
		}
	}
	if len(db.layers) != 1 {
		t.Error("Only disk layer is expected")
	}
}

func TestDatabaseBatchRollback(t *testing.T) {
	var (
		db, _, roots, testKeys, testVals, teardown = fillDB()
		dl                                         = db.disklayer()
		diskIndex                                  int
	)
	defer teardown()
	for diskIndex = 0; diskIndex < len(roots); diskIndex++ {
		if roots[diskIndex] == dl.root {
			break
		}
	}
	// Revert the db to historical point with reverse state available
	if err := db.Rollback(common.Hash{}); err != nil {
		t.Error("Failed to revert db status", "err", err)
	}
	ndl := db.disklayer()
	if ndl.Root() != emptyRoot {
		t.Error("Unexpected disk layer root")
	}
	if len(db.layers) != 1 {
		t.Error("Only disk layer is expected")
	}
	// Ensure all the states are deleted by reverting.
	for i, keys := range testKeys {
		vals := testVals[i]
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
