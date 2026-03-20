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
	"slices"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

type (
	// NodeFlushFn is called for each node during CollectNodes.
	NodeFlushFn func(path []byte, ref NodeRef)

	// NodeResolverFn resolves a hashed node from the database.
	NodeResolverFn func(path []byte, hash common.Hash) ([]byte, error)
)

// parallelDepth returns the tree depth below which ComputeHash spawns goroutines.
func parallelDepth() int {
	return min(bits.Len(uint(runtime.NumCPU())), 8)
}

// ComputeHash computes (and caches) the hash rooted at ref.
func (s *NodeStore) ComputeHash(ref NodeRef) common.Hash {
	switch ref.Kind() {
	case KindEmpty:
		return common.Hash{}
	case KindStem:
		sn := s.getStem(ref.Index())
		return sn.Hash()
	case KindInternal:
		n := s.getInternal(ref.Index())
		if !n.mustRecompute {
			return n.hash
		}
		leftDirty := s.isDirty(n.left)
		rightDirty := s.isDirty(n.right)

		if n.depth < parallelDepth() && leftDirty && rightDirty {
			var input [64]byte
			var lh common.Hash
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				lh = s.ComputeHash(n.left)
			}()
			rh := s.ComputeHash(n.right)
			copy(input[32:], rh[:])
			wg.Wait()
			copy(input[:32], lh[:])
			n.hash = sha256.Sum256(input[:])
			n.mustRecompute = false
			return n.hash
		}

		h := newSha256()
		defer returnSha256(h)
		lh := s.ComputeHash(n.left)
		h.Write(lh[:])
		rh := s.ComputeHash(n.right)
		h.Write(rh[:])
		n.hash = common.BytesToHash(h.Sum(nil))
		n.mustRecompute = false
		return n.hash
	case KindHashed:
		return s.getHashed(ref.Index()).hash
	default:
		panic("invalid node kind")
	}
}

// isDirty reports whether a node needs rehashing.
func (s *NodeStore) isDirty(ref NodeRef) bool {
	switch ref.Kind() {
	case KindInternal:
		return s.getInternal(ref.Index()).mustRecompute
	case KindStem:
		return s.getStem(ref.Index()).mustRecompute
	default:
		return false
	}
}

// SerializeNode serializes the node referenced by ref into a byte slice.
// Uses master's exact flat format: InternalNode = [type:1][leftHash:32][rightHash:32].
func (s *NodeStore) SerializeNode(ref NodeRef) []byte {
	switch ref.Kind() {
	case KindInternal:
		n := s.getInternal(ref.Index())
		var serialized [NodeTypeBytes + HashSize + HashSize]byte
		serialized[0] = nodeTypeInternal
		lh := s.ComputeHash(n.left)
		copy(serialized[1:33], lh[:])
		rh := s.ComputeHash(n.right)
		copy(serialized[33:65], rh[:])
		return serialized[:]
	case KindStem:
		sn := s.getStem(ref.Index())
		var serialized [NodeTypeBytes + StemSize + BitmapSize + StemNodeWidth*HashSize]byte
		serialized[0] = nodeTypeStem
		copy(serialized[NodeTypeBytes:NodeTypeBytes+StemSize], sn.Stem)
		bitmap := serialized[NodeTypeBytes+StemSize : NodeTypeBytes+StemSize+BitmapSize]
		offset := NodeTypeBytes + StemSize + BitmapSize
		for i, v := range sn.Values {
			if v != nil {
				bitmap[i/8] |= 1 << (7 - (i % 8))
				copy(serialized[offset:offset+HashSize], v)
				offset += HashSize
			}
		}
		return serialized[:offset]
	default:
		panic("invalid node type for serialization")
	}
}

// DeserializeNode deserializes a binary trie node from a byte slice.
// The hash will be recomputed from the deserialized data.
func (s *NodeStore) DeserializeNode(serialized []byte, depth int) (NodeRef, error) {
	return s.deserializeNode(serialized, depth, common.Hash{}, true)
}

// DeserializeNodeWithHash deserializes a binary trie node from a byte slice,
// using the provided hash.
func (s *NodeStore) DeserializeNodeWithHash(serialized []byte, depth int, hn common.Hash) (NodeRef, error) {
	return s.deserializeNode(serialized, depth, hn, false)
}

