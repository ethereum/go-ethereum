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
	"hash"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
	"golang.org/x/crypto/sha3"
)

type hasher struct {
	tmp    []sliceBuffer
	sha    []keccakState
	onleaf LeafCallback
}

// keccakState wraps sha3.state. In addition to the usual hash methods, it also supports
// Read to get a variable amount of data from the hash state. Read is faster than Sum
// because it doesn't copy the internal state, but also modifies the internal state.
type keccakState interface {
	hash.Hash
	Read([]byte) (int, error)
}

type sliceBuffer []byte

func (b *sliceBuffer) Write(data []byte) (n int, err error) {
	*b = append(*b, data...)
	return len(data), nil
}

func (b *sliceBuffer) Reset() {
	*b = (*b)[:0]
}

// hashers live in a global db.
var hasherPool = sync.Pool{
	New: func() interface{} {
		return &hasher{
			tmp: []sliceBuffer{
				make(sliceBuffer, 0, 550), // cap is as large as a full fullNode.
				make(sliceBuffer, 0, 550), // cap is as large as a full fullNode.
				make(sliceBuffer, 0, 550), // cap is as large as a full fullNode.
				make(sliceBuffer, 0, 550), // cap is as large as a full fullNode.
			},
			sha: []keccakState{
				sha3.NewLegacyKeccak256().(keccakState),
				sha3.NewLegacyKeccak256().(keccakState),
				sha3.NewLegacyKeccak256().(keccakState),
				sha3.NewLegacyKeccak256().(keccakState),
			},
		}
	},
}

func newHasher(onleaf LeafCallback) *hasher {
	h := hasherPool.Get().(*hasher)
	h.onleaf = onleaf
	return h
}

func returnHasherToPool(h *hasher) {
	hasherPool.Put(h)
}

// hash collapses a node down into a hash node, also returning a copy of the
// original node initialized with the computed hash to replace the original one.
func (h *hasher) hash(n node, db *Database, force bool) (node, node, error) {
	return h.hashParalell(n, db, force, 0)
}
func (h *hasher) hashParalell(n node, db *Database, force bool, id int) (node, node, error) {
	// If we're not storing the node, just hashing, use available cached data
	if hash, dirty := n.cache(); hash != nil {
		if db == nil {
			return hash, n, nil
		}
		if !dirty {
			switch n.(type) {
			case *fullNode, *shortNode:
				return hash, hash, nil
			default:
				return hash, n, nil
			}
		}
	}
	// Trie not processed yet or needs storage, walk the children
	collapsed, cached, err := h.hashChildrenParalell(n, db, id)
	if err != nil {
		return hashNode{}, n, err
	}
	if id == -1 {
		id = 0
	}
	hashed, err := h.store(collapsed, db, force, id)
	if err != nil {
		return hashNode{}, n, err
	}
	// Cache the hash of the node for later reuse and remove
	// the dirty flag in commit mode. It's fine to assign these values directly
	// without copying the node first because hashChildren copies it.
	cachedHash, _ := hashed.(hashNode)
	switch cn := cached.(type) {
	case *shortNode:
		cn.flags.hash = cachedHash
		if db != nil {
			cn.flags.dirty = false
		}
	case *fullNode:
		cn.flags.hash = cachedHash
		if db != nil {
			cn.flags.dirty = false
		}
	}
	return hashed, cached, nil
}

// hashChildren replaces the children of a node with their hashes if the encoded
// size of the child is larger than a hash, returning the collapsed node as well
// as a replacement for the original node with the child hashes cached in.
func (h *hasher) hashChildren(original node, db *Database) (node, node, error) {
	return h.hashChildrenParalell(original, db, 0)
}

