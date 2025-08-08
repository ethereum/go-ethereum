// Copyright 2025 go-ethereum Authors
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
	"encoding/binary"
	"errors"
	"fmt"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/trie/utils"
	"github.com/ethereum/go-ethereum/triedb/database"
	"github.com/ethereum/go-verkle"
	"github.com/holiman/uint256"
	"github.com/zeebo/blake3"
)

type (
	NodeFlushFn    func([]byte, BinaryNode)
	NodeResolverFn func([]byte, common.Hash) ([]byte, error)
)

type BinaryNode interface {
	Get([]byte, NodeResolverFn) ([]byte, error)
	Insert([]byte, []byte, NodeResolverFn) (BinaryNode, error)
	Commit() common.Hash
	Copy() BinaryNode
	Hash() common.Hash
	GetValuesAtStem([]byte, NodeResolverFn) ([][]byte, error)
	InsertValuesAtStem([]byte, [][]byte, NodeResolverFn, int) (BinaryNode, error)
	CollectNodes([]byte, NodeFlushFn) error

	toDot(parent, path string) string
	GetHeight() int
}

type Empty struct{}

func (e Empty) Get(_ []byte, _ NodeResolverFn) ([]byte, error) {
	return nil, nil
}

func (e Empty) Insert(key []byte, value []byte, _ NodeResolverFn) (BinaryNode, error) {
	var values [256][]byte
	values[key[31]] = value
	return &StemNode{
		Stem:   slices.Clone(key[:31]),
		Values: values[:],
	}, nil
}

func (e Empty) Commit() common.Hash {
	return common.Hash{}
}

func (e Empty) Copy() BinaryNode {
	return Empty{}
}

func (e Empty) Hash() common.Hash {
	return common.Hash{}
}

func (e Empty) GetValuesAtStem(_ []byte, _ NodeResolverFn) ([][]byte, error) {
	var values [256][]byte
	return values[:], nil
}

func (e Empty) InsertValuesAtStem(key []byte, values [][]byte, _ NodeResolverFn, depth int) (BinaryNode, error) {
	return &StemNode{
		Stem:   slices.Clone(key[:31]),
		Values: values,
		depth:  depth,
	}, nil
}

func (e Empty) CollectNodes(_ []byte, _ NodeFlushFn) error {
	return nil
}

func (e Empty) toDot(parent string, path string) string {
	return ""
}

func (e Empty) GetHeight() int {
	return 0
}

type HashedNode common.Hash

func (h HashedNode) Get(_ []byte, _ NodeResolverFn) ([]byte, error) {
	panic("not implemented") // TODO: Implement
}

func (h HashedNode) Insert(key []byte, value []byte, resolver NodeResolverFn) (BinaryNode, error) {
	return nil, errors.New("insert not implemented for hashed node")
}

func (h HashedNode) Commit() common.Hash {
	return common.Hash(h)
}

func (h HashedNode) Copy() BinaryNode {
	nh := common.Hash(h)
	return HashedNode(nh)
}

func (h HashedNode) Hash() common.Hash {
	return common.Hash(h)
}

func (h HashedNode) GetValuesAtStem(_ []byte, _ NodeResolverFn) ([][]byte, error) {
	return nil, errors.New("attempted to get values from an unresolved node")
}

func (h HashedNode) InsertValuesAtStem(key []byte, values [][]byte, resolver NodeResolverFn, depth int) (BinaryNode, error) {
	return nil, errors.New("insertValuesAtStem not implemented for hashed node")
}

func (h HashedNode) toDot(parent string, path string) string {
	me := fmt.Sprintf("hash%s", path)
	ret := fmt.Sprintf("%s [label=\"%x\"]\n", me, h)
	ret = fmt.Sprintf("%s %s -> %s\n", ret, parent, me)
	return ret
}

func (h HashedNode) CollectNodes([]byte, NodeFlushFn) error {
	panic("not implemented") // TODO: Implement
}

