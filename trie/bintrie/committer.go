// Copyright 2026 The go-ethereum Authors
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

package bintrie

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie/trienode"
)

// opTracer tracks the database records created and removed by structural
// mutations, so Commit can emit deletion markers alongside updated nodes.
// Creating and deleting the same path within one session cancels out; a
// record replaced in place at the same path needs no tracking (the dirty
// walk re-emits it). This mirrors the MPT's unexported tracer discipline.
type opTracer struct {
	inserts map[string]struct{}
	deletes map[string]struct{}
}

func newOpTracer() *opTracer {
	return &opTracer{
		inserts: make(map[string]struct{}),
		deletes: make(map[string]struct{}),
	}
}

func (t *opTracer) onInsert(path []byte) {
	p := string(path)
	if _, ok := t.deletes[p]; ok {
		delete(t.deletes, p)
		return
	}
	t.inserts[p] = struct{}{}
}

func (t *opTracer) onDelete(path []byte) {
	p := string(path)
	if _, ok := t.inserts[p]; ok {
		delete(t.inserts, p)
		return
	}
	t.deletes[p] = struct{}{}
}

func (t *opTracer) deletedList() [][]byte {
	list := make([][]byte, 0, len(t.deletes))
	for p := range t.deletes {
		list = append(list, []byte(p))
	}
	return list
}

func (t *opTracer) copy() *opTracer {
	cp := newOpTracer()
	for p := range t.inserts {
		cp.inserts[p] = struct{}{}
	}
	for p := range t.deletes {
		cp.deletes[p] = struct{}{}
	}
	return cp
}

// deletedNodes returns the paths of records removed from the database by
// this session's mutations, filtered to those that actually existed on disk
// (records created and dropped in the same session never hit the database
// and carry no witness entry).
func (t *BinaryTrie) deletedNodes() [][]byte {
	var (
		pos   int
		list  = t.ops.deletedList()
		flags = t.tracer.HasList(list)
	)
	for i := 0; i < len(list); i++ {
		if flags[i] {
			list[pos] = list[i]
			pos++
		}
	}
	return list[:pos]
}

// Commit collects the trie's modified records into a NodeSet: updated and
// created records from a dirty-node walk, deletions (with previous values)
// from the mutation tracer. The trie is unusable afterwards.
func (t *BinaryTrie) Commit(collectLeaf bool) (common.Hash, *trienode.NodeSet) {
	defer func() { t.committed = true }()

	// An empty trie either had nothing to commit, or every record was
	// deleted; the deletion set distinguishes the two.
	if _, isEmpty := t.root.(empty); isEmpty {
		paths := t.deletedNodes()
		if len(paths) == 0 {
			return types.EmptyBinaryHash, nil
		}
		set := trienode.NewNodeSet(common.Hash{})
		for _, path := range paths {
			set.AddNode(path, trienode.NewDeletedWithPrev(t.tracer.Get(path)))
		}
		return types.EmptyBinaryHash, set
	}

	// Hash first: the collect walk assumes every cached hash is current.
	rootHash := t.Hash()

	set := trienode.NewNodeSet(common.Hash{})
	for _, path := range t.deletedNodes() {
		set.AddNode(path, trienode.NewDeletedWithPrev(t.tracer.Get(path)))
	}
	t.collect(t.root, bitstr{}, 0, set)
	if len(set.Nodes) == 0 {
		return rootHash, nil
	}
	t.root = hashedNode(rootHash)
	return rootHash, set
}

// collect walks the dirty spine, serializing every modified record at its
// path. Clean subtrees (including unresolved hashedNodes) are skipped: a
// mutation dirties every node on its path to the root.
func (t *BinaryTrie) collect(n binaryNode, walk bitstr, pos int, set *trienode.NodeSet) {
	switch nn := n.(type) {
	case empty, hashedNode:
		return
	case *groupNode:
		if !nn.modified {
			return
		}
		nn.modified = false
		path := pathOf(walk)
		set.AddNode(path, trienode.NewNodeWithPrev(nn.hashAt(pos), serializeNode(nn, pos), t.tracer.Get(path)))
	case *branchNode:
		if !nn.modified {
			return
		}
		nn.modified = false
		child := pos + nn.prefix.n + 1
		t.collect(nn.left, walk.append(nn.prefix).concat(0, bitstr{}), child, set)
		t.collect(nn.right, walk.append(nn.prefix).concat(1, bitstr{}), child, set)
		path := pathOf(walk)
		set.AddNode(path, trienode.NewNodeWithPrev(nn.hashAt(pos), serializeNode(nn, pos), t.tracer.Get(path)))
	}
}
