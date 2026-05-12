// Copyright 2026 go-ethereum Authors
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
	"crypto/sha256"
	"errors"
	"fmt"
	"math/bits"
	"runtime"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

type nodeFlushFn func(path []byte, hash common.Hash, serialized []byte)

func (s *nodeStore) Hash() common.Hash {
	return s.computeHash(s.root)
}

func (s *nodeStore) computeHash(ref nodeRef) common.Hash {
	switch ref.Kind() {
	case kindInternal:
		return s.hashInternal(ref.Index())
	case kindStem:
		return s.getStem(ref.Index()).Hash()
	case kindHashed:
		return s.getHashed(ref.Index()).Hash()
	case kindEmpty:
		return common.Hash{}
	default:
		return common.Hash{}
	}
}

// parallelHashDepth is the tree depth below which hashInternal spawns
// goroutines for shallow-depth parallelism. Computed once at init because
// NumCPU() never changes after startup.
var parallelHashDepth = min(bits.Len(uint(runtime.NumCPU())), 8)

// hashInternal hashes an InternalNode and caches the result.
//
// At shallow depths (< parallelHashDepth) the left subtree is hashed in a
// goroutine while the right subtree is hashed inline, then the two digests
// are combined. Below that threshold the goroutine spawn cost outweighs the
// hashing work, so deeper nodes hash both children sequentially.
func (s *nodeStore) hashInternal(idx uint32) common.Hash {
	node := s.getInternal(idx)
	if !node.mustRecompute {
		return node.hash
	}

	if int(node.depth) < parallelHashDepth {
		var input [64]byte
		var lh common.Hash
		var wg sync.WaitGroup
		if !node.left.IsEmpty() {
			wg.Add(1)
			go func() {
				// defer wg.Done() so a panic in computeHash still releases
				// the waiter; without this, a recover() higher in the call
				// stack would leave the parent stuck in wg.Wait forever.
				defer wg.Done()
				lh = s.computeHash(node.left)
			}()
		}
		if !node.right.IsEmpty() {
			rh := s.computeHash(node.right)
			copy(input[32:], rh[:])
		}
		wg.Wait()
		copy(input[:32], lh[:])
		node.hash = sha256.Sum256(input[:])
		node.mustRecompute = false
		return node.hash
	}

	// Deep sequential branch — mirrors the shallow branch's shape to keep
	// input on the stack. Writing lh/rh through hash.Hash (interface)
	// forces escape; copy into a local [64]byte and hash it in one shot.
	var input [64]byte
	if !node.left.IsEmpty() {
		lh := s.computeHash(node.left)
		copy(input[:HashSize], lh[:])
	}
	if !node.right.IsEmpty() {
		rh := s.computeHash(node.right)
		copy(input[HashSize:], rh[:])
	}
	node.hash = sha256.Sum256(input[:])
	node.mustRecompute = false
	return node.hash
}

// serializeSubtree recursively collects child hashes from a subtree of InternalNodes.
// It traverses up to `remainingDepth` levels, storing hashes of bottom-layer children.
// position tracks the current index (0 to 2^groupDepth - 1) for bitmap placement.
// hashes collects the hashes of present children, bitmap tracks which positions are present.
func (s *nodeStore) serializeSubtree(ref nodeRef, remainingDepth int, position int, absoluteDepth int, bitmap []byte, hashes *[]common.Hash) {
	if remainingDepth == 0 {
		// Bottom layer: store hash if not empty
		switch ref.Kind() {
		case kindEmpty:
			// Leave bitmap bit unset, don't add hash
			return
		default:
			// StemNode, HashedNode, or InternalNode at boundary: store hash
			bitmap[position/8] |= 1 << (7 - (position % 8))
			*hashes = append(*hashes, s.computeHash(ref))
		}
		return
	}

	switch ref.Kind() {
	case kindInternal:
		leftPos := position * 2
		rightPos := position*2 + 1
		s.serializeSubtree(s.getInternal(ref.Index()).left, remainingDepth-1, leftPos, absoluteDepth+1, bitmap, hashes)
		s.serializeSubtree(s.getInternal(ref.Index()).right, remainingDepth-1, rightPos, absoluteDepth+1, bitmap, hashes)
	case kindEmpty:
		return
	default:
		// StemNode or HashedNode encountered before reaching the group's bottom
		// layer. Compute the leaf bitmap position where this node's hash will
		// be stored.
		leafPos := position
		switch ref.Kind() {
		case kindStem:
			sn := s.getStem(ref.Index())
			// Extend position using the stem's key bits so that
			// GetValuesAtStem traversal (which follows key bits) finds the hash.
			for d := 0; d < remainingDepth; d++ {
				bit := sn.Stem[(absoluteDepth+d)/8] >> (7 - ((absoluteDepth + d) % 8)) & 1
				leafPos = leafPos*2 + int(bit)
			}
		default:
			// HashedNode or unknown: extend all-left (no key bits available).
			// This matches the all-zero path that resolveNode would follow.
			leafPos = position << remainingDepth
		}
		bitmap[leafPos/8] |= 1 << (7 - (leafPos % 8))
		*hashes = append(*hashes, s.computeHash(ref))
	}
}