func (h HashedNode) GetHeight() int {
	panic("tried to get the height of a hashed node, this is a bug")
}

type StemNode struct {
	Stem   []byte
	Values [][]byte
	depth  int
}

func (bt *StemNode) Get(key []byte, _ NodeResolverFn) ([]byte, error) {
	panic("this should not be called directly")
}

func (bt *StemNode) Insert(key []byte, value []byte, _ NodeResolverFn) (BinaryNode, error) {
	if !bytes.Equal(bt.Stem, key[:31]) {
		bitStem := bt.Stem[bt.depth/8] >> (7 - (bt.depth % 8)) & 1

		new := &InternalNode{depth: bt.depth}
		bt.depth++
		var child, other *BinaryNode
		if bitStem == 0 {
			new.left = bt
			child = &new.left
			other = &new.right
		} else {
			new.right = bt
			child = &new.right
			other = &new.left
		}

		bitKey := key[new.depth/8] >> (7 - (new.depth % 8)) & 1
		if bitKey == bitStem {
			var err error
			*child, err = (*child).Insert(key, value, nil)
			if err != nil {
				return new, fmt.Errorf("insert error: %w", err)
			}
			*other = Empty{}
		} else {
			var values [256][]byte
			values[key[31]] = value
			*other = &StemNode{
				Stem:   slices.Clone(key[:31]),
				Values: values[:],
				depth:  new.depth + 1,
			}
		}

		return new, nil
	}
	if len(value) != 32 {
		return bt, errors.New("invalid insertion: value length")
	}

	bt.Values[key[31]] = value
	return bt, nil
}

func (bt *StemNode) Commit() common.Hash {
	return bt.Hash()
}

func (bt *StemNode) Copy() BinaryNode {
	var values [256][]byte
	for i, v := range bt.Values {
		values[i] = slices.Clone(v)
	}
	return &StemNode{
		Stem:   slices.Clone(bt.Stem),
		Values: values[:],
		depth:  bt.depth,
	}
}

func (bt *StemNode) GetHeight() int {
	return 1
}

func (bt *StemNode) Hash() common.Hash {
	var data [verkle.NodeWidth]common.Hash
	for i, v := range bt.Values {
		if v != nil {
			h := blake3.Sum256(v)
			data[i] = common.BytesToHash(h[:])
		}
	}

	h := blake3.New()
	for level := 1; level <= 8; level++ {
		for i := range verkle.NodeWidth / (1 << level) {
			h.Reset()

			if data[i*2] == (common.Hash{}) && data[i*2+1] == (common.Hash{}) {
				data[i] = common.Hash{}
				continue
			}

			h.Write(data[i*2][:])
			h.Write(data[i*2+1][:])
			data[i] = common.Hash(h.Sum(nil))
		}
	}

	h.Reset()
	h.Write(bt.Stem)
	h.Write([]byte{0})
	h.Write(data[0][:])
	return common.BytesToHash(h.Sum(nil))
}

func (bt *StemNode) CollectNodes(path []byte, flush NodeFlushFn) error {
	flush(path, bt)
	return nil
}

func (bt *StemNode) GetValuesAtStem(_ []byte, _ NodeResolverFn) ([][]byte, error) {
	return bt.Values[:], nil
}

func (bt *StemNode) InsertValuesAtStem(key []byte, values [][]byte, _ NodeResolverFn, depth int) (BinaryNode, error) {
	if !bytes.Equal(bt.Stem, key[:31]) {
		bitStem := bt.Stem[bt.depth/8] >> (7 - (bt.depth % 8)) & 1

		new := &InternalNode{depth: bt.depth}
		bt.depth++
		var child, other *BinaryNode
		if bitStem == 0 {
			new.left = bt
			child = &new.left
			other = &new.right
		} else {
			new.right = bt
			child = &new.right
			other = &new.left
		}

		bitKey := key[new.depth/8] >> (7 - (new.depth % 8)) & 1
		if bitKey == bitStem {
			var err error
			*child, err = (*child).InsertValuesAtStem(key, values, nil, depth+1)
			if err != nil {
				return new, fmt.Errorf("insert error: %w", err)
			}
			*other = Empty{}
		} else {
			*other = &StemNode{
				Stem:   slices.Clone(key[:31]),
				Values: values,
				depth:  new.depth + 1,
			}
		}

		return new, nil
	}

	// same stem, just merge the two value lists
	for i, v := range values {
		if v != nil {
			bt.Values[i] = v
		}
	}
	return bt, nil
}

