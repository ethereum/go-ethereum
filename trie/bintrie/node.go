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
	"math/bits"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/blake3"
)

// The EIP-8297 node model. The in-memory tree has four node kinds:
//
//   - branchNode: an upper-tree branch splitting on one bit, carrying the
//     packed run of bits (prefix) shared by every key below it beyond the
//     bits consumed by its ancestors. Both children are always non-empty in
//     canonical form.
//   - groupNode: the subtree spanning every present value of one stem (all
//     keys equal except the final sub-index byte). Its canonical leaf/branch
//     structure is folded on demand by hashAt without materializing nodes.
//     A single-value group is the spec's LeafNode.
//   - hashedNode: an unresolved child reference, loaded lazily by path+hash.
//   - empty: the empty tree; appears transiently during deletion.
//
// Nodes are positionless: algorithms thread the bit position (bits consumed
// above the node) through the walk, and relative branch prefixes plus group
// depths are derived from it.

// nodeTag values prefix every hash preimage and database record.
const (
	tagLeaf   = 0x00
	tagBranch = 0x01
	tagGroup  = 0x02 // database record only, never a hash preimage tag
)

type binaryNode interface {
	// hashAt returns the node's EIP-8297 hash, given the number of key bits
	// consumed by its ancestors. Cached hashes are reused when clean.
	hashAt(pos int) common.Hash

	// copy deep-copies the node (values are treated as immutable).
	copy() binaryNode
}

// branchNode is an upper-tree branch: its split point lies strictly within
// the stem bits, i.e. it separates at least two distinct stems.
type branchNode struct {
	prefix bitstr
	left   binaryNode
	right  binaryNode

	cachedHash common.Hash
	dirty      bool // hash cache invalid
	modified   bool // record changed since load; cleared by commit
}

// groupNode holds the present values of one stem, sparse and sorted by
// sub-index. Value slices are 32 bytes and treated as immutable.
type groupNode struct {
	stem []byte   // full stem: zone byte through byte len(key)-2 (33 or 65 bytes)
	subs []byte   // ascending sub-indices, len >= 1
	vals [][]byte // matching values, each 32 bytes

	cachedHash common.Hash
	dirty      bool // hash cache invalid
	modified   bool // record changed since load; cleared by commit
	cachedAt   int  // position the cache was computed at; group hashes are position-dependent
}

// hashedNode is an unresolved node reference.
type hashedNode common.Hash

// empty is the empty tree.
type empty struct{}

// hashScratch pools the preimage assembly buffers. The largest preimage is a
// branch with a full 527-bit prefix: 1 + 2 + 66 + 32 + 32 = 133 bytes; leaf
// preimages reach 1 + 66 + 32 = 99 bytes.
var hashScratch = sync.Pool{New: func() any { b := make([]byte, 0, 144); return &b }}

// leafHash computes H(LEAF_TAG ‖ stem ‖ sub ‖ value), the hash of the leaf
// holding the complete key stem‖sub.
func leafHash(stem []byte, sub byte, value []byte) common.Hash {
	bufp := hashScratch.Get().(*[]byte)
	buf := (*bufp)[:0]
	buf = append(buf, tagLeaf)
	buf = append(buf, stem...)
	buf = append(buf, sub)
	buf = append(buf, value...)
	h := blake3.Sum256(buf)
	*bufp = buf
	hashScratch.Put(bufp)
	return common.Hash(h)
}

// branchHash computes H(BRANCH_TAG ‖ encode_bit_prefix(prefix) ‖ left ‖ right).
func branchHash(prefix bitstr, left, right common.Hash) common.Hash {
	bufp := hashScratch.Get().(*[]byte)
	buf := (*bufp)[:0]
	buf = append(buf, tagBranch)
	buf = appendBitPrefix(buf, prefix)
	buf = append(buf, left[:]...)
	buf = append(buf, right[:]...)
	h := blake3.Sum256(buf)
	*bufp = buf
	hashScratch.Put(bufp)
	return common.Hash(h)
}

// parallelHashDepth bounds the node depth at which hashing forks: the top
// levels cover the whole tree, so a handful of goroutines saturates the
// available cores without spawning one per node.
const parallelHashDepth = 4

func (n *branchNode) hashAt(pos int) common.Hash {
	return n.hashAtDepth(pos, 0)
}

func (n *branchNode) hashAtDepth(pos, depth int) common.Hash {
	if !n.dirty && n.cachedHash != (common.Hash{}) {
		return n.cachedHash
	}
	childPos := pos + n.prefix.n + 1

	// Fork only near the root, and only when both sides actually need
	// hashing: a clean side returns its cached hash immediately, so a
	// goroutine would cost more than it saves.
	var left, right common.Hash
	if depth < parallelHashDepth && isDirty(n.left) && isDirty(n.right) {
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			left = hashChild(n.left, childPos, depth+1)
		}()
		right = hashChild(n.right, childPos, depth+1)
		wg.Wait()
	} else {
		left = hashChild(n.left, childPos, depth+1)
		right = hashChild(n.right, childPos, depth+1)
	}
	n.cachedHash = branchHash(n.prefix, left, right)
	n.dirty = false
	return n.cachedHash
}

