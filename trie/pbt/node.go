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

package pbt

import (
	"math/bits"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/zeebo/blake3"
)

const (
	tagLeaf   = 0x00
	tagBranch = 0x01
	tagGroup  = 0x02 // database record only
)

// node is positionless: walks pass pos, the number of key bits consumed by
// the node's ancestors.
type node interface {
	hashAt(pos int) common.Hash
	copy() node
}

type (
	// branchNode splits on the bit after its prefix. Both children are
	// non-empty.
	branchNode struct {
		prefix      bitstr
		left, right node
		hash        common.Hash
		dirty       bool // hash is stale
		modified    bool // record changed since load
	}
	// groupNode holds the values of one stem sorted by sub-index, standing for
	// the canonical subtree over them. A single value is a leaf.
	groupNode struct {
		stem     []byte
		subs     []byte
		vals     [][]byte
		hash     common.Hash
		hashPos  int // groups hash relative to their position
		dirty    bool
		modified bool
	}
	hashedNode common.Hash
	emptyNode  struct{}
)

func leafHash(stem []byte, sub byte, value []byte) common.Hash {
	var buf [1 + storageKeyLength + 32]byte
	b := append(append(append(append(buf[:0], tagLeaf), stem...), sub), value...)
	return blake3.Sum256(b)
}

func branchHash(prefix bitstr, left, right common.Hash) common.Hash {
	var buf [1 + 2 + maxPathBits/8 + 64]byte
	b := appendBitPrefix(append(buf[:0], tagBranch), prefix)
	return blake3.Sum256(append(append(b, left[:]...), right[:]...))
}

func (n *branchNode) hashAt(pos int) common.Hash {
	if n.dirty || n.hash == (common.Hash{}) {
		child := pos + n.prefix.n + 1
		n.hash = branchHash(n.prefix, n.left.hashAt(child), n.right.hashAt(child))
		n.dirty = false
	}
	return n.hash
}

func (n *branchNode) copy() node {
	cp := *n
	cp.prefix.b = slices.Clone(n.prefix.b)
	cp.left, cp.right = n.left.copy(), n.right.copy()
	return &cp
}

func (g *groupNode) hashAt(pos int) common.Hash {
	if g.dirty || g.hash == (common.Hash{}) || g.hashPos != pos {
		g.hash, g.hashPos, g.dirty = g.fold(pos), pos, false
	}
	return g.hash
}

// fold hashes the canonical subtree over the group's values at pos: the top
// branch prefix carries the stem bits below pos.
func (g *groupNode) fold(pos int) common.Hash {
	if len(g.subs) == 1 {
		return leafHash(g.stem, g.subs[0], g.vals[0])
	}
	return g.foldRange(0, len(g.subs), 0, pos, 8*len(g.stem))
}

// foldRange hashes the subtree over subs[i:j], whose sub-index bits agree on
// [0, from). Its prefix is stem bits [lo, hi) followed by the shared sub bits.
func (g *groupNode) foldRange(i, j, from, lo, hi int) common.Hash {
	if j-i == 1 {
		return leafHash(g.stem, g.subs[i], g.vals[i])
	}
	b := 8 - bits.Len8(g.subs[i]^g.subs[j-1])
	m := i + 1
	for g.subs[m]>>(7-b)&1 == 0 {
		m++
	}
	var prefix bitstr
	if hi > lo {
		prefix = slice(g.stem, lo, hi-lo)
		for k := from; k < b; k++ {
			prefix = prefix.concat(g.subs[i]>>(7-k)&1, bitstr{})
		}
	} else {
		prefix = slice([]byte{g.subs[i]}, from, b-from)
	}
	return branchHash(prefix, g.foldRange(i, m, b+1, 0, 0), g.foldRange(m, j, b+1, 0, 0))
}

func (g *groupNode) lookup(sub byte) []byte {
	if i, ok := slices.BinarySearch(g.subs, sub); ok {
		return g.vals[i]
	}
	return nil
}

// set writes value at sub; nil deletes.
func (g *groupNode) set(sub byte, value []byte) {
	i, ok := slices.BinarySearch(g.subs, sub)
	switch {
	case ok && value == nil:
		g.subs = slices.Delete(g.subs, i, i+1)
		g.vals = slices.Delete(g.vals, i, i+1)
	case ok:
		g.vals[i] = value
	case value != nil:
		g.subs = slices.Insert(g.subs, i, sub)
		g.vals = slices.Insert(g.vals, i, value)
	}
	g.dirty, g.modified = true, true
}

func (g *groupNode) copy() node {
	cp := *g
	cp.stem, cp.subs, cp.vals = slices.Clone(g.stem), slices.Clone(g.subs), slices.Clone(g.vals)
	return &cp
}

func (h hashedNode) hashAt(int) common.Hash { return common.Hash(h) }
func (h hashedNode) copy() node             { return h }
func (emptyNode) hashAt(int) common.Hash    { return common.Hash{} }
func (emptyNode) copy() node                { return emptyNode{} }