func (s *NodeStore) deserializeNode(serialized []byte, depth int, hn common.Hash, mustRecompute bool) (NodeRef, error) {
	if len(serialized) == 0 {
		return EmptyRef, nil
	}

	switch serialized[0] {
	case nodeTypeInternal:
		if len(serialized) != 65 {
			return EmptyRef, invalidSerializedLength
		}
		leftHash := common.BytesToHash(serialized[1:33])
		rightHash := common.BytesToHash(serialized[33:65])
		var left, right NodeRef
		if leftHash == (common.Hash{}) {
			left = EmptyRef
		} else {
			left = s.allocHashed(HashedNode{hash: leftHash})
		}
		if rightHash == (common.Hash{}) {
			right = EmptyRef
		} else {
			right = s.allocHashed(HashedNode{hash: rightHash})
		}
		return s.allocInternal(InternalNode{
			depth:         depth,
			left:          left,
			right:         right,
			hash:          hn,
			mustRecompute: mustRecompute,
		}), nil
	case nodeTypeStem:
		if len(serialized) < 64 {
			return EmptyRef, invalidSerializedLength
		}
		var values [StemNodeWidth][]byte
		bitmap := serialized[NodeTypeBytes+StemSize : NodeTypeBytes+StemSize+BitmapSize]
		offset := NodeTypeBytes + StemSize + BitmapSize

		for i := range StemNodeWidth {
			if bitmap[i/8]>>(7-(i%8))&1 == 1 {
				if len(serialized) < offset+HashSize {
					return EmptyRef, invalidSerializedLength
				}
				values[i] = serialized[offset : offset+HashSize]
				offset += HashSize
			}
		}
		return s.allocStem(StemNode{
			Stem:          slices.Clone(serialized[NodeTypeBytes : NodeTypeBytes+StemSize]),
			Values:        values[:],
			depth:         depth,
			hash:          hn,
			mustRecompute: mustRecompute,
		}), nil
	default:
		return EmptyRef, errors.New("invalid node type")
	}
}

var invalidSerializedLength = errors.New("invalid serialized node length")

// CollectNodes collects all dirty (non-hashed) nodes rooted at ref
// in depth-first order and calls flushfn for each.
func (s *NodeStore) CollectNodes(ref NodeRef, path []byte, flushfn NodeFlushFn) error {
	switch ref.Kind() {
	case KindEmpty:
		return nil
	case KindHashed:
		// Already persisted
		return nil
	case KindStem:
		flushfn(path, ref)
		return nil
	case KindInternal:
		n := s.getInternal(ref.Index())
		if !n.left.IsEmpty() {
			var p [256]byte
			copy(p[:], path)
			childpath := p[:len(path)]
			childpath = append(childpath, 0)
			if err := s.CollectNodes(n.left, childpath, flushfn); err != nil {
				return err
			}
		}
		if !n.right.IsEmpty() {
			var p [256]byte
			copy(p[:], path)
			childpath := p[:len(path)]
			childpath = append(childpath, 1)
			if err := s.CollectNodes(n.right, childpath, flushfn); err != nil {
				return err
			}
		}
		flushfn(path, ref)
		return nil
	default:
		panic("invalid node kind")
	}
}

// ToDot converts the subtree rooted at ref to a DOT language representation.
func (s *NodeStore) ToDot(ref NodeRef) string {
	return s.toDot(ref, "", "")
}

func (s *NodeStore) toDot(ref NodeRef, parent, path string) string {
	switch ref.Kind() {
	case KindEmpty:
		return ""
	case KindHashed:
		hn := s.getHashed(ref.Index())
		me := fmt.Sprintf("hash%s", path)
		ret := fmt.Sprintf("%s [label=\"%x\"]\n", me, hn.hash)
		ret = fmt.Sprintf("%s %s -> %s\n", ret, parent, me)
		return ret
	case KindStem:
		sn := s.getStem(ref.Index())
		me := fmt.Sprintf("stem%s", path)
		ret := fmt.Sprintf("%s [label=\"stem=%x c=%x\"]\n", me, sn.Stem, sn.Hash())
		if len(parent) > 0 {
			ret = fmt.Sprintf("%s %s -> %s\n", ret, parent, me)
		}
		for i, v := range sn.Values {
			if v != nil {
				ret = fmt.Sprintf("%s%s%x [label=\"%x\"]\n", ret, me, i, v)
				ret = fmt.Sprintf("%s%s -> %s%x\n", ret, me, me, i)
			}
		}
		return ret
	case KindInternal:
		n := s.getInternal(ref.Index())
		me := fmt.Sprintf("internal%s", path)
		ret := fmt.Sprintf("%s [label=\"I: %x\"]\n", me, s.ComputeHash(ref))
		if len(parent) > 0 {
			ret = fmt.Sprintf("%s %s -> %s\n", ret, parent, me)
		}
		ret += s.toDot(n.left, me, fmt.Sprintf("%s%02x", path, 0))
		ret += s.toDot(n.right, me, fmt.Sprintf("%s%02x", path, 1))
		return ret
	default:
		return ""
	}
}