func (bt *StemNode) toDot(parent, path string) string {
	me := fmt.Sprintf("stem%s", path)
	ret := fmt.Sprintf("%s [label=\"stem=%x c=%x\"]\n", me, bt.Stem, bt.Hash())
	ret = fmt.Sprintf("%s %s -> %s\n", ret, parent, me)
	for i, v := range bt.Values {
		if v != nil {
			ret = fmt.Sprintf("%s%s%x [label=\"%x\"]\n", ret, me, i, v)
			ret = fmt.Sprintf("%s%s -> %s%x\n", ret, me, me, i)
		}
	}
	return ret
}

func (bt *StemNode) Key(i int) []byte {
	var ret [32]byte
	copy(ret[:], bt.Stem)
	ret[verkle.StemSize] = byte(i)
	return ret[:]
}

type InternalNode struct {
	left, right BinaryNode
	depth       int
}

func NewBinaryNode() BinaryNode {
	return Empty{}
}

func keyToPath(depth int, key []byte) ([]byte, error) {
	path := make([]byte, 0, depth+1)

	if depth > 31*8 {
		return nil, errors.New("node too deep")
	}

	for i := range depth + 1 {
		bit := key[i/8] >> (7 - (i % 8)) & 1
		path = append(path, bit)
	}

	return path, nil
}

func (bt *InternalNode) GetValuesAtStem(stem []byte, resolver NodeResolverFn) ([][]byte, error) {
	if bt.depth > 31*8 {
		return nil, errors.New("node too deep")
	}

	bit := stem[bt.depth/8] >> (7 - (bt.depth % 8)) & 1
	var child *BinaryNode
	if bit == 0 {
		child = &bt.left
	} else {
		child = &bt.right
	}

	if hn, ok := (*child).(HashedNode); ok {
		path, err := keyToPath(bt.depth, stem)
		if err != nil {
			return nil, fmt.Errorf("GetValuesAtStem resolve error: %w", err)
		}
		data, err := resolver(path, common.Hash(hn))
		if err != nil {
			return nil, fmt.Errorf("GetValuesAtStem resolve error: %w", err)
		}
		node, err := DeserializeNode(data, bt.depth+1)
		if err != nil {
			return nil, fmt.Errorf("GetValuesAtStem node deserialization error: %w", err)
		}
		*child = node
	}
	return (*child).GetValuesAtStem(stem, resolver)
}

func (bt *InternalNode) Get(key []byte, resolver NodeResolverFn) ([]byte, error) {
	values, err := bt.GetValuesAtStem(key[:31], resolver)
	if err != nil {
		return nil, fmt.Errorf("Get error: %w", err)
	}
	return values[key[31]], nil
}

func (bt *InternalNode) Insert(key []byte, value []byte, resolver NodeResolverFn) (BinaryNode, error) {
	var values [256][]byte
	values[key[31]] = value
	return bt.InsertValuesAtStem(key[:31], values[:], resolver, 0)
}

func (bt *InternalNode) Commit() common.Hash {
	hasher := blake3.New()
	hasher.Write(bt.left.Commit().Bytes())
	hasher.Write(bt.right.Commit().Bytes())
	sum := hasher.Sum(nil)
	return common.BytesToHash(sum)
}

func (bt *InternalNode) Copy() BinaryNode {
	return &InternalNode{
		left:  bt.left.Copy(),
		right: bt.right.Copy(),
		depth: bt.depth,
	}
}

