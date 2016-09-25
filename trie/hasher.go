// Copyright 2016 The go-ethereum Authors
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
	"hash"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/rlp"
)

type hasher struct {
	tmp *bytes.Buffer
	sha hash.Hash
}

// hashers live in a global pool.
var hasherPool = sync.Pool{
	New: func() interface{} {
		return &hasher{tmp: new(bytes.Buffer), sha: sha3.NewKeccak256()}
	},
}

func newHasher() *hasher {
	return hasherPool.Get().(*hasher)
}

func returnHasherToPool(h *hasher) {
	hasherPool.Put(h)
}

// hash collapses a node down into a hash node, also returning a copy of the
// original node initialzied with the computed hash to replace the original one.
func (h *hasher) hash(n node, db DatabaseWriter, force bool) (node, node, error) {
	// If we're not storing the node, just hashing, use avaialble cached data
	if hash, dirty := n.cache(); hash != nil && (db == nil || !dirty) {
		return hash, n, nil
	}
	// Trie not processed yet or needs storage, walk the children
	collapsed, cached, err := h.hashChildren(n, db)
	if err != nil {
		return hashNode{}, n, err
	}
	hashed, err := h.store(collapsed, db, force)
	if err != nil {
		return hashNode{}, n, err
	}
	// Cache the hash and RLP blob of the ndoe for later reuse
	if hash, ok := hashed.(hashNode); ok && !force {
		switch cached := cached.(type) {
		case shortNode:
			cached.hash = hash
			if db != nil {
				cached.dirty = false
			}
			return hashed, cached, nil
		case fullNode:
			cached.hash = hash
			if db != nil {
				cached.dirty = false
			}
			return hashed, cached, nil
		}
	}
	return hashed, cached, nil
}

// hashChildren replaces the children of a node with their hashes if the encoded
// size of the child is larger than a hash, returning the collapsed node as well
// as a replacement for the original node with the child hashes cached in.
func (h *hasher) hashChildren(original node, db DatabaseWriter) (node, node, error) {
	var err error

	switch n := original.(type) {
	case shortNode:
		// Hash the short node's child, caching the newly hashed subtree
		cached := n
		cached.Key = common.CopyBytes(cached.Key)

		n.Key = compactEncode(n.Key)
		if _, ok := n.Val.(valueNode); !ok {
			if n.Val, cached.Val, err = h.hash(n.Val, db, false); err != nil {
				return n, original, err
			}
		}
		if n.Val == nil {
			n.Val = valueNode(nil) // Ensure that nil children are encoded as empty strings.
		}
		return n, cached, nil

	case fullNode:
		// Hash the full node's children, caching the newly hashed subtrees
		cached := fullNode{dirty: n.dirty}

		for i := 0; i < 16; i++ {
			if n.Children[i] != nil {
				if n.Children[i], cached.Children[i], err = h.hash(n.Children[i], db, false); err != nil {
					return n, original, err
				}
			} else {
				n.Children[i] = valueNode(nil) // Ensure that nil children are encoded as empty strings.
			}
		}
		cached.Children[16] = n.Children[16]
		if n.Children[16] == nil {
			n.Children[16] = valueNode(nil)
		}
		return n, cached, nil

	default:
		// Value and hash nodes don't have children so they're left as were
		return n, original, nil
	}
}

func (h *hasher) store(n node, db DatabaseWriter, force bool) (node, error) {
	// Don't store hashes or empty nodes.
	if _, isHash := n.(hashNode); n == nil || isHash {
		return n, nil
	}
	// Generate the RLP encoding of the node
	h.tmp.Reset()
	if err := rlp.Encode(h.tmp, n); err != nil {
		panic("encode error: " + err.Error())
	}
	if h.tmp.Len() < 32 && !force {
		return n, nil // Nodes smaller than 32 bytes are stored inside their parent
	}
	// Larger nodes are replaced by their hash and stored in the database.
	hash, _ := n.cache()
	if hash == nil {
		h.sha.Reset()
		h.sha.Write(h.tmp.Bytes())
		hash = hashNode(h.sha.Sum(nil))
	}
	if db != nil {
		return hash, db.Put(hash, h.tmp.Bytes())
	}
	return hash, nil
}
