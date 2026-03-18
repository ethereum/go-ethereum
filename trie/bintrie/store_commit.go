// Copyright 2025 go-ethereum Authors
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
	"errors"
	"fmt"
	"math/bits"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

// NodeFlushFn is called during commit to flush serialized nodes.
type NodeFlushFn func(path []byte, hash common.Hash, serialized []byte)

// Hash computes and returns the root hash.
func (s *NodeStore) Hash() common.Hash {
	return s.ComputeHash(s.root)
}

// ComputeHash computes the hash of the node referenced by ref.
func (s *NodeStore) ComputeHash(ref NodeRef) common.Hash {
	switch ref.Kind() {
	case KindInternal:
		return s.hashInternal(ref.Index())
	case KindStem:
		return s.getStem(ref.Index()).Hash()
	case KindHashed:
		return s.getHashed(ref.Index()).hash
	case KindEmpty:
		return common.Hash{}
	default:
		return common.Hash{}
	}
}

// hashInternal computes the hash of an InternalNode. At shallow depths
// (< parallelHashDepth), the left subtree is hashed in a goroutine while
// the right subtree is hashed inline. This is safe because left and right
// subtrees are disjoint in a well-formed tree — no node appears in both.
// ComputeHash must not be called concurrently with mutations to the NodeStore.
func (s *NodeStore) hashInternal(idx uint32) common.Hash {
	node := s.getInternal(idx)
	if !node.mustRecompute {
		return node.hash
	}

	if node.depth < parallelHashDepth {
		var input [64]byte
		var lh common.Hash
		var wg sync.WaitGroup
		if !node.left.IsEmpty() {
			wg.Add(1)
			go func() {
				lh = s.ComputeHash(node.left)
				wg.Done()
			}()
		}
		if !node.right.IsEmpty() {
			rh := s.ComputeHash(node.right)
			copy(input[32:], rh[:])
		}
		wg.Wait()
		copy(input[:32], lh[:])
		node.hash = sha256Sum256(input[:])
		node.mustRecompute = false
		return node.hash
	}

	var input [64]byte
	if !node.left.IsEmpty() {
		lh := s.ComputeHash(node.left)
		copy(input[:32], lh[:])
	}
	if !node.right.IsEmpty() {
		rh := s.ComputeHash(node.right)
		copy(input[32:], rh[:])
	}
	node.hash = sha256Sum256(input[:])
	node.mustRecompute = false
	return node.hash
}

// --- Serialization ---

// countSubtreeChildren counts non-empty children at the bottom layer of a subtree.
func (s *NodeStore) countSubtreeChildren(ref NodeRef, remainingDepth int) int {
	if remainingDepth == 0 {
		if ref.IsEmpty() {
			return 0
		}
		return 1
	}
	if ref.Kind() == KindInternal {
		node := s.getInternal(ref.Index())
		return s.countSubtreeChildren(node.left, remainingDepth-1) + s.countSubtreeChildren(node.right, remainingDepth-1)
	}
	if ref.IsEmpty() {
		return 0
	}
	return 1
}

// serializeSubtreeDirect writes child hashes directly into the serialized buffer.
func (s *NodeStore) serializeSubtreeDirect(ref NodeRef, remainingDepth int, position int, absoluteDepth int, bitmap []byte, out []byte, hashOffset *int) {
	if remainingDepth == 0 {
		if ref.IsEmpty() {
			return
		}
		bitmap[position/8] |= 1 << (7 - (position % 8))
		h := s.ComputeHash(ref)
		copy(out[*hashOffset:*hashOffset+HashSize], h[:])
		*hashOffset += HashSize
		return
	}

	if ref.Kind() == KindInternal {
		node := s.getInternal(ref.Index())
		leftPos := position * 2
		rightPos := position*2 + 1
		s.serializeSubtreeDirect(node.left, remainingDepth-1, leftPos, absoluteDepth+1, bitmap, out, hashOffset)
		s.serializeSubtreeDirect(node.right, remainingDepth-1, rightPos, absoluteDepth+1, bitmap, out, hashOffset)
		return
	}

	if ref.IsEmpty() {
		return
	}

	// Leaf (StemNode or HashedNode) at a non-zero remaining depth: compute its position.
	leafPos := position
	if ref.Kind() == KindStem {
		sn := s.getStem(ref.Index())
		for d := 0; d < remainingDepth; d++ {
			bit := sn.Stem[(absoluteDepth+d)/8] >> (7 - ((absoluteDepth + d) % 8)) & 1
			leafPos = leafPos*2 + int(bit)
		}
	} else {
		leafPos = position << remainingDepth
	}
	bitmap[leafPos/8] |= 1 << (7 - (leafPos % 8))
	h := s.ComputeHash(ref)
	copy(out[*hashOffset:*hashOffset+HashSize], h[:])
	*hashOffset += HashSize
}

