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
	"bytes"
	"errors"

	"github.com/ethereum/go-ethereum/common"
)

// OnNode receives every record the builder emits: its database path, hash
// and blob. Records arrive children before parents.
type OnNode func(path []byte, hash common.Hash, blob []byte)

// StackBuilder constructs a binary tree from a strictly-ascending stream of
// key/value pairs in one bottom-up pass with O(depth) memory, emitting each
// database record exactly once as its key range closes. It is the
// stack-trie analogue for the EIP-8297 tree: the ingester for sorted bulk
// loads (state conversion, flat-state verification), where incremental
// insertion would mean billions of random walks.
//
// Nothing calls it yet. The obvious consumer is the offline conversion in
// cmd/geth, which currently inserts one stem at a time, but wiring it in is a
// behavioural change to that command and has not been made. It is kept rather
// than deleted because it is exercised against the engine's own incremental
// path by TestStackBuilderVsIncremental, so it is unused rather than unproven.
// If conversion is benchmarked and still does not use it, delete it.
//
// Invariant: the right spine holds one frame per pending branch, each
// carrying the split bit it branches on and the hash of its completed left
// child. A node's own position is not known when it is built - it depends
// on the divergence bit with the key that comes next - so positions are
// resolved as max(deepest open frame, that divergence) + 1 at close time.
type StackBuilder struct {
	onNode OnNode

	// Pending group: the stem currently being filled.
	stem []byte
	subs []byte
	vals [][]byte

	frames  []builderFrame
	lastKey []byte
	root    common.Hash
	done    bool
}

type builderFrame struct {
	split int         // bit position this branch splits on
	left  common.Hash // hash of the completed left subtree
}

// NewStackBuilder creates a builder streaming records into onNode, which may
// be nil for a hash-only run.
func NewStackBuilder(onNode OnNode) *StackBuilder {
	return &StackBuilder{onNode: onNode}
}

// Add appends the next key/value pair. Keys must be zone-conformant,
// strictly ascending, with 32-byte values.
func (b *StackBuilder) Add(key, value []byte) error {
	if b.done {
		return errors.New("bintrie: builder already finished")
	}
	if err := validateKey(key); err != nil {
		return err
	}
	if len(value) != 32 {
		return errors.New("bintrie: builder values must be 32 bytes")
	}
	if b.lastKey != nil && bytes.Compare(key, b.lastKey) <= 0 {
		return errors.New("bintrie: builder keys must be strictly ascending")
	}
	stem, sub := key[:len(key)-1], key[len(key)-1]

	if b.stem == nil {
		b.stem = append([]byte{}, stem...)
	} else if !bytes.Equal(b.stem, stem) {
		// The pending stem is complete. Everything below the divergence bit
		// closes; the result becomes the left child of a new frame at that
		// bit, whose right child is the stem starting now.
		div := commonPrefixLen(b.lastKey, key, 0)
		left := b.closeBelow(div)
		b.frames = append(b.frames, builderFrame{split: div, left: left})
		b.stem = append(b.stem[:0], stem...)
		b.subs, b.vals = b.subs[:0], b.vals[:0]
	}
	b.subs = append(b.subs, sub)
	b.vals = append(b.vals, append([]byte{}, value...))
	b.lastKey = append(b.lastKey[:0], key...)
	return nil
}

// deepestSplit returns the split bit of the deepest open frame, or -1 when
// the spine is empty.
func (b *StackBuilder) deepestSplit() int {
	if n := len(b.frames); n > 0 {
		return b.frames[n-1].split
	}
	return -1
}

// posBelow returns the position a node occupies given the deepest open
// frame and div, the split of the branch about to be created above it
// (-1 while finishing). A node's parent is whichever of the two is deeper:
// the group closing at div becomes that branch's left child, while a group
// under an already-open deeper frame is that frame's right child.
func (b *StackBuilder) posBelow(div int) int {
	return max(b.deepestSplit(), div) + 1
}

// flushStem hashes and emits the pending stem group at its final position.
func (b *StackBuilder) flushStem(div int) common.Hash {
	pos := b.posBelow(div)
	g := &groupNode{
		stem: b.stem,
		subs: append([]byte{}, b.subs...),
		vals: append([][]byte{}, b.vals...),
	}
	h := g.hashAt(pos)
	if b.onNode != nil {
		b.onNode(encodePath(b.stem, pos), h, serializeNode(g, pos))
	}
	return h
}

// closeBelow finishes the pending stem and folds every frame deeper than
// div, returning the hash of the resulting subtree. The branch prefixes are
// reconstructed from the last key, which shares every bit above each fold's
// split with the subtree below it.
func (b *StackBuilder) closeBelow(div int) common.Hash {
	right := b.flushStem(div)
	rightKey := append([]byte{}, b.lastKey...)
	for len(b.frames) > 0 && b.frames[len(b.frames)-1].split > div {
		frame := b.frames[len(b.frames)-1]
		b.frames = b.frames[:len(b.frames)-1]
		parentPos := b.posBelow(div)
		branch := &branchNode{
			prefix: slice(rightKey, parentPos, frame.split-parentPos),
			left:   hashedNode(frame.left),
			right:  hashedNode(right),
		}
		right = branch.hashAt(parentPos)
		if b.onNode != nil {
			b.onNode(encodePath(rightKey, parentPos), right, serializeNode(branch, parentPos))
		}
	}
	return right
}

// Finish closes every open frame and returns the root hash. The builder is
// unusable afterwards.
func (b *StackBuilder) Finish() common.Hash {
	if b.done {
		return b.root
	}
	b.done = true
	if b.stem == nil {
		return common.Hash{} // nothing added: the empty tree
	}
	b.root = b.closeBelow(-1)
	return b.root
}
