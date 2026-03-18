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

	"github.com/ethereum/go-ethereum/common"
)

// NodeResolverFn resolves a hashed node from the database.
type NodeResolverFn func([]byte, common.Hash) ([]byte, error)

// GetSingle retrieves a single value at stem[suffix] from the trie root.
func (s *NodeStore) GetSingle(stem []byte, suffix byte, resolver NodeResolverFn) ([]byte, error) {
	return s.getSingle(s.root, stem, suffix, resolver)
}

// getSingle retrieves a single value using iterative traversal.
func (s *NodeStore) getSingle(ref NodeRef, stem []byte, suffix byte, resolver NodeResolverFn) ([]byte, error) {
	cur := ref
	// Track parent for HashedNode resolution (update parent's child ref).
	var parentIdx uint32
	var parentIsLeft bool
	hasParent := false

	for {
		switch cur.Kind() {
		case KindInternal:
			node := s.getInternal(cur.Index())
			if node.depth >= 31*8 {
				return nil, errors.New("node too deep")
			}
			bit := stem[node.depth/8] >> (7 - (node.depth % 8)) & 1
			parentIdx = cur.Index()
			hasParent = true
			if bit == 0 {
				parentIsLeft = true
				cur = node.left
			} else {
				parentIsLeft = false
				cur = node.right
			}

		case KindStem:
			sn := s.getStem(cur.Index())
			if sn.Stem != [StemSize]byte(stem[:StemSize]) {
				return nil, nil
			}
			return sn.getValue(suffix), nil

		case KindHashed:
			if !hasParent {
				return nil, errors.New("getSingle: hashed node at root")
			}
			hn := s.getHashed(cur.Index())
			parentNode := s.getInternal(parentIdx)
			path := makeKeyPath(int(parentNode.depth), stem)
			data, err := resolver(path, hn.hash)
			if err != nil {
				return nil, fmt.Errorf("getSingle resolve error: %w", err)
			}
			resolved, err := s.DeserializeNodeWithHash(data, int(parentNode.depth)+1, hn.hash)
			if err != nil {
				return nil, fmt.Errorf("getSingle deserialization error: %w", err)
			}
			// Update parent's child ref
			s.freeHashedNode(cur.Index())
			if parentIsLeft {
				parentNode.left = resolved
			} else {
				parentNode.right = resolved
			}
			cur = resolved

		case KindEmpty:
			return nil, nil

		default:
			return nil, fmt.Errorf("getSingle: unexpected node kind %d", cur.Kind())
		}
	}
}

// GetValuesAtStem retrieves all 256 values at a stem.
func (s *NodeStore) GetValuesAtStem(stem []byte, resolver NodeResolverFn) ([][]byte, error) {
	return s.getValuesAtStem(s.root, stem, resolver)
}

// getValuesAtStem uses iterative traversal to find the StemNode.
func (s *NodeStore) getValuesAtStem(ref NodeRef, stem []byte, resolver NodeResolverFn) ([][]byte, error) {
	cur := ref
	var parentIdx uint32
	var parentIsLeft bool
	hasParent := false

	for {
		switch cur.Kind() {
		case KindInternal:
			node := s.getInternal(cur.Index())
			if node.depth >= 31*8 {
				return nil, errors.New("node too deep")
			}
			bit := stem[node.depth/8] >> (7 - (node.depth % 8)) & 1
			parentIdx = cur.Index()
			hasParent = true
			if bit == 0 {
				parentIsLeft = true
				cur = node.left
			} else {
				parentIsLeft = false
				cur = node.right
			}

		case KindStem:
			sn := s.getStem(cur.Index())
			if sn.Stem != [StemSize]byte(stem[:StemSize]) {
				return nil, nil
			}
			return sn.allValues(), nil

		case KindHashed:
			if !hasParent {
				return nil, errors.New("getValuesAtStem: hashed node at root")
			}
			hn := s.getHashed(cur.Index())
			parentNode := s.getInternal(parentIdx)
			path := makeKeyPath(int(parentNode.depth), stem)
			data, err := resolver(path, hn.hash)
			if err != nil {
				return nil, fmt.Errorf("getValuesAtStem resolve error: %w", err)
			}
			resolved, err := s.DeserializeNodeWithHash(data, int(parentNode.depth)+1, hn.hash)
			if err != nil {
				return nil, fmt.Errorf("getValuesAtStem deserialization error: %w", err)
			}
			s.freeHashedNode(cur.Index())
			if parentIsLeft {
				parentNode.left = resolved
			} else {
				parentNode.right = resolved
			}
			cur = resolved

		case KindEmpty:
			var values [StemNodeWidth][]byte
			return values[:], nil

		default:
			return nil, fmt.Errorf("getValuesAtStem: unexpected node kind %d", cur.Kind())
		}
	}
}

