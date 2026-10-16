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

// Package pbt implements the EIP-8297 partitioned binary tree.
package pbt

import (
	"bytes"
	"errors"
	"fmt"
	"maps"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/bintrie"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/triedb/database"
	"github.com/holiman/uint256"
)

var (
	errInvalidKey    = errors.New("pbt: invalid key")
	errInvalidRecord = errors.New("pbt: invalid node record")
	errUnsupported   = errors.New("pbt: not supported")
)

// Trie is a partitioned binary tree over the whole state. It is not safe for
// concurrent use.
type Trie struct {
	root      node
	reader    *trie.Reader
	tracer    *trie.PrevalueTracer
	inserts   map[string]struct{} // record paths created since load
	deletes   map[string]struct{} // record paths removed since load
	committed bool
}

// New opens the tree with the given root.
func New(root common.Hash, db database.NodeDatabase) (*Trie, error) {
	reader, err := trie.NewReader(root, common.Hash{}, db)
	if err != nil {
		return nil, err
	}
	t := &Trie{
		root:    emptyNode{},
		reader:  reader,
		tracer:  trie.NewPrevalueTracer(),
		inserts: make(map[string]struct{}),
		deletes: make(map[string]struct{}),
	}
	if root != types.EmptyBinaryHash && root != types.EmptyRootHash {
		if t.root, err = t.resolve(hashedNode(root), bitstr{}); err != nil {
			return nil, err
		}
	}
	return t, nil
}

// pathOf is the database path of the node at walk.
func pathOf(walk bitstr) []byte {
	if walk.n == 0 {
		return nil
	}
	return appendBitPrefix(nil, walk)
}

func (t *Trie) resolve(h hashedNode, walk bitstr) (node, error) {
	path := pathOf(walk)
	blob, err := t.reader.Node(path, common.Hash(h))
	if err != nil {
		return nil, err
	}
	t.tracer.Put(path, blob)
	return decodeNode(blob, common.Hash(h))
}

func (t *Trie) onInsert(walk bitstr) {
	p := string(pathOf(walk))
	if _, ok := t.deletes[p]; ok {
		delete(t.deletes, p)
	} else {
		t.inserts[p] = struct{}{}
	}
}

func (t *Trie) onDelete(walk bitstr) {
	p := string(pathOf(walk))
	if _, ok := t.inserts[p]; ok {
		delete(t.inserts, p)
	} else {
		t.deletes[p] = struct{}{}
	}
}

// getStem returns the group of stem, or nil if absent.
func (t *Trie) getStem(stem []byte) (*groupNode, error) {
	if t.committed {
		return nil, trie.ErrCommitted
	}
	root, g, err := t.findStem(t.root, stem, 0)
	t.root = root
	return g, err
}

// findStem walks n, sitting at bit position pos, to the group of stem, or nil
// if absent. It returns n with the nodes resolved on the way swapped in.
func (t *Trie) findStem(n node, stem []byte, pos int) (node, *groupNode, error) {
	switch n := n.(type) {
	case hashedNode:
		resolved, err := t.resolve(n, slice(stem, 0, pos))
		if err != nil {
			return n, nil, err
		}
		return t.findStem(resolved, stem, pos)
	case *groupNode:
		if bytes.Equal(n.stem, stem) {
			return n, n, nil
		}
	case *branchNode:
		if n.prefix.matchKey(stem, pos) < n.prefix.n {
			return n, nil, nil
		}
		split := pos + n.prefix.n
		child := &n.left
		if bitAt(stem, split) == 1 {
			child = &n.right
		}
		resolved, g, err := t.findStem(*child, stem, split+1)
		*child = resolved
		return n, g, err
	}
	return n, nil, nil
}

// stateWrite writes values at subs of stem, a zero value deleting the key.
func (t *Trie) stateWrite(stem []byte, subs []byte, vals [][]byte) error {
	for i, v := range vals {
		if v != nil && bytes.Equal(v, zero[:]) {
			vals[i] = nil
		}
	}
	return t.updateStem(stem, subs, vals)
}

var zero [32]byte