// SerializeNode serializes a node referenced by ref.
func (s *NodeStore) SerializeNode(ref NodeRef, groupDepth int) []byte {
	if groupDepth < 1 || groupDepth > MaxGroupDepth {
		panic("groupDepth must be between 1 and 8")
	}

	switch ref.Kind() {
	case KindInternal:
		node := s.getInternal(ref.Index())
		bitmapSize := BitmapSizeForDepth(groupDepth)
		hashCount := s.countSubtreeChildren(ref, groupDepth)
		serializedLen := NodeTypeBytes + 1 + bitmapSize + hashCount*HashSize
		serialized := make([]byte, serializedLen)
		serialized[0] = nodeTypeInternal
		serialized[1] = byte(groupDepth)
		bitmap := serialized[2 : 2+bitmapSize]
		hashOffset := 2 + bitmapSize
		s.serializeSubtreeDirect(ref, groupDepth, 0, int(node.depth), bitmap, serialized, &hashOffset)
		return serialized

	case KindStem:
		sn := s.getStem(ref.Index())
		serializedLen := NodeTypeBytes + StemSize + StemBitmapSize + len(sn.valueData)
		serialized := make([]byte, serializedLen)
		serialized[0] = nodeTypeStem
		copy(serialized[NodeTypeBytes:NodeTypeBytes+StemSize], sn.Stem[:])
		copy(serialized[NodeTypeBytes+StemSize:NodeTypeBytes+StemSize+StemBitmapSize], sn.bitmap[:])
		copy(serialized[NodeTypeBytes+StemSize+StemBitmapSize:], sn.valueData)
		return serialized

	default:
		panic(fmt.Sprintf("SerializeNode: unexpected node kind %d", ref.Kind()))
	}
}

// --- Deserialization ---

var invalidSerializedLength = errors.New("invalid serialized node length")

// DeserializeNode deserializes a node from bytes, recomputing its hash.
func (s *NodeStore) DeserializeNode(serialized []byte, depth int) (NodeRef, error) {
	return s.deserializeNode(serialized, depth, common.Hash{}, true)
}

// DeserializeNodeWithHash deserializes a node, using the provided hash.
func (s *NodeStore) DeserializeNodeWithHash(serialized []byte, depth int, hn common.Hash) (NodeRef, error) {
	return s.deserializeNode(serialized, depth, hn, false)
}

func (s *NodeStore) deserializeNode(serialized []byte, depth int, hn common.Hash, mustRecompute bool) (NodeRef, error) {
	if len(serialized) == 0 {
		return EmptyRef, nil
	}

	switch serialized[0] {
	case nodeTypeInternal:
		if len(serialized) < NodeTypeBytes+1 {
			return EmptyRef, invalidSerializedLength
		}
		groupDepth := int(serialized[1])
		if groupDepth < 1 || groupDepth > MaxGroupDepth {
			return EmptyRef, errors.New("invalid group depth")
		}
		bitmapSize := BitmapSizeForDepth(groupDepth)
		if len(serialized) < NodeTypeBytes+1+bitmapSize {
			return EmptyRef, invalidSerializedLength
		}
		bitmap := serialized[2 : 2+bitmapSize]
		hashData := serialized[2+bitmapSize:]

		hashIdx := 0
		ref, err := s.deserializeSubtree(groupDepth, 0, depth, bitmap, hashData, &hashIdx, mustRecompute)
		if err != nil {
			return EmptyRef, err
		}
		if ref.Kind() == KindInternal && !mustRecompute {
			s.getInternal(ref.Index()).hash = hn
		}
		return ref, nil

	case nodeTypeStem:
		if len(serialized) < 64 {
			return EmptyRef, invalidSerializedLength
		}
		stemIdx := s.allocStem()
		sn := s.getStem(stemIdx)
		copy(sn.Stem[:], serialized[NodeTypeBytes:NodeTypeBytes+StemSize])
		copy(sn.bitmap[:], serialized[NodeTypeBytes+StemSize:NodeTypeBytes+StemSize+StemBitmapSize])

		var count uint16
		for i := range StemBitmapSize {
			count += uint16(bits.OnesCount8(sn.bitmap[i]))
		}
		sn.count = count
		dataStart := NodeTypeBytes + StemSize + StemBitmapSize
		dataEnd := dataStart + int(count)*HashSize
		if len(serialized) < dataEnd {
			return EmptyRef, invalidSerializedLength
		}
		// Zero-copy sub-slice of serialized data
		sn.valueData = serialized[dataStart:dataEnd]
		sn.shared = true
		sn.depth = uint8(depth)
		sn.hash = hn
		sn.mustRecompute = mustRecompute
		return MakeRef(KindStem, stemIdx), nil

	default:
		return EmptyRef, errors.New("invalid node type")
	}
}