// SerializeNode serializes a node into the flat on-disk format.
func (s *nodeStore) serializeNode(ref nodeRef, groupDepth int) []byte {
	switch ref.Kind() {
	case kindInternal:
		// InternalNode group: 1 byte type + 1 byte group depth + variable bitmap + N×32 byte hashes
		bitmapSize := bitmapSizeForDepth(groupDepth)
		bitmap := make([]byte, bitmapSize)
		var hashes []common.Hash

		node := s.getInternal(ref.Index())
		s.serializeSubtree(ref, groupDepth, 0, int(node.depth), bitmap, &hashes)

		// Build serialized output
		serializedLen := NodeTypeBytes + 1 + bitmapSize + len(hashes)*HashSize
		serialized := make([]byte, serializedLen)
		serialized[0] = nodeTypeInternal
		serialized[1] = byte(groupDepth) // group depth => bitmap size for a sparse group
		copy(serialized[2:2+bitmapSize], bitmap)

		offset := NodeTypeBytes + 1 + bitmapSize
		for _, h := range hashes {
			copy(serialized[offset:offset+HashSize], h.Bytes())
			offset += HashSize
		}

		return serialized

	case kindStem:
		sn := s.getStem(ref.Index())
		// Count present slots to size the blob.
		var count int
		for _, v := range sn.values {
			if v != nil {
				count++
			}
		}
		serializedLen := NodeTypeBytes + StemSize + StemBitmapSize + count*HashSize
		serialized := make([]byte, serializedLen)
		serialized[0] = nodeTypeStem
		copy(serialized[NodeTypeBytes:NodeTypeBytes+StemSize], sn.Stem[:])
		bitmap := serialized[NodeTypeBytes+StemSize : NodeTypeBytes+StemSize+StemBitmapSize]
		offset := NodeTypeBytes + StemSize + StemBitmapSize
		for i, v := range sn.values {
			if v != nil {
				bitmap[i/8] |= 1 << (7 - (i % 8))
				copy(serialized[offset:offset+HashSize], v)
				offset += HashSize
			}
		}
		return serialized

	default:
		panic(fmt.Sprintf("SerializeNode: unexpected node kind %d", ref.Kind()))
	}
}

var errInvalidSerializedLength = errors.New("invalid serialized node length")

// DeserializeNode deserializes a node from bytes, recomputing its hash. The
// returned node is marked dirty (provenance unknown, safe re-flush default).
func (s *nodeStore) deserializeNode(serialized []byte, depth int) (nodeRef, error) {
	return s.decodeNode(serialized, depth, common.Hash{}, true, true)
}

// DeserializeNodeWithHash deserializes a node whose hash is already known and
// whose blob is already on disk (mustRecompute=false, dirty=false).
func (s *nodeStore) deserializeNodeWithHash(serialized []byte, depth int, hn common.Hash) (nodeRef, error) {
	return s.decodeNode(serialized, depth, hn, false, false)
}

// deserializeSubtree reconstructs an InternalNode subtree from grouped serialization.
// remainingDepth is how many more levels to build, position is current index in the bitmap,
// nodeDepth is the actual trie depth for the node being created.
// hashIdx tracks the current position in the hash data (incremented as hashes are consumed).
func (s *nodeStore) deserializeSubtree(hn common.Hash, remainingDepth int, position int, nodeDepth int, bitmap []byte, hashData []byte, hashIdx *int, mustRecompute bool, dirty bool) (nodeRef, error) {
	if remainingDepth == 0 {
		// Bottom layer: check bitmap and return HashedNode or Empty
		if bitmap[position/8]>>(7-(position%8))&1 == 1 {
			if len(hashData) < (*hashIdx+1)*HashSize {
				return emptyRef, errInvalidSerializedLength
			}
			hash := common.BytesToHash(hashData[*hashIdx*HashSize : (*hashIdx+1)*HashSize])
			*hashIdx++
			return s.newHashedRef(hash), nil
		}
		return emptyRef, nil
	}

	// Check if this entire subtree is empty by examining all relevant bitmap bits
	leftPos := position * 2
	rightPos := position*2 + 1

	// note that the parent might not need root computations, but the children
	// do, because their hash isn't saved. Hence `mustRecompute` is set to `true`.
	left, err := s.deserializeSubtree(common.Hash{}, remainingDepth-1, leftPos, nodeDepth+1, bitmap, hashData, hashIdx, true, dirty)
	if err != nil {
		return emptyRef, err
	}
	right, err := s.deserializeSubtree(common.Hash{}, remainingDepth-1, rightPos, nodeDepth+1, bitmap, hashData, hashIdx, true, dirty)
	if err != nil {
		return emptyRef, err
	}

	// If both children are empty, return Empty
	if left.IsEmpty() && right.IsEmpty() {
		return emptyRef, nil
	}

	ref := s.newInternalRef(nodeDepth)
	node := s.getInternal(ref.Index())
	node.left = left
	node.right = right
	node.mustRecompute = mustRecompute
	if !mustRecompute {
		// mustRecompute will only be false for the root of the subtree,
		// for which we already know the hash.
		node.hash = hn
		node.mustRecompute = false
	}
	node.dirty = dirty
	return ref, nil
}

