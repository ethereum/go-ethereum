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

package bintrie

import (
	"bytes"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/triedb/database"
	"github.com/holiman/uint256"
)

// BinaryTrie is the EIP-8297 Partitioned Binary Tree: a single unified state
// tree over variable-length, prefix-free keys, holding account headers,
// contract storage and contract code in zone-partitioned key space.
//
// The trie is not safe for concurrent use: lazy resolution mutates the tree
// on reads.
type BinaryTrie struct {
	root      binaryNode
	reader    *trie.Reader
	tracer    *trie.PrevalueTracer
	ops       *opTracer
	committed bool
}

// NewBinaryTrie creates a binary trie rooted at the given hash, backed by
// the node database. Both empty-root sentinels (the zero EmptyBinaryHash and
// the MPT EmptyRootHash) denote a fresh tree.
func NewBinaryTrie(root common.Hash, db database.NodeDatabase) (*BinaryTrie, error) {
	reader, err := trie.NewReader(root, common.Hash{}, db)
	if err != nil {
		return nil, err
	}
	t := &BinaryTrie{
		root:   empty{},
		reader: reader,
		tracer: trie.NewPrevalueTracer(),
		ops:    newOpTracer(),
	}
	if root != types.EmptyBinaryHash && root != types.EmptyRootHash {
		blob, err := t.resolveBlob(nil, root)
		if err != nil {
			return nil, err
		}
		node, err := decodeNodeWithHash(blob, root)
		if err != nil {
			return nil, err
		}
		t.root = node
	}
	return t, nil
}

// resolveBlob loads a node blob by path and hash, recording it in the
// witness tracer.
func (t *BinaryTrie) resolveBlob(path []byte, hash common.Hash) ([]byte, error) {
	blob, err := t.reader.Node(path, hash)
	if err != nil {
		return nil, err
	}
	t.tracer.Put(path, blob)
	return blob, nil
}

// resolve loads and decodes the node behind a hashedNode sitting at the
// given walk position.
func (t *BinaryTrie) resolve(h hashedNode, walk bitstr) (binaryNode, error) {
	blob, err := t.resolveBlob(pathOf(walk), common.Hash(h))
	if err != nil {
		return nil, err
	}
	return decodeNodeWithHash(blob, common.Hash(h))
}

// pathOf encodes a walk position as a database path.
func pathOf(walk bitstr) []byte {
	if walk.n == 0 {
		return nil
	}
	return encodeBitPrefix(walk)
}

// keyWalk returns the walk position after consuming n bits of key.
func keyWalk(key []byte, n int) bitstr {
	return slice(key, 0, n)
}

//
// Reads
//

// getStemGroup walks to the group holding stem, returning nil if the stem is
// absent. The tree is mutated in place to cache resolved nodes.
func (t *BinaryTrie) getStemGroup(stem []byte) (*groupNode, error) {
	if t.committed {
		return nil, trie.ErrCommitted
	}
	n, g, err := t.getStem(t.root, stem, 0)
	if err != nil {
		return nil, err
	}
	t.root = n
	return g, nil
}

func (t *BinaryTrie) getStem(n binaryNode, stem []byte, pos int) (binaryNode, *groupNode, error) {
	switch nn := n.(type) {
	case empty:
		return n, nil, nil
	case hashedNode:
		resolved, err := t.resolve(nn, keyWalk(stem, pos))
		if err != nil {
			return n, nil, err
		}
		return t.getStem(resolved, stem, pos)
	case *groupNode:
		if !bytes.Equal(nn.stem, stem) {
			return n, nil, nil
		}
		return n, nn, nil
	case *branchNode:
		m := nn.prefix.matchKey(stem, pos)
		if m < nn.prefix.n {
			return n, nil, nil // diverges inside the prefix: absent
		}
		split := pos + nn.prefix.n
		if split >= 8*len(stem) {
			return n, nil, nil // ran out of stem bits: different (longer) zone
		}
		if bitAt(stem, split) == 0 {
			child, g, err := t.getStem(nn.left, stem, split+1)
			if child != nn.left {
				nn.left = child // only on resolution: a write barrier per level otherwise
			}
			return n, g, err
		}
		child, g, err := t.getStem(nn.right, stem, split+1)
		if child != nn.right {
			nn.right = child
		}
		return n, g, err
	default:
		return n, nil, fmt.Errorf("bintrie: unknown node type %T", n)
	}
}

