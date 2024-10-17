// Copyright 2024 The go-ethereum Authors
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
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>

package pathdb

import (
	"bytes"
	"fmt"
	"io"
	"maps"

	"github.com/VictoriaMetrics/fastcache"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie/trienode"
)

// nodeSet represents a collection of modified trie nodes resulting from a state
// transition, typically corresponding to a block execution. It can also represent
// the combined trie node set from several aggregated state transitions.
type nodeSet struct {
	size  uint64                                    // aggregated size of the trie node
	nodes map[common.Hash]map[string]*trienode.Node // node set, mapped by owner and path
}

// newNodeSet constructs the set with the provided dirty trie nodes.
func newNodeSet(nodes map[common.Hash]map[string]*trienode.Node) *nodeSet {
	// Don't panic for the lazy callers, initialize the nil map instead
	if nodes == nil {
		nodes = make(map[common.Hash]map[string]*trienode.Node)
	}
	s := &nodeSet{nodes: nodes}
	s.computeSize()
	return s
}

// computeSize calculates the database size of the held trie nodes.
func (s *nodeSet) computeSize() {
	var size uint64
	for owner, subset := range s.nodes {
		var prefix int
		if owner != (common.Hash{}) {
			prefix = common.HashLength // owner (32 bytes) for storage trie nodes
		}
		for path, n := range subset {
			size += uint64(prefix + len(n.Blob) + len(path))
		}
	}
	s.size = size
}

// updateSize updates the total cache size by the given delta.
func (s *nodeSet) updateSize(delta int64) {
	size := int64(s.size) + delta
	if size >= 0 {
		s.size = uint64(size)
		return
	}
	log.Error("Nodeset size underflow", "prev", common.StorageSize(s.size), "delta", common.StorageSize(delta))
	s.size = 0
}

// node retrieves the trie node with node path and its trie identifier.
func (s *nodeSet) node(owner common.Hash, path []byte) (*trienode.Node, bool) {
	subset, ok := s.nodes[owner]
	if !ok {
		return nil, false
	}
	n, ok := subset[string(path)]
	if !ok {
		return nil, false
	}
	return n, true
}

// merge integrates the provided dirty nodes into the set. The provided nodeset
// will remain unchanged, as it may still be referenced by other layers.
func (s *nodeSet) merge(set *nodeSet) {
	var (
		delta     int64   // size difference resulting from node merging
		overwrite counter // counter of nodes being overwritten
	)
	for owner, subset := range set.nodes {
		var prefix int
		if owner != (common.Hash{}) {
			prefix = common.HashLength
		}
		current, exist := s.nodes[owner]
		if !exist {
			for path, n := range subset {
				delta += int64(prefix + len(n.Blob) + len(path))
			}
			// Perform a shallow copy of the map for the subset instead of claiming it
			// directly from the provided nodeset to avoid potential concurrent map
			// read/write issues. The nodes belonging to the original diff layer remain
			// accessible even after merging. Therefore, ownership of the nodes map
			// should still belong to the original layer, and any modifications to it
			// should be prevented.
			s.nodes[owner] = maps.Clone(subset)
			continue
		}
		for path, n := range subset {
			if orig, exist := current[path]; !exist {
				delta += int64(prefix + len(n.Blob) + len(path))
			} else {
				delta += int64(len(n.Blob) - len(orig.Blob))
				overwrite.add(prefix + len(orig.Blob) + len(path))
			}
			current[path] = n
		}
		s.nodes[owner] = current
	}
	overwrite.report(gcTrieNodeMeter, gcTrieNodeBytesMeter)
	s.updateSize(delta)
}

// revert merges the provided trie nodes into the set. This should reverse the
// changes made by the most recent state transition.
func (s *nodeSet) revert(db ethdb.KeyValueReader, nodes map[common.Hash]map[string]*trienode.Node) {
	var delta int64
	for owner, subset := range nodes {
		current, ok := s.nodes[owner]
		if !ok {
			panic(fmt.Sprintf("non-existent subset (%x)", owner))
		}
		for path, n := range subset {
			orig, ok := current[path]
			if !ok {
				// There is a special case in merkle tree that one child is removed
				// from a fullNode which only has two children, and then a new child
				// with different position is immediately inserted into the fullNode.
				// In this case, the clean child of the fullNode will also be marked
				// as dirty because of node collapse and expansion. In case of database
				// rollback, don't panic if this "clean" node occurs which is not
				// present in buffer.
				var blob []byte
				if owner == (common.Hash{}) {
					blob = rawdb.ReadAccountTrieNode(db, []byte(path))
				} else {
					blob = rawdb.ReadStorageTrieNode(db, owner, []byte(path))
				}
				// Ignore the clean node in the case described above.
				if bytes.Equal(blob, n.Blob) {
					continue
				}
				panic(fmt.Sprintf("non-existent node (%x %v) blob: %v", owner, path, crypto.Keccak256Hash(n.Blob).Hex()))
			}
			current[path] = n
			delta += int64(len(n.Blob)) - int64(len(orig.Blob))
		}
	}
	s.updateSize(delta)
}

// journalNode represents a trie node persisted in the journal.
type journalNode struct {
	Path []byte // Path of the node in the trie
	Blob []byte // RLP-encoded trie node blob, nil means the node is deleted
}

// journalNodes represents a list trie nodes belong to a single account
// or the main account trie.
type journalNodes struct {
	Owner common.Hash
	Nodes []journalNode
}

// encode serializes the content of trie nodes into the provided writer.
func (s *nodeSet) encode(w io.Writer) error {
	nodes := make([]journalNodes, 0, len(s.nodes))
	for owner, subset := range s.nodes {
		entry := journalNodes{Owner: owner}
		for path, node := range subset {
			entry.Nodes = append(entry.Nodes, journalNode{
				Path: []byte(path),
				Blob: node.Blob,
			})
		}
		nodes = append(nodes, entry)
	}
	return rlp.Encode(w, nodes)
}

// decode deserializes the content from the rlp stream into the nodeset.
func (s *nodeSet) decode(r *rlp.Stream) error {
	var encoded []journalNodes
	if err := r.Decode(&encoded); err != nil {
		return fmt.Errorf("load nodes: %v", err)
	}
	nodes := make(map[common.Hash]map[string]*trienode.Node)
	for _, entry := range encoded {
		subset := make(map[string]*trienode.Node)
		for _, n := range entry.Nodes {
			if len(n.Blob) > 0 {
				subset[string(n.Path)] = trienode.New(crypto.Keccak256Hash(n.Blob), n.Blob)
			} else {
				subset[string(n.Path)] = trienode.NewDeleted()
			}
		}
		nodes[entry.Owner] = subset
	}
	s.nodes = nodes
	s.computeSize()
	return nil
}

// write flushes nodes into the provided database batch as a whole.
func (s *nodeSet) write(batch ethdb.Batch, clean *fastcache.Cache) int {
	return writeNodes(batch, s.nodes, clean)
}

// reset clears all cached trie node data.
func (s *nodeSet) reset() {
	s.nodes = make(map[common.Hash]map[string]*trienode.Node)
	s.size = 0
}

// dbsize returns the approximate size of db write.
func (s *nodeSet) dbsize() int {
	var m int
	for owner, nodes := range s.nodes {
		if owner == (common.Hash{}) {
			m += len(nodes) * len(rawdb.TrieNodeAccountPrefix) // database key prefix
		} else {
			m += len(nodes) * (len(rawdb.TrieNodeStoragePrefix)) // database key prefix
		}
	}
	return m + int(s.size)
}