func (bt *InternalNode) Hash() common.Hash {
	h := blake3.New()
	if bt.left != nil {
		h.Write(bt.left.Hash().Bytes())
	} else {
		h.Write(zero[:])
	}
	if bt.right != nil {
		h.Write(bt.right.Hash().Bytes())
	} else {
		h.Write(zero[:])
	}
	return common.BytesToHash(h.Sum(nil))
}

// InsertValuesAtStem inserts a full value group at the given stem in the internal node.
// Already-existing values will be overwritten.
func (bt *InternalNode) InsertValuesAtStem(stem []byte, values [][]byte, resolver NodeResolverFn, depth int) (BinaryNode, error) {
	bit := stem[bt.depth/8] >> (7 - (bt.depth % 8)) & 1
	var (
		child *BinaryNode
		err   error
	)
	if bit == 0 {
		child = &bt.left
	} else {
		child = &bt.right
	}

	*child, err = (*child).InsertValuesAtStem(stem, values, resolver, depth+1)
	return bt, err
}

func (bt *InternalNode) CollectNodes(path []byte, flushfn NodeFlushFn) error {
	if bt.left != nil {
		var p [256]byte
		copy(p[:], path)
		childpath := p[:len(path)]
		childpath = append(childpath, 0)
		if err := bt.left.CollectNodes(childpath, flushfn); err != nil {
			return err
		}
	}
	if bt.right != nil {
		var p [256]byte
		copy(p[:], path)
		childpath := p[:len(path)]
		childpath = append(childpath, 1)
		if err := bt.right.CollectNodes(childpath, flushfn); err != nil {
			return err
		}
	}
	flushfn(path, bt)
	return nil
}

func (bt *InternalNode) GetHeight() int {
	var (
		leftHeight  int
		rightHeight int
	)
	if bt.left != nil {
		leftHeight = bt.left.GetHeight()
	}
	if bt.right != nil {
		rightHeight = bt.right.GetHeight()
	}
	return 1 + max(leftHeight, rightHeight)
}

func (bt *InternalNode) toDot(parent, path string) string {
	me := fmt.Sprintf("internal%s", path)
	ret := fmt.Sprintf("%s [label=\"I: %x\"]\n", me, bt.Hash())
	if len(parent) > 0 {
		ret = fmt.Sprintf("%s %s -> %s\n", ret, parent, me)
	}

	if bt.left != nil {
		ret = fmt.Sprintf("%s%s", ret, bt.left.toDot(me, fmt.Sprintf("%s%02x", path, 0)))
	}
	if bt.right != nil {
		ret = fmt.Sprintf("%s%s", ret, bt.right.toDot(me, fmt.Sprintf("%s%02x", path, 1)))
	}

	return ret
}

func SerializeNode(node BinaryNode) []byte {
	switch n := (node).(type) {
	case *InternalNode:
		var serialized [65]byte
		serialized[0] = 1
		copy(serialized[1:33], n.left.Hash().Bytes())
		copy(serialized[33:65], n.right.Hash().Bytes())
		return serialized[:]
	case *StemNode:
		var serialized [32 + 256*32]byte
		serialized[0] = 2
		copy(serialized[1:32], node.(*StemNode).Stem)
		bitmap := serialized[32:64]
		offset := 64
		for i, v := range node.(*StemNode).Values {
			if v != nil {
				bitmap[i/8] |= 1 << (7 - (i % 8))
				copy(serialized[offset:offset+32], v)
				offset += 32
			}
		}
		return serialized[:]
	default:
		panic("invalid node type")
	}
}

