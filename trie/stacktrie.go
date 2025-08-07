// Copyright 2020 The go-ethereum Authors
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
	"errors"
	"sync"
	"unsafe"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/metrics"
)

var stNodeAllocationMeter = metrics.NewRegisteredMeter("stacktrie/allocation/node", nil)
var byteAllocationMeter = metrics.NewRegisteredMeter("stacktrie/allocation/byte", nil)

var (
	stPageSize = 1024
	stPagePool = sync.Pool{
		New: func() any {
			stNodeAllocationMeter.Mark(int64(unsafe.Sizeof(stNode{})) * int64(stPageSize))
			return make([]stNode, stPageSize)
		},
	}
	bytePagePool = sync.Pool{
		New: func() any {
			byteAllocationMeter.Mark(int64(unsafe.Sizeof([32]byte{})) * int64(stPageSize))
			return make([][32]byte, stPageSize)
		},
	}
	_ = types.TrieHasher((*StackTrie)(nil))
)

// OnTrieNode is a callback method invoked when a trie node is committed
// by the stack trie. The node is only committed if it's considered complete.
//
// The caller should not modify the contents of the returned path and blob
// slice, and their contents may be changed after the call. It is up to the
// `onTrieNode` receiver function to deep-copy the data if it wants to retain
// it after the call ends.
type OnTrieNode func(path []byte, hash common.Hash, blob []byte)

// allocationFrame keeps track of the position of the allocators in StackTrie
// at the time of a new branch/ext node being created. At the point where this
// node is hashed, allocators can be reset to the positions stored in the allocation
// frame
type allocationFrame struct {
	node, bytes uint32
}

// StackTrie is a trie implementation that expects keys to be inserted
// in order. Once it determines that a subtree will no longer be inserted
// into, it will hash it and free up the memory it uses.
type StackTrie struct {
	root       *stNode
	h          *hasher
	last       []byte
	onTrieNode OnTrieNode

	nodeAllocator         common.Arena[stNode]
	byteAllocator         common.Arena[[32]byte]
	allocationStackFrames []allocationFrame

	kBuf    []byte  // buf space used for hex-key during insertions
	pBuf    []byte  // buf space used for path during insertions
	tmpNode *stNode // used as a temporary ext node when needed
}

// NewStackTrie allocates and initializes an empty trie. The committed nodes
// will be discarded immediately if no callback is configured.
func NewStackTrie(onTrieNode OnTrieNode) *StackTrie {
	t := StackTrie{
		h:             newHasher(false),
		onTrieNode:    onTrieNode,
		kBuf:          make([]byte, 64),
		pBuf:          make([]byte, 64),
		nodeAllocator: *common.NewArena[stNode](uint32(stPageSize), stPagePool.Get, stPagePool.Put),
		byteAllocator: *common.NewArena[[32]byte](uint32(stPageSize), bytePagePool.Get, bytePagePool.Put),
	}
	t.root = t.nodeAllocator.Alloc().reset()
	t.tmpNode = t.nodeAllocator.Alloc().reset()
	return &t
}

// creates a new allocation frame by saving positions of internal allocators
func (t *StackTrie) pushAllocationFrame() {
	t.allocationStackFrames = append(t.allocationStackFrames, allocationFrame{
		node:  t.nodeAllocator.Used(),
		bytes: t.byteAllocator.Used(),
	})
}

// pops a saved allocation frame and rollsback allocators to the state they were
// at the beginning of the frame
func (t *StackTrie) popAllocationFrame() {
	allocationFrame := t.allocationStackFrames[len(t.allocationStackFrames)-1]
	t.nodeAllocator.Reset(allocationFrame.node)
	t.byteAllocator.Reset(allocationFrame.bytes)
	t.allocationStackFrames = t.allocationStackFrames[:len(t.allocationStackFrames)-1]
}

