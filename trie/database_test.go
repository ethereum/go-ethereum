// Copyright 2019 The go-ethereum Authors
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
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/triedb/database"
)

// testReader implements database.Reader interface, providing function to
// access trie nodes.
type testReader struct {
	db     ethdb.Database
	scheme string
	nodes  []*trienode.MergedNodeSet // sorted from new to old
}

// Node implements database.Reader interface, retrieving trie node with
// all available cached layers.
func (r *testReader) Node(owner common.Hash, path []byte, hash common.Hash) ([]byte, error) {
	// Check the node presence with the cached layer, from latest to oldest.
	for _, nodes := range r.nodes {
		if _, ok := nodes.Sets[owner]; !ok {
			continue
		}
		n, ok := nodes.Sets[owner].Nodes[string(path)]
		if !ok {
			continue
		}
		if n.IsDeleted() || n.Hash != hash {
			return nil, &MissingNodeError{Owner: owner, Path: path, NodeHash: hash}
		}
		return n.Blob, nil
	}
	// Check the node presence in database.
	return rawdb.ReadTrieNode(r.db, owner, path, hash, r.scheme), nil
}

// testDb implements database.Database interface, using for testing purpose.
type testDb struct {
	disk    ethdb.Database
	root    common.Hash
	scheme  string
	nodes   map[common.Hash]*trienode.MergedNodeSet
	parents map[common.Hash]common.Hash
}

func newTestDatabase(diskdb ethdb.Database, scheme string) *testDb {
	return &testDb{
		disk:    diskdb,
		root:    types.EmptyRootHash,
		scheme:  scheme,
		nodes:   make(map[common.Hash]*trienode.MergedNodeSet),
		parents: make(map[common.Hash]common.Hash),
	}
}

func (db *testDb) Reader(stateRoot common.Hash) (database.Reader, error) {
	nodes, _ := db.dirties(stateRoot, true)
	return &testReader{db: db.disk, scheme: db.scheme, nodes: nodes}, nil
}

func (db *testDb) Preimage(hash common.Hash) []byte {
	return rawdb.ReadPreimage(db.disk, hash)
}

func (db *testDb) InsertPreimage(preimages map[common.Hash][]byte) {
	rawdb.WritePreimages(db.disk, preimages)
}

func (db *testDb) Scheme() string { return db.scheme }

func (db *testDb) Update(root common.Hash, parent common.Hash, nodes *trienode.MergedNodeSet) error {
	if root == parent {
		return nil
	}
	if _, ok := db.nodes[root]; ok {
		return nil
	}
	db.parents[root] = parent
	db.nodes[root] = nodes
	return nil
}

func (db *testDb) dirties(root common.Hash, topToBottom bool) ([]*trienode.MergedNodeSet, []common.Hash) {
	var (
		pending []*trienode.MergedNodeSet
		roots   []common.Hash
	)
	for {
		if root == db.root {
			break
		}
		nodes, ok := db.nodes[root]
		if !ok {
			break
		}
		if topToBottom {
			pending = append(pending, nodes)
			roots = append(roots, root)
		} else {
			pending = append([]*trienode.MergedNodeSet{nodes}, pending...)
			roots = append([]common.Hash{root}, roots...)
		}
		root = db.parents[root]
	}
	return pending, roots
}

func (db *testDb) Commit(root common.Hash) error {
	if root == db.root {
		return nil
	}
	pending, roots := db.dirties(root, false)
	for i, nodes := range pending {
		for owner, set := range nodes.Sets {
			if owner == (common.Hash{}) {
				continue
			}
			set.ForEachWithOrder(func(path string, n *trienode.Node) {
				rawdb.WriteTrieNode(db.disk, owner, []byte(path), n.Hash, n.Blob, db.scheme)
			})
		}
		nodes.Sets[common.Hash{}].ForEachWithOrder(func(path string, n *trienode.Node) {
			rawdb.WriteTrieNode(db.disk, common.Hash{}, []byte(path), n.Hash, n.Blob, db.scheme)
		})
		db.root = roots[i]
	}
	for _, root := range roots {
		delete(db.nodes, root)
		delete(db.parents, root)
	}
	return nil
}