func DeserializeNode(serialized []byte, depth int) (BinaryNode, error) {
	if len(serialized) == 0 {
		return Empty{}, nil
	}

	switch serialized[0] {
	case 1:
		if len(serialized) != 65 {
			return nil, errors.New("invalid serialized node length")
		}
		return &InternalNode{
			depth: depth,
			left:  HashedNode(common.BytesToHash(serialized[1:33])),
			right: HashedNode(common.BytesToHash(serialized[33:65])),
		}, nil
	case 2:
		var values [256][]byte
		bitmap := serialized[32:64]
		offset := 64
		for i := range 256 {
			if bitmap[i/8]>>(7-(i%8))&1 == 1 {
				values[i] = serialized[offset : offset+32]
				offset += 32
			}
		}
		return &StemNode{
			Stem:   serialized[1:32],
			Values: values[:],
			depth:  depth,
		}, nil
	default:
		return nil, errors.New("invalid node type")
	}
}

// BinaryTrie is a wrapper around VerkleNode that implements the trie.Trie
// interface so that Verkle trees can be reused verbatim.
type BinaryTrie struct {
	root   BinaryNode
	reader *trieReader
}

func (trie *BinaryTrie) ToDot() string {
	trie.root.Commit()
	return trie.root.toDot("", "")
}

func NewBinaryTrie(root common.Hash, db database.NodeDatabase) (*BinaryTrie, error) {
	reader, err := newTrieReader(root, common.Hash{}, db)
	if err != nil {
		return nil, err
	}
	// Parse the root verkle node if it's not empty.
	node := NewBinaryNode()
	if root != types.EmptyVerkleHash && root != types.EmptyRootHash {
		blob, err := reader.node(nil, common.Hash{})
		if err != nil {
			return nil, err
		}
		node, err = DeserializeNode(blob, 0)
		if err != nil {
			return nil, err
		}
	}
	return &BinaryTrie{
		root:   node,
		reader: reader,
	}, nil
}

func (trie *BinaryTrie) FlatdbNodeResolver(path []byte, hash common.Hash) ([]byte, error) {
	if hash == (common.Hash{}) {
		return nil, nil // empty node
	}
	return trie.reader.node(path, hash)
}

var FlatDBVerkleNodeKeyPrefix = []byte("flat-") // prefix for flatdb keys

// GetKey returns the sha3 preimage of a hashed key that was previously used
// to store a value.
func (trie *BinaryTrie) GetKey(key []byte) []byte {
	return key
}

// Get returns the value for key stored in the trie. The value bytes must
// not be modified by the caller. If a node was not found in the database, a
// trie.MissingNodeError is returned.
func (trie *BinaryTrie) GetStorage(addr common.Address, key []byte) ([]byte, error) {
	return trie.root.Get(utils.GetBinaryTreeKey(addr, key), trie.FlatdbNodeResolver)
}

// GetWithHashedKey returns the value, assuming that the key has already
// been hashed.
func (trie *BinaryTrie) GetWithHashedKey(key []byte) ([]byte, error) {
	return trie.root.Get(key, trie.FlatdbNodeResolver)
}

func (trie *BinaryTrie) GetAccount(addr common.Address) (*types.StateAccount, error) {
	acc := &types.StateAccount{}
	versionkey := utils.GetBinaryTreeKey(addr, zero[:])
	var (
		values [][]byte
		err    error
	)
	switch r := trie.root.(type) {
	case *InternalNode:
		values, err = r.GetValuesAtStem(versionkey[:31], trie.FlatdbNodeResolver)
	case *StemNode:
		values = r.Values
	case Empty:
		return nil, nil
	default:
		// This will cover HashedNode but that should be fine since the
		// root node should always be resolved.
		return nil, errInvalidRootType
	}
	if err != nil {
		return nil, fmt.Errorf("GetAccount (%x) error: %v", addr, err)
	}

	// The following code is required for the MPT->VKT conversion.
	// An account can be partially migrated, where storage slots were moved to the VKT
	// but not yet the account. This means some account information as (header) storage slots
	// are in the VKT but basic account information must be read in the base tree (MPT).
	// TODO: we can simplify this logic depending if the conversion is in progress or finished.
	emptyAccount := true
	for i := 0; values != nil && i <= utils.CodeHashLeafKey && emptyAccount; i++ {
		emptyAccount = emptyAccount && values[i] == nil
	}
	if emptyAccount {
		return nil, nil
	}

	// if the account has been deleted, then values[10] will be 0 and not nil. If it has
	// been recreated after that, then its code keccak will NOT be 0. So return `nil` if
	// the nonce, and values[10], and code keccak is 0.
	if bytes.Equal(values[utils.BasicDataLeafKey], zero[:]) && len(values) > 10 && len(values[10]) > 0 && bytes.Equal(values[utils.CodeHashLeafKey], zero[:]) {
		return nil, nil
	}

	acc.Nonce = binary.BigEndian.Uint64(values[utils.BasicDataLeafKey][utils.BasicDataNonceOffset:])
	var balance [16]byte
	copy(balance[:], values[utils.BasicDataLeafKey][utils.BasicDataBalanceOffset:])
	acc.Balance = new(uint256.Int).SetBytes(balance[:])
	acc.CodeHash = values[utils.CodeHashLeafKey]

	return acc, nil
}