func (t *StackTrie) grow(key []byte) {
	if cap(t.kBuf) < 2*len(key) {
		t.kBuf = make([]byte, 2*len(key))
	}
	if cap(t.pBuf) < 2*len(key) {
		t.pBuf = make([]byte, 2*len(key))
	}
}

// Update inserts a (key, value) pair into the stack trie.
func (t *StackTrie) Update(key, value []byte) error {
	if len(value) == 0 {
		return errors.New("trying to insert empty (deletion)")
	}
	t.grow(key)
	k := writeHexKey(t.kBuf, key)
	if bytes.Compare(t.last, k) >= 0 {
		return errors.New("non-ascending key order")
	}
	if t.last == nil {
		t.last = append([]byte{}, k...) // allocate key slice
	} else {
		t.last = append(t.last[:0], k...) // reuse key slice
	}
	t.insert(t.root, k, value, t.pBuf[:0])
	return nil
}

// Reset resets the stack trie object to empty state.
func (t *StackTrie) Reset() {
	t.nodeAllocator.Reset(0)
	t.byteAllocator.Reset(0)
	t.root = t.nodeAllocator.Alloc().reset()
	t.tmpNode = t.nodeAllocator.Alloc().reset()
	t.last = nil
}

// TrieKey returns the internal key representation for the given user key.
func (t *StackTrie) TrieKey(key []byte) []byte {
	k := keybytesToHex(key)
	k = k[:len(k)-1] // chop the termination flag
	return k
}

// stNode represents a node within a StackTrie
type stNode struct {
	typ      uint8       // node type (as in branch, ext, leaf)
	key      []byte      // key chunk covered by this (leaf|ext) node
	val      []byte      // value contained by this node if it's a leaf
	children [16]*stNode // list of children (for branch and exts)
}

// newLeaf constructs a leaf node with provided node key and value. The key
// will be deep-copied in the function and safe to modify afterwards, but
// value is not.
func (t *StackTrie) newLeaf(key, val []byte) *stNode {
	st := t.nodeAllocator.Alloc().reset()
	st.typ = leafNode
	st.key = append(st.key, key...)
	st.val = val
	return st
}

// makeExt constructs an extension node with provided node key and child. The
// key will be deep-copied in the function and safe to modify afterwards.
func makeExt(st *stNode, key []byte, child *stNode) *stNode {
	st.typ = extNode
	st.key = append(st.key, key...)
	st.children[0] = child
	return st
}

// List all values that stNode#nodeType can hold
const (
	emptyNode = iota
	branchNode
	extNode
	leafNode
	hashedNode
)

func (n *stNode) reset() *stNode {
	n.key = n.key[:0]
	n.val = nil
	for i := range n.children {
		n.children[i] = nil
	}
	n.typ = emptyNode
	return n
}

// Helper function that, given a full key, determines the index
// at which the chunk pointed by st.keyOffset is different from
// the same chunk in the full key.
func (n *stNode) getDiffIndex(key []byte) int {
	for idx, nibble := range n.key {
		if nibble != key[idx] {
			return idx
		}
	}
	return len(n.key)
}

