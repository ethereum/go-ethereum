// Copyright 2019 The go-ethereum Authors
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

	"github.com/XinFinOrg/XDPoSChain/rlp"
	"golang.org/x/crypto/sha3"
)

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

// hasher is a type used for the trie Hash operation. A hasher has some
// internal preallocated temp space
type hasher struct {
	sha      keccakState
	tmp      sliceBuffer
	parallel bool // Whether to use paralallel threads when hashing
}

// hasherPool holds pureHashers
var hasherPool = sync.Pool{
	New: func() interface{} {
		return &hasher{
			tmp: make(sliceBuffer, 0, 550), // cap is as large as a full FullNode.
			sha: sha3.NewLegacyKeccak256().(keccakState),
		}
	},
}

func newHasher(parallel bool) *hasher {
	h := hasherPool.Get().(*hasher)
	h.parallel = parallel
	return h
}

func returnHasherToPool(h *hasher) {
	hasherPool.Put(h)
}

// hash collapses a Node down into a hash Node, also returning a copy of the
// original Node initialized with the computed hash to replace the original one.
func (h *hasher) hash(n Node, force bool) (hashed Node, cached Node) {
	// We're not storing the Node, just hashing, use available cached data
	if hash, _ := n.Cache(); hash != nil {
		return hash, n
	}
	// Trie not processed yet or needs storage, walk the children
	switch n := n.(type) {
	case *ShortNode:
		collapsed, cached := h.hashShortNodeChildren(n)
		hashed := h.shortnodeToHash(collapsed, force)
		// We need to retain the possibly _not_ hashed Node, in case it was too
		// small to be hashed
		if hn, ok := hashed.(HashNode); ok {
			cached.flags.hash = hn
		} else {
			cached.flags.hash = nil
		}
		return hashed, cached
	case *FullNode:
		collapsed, cached := h.hashFullNodeChildren(n)
		hashed = h.fullnodeToHash(collapsed, force)
		if hn, ok := hashed.(HashNode); ok {
			cached.flags.hash = hn
		} else {
			cached.flags.hash = nil
		}
		return hashed, cached
	default:
		// Value and hash nodes don't have children so they're left as were
		return n, n
	}
}

// hashShortNodeChildren collapses the short Node. The returned collapsed Node
// holds a live reference to the Key, and must not be modified.
// The cached
func (h *hasher) hashShortNodeChildren(n *ShortNode) (collapsed, cached *ShortNode) {
	// Hash the short Node's child, caching the newly hashed subtree
	collapsed, cached = n.copy(), n.copy()
	// Previously, we did copy this one. We don't seem to need to actually
	// do that, since we don't overwrite/reuse keys
	//cached.Key = common.CopyBytes(n.Key)
	collapsed.Key = hexToCompact(n.Key)
	// Unless the child is a valuenode or hashnode, hash it
	switch n.Val.(type) {
	case *FullNode, *ShortNode:
		collapsed.Val, cached.Val = h.hash(n.Val, false)
	}
	return collapsed, cached
}

func (h *hasher) hashFullNodeChildren(n *FullNode) (collapsed *FullNode, cached *FullNode) {
	// Hash the full Node's children, caching the newly hashed subtrees
	cached = n.copy()
	collapsed = n.copy()
	if h.parallel {
		var wg sync.WaitGroup
		wg.Add(16)
		for i := 0; i < 16; i++ {
			go func(i int) {
				hasher := newHasher(false)
				if child := n.Children[i]; child != nil {
					collapsed.Children[i], cached.Children[i] = hasher.hash(child, false)
				} else {
					collapsed.Children[i] = nilValueNode
				}
				returnHasherToPool(hasher)
				wg.Done()
			}(i)
		}
		wg.Wait()
	} else {
		for i := 0; i < 16; i++ {
			if child := n.Children[i]; child != nil {
				collapsed.Children[i], cached.Children[i] = h.hash(child, false)
			} else {
				collapsed.Children[i] = nilValueNode
			}
		}
	}
	return collapsed, cached
}

// shortnodeToHash creates a HashNode from a ShortNode. The supplied shortnode
// should have hex-type Key, which will be converted (without modification)
// into compact form for RLP encoding.
// If the rlp data is smaller than 32 bytes, `nil` is returned.
func (h *hasher) shortnodeToHash(n *ShortNode, force bool) Node {
	h.tmp.Reset()
	if err := rlp.Encode(&h.tmp, n); err != nil {
		panic("encode error: " + err.Error())
	}

	if len(h.tmp) < 32 && !force {
		return n // Nodes smaller than 32 bytes are stored inside their parent
	}
	return h.hashData(h.tmp)
}

// shortnodeToHash is used to creates a HashNode from a set of hashNodes, (which
// may contain nil values)
func (h *hasher) fullnodeToHash(n *FullNode, force bool) Node {
	h.tmp.Reset()
	// Generate the RLP encoding of the Node
	if err := n.EncodeRLP(&h.tmp); err != nil {
		panic("encode error: " + err.Error())
	}

	if len(h.tmp) < 32 && !force {
		return n // Nodes smaller than 32 bytes are stored inside their parent
	}
	return h.hashData(h.tmp)
}

// hashData hashes the provided data
func (h *hasher) hashData(data []byte) HashNode {
	n := make(HashNode, 32)
	h.sha.Reset()
	h.sha.Write(data)
	h.sha.Read(n)
	return n
}

// proofHash is used to construct trie proofs, and returns the 'collapsed'
// Node (for later RLP encoding) aswell as the hashed Node -- unless the
// Node is smaller than 32 bytes, in which case it will be returned as is.
// This method does not do anything on value- or hash-nodes.
func (h *hasher) proofHash(original Node) (collapsed, hashed Node) {
	switch n := original.(type) {
	case *ShortNode:
		sn, _ := h.hashShortNodeChildren(n)
		return sn, h.shortnodeToHash(sn, false)
	case *FullNode:
		fn, _ := h.hashFullNodeChildren(n)
		return fn, h.fullnodeToHash(fn, false)
	default:
		// Value and hash nodes don't have children so they're left as were
		return n, n
	}
}
