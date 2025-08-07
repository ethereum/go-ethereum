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
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/trie/bintrie"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/trie/utils"
	"github.com/ethereum/go-ethereum/triedb/database"
	"github.com/holiman/uint256"
)

// zero is the zero value for a 32-byte array.
var zero [32]byte

// NewBinaryNode creates a new empty binary trie
func NewBinaryNode() bintrie.BinaryNode {
	return bintrie.Empty{}
}

// BinaryTrie is a wrapper around VerkleNode that implements the trie.Trie
// interface so that Verkle trees can be reused verbatim.
type BinaryTrie struct {
	root   bintrie.BinaryNode
	reader *trieReader
}

// ToDot converts the binary trie to a DOT language representation. Useful for debugging.
func (trie *BinaryTrie) ToDot() string {
	trie.root.Hash()
	return bintrie.ToDot(trie.root)
}

// NewBinaryTrie creates a new binary trie.
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
		node, err = bintrie.DeserializeNode(blob, 0)
		if err != nil {
			return nil, err
		}
	}
	return &BinaryTrie{
		root:   node,
		reader: reader,
	}, nil
}

// FlatdbNodeResolver is a node resolver that reads nodes from the flatdb.
func (trie *BinaryTrie) FlatdbNodeResolver(path []byte, hash common.Hash) ([]byte, error) {
	// empty nodes will be serialized as common.Hash{}, so capture
	// this special use case.
	if hash == (common.Hash{}) {
		return nil, nil // empty node
	}
	return trie.reader.node(path, hash)
}

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

// GetAccount returns the account information for the given address.
func (trie *BinaryTrie) GetAccount(addr common.Address) (*types.StateAccount, error) {
	acc := &types.StateAccount{}
	versionkey := utils.GetBinaryTreeKey(addr, zero[:])
	var (
		values [][]byte
		err    error
	)
	switch r := trie.root.(type) {
	case *bintrie.InternalNode:
		values, err = r.GetValuesAtStem(versionkey[:31], trie.FlatdbNodeResolver)
	case *bintrie.StemNode:
		values = r.Values
	case bintrie.Empty:
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

// UpdateAccount updates the account information for the given address.
func (trie *BinaryTrie) UpdateAccount(addr common.Address, acc *types.StateAccount, codeLen int) error {
	var (
		err       error
		basicData [32]byte
		values    = make([][]byte, bintrie.NodeWidth)
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

// UpdateStem updates the values for the given stem key.
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

// DeleteAccount is a no-op as it is disabled in stateless.
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
	return trie.root.Hash()
}

// Commit writes all nodes to the trie's memory database, tracking the internal
// and external (for account tries) references.
func (trie *BinaryTrie) Commit(_ bool) (common.Hash, *trienode.NodeSet, error) {
	root := trie.root.(*bintrie.InternalNode)
	nodeset := trienode.NewNodeSet(common.Hash{})

	err := root.CollectNodes(nil, func(path []byte, node bintrie.BinaryNode) {
		serialized := bintrie.SerializeNode(node)
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

// Copy creates a deep copy of the trie.
func (trie *BinaryTrie) Copy() *BinaryTrie {
	return &BinaryTrie{
		root:   trie.root.Copy(),
		reader: trie.reader,
	}
}

// IsVerkle returns true if the trie is a Verkle tree.
func (trie *BinaryTrie) IsVerkle() bool {
	// TODO @gballet This is technically NOT a verkle tree, but it has the same
	// behavior and basic structure, so for all intents and purposes, it can be
	// treated as such. Rename this when verkle gets removed.
	return true
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
			values = make([][]byte, bintrie.NodeWidth)
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