// Helper function to that inserts a (key, value) pair into the trie.
//
//   - The key is not retained by this method, but always copied if needed.
//   - The value is retained by this method, as long as the leaf that it represents
//     remains unhashed. However: it is never modified.
//   - The path is not retained by this method.
func (t *StackTrie) insert(st *stNode, key, value []byte, path []byte) {
	switch st.typ {
	case branchNode: /* Branch */
		idx := int(key[0])

		// Unresolve elder siblings
		for i := idx - 1; i >= 0; i-- {
			if st.children[i] != nil {
				if st.children[i].typ != hashedNode {
					t.hash(st.children[i], append(path, byte(i)))
				}
				break
			}
		}

		// Add new child
		if st.children[idx] == nil {
			st.children[idx] = t.newLeaf(key[1:], value)
		} else {
			t.insert(st.children[idx], key[1:], value, append(path, key[0]))
		}

	case extNode: /* Ext */
		// Compare both key chunks and see where they differ
		diffidx := st.getDiffIndex(key)

		// Check if chunks are identical. If so, recurse into
		// the child node. Otherwise, the key has to be split
		// into 1) an optional common prefix, 2) the fullnode
		// representing the two differing path, and 3) a leaf
		// for each of the differentiated subtrees.
		if diffidx == len(st.key) {
			// Ext key and key segment are identical, recurse into
			// the child node.
			t.insert(st.children[0], key[diffidx:], value, append(path, key[:diffidx]...))
			return
		}
		// Save the original part. Depending if the break is
		// at the extension's last byte or not, create an
		// intermediate extension or use the extension's child
		// node directly.
		var n *stNode
		if diffidx < len(st.key)-1 {
			// Break on the non-last byte, insert an intermediate
			// extension. The path prefix of the newly-inserted
			// extension should also contain the different byte.
			e := makeExt(t.tmpNode.reset(), st.key[diffidx+1:], st.children[0]) // build a temporary extension node to hash
			t.hash(e, append(path, st.key[:diffidx+1]...))                      // frame belonging to st gets popped here

			// allocate a new node to hold the hashed e
			n = t.nodeAllocator.Alloc().reset()
			n.typ = hashedNode
			n.val = e.val
			t.pushAllocationFrame() // for the "new" st that might end up being a branch or a ext
		} else {
			// Break on the last byte, no need to insert
			// an extension node: reuse the current node.
			// The path prefix of the original part should
			// still be same.
			n = st.children[0]
			t.hash(n, append(path, st.key...))
		}
		var p *stNode
		if diffidx == 0 {
			// the break is on the first byte, so
			// the current node is converted into
			// a branch node.
			st.children[0] = nil
			p = st
			st.typ = branchNode
		} else {
			// the common prefix is at least one byte
			// long, insert a new intermediate branch
			// node.
			st.children[0] = t.nodeAllocator.Alloc().reset()
			st.children[0].typ = branchNode
			t.pushAllocationFrame() // for the new branch child
			p = st.children[0]
		}

		// Create a leaf for the inserted part
		o := t.newLeaf(key[diffidx+1:], value)

		// Insert both child leaves where they belong:
		origIdx := st.key[diffidx]
		newIdx := key[diffidx]
		p.children[origIdx] = n
		p.children[newIdx] = o
		st.key = st.key[:diffidx]

	case leafNode: /* Leaf */
		// Compare both key chunks and see where they differ
		diffidx := st.getDiffIndex(key)

		// Overwriting a key isn't supported, which means that
		// the current leaf is expected to be split into 1) an
		// optional extension for the common prefix of these 2
		// keys, 2) a fullnode selecting the path on which the
		// keys differ, and 3) one leaf for the differentiated
		// component of each key.
		if diffidx >= len(st.key) {
			panic("Trying to insert into existing key")
		}

		// Check if the split occurs at the first nibble of the
		// chunk. In that case, no prefix extnode is necessary.
		// Otherwise, create that
		var p *stNode
		if diffidx == 0 {
			// Convert current leaf into a branch
			st.typ = branchNode
			p = st
			st.children[0] = nil
			t.pushAllocationFrame() // leafnode turning into a branch node
		} else {
			// Convert current node into an ext,
			// and insert a child branch node.
			st.typ = extNode
			t.pushAllocationFrame() // leafnode turning into a ext node
			st.children[0] = t.nodeAllocator.Alloc().reset()
			t.pushAllocationFrame() // new branch node
			st.children[0].typ = branchNode
			p = st.children[0]
		}

		// Create the two child leaves: one containing the original
		// value and another containing the new value. The child leaf
		// is hashed directly in order to free up some memory.
		origIdx := st.key[diffidx]
		p.children[origIdx] = t.newLeaf(st.key[diffidx+1:], st.val)
		t.hash(p.children[origIdx], append(path, st.key[:diffidx+1]...))

		newIdx := key[diffidx]
		p.children[newIdx] = t.newLeaf(key[diffidx+1:], value)

		// Finally, cut off the key part that has been passed
		// over to the children.
		st.key = st.key[:diffidx]
		st.val = nil

	case emptyNode: /* Empty */
		st.typ = leafNode
		st.key = append(st.key, key...) // deep-copy the key as it's volatile
		st.val = value

	case hashedNode:
		panic("trying to insert into hash")

	default:
		panic("invalid type")
	}
}

