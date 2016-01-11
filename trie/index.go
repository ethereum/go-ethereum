// Copyright 2015 The go-ethereum Authors
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

package trie

import "bytes"

var (
	// ParentReferenceIndexPrefix is the database key prefix storing parent references index entries
	ParentReferenceIndexPrefix = []byte("ref-")
)

// ParentReferenceIndexKey constructs a child->parent database index key.
func ParentReferenceIndexKey(parent []byte, child []byte) []byte {
	return append(append(ParentReferenceIndexPrefix, child...), parent...)
}

// storeParentReferences expands a trie node to find all its stored children and
// adds a database reference pointing to the parent to permit reference tracking.
func storeParentReferences(key []byte, node node, db DatabaseWriter) error {
	switch node := node.(type) {
	case fullNode:
		for _, child := range node {
			if child != nil {
				if err := storeParentReferences(key, child, db); err != nil {
					return err
				}
			}
		}
	case shortNode:
		if child := node.Val; child != nil {
			return storeParentReferences(key, child, db)
		}
	case hashNode:
		return db.Put(ParentReferenceIndexKey(key, node), nil)

	case valueNode:
		for _, child := range node.refs {
			if bytes.Compare(child, emptyRoot.Bytes()) == 0 {
				continue // don't index an empty trie
			}
			if err := db.Put(ParentReferenceIndexKey(key, child), nil); err != nil {
				return err
			}
		}
	}
	return nil
}

// storeParentReferenceEntry manually inserts a parent->child reference entry into
// the database index without requiring direct, trie-internal descendancy. This is
// useful to store references to external entities, like accounts or blocks.
func storeParentReferenceEntry(parent []byte, child []byte, db DatabaseWriter) error {
	return db.Put(ParentReferenceIndexKey(parent, child), nil)
}
