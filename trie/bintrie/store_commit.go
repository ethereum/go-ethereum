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

type nodeFlushFn func(path BitArray, hash common.Hash, serialized []byte)

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
func (s *nodeStore) serializeSubtree(ref nodeRef, remainingDepth int, position int, groupDepth int, bitmap []byte, hashes *[]common.Hash, depths *[]uint8) {
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
			*depths = append(*depths, uint8(groupDepth))
		}
		return
	}

	switch ref.Kind() {
	case kindInternal:
		leftPos := position * 2
		rightPos := position*2 + 1
		s.serializeSubtree(s.getInternal(ref.Index()).left, remainingDepth-1, leftPos, groupDepth, bitmap, hashes, depths)
		s.serializeSubtree(s.getInternal(ref.Index()).right, remainingDepth-1, rightPos, groupDepth, bitmap, hashes, depths)
	case kindEmpty:
		return
	default:
		// StemNode or HashedNode encountered before reaching the group's bottom
		// layer. Compute the leaf bitmap position where this node's hash will
		// be stored.
		bitmapPos := position << remainingDepth
		bitmap[bitmapPos/8] |= 1 << (7 - (bitmapPos % 8))
		*hashes = append(*hashes, s.computeHash(ref))
		*depths = append(*depths, uint8(groupDepth-remainingDepth))
	}
}

// depthBits is the number of bits used to encode one depth offset.
const depthBits = 3

// packedDepthsLen returns the byte length of k packed depth entries
func packedDepthsLen(k int) int {
	return (k*depthBits + 7) / 8
}

// writeDepth writes a depth entry at idx into the buf, MSB-first.
func writeDepth(buf []byte, idx int, v uint8) {
	pos := idx * depthBits
	for i := range depthBits {
		bit := (v >> (depthBits - 1 - i)) & 1
		p := pos + i
		buf[p>>3] |= bit << (7 - (p & 7))
	}
}

// readDepth reads a depth for entry idx from buf.
func readDepth(buf []byte, idx int) uint8 {
	pos := idx * depthBits
	var v uint8
	for i := range depthBits {
		p := pos + i
		bit := (buf[p>>3] >> (7 - (p & 7))) & 1
		v = v<<1 | bit
	}
	return v
}