// getValue returns the value stored at key, or nil if absent.
func (t *BinaryTrie) getValue(key []byte) ([]byte, error) {
	if err := validateKey(key); err != nil {
		return nil, err
	}
	g, err := t.getStemGroup(key[:len(key)-1])
	if err != nil || g == nil {
		return nil, err
	}
	return g.lookup(key[len(key)-1]), nil
}

//
// Writes
//

// newGroup builds a group from parallel sub/value slices, skipping nil
// (deleted) values. Returns nil if no value remains.
func newGroup(stem []byte, subs []byte, vals [][]byte) *groupNode {
	g := &groupNode{stem: append([]byte{}, stem...), dirty: true, modified: true}
	for i, v := range vals {
		if v != nil {
			g.set(subs[i], v)
		}
	}
	if len(g.subs) == 0 {
		return nil
	}
	return g
}

// UpdateStem applies a batch of writes to one stem in a single walk. subs
// and values run in parallel; a nil value deletes that sub-index. This is
// the engine's core mutation: single-value updates and deletions are
// one-element batches.
func (t *BinaryTrie) UpdateStem(stem []byte, subs []byte, values [][]byte) error {
	if t.committed {
		return trie.ErrCommitted
	}
	if err := validateStem(stem); err != nil {
		return err
	}
	if len(subs) != len(values) || len(subs) == 0 {
		return fmt.Errorf("bintrie: malformed stem batch (%d subs, %d values)", len(subs), len(values))
	}
	for _, v := range values {
		if v != nil && len(v) != 32 {
			return fmt.Errorf("bintrie: invalid value length %d", len(v))
		}
	}
	root, err := t.insStem(t.root, stem, subs, values, 0)
	if err != nil {
		return err
	}
	t.root = root
	return nil
}

func (t *BinaryTrie) insStem(n binaryNode, stem []byte, subs []byte, vals [][]byte, pos int) (binaryNode, error) {
	switch nn := n.(type) {
	case empty:
		g := newGroup(stem, subs, vals)
		if g == nil {
			return n, nil // pure deletion on an absent stem
		}
		t.ops.onInsert(pathOf(keyWalk(stem, pos)))
		return g, nil

	case hashedNode:
		resolved, err := t.resolve(nn, keyWalk(stem, pos))
		if err != nil {
			return n, err
		}
		return t.insStem(resolved, stem, subs, vals, pos)

	case *groupNode:
		if bytes.Equal(nn.stem, stem) {
			for i, v := range vals {
				nn.set(subs[i], v)
			}
			if len(nn.subs) == 0 {
				t.ops.onDelete(pathOf(keyWalk(stem, pos)))
				return empty{}, nil
			}
			return nn, nil
		}
		// Different stem: split at the divergence point.
		g := newGroup(stem, subs, vals)
		if g == nil {
			return n, nil // pure deletion on an absent stem
		}
		run := commonPrefixLen(stem, nn.stem, pos)
		split := pos + run
		// The existing group's record moves below the new branch.
		t.ops.onDelete(pathOf(keyWalk(nn.stem, pos)))
		t.ops.onInsert(pathOf(keyWalk(nn.stem, split+1)))
		nn.dirty, nn.modified = true, true // position (and, for multi-value groups, hash) changes
		t.ops.onInsert(pathOf(keyWalk(stem, split+1)))
		branch := &branchNode{prefix: slice(stem, pos, run), dirty: true, modified: true}
		if bitAt(stem, split) == 0 {
			branch.left, branch.right = g, nn
		} else {
			branch.left, branch.right = nn, g
		}
		t.ops.onInsert(pathOf(keyWalk(stem, pos)))
		return branch, nil

	case *branchNode:
		m := nn.prefix.matchKey(stem, pos)
		if m == nn.prefix.n {
			split := pos + nn.prefix.n
			bit := bitAt(stem, split)
			var (
				child binaryNode
				err   error
			)
			if bit == 0 {
				child, err = t.insStem(nn.left, stem, subs, vals, split+1)
			} else {
				child, err = t.insStem(nn.right, stem, subs, vals, split+1)
			}
			if err != nil {
				return n, err
			}
			if _, gone := child.(empty); gone {
				return t.collapse(nn, keyWalk(stem, pos), bit)
			}
			if bit == 0 {
				nn.left = child
			} else {
				nn.right = child
			}
			nn.dirty, nn.modified = true, true
			return nn, nil
		}
		// Diverges inside the prefix: split the branch. The survivor keeps
		// the bits after the split; a new top branch keeps the bits before
		// it and occupies this node's path.
		g := newGroup(stem, subs, vals)
		if g == nil {
			return n, nil
		}
		split := pos + m
		survivorBit := nn.prefix.bit(m)
		survivorPath := keyWalk(stem, split).concat(survivorBit, bitstr{})
		t.ops.onInsert(pathOf(survivorPath))
		survivor := &branchNode{
			prefix:   nn.prefix.tail(m + 1),
			left:     nn.left,
			right:    nn.right,
			dirty:    true,
			modified: true,
		}
		t.ops.onInsert(pathOf(keyWalk(stem, split+1)))
		top := &branchNode{prefix: nn.prefix.head(m), dirty: true, modified: true}
		if bitAt(stem, split) == 0 {
			top.left, top.right = g, survivor
		} else {
			top.left, top.right = survivor, g
		}
		return top, nil

	default:
		return n, fmt.Errorf("bintrie: unknown node type %T", n)
	}
}