func (s *NodeStore) deserializeSubtree(remainingDepth int, position int, nodeDepth int, bitmap []byte, hashData []byte, hashIdx *int, mustRecompute bool) (NodeRef, error) {
	if remainingDepth == 0 {
		if bitmap[position/8]>>(7-(position%8))&1 == 1 {
			if len(hashData) < (*hashIdx+1)*HashSize {
				return EmptyRef, invalidSerializedLength
			}
			hash := common.BytesToHash(hashData[*hashIdx*HashSize : (*hashIdx+1)*HashSize])
			*hashIdx++
			return s.newHashedRef(hash), nil
		}
		return EmptyRef, nil
	}

	leftPos := position * 2
	rightPos := position*2 + 1

	left, err := s.deserializeSubtree(remainingDepth-1, leftPos, nodeDepth+1, bitmap, hashData, hashIdx, true)
	if err != nil {
		return EmptyRef, err
	}
	right, err := s.deserializeSubtree(remainingDepth-1, rightPos, nodeDepth+1, bitmap, hashData, hashIdx, true)
	if err != nil {
		return EmptyRef, err
	}

	if left.IsEmpty() && right.IsEmpty() {
		return EmptyRef, nil
	}

	if nodeDepth > 248 {
		panic("node depth exceeds maximum binary trie depth")
	}
	idx := s.allocInternal()
	n := s.getInternal(idx)
	n.depth = uint8(nodeDepth)
	n.left = left
	n.right = right
	n.mustRecompute = mustRecompute
	return MakeRef(KindInternal, idx), nil
}

// --- CollectNodes (Commit) ---

// CollectNodes traverses the trie, flushing nodes at group boundaries.
func (s *NodeStore) CollectNodes(ref NodeRef, path []byte, flushfn NodeFlushFn, groupDepth int) error {
	if groupDepth < 1 || groupDepth > MaxGroupDepth {
		return errors.New("groupDepth must be between 1 and 8")
	}
	buf := make([]byte, len(path), len(path)+MaxGroupDepth+1)
	copy(buf, path)
	return s.collectNodesBuf(ref, buf, flushfn, groupDepth)
}

func (s *NodeStore) collectNodesBuf(ref NodeRef, buf []byte, flushfn NodeFlushFn, groupDepth int) error {
	switch ref.Kind() {
	case KindInternal:
		node := s.getInternal(ref.Index())
		if int(node.depth)%groupDepth == 0 {
			if err := s.collectChildGroupsBuf(ref, buf, flushfn, groupDepth, groupDepth-1); err != nil {
				return err
			}
			serialized := s.SerializeNode(ref, groupDepth)
			flushfn(buf, s.ComputeHash(ref), serialized)
			return nil
		}
		return s.collectChildGroupsBuf(ref, buf, flushfn, groupDepth, groupDepth-(int(node.depth)%groupDepth)-1)

	case KindStem:
		serialized := s.SerializeNode(ref, groupDepth)
		flushfn(buf, s.ComputeHash(ref), serialized)
		return nil

	case KindHashed, KindEmpty:
		return nil

	default:
		return fmt.Errorf("collectNodesBuf: unexpected kind %d", ref.Kind())
	}
}