// SerializeNode serializes a node into the flat on-disk format.
func (s *nodeStore) serializeNode(ref nodeRef, groupDepth int) []byte {
	switch ref.Kind() {
	case kindInternal:
		// InternalNode group format:
		//   [type(1)] [groupDepth(1)] [bitmap (2^groupDepth bits)] [depths(3 bits × K, padded)] [hashes(32B × K)]
		bitmapSize := bitmapSizeForDepth(groupDepth)
		bitmap := make([]byte, bitmapSize)
		var hashes []common.Hash
		var depths []uint8

		s.serializeSubtree(ref, groupDepth, 0, groupDepth, bitmap, &hashes, &depths)

		// Build serialized output
		k := len(hashes)
		depthsLen := packedDepthsLen(k)
		serializedLen := NodeTypeBytes + 1 + bitmapSize + depthsLen + k*HashSize
		serialized := make([]byte, serializedLen)
		serialized[0] = nodeTypeInternal
		serialized[1] = byte(groupDepth)
		copy(serialized[2:2+bitmapSize], bitmap)

		depthsOff := NodeTypeBytes + 1 + bitmapSize
		for i, d := range depths {
			writeDepth(serialized[depthsOff:depthsOff+depthsLen], i, d-1)
		}

		hashesOff := depthsOff + depthsLen
		for i, h := range hashes {
			copy(serialized[hashesOff+i*HashSize:hashesOff+(i+1)*HashSize], h.Bytes())
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
func (s *nodeStore) deserializeSubtree(hn common.Hash, groupDepth int, nodeDepth int, bitmap []byte, depths []byte, hashData []byte, mustRecompute bool, dirty bool) (nodeRef, error) {
	if len(hashData)%HashSize != 0 {
		return emptyRef, errInvalidSerializedLength
	}
	k := len(hashData) / HashSize
	if len(depths) != packedDepthsLen(k) {
		return emptyRef, errInvalidSerializedLength
	}
	if k == 0 {
		return emptyRef, nil
	}

	rootRef := s.newInternalRef(nodeDepth)
	rootNode := s.getInternal(rootRef.Index())
	rootNode.mustRecompute = mustRecompute
	if !mustRecompute {
		rootNode.hash = hn
	}
	rootNode.dirty = dirty

	bitmapBits := 1 << groupDepth
	entryIdx := 0
	for bit := 0; bit < bitmapBits; bit++ {
		if bitmap[bit/8]>>(7-(bit%8))&1 == 0 {
			continue
		}
		depthOffset := int(readDepth(depths, entryIdx)) + 1
		if depthOffset > groupDepth {
			return emptyRef, errors.New("invalid depth offset")
		}
		// Canonical-encoding check: trailing position bits must be zero.
		mask := (1 << (groupDepth - depthOffset)) - 1
		if bit&mask != 0 {
			return emptyRef, errors.New("non-canonical bitmap position")
		}
		var hash common.Hash
		copy(hash[:], hashData[entryIdx*HashSize:(entryIdx+1)*HashSize])
		if err := s.attachInGroup(rootRef, nodeDepth, groupDepth, depthOffset, bit, hash, dirty); err != nil {
			return emptyRef, err
		}
		entryIdx++
	}
	return rootRef, nil
}

func (s *nodeStore) attachInGroup(rootRef nodeRef, rootDepth, groupDepth, depthOffset, bitmapPos int, hash common.Hash, dirty bool) error {
	cur := rootRef
	for level := 0; level < depthOffset-1; level++ {
		bit := (bitmapPos >> (groupDepth - 1 - level)) & 1
		node := s.getInternal(cur.Index())
		childRef := node.left
		if bit == 1 {
			childRef = node.right
		}
		if childRef.IsEmpty() {
			newRef := s.newInternalRef(rootDepth + level + 1)
			s.getInternal(newRef.Index()).dirty = dirty
			if bit == 0 {
				node.left = newRef
			} else {
				node.right = newRef
			}
			cur = newRef
			continue
		}
		if childRef.Kind() != kindInternal {
			return errors.New("overlapping entries in group blob")
		}
		cur = childRef
	}
	leafBit := (bitmapPos >> (groupDepth - depthOffset)) & 1
	node := s.getInternal(cur.Index())
	if leafBit == 0 {
		if !node.left.IsEmpty() {
			return errors.New("overlapping entries in group blob")
		}
		node.left = s.newHashedRef(hash)
	} else {
		if !node.right.IsEmpty() {
			return errors.New("overlapping entries in group blob")
		}
		node.right = s.newHashedRef(hash)
	}
	return nil
}

func (s *nodeStore) decodeNode(serialized []byte, depth int, hn common.Hash, mustRecompute, dirty bool) (nodeRef, error) {
	if len(serialized) == 0 {
		return emptyRef, nil
	}

	switch serialized[0] {
	case nodeTypeInternal:
		// Grouped format:
		//   [type(1)] [groupDepth(1)] [bitmap (2^groupDepth bits, padded to bitmapSize bytes)]
		//   [depthOffsets (3 bits × K, padded to bytes)] [hashes (32B × K)]
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

		bitmapBits := 1 << groupDepth
		if bitmapBits < 8 {
			padMask := byte(0xFF) >> bitmapBits
			if bitmap[0]&padMask != 0 {
				return emptyRef, errors.New("non-canonical bitmap padding")
			}
		}

		k := 0
		for _, b := range bitmap {
			k += bits.OnesCount8(b)
		}
		depthsLen := packedDepthsLen(k)
		expectedLen := NodeTypeBytes + 1 + bitmapSize + depthsLen + k*HashSize
		if len(serialized) != expectedLen {
			return emptyRef, errInvalidSerializedLength
		}
		depthsOff := NodeTypeBytes + 1 + bitmapSize
		depths := serialized[depthsOff : depthsOff+depthsLen]
		hashData := serialized[depthsOff+depthsLen : depthsOff+depthsLen+k*HashSize]

		// Canonical-encoding check: the unused low bits of the last packed
		// depth byte must be zero.
		if usedBits := k * depthBits; usedBits%8 != 0 {
			padMask := byte(0xFF) >> (usedBits % 8)
			if depths[depthsLen-1]&padMask != 0 {
				return emptyRef, errors.New("non-canonical depth padding")
			}
		}

		return s.deserializeSubtree(hn, groupDepth, depth, bitmap, depths, hashData, mustRecompute, dirty)

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
//
// BitArray is passed by value (33 bytes) to keep child paths on the stack.
// Passing by pointer causes escape to heap per recursive call.
func (s *nodeStore) collectNodes(ref nodeRef, path BitArray, flushfn nodeFlushFn, groupDepth int) {
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
func (s *nodeStore) collectChildGroups(node *InternalNode, path BitArray, flushfn nodeFlushFn, groupDepth int, remainingLevels int) error {
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
			s.collectNodes(node.left, appendBit(path, 0), flushfn, groupDepth)
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
			s.collectNodes(node.right, appendBit(path, 1), flushfn, groupDepth)
		}
	}
	return nil
}

// appendBit returns a new BitArray with bit appended to path.
func appendBit(path BitArray, bit uint8) BitArray {
	var p BitArray
	p.AppendBit(&path, bit)
	return p
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