func (h *hasher) hashChildrenParalell(original node, db *Database, id int) (node, node, error) {
	var err error

	switch n := original.(type) {
	case *shortNode:
		// Hash the short node's child, caching the newly hashed subtree
		collapsed, cached := n.copy(), n.copy()
		collapsed.Key = hexToCompact(n.Key)
		cached.Key = common.CopyBytes(n.Key)

		if _, ok := n.Val.(valueNode); !ok {
			collapsed.Val, cached.Val, err = h.hashParalell(n.Val, db, false, id)
			if err != nil {
				return original, original, err
			}
		}
		return collapsed, cached, nil

	case *fullNode:
		// Hash the full node's children, caching the newly hashed subtrees
		collapsed, cached := n.copy(), n.copy()
		if id == -1 { // Top level, thread out
			var wg sync.WaitGroup
			wg.Add(3)
			var e1, e2, e3, e4 error
			go func() {
				for i := 0; i < 4; i++ {
					if n.Children[i] != nil {
						collapsed.Children[i], cached.Children[i], e1 = h.hashParalell(n.Children[i], db, false, 0)
						if err != nil {
							return
						}
					}
				}
				wg.Done()
			}()
			go func() {
				for i := 4; i < 8; i++ {
					if n.Children[i] != nil {
						collapsed.Children[i], cached.Children[i], e2 = h.hashParalell(n.Children[i], db, false, 1)
						if err != nil {
							return
						}
					}
				}
				wg.Done()
			}()
			go func() {
				for i := 8; i < 12; i++ {
					if n.Children[i] != nil {
						collapsed.Children[i], cached.Children[i], e3 = h.hashParalell(n.Children[i], db, false, 2)
						if err != nil {
							return
						}
					}
				}
				wg.Done()
			}()
			for i := 12; i < 16; i++ {
				if n.Children[i] != nil {
					collapsed.Children[i], cached.Children[i], e4 = h.hashParalell(n.Children[i], db, false, 3)
					if err != nil {
						break
					}
				}
			}
			wg.Wait()
			if e1 != nil {
				return original, original, e1
			}
			if e2 != nil {
				return original, original, e2
			}

			if e3 != nil {
				return original, original, e3
			}

			if e4 != nil {
				return original, original, e4
			}

		} else {
			for i := 0; i < 16; i++ {
				if n.Children[i] != nil {
					collapsed.Children[i], cached.Children[i], err = h.hashParalell(n.Children[i], db, false, id)
					if err != nil {
						return original, original, err
					}
				}
			}
		}
		cached.Children[16] = n.Children[16]
		return collapsed, cached, nil

	default:
		// Value and hash nodes don't have children so they're left as were
		return n, original, nil
	}
}

// store hashes the node n and if we have a storage layer specified, it writes
// the key/value pair to it and tracks any node->child references as well as any
// node->external trie references.
func (h *hasher) store(n node, db *Database, force bool, id int) (node, error) {
	// Don't store hashes or empty nodes.
	if _, isHash := n.(hashNode); n == nil || isHash {
		return n, nil
	}
	// We might already have the hash
	hash, _ := n.cache()
	if hash == nil {
		// Generate the RLP encoding of the node
		h.tmp[id].Reset()
		if err := rlp.Encode(&h.tmp[id], n); err != nil {
			panic("encode error: " + err.Error())
		}
		if len(h.tmp[id]) < 32 && !force {
			return n, nil // Nodes smaller than 32 bytes are stored inside their parent
		}
		// Larger nodes are replaced by their hash and stored in the database.
		hash = h.makeHashNode(id)
	}
	if db != nil {
		// We are pooling the trie nodes into an intermediate memory cache
		hash := common.BytesToHash(hash)

		db.lock.Lock()
		db.insert(hash, h.tmp[id], n)
		db.lock.Unlock()

		// Track external references from account->storage trie
		if h.onleaf != nil {
			switch n := n.(type) {
			case *shortNode:
				if child, ok := n.Val.(valueNode); ok {
					h.onleaf(child, hash)
				}
			case *fullNode:
				for i := 0; i < 16; i++ {
					if child, ok := n.Children[i].(valueNode); ok {
						h.onleaf(child, hash)
					}
				}
			}
		}
	}
	return hash, nil
}

func (h *hasher) makeHashNode(id int) hashNode {
	n := make(hashNode, h.sha[id].Size())
	h.sha[id].Reset()
	h.sha[id].Write(h.tmp[id])
	h.sha[id].Read(n)
	return n
}
