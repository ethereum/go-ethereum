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
	"bytes"
	"errors"
	"fmt"
	"slices"
)

// Get retrieves the value for the given key starting from ref.
func (s *NodeStore) Get(ref NodeRef, key []byte, resolver NodeResolverFn) ([]byte, error) {
	values, err := s.GetValuesAtStem(ref, key[:31], resolver)
	if err != nil {
		return nil, fmt.Errorf("get error: %w", err)
	}
	if values == nil {
		return nil, nil
	}
	return values[key[31]], nil
}

// Insert inserts a key-value pair starting from ref and returns the new root ref.
func (s *NodeStore) Insert(ref NodeRef, key []byte, value []byte, resolver NodeResolverFn, depth int) (NodeRef, error) {
	var values [256][]byte
	values[key[31]] = value
	return s.InsertValuesAtStem(ref, key[:31], values[:], resolver, depth)
}

// GetValuesAtStem retrieves the group of values located at the given stem key.
func (s *NodeStore) GetValuesAtStem(ref NodeRef, stem []byte, resolver NodeResolverFn) ([][]byte, error) {
	switch ref.Kind() {
	case KindEmpty:
		var values [256][]byte
		return values[:], nil
	case KindStem:
		sn := s.getStem(ref.Index())
		if !bytes.Equal(sn.Stem, stem) {
			return nil, nil
		}
		return sn.Values[:], nil
	case KindInternal:
		n := s.getInternal(ref.Index())
		if n.depth > 31*8 {
			return nil, errors.New("node too deep")
		}
		bit := stem[n.depth/8] >> (7 - (n.depth % 8)) & 1
		child := n.left
		if bit != 0 {
			child = n.right
		}
		if child.Kind() == KindHashed {
			resolved, err := s.resolveHashed(child, n.depth, stem, resolver)
			if err != nil {
				return nil, fmt.Errorf("GetValuesAtStem resolve error: %w", err)
			}
			if bit == 0 {
				n.left = resolved
			} else {
				n.right = resolved
			}
			child = resolved
		}
		return s.GetValuesAtStem(child, stem, resolver)
	case KindHashed:
		return nil, errors.New("attempted to get values from an unresolved node")
	default:
		panic("invalid node kind")
	}
}

// InsertValuesAtStem inserts a full value group at the given stem.
func (s *NodeStore) InsertValuesAtStem(ref NodeRef, stem []byte, values [][]byte, resolver NodeResolverFn, depth int) (NodeRef, error) {
	if depth > maxDepth {
		panic("bintrie: depth exceeds maximum (248)")
	}
	switch ref.Kind() {
	case KindEmpty:
		return s.allocStem(StemNode{
			Stem:          slices.Clone(stem[:StemSize]),
			Values:        values,
			depth:         depth,
			mustRecompute: true,
		}), nil
	case KindStem:
		sn := s.getStem(ref.Index())
		if bytes.Equal(sn.Stem, stem[:StemSize]) {
			// same stem, merge values
			for i, v := range values {
				if v != nil {
					sn.Values[i] = v
					sn.mustRecompute = true
				}
			}
			return ref, nil
		}
		// Different stem: split into internal node
		return s.splitStem(ref, sn, stem, values, resolver, depth)
	case KindInternal:
		n := s.getInternal(ref.Index())
		bit := stem[n.depth/8] >> (7 - (n.depth % 8)) & 1
		var err error
		if bit == 0 {
			if n.left.Kind() == KindHashed {
				resolved, rerr := s.resolveHashed(n.left, n.depth, stem, resolver)
				if rerr != nil {
					return ref, fmt.Errorf("InsertValuesAtStem resolve error: %w", rerr)
				}
				n.left = resolved
			}
			n.left, err = s.InsertValuesAtStem(n.left, stem, values, resolver, depth+1)
		} else {
			if n.right.Kind() == KindHashed {
				resolved, rerr := s.resolveHashed(n.right, n.depth, stem, resolver)
				if rerr != nil {
					return ref, fmt.Errorf("InsertValuesAtStem resolve error: %w", rerr)
				}
				n.right = resolved
			}
			n.right, err = s.InsertValuesAtStem(n.right, stem, values, resolver, depth+1)
		}
		n.mustRecompute = true
		return ref, err
	case KindHashed:
		hn := s.getHashed(ref.Index())
		path, err := keyToPath(depth, stem)
		if err != nil {
			return ref, fmt.Errorf("InsertValuesAtStem path generation error: %w", err)
		}
		if resolver == nil {
			return ref, errors.New("InsertValuesAtStem resolve error: resolver is nil")
		}
		data, err := resolver(path, hn.hash)
		if err != nil {
			return ref, fmt.Errorf("InsertValuesAtStem resolve error: %w", err)
		}
		resolved, err := s.DeserializeNodeWithHash(data, depth, hn.hash)
		if err != nil {
			return ref, fmt.Errorf("InsertValuesAtStem node deserialization error: %w", err)
		}
		return s.InsertValuesAtStem(resolved, stem, values, resolver, depth)
	default:
		panic("invalid node kind")
	}
}

// splitStem splits a StemNode when a different stem is inserted at the same location.
func (s *NodeStore) splitStem(stemRef NodeRef, sn *StemNode, newStem []byte, values [][]byte, resolver NodeResolverFn, depth int) (NodeRef, error) {
	bitStem := sn.Stem[sn.depth/8] >> (7 - (sn.depth % 8)) & 1
	sn.depth++

	n := InternalNode{depth: depth, mustRecompute: true}
	var child, other *NodeRef
	if bitStem == 0 {
		n.left = stemRef
		child = &n.left
		other = &n.right
	} else {
		n.right = stemRef
		child = &n.right
		other = &n.left
	}

	bitKey := newStem[n.depth/8] >> (7 - (n.depth % 8)) & 1
	if bitKey == bitStem {
		var err error
		*child, err = s.InsertValuesAtStem(*child, newStem, values, resolver, depth+1)
		if err != nil {
			ref := s.allocInternal(n)
			return ref, fmt.Errorf("insert error: %w", err)
		}
		*other = EmptyRef
	} else {
		*other = s.allocStem(StemNode{
			Stem:          slices.Clone(newStem[:StemSize]),
			Values:        values,
			depth:         n.depth + 1,
			mustRecompute: true,
		})
	}
	return s.allocInternal(n), nil
}

// resolveHashed resolves a KindHashed ref by reading from the resolver.
func (s *NodeStore) resolveHashed(ref NodeRef, parentDepth int, stem []byte, resolver NodeResolverFn) (NodeRef, error) {
	hn := s.getHashed(ref.Index())
	path, err := keyToPath(parentDepth, stem)
	if err != nil {
		return ref, err
	}
	data, err := resolver(path, hn.hash)
	if err != nil {
		return ref, err
	}
	return s.DeserializeNodeWithHash(data, parentDepth+1, hn.hash)
}

// GetHeight returns the height of the subtree rooted at ref.
func (s *NodeStore) GetHeight(ref NodeRef) int {
	switch ref.Kind() {
	case KindEmpty:
		return 0
	case KindStem:
		return 1
	case KindInternal:
		n := s.getInternal(ref.Index())
		lh := s.GetHeight(n.left)
		rh := s.GetHeight(n.right)
		return 1 + max(lh, rh)
	case KindHashed:
		panic("tried to get the height of a hashed node, this is a bug")
	default:
		panic("invalid node kind")
	}
}
