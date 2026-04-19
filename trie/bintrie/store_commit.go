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

// SerializeNode serializes a node into the flat on-disk format.
func (s *nodeStore) serializeNode(ref nodeRef) []byte {
	switch ref.Kind() {
	case kindInternal:
		node := s.getInternal(ref.Index())
		var serialized [NodeTypeBytes + HashSize + HashSize]byte
		serialized[0] = nodeTypeInternal
		lh := s.computeHash(node.left)
		rh := s.computeHash(node.right)
		copy(serialized[NodeTypeBytes:NodeTypeBytes+HashSize], lh[:])
		copy(serialized[NodeTypeBytes+HashSize:], rh[:])
		return serialized[:]

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

func (s *nodeStore) decodeNode(serialized []byte, depth int, hn common.Hash, mustRecompute, dirty bool) (nodeRef, error) {
	if len(serialized) == 0 {
		return emptyRef, nil
	}

	switch serialized[0] {
	case nodeTypeInternal:
		if len(serialized) != NodeTypeBytes+2*HashSize {
			return emptyRef, errInvalidSerializedLength
		}
		var leftHash, rightHash common.Hash
		copy(leftHash[:], serialized[NodeTypeBytes:NodeTypeBytes+HashSize])
		copy(rightHash[:], serialized[NodeTypeBytes+HashSize:])

		var leftRef, rightRef nodeRef
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
		node.dirty = dirty
		return ref, nil

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
func (s *nodeStore) collectNodes(ref nodeRef, path []byte, flushfn nodeFlushFn) error {
	switch ref.Kind() {
	case kindEmpty:
		return nil
	case kindInternal:
		node := s.getInternal(ref.Index())
		if !node.dirty {
			return nil
		}
		// Reuse path buffer across children: flushfn consumers
		// (NodeSet.AddNode, tracer.Get) clone via string(path), so in-place
		// mutation is safe.
		path = append(path, 0)
		if err := s.collectNodes(node.left, path, flushfn); err != nil {
			return err
		}
		path[len(path)-1] = 1
		if err := s.collectNodes(node.right, path, flushfn); err != nil {
			return err
		}
		path = path[:len(path)-1]
		flushfn(path, s.computeHash(ref), s.serializeNode(ref))
		node.dirty = false
		return nil
	case kindStem:
		sn := s.getStem(ref.Index())
		if !sn.dirty {
			return nil
		}
		flushfn(path, s.computeHash(ref), s.serializeNode(ref))
		sn.dirty = false
		return nil
	case kindHashed:
		return nil // Already committed
	default:
		return fmt.Errorf("CollectNodes: unexpected kind %d", ref.Kind())
	}
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
			ret += s.toDot(node.left, me, fmt.Sprintf("%s%02x", path, 0))
		}
		if !node.right.IsEmpty() {
			ret += s.toDot(node.right, me, fmt.Sprintf("%s%02x", path, 1))
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
