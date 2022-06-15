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
	"github.com/VictoriaMetrics/fastcache"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"golang.org/x/crypto/sha3"
)

// hashReader is reader of hashDatabase which implements the Reader interface.
type hashReader struct {
	db *hashDatabase
}

// newHashReader initializes the hash reader.
func newHashReader(db *hashDatabase) *hashReader {
	return &hashReader{db: db}
}

// Node retrieves the trie node associated with the given node hash. The
// returned node is in a wrapper through which callers can obtain the
// RLP-format or canonical node representation easily.
// No error will be returned if the node is not found.
func (reader *hashReader) Node(path []byte, hash common.Hash) (*cachedNode, error) {
	return reader.db.node(hash), nil
}

// NodeBlob retrieves the RLP-encoded trie node blob with the given node hash.
// No error will be returned if the node is not found.
func (reader *hashReader) NodeBlob(path []byte, hash common.Hash) ([]byte, error) {
	node := reader.db.node(hash)
	if node == nil {
		return nil, nil
	}
	return node.rlp(), nil
}

// hashDatabase is the legacy version database for maintaining trie nodes.
// All nodes will be persisted by its hash and mostly used by archive node
// and some other tries(e.g. state trie in LES, canonical hash trie etc).
type hashDatabase struct {
	readOnly bool
	diskdb   ethdb.KeyValueStore
	cleans   *fastcache.Cache
}

// openHashDatabase initializes the hash-based node database.
func openHashDatabase(diskdb ethdb.KeyValueStore, readOnly bool, cleans *fastcache.Cache) *hashDatabase {
	return &hashDatabase{
		readOnly: readOnly,
		diskdb:   diskdb,
		cleans:   cleans,
	}
}

// GetReader retrieves a node reader belonging to the given state root.
func (db *hashDatabase) GetReader(root common.Hash) Reader {
	return newHashReader(db)
}

// Commit flushes the given dirty nodes into disk. Since the ordering of
// trie nodes is already lost, commit them in a single database batch.
func (db *hashDatabase) Commit(root common.Hash, parentRoot common.Hash, result *NodeSet) error {
	var (
		batch  = db.diskdb.NewBatch()
		hash   = make([]byte, 32)
		hasher = sha3.NewLegacyKeccak256().(crypto.KeccakState)
	)
	for _, node := range result.nodes {
		// Never delete any node in hash-based scheme, since
		// it can be referenced by other tries.
		if node.node == nil {
			continue
		}
		val := node.rlp()
		hasher.Reset()
		hasher.Write(val)
		hasher.Read(hash)
		rawdb.WriteLegacyTrieNode(batch, hash, val)
		if db.cleans != nil {
			db.cleans.Set(hash, val)
		}
	}
	return batch.Write()
}

// node retrieves the node with given hash. Return nil if node is not found.
func (db *hashDatabase) node(hash common.Hash) *cachedNode {
	var blob []byte
	if db.cleans != nil {
		blob = db.cleans.Get(nil, hash.Bytes())
	}
	if blob == nil {
		blob = rawdb.ReadLegacyTrieNode(db.diskdb, hash)
	}
	if len(blob) == 0 {
		return nil
	}
	if db.cleans != nil {
		db.cleans.Set(hash.Bytes(), blob)
	}
	return &cachedNode{node: rawNode(blob), hash: hash, size: uint16(len(blob))}
}
