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
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/crypto/sha3"
)

// leafChanSize is the size of the leafCh. It's a pretty arbitrary number, to allow
// some parallelism but not incur too much memory overhead.
const leafChanSize = 200

// leaf represents a trie leaf value
type leaf struct {
	size int         // size of the rlp data (estimate)
	hash common.Hash // hash of rlp data
	node node        // the node to commit
	path []byte      // the path from the root node
}

// committer is a type used for the trie Commit operation. A committer has some
// internal preallocated temp space, and also a callback that is invoked when
// leaves are committed. The leafs are passed through the `leafCh`,  to allow
// some level of parallelism.
// By 'some level' of parallelism, it's still the case that all leaves will be
// processed sequentially - onleaf will never be called in parallel or out of order.
type committer struct {
	tmp sliceBuffer
	sha crypto.KeccakState

	owner     common.Hash
	onleaf    LeafCallback
	leafCh    chan *leaf
	committed *nodeSet
}

// committers live in a global sync.Pool
var committerPool = sync.Pool{
	New: func() interface{} {
		return &committer{
			tmp: make(sliceBuffer, 0, 550), // cap is as large as a full fullNode.
			sha: sha3.NewLegacyKeccak256().(crypto.KeccakState),
		}
	},
}

// newCommitter creates a new committer or picks one from the pool.
func newCommitter() *committer {
	ret := committerPool.Get().(*committer)
	ret.committed = newNodeSet()
	return ret
}

func returnCommitterToPool(h *committer) {
	h.onleaf = nil
	h.leafCh = nil
	h.committed = nil
	h.owner = common.Hash{}
	committerPool.Put(h)
}

// Commit collapses a node down into a hash node and inserts it into the database
func (c *committer) Commit(n node) (hashNode, error) {
	h, err := c.commit(nil, n)
	if err != nil {
		return nil, err
	}
	return h.(hashNode), nil
}

// commit collapses a node down into a hash node and inserts it into the database
func (c *committer) commit(path []byte, n node) (node, error) {
	// if this path is clean, use available cached data
	hash, dirty := n.cache()
	if hash != nil && !dirty {
		return hash, nil
	}
	// Commit children, then parent, and remove the dirty flag.
	switch cn := n.(type) {
	case *shortNode:
		// Commit child
		collapsed := cn.copy()

		// If the child is fullNode, recursively commit,
		// otherwise it can only be hashNode or valueNode.
		if _, ok := cn.Val.(*fullNode); ok {
			// Use concat here instead of append since the passed path
			// might be mutated.
			childV, err := c.commit(concat(path, cn.Key...), cn.Val)
			if err != nil {
				return nil, err
			}
			collapsed.Val = childV
		}
		// The key needs to be copied, since we're delivering it to database
		collapsed.Key = hexToCompact(cn.Key)
		hashedNode := c.store(path, collapsed)
		if hn, ok := hashedNode.(hashNode); ok {
			return hn, nil
		}
		return collapsed, nil
	case *fullNode:
		hashedKids, err := c.commitChildren(path, cn)
		if err != nil {
			return nil, err
		}
		collapsed := cn.copy()
		collapsed.Children = hashedKids

		hashedNode := c.store(path, collapsed)
		if hn, ok := hashedNode.(hashNode); ok {
			return hn, nil
		}
		return collapsed, nil
	case hashNode:
		return cn, nil
	default:
		// nil, valuenode shouldn't be committed
		panic(fmt.Sprintf("%T: invalid node: %v", n, n))
	}
}

// commitChildren commits the children of the given fullnode
func (c *committer) commitChildren(path []byte, n *fullNode) ([17]node, error) {
	var children [17]node
	for i := 0; i < 16; i++ {
		child := n.Children[i]
		if child == nil {
			continue
		}
		// If it's the hashed child, save the hash value directly.
		// Note: it's impossible that the child in range [0, 15]
		// is a valueNode.
		if hn, ok := child.(hashNode); ok {
			children[i] = hn
			continue
		}
		// Commit the child recursively and store the "hashed" value.
		// Note the returned node can be some embedded nodes, so it's
		// possible the type is not hashNode.
		hashed, err := c.commit(concat(path, byte(i)), child)
		if err != nil {
			return children, err
		}
		children[i] = hashed
	}
	// For the 17th child, it's possible the type is valuenode.
	if n.Children[16] != nil {
		children[16] = n.Children[16]
	}
	return children, nil
}

