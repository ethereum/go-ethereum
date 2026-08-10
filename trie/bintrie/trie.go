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
	"github.com/ethereum/go-ethereum/crypto"
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
			// matchKey stops at the end of the stem, so a prefix running past
			// that end is indistinguishable here from one that conflicts.
			// Agreement all the way to the boundary means this branch belongs
			// to an expanded stem rather than to a different one, and the
			// group this walk exists to find does not exist as a single node.
			if pos+m >= 8*len(stem) {
				return n, nil, ErrPartialStem
			}
			return n, nil, nil // diverges inside the prefix: absent
		}
		split := pos + nn.prefix.n
		if split >= 8*len(stem) {
			// The prefix consumed the stem exactly and the tree continues
			// below, which no whole-group tree produces: two distinct stems of
			// one zone differ before their end, so a branch above a group
			// always stops short of it. This is the expanded form again, with
			// the two leaves diverging on the first sub-index bit.
			return n, nil, ErrPartialStem
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
	if t.committed {
		return nil, trie.ErrCommitted
	}
	n, v, err := t.getKey(t.root, key, 0)
	if err != nil {
		return nil, err
	}
	t.root = n
	return v, nil
}

// getKey walks to a single value by its whole key rather than by its stem.
//
// The distinction matters only where a stem is expanded into branches and
// leaves instead of held as one group, which is what a tree built from a
// proof looks like. getStem cannot cross that boundary: matchKey stops at the
// end of the stem, so the branch whose prefix carries the stem's remaining
// bits plus a few sub-index bits never matches, and the walk reports absence
// for a value that is present. Walking the whole key gives matchKey the sub
// bits it needs, and the branch arithmetic below is otherwise identical.
//
// For a tree holding whole groups the two walks agree: every branch above a
// group has a prefix ending before the stem does, so the extra key bits are
// never consulted.
func (t *BinaryTrie) getKey(n binaryNode, key []byte, pos int) (binaryNode, []byte, error) {
	switch nn := n.(type) {
	case empty:
		return n, nil, nil
	case hashedNode:
		resolved, err := t.resolve(nn, keyWalk(key, pos))
		if err != nil {
			return n, nil, err
		}
		return t.getKey(resolved, key, pos)
	case *groupNode:
		if !bytes.Equal(nn.stem, key[:len(key)-1]) {
			return n, nil, nil
		}
		return n, nn.lookup(key[len(key)-1]), nil
	case *branchNode:
		if m := nn.prefix.matchKey(key, pos); m < nn.prefix.n {
			return n, nil, nil // diverges inside the prefix: absent
		}
		split := pos + nn.prefix.n
		if split >= 8*len(key) {
			return n, nil, nil // no bit left to branch on
		}
		if bitAt(key, split) == 0 {
			child, v, err := t.getKey(nn.left, key, split+1)
			if child != nn.left {
				nn.left = child // only on resolution: a write barrier per level otherwise
			}
			return n, v, err
		}
		child, v, err := t.getKey(nn.right, key, split+1)
		if child != nn.right {
			nn.right = child
		}
		return n, v, err
	default:
		return n, nil, fmt.Errorf("bintrie: unknown node type %T", n)
	}
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

// zeroValue is the 32-byte value the state model resolves to absence.
var zeroValue [32]byte

// isZeroValue reports whether v is the all-zero leaf value.
func isZeroValue(v []byte) bool {
	return len(v) == len(zeroValue) && bytes.Equal(v, zeroValue[:])
}

// stateWrite writes values at subs of stem, resolving an all-zero value to a
// deletion rather than an insertion.
//
// EIP-8297 assigns this to the state transition function, not the tree: "no
// key in the state's tree holds 32 zero bytes", so zero and absent are one
// state committing to one root. Only the typed writers are that function;
// UpdateStem stays raw and can still hold a deliberately-zero leaf, as the
// zero_value_present vector pins.
//
// values is rewritten in place. Every caller builds it locally.
func (t *BinaryTrie) stateWrite(stem []byte, subs []byte, values [][]byte) error {
	for i, v := range values {
		if isZeroValue(v) {
			values[i] = nil
		}
	}
	return t.UpdateStem(stem, subs, values)
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
		// Running out of stem bits here means the incoming stem is a prefix of
		// the resident one, which conformant keys cannot be. getStem guards the
		// same indexing; without it bitAt below reads past the stem.
		if split >= 8*len(stem) {
			return nil, ErrNonConformantKey
		}
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
			if split >= 8*len(stem) {
				// As in getStem: a branch consuming the stem exactly, with the
				// tree continuing below, is an expanded stem rather than a
				// malformed key. Writing into it would have to rebuild a fold
				// this walk cannot see all of.
				return n, ErrPartialStem
			}
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
		// matchKey stops at the end of the stem, so agreement all the way to
		// that boundary is not a divergence: the branch belongs to an
		// expanded stem, and the group this insert would merge into is spread
		// across nodes below. Refuse before acting on it. Splitting here would
		// index past the stem in bitAt below, and the pure-deletion path above
		// would return the tree unchanged - a wrong root reported as success.
		if pos+m >= 8*len(stem) {
			return n, ErrPartialStem
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
		// Both boundary arms mirror getStem: agreement past the stem's end means
		// an expanded stem, and "nothing removed" there would leave it in place.
		m := nn.prefix.matchKey(stem, pos)
		if m < nn.prefix.n {
			if pos+m >= 8*len(stem) {
				return n, false, ErrPartialStem
			}
			return n, false, nil
		}
		split := pos + nn.prefix.n
		if split >= 8*len(stem) {
			return n, false, ErrPartialStem
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

// GetAccount reads the account header stem and decodes the BASIC_DATA leaf
// alongside whichever of CODE_HASH and DELEGATION it holds. A missing stem, or
// one holding none of the three, is a missing account.
//
// A delegated account carries no code hash to read, so this derives one by
// hashing the indicator. Everything above the tree - EIP-161 emptiness, the
// txpools' delegation probe, EXTCODEHASH - reads it through
// types.StateAccount.CodeHash, so the synthesis keeps those callers correct
// without any of them knowing the leaf exists.
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
	delegation := g.lookup(DelegationLeafKey)
	if basic == nil && codeHash == nil && delegation == nil {
		return nil, nil
	}
	acc := &types.StateAccount{Root: types.EmptyRootHash, CodeHash: types.EmptyCodeHash[:]}
	var codeSize uint32
	if basic != nil {
		var (
			nonce   uint64
			balance *uint256.Int
		)
		_, codeSize, nonce, balance = DecodeBasicData(basic)
		acc.Nonce, acc.Balance = nonce, balance
	} else {
		acc.Balance = new(uint256.Int)
	}
	switch {
	case delegation != nil:
		// The code is the leading code_size bytes of the leaf, so the hash is
		// over those. Reading the whole padded value instead would hash nine
		// bytes of padding into it and disagree with EXTCODEHASH.
		//
		// A zero size is refused rather than hashed: it would produce the
		// empty-code hash, and the account would read back as codeless and
		// EIP-161-empty while holding a delegation - the one wrong answer this
		// synthesis could give that no caller could detect.
		if codeSize == 0 {
			return nil, fmt.Errorf("bintrie: GetAccount (%x): delegation leaf with a zero code size", addr)
		}
		if int(codeSize) > len(delegation) {
			return nil, fmt.Errorf("bintrie: GetAccount (%x): code size %d exceeds the %d-byte delegation leaf", addr, codeSize, len(delegation))
		}
		acc.CodeHash = crypto.Keccak256(delegation[:codeSize])
	case codeHash != nil:
		acc.CodeHash = append([]byte{}, codeHash...)
	}
	return acc, nil
}

// UpdateAccountBatch writes a list of accounts one at a time. See
// UpdateStorageBatch for why this is a loop rather than a batch.
func (t *BinaryTrie) UpdateAccountBatch(addrs []common.Address, accounts []*types.StateAccount, codeLens []int, delegations [][]byte) error {
	if len(addrs) != len(accounts) {
		return fmt.Errorf("addresses and accounts length mismatch: %d != %d", len(addrs), len(accounts))
	}
	if len(addrs) != len(codeLens) {
		return fmt.Errorf("addresses and code lengths mismatch: %d != %d", len(addrs), len(codeLens))
	}
	if len(addrs) != len(delegations) {
		return fmt.Errorf("addresses and delegations length mismatch: %d != %d", len(addrs), len(delegations))
	}
	for i, addr := range addrs {
		if err := t.UpdateAccount(addr, accounts[i], codeLens[i], delegations[i]); err != nil {
			return err
		}
	}
	return nil
}

// UpdateAccount writes the BASIC_DATA leaf of the account header stem, and
// whichever of CODE_HASH and DELEGATION the account holds, in one walk.
//
// A non-nil delegation is the EIP-7702 designator: the indicator becomes the
// account's code, so the size is 23 and the code-hash leaf is removed. A nil
// delegation with a stated size removes the delegation leaf instead. Both move
// in the single walk below, so their exclusivity is never momentarily broken.
//
// The designator is passed rather than recognised from the code, because
// reading the blob back to test for the marker is the unwitnessed read codeLen
// exists to avoid. Callers hold it already whenever they set it.
//
// A negative codeLen means the caller declines to state the size, not that it
// necessarily lacks one: the account-write path passes it for every account
// whose code was not set this block, whether or not the code happens to be
// resident, because asking can fault a whole contract in off the reader
// unwitnessed while the tree already holds the answer. The size is then taken
// from the stem about to be written. The group read here is the one insStem
// walks to below, so it is resolved either way.
//
// The size is preserved only for an account that still carries code. Both
// leaves are written here, so the size must agree with the code hash going
// out, and an empty hash means zero however the stem got that way. Without
// that the resident value outlives what it measured: a destroy and a recreate
// of one address coalesce into a single update mutation, so nothing deletes
// the old stem and the dead contract's size would be inherited by the account
// that replaced it.
func (t *BinaryTrie) UpdateAccount(addr common.Address, acc *types.StateAccount, codeLen int, delegation []byte) error {
	if len(acc.CodeHash) != 32 {
		return fmt.Errorf("bintrie: UpdateAccount (%x): code hash is %d bytes, want 32", addr, len(acc.CodeHash))
	}
	stem := HeaderStem(addr)
	if delegation != nil {
		// Checked rather than trusted: a non-indicator stored here would be
		// hashed into a code hash naming bytecode nothing can produce.
		if _, ok := types.ParseDelegation(delegation); !ok {
			return fmt.Errorf("bintrie: UpdateAccount (%x): the %d-byte delegation is not an indicator", addr, len(delegation))
		}
		// The leaf becomes the account's code, so the hash GetAccount reads
		// back is the indicator's, not the one passed here. Callers derive
		// both from one blob; checking says so rather than letting a
		// disagreement surface as an untraceable wrong code hash.
		if got := crypto.Keccak256(delegation); !bytes.Equal(got, acc.CodeHash) {
			return fmt.Errorf("bintrie: UpdateAccount (%x): the delegation hashes to %x but the account carries %x", addr, got, acc.CodeHash)
		}
		basic, err := EncodeBasicData(uint32(len(delegation)), acc.Nonce, acc.Balance)
		if err != nil {
			return fmt.Errorf("bintrie: UpdateAccount (%x): %w", addr, err)
		}
		return t.stateWrite(stem,
			[]byte{BasicDataLeafKey, CodeHashLeafKey, DelegationLeafKey},
			[][]byte{basic[:], nil, EncodeDelegation(delegation)})
	}
	codeSize := uint32(codeLen)
	// Set when the account is delegated and the caller stated no size, the one
	// case where the code leaves must be left exactly as they are: rewriting
	// them from acc would clear the indicator and put back a code-hash leaf,
	// silently un-delegating an account that was only touched for its balance.
	keepCodeLeaves := false
	if codeLen < 0 {
		codeSize = 0
		if !bytes.Equal(acc.CodeHash, types.EmptyCodeHash[:]) {
			// Nothing to preserve from is a broken invariant rather than a
			// small contract: the account says it has code, so its basic data
			// has to be there to say how much. Writing zero would put a size
			// of zero beside a non-empty code hash and commit it as a root.
			g, err := t.getStemGroup(stem)
			if err != nil {
				return fmt.Errorf("bintrie: UpdateAccount (%x): %w", addr, err)
			}
			if g == nil {
				return fmt.Errorf("bintrie: UpdateAccount (%x): code hash %x with no header stem to preserve a size from", addr, acc.CodeHash)
			}
			basic := g.lookup(BasicDataLeafKey)
			if basic == nil {
				return fmt.Errorf("bintrie: UpdateAccount (%x): code hash %x with no basic data to preserve a size from", addr, acc.CodeHash)
			}
			_, codeSize, _, _ = DecodeBasicData(basic)
			// The resident size measures the resident code, so it may only be
			// carried forward under the same hash. A caller changing the hash
			// while declining to state a length is asking for a size that
			// measures a different contract.
			if resident := g.lookup(DelegationLeafKey); resident != nil {
				// A delegated account holds no code-hash leaf to compare, so
				// the same question is put to the hash its indicator implies.
				if int(codeSize) > len(resident) {
					return fmt.Errorf("bintrie: UpdateAccount (%x): code size %d exceeds the %d-byte delegation leaf", addr, codeSize, len(resident))
				}
				if got := crypto.Keccak256(resident[:codeSize]); !bytes.Equal(got, acc.CodeHash) {
					return fmt.Errorf("bintrie: UpdateAccount (%x): no code length given and the code hash is changing, %x to %x", addr, got, acc.CodeHash)
				}
				keepCodeLeaves = true
			} else if resident := g.lookup(CodeHashLeafKey); !bytes.Equal(resident, acc.CodeHash) {
				return fmt.Errorf("bintrie: UpdateAccount (%x): no code length given and the code hash is changing, %x to %x", addr, resident, acc.CodeHash)
			}
		}
	}
	// No bytecode of non-zero length hashes to the empty hash, so this pair is
	// contradictory whichever way it was reached - including a caller that
	// reported a length it did not actually have.
	if codeSize == 0 && !bytes.Equal(acc.CodeHash, types.EmptyCodeHash[:]) {
		return fmt.Errorf("bintrie: UpdateAccount (%x): zero code size for code hash %x", addr, acc.CodeHash)
	}
	basic, err := EncodeBasicData(codeSize, acc.Nonce, acc.Balance)
	if err != nil {
		return fmt.Errorf("bintrie: UpdateAccount (%x): %w", addr, err)
	}
	if keepCodeLeaves {
		return t.stateWrite(stem, []byte{BasicDataLeafKey}, [][]byte{basic[:]})
	}
	codeHash := make([]byte, 32)
	copy(codeHash, acc.CodeHash)
	return t.stateWrite(stem,
		[]byte{BasicDataLeafKey, CodeHashLeafKey, DelegationLeafKey},
		[][]byte{basic[:], codeHash, nil})
}

// DeleteAccount removes everything the account owns in the tree: its whole
// header stem (basic data, code hash and header storage slots) and its
// overflow storage bucket. Dropping the bucket is required, not optional: in
// the merkle-patricia world a destroyed account's storage trie merely becomes
// unreachable from the state root, so a conversion of that state contains none
// of it, and leaving those leaves behind here would diverge. The
// content-addressed code zone is never touched, since its chunks may be shared
// with living contracts. See TODO.md.
func (t *BinaryTrie) DeleteAccount(addr common.Address) error {
	if err := t.removeStem(HeaderStem(addr)); err != nil {
		return err
	}
	return t.DeletePrefix(StorageBucketPrefix(addr))
}

// DeleteStorageBucket drops the account's whole overflow storage bucket (all
// keys under StorageZone ‖ KeyHash(addr32)) as one structural detach.
func (t *BinaryTrie) DeleteStorageBucket(addr common.Address) error {
	return t.DeletePrefix(StorageBucketPrefix(addr))
}

// GetStemValue returns the raw 32-byte value stored at a tree key, or nil
// if absent. It is the low-level accessor behind the typed readers, useful
// for tooling and tests that inspect leaves directly.
func (t *BinaryTrie) GetStemValue(key []byte) ([]byte, error) {
	return t.getValue(key)
}

// HasHeaderStorage reports whether the account's header stem holds any of
// the storage slots that live there (slots 0..63, at sub-indices
// HeaderStorageOffset..HeaderStorageOffset+HeaderStorageSlots-1).
func (t *BinaryTrie) HasHeaderStorage(addr common.Address) (bool, error) {
	g, err := t.getStemGroup(HeaderStem(addr))
	if err != nil || g == nil {
		return false, err
	}
	for _, sub := range g.subs {
		if sub >= HeaderStorageOffset && sub < HeaderStorageOffset+HeaderStorageSlots {
			return true, nil
		}
	}
	return false, nil
}

// GetStorage returns the value of the given raw storage slot, or nil if
// absent.
func (t *BinaryTrie) GetStorage(addr common.Address, key []byte) ([]byte, error) {
	if t.committed {
		return nil, trie.ErrCommitted
	}
	return t.getValue(StorageSlotKey(addr, key))
}

// UpdateStorageBatch writes a list of storage slots one at a time.
//
// The interface asks for a batch but documents no atomicity or ordering
// contract, and nothing calls it yet. UpdateStem is where genuine batching
// belongs when a caller appears: slots sharing a stem could be written in one
// walk rather than one walk each.
func (t *BinaryTrie) UpdateStorageBatch(addr common.Address, keys [][]byte, values [][]byte) error {
	if len(keys) != len(values) {
		return fmt.Errorf("keys and values length mismatch: %d != %d", len(keys), len(values))
	}
	for i, key := range keys {
		if err := t.UpdateStorage(addr, key, values[i]); err != nil {
			return err
		}
	}
	return nil
}

// UpdateStorage writes a storage slot. The value is left-padded to 32 bytes;
// a zero value deletes the slot, whether it arrives empty or as 32 zero
// bytes, because zero is absence at the state layer.
func (t *BinaryTrie) UpdateStorage(addr common.Address, key, value []byte) error {
	k := StorageSlotKey(addr, key)
	var padded [32]byte
	if len(value) > 32 {
		value = value[:32]
	}
	copy(padded[32-len(value):], value)
	return t.stateWrite(k[:len(k)-1], []byte{k[len(k)-1]}, [][]byte{padded[:]})
}

// DeleteStorage removes a storage slot's leaf.
func (t *BinaryTrie) DeleteStorage(addr common.Address, key []byte) error {
	k := StorageSlotKey(addr, key)
	return t.UpdateStem(k[:len(k)-1], []byte{k[len(k)-1]}, [][]byte{nil})
}

// UpdateContractCode writes the account's code chunks into the
// content-addressed code zone, grouped StemSubtreeWidth to a stem. The address
// takes no part and stays only because the Trie interface carries it.
//
// Shorter code does not clear what longer code left behind: those leaves are
// keyed by the old hash, which this call does not have, and they may be
// shared. See TODO.md.
//
// Nothing reference-counts the shared chunks and nothing needs to. Reorgs drop
// layers rather than reverse them, so chunks written by a reverted block go
// away with the layer, and chunks already persisted sit at or below the disk
// layer, which the tree refuses to fork beneath. Whether some other account
// still holds this bytecode is answered by the ancestor state, not by a count.
// core.TestPBTReorgKeepsSharedCodeChunks pins it.
func (t *BinaryTrie) UpdateContractCode(_ common.Address, codeHash common.Hash, code []byte) error {
	// A delegation indicator is not code here: it lives in its own account's
	// header, written by UpdateAccount, precisely so it is not shared.
	// Chunking it would leave a shared leaf nothing ever removes.
	if _, ok := types.ParseDelegation(code); ok {
		return nil
	}
	chunks := ChunkifyCode(code)
	numChunks := len(chunks) / 32

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
	for chunk := 0; chunk < numChunks; chunk++ {
		treeIndex, sub := CodeChunkIndex(uint64(chunk))
		if treeIndex != tree {
			if err := flush(); err != nil {
				return err
			}
			tree = treeIndex
		}
		v := chunks[32*chunk : 32*(chunk+1)]
		// A run of 31 zero code bytes encodes to 32 zero bytes, which resolves
		// to absence; reads recover the zero it stood for, and code_size
		// delimits the code rather than chunk presence. Skipped rather than
		// deleted, which is the same thing here and avoids an empty batch:
		// chunks are content-addressed, so no stale value can sit at this key.
		if isZeroValue(v) {
			continue
		}
		subs = append(subs, sub)
		vals = append(vals, v)
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
