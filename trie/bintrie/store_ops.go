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
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

// nodeResolverFn resolves a hashed node from the database.
type nodeResolverFn func([]byte, common.Hash) ([]byte, error)

// GetValue returns the value at (stem, suffix) or nil if absent. Thin
// wrapper over GetValuesAtStem — the underlying StemNode returns its
// 256-slot array as a slice header (no allocation), so the per-call cost
// is the tree walk plus one index.
func (s *nodeStore) GetValue(stem []byte, suffix byte, resolver nodeResolverFn) ([]byte, error) {
	values, err := s.GetValuesAtStem(stem, resolver)
	if err != nil || values == nil {
		return nil, err
	}
	return values[suffix], nil
}

// GetValuesAtStem returns the 256 value slots at stem, or nil if the stem
// is not in the trie. The returned slice is a view over the in-place
// StemNode values array (no allocation) and must be treated read-only.
func (s *nodeStore) GetValuesAtStem(stem []byte, resolver nodeResolverFn) ([][]byte, error) {
	cur := s.root
	var parentIdx uint32
	var parentIsLeft bool

	for {
		switch cur.Kind() {
		case kindInternal:
			node := s.getInternal(cur.Index())
			if node.depth >= 31*8 {
				return nil, errors.New("node too deep")
			}
			bit := stem[node.depth/8] >> (7 - (node.depth % 8)) & 1
			parentIdx = cur.Index()
			if bit == 0 {
				parentIsLeft = true
				cur = node.left
			} else {
				parentIsLeft = false
				cur = node.right
			}

		case kindStem:
			sn := s.getStem(cur.Index())
			if sn.Stem != [StemSize]byte(stem[:StemSize]) {
				return nil, nil
			}
			return sn.allValues(), nil

		case kindHashed:
			// HashedNode at root is impossible: NewBinaryTrie resolves the
			// root eagerly before any query. Any HashedNode we encounter here
			// is necessarily a child of a previously-visited internal node.
			if resolver == nil {
				return nil, errors.New("getValuesAtStem: cannot resolve hashed node without resolver")
			}
			hn := s.getHashed(cur.Index())
			parentNode := s.getInternal(parentIdx)
			path, err := keyToPath(int(parentNode.depth), stem)
			if err != nil {
				return nil, fmt.Errorf("getValuesAtStem path error: %w", err)
			}
			data, err := resolver(path, hn.Hash())
			if err != nil {
				return nil, fmt.Errorf("getValuesAtStem resolve error: %w", err)
			}
			resolved, err := s.deserializeNodeWithHash(data, int(parentNode.depth)+1, hn.Hash())
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

		case kindEmpty:
			var values [StemNodeWidth][]byte
			return values[:], nil

		default:
			return nil, fmt.Errorf("getValuesAtStem: unexpected node kind %d", cur.Kind())
		}
	}
}

// InsertSingle writes a single value slot at (stem, suffix). Thin wrapper
// over InsertValuesAtStem — builds a stack-allocated 256-slot array with
// only the target slot set and delegates. Matches the original design
// gballet referenced (comment 3101751325): one primary insert path; the
// single-slot variant dispatches through it so the split / resolve logic
// lives in one place.
func (s *nodeStore) InsertSingle(stem []byte, suffix byte, value []byte, resolver nodeResolverFn) error {
	if len(value) != HashSize {
		return errors.New("invalid insertion: value length")
	}
	var values [StemNodeWidth][]byte
	values[suffix] = value
	return s.InsertValuesAtStem(stem, values[:], resolver)
}

// InsertValuesAtStem writes the supplied value slots at stem. values may be
// sparse (nil entries are ignored). The recursive implementation dispatches
// through the same body, so a single code path handles internal descent,
// HashedNode resolution, stem merge, and stem split.
func (s *nodeStore) InsertValuesAtStem(stem []byte, values [][]byte, resolver nodeResolverFn) error {
	var err error
	s.root, err = s.insertValuesAtStem(s.root, stem, values, resolver, 0)
	return err
}