var zero [32]byte

func (trie *BinaryTrie) UpdateAccount(addr common.Address, acc *types.StateAccount, codeLen int) error {
	var (
		err       error
		basicData [32]byte
		values    = make([][]byte, verkle.NodeWidth)
		stem      = utils.GetBinaryTreeKey(addr, zero[:])
	)

	binary.BigEndian.PutUint32(basicData[utils.BasicDataCodeSizeOffset-1:], uint32(codeLen))
	binary.BigEndian.PutUint64(basicData[utils.BasicDataNonceOffset:], acc.Nonce)
	// Because the balance is a max of 16 bytes, truncate
	// the extra values. This happens in devmode, where
	// 0xff**32 is allocated to the developer account.
	balanceBytes := acc.Balance.Bytes()
	// TODO: reduce the size of the allocation in devmode, then panic instead
	// of truncating.
	if len(balanceBytes) > 16 {
		balanceBytes = balanceBytes[16:]
	}
	copy(basicData[32-len(balanceBytes):], balanceBytes[:])
	values[utils.BasicDataLeafKey] = basicData[:]
	values[utils.CodeHashLeafKey] = acc.CodeHash[:]

	trie.root, err = trie.root.InsertValuesAtStem(stem, values, trie.FlatdbNodeResolver, 0)
	return err
}

func (trie *BinaryTrie) UpdateStem(key []byte, values [][]byte) error {
	var err error
	trie.root, err = trie.root.InsertValuesAtStem(key, values, trie.FlatdbNodeResolver, 0)
	return err
}

// Update associates key with value in the trie. If value has length zero, any
// existing value is deleted from the trie. The value bytes must not be modified
// by the caller while they are stored in the trie. If a node was not found in the
// database, a trie.MissingNodeError is returned.
func (trie *BinaryTrie) UpdateStorage(address common.Address, key, value []byte) error {
	k := utils.GetBinaryTreeKeyStorageSlot(address, key)
	var v [32]byte
	if len(value) >= 32 {
		copy(v[:], value[:32])
	} else {
		copy(v[32-len(value):], value[:])
	}
	root, err := trie.root.Insert(k, v[:], trie.FlatdbNodeResolver)
	if err != nil {
		return fmt.Errorf("UpdateStorage (%x) error: %v", address, err)
	}
	trie.root = root
	return nil
}

func (trie *BinaryTrie) DeleteAccount(addr common.Address) error {
	return nil
}

// Delete removes any existing value for key from the trie. If a node was not
// found in the database, a trie.MissingNodeError is returned.
func (trie *BinaryTrie) DeleteStorage(addr common.Address, key []byte) error {
	k := utils.GetBinaryTreeKey(addr, key)
	var zero [32]byte
	root, err := trie.root.Insert(k, zero[:], trie.FlatdbNodeResolver)
	if err != nil {
		return fmt.Errorf("DeleteStorage (%x) error: %v", addr, err)
	}
	trie.root = root
	return nil
}