// hashChild hashes a child, threading the node depth through branches so
// the fork budget is spent near the root.
func hashChild(n binaryNode, pos, depth int) common.Hash {
	if b, ok := n.(*branchNode); ok {
		return b.hashAtDepth(pos, depth)
	}
	return n.hashAt(pos)
}

// isDirty reports whether a node's hash still has to be computed.
func isDirty(n binaryNode) bool {
	switch nn := n.(type) {
	case *branchNode:
		return nn.dirty || nn.cachedHash == (common.Hash{})
	case *groupNode:
		return nn.dirty || nn.cachedHash == (common.Hash{})
	default:
		return false
	}
}

func (n *branchNode) copy() binaryNode {
	cp := &branchNode{
		prefix:     bitstr{b: append([]byte{}, n.prefix.b...), n: n.prefix.n},
		left:       n.left.copy(),
		right:      n.right.copy(),
		cachedHash: n.cachedHash,
		dirty:      n.dirty,
		modified:   n.modified,
	}
	return cp
}

func (g *groupNode) hashAt(pos int) common.Hash {
	if !g.dirty && g.cachedHash != (common.Hash{}) && g.cachedAt == pos {
		return g.cachedHash
	}
	g.cachedHash = g.fold(pos)
	g.cachedAt = pos
	g.dirty = false
	return g.cachedHash
}

// fold computes the canonical EIP-8297 hash of the group's leaf set, rooted
// at ancestor-consumed position pos. A single value is a bare leaf (position
// independent); multiple values form branches whose top prefix carries the
// stem's remaining bits.
func (g *groupNode) fold(pos int) common.Hash {
	if len(g.subs) == 1 {
		return leafHash(g.stem, g.subs[0], g.vals[0])
	}
	stemBits := 8 * len(g.stem)
	return g.foldRange(0, len(g.subs), 0, pos, stemBits)
}

// foldRange hashes the canonical subtree over subs[i:j), whose sub-index
// bits agree on [0, from). The subtree root's prefix spans extraLo..extraHi
// of the stem (top call: the stem bits below pos) plus the shared sub bits
// [from, b) where b is the first diverging sub bit.
func (g *groupNode) foldRange(i, j, from, extraLo, extraHi int) common.Hash {
	if j-i == 1 {
		return leafHash(g.stem, g.subs[i], g.vals[i])
	}
	// The extremes bound divergence in a sorted run: the first differing
	// (MSB-first) bit between subs[i] and subs[j-1].
	b := 8 - bits.Len8(g.subs[i]^g.subs[j-1])
	// Split at the first sub with bit b set.
	m := i + 1
	for m < j && g.subs[m]>>(7-b)&1 == 0 {
		m++
	}
	// Assemble the prefix: stem bits [extraLo, extraHi) then sub bits [from, b).
	var prefix bitstr
	if extraHi > extraLo {
		prefix = slice(g.stem, extraLo, extraHi-extraLo)
		for k := from; k < b; k++ {
			prefix = prefix.concat(g.subs[i]>>(7-k)&1, bitstr{})
		}
	} else {
		prefix = slice([]byte{g.subs[i]}, from, b-from)
	}
	left := g.foldRange(i, m, b+1, 0, 0)
	right := g.foldRange(m, j, b+1, 0, 0)
	return branchHash(prefix, left, right)
}

// lookup returns the value at sub, or nil if absent.
func (g *groupNode) lookup(sub byte) []byte {
	lo, hi := 0, len(g.subs)
	for lo < hi {
		mid := (lo + hi) / 2
		if g.subs[mid] < sub {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	if lo < len(g.subs) && g.subs[lo] == sub {
		return g.vals[lo]
	}
	return nil
}

// set inserts or replaces the value at sub. A nil value deletes the entry.
// It reports whether the group still holds any value.
func (g *groupNode) set(sub byte, value []byte) bool {
	lo, hi := 0, len(g.subs)
	for lo < hi {
		mid := (lo + hi) / 2
		if g.subs[mid] < sub {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	present := lo < len(g.subs) && g.subs[lo] == sub
	switch {
	case value == nil && present:
		g.subs = append(g.subs[:lo], g.subs[lo+1:]...)
		g.vals = append(g.vals[:lo], g.vals[lo+1:]...)
	case value == nil:
		// Deleting an absent entry: no-op.
	case present:
		g.vals[lo] = value
	default:
		g.subs = append(g.subs, 0)
		copy(g.subs[lo+1:], g.subs[lo:])
		g.subs[lo] = sub
		g.vals = append(g.vals, nil)
		copy(g.vals[lo+1:], g.vals[lo:])
		g.vals[lo] = value
	}
	g.dirty = true
	g.modified = true
	return len(g.subs) > 0
}

func (g *groupNode) copy() binaryNode {
	cp := &groupNode{
		stem:       append([]byte{}, g.stem...),
		subs:       append([]byte{}, g.subs...),
		vals:       make([][]byte, len(g.vals)),
		cachedHash: g.cachedHash,
		dirty:      g.dirty,
		modified:   g.modified,
		cachedAt:   g.cachedAt,
	}
	copy(cp.vals, g.vals) // values are immutable, share them
	return cp
}

func (h hashedNode) hashAt(int) common.Hash { return common.Hash(h) }
func (h hashedNode) copy() binaryNode       { return h }

func (empty) hashAt(int) common.Hash { return common.Hash{} }
func (empty) copy() binaryNode       { return empty{} }
