// Copyright 2025 The go-ethereum Authors
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

import (
	"bytes"

	"github.com/ethereum/go-ethereum/common"
)

// ListHasher is the wrapper of Merkle-Patricia-Trie, implementing the
// types.ListHasher by always deep-copying the supplied slices.
//
// This implementation is very inefficient in terms of memory allocation
// compared with StackTrie, only for correctness comparison purpose.
type ListHasher struct {
	tr *Trie
}

// NewListHasher initializes the list hasher.
func NewListHasher() *ListHasher {
	return &ListHasher{
		tr: NewEmpty(nil),
	}
}

// Reset implements types.ListHasher, clearing the internal state and prepare
// it for reuse.
func (h *ListHasher) Reset() {
	h.tr.reset()
}

// Update implements types.ListHasher, inserting the given key-value pair into
// the trie. The supplied key-value pair should be deep-copied to satisfy the
// interface restriction.
func (h *ListHasher) Update(key []byte, value []byte) error {
	key, value = bytes.Clone(key), bytes.Clone(value)
	return h.tr.Update(key, value)
}

// Hash implements types.ListHasher, computing the final hash of all inserted
// key-value pairs.
func (h *ListHasher) Hash() common.Hash {
	return h.tr.Hash()
}
