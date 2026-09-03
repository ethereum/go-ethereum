// Copyright 2026 The go-ethereum Authors
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
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

// PartialStackTrie builds the subtrie of a single first-nibble partition of a
// larger trie on top of a StackTrie. It is used for parallel trie generation,
// where the key space is split into 16 partitions by the first nibble and each
// partition is built independently before being mounted under a common root.
//
// Two adjustments make the produced subtrie line up with its position in the
// full trie:
//
//   - Keys are inserted with their leading nibble stripped. That nibble is
//     implied by the partition's slot in the parent branch, so duplicating it
//     inside the keys would corrupt every node hash below the root.
//
//   - Paths reported to onTrieNode are prefixed with the partition nibble, so
//     a node's path matches its absolute position in the full trie. This is
//     required by the path-based storage scheme, which keys nodes by path.
//
// The hashes themselves are independent of the absolute path, so prefixing the
// path does not change any node hash.
//
// All inserted keys must share the same leading nibble equal to `nibble`; the
// caller guarantees this by construction (e.g. by partitioning a hash range).
type PartialStackTrie struct {
	nibble  byte
	inner   *StackTrie
	pathBuf []byte // reusable buffer for the nibble-prefixed path
}

// NewPartialStackTrie creates a partition builder for the given leading nibble.
// The onTrieNode callback, if non-nil, is invoked for every committed node with
// its absolute path already prefixed with the partition nibble.
func NewPartialStackTrie(nibble byte, onTrieNode OnTrieNode) *PartialStackTrie {
	p := &PartialStackTrie{nibble: nibble}
	p.inner = NewStackTrie(func(path []byte, hash common.Hash, blob []byte) {
		if onTrieNode == nil {
			return
		}
		// Prefix the path with the partition nibble. The buffer is reused across
		// calls, so the callback must consume it synchronously.
		p.pathBuf = append(p.pathBuf[:0], nibble)
		p.pathBuf = append(p.pathBuf, path...)
		onTrieNode(p.pathBuf, hash, blob)
	})
	return p
}

// Update inserts a (key, value) pair, stripping the key's leading nibble, which
// is implied by the partition. The key must begin with the partition nibble.
func (p *PartialStackTrie) Update(key, value []byte) error {
	if len(value) == 0 {
		return errors.New("trying to insert empty (deletion)")
	}
	t := p.inner
	t.grow(key)
	k := writeHexKey(t.kBuf, key)

	if k[0] != p.nibble {
		return fmt.Errorf("unexpected nibble %v, expected %x", k[0], p.nibble)
	}
	return t.update(k[1:], value)
}

// Hash returns the root hash of the partition subtrie (built with the leading
// nibble stripped). It is the reference the parent branch mounts in slot `nibble`.
func (p *PartialStackTrie) Hash() common.Hash {
	return p.inner.Hash()
}