// store hashes the node n and if we have a storage layer specified, it writes
// the key/value pair to it and tracks any node->child references as well as any
// node->external trie references.
func (c *committer) store(path []byte, n node) node {
	// Larger nodes are replaced by their hash and stored in the database.
	var hash, _ = n.cache()

	// This was not generated - must be a small node stored in the parent.
	// In theory, we should apply the leafCall here if it's not nil(embedded
	// node usually contains value). But small value(less than 32bytes) is
	// not our target.
	if hash == nil {
		return n
	}
	var (
		// We have the hash already, estimate the RLP encoding-size of the node.
		// The size is used for mem tracking, does not need to be exact
		size = estimateSize(n)
		slim = simplifyNode(n)
		key  = EncodeInternalKeyWithPath(c.owner, path, common.BytesToHash(hash))
	)
	c.committed.put(key, slim, size)

	// If we're using channel-based leaf-reporting, send to channel.
	// The leaf channel will be active only when there an active leaf-callback
	if c.leafCh != nil {
		c.leafCh <- &leaf{
			size: size,
			hash: common.BytesToHash(hash),
			node: n,
			path: path,
		}
	}
	return hash
}

// commitLoop does the actual insert + leaf callback for nodes.
func (c *committer) commitLoop() {
	for item := range c.leafCh {
		var (
			hash = item.hash
			n    = item.node
			path = item.path
		)
		if c.onleaf != nil {
			switch n := n.(type) {
			case *shortNode:
				if child, ok := n.Val.(valueNode); ok {
					key := compactToHex(n.Key)
					if hasTerm(key) {
						key = key[:len(key)-1]
					}
					var (
						keys     [][]byte
						leafPath = append(append([]byte(nil), path...), key...)
					)
					if len(leafPath) == 2*common.HashLength {
						keys = append(keys, hexToKeybytes(leafPath))
					} else if len(leafPath) == 4*common.HashLength {
						keys = append(keys, hexToKeybytes(leafPath[:2*common.HashLength]))
						keys = append(keys, hexToKeybytes(leafPath[2*common.HashLength:]))
					}
					c.onleaf(keys, leafPath, child, hash, path)
				}
			case *fullNode:
				// For children in range [0, 15], it's impossible
				// to contain valueNode. Only check the 17th child.
				if n.Children[16] != nil {
					c.onleaf(nil, nil, n.Children[16].(valueNode), hash, nil)
				}
			}
		}
	}
}

func (c *committer) makeHashNode(data []byte) hashNode {
	n := make(hashNode, c.sha.Size())
	c.sha.Reset()
	c.sha.Write(data)
	c.sha.Read(n)
	return n
}

// estimateSize estimates the size of an rlp-encoded node, without actually
// rlp-encoding it (zero allocs). This method has been experimentally tried, and with a trie
// with 1000 leafs, the only errors above 1% are on small shortnodes, where this
// method overestimates by 2 or 3 bytes (e.g. 37 instead of 35)
func estimateSize(n node) int {
	switch n := n.(type) {
	case *shortNode:
		// A short node contains a compacted key, and a value.
		return 3 + len(n.Key) + estimateSize(n.Val)
	case *fullNode:
		// A full node contains up to 16 hashes (some nils), and a key
		s := 3
		for i := 0; i < 16; i++ {
			if child := n.Children[i]; child != nil {
				s += estimateSize(child)
			} else {
				s++
			}
		}
		return s
	case valueNode:
		return 1 + len(n)
	case hashNode:
		return 1 + len(n)
	default:
		panic(fmt.Sprintf("node type %T", n))
	}
}