func (s *nodeStore) decodeNode(serialized []byte, depth int, hn common.Hash, mustRecompute, dirty bool) (nodeRef, error) {
	if len(serialized) == 0 {
		return emptyRef, nil
	}

	switch serialized[0] {
	case nodeTypeInternal:
		// Grouped format: 1 byte type + 1 byte group depth + variable bitmap + N×32 byte hashes
		if len(serialized) < NodeTypeBytes+1 {
			return emptyRef, errInvalidSerializedLength
		}
		groupDepth := int(serialized[1])
		if groupDepth < 1 || groupDepth > MaxGroupDepth {
			return 0, errors.New("invalid group depth")
		}
		bitmapSize := bitmapSizeForDepth(groupDepth)
		if len(serialized) < NodeTypeBytes+1+bitmapSize {
			return 0, errInvalidSerializedLength
		}
		bitmap := serialized[2 : 2+bitmapSize]
		hashData := serialized[2+bitmapSize:]

		hashIdx := 0
		return s.deserializeSubtree(hn, groupDepth, 0, depth, bitmap, hashData, &hashIdx, mustRecompute, dirty)

	case nodeTypeStem:
		if len(serialized) < NodeTypeBytes+StemSize+StemBitmapSize {
			return emptyRef, errInvalidSerializedLength
		}
		stemIdx := s.allocStem()
		sn := s.getStem(stemIdx)
		copy(sn.Stem[:], serialized[NodeTypeBytes:NodeTypeBytes+StemSize])
		bitmap := serialized[NodeTypeBytes+StemSize : NodeTypeBytes+StemSize+StemBitmapSize]
		offset := NodeTypeBytes + StemSize + StemBitmapSize
		for i := range StemNodeWidth {
			if bitmap[i/8]>>(7-(i%8))&1 != 1 {
				continue
			}
			if len(serialized) < offset+HashSize {
				return emptyRef, errInvalidSerializedLength
			}
			// Zero-copy: each slot aliases the serialized input buffer.
			sn.values[i] = serialized[offset : offset+HashSize]
			offset += HashSize
		}
		sn.depth = uint8(depth)
		sn.hash = hn
		sn.mustRecompute = mustRecompute
		sn.dirty = dirty
		return makeRef(kindStem, stemIdx), nil

	default:
		return emptyRef, errors.New("invalid node type")
	}
}

// CollectNodes flushes every node that needs flushing via flushfn in post-order.
// Invariant: any ancestor of a node that needs flushing is itself marked, so a
// clean root means the whole subtree is clean.
func (s *nodeStore) collectNodes(ref nodeRef, path []byte, flushfn nodeFlushFn, groupDepth int) {
	switch ref.Kind() {
	case kindInternal:
		node := s.getInternal(ref.Index())
		if !node.dirty {
			return
		}
		// Only flush at group boundaries (depth % groupDepth == 0)
		if int(node.depth)%groupDepth == 0 {
			// We're at a group boundary - first collect any nodes in deeper groups,
			// then flush this group
			s.collectChildGroups(node, path, flushfn, groupDepth, groupDepth-1)
			flushfn(path, s.computeHash(ref), s.serializeNode(ref, groupDepth))
			node.dirty = false
			return
		}
		// Not at a group boundary - this shouldn't happen if we're called correctly from root
		// but handle it by continuing to traverse
		s.collectChildGroups(node, path, flushfn, groupDepth, groupDepth-(int(node.depth)%groupDepth)-1)
	case kindStem:
		sn := s.getStem(ref.Index())
		if !sn.dirty {
			return
		}
		flushfn(path, s.computeHash(ref), s.serializeNode(ref, groupDepth))
		sn.dirty = false
	case kindHashed, kindEmpty:
	default:
		panic(fmt.Sprintf("CollectNodes: unexpected kind %d", ref.Kind()))
	}
}