// Hash returns the root hash of the trie. It does not write to the database and
// can be used even if the trie doesn't have one.
func (trie *BinaryTrie) Hash() common.Hash {
	return trie.root.Commit()
}

// Commit writes all nodes to the trie's memory database, tracking the internal
// and external (for account tries) references.
func (trie *BinaryTrie) Commit(_ bool) (common.Hash, *trienode.NodeSet, error) {
	root := trie.root.(*InternalNode)
	nodeset := trienode.NewNodeSet(common.Hash{})

	err := root.CollectNodes(nil, func(path []byte, node BinaryNode) {
		serialized := SerializeNode(node)
		nodeset.AddNode(path, trienode.New(common.Hash{}, serialized))
	})
	if err != nil {
		panic(fmt.Errorf("CollectNodes failed: %v", err))
	}

	// Serialize root commitment form
	return trie.Hash(), nodeset, nil
}

// NodeIterator returns an iterator that returns nodes of the trie. Iteration
// starts at the key after the given start key.
func (trie *BinaryTrie) NodeIterator(startKey []byte) (NodeIterator, error) {
	return newBinaryNodeIterator(trie, nil)
}

// Prove constructs a Merkle proof for key. The result contains all encoded nodes
// on the path to the value at key. The value itself is also included in the last
// node and can be retrieved by verifying the proof.
//
// If the trie does not contain a value for key, the returned proof contains all
// nodes of the longest existing prefix of the key (at least the root), ending
// with the node that proves the absence of the key.
func (trie *BinaryTrie) Prove(key []byte, proofDb ethdb.KeyValueWriter) error {
	panic("not implemented")
}

func (trie *BinaryTrie) Copy() *BinaryTrie {
	return &BinaryTrie{
		root:   trie.root.Copy(),
		reader: trie.reader,
	}
}

func (trie *BinaryTrie) IsVerkle() bool {
	return true
}

func MakeBinaryMultiProof(pretrie, posttrie BinaryNode, keys [][]byte, resolver NodeResolverFn) (*verkle.VerkleProof, [][]byte, [][]byte, [][]byte, error) {
	return nil, nil, nil, nil, nil
}

func SerializeProof(proof *verkle.VerkleProof) (*verkle.VerkleProof, verkle.StateDiff, error) {
	return nil, nil, nil
}

func ProveAndSerialize(pretrie, posttrie *BinaryTrie, keys [][]byte, resolver NodeResolverFn) (*verkle.VerkleProof, verkle.StateDiff, error) {
	var postroot BinaryNode
	if posttrie != nil {
		postroot = posttrie.root
	}
	proof, _, _, _, err := MakeBinaryMultiProof(pretrie.root, postroot, keys, resolver)
	if err != nil {
		return nil, nil, err
	}

	p, kvps, err := SerializeProof(proof)
	if err != nil {
		return nil, nil, err
	}

	return p, kvps, nil
}

// Note: the basic data leaf needs to have been previously created for this to work
func (trie *BinaryTrie) UpdateContractCode(addr common.Address, codeHash common.Hash, code []byte) error {
	var (
		chunks = ChunkifyCode(code)
		values [][]byte
		key    []byte
		err    error
	)
	for i, chunknr := 0, uint64(0); i < len(chunks); i, chunknr = i+32, chunknr+1 {
		groupOffset := (chunknr + 128) % 256
		if groupOffset == 0 /* start of new group */ || chunknr == 0 /* first chunk in header group */ {
			values = make([][]byte, verkle.NodeWidth)
			var offset [32]byte
			binary.LittleEndian.PutUint64(offset[24:], chunknr+128)
			key = utils.GetBinaryTreeKey(addr, offset[:])
		}
		values[groupOffset] = chunks[i : i+32]

		if groupOffset == 255 || len(chunks)-i <= 32 {
			err = trie.UpdateStem(key[:31], values)

			if err != nil {
				return fmt.Errorf("UpdateContractCode (addr=%x) error: %w", addr[:], err)
			}
		}
	}
	return nil
}