// InsertSingle inserts a single value at stem[suffix] into the trie.
func (s *NodeStore) InsertSingle(stem []byte, suffix byte, value []byte, resolver NodeResolverFn) error {
	if len(value) != HashSize {
		return errors.New("invalid insertion: value length")
	}

	// Handle root-is-empty case
	if s.root.IsEmpty() {
		ref := s.newStemRef(stem, 0)
		sn := s.getStem(ref.Index())
		sn.setValue(suffix, value)
		s.root = ref
		return nil
	}

	// Handle root-is-stem case
	if s.root.Kind() == KindStem {
		sn := s.getStem(s.root.Index())
		if sn.Stem == [StemSize]byte(stem[:StemSize]) {
			sn.ensureWritable()
			sn.setValue(suffix, value)
			sn.mustRecompute = true
			return nil
		}
		// Different stem — promote root to internal node via split
		newRoot := s.splitStemInsert(s.root, stem, suffix, value, int(sn.depth))
		s.root = newRoot
		return nil
	}

	// Root is an InternalNode — iterative descent
	return s.insertSingleInternal(stem, suffix, value, resolver)
}

// insertSingleInternal handles insertion when root is an InternalNode.
func (s *NodeStore) insertSingleInternal(stem []byte, suffix byte, value []byte, resolver NodeResolverFn) error {
	type pathEntry struct {
		internalIdx uint32
		isLeft      bool
	}
	var pathStack [256]pathEntry // stack-allocated, max depth 248
	pathLen := 0

	cur := s.root

	for {
		switch cur.Kind() {
		case KindInternal:
			node := s.getInternal(cur.Index())
			node.mustRecompute = true
			bit := stem[node.depth/8] >> (7 - (node.depth % 8)) & 1
			pathStack[pathLen] = pathEntry{internalIdx: cur.Index(), isLeft: bit == 0}
			pathLen++
			if bit == 0 {
				cur = node.left
			} else {
				cur = node.right
			}

		case KindStem:
			sn := s.getStem(cur.Index())
			if sn.Stem == [StemSize]byte(stem[:StemSize]) {
				sn.ensureWritable()
				sn.setValue(suffix, value)
				sn.mustRecompute = true
				return nil
			}
			// Different stem — split
			parentDepth := int(s.getInternal(pathStack[pathLen-1].internalIdx).depth) + 1
			newRef := s.splitStemInsert(cur, stem, suffix, value, parentDepth)
			p := pathStack[pathLen-1]
			parent := s.getInternal(p.internalIdx)
			if p.isLeft {
				parent.left = newRef
			} else {
				parent.right = newRef
			}
			return nil

		case KindHashed:
			if pathLen == 0 {
				return errors.New("insertSingle: hashed node at root")
			}
			p := pathStack[pathLen-1]
			parentNode := s.getInternal(p.internalIdx)
			hn := s.getHashed(cur.Index())
			path := makeKeyPath(int(parentNode.depth), stem)
			data, err := resolver(path, hn.hash)
			if err != nil {
				return fmt.Errorf("insertSingle resolve error: %w", err)
			}
			resolved, err := s.DeserializeNodeWithHash(data, int(parentNode.depth)+1, hn.hash)
			if err != nil {
				return fmt.Errorf("insertSingle deserialization error: %w", err)
			}
			s.freeHashedNode(cur.Index())
			if p.isLeft {
				parentNode.left = resolved
			} else {
				parentNode.right = resolved
			}
			cur = resolved

		case KindEmpty:
			parentDepth := int(s.getInternal(pathStack[pathLen-1].internalIdx).depth) + 1
			ref := s.newStemRef(stem, parentDepth)
			sn := s.getStem(ref.Index())
			sn.setValue(suffix, value)
			p := pathStack[pathLen-1]
			parent := s.getInternal(p.internalIdx)
			if p.isLeft {
				parent.left = ref
			} else {
				parent.right = ref
			}
			return nil

		default:
			return fmt.Errorf("insertSingle: unexpected node kind %d", cur.Kind())
		}
	}
}