// collectChildGroups traverses within a group to find and collect nodes in the next group.
// remainingLevels is how many more levels below the current node until we reach the group boundary.
// When remainingLevels=0, the current node's children are at the next group boundary.
func (s *nodeStore) collectChildGroups(node *InternalNode, path []byte, flushfn nodeFlushFn, groupDepth int, remainingLevels int) error {
	if remainingLevels == 0 {
		// Current node is at depth (groupBoundary - 1), its children are at the next group boundary
		if !node.left.IsEmpty() {
			s.collectNodes(node.left, appendBit(path, 0), flushfn, groupDepth)
		}
		if !node.right.IsEmpty() {
			s.collectNodes(node.right, appendBit(path, 1), flushfn, groupDepth)
		}
		return nil
	}

	if !node.left.IsEmpty() {
		switch node.left.Kind() {
		case kindInternal:
			n := s.getInternal(node.left.Index())
			if err := s.collectChildGroups(n, appendBit(path, 0), flushfn, groupDepth, remainingLevels-1); err != nil {
				return err
			}
		default:
			extPath := s.extendPathToGroupLeaf(appendBit(path, 0), node.left, remainingLevels)
			s.collectNodes(node.left, extPath, flushfn, groupDepth)
		}
	}
	if !node.right.IsEmpty() {
		switch node.right.Kind() {
		case kindInternal:
			n := s.getInternal(node.right.Index())
			if err := s.collectChildGroups(n, appendBit(path, 1), flushfn, groupDepth, remainingLevels-1); err != nil {
				return err
			}
		default:
			extPath := s.extendPathToGroupLeaf(appendBit(path, 1), node.right, remainingLevels)
			s.collectNodes(node.right, extPath, flushfn, groupDepth)
		}
	}
	return nil
}

// extendPathToGroupLeaf extends a storage path to the group's leaf boundary,
// matching the projection done by serializeSubtree. For StemNodes, the path
// is extended using the stem's key bits (same as serializeSubtree). For other
// node types, the path is extended with all-zero (left) bits.
func (s *nodeStore) extendPathToGroupLeaf(path []byte, node nodeRef, remainingLevels int) []byte {
	if remainingLevels <= 0 {
		return path
	}
	if node.Kind() == kindStem {
		sn := s.getStem(node.Index())
		for _ = range remainingLevels {
			bit := sn.Stem[len(path)/8] >> (7 - (len(path) % 8)) & 1
			path = appendBit(path, bit)
		}
	} else {
		// HashedNode or other: all-left extension (matches serializeSubtree's
		// position << remainingDepth behavior).
		for _ = range remainingLevels {
			path = appendBit(path, 0)
		}
	}
	return path
}

// appendBit appends a bit to a path, returning a new slice
func appendBit(path []byte, bit byte) []byte {
	var p [256]byte
	copy(p[:], path)
	result := p[:len(path)]
	return append(result, bit)
}

func (s *nodeStore) toDot(ref nodeRef, parent, path string) string {
	switch ref.Kind() {
	case kindInternal:
		node := s.getInternal(ref.Index())
		me := fmt.Sprintf("internal%s", path)
		ret := fmt.Sprintf("%s [label=\"I: %x\"]\n", me, s.computeHash(ref))
		if len(parent) > 0 {
			ret = fmt.Sprintf("%s %s -> %s\n", ret, parent, me)
		}
		if !node.left.IsEmpty() {
			ret += s.toDot(node.left, me, fmt.Sprintf("%s%b", path, 0))
		}
		if !node.right.IsEmpty() {
			ret += s.toDot(node.right, me, fmt.Sprintf("%s%b", path, 1))
		}
		return ret
	case kindStem:
		sn := s.getStem(ref.Index())
		me := fmt.Sprintf("stem%s", path)
		ret := fmt.Sprintf("%s [label=\"stem=%x c=%x\"]\n", me, sn.Stem, sn.Hash())
		ret = fmt.Sprintf("%s %s -> %s\n", ret, parent, me)
		for i, v := range sn.values {
			if v == nil {
				continue
			}
			ret += fmt.Sprintf("%s%x [label=\"%x\"]\n", me, i, v)
			ret += fmt.Sprintf("%s -> %s%x\n", me, me, i)
		}
		return ret
	case kindHashed:
		hn := s.getHashed(ref.Index())
		me := fmt.Sprintf("hash%s", path)
		ret := fmt.Sprintf("%s [label=\"%x\"]\n", me, hn.Hash())
		ret = fmt.Sprintf("%s %s -> %s\n", ret, parent, me)
		return ret
	default:
		return ""
	}
}
