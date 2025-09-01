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

package bintrie

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/triedb/database"
	"github.com/holiman/uint256"
)

var errInvalidRootType = errors.New("invalid root type")

// NewBinaryNode creates a new empty binary trie
func NewBinaryNode() BinaryNode {
	return Empty{}
}

// BinaryTrie is the implementation of https://eips.ethereum.org/EIPS/eip-7864.
type BinaryTrie struct {
	root   BinaryNode
	reader *trie.Reader
	tracer *trie.PrevalueTracer
}

// ToDot converts the binary trie to a DOT language representation. Useful for debugging.
func (t *BinaryTrie) ToDot() string {
	t.root.Hash()
	return ToDot(t.root)
}

// NewBinaryTrie creates a new binary trie.
func NewBinaryTrie(root common.Hash, db database.NodeDatabase) (*BinaryTrie, error) {
	reader, err := trie.NewReader(root, common.Hash{}, db)
	if err != nil {
		return nil, err
	}
	t := &BinaryTrie{
		root:   NewBinaryNode(),
		reader: reader,
		tracer: trie.NewPrevalueTracer(),
	}
	// Parse the root node if it's not empty
	if root != types.EmptyBinaryHash && root != types.EmptyRootHash {
		blob, err := t.nodeResolver(nil, root)
		if err != nil {
			return nil, err
		}
		node, err := DeserializeNode(blob, 0)
		if err != nil {
			return nil, err
		}
		t.root = node
	}
	return t, nil
}

// nodeResolver is a node resolver that reads nodes from the flatdb.
func (t *BinaryTrie) nodeResolver(path []byte, hash common.Hash) ([]byte, error) {
	// empty nodes will be serialized as common.Hash{}, so capture
	// this special use case.
	if hash == (common.Hash{}) {
		return nil, nil // empty node
	}
	blob, err := t.reader.Node(path, hash)
	if err != nil {
		return nil, err
	}
	t.tracer.Put(path, blob)
	return blob, nil
}

// GetKey returns the sha3 preimage of a hashed key that was previously used
// to store a value.
func (t *BinaryTrie) GetKey(key []byte) []byte {
	return key
}

// GetWithHashedKey returns the value, assuming that the key has already
// been hashed.
func (t *BinaryTrie) GetWithHashedKey(key []byte) ([]byte, error) {
	return t.root.Get(key, t.nodeResolver)
}