// updateStem writes values at subs of stem in one walk; nil deletes.
func (t *Trie) updateStem(stem []byte, subs []byte, vals [][]byte) error {
	if t.committed {
		return trie.ErrCommitted
	}
	if err := validateStem(stem); err != nil {
		return err
	}
	root, err := t.insert(t.root, stem, subs, vals, 0)
	if err != nil {
		return err
	}
	t.root = root
	return nil
}

func newGroup(stem []byte, subs []byte, vals [][]byte) *groupNode {
	g := &groupNode{stem: common.CopyBytes(stem)}
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

// emptiedBy reports whether the writes delete every value of g.
func emptiedBy(g *groupNode, subs []byte, vals [][]byte) bool {
	for _, v := range vals {
		if v != nil {
			return false
		}
	}
	for _, sub := range g.subs {
		if bytes.IndexByte(subs, sub) < 0 {
			return false
		}
	}
	return true
}

// insert applies the writes to stem below n, sitting at bit position pos, and
// returns what replaces n. A new stem splits the group or branch it diverges
// from; a group emptied by deletes collapses its parent branch.
func (t *Trie) insert(n node, stem []byte, subs []byte, vals [][]byte, pos int) (node, error) {
	switch n := n.(type) {
	case emptyNode:
		g := newGroup(stem, subs, vals)
		if g == nil {
			return n, nil
		}
		t.onInsert(slice(stem, 0, pos))
		return g, nil

	case hashedNode:
		resolved, err := t.resolve(n, slice(stem, 0, pos))
		if err != nil {
			return n, err
		}
		return t.insert(resolved, stem, subs, vals, pos)

	case *groupNode:
		if bytes.Equal(n.stem, stem) {
			if emptiedBy(n, subs, vals) {
				t.onDelete(slice(stem, 0, pos))
				return emptyNode{}, nil
			}
			for i, v := range vals {
				n.set(subs[i], v)
			}
			return n, nil
		}
		g := newGroup(stem, subs, vals)
		if g == nil {
			return n, nil
		}
		run := commonPrefixLen(stem, n.stem, pos)
		split := pos + run
		t.onDelete(slice(n.stem, 0, pos))
		t.onInsert(slice(n.stem, 0, split+1))
		t.onInsert(slice(stem, 0, split+1))
		t.onInsert(slice(stem, 0, pos))
		n.dirty, n.modified = true, true
		branch := &branchNode{prefix: slice(stem, pos, run), left: g, right: n, dirty: true, modified: true}
		if bitAt(stem, split) == 1 {
			branch.left, branch.right = n, g
		}
		return branch, nil

	case *branchNode:
		m := n.prefix.matchKey(stem, pos)
		if m == n.prefix.n {
			split := pos + n.prefix.n
			bit := bitAt(stem, split)
			child := &n.left
			if bit == 1 {
				child = &n.right
			}
			updated, err := t.insert(*child, stem, subs, vals, split+1)
			if err != nil {
				return n, err
			}
			if _, ok := updated.(emptyNode); ok {
				return t.collapse(n, slice(stem, 0, pos), bit)
			}
			*child = updated
			n.dirty, n.modified = true, true
			return n, nil
		}
		// The stem diverges inside the prefix: split the branch.
		g := newGroup(stem, subs, vals)
		if g == nil {
			return n, nil
		}
		split := pos + m
		t.onInsert(slice(stem, 0, split).concat(n.prefix.bit(m), bitstr{}))
		t.onInsert(slice(stem, 0, split+1))
		survivor := &branchNode{prefix: n.prefix.tail(m + 1), left: n.left, right: n.right, dirty: true, modified: true}
		top := &branchNode{prefix: n.prefix.head(m), left: g, right: survivor, dirty: true, modified: true}
		if bitAt(stem, split) == 1 {
			top.left, top.right = survivor, g
		}
		return top, nil
	}
	return n, fmt.Errorf("pbt: unexpected node type %T", n)
}

// collapse replaces branch n at walk, whose child on side bit was emptied, by
// its other child.
func (t *Trie) collapse(n *branchNode, walk bitstr, bit byte) (node, error) {
	sibling, sbit := n.right, byte(1)
	if bit == 1 {
		sibling, sbit = n.left, 0
	}
	swalk := walk.append(n.prefix).concat(sbit, bitstr{})
	if h, ok := sibling.(hashedNode); ok {
		resolved, err := t.resolve(h, swalk)
		if err != nil {
			return n, err
		}
		sibling = resolved
	}
	t.onDelete(swalk)
	t.onInsert(walk)
	switch s := sibling.(type) {
	case *branchNode:
		s.prefix = n.prefix.concat(sbit, s.prefix)
		s.dirty, s.modified = true, true
		return s, nil
	case *groupNode:
		s.dirty, s.modified = true, true
		return s, nil
	}
	return n, fmt.Errorf("pbt: unexpected collapse survivor %T", sibling)
}

// deletePrefix removes every key starting with the byte string prefix.
func (t *Trie) deletePrefix(prefix []byte) error {
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

// delPrefix removes the keys below n, sitting at walk, that start with prefix,
// and returns what replaces n and whether anything was removed.
func (t *Trie) delPrefix(n node, prefix []byte, walk bitstr, pos int) (node, bool, error) {
	switch n := n.(type) {
	case hashedNode:
		resolved, err := t.resolve(n, walk)
		if err != nil {
			return n, false, err
		}
		return t.delPrefix(resolved, prefix, walk, pos)
	case *groupNode:
		if bytes.HasPrefix(n.stem, prefix) {
			t.onDelete(walk)
			return emptyNode{}, true, nil
		}
	case *branchNode:
		m := n.prefix.matchKey(prefix, pos)
		if pos+m >= 8*len(prefix) {
			// Every key below starts with prefix.
			return emptyNode{}, true, t.deleteSubtree(n, walk, pos)
		}
		if m < n.prefix.n {
			return n, false, nil
		}
		split := pos + n.prefix.n
		bit := bitAt(prefix, split)
		child := &n.left
		if bit == 1 {
			child = &n.right
		}
		updated, removed, err := t.delPrefix(*child, prefix, walk.append(n.prefix).concat(bit, bitstr{}), split+1)
		if err != nil {
			return n, false, err
		}
		if _, ok := updated.(emptyNode); ok {
			collapsed, err := t.collapse(n, walk, bit)
			return collapsed, true, err
		}
		*child = updated
		if removed {
			n.dirty, n.modified = true, true
		}
		return n, removed, nil
	}
	return n, false, nil
}

// deleteSubtree records the removal of every stored record below n.
func (t *Trie) deleteSubtree(n node, walk bitstr, pos int) error {
	switch n := n.(type) {
	case hashedNode:
		resolved, err := t.resolve(n, walk)
		if err != nil {
			return err
		}
		return t.deleteSubtree(resolved, walk, pos)
	case *groupNode:
		t.onDelete(walk)
	case *branchNode:
		t.onDelete(walk)
		split := pos + n.prefix.n
		if err := t.deleteSubtree(n.left, walk.append(n.prefix).concat(0, bitstr{}), split+1); err != nil {
			return err
		}
		return t.deleteSubtree(n.right, walk.append(n.prefix).concat(1, bitstr{}), split+1)
	}
	return nil
}

// GetKey returns the key itself: tree keys are not hashed preimages.
func (t *Trie) GetKey(key []byte) []byte { return key }

// GetAccount reads the account header. A delegated account's code hash is the
// hash of its indicator.
func (t *Trie) GetAccount(addr common.Address) (*types.StateAccount, error) {
	g, err := t.getStem(headerStem(addr))
	if err != nil || g == nil {
		return nil, err
	}
	basic, codeHash, delegation := g.lookup(basicDataLeafKey), g.lookup(codeHashLeafKey), g.lookup(delegationLeafKey)
	if basic == nil && codeHash == nil && delegation == nil {
		return nil, nil
	}
	acc := &types.StateAccount{Balance: new(uint256.Int), Root: types.EmptyRootHash, CodeHash: types.EmptyCodeHash[:]}
	var size uint32
	if basic != nil {
		size, acc.Nonce, acc.Balance = decodeBasicData(basic)
	}
	switch {
	case delegation != nil:
		if size == 0 || size > 32 {
			return nil, fmt.Errorf("pbt: account %x delegation with code size %d", addr, size)
		}
		acc.CodeHash = crypto.Keccak256(delegation[:size])
	case codeHash != nil:
		acc.CodeHash = common.CopyBytes(codeHash)
	}
	return acc, nil
}

// UpdateAccount writes the account's basic data. The code leaves are written
// by UpdateContractCode when the code changes; a zero codeLen with code means
// the code was not loaded, so the stored size is kept.
func (t *Trie) UpdateAccount(addr common.Address, acc *types.StateAccount, codeLen int) error {
	var (
		stem = headerStem(addr)
		size = uint32(codeLen)
		subs = []byte{basicDataLeafKey}
		vals = [][]byte{nil}
	)
	switch {
	case bytes.Equal(acc.CodeHash, types.EmptyCodeHash[:]):
		size = 0
		subs = append(subs, codeHashLeafKey, delegationLeafKey)
		vals = append(vals, types.EmptyCodeHash.Bytes(), nil)
	case codeLen == 0:
		g, err := t.getStem(stem)
		if err != nil {
			return err
		}
		if g == nil || g.lookup(basicDataLeafKey) == nil {
			return fmt.Errorf("pbt: account %x has code but no code size", addr)
		}
		size, _, _ = decodeBasicData(g.lookup(basicDataLeafKey))
	}
	basic, err := encodeBasicData(size, acc.Nonce, acc.Balance)
	if err != nil {
		return err
	}
	vals[0] = basic
	return t.stateWrite(stem, subs, vals)
}

// UpdateAccountBatch writes the accounts one by one.
func (t *Trie) UpdateAccountBatch(addrs []common.Address, accounts []*types.StateAccount, codeLens []int) error {
	for i, addr := range addrs {
		if err := t.UpdateAccount(addr, accounts[i], codeLens[i]); err != nil {
			return err
		}
	}
	return nil
}

// DeleteAccount removes the account header and storage. Code is shared by
// hash and stays.
func (t *Trie) DeleteAccount(addr common.Address) error {
	if err := t.deletePrefix(headerStem(addr)); err != nil {
		return err
	}
	return t.deletePrefix(zoneStem(storageZone, addr))
}

// GetStorage returns the value of a raw storage slot, or nil if absent.
func (t *Trie) GetStorage(addr common.Address, key []byte) ([]byte, error) {
	k := storageKey(addr, key)
	g, err := t.getStem(k[:len(k)-1])
	if err != nil || g == nil {
		return nil, err
	}
	return g.lookup(k[len(k)-1]), nil
}

// UpdateStorage writes a storage slot; zero deletes it.
func (t *Trie) UpdateStorage(addr common.Address, key, value []byte) error {
	k := storageKey(addr, key)
	v := make([]byte, 32)
	copy(v[32-min(len(value), 32):], value)
	return t.stateWrite(k[:len(k)-1], k[len(k)-1:], [][]byte{v})
}

// UpdateStorageBatch writes the slots one by one.
func (t *Trie) UpdateStorageBatch(addr common.Address, keys [][]byte, values [][]byte) error {
	for i, key := range keys {
		if err := t.UpdateStorage(addr, key, values[i]); err != nil {
			return err
		}
	}
	return nil
}

// DeleteStorage removes a storage slot.
func (t *Trie) DeleteStorage(addr common.Address, key []byte) error {
	k := storageKey(addr, key)
	return t.updateStem(k[:len(k)-1], k[len(k)-1:], [][]byte{nil})
}

// UpdateContractCode writes the account's code leaf: the EIP-7702 indicator
// in its header, or the code hash with the code chunks in the code zone.
func (t *Trie) UpdateContractCode(addr common.Address, codeHash common.Hash, code []byte) error {
	subs := []byte{codeHashLeafKey, delegationLeafKey}
	if _, ok := types.ParseDelegation(code); ok {
		v := make([]byte, 32)
		copy(v, code)
		return t.stateWrite(headerStem(addr), subs, [][]byte{nil, v})
	}
	if len(code) == 0 {
		return nil
	}
	if err := t.stateWrite(headerStem(addr), subs, [][]byte{codeHash.Bytes(), nil}); err != nil {
		return err
	}
	chunks := bintrie.ChunkifyCode(code)
	for group := 0; 32*stemSubtreeWidth*group < len(chunks); group++ {
		var (
			subs []byte
			vals [][]byte
		)
		for i := stemSubtreeWidth * group; i < min(stemSubtreeWidth*(group+1), len(chunks)/32); i++ {
			if v := chunks[32*i : 32*(i+1)]; !bytes.Equal(v, zero[:]) {
				subs, vals = append(subs, byte(i)), append(vals, v)
			}
		}
		if len(subs) > 0 {
			if err := t.updateStem(codeStem(codeHash, uint64(group)), subs, vals); err != nil {
				return err
			}
		}
	}
	return nil
}

// Hash returns the root hash.
func (t *Trie) Hash() common.Hash {
	return t.root.hashAt(0)
}

// Commit collects the modified records. The trie is unusable afterwards.
func (t *Trie) Commit(_ bool) (common.Hash, *trienode.NodeSet) {
	defer func() { t.committed = true }()

	var (
		root = t.Hash()
		set  = trienode.NewNodeSet(common.Hash{})
	)
	for path := range t.deletes {
		if prev := t.tracer.Get([]byte(path)); prev != nil {
			set.AddNode([]byte(path), trienode.NewDeletedWithPrev(prev))
		}
	}
	t.collect(t.root, bitstr{}, 0, set)
	if len(set.Nodes) == 0 {
		return root, nil
	}
	t.root = hashedNode(root)
	return root, set
}

// collect adds the records modified below n, sitting at walk, to set.
func (t *Trie) collect(n node, walk bitstr, pos int, set *trienode.NodeSet) {
	switch n := n.(type) {
	case *groupNode:
		if !n.modified {
			return
		}
		n.modified = false
	case *branchNode:
		if !n.modified {
			return
		}
		n.modified = false
		child := pos + n.prefix.n + 1
		t.collect(n.left, walk.append(n.prefix).concat(0, bitstr{}), child, set)
		t.collect(n.right, walk.append(n.prefix).concat(1, bitstr{}), child, set)
	default:
		return
	}
	path := pathOf(walk)
	set.AddNode(path, trienode.NewNodeWithPrev(n.hashAt(pos), serializeNode(n, pos), t.tracer.Get(path)))
}

// Witness returns the records resolved from the database, keyed by path.
func (t *Trie) Witness() map[string][]byte {
	return t.tracer.Values()
}

// NodeIterator is not supported.
func (t *Trie) NodeIterator(startKey []byte) (trie.NodeIterator, error) {
	return nil, errUnsupported
}

// Prove is not supported.
func (t *Trie) Prove(key []byte, proofDb ethdb.KeyValueWriter) error {
	return errUnsupported
}

// IsUBT reports false: this is not the EIP-7864 tree.
func (t *Trie) IsUBT() bool { return false }

// IsPBT reports true.
func (t *Trie) IsPBT() bool { return true }

// PrefetchAccount resolves the accounts' header stems.
func (t *Trie) PrefetchAccount(addrs []common.Address) error {
	for _, addr := range addrs {
		if _, err := t.getStem(headerStem(addr)); err != nil {
			return err
		}
	}
	return nil
}

// PrefetchStorage resolves the slots' stems.
func (t *Trie) PrefetchStorage(addr common.Address, keys [][]byte) error {
	for _, key := range keys {
		k := storageKey(addr, key)
		if _, err := t.getStem(k[:len(k)-1]); err != nil {
			return err
		}
	}
	return nil
}

// Copy returns a deep copy of the trie.
func (t *Trie) Copy() *Trie {
	return &Trie{
		root:      t.root.copy(),
		reader:    t.reader,
		tracer:    t.tracer.Copy(),
		inserts:   maps.Clone(t.inserts),
		deletes:   maps.Clone(t.deletes),
		committed: t.committed,
	}
}