// splitStemInsert handles the case where we need to split a StemNode
// into a chain of InternalNodes because the new key has a different stem.
func (s *NodeStore) splitStemInsert(existingRef NodeRef, newStem []byte, suffix byte, value []byte, depth int) NodeRef {
	existing := s.getStem(existingRef.Index())
	existingDepth := depth

	var firstRef NodeRef
	var lastInternalIdx uint32
	var lastIsLeft bool
	first := true

	for {
		bitExisting := existing.Stem[existingDepth/8] >> (7 - (existingDepth % 8)) & 1
		bitNew := newStem[existingDepth/8] >> (7 - (existingDepth % 8)) & 1

		newInternalIdx := s.allocInternal()
		newInternal := s.getInternal(newInternalIdx)
		newInternal.depth = uint8(existingDepth)
		newInternal.mustRecompute = true
		newRef := MakeRef(KindInternal, newInternalIdx)

		if first {
			firstRef = newRef
			first = false
		} else {
			parent := s.getInternal(lastInternalIdx)
			if lastIsLeft {
				parent.left = newRef
			} else {
				parent.right = newRef
			}
		}

		if bitExisting != bitNew {
			// Divergence point
			existing.depth = uint8(existingDepth + 1)

			newStemIdx := s.allocStem()
			newSn := s.getStem(newStemIdx)
			copy(newSn.Stem[:], newStem[:StemSize])
			newSn.depth = uint8(existingDepth + 1)
			newSn.mustRecompute = true
			newSn.setValue(suffix, value)
			newStemRef := MakeRef(KindStem, newStemIdx)

			if bitExisting == 0 {
				newInternal.left = existingRef
				newInternal.right = newStemRef
			} else {
				newInternal.left = newStemRef
				newInternal.right = existingRef
			}
			return firstRef
		}

		// Same bit — continue splitting
		lastInternalIdx = newInternalIdx
		lastIsLeft = (bitExisting == 0)
		existingDepth++
	}
}

// InsertValuesAtStem inserts a full group of values at the given stem.
func (s *NodeStore) InsertValuesAtStem(stem []byte, values [][]byte, resolver NodeResolverFn) error {
	newRoot, err := s.insertValuesAtStem(s.root, stem, values, resolver, 0)
	if err != nil {
		return err
	}
	s.root = newRoot
	return nil
}

// insertValuesAtStem recursively inserts values at a stem.
func (s *NodeStore) insertValuesAtStem(ref NodeRef, stem []byte, values [][]byte, resolver NodeResolverFn, depth int) (NodeRef, error) {
	switch ref.Kind() {
	case KindInternal:
		node := s.getInternal(ref.Index())
		bit := stem[node.depth/8] >> (7 - (node.depth % 8)) & 1
		if bit == 0 {
			if node.left.IsEmpty() {
				// left is empty
			}
			if node.left.Kind() == KindHashed {
				hn := s.getHashed(node.left.Index())
				path := makeKeyPath(int(node.depth), stem)
				data, err := resolver(path, hn.hash)
				if err != nil {
					return ref, fmt.Errorf("InsertValuesAtStem resolve error: %w", err)
				}
				resolved, err := s.DeserializeNodeWithHash(data, int(node.depth)+1, hn.hash)
				if err != nil {
					return ref, fmt.Errorf("InsertValuesAtStem deserialization error: %w", err)
				}
				s.freeHashedNode(node.left.Index())
				node.left = resolved
			}
			newChild, err := s.insertValuesAtStem(node.left, stem, values, resolver, depth+1)
			if err != nil {
				return ref, err
			}
			node.left = newChild
		} else {
			if node.right.Kind() == KindHashed {
				hn := s.getHashed(node.right.Index())
				path := makeKeyPath(int(node.depth), stem)
				data, err := resolver(path, hn.hash)
				if err != nil {
					return ref, fmt.Errorf("InsertValuesAtStem resolve error: %w", err)
				}
				resolved, err := s.DeserializeNodeWithHash(data, int(node.depth)+1, hn.hash)
				if err != nil {
					return ref, fmt.Errorf("InsertValuesAtStem deserialization error: %w", err)
				}
				s.freeHashedNode(node.right.Index())
				node.right = resolved
			}
			newChild, err := s.insertValuesAtStem(node.right, stem, values, resolver, depth+1)
			if err != nil {
				return ref, err
			}
			node.right = newChild
		}
		node.mustRecompute = true
		return ref, nil

	case KindStem:
		sn := s.getStem(ref.Index())
		if sn.Stem == [StemSize]byte(stem[:StemSize]) {
			// Same stem — merge values
			sn.ensureWritable()
			for i, v := range values {
				if v != nil {
					sn.setValue(byte(i), v)
					sn.mustRecompute = true
				}
			}
			return ref, nil
		}
		// Different stem — split
		return s.splitStemValuesInsert(ref, stem, values, resolver, depth)

	case KindHashed:
		hn := s.getHashed(ref.Index())
		path, err := keyToPath(depth, stem)
		if err != nil {
			return ref, fmt.Errorf("InsertValuesAtStem path error: %w", err)
		}
		if resolver == nil {
			return ref, errors.New("InsertValuesAtStem: resolver is nil")
		}
		data, err := resolver(path, hn.hash)
		if err != nil {
			return ref, fmt.Errorf("InsertValuesAtStem resolve error: %w", err)
		}
		resolved, err := s.DeserializeNodeWithHash(data, depth, hn.hash)
		if err != nil {
			return ref, fmt.Errorf("InsertValuesAtStem deserialization error: %w", err)
		}
		s.freeHashedNode(ref.Index())
		return s.insertValuesAtStem(resolved, stem, values, resolver, depth)

	case KindEmpty:
		// Create new StemNode
		stemIdx := s.allocStem()
		sn := s.getStem(stemIdx)
		copy(sn.Stem[:], stem[:StemSize])
		sn.depth = uint8(depth)
		sn.mustRecompute = true
		for i, v := range values {
			if v != nil {
				sn.count++
				sn.bitmap[i/8] |= 1 << (7 - (i % 8))
				sn.valueData = append(sn.valueData, v[:HashSize]...)
			}
		}
		return MakeRef(KindStem, stemIdx), nil

	default:
		return ref, fmt.Errorf("insertValuesAtStem: unexpected kind %d", ref.Kind())
	}
}

