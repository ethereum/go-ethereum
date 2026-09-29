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
// and blob. Records arrive children before parents. path and blob are only
// valid for the call: the builder reuses their backing arrays for the next
// record, so a callback that needs one past its return must copy it.
type OnNode func(path []byte, hash common.Hash, blob []byte)

// StackBuilder constructs a binary tree from a strictly-ascending stream of
// key/value pairs in one bottom-up pass with O(depth) memory, emitting each
// database record exactly once as its key range closes. It is the
// stack-trie analogue for the EIP-8297 tree: the ingester for sorted bulk
// loads (state conversion, flat-state verification), where incremental
// insertion would mean billions of random walks.
//
// Invariant: the right spine holds one frame per pending branch, each
// carrying the split bit it branches on and the hash of its completed left
// child. A node's own position is not known when it is built - it depends
// on the divergence bit with the key that comes next - so positions are
// resolved as max(deepest open frame, that divergence) + 1 at close time.
type StackBuilder struct {
	onNode OnNode

	// Pending group: the stem currently being filled.
	stem     []byte
	subs     []byte
	valArena []byte // pending group's 32-byte values, packed back to back

	frames  []builderFrame
	lastKey []byte
	root    common.Hash
	done    bool

	// Scratch reused across every emitted record: each node is built,
	// hashed and handed to onNode once, so one instance of each serves the
	// whole build, marked dirty before every use. valViews slices valArena
	// into the [][]byte groupNode.vals wants.
	groupScratch  groupNode
	branchScratch branchNode
	valViews      [][]byte
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
// strictly ascending, with 32-byte values. Add copies the value into its own
// arena; the caller's slices are free to reuse the moment Add returns.
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
	if isZeroValue(value) {
		// Zero values resolve to absence; folding one would bless corruption.
		return errors.New("bintrie: builder values must not be 32 zero bytes")
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
		b.subs, b.valArena = b.subs[:0], b.valArena[:0]
	}
	b.subs = append(b.subs, sub)
	b.valArena = append(b.valArena, value...)
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
	n := len(b.subs)
	b.valViews = b.valViews[:0]
	for i := range n {
		b.valViews = append(b.valViews, b.valArena[i*32:i*32+32])
	}
	b.groupScratch.stem = b.stem
	b.groupScratch.subs = b.subs
	b.groupScratch.vals = b.valViews
	b.groupScratch.dirty = true
	h := b.groupScratch.hashAt(pos)
	if b.onNode != nil {
		b.onNode(encodePath(b.stem, pos), h, serializeNode(&b.groupScratch, pos))
	}
	return h
}

// closeBelow finishes the pending stem and folds every frame deeper than
// div, returning the hash of the resulting subtree. The branch prefixes are
// reconstructed from the last key, which shares every bit above each fold's
// split with the subtree below it. b.lastKey is stable for the duration of
// this call - Add only overwrites it once closeBelow has returned - so the
// fold reads it directly rather than through a defensive copy.
func (b *StackBuilder) closeBelow(div int) common.Hash {
	right := b.flushStem(div)
	for len(b.frames) > 0 && b.frames[len(b.frames)-1].split > div {
		frame := b.frames[len(b.frames)-1]
		b.frames = b.frames[:len(b.frames)-1]
		parentPos := b.posBelow(div)
		b.branchScratch.prefix = slice(b.lastKey, parentPos, frame.split-parentPos)
		b.branchScratch.left = hashedNode(frame.left)
		b.branchScratch.right = hashedNode(right)
		b.branchScratch.dirty = true
		right = b.branchScratch.hashAt(parentPos)
		if b.onNode != nil {
			b.onNode(encodePath(b.lastKey, parentPos), right, serializeNode(&b.branchScratch, parentPos))
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