// collapse handles a branch losing one child: the surviving sibling merges
// up into the branch's position. walk is the branch's own position, bit the
// emptied side.
func (t *BinaryTrie) collapse(n *branchNode, walk bitstr, bit byte) (binaryNode, error) {
	sibling := n.right
	siblingBit := byte(1)
	if bit == 1 {
		sibling, siblingBit = n.left, 0
	}
	siblingWalk := walk.append(n.prefix).concat(siblingBit, bitstr{})
	if h, ok := sibling.(hashedNode); ok {
		resolved, err := t.resolve(h, siblingWalk)
		if err != nil {
			return n, err
		}
		sibling = resolved
	}
	t.ops.onDelete(pathOf(siblingWalk))
	t.ops.onInsert(pathOf(walk))
	switch s := sibling.(type) {
	case *branchNode:
		s.prefix = n.prefix.concat(siblingBit, s.prefix)
		s.dirty, s.modified = true, true
		return s, nil
	case *groupNode:
		s.dirty, s.modified = true, true // stored depth changes; single-value groups just move
		return s, nil
	default:
		return n, fmt.Errorf("bintrie: invalid collapse survivor %T", sibling)
	}
}

// removeStem deletes an entire stem group, bubbling the collapse upward.
// Used by DeleteAccount to drop a header stem wholesale.
func (t *BinaryTrie) removeStem(stem []byte) error {
	if t.committed {
		return trie.ErrCommitted
	}
	root, _, err := t.delStem(t.root, stem, 0)
	if err != nil {
		return err
	}
	t.root = root
	return nil
}

func (t *BinaryTrie) delStem(n binaryNode, stem []byte, pos int) (binaryNode, bool, error) {
	switch nn := n.(type) {
	case empty:
		return n, false, nil
	case hashedNode:
		resolved, err := t.resolve(nn, keyWalk(stem, pos))
		if err != nil {
			return n, false, err
		}
		return t.delStem(resolved, stem, pos)
	case *groupNode:
		if !bytes.Equal(nn.stem, stem) {
			return n, false, nil
		}
		t.ops.onDelete(pathOf(keyWalk(stem, pos)))
		return empty{}, true, nil
	case *branchNode:
		m := nn.prefix.matchKey(stem, pos)
		if m < nn.prefix.n {
			return n, false, nil
		}
		split := pos + nn.prefix.n
		if split >= 8*len(stem) {
			return n, false, nil
		}
		bit := bitAt(stem, split)
		var (
			child   binaryNode
			removed bool
			err     error
		)
		if bit == 0 {
			child, removed, err = t.delStem(nn.left, stem, split+1)
		} else {
			child, removed, err = t.delStem(nn.right, stem, split+1)
		}
		if err != nil {
			return n, false, err
		}
		if _, gone := child.(empty); gone {
			merged, err := t.collapse(nn, keyWalk(stem, pos), bit)
			return merged, removed, err
		}
		if bit == 0 {
			nn.left = child
		} else {
			nn.right = child
		}
		if removed {
			nn.dirty, nn.modified = true, true
		}
		return nn, removed, nil
	default:
		return n, false, fmt.Errorf("bintrie: unknown node type %T", n)
	}
}

//
// Prefix operations (storage buckets)
//