func (s *NodeStore) collectChildGroupsBuf(ref NodeRef, buf []byte, flushfn NodeFlushFn, groupDepth int, remainingLevels int) error {
	if ref.Kind() != KindInternal {
		return nil
	}
	node := s.getInternal(ref.Index())
	saved := len(buf)
	childDepth := int(node.depth) + 1

	if remainingLevels == 0 {
		if !node.left.IsEmpty() {
			buf = append(buf, 0)
			if err := s.collectNodesBuf(node.left, buf, flushfn, groupDepth); err != nil {
				return err
			}
			buf = buf[:saved]
		}
		if !node.right.IsEmpty() {
			buf = append(buf, 1)
			if err := s.collectNodesBuf(node.right, buf, flushfn, groupDepth); err != nil {
				return err
			}
			buf = buf[:saved]
		}
		return nil
	}

	// Left child
	if !node.left.IsEmpty() {
		if node.left.Kind() == KindInternal {
			buf = append(buf, 0)
			if err := s.collectChildGroupsBuf(node.left, buf, flushfn, groupDepth, remainingLevels-1); err != nil {
				return err
			}
			buf = buf[:saved]
		} else {
			buf = append(buf, 0)
			buf = s.extendPathBuf(buf, node.left, remainingLevels, childDepth)
			if err := s.collectNodesBuf(node.left, buf, flushfn, groupDepth); err != nil {
				return err
			}
			buf = buf[:saved]
		}
	}

	// Right child
	if !node.right.IsEmpty() {
		if node.right.Kind() == KindInternal {
			buf = append(buf, 1)
			if err := s.collectChildGroupsBuf(node.right, buf, flushfn, groupDepth, remainingLevels-1); err != nil {
				return err
			}
			buf = buf[:saved]
		} else {
			buf = append(buf, 1)
			buf = s.extendPathBuf(buf, node.right, remainingLevels, childDepth)
			if err := s.collectNodesBuf(node.right, buf, flushfn, groupDepth); err != nil {
				return err
			}
			buf = buf[:saved]
		}
	}

	return nil
}

// extendPathBuf extends the path buffer to the group's leaf boundary.
func (s *NodeStore) extendPathBuf(buf []byte, ref NodeRef, remainingLevels int, absoluteDepth int) []byte {
	if remainingLevels <= 0 {
		return buf
	}
	if ref.Kind() == KindStem {
		sn := s.getStem(ref.Index())
		for d := 0; d < remainingLevels; d++ {
			bit := sn.Stem[(absoluteDepth+d)/8] >> (7 - ((absoluteDepth + d) % 8)) & 1
			buf = append(buf, bit)
		}
	} else {
		for d := 0; d < remainingLevels; d++ {
			buf = append(buf, 0)
		}
	}
	return buf
}

// ToDot generates a DOT representation for debugging.
func (s *NodeStore) ToDot(ref NodeRef, parent, path string) string {
	switch ref.Kind() {
	case KindInternal:
		node := s.getInternal(ref.Index())
		me := fmt.Sprintf("internal%s", path)
		ret := fmt.Sprintf("%s [label=\"I: %x\"]\n", me, s.ComputeHash(ref))
		if len(parent) > 0 {
			ret = fmt.Sprintf("%s %s -> %s\n", ret, parent, me)
		}
		if !node.left.IsEmpty() {
			ret += s.ToDot(node.left, me, fmt.Sprintf("%s%02x", path, 0))
		}
		if !node.right.IsEmpty() {
			ret += s.ToDot(node.right, me, fmt.Sprintf("%s%02x", path, 1))
		}
		return ret
	case KindStem:
		sn := s.getStem(ref.Index())
		me := fmt.Sprintf("stem%s", path)
		ret := fmt.Sprintf("%s [label=\"stem=%x c=%x\"]\n", me, sn.Stem, sn.Hash())
		ret = fmt.Sprintf("%s %s -> %s\n", ret, parent, me)
		idx := 0
		for i := range StemNodeWidth {
			if sn.bitmap[i/8]>>(7-i%8)&1 != 1 {
				continue
			}
			v := sn.valueData[idx*HashSize : (idx+1)*HashSize]
			idx++
			ret += fmt.Sprintf("%s%x [label=\"%x\"]\n", me, i, v)
			ret += fmt.Sprintf("%s -> %s%x\n", me, me, i)
		}
		return ret
	case KindHashed:
		hn := s.getHashed(ref.Index())
		me := fmt.Sprintf("hash%s", path)
		ret := fmt.Sprintf("%s [label=\"%x\"]\n", me, hn.hash)
		ret = fmt.Sprintf("%s %s -> %s\n", ret, parent, me)
		return ret
	default:
		return ""
	}
}
