// Copyright 2020 The go-ethereum Authors
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
	"github.com/ethereum/go-ethereum/trie/trienode"
)

// committer is the tool used for the trie Commit operation. The committer will
// capture all dirty nodes during the commit process and keep them cached in
// insertion order.
type committer struct {
	nodes       *trienode.NodeSet
	tracer      *tracer
	collectLeaf bool
}

// newCommitter creates a new committer or picks one from the pool.
func newCommitter(nodeset *trienode.NodeSet, tracer *tracer, collectLeaf bool) *committer {
	return &committer{
		nodes:       nodeset,
		tracer:      tracer,
		collectLeaf: collectLeaf,
	}
}

// Commit collapses a node down into a hash node.
func (c *committer) Commit(n node, parallel bool) hashNode {
	return c.commit(nil, n, parallel).(hashNode)
}

// commit collapses a node down into a hash node and returns it.
func (c *committer) commit(path []byte, n node, parallel bool) node {
	// if this path is clean, use available cached data
	hash, dirty := n.cache()
	if hash != nil && !dirty {
		return hash
	}
	// Commit children, then parent, and remove the dirty flag.
	switch cn := n.(type) {
	case *shortNode:
		// If the child is fullNode, recursively commit,
		// otherwise it can only be hashNode or valueNode.
		if _, ok := cn.Val.(*fullNode); ok {
			cn.Val = c.commit(append(path, cn.Key...), cn.Val, false)
		}
		// The key needs to be copied, since we're adding it to the
		// modified nodeset.
		cn.Key = hexToCompact(cn.Key)
		hashedNode := c.store(path, cn)
		if hn, ok := hashedNode.(hashNode); ok {
			return hn
		}
		return cn
	case *fullNode:
		c.commitChildren(path, cn, parallel)
		hashedNode := c.store(path, cn)
		if hn, ok := hashedNode.(hashNode); ok {
			return hn
		}
		return cn
	case hashNode:
		return cn
	default:
		// nil, valuenode shouldn't be committed
		panic(fmt.Sprintf("%T: invalid node: %v", n, n))
	}
}

// commitChildren commits the children of the given fullnode
func (c *committer) commitChildren(path []byte, n *fullNode, parallel bool) {
	var (
		wg      sync.WaitGroup
		nodesMu sync.Mutex
	)
	for i := 0; i < 16; i++ {
		child := n.Children[i]
		if child == nil {
			continue
		}
		// If it's the hashed child, save the hash value directly.
		// Note: it's impossible that the child in range [0, 15]
		// is a valueNode.
		if _, ok := child.(hashNode); ok {
			continue
		}
		// Commit the child recursively and store the "hashed" value.
		// Note the returned node can be some embedded nodes, so it's
		// possible the type is not hashNode.
		if !parallel {
			n.Children[i] = c.commit(append(path, byte(i)), child, false)
		} else {
			wg.Add(1)
			go func(index int) {
				p := append(path, byte(index))
				childSet := trienode.NewNodeSet(c.nodes.Owner)
				childCommitter := newCommitter(childSet, c.tracer, c.collectLeaf)
				n.Children[index] = childCommitter.commit(p, child, false)
				nodesMu.Lock()
				c.nodes.MergeSet(childSet)
				nodesMu.Unlock()
				wg.Done()
			}(i)
		}
	}
	if parallel {
		wg.Wait()
	}
}

// store hashes the node n and adds it to the modified nodeset. If leaf collection
// is enabled, leaf nodes will be tracked in the modified nodeset as well.
func (c *committer) store(path []byte, n node) node {
	// Larger nodes are replaced by their hash and stored in the database.
	var hash, _ = n.cache()

	// This was not generated - must be a small node stored in the parent.
	// In theory, we should check if the node is leaf here (embedded node
	// usually is leaf node). But small value (less than 32bytes) is not
	// our target (leaves in account trie only).
	if hash == nil {
		// The node is embedded in its parent, in other words, this node
		// will not be stored in the database independently, mark it as
		// deleted only if the node was existent in database before.
		_, ok := c.tracer.accessList[string(path)]
		if ok {
			c.nodes.AddNode(path, trienode.NewDeleted())
		}
		return n
	}
	// Collect the dirty node to nodeset for return.
	nhash := common.BytesToHash(hash)
	c.nodes.AddNode(path, trienode.New(nhash, nodeToBytes(n)))

	// Collect the corresponding leaf node if it's required. We don't check
	// full node since it's impossible to store value in fullNode. The key
	// length of leaves should be exactly same.
	if c.collectLeaf {
		if sn, ok := n.(*shortNode); ok {
			if val, ok := sn.Val.(valueNode); ok {
				c.nodes.AddLeaf(nhash, val)
			}
		}
	}
	return hash
}

// ForGatherChildren decodes the provided node and traverses the children inside.
func ForGatherChildren(node []byte, onChild func(common.Hash)) {
	forGatherChildren(mustDecodeNodeUnsafe(nil, node), onChild)
}

// forGatherChildren traverses the node hierarchy and invokes the callback
// for all the hashnode children.
func forGatherChildren(n node, onChild func(hash common.Hash)) {
	switch n := n.(type) {
	case *shortNode:
		forGatherChildren(n.Val, onChild)
	case *fullNode:
		for i := 0; i < 16; i++ {
			forGatherChildren(n.Children[i], onChild)
		}
	case hashNode:
		onChild(common.BytesToHash(n))
	case valueNode, nil:
	default:
		panic(fmt.Sprintf("unknown node type: %T", n))
	}
}