// DeletePrefix removes every key under the given byte-aligned prefix,
// restoring canonical form. It is the structural storage-bucket drop used by
// account destruction: no per-slot enumeration, one subtree detach.
func (t *BinaryTrie) DeletePrefix(prefix []byte) error {
	if t.committed {
		return trie.ErrCommitted
	}
	root, _, err := t.delPrefix(t.root, prefix, bitstr{}, 0)
	if err != nil {
		return err
	}
	t.root = root
	return nil
}

// HasPrefix reports whether any key exists under the given byte-aligned
// prefix. It is the storage-emptiness probe behind EIP-7610.
func (t *BinaryTrie) HasPrefix(prefix []byte) (bool, error) {
	if t.committed {
		return false, trie.ErrCommitted
	}
	root, found, err := t.findPrefix(t.root, prefix, 0)
	t.root = root
	return found, err
}

// coversPrefix reports whether a branch at position pos, carrying the given
// run, subsumes the whole prefix P: every key below then shares all of P's
// bits. Divergence before the end of P means no key matches.
func coversPrefix(P []byte, pos int, prefix bitstr) (covers bool, diverged bool, matched int) {
	pBits := 8 * len(P)
	m := prefix.matchKey(P, pos)
	if pos+m >= pBits {
		return true, false, m
	}
	if m < prefix.n {
		return false, true, m
	}
	return false, false, m
}

func (t *BinaryTrie) delPrefix(n binaryNode, P []byte, walk bitstr, pos int) (binaryNode, bool, error) {
	switch nn := n.(type) {
	case empty:
		return n, false, nil
	case hashedNode:
		resolved, err := t.resolve(nn, walk)
		if err != nil {
			return n, false, err
		}
		return t.delPrefix(resolved, P, walk, pos)
	case *groupNode:
		if len(nn.stem) >= len(P) && bytes.Equal(nn.stem[:len(P)], P) {
			t.ops.onDelete(pathOf(walk))
			return empty{}, true, nil
		}
		return n, false, nil
	case *branchNode:
		covers, diverged, _ := coversPrefix(P, pos, nn.prefix)
		if diverged {
			return n, false, nil
		}
		if covers {
			// The whole subtree lies inside the bucket.
			if err := t.deleteSubtree(nn, walk, pos); err != nil {
				return n, false, err
			}
			return empty{}, true, nil
		}
		split := pos + nn.prefix.n
		bit := bitAt(P, split)
		childWalk := walk.append(nn.prefix).concat(bit, bitstr{})
		var (
			child   binaryNode
			removed bool
			err     error
		)
		if bit == 0 {
			child, removed, err = t.delPrefix(nn.left, P, childWalk, split+1)
		} else {
			child, removed, err = t.delPrefix(nn.right, P, childWalk, split+1)
		}
		if err != nil {
			return n, false, err
		}
		if _, gone := child.(empty); gone {
			merged, err := t.collapse(nn, walk, bit)
			return merged, removed, err
		}
		if bit == 0 {
			nn.left = child
		} else {
			nn.right = child
		}
		if removed {
			nn.dirty, nn.modified = true, true
		}
		return nn, removed, nil
	default:
		return n, false, fmt.Errorf("bintrie: unknown node type %T", n)
	}
}

func (t *BinaryTrie) findPrefix(n binaryNode, P []byte, pos int) (binaryNode, bool, error) {
	switch nn := n.(type) {
	case empty:
		return n, false, nil
	case hashedNode:
		resolved, err := t.resolve(nn, keyWalk(P, min(pos, 8*len(P))))
		if err != nil {
			return n, false, err
		}
		return t.findPrefix(resolved, P, pos)
	case *groupNode:
		found := len(nn.stem) >= len(P) && bytes.Equal(nn.stem[:len(P)], P)
		return n, found, nil
	case *branchNode:
		covers, diverged, _ := coversPrefix(P, pos, nn.prefix)
		if diverged {
			return n, false, nil
		}
		if covers {
			return n, true, nil // canonical branches are never empty
		}
		split := pos + nn.prefix.n
		bit := bitAt(P, split)
		if bit == 0 {
			child, found, err := t.findPrefix(nn.left, P, split+1)
			nn.left = child
			return n, found, err
		}
		child, found, err := t.findPrefix(nn.right, P, split+1)
		nn.right = child
		return n, found, err
	default:
		return n, false, fmt.Errorf("bintrie: unknown node type %T", n)
	}
}