// splitStemValuesInsert handles splitting a StemNode when inserting values with a different stem.
func (s *NodeStore) splitStemValuesInsert(existingRef NodeRef, newStem []byte, values [][]byte, resolver NodeResolverFn, depth int) (NodeRef, error) {
	existing := s.getStem(existingRef.Index())

	bitStem := existing.Stem[existing.depth/8] >> (7 - (existing.depth % 8)) & 1
	nRef := s.newInternalRef(int(existing.depth))
	nNode := s.getInternal(nRef.Index())
	existing.depth++

	bitKey := newStem[nNode.depth/8] >> (7 - (nNode.depth % 8)) & 1
	if bitKey == bitStem {
		// Same direction — need deeper split
		var child NodeRef
		if bitStem == 0 {
			nNode.left = existingRef
			child = nNode.left
		} else {
			nNode.right = existingRef
			child = nNode.right
		}
		newChild, err := s.insertValuesAtStem(child, newStem, values, resolver, depth+1)
		if err != nil {
			return nRef, err
		}
		if bitStem == 0 {
			nNode.left = newChild
			nNode.right = EmptyRef
		} else {
			nNode.right = newChild
			nNode.left = EmptyRef
		}
	} else {
		// Divergence — create new StemNode for the new values
		newStemIdx := s.allocStem()
		newSn := s.getStem(newStemIdx)
		copy(newSn.Stem[:], newStem[:StemSize])
		newSn.depth = nNode.depth + 1
		newSn.mustRecompute = true
		for i, v := range values {
			if v != nil {
				newSn.setValue(byte(i), v)
			}
		}
		newStemRef := MakeRef(KindStem, newStemIdx)

		if bitStem == 0 {
			nNode.left = existingRef
			nNode.right = newStemRef
		} else {
			nNode.left = newStemRef
			nNode.right = existingRef
		}
	}
	return nRef, nil
}

// Insert inserts a key-value pair using the full 32-byte key.
func (s *NodeStore) Insert(key []byte, value []byte, resolver NodeResolverFn) error {
	return s.InsertSingle(key[:StemSize], key[StemSize], value, resolver)
}

// Get retrieves the value for the given 32-byte key.
func (s *NodeStore) Get(key []byte, resolver NodeResolverFn) ([]byte, error) {
	return s.GetSingle(key[:StemSize], key[StemSize], resolver)
}

// GetHeight returns the height of the trie rooted at ref.
func (s *NodeStore) GetHeight(ref NodeRef) int {
	switch ref.Kind() {
	case KindInternal:
		node := s.getInternal(ref.Index())
		lh := s.GetHeight(node.left)
		rh := s.GetHeight(node.right)
		if lh > rh {
			return 1 + lh
		}
		return 1 + rh
	case KindStem:
		return 1
	case KindEmpty:
		return 0
	default:
		return 0
	}
}