// GetAccount returns the account information for the given address.
func (t *BinaryTrie) GetAccount(addr common.Address) (*types.StateAccount, error) {
	var (
		values [][]byte
		err    error
		acc    = &types.StateAccount{}
		key    = GetBinaryTreeKey(addr, zero[:])
	)
	switch r := t.root.(type) {
	case *InternalNode:
		values, err = r.GetValuesAtStem(key[:31], t.nodeResolver)
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

	// The following code is required for the MPT->Binary conversion.
	// An account can be partially migrated, where storage slots were moved to the binary
	// but not yet the account. This means some account information as (header) storage slots
	// are in the binary trie but basic account information must be read in the base tree (MPT).
	// TODO: we can simplify this logic depending if the conversion is in progress or finished.
	emptyAccount := true
	for i := 0; values != nil && i <= CodeHashLeafKey && emptyAccount; i++ {
		emptyAccount = emptyAccount && values[i] == nil
	}
	if emptyAccount {
		return nil, nil
	}

	// If the account has been deleted, then values[10] will be 0 and not nil. If it has
	// been recreated after that, then its code keccak will NOT be 0. So return `nil` if
	// the nonce, and values[10], and code keccak is 0.
	if bytes.Equal(values[BasicDataLeafKey], zero[:]) && len(values) > 10 && len(values[10]) > 0 && bytes.Equal(values[CodeHashLeafKey], zero[:]) {
		return nil, nil
	}

	acc.Nonce = binary.BigEndian.Uint64(values[BasicDataLeafKey][BasicDataNonceOffset:])
	var balance [16]byte
	copy(balance[:], values[BasicDataLeafKey][BasicDataBalanceOffset:])
	acc.Balance = new(uint256.Int).SetBytes(balance[:])
	acc.CodeHash = values[CodeHashLeafKey]

	return acc, nil
}

// GetStorage returns the value for key stored in the trie. The value bytes must
// not be modified by the caller. If a node was not found in the database, a
// trie.MissingNodeError is returned.
func (t *BinaryTrie) GetStorage(addr common.Address, key []byte) ([]byte, error) {
	return t.root.Get(GetBinaryTreeKey(addr, key), t.nodeResolver)
}

// UpdateAccount updates the account information for the given address.
func (t *BinaryTrie) UpdateAccount(addr common.Address, acc *types.StateAccount, codeLen int) error {
	var (
		err       error
		basicData [32]byte
		values    = make([][]byte, NodeWidth)
		stem      = GetBinaryTreeKey(addr, zero[:])
	)
	binary.BigEndian.PutUint32(basicData[BasicDataCodeSizeOffset-1:], uint32(codeLen))
	binary.BigEndian.PutUint64(basicData[BasicDataNonceOffset:], acc.Nonce)

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
	values[BasicDataLeafKey] = basicData[:]
	values[CodeHashLeafKey] = acc.CodeHash[:]

	t.root, err = t.root.InsertValuesAtStem(stem, values, t.nodeResolver, 0)
	return err
}

// UpdateStem updates the values for the given stem key.
func (t *BinaryTrie) UpdateStem(key []byte, values [][]byte) error {
	var err error
	t.root, err = t.root.InsertValuesAtStem(key, values, t.nodeResolver, 0)
	return err
}

// UpdateStorage associates key with value in the trie. If value has length zero, any
// existing value is deleted from the trie. The value bytes must not be modified
// by the caller while they are stored in the trie. If a node was not found in the
// database, a trie.MissingNodeError is returned.
func (t *BinaryTrie) UpdateStorage(address common.Address, key, value []byte) error {
	k := GetBinaryTreeKeyStorageSlot(address, key)
	var v [32]byte
	if len(value) >= 32 {
		copy(v[:], value[:32])
	} else {
		copy(v[32-len(value):], value[:])
	}
	root, err := t.root.Insert(k, v[:], t.nodeResolver, 0)
	if err != nil {
		return fmt.Errorf("UpdateStorage (%x) error: %v", address, err)
	}
	t.root = root
	return nil
}

// DeleteAccount is a no-op as it is disabled in stateless.
func (t *BinaryTrie) DeleteAccount(addr common.Address) error {
	return nil
}

// DeleteStorage removes any existing value for key from the trie. If a node was not
// found in the database, a trie.MissingNodeError is returned.
func (t *BinaryTrie) DeleteStorage(addr common.Address, key []byte) error {
	k := GetBinaryTreeKey(addr, key)
	var zero [32]byte
	root, err := t.root.Insert(k, zero[:], t.nodeResolver, 0)
	if err != nil {
		return fmt.Errorf("DeleteStorage (%x) error: %v", addr, err)
	}
	t.root = root
	return nil
}

// Hash returns the root hash of the trie. It does not write to the database and
// can be used even if the trie doesn't have one.
func (t *BinaryTrie) Hash() common.Hash {
	return t.root.Hash()
}

// Commit writes all nodes to the trie's memory database, tracking the internal
// and external (for account tries) references.
func (t *BinaryTrie) Commit(_ bool) (common.Hash, *trienode.NodeSet) {
	root := t.root.(*InternalNode)
	nodeset := trienode.NewNodeSet(common.Hash{})

	err := root.CollectNodes(nil, func(path []byte, node BinaryNode) {
		serialized := SerializeNode(node)
		nodeset.AddNode(path, trienode.NewNodeWithPrev(common.Hash{}, serialized, t.tracer.Get(path)))
	})
	if err != nil {
		panic(fmt.Errorf("CollectNodes failed: %v", err))
	}
	// Serialize root commitment form
	return t.Hash(), nodeset
}

// NodeIterator returns an iterator that returns nodes of the trie. Iteration
// starts at the key after the given start key.
func (t *BinaryTrie) NodeIterator(startKey []byte) (trie.NodeIterator, error) {
	return newBinaryNodeIterator(t, nil)
}

// Prove constructs a Merkle proof for key. The result contains all encoded nodes
// on the path to the value at key. The value itself is also included in the last
// node and can be retrieved by verifying the proof.
//
// If the trie does not contain a value for key, the returned proof contains all
// nodes of the longest existing prefix of the key (at least the root), ending
// with the node that proves the absence of the key.
func (t *BinaryTrie) Prove(key []byte, proofDb ethdb.KeyValueWriter) error {
	panic("not implemented")
}

// Copy creates a deep copy of the trie.
func (t *BinaryTrie) Copy() *BinaryTrie {
	return &BinaryTrie{
		root:   t.root.Copy(),
		reader: t.reader,
		tracer: t.tracer.Copy(),
	}
}

// IsVerkle returns true if the trie is a Verkle tree.
func (t *BinaryTrie) IsVerkle() bool {
	// TODO @gballet This is technically NOT a verkle tree, but it has the same
	// behavior and basic structure, so for all intents and purposes, it can be
	// treated as such. Rename this when verkle gets removed.
	return true
}

// UpdateContractCode updates the contract code into the trie.
//
// Note: the basic data leaf needs to have been previously created for this to work
func (t *BinaryTrie) UpdateContractCode(addr common.Address, codeHash common.Hash, code []byte) error {
	var (
		chunks = trie.ChunkifyCode(code)
		values [][]byte
		key    []byte
		err    error
	)
	for i, chunknr := 0, uint64(0); i < len(chunks); i, chunknr = i+32, chunknr+1 {
		groupOffset := (chunknr + 128) % 256
		if groupOffset == 0 /* start of new group */ || chunknr == 0 /* first chunk in header group */ {
			values = make([][]byte, NodeWidth)
			var offset [32]byte
			binary.LittleEndian.PutUint64(offset[24:], chunknr+128)
			key = GetBinaryTreeKey(addr, offset[:])
		}
		values[groupOffset] = chunks[i : i+32]

		if groupOffset == 255 || len(chunks)-i <= 32 {
			err = t.UpdateStem(key[:31], values)

			if err != nil {
				return fmt.Errorf("UpdateContractCode (addr=%x) error: %w", addr[:], err)
			}
		}
	}
	return nil
}

// PrefetchAccount attempts to resolve specific accounts from the database
// to accelerate subsequent trie operations.
func (t *BinaryTrie) PrefetchAccount(addresses []common.Address) error {
	for _, addr := range addresses {
		if _, err := t.GetAccount(addr); err != nil {
			return err
		}
	}
	return nil
}

// PrefetchStorage attempts to resolve specific storage slots from the database
// to accelerate subsequent trie operations.
func (t *BinaryTrie) PrefetchStorage(addr common.Address, keys [][]byte) error {
	for _, key := range keys {
		if _, err := t.GetStorage(addr, key); err != nil {
			return err
		}
	}
	return nil
}

// Witness returns a set containing all trie nodes that have been accessed.
func (t *BinaryTrie) Witness() map[string][]byte {
	panic("not implemented")
}
