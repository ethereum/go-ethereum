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
	"runtime"
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
	parallel    bool
}

// newCommitter creates a new committer or picks one from the pool.
func newCommitter(nodes *trienode.NodeSet, tracer *tracer, collectLeaf bool, parallel bool) *committer {
	return &committer{
		nodes:       nodes,
		tracer:      tracer,
		collectLeaf: collectLeaf,
		parallel:    parallel,
	}
}

type wrapNode struct {
	node     *trienode.Node
	path     string
	leafHash common.Hash // optional, the parent hash of the relative leaf
	leafBlob []byte      // optional, the blob of the relative leaf
}

// Commit collapses a node down into a hash node.
func (c *committer) Commit(n node) hashNode {
	hn, wnodes := c.commit(nil, n, true)
	for _, wn := range wnodes {
		c.nodes.AddNode(wn.path, wn.node)
		if wn.leafHash != (common.Hash{}) {
			c.nodes.AddLeaf(wn.leafHash, wn.leafBlob)
		}
	}
	return hn.(hashNode)
}

// commit collapses a node down into a hash node and returns it.
func (c *committer) commit(path []byte, n node, topmost bool) (node, []*wrapNode) {
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
		var nodes []*wrapNode
		if _, ok := cn.Val.(*fullNode); ok {
			collapsed.Val, nodes = c.commit(append(path, cn.Key...), cn.Val, false)
		}
		// The key needs to be copied, since we're adding it to the
		// modified nodeset.
		collapsed.Key = hexToCompact(cn.Key)
		hashedNode, wNode := c.store(path, collapsed)
		if wNode != nil {
			nodes = append(nodes, wNode)
		}
		if hn, ok := hashedNode.(hashNode); ok {
			return hn, nodes
		}
		return collapsed, nodes
	case *fullNode:
		hashedKids, nodes := c.commitChildren(path, cn, topmost && c.parallel)
		collapsed := cn.copy()
		collapsed.Children = hashedKids

		hashedNode, wNode := c.store(path, collapsed)
		if wNode != nil {
			nodes = append(nodes, wNode)
		}
		if hn, ok := hashedNode.(hashNode); ok {
			return hn, nodes
		}
		return collapsed, nodes
	case hashNode:
		return cn, nil
	default:
		// nil, valuenode shouldn't be committed
		panic(fmt.Sprintf("%T: invalid node: %v", n, n))
	}
}

type task struct {
	node  node
	index int
	path  []byte
}

// commitChildren commits the children of the given fullnode
func (c *committer) commitChildren(path []byte, n *fullNode, parallel bool) ([17]node, []*wrapNode) {
	var (
		wg       sync.WaitGroup
		children [17]node
		results  [16][]*wrapNode
		tasks    = make(chan task)
	)
	if parallel {
		worker := func() {
			defer wg.Done()
			for t := range tasks {
				children[t.index], results[t.index] = c.commit(t.path, t.node, false)
			}
		}
		threads := runtime.NumCPU()
		if threads > 16 {
			threads = 16
		}
		for i := 0; i < threads; i++ {
			wg.Add(1)
			go worker()
		}
	}
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
		if !parallel {
			children[i], results[i] = c.commit(append(path, byte(i)), child, false)
		} else {
			tasks <- task{
				index: i,
				node:  child,
				path:  append(path, byte(i)),
			}
		}
	}
	if parallel {
		close(tasks)
		wg.Wait()
	}
	// For the 17th child, it's possible the type is valuenode.
	if n.Children[16] != nil {
		children[16] = n.Children[16]
	}
	var wnodes []*wrapNode
	for i := 0; i < 16; i++ {
		if results[i] != nil {
			wnodes = append(wnodes, results[i]...)
		}
	}
	return children, wnodes
}

// store hashes the node n and adds it to the modified nodeset. If leaf collection
// is enabled, leaf nodes will be tracked in the modified nodeset as well.
func (c *committer) store(path []byte, n node) (node, *wrapNode) {
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
			return n, &wrapNode{
				path: string(path),
				node: trienode.NewDeleted(),
			}
		}
		return n, nil
	}
	nhash := common.BytesToHash(hash)
	wNode := &wrapNode{
		path: string(path),
		node: trienode.New(nhash, nodeToBytes(n)),
	}

	// Collect the corresponding leaf node if it's required. We don't check
	// full node since it's impossible to store value in fullNode. The key
	// length of leaves should be exactly same..
	if c.collectLeaf {
		if sn, ok := n.(*shortNode); ok {
			if val, ok := sn.Val.(valueNode); ok {
				wNode.leafHash = nhash
				wNode.leafBlob = val
			}
		}
	}

	return hash, wNode
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