// deleteSubtree resolves and records the deletion of every stored record in
// the subtree rooted at n (which sits at walk/pos). Resolution funnels every
// record through the witness tracer, so Commit can emit the deletions with
// their previous values.
func (t *BinaryTrie) deleteSubtree(n binaryNode, walk bitstr, pos int) error {
	switch nn := n.(type) {
	case empty:
		return nil
	case hashedNode:
		resolved, err := t.resolve(nn, walk)
		if err != nil {
			return err
		}
		return t.deleteSubtree(resolved, walk, pos)
	case *groupNode:
		t.ops.onDelete(pathOf(walk))
		return nil
	case *branchNode:
		t.ops.onDelete(pathOf(walk))
		split := pos + nn.prefix.n
		if err := t.deleteSubtree(nn.left, walk.append(nn.prefix).concat(0, bitstr{}), split+1); err != nil {
			return err
		}
		return t.deleteSubtree(nn.right, walk.append(nn.prefix).concat(1, bitstr{}), split+1)
	default:
		return fmt.Errorf("bintrie: unknown node type %T", n)
	}
}

//
// State interface (core/state.Trie)
//

// GetKey returns the preimage of a hashed key. Binary tree keys are the
// exposed keys themselves, so this is the identity.
func (t *BinaryTrie) GetKey(key []byte) []byte { return key }

// GetAccount reads the account header stem and decodes the BASIC_DATA and
// CODE_HASH leaves. A missing stem (or missing both leaves) is a missing
// account.
func (t *BinaryTrie) GetAccount(addr common.Address) (*types.StateAccount, error) {
	if t.committed {
		return nil, trie.ErrCommitted
	}
	g, err := t.getStemGroup(HeaderStem(addr))
	if err != nil || g == nil {
		return nil, err
	}
	basic := g.lookup(BasicDataLeafKey)
	codeHash := g.lookup(CodeHashLeafKey)
	if basic == nil && codeHash == nil {
		return nil, nil
	}
	acc := &types.StateAccount{Root: types.EmptyRootHash, CodeHash: types.EmptyCodeHash[:]}
	if basic != nil {
		_, _, nonce, balance := DecodeBasicData(basic)
		acc.Nonce, acc.Balance = nonce, balance
	} else {
		acc.Balance = new(uint256.Int)
	}
	if codeHash != nil {
		acc.CodeHash = append([]byte{}, codeHash...)
	}
	return acc, nil
}

// UpdateAccount writes the BASIC_DATA and CODE_HASH leaves of the account
// header stem in one walk.
func (t *BinaryTrie) UpdateAccount(addr common.Address, acc *types.StateAccount, codeLen int) error {
	basic, err := EncodeBasicData(uint32(codeLen), acc.Nonce, acc.Balance)
	if err != nil {
		return fmt.Errorf("bintrie: UpdateAccount (%x): %w", addr, err)
	}
	codeHash := make([]byte, 32)
	copy(codeHash, acc.CodeHash)
	return t.UpdateStem(HeaderStem(addr),
		[]byte{BasicDataLeafKey, CodeHashLeafKey},
		[][]byte{basic[:], codeHash})
}

// DeleteAccount removes the account's entire header stem: basic data, code
// hash, header storage slots and header code chunks. The account's overflow
// storage bucket is dropped separately via DeleteStorageBucket; the
// content-addressed code zone is never touched.
func (t *BinaryTrie) DeleteAccount(addr common.Address) error {
	return t.removeStem(HeaderStem(addr))
}

// DeleteStorageBucket drops the account's whole overflow storage bucket (all
// keys under StorageZone ‖ KeyHash(addr32)) as one structural detach.
func (t *BinaryTrie) DeleteStorageBucket(addr common.Address) error {
	return t.DeletePrefix(StorageBucketPrefix(addr))
}

// GetStorage returns the value of the given raw storage slot, or nil if
// absent.
func (t *BinaryTrie) GetStorage(addr common.Address, key []byte) ([]byte, error) {
	if t.committed {
		return nil, trie.ErrCommitted
	}
	return t.getValue(StorageSlotKey(addr, key))
}

// UpdateStorage writes a storage slot. The value is left-padded to 32
// bytes; an empty value deletes the slot (zero is encoded as absence at the
// state layer).
func (t *BinaryTrie) UpdateStorage(addr common.Address, key, value []byte) error {
	k := StorageSlotKey(addr, key)
	if len(value) == 0 {
		return t.UpdateStem(k[:len(k)-1], []byte{k[len(k)-1]}, [][]byte{nil})
	}
	var padded [32]byte
	if len(value) > 32 {
		value = value[:32]
	}
	copy(padded[32-len(value):], value)
	return t.UpdateStem(k[:len(k)-1], []byte{k[len(k)-1]}, [][]byte{padded[:]})
}

