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

// SerializeNode serializes a node referenced by ref into the flat format:
//   - InternalNode: [nodeTypeInternal(1)][leftHash(32)][rightHash(32)] = 65 bytes
//   - StemNode:     [nodeTypeStem(1)][stem(31)][bitmap(32)][valueData(variable)]
func (s *NodeStore) SerializeNode(ref NodeRef) []byte {
	switch ref.Kind() {
	case KindInternal:
		node := s.getInternal(ref.Index())
		var serialized [NodeTypeBytes + HashSize + HashSize]byte
		serialized[0] = nodeTypeInternal
		lh := s.ComputeHash(node.left)
		rh := s.ComputeHash(node.right)
		copy(serialized[NodeTypeBytes:NodeTypeBytes+HashSize], lh[:])
		copy(serialized[NodeTypeBytes+HashSize:], rh[:])
		return serialized[:]

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

var errInvalidSerializedLength = errors.New("invalid serialized node length")

// DeserializeNode deserializes a node from bytes, recomputing its hash.
// The serialized buffer must not be modified after this call.
func (s *NodeStore) DeserializeNode(serialized []byte, depth int) (NodeRef, error) {
	return s.deserializeNode(serialized, depth, common.Hash{}, true)
}

// DeserializeNodeWithHash deserializes a node, using the provided hash.
// The serialized buffer must not be modified after this call.
func (s *NodeStore) DeserializeNodeWithHash(serialized []byte, depth int, hn common.Hash) (NodeRef, error) {
	return s.deserializeNode(serialized, depth, hn, false)
}

func (s *NodeStore) deserializeNode(serialized []byte, depth int, hn common.Hash, mustRecompute bool) (NodeRef, error) {
	if len(serialized) == 0 {
		return EmptyRef, nil
	}

	switch serialized[0] {
	case nodeTypeInternal:
		if len(serialized) != NodeTypeBytes+2*HashSize {
			return EmptyRef, errInvalidSerializedLength
		}
		var leftHash, rightHash common.Hash
		copy(leftHash[:], serialized[NodeTypeBytes:NodeTypeBytes+HashSize])
		copy(rightHash[:], serialized[NodeTypeBytes+HashSize:])

		var leftRef, rightRef NodeRef
		if leftHash != (common.Hash{}) {
			leftRef = s.newHashedRef(leftHash)
		}
		if rightHash != (common.Hash{}) {
			rightRef = s.newHashedRef(rightHash)
		}

		ref := s.newInternalRef(depth)
		node := s.getInternal(ref.Index())
		node.left = leftRef
		node.right = rightRef
		if !mustRecompute {
			node.hash = hn
			node.mustRecompute = false
		}
		return ref, nil

	case nodeTypeStem:
		if len(serialized) < 64 {
			return EmptyRef, errInvalidSerializedLength
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
			return EmptyRef, errInvalidSerializedLength
		}
		// Zero-copy: valueData aliases the serialized buffer. The shared
		// flag triggers copy-on-write via ensureWritable() before mutation.
		// Callers must not modify serialized after this call.
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

// --- CollectNodes (Commit) ---

// CollectNodes traverses the trie, serializing and flushing each node via flushfn.
// Children are flushed before their parents (post-order traversal).
func (s *NodeStore) CollectNodes(ref NodeRef, path []byte, flushfn NodeFlushFn) error {
	switch ref.Kind() {
	case KindEmpty:
		return nil
	case KindInternal:
		node := s.getInternal(ref.Index())
		leftPath := make([]byte, len(path)+1)
		copy(leftPath, path)
		leftPath[len(path)] = 0
		if err := s.CollectNodes(node.left, leftPath, flushfn); err != nil {
			return err
		}
		rightPath := make([]byte, len(path)+1)
		copy(rightPath, path)
		rightPath[len(path)] = 1
		if err := s.CollectNodes(node.right, rightPath, flushfn); err != nil {
			return err
		}
		flushfn(path, s.ComputeHash(ref), s.SerializeNode(ref))
		return nil
	case KindStem:
		flushfn(path, s.ComputeHash(ref), s.SerializeNode(ref))
		return nil
	case KindHashed:
		return nil // Already committed
	default:
		return fmt.Errorf("CollectNodes: unexpected kind %d", ref.Kind())
	}
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