// hash converts st into a 'hashedNode', if possible. Possible outcomes:
//
// 1. The rlp-encoded value was >= 32 bytes:
//   - Then the 32-byte `hash` will be accessible in `st.val`.
//   - And the 'st.type' will be 'hashedNode'
//
// 2. The rlp-encoded value was < 32 bytes
//   - Then the <32 byte rlp-encoded value will be accessible in 'st.val'.
//   - And the 'st.type' will be 'hashedNode' AGAIN
//
// This method also sets 'st.type' to hashedNode, and clears 'st.key'.
func (t *StackTrie) hash(st *stNode, path []byte) {
	var blob []byte // RLP-encoded node blob
	switch st.typ {
	case hashedNode:
		return

	case emptyNode:
		st.val = types.EmptyRootHash.Bytes()
		st.key = st.key[:0]
		st.typ = hashedNode
		return

	case branchNode:
		var nodes fullnodeEncoder
		for i, child := range st.children {
			if child == nil {
				continue
			}
			t.hash(child, append(path, byte(i)))
			nodes.Children[i] = child.val
		}
		nodes.encode(t.h.encbuf)
		blob = t.h.encodedBytes()

		for i, child := range st.children {
			if child == nil {
				continue
			}
			st.children[i] = nil
		}
		t.popAllocationFrame()
	case extNode:
		// recursively hash and commit child as the first step
		t.hash(st.children[0], append(path, st.key...))

		// encode the extension node
		n := extNodeEncoder{
			Key: hexToCompactInPlace(st.key),
			Val: st.children[0].val,
		}
		n.encode(t.h.encbuf)
		blob = t.h.encodedBytes()
		st.children[0] = nil
		t.popAllocationFrame()
	case leafNode:
		st.key = append(st.key, byte(16))
		n := leafNodeEncoder{
			Key: hexToCompactInPlace(st.key),
			Val: st.val,
		}
		n.encode(t.h.encbuf)
		blob = t.h.encodedBytes()

	default:
		panic("invalid node type")
	}
	// Convert the node type to hashNode and reset the key slice.
	st.typ = hashedNode
	st.key = st.key[:0]

	st.val = nil // Release reference to potentially externally held slice.

	// Skip committing the non-root node if the size is smaller than 32 bytes
	// as tiny nodes are always embedded in their parent except root node.
	if len(blob) < 32 && len(path) > 0 {
		st.val = t.byteAllocator.Alloc()[:len(blob)]
		copy(st.val, blob)
		return
	}
	// Write the hash to the 'val'. We allocate a new val here to not mutate
	// input values.
	st.val = t.byteAllocator.Alloc()[:32]
	t.h.hashDataTo(st.val, blob)

	// Invoke the callback it's provided. Notably, the path and blob slices are
	// volatile, please deep-copy the slices in callback if the contents need
	// to be retained.
	if t.onTrieNode != nil {
		t.onTrieNode(path, common.BytesToHash(st.val), blob)
	}
}

// Hash will firstly hash the entire trie if it's still not hashed and then commit
// all leftover nodes to the associated database. Actually most of the trie nodes
// have been committed already. The main purpose here is to commit the nodes on
// right boundary.
func (t *StackTrie) Hash() common.Hash {
	n := t.root
	t.hash(n, nil)
	hash := common.BytesToHash(n.val)
	t.byteAllocator.Release()
	t.nodeAllocator.Release()
	return hash
}
