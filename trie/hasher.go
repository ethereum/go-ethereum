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
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

// hasher is a type used for the trie Hash operation. A hasher has some
// internal preallocated temp space
type hasher struct {
	sha      crypto.KeccakState
	tmp      []byte
	encbuf   rlp.EncoderBuffer
	parallel bool // Whether to use parallel threads when hashing
}

// hasherPool holds pureHashers
var hasherPool = sync.Pool{
	New: func() any {
		return &hasher{
			tmp:    make([]byte, 0, 550), // cap is as large as a full fullNode.
			sha:    crypto.NewKeccakState(),
			encbuf: rlp.NewEncoderBuffer(nil),
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

// hash collapses a node down into a hash node.
func (h *hasher) hash(n node, force bool) []byte {
	// Return the cached hash if it's available
	if hash, _ := n.cache(); hash != nil {
		return hash
	}
	// Trie not processed yet, walk the children
	switch n := n.(type) {
	case *shortNode:
		enc := h.encodeShortNode(n)
		if len(enc) < 32 && !force {
			// Nodes smaller than 32 bytes are embedded directly in their parent.
			// In such cases, return the raw encoded blob instead of the node hash.
			// It's essential to deep-copy the node blob, as the underlying buffer
			// of enc will be reused later.
			buf := make([]byte, len(enc))
			copy(buf, enc)
			return buf
		}
		hash := h.hashData(enc)
		n.flags.hash = hash
		return hash

	case *fullNode:
		enc := h.encodeFullNode(n)
		if len(enc) < 32 && !force {
			// Nodes smaller than 32 bytes are embedded directly in their parent.
			// In such cases, return the raw encoded blob instead of the node hash.
			// It's essential to deep-copy the node blob, as the underlying buffer
			// of enc will be reused later.
			buf := make([]byte, len(enc))
			copy(buf, enc)
			return buf
		}
		hash := h.hashData(enc)
		n.flags.hash = hash
		return hash

	case hashNode:
		// hash nodes don't have children, so they're left as were
		return n

	default:
		panic(fmt.Errorf("unexpected node type, %T", n))
	}
}

// encodeShortNode encodes the provided shortNode into the bytes. Notably, the
// return slice must be deep-copied explicitly, otherwise the underlying slice
// will be reused later.
func (h *hasher) encodeShortNode(n *shortNode) []byte {
	// Encode leaf node
	if hasTerm(n.Key) {
		var ln leafNodeEncoder
		ln.Key = hexToCompact(n.Key)
		ln.Val = n.Val.(valueNode)
		ln.encode(h.encbuf)
		return h.encodedBytes()
	}
	// Encode extension node
	var en extNodeEncoder
	en.Key = hexToCompact(n.Key)
	en.Val = h.hash(n.Val, false)
	en.encode(h.encbuf)
	return h.encodedBytes()
}

// fnEncoderPool is the pool for storing shared fullNode encoder to mitigate
// the significant memory allocation overhead.
var fnEncoderPool = sync.Pool{
	New: func() interface{} {
		var enc fullnodeEncoder
		return &enc
	},
}

// encodeFullNode encodes the provided fullNode into the bytes. Notably, the
// return slice must be deep-copied explicitly, otherwise the underlying slice
// will be reused later.
func (h *hasher) encodeFullNode(n *fullNode) []byte {
	fn := fnEncoderPool.Get().(*fullnodeEncoder)
	fn.reset()

	if h.parallel {
		var wg sync.WaitGroup
		for i := 0; i < 16; i++ {
			if n.Children[i] == nil {
				continue
			}
			wg.Add(1)
			go func(i int) {
				defer wg.Done()

				h := newHasher(false)
				fn.Children[i] = h.hash(n.Children[i], false)
				returnHasherToPool(h)
			}(i)
		}
		wg.Wait()
	} else {
		for i := 0; i < 16; i++ {
			if child := n.Children[i]; child != nil {
				fn.Children[i] = h.hash(child, false)
			}
		}
	}
	if n.Children[16] != nil {
		fn.Children[16] = n.Children[16].(valueNode)
	}
	fn.encode(h.encbuf)
	fnEncoderPool.Put(fn)

	return h.encodedBytes()
}

// encodedBytes returns the result of the last encoding operation on h.encbuf.
// This also resets the encoder buffer.
//
// All node encoding must be done like this:
//
//	node.encode(h.encbuf)
//	enc := h.encodedBytes()
//
// This convention exists because node.encode can only be inlined/escape-analyzed when
// called on a concrete receiver type.
func (h *hasher) encodedBytes() []byte {
	h.tmp = h.encbuf.AppendToBytes(h.tmp[:0])
	h.encbuf.Reset(nil)
	return h.tmp
}

// hashData hashes the provided data. It is safe to modify the returned slice after
// the function returns.
func (h *hasher) hashData(data []byte) []byte {
	n := make([]byte, 32)
	h.sha.Reset()
	h.sha.Write(data)
	h.sha.Read(n)
	return n
}

// hashDataTo hashes the provided data to the given destination buffer. The caller
// must ensure that the dst buffer is of appropriate size.
func (h *hasher) hashDataTo(dst, data []byte) {
	h.sha.Reset()
	h.sha.Write(data)
	h.sha.Read(dst)
}

// proofHash is used to construct trie proofs, returning the rlp-encoded node blobs.
// Note, only resolved node (shortNode or fullNode) is expected for proofing.
//
// It is safe to modify the returned slice after the function returns.
func (h *hasher) proofHash(original node) []byte {
	switch n := original.(type) {
	case *shortNode:
		return bytes.Clone(h.encodeShortNode(n))
	case *fullNode:
		return bytes.Clone(h.encodeFullNode(n))
	default:
		panic(fmt.Errorf("unexpected node type, %T", original))
	}
}