func (s *nodeStore) insertValuesAtStem(ref nodeRef, stem []byte, values [][]byte, resolver nodeResolverFn, depth int) (nodeRef, error) {
	switch ref.Kind() {
	case kindInternal:
		node := s.getInternal(ref.Index())
		bit := stem[node.depth/8] >> (7 - (node.depth % 8)) & 1
		if bit == 0 {
			if node.left.Kind() == kindHashed {
				if resolver == nil {
					return ref, errors.New("insertValuesAtStem: cannot resolve hashed node without resolver")
				}
				hn := s.getHashed(node.left.Index())
				path, err := keyToPath(int(node.depth), stem)
				if err != nil {
					return ref, fmt.Errorf("InsertValuesAtStem path error: %w", err)
				}
				data, err := resolver(path, hn.Hash())
				if err != nil {
					return ref, fmt.Errorf("InsertValuesAtStem resolve error: %w", err)
				}
				resolved, err := s.deserializeNodeWithHash(data, int(node.depth)+1, hn.Hash())
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
			if node.right.Kind() == kindHashed {
				if resolver == nil {
					return ref, errors.New("insertValuesAtStem: cannot resolve hashed node without resolver")
				}
				hn := s.getHashed(node.right.Index())
				path, err := keyToPath(int(node.depth), stem)
				if err != nil {
					return ref, fmt.Errorf("InsertValuesAtStem path error: %w", err)
				}
				data, err := resolver(path, hn.Hash())
				if err != nil {
					return ref, fmt.Errorf("InsertValuesAtStem resolve error: %w", err)
				}
				resolved, err := s.deserializeNodeWithHash(data, int(node.depth)+1, hn.Hash())
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
		node.dirty = true
		return ref, nil

	case kindStem:
		sn := s.getStem(ref.Index())
		if sn.Stem == [StemSize]byte(stem[:StemSize]) {
			// Same stem — merge values (setValue marks dirty+mustRecompute)
			for i, v := range values {
				if v != nil {
					sn.setValue(byte(i), v)
				}
			}
			return ref, nil
		}
		// Different stem — split
		return s.splitStemValuesInsert(ref, stem, values, resolver, depth)

	case kindHashed:
		hn := s.getHashed(ref.Index())
		path, err := keyToPath(depth, stem)
		if err != nil {
			return ref, fmt.Errorf("InsertValuesAtStem path error: %w", err)
		}
		if resolver == nil {
			return ref, errors.New("InsertValuesAtStem: resolver is nil")
		}
		data, err := resolver(path, hn.Hash())
		if err != nil {
			return ref, fmt.Errorf("InsertValuesAtStem resolve error: %w", err)
		}
		resolved, err := s.deserializeNodeWithHash(data, depth, hn.Hash())
		if err != nil {
			return ref, fmt.Errorf("InsertValuesAtStem deserialization error: %w", err)
		}
		s.freeHashedNode(ref.Index())
		return s.insertValuesAtStem(resolved, stem, values, resolver, depth)

	case kindEmpty:
		// Create new StemNode. Flag flips before the value loop so an
		// all-nil values input still marks the newly-created stem dirty.
		stemIdx := s.allocStem()
		sn := s.getStem(stemIdx)
		copy(sn.Stem[:], stem[:StemSize])
		sn.depth = uint8(depth)
		sn.mustRecompute = true
		sn.dirty = true
		for i, v := range values {
			if v != nil {
				sn.setValue(byte(i), v)
			}
		}
		return makeRef(kindStem, stemIdx), nil

	default:
		return ref, fmt.Errorf("insertValuesAtStem: unexpected kind %d", ref.Kind())
	}
}

// splitStemValuesInsert splits a StemNode when the new stem diverges.
func (s *nodeStore) splitStemValuesInsert(existingRef nodeRef, newStem []byte, values [][]byte, resolver nodeResolverFn, depth int) (nodeRef, error) {
	existing := s.getStem(existingRef.Index())

	if int(existing.depth) >= StemSize*8 {
		panic("splitStemValuesInsert: identical stems")
	}

	bitStem := existing.Stem[existing.depth/8] >> (7 - (existing.depth % 8)) & 1
	nRef := s.newInternalRef(int(existing.depth))
	nNode := s.getInternal(nRef.Index())
	existing.depth++

	bitKey := newStem[nNode.depth/8] >> (7 - (nNode.depth % 8)) & 1
	if bitKey == bitStem {
		// Same direction — need deeper split
		var child nodeRef
		if bitStem == 0 {
			nNode.left = existingRef
			child = nNode.left
		} else {
			nNode.right = existingRef
			child = nNode.right
		}
		newChild, err := s.insertValuesAtStem(child, newStem, values, resolver, depth+1)
		if err != nil {
			// Roll back the depth increment so a retry sees the same
			// existing state and extracts bitStem at the correct offset.
			// nRef itself leaks (no internal free-list), but the slot is
			// unreachable from the tree and harmless.
			existing.depth--
			return nRef, err
		}
		if bitStem == 0 {
			nNode.left = newChild
			nNode.right = emptyRef
		} else {
			nNode.right = newChild
			nNode.left = emptyRef
		}
	} else {
		// Divergence — create new StemNode for the new values
		newStemIdx := s.allocStem()
		newSn := s.getStem(newStemIdx)
		copy(newSn.Stem[:], newStem[:StemSize])
		newSn.depth = nNode.depth + 1
		newSn.mustRecompute = true
		newSn.dirty = true
		for i, v := range values {
			if v != nil {
				newSn.setValue(byte(i), v)
			}
		}
		newStemRef := makeRef(kindStem, newStemIdx)

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

func (s *nodeStore) Insert(key []byte, value []byte, resolver nodeResolverFn) error {
	return s.InsertSingle(key[:StemSize], key[StemSize], value, resolver)
}

func (s *nodeStore) Get(key []byte, resolver nodeResolverFn) ([]byte, error) {
	return s.GetValue(key[:StemSize], key[StemSize], resolver)
}

func (s *nodeStore) getHeight(ref nodeRef) int {
	switch ref.Kind() {
	case kindInternal:
		node := s.getInternal(ref.Index())
		lh := s.getHeight(node.left)
		rh := s.getHeight(node.right)
		if lh > rh {
			return 1 + lh
		}
		return 1 + rh
	case kindStem:
		return 1
	case kindEmpty:
		return 0
	default:
		return 0
	}
}
