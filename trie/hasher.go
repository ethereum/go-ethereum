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
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/rlp"
)

type hasher struct {
	tmp                  *bytes.Buffer
	sha                  hash.Hash
	cachegen, cachelimit uint16
}

// hashers live in a global pool.
var hasherPool = sync.Pool{
	New: func() interface{} {
		return &hasher{tmp: new(bytes.Buffer), sha: sha3.NewKeccak256()}
	},
}

func newHasher(cachegen, cachelimit uint16) *hasher {
	h := hasherPool.Get().(*hasher)
	h.cachegen, h.cachelimit = cachegen, cachelimit
	return h
}

func returnHasherToPool(h *hasher) {
	hasherPool.Put(h)
}

// hash collapses a node down into a hash node, also returning a copy of the
// original node initialized with the computed hash to replace the original one.
func (h *hasher) hash(n node, pool *MemPool, force bool) (node, node, error) {
	// If we're not storing the node, just hashing, use available cached data
	if hash, dirty := n.cache(); hash != nil {
		if pool == nil {
			return hash, n, nil
		}
		if n.canUnload(h.cachegen, h.cachelimit) {
			// Unload the node from cache. All of its subnodes will have a lower or equal
			// cache generation number.
			cacheUnloadCounter.Inc(1)
			return hash, hash, nil
		}
		if !dirty {
			return hash, n, nil
		}
	}
	// Trie not processed yet or needs storage, walk the children
	collapsed, cached, refs, err := h.hashChildren(n, pool)
	if err != nil {
		return hashNode{}, n, err
	}
	hashed, refs, err := h.store(collapsed, refs, pool, force)
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
		if pool != nil {
			cn.flags.dirty = false
		}
	case *fullNode:
		cn.flags.hash = cachedHash
		if pool != nil {
			cn.flags.dirty = false
		}
	}
	return hashed, cached, nil
}

// hashChildren replaces the children of a node with their hashes if the encoded
// size of the child is larger than a hash, returning the collapsed node as well
// as a replacement for the original node with the child hashes cached in.
func (h *hasher) hashChildren(original node, pool *MemPool) (node, node, []common.Hash, error) {
	var err error

	switch n := original.(type) {
	case *shortNode:
		// Hash the short node's child, caching the newly hashed subtree
		collapsed, cached := n.copy(), n.copy()
		collapsed.Key = hexToCompact(n.Key)
		cached.Key = common.CopyBytes(n.Key)

		if _, ok := n.Val.(valueNode); !ok {
			collapsed.Val, cached.Val, err = h.hash(n.Val, pool, false)
			if err != nil {
				return original, original, nil, err
			}
		}
		if collapsed.Val == nil {
			collapsed.Val = valueNode(nil) // Ensure that nil children are encoded as empty strings.
		}
		return collapsed, cached, h.externals(collapsed.Val), nil

	case *fullNode:
		// Hash the full node's children, caching the newly hashed subtrees
		collapsed, cached := n.copy(), n.copy()

		for i := 0; i < 16; i++ {
			if n.Children[i] != nil {
				collapsed.Children[i], cached.Children[i], err = h.hash(n.Children[i], pool, false)
				if err != nil {
					return original, original, nil, err
				}
			} else {
				collapsed.Children[i] = valueNode(nil) // Ensure that nil children are encoded as empty strings.
			}
		}
		cached.Children[16] = n.Children[16]
		if collapsed.Children[16] == nil {
			collapsed.Children[16] = valueNode(nil)
		}
		var refs []common.Hash
		for i := 0; i < 16; i++ {
			refs = append(refs, h.externals(collapsed.Children[i])...)
		}
		return collapsed, cached, refs, nil

	default:
		// Value and hash nodes don't have children so they're left as were
		return n, original, h.externals(n), nil
	}
}

// externals returns any external nodes referenced by a particular node. The only
// current case for it is when an account trie references its storage trie.
func (h *hasher) externals(n node) []common.Hash {
	// Only value nodes can reference external nodes
	val, ok := n.(valueNode)
	if !ok {
		return nil
	}
	// Account nodes have very specific sizes, discard anything else
	// TODO(karalabe): Seriously? Dafuq man?!
	if size := len(val); size < 70 || size > 102 {
		return nil
	}
	// Only account nodes can reference external storage tries
	var account struct {
		Nonce    uint64
		Balance  *big.Int
		Root     common.Hash
		CodeHash []byte
	}
	if err := rlp.DecodeBytes(val, &account); err != nil {
		//fmt.Printf(".")
		return nil
	}
	// Empty tries are not referenced
	if account.Root == emptyState {
		return nil
	}
	return []common.Hash{account.Root}
}

func (h *hasher) store(n node, refs []common.Hash, pool *MemPool, force bool) (node, []common.Hash, error) {
	// Don't store hashes or empty nodes.
	if _, isHash := n.(hashNode); n == nil || isHash {
		return n, refs, nil
	}
	// Generate the RLP encoding of the node
	h.tmp.Reset()
	if err := rlp.Encode(h.tmp, n); err != nil {
		panic("encode error: " + err.Error())
	}

	if h.tmp.Len() < 32 && !force {
		return n, refs, nil // Nodes smaller than 32 bytes are stored inside their parent
	}
	// Larger nodes are replaced by their hash and stored in the database.
	hash, _ := n.cache()
	if hash == nil {
		h.sha.Reset()
		h.sha.Write(h.tmp.Bytes())
		hash = hashNode(h.sha.Sum(nil))
	}
	if pool != nil {
		// We are pooling the trie nodes into an intermediate memory cache
		pool.lock.Lock()
		defer pool.lock.Unlock()

		hash := common.BytesToHash(hash)
		pool.insert(hash, h.tmp.Bytes())

		// Track all direct parent->child node references
		switch n := n.(type) {
		case *shortNode:
			if child, ok := n.Val.(hashNode); ok {
				pool.reference(common.BytesToHash(child), hash)
			}
		case *fullNode:
			for i := 0; i < 16; i++ {
				if child, ok := n.Children[i].(hashNode); ok {
					pool.reference(common.BytesToHash(child), hash)
				}
			}
		}
		// Track external references from account->storage trie
		for _, ext := range refs {
			pool.reference(ext, hash)
		}
	}
	return hash, nil, nil
}