// DeleteStorage removes a storage slot's leaf.
func (t *BinaryTrie) DeleteStorage(addr common.Address, key []byte) error {
	k := StorageSlotKey(addr, key)
	return t.UpdateStem(k[:len(k)-1], []byte{k[len(k)-1]}, [][]byte{nil})
}

// UpdateContractCode writes the account's code chunks: chunks 0..127 into
// the header stem (clearing any leftover header chunks beyond the new code,
// which shrinks under EIP-7702 delegation clears), chunks 128 and above into
// the content-addressed code zone shared across contracts with identical
// bytecode. The code zone is append-only.
func (t *BinaryTrie) UpdateContractCode(addr common.Address, codeHash common.Hash, code []byte) error {
	chunks := ChunkifyCode(code)
	numChunks := len(chunks) / 32

	// Header chunks 0..127, plus explicit clears through sub-index 255 so
	// stale chunks of longer previous code disappear.
	headerSubs := make([]byte, 0, StemSubtreeWidth-CodeOffset)
	headerVals := make([][]byte, 0, StemSubtreeWidth-CodeOffset)
	for i := 0; i < StemSubtreeWidth-CodeOffset; i++ {
		headerSubs = append(headerSubs, byte(CodeOffset+i))
		if i < numChunks {
			headerVals = append(headerVals, chunks[32*i:32*(i+1)])
		} else {
			headerVals = append(headerVals, nil)
		}
	}
	if err := t.UpdateStem(HeaderStem(addr), headerSubs, headerVals); err != nil {
		return err
	}

	// Overflow chunks, grouped per content-addressed stem.
	var (
		subs []byte
		vals [][]byte
		tree uint64
	)
	flush := func() error {
		if len(subs) == 0 {
			return nil
		}
		err := t.UpdateStem(CodeChunkStem(codeHash, tree), subs, vals)
		subs, vals = nil, nil
		return err
	}
	for chunk := StemSubtreeWidth - CodeOffset; chunk < numChunks; chunk++ {
		_, treeIndex, sub := CodeChunkIndex(uint64(chunk))
		if treeIndex != tree {
			if err := flush(); err != nil {
				return err
			}
			tree = treeIndex
		}
		subs = append(subs, sub)
		vals = append(vals, chunks[32*chunk:32*(chunk+1)])
	}
	return flush()
}

// Hash returns the root hash of the trie. It does not write to the database.
func (t *BinaryTrie) Hash() common.Hash {
	return t.root.hashAt(0)
}

// Witness returns every node blob resolved from the database since the trie
// was opened, keyed by path.
func (t *BinaryTrie) Witness() map[string][]byte {
	return t.tracer.Values()
}

// Copy creates a deep copy of the trie, including its mutation and witness
// tracers (a copy that dropped them would emit a wrong node set at commit).
func (t *BinaryTrie) Copy() *BinaryTrie {
	return &BinaryTrie{
		root:      t.root.copy(),
		reader:    t.reader,
		tracer:    t.tracer.Copy(),
		ops:       t.ops.copy(),
		committed: t.committed,
	}
}

// IsPBT reports the trie flavor for the state-layer dispatch.
func (t *BinaryTrie) IsPBT() bool { return true }

// PrefetchAccount warms the trie with the given accounts' header stems.
func (t *BinaryTrie) PrefetchAccount(addrs []common.Address) error {
	for _, addr := range addrs {
		if _, err := t.getStemGroup(HeaderStem(addr)); err != nil {
			return err
		}
	}
	return nil
}

// PrefetchStorage warms the trie with the given storage slots.
func (t *BinaryTrie) PrefetchStorage(addr common.Address, keys [][]byte) error {
	for _, key := range keys {
		k := StorageSlotKey(addr, key)
		if _, err := t.getStemGroup(k[:len(k)-1]); err != nil {
			return err
		}
	}
	return nil
}

// Prove constructs a merkle proof for key: the preimages of every canonical
// node from the root towards the key, ending at the leaf (inclusion) or the
// divergence witness (exclusion). Implemented in proof.go.
func (t *BinaryTrie) Prove(key []byte, proofDb ethdb.KeyValueWriter) error {
	return t.prove(key, proofDb)
}
