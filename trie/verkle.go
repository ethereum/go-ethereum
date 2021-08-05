// Copyright 2021 go-ethereum Authors
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
	"crypto/sha256"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/gballet/go-verkle"
	"github.com/protolambda/go-kzg/bls"
)

const (
	VersionLeafKey    = 0
	BalanceLeafKey    = 1
	NonceLeafKey      = 2
	CodeKeccakLeafKey = 3
	CodeSizeLeafKey   = 4
)

var (
	zero                = big.NewInt(0)
	HeaderStorageOffset = big.NewInt(64)
	CodeOffset          = big.NewInt(128)
	MainStorageOffset   = big.NewInt(0).Lsh(big.NewInt(256), 31)
	VerkleNodeWidth     = big.NewInt(8)
	codeStorageDelta    = big.NewInt(0).Sub(HeaderStorageOffset, CodeOffset)
)

func GetTreeKey(address common.Address, treeIndex *big.Int, subIndex byte) []byte {
	digest := sha256.New()
	digest.Write(address[:])
	treeIndexBytes := treeIndex.Bytes()
	var payload [32]byte
	copy(payload[:len(treeIndexBytes)], treeIndexBytes)
	digest.Write(payload[:])
	h := digest.Sum(nil)
	h[31] = byte(subIndex)
	return h
}

func GetTreeKeyVersion(address common.Address) []byte {
	return GetTreeKey(address, zero, VersionLeafKey)
}

func GetTreeKeyBalance(address common.Address) []byte {
	return GetTreeKey(address, zero, BalanceLeafKey)
}

func GetTreeKeyNonce(address common.Address) []byte {
	return GetTreeKey(address, zero, NonceLeafKey)
}

func GetTreeKeyCodeKeccak(address common.Address) []byte {
	return GetTreeKey(address, zero, CodeKeccakLeafKey)
}

func GetTreeKeyCodeSize(address common.Address) []byte {
	return GetTreeKey(address, zero, CodeSizeLeafKey)
}

func GetTreeKeyCodeChunk(address common.Address, chunk *big.Int) []byte {
	chunkOffset := big.NewInt(0).Add(CodeOffset, chunk)
	treeIndex := big.NewInt(0).Div(chunkOffset, VerkleNodeWidth)
	subIndex := big.NewInt(0).Mod(chunkOffset, VerkleNodeWidth).Bytes()[0]
	return GetTreeKey(address, treeIndex, subIndex)
}

func GetTreeKeyStorageSlot(address common.Address, storageKey *big.Int) []byte {
	if storageKey.Cmp(codeStorageDelta) < 0 {
		storageKey.Add(HeaderStorageOffset, storageKey)
	} else {
		storageKey.Add(MainStorageOffset, storageKey)
	}
	treeIndex := big.NewInt(0).Div(storageKey, VerkleNodeWidth)
	subIndex := big.NewInt(0).Mod(storageKey, VerkleNodeWidth).Bytes()[0]
	return GetTreeKey(address, treeIndex, subIndex)
}

// VerkleTrie is a wrapper around VerkleNode that implements the trie.Trie
// interface so that Verkle trees can be reused verbatim.
type VerkleTrie struct {
	root verkle.VerkleNode
	db   *Database
}

func NewVerkleTrie(root verkle.VerkleNode, db *Database) *VerkleTrie {
	return &VerkleTrie{
		root: root,
		db:   db,
	}
}

// GetKey returns the sha3 preimage of a hashed key that was previously used
// to store a value.
func (trie *VerkleTrie) GetKey(key []byte) []byte {
	return key
}

// TryGet returns the value for key stored in the trie. The value bytes must
// not be modified by the caller. If a node was not found in the database, a
// trie.MissingNodeError is returned.
func (trie *VerkleTrie) TryGet(key []byte) ([]byte, error) {
	return trie.root.Get(key, trie.db.DiskDB().Get)
}

// TryUpdate associates key with value in the trie. If value has length zero, any
// existing value is deleted from the trie. The value bytes must not be modified
// by the caller while they are stored in the trie. If a node was not found in the
// database, a trie.MissingNodeError is returned.
func (trie *VerkleTrie) TryUpdate(key, value []byte) error {
	return trie.root.Insert(key, value)
}

// TryDelete removes any existing value for key from the trie. If a node was not
// found in the database, a trie.MissingNodeError is returned.
func (trie *VerkleTrie) TryDelete(key []byte) error {
	return trie.root.Delete(key)
}

// Hash returns the root hash of the trie. It does not write to the database and
// can be used even if the trie doesn't have one.
func (trie *VerkleTrie) Hash() common.Hash {
	// TODO cache this value
	rootC := trie.root.ComputeCommitment()
	return bls.FrTo32(rootC)
}

func nodeToDBKey(n verkle.VerkleNode) []byte {
	ret := bls.FrTo32(n.ComputeCommitment())
	return ret[:]
}

// Commit writes all nodes to the trie's memory database, tracking the internal
// and external (for account tries) references.
func (trie *VerkleTrie) Commit(onleaf LeafCallback) (common.Hash, error) {
	flush := make(chan verkle.VerkleNode)
	go func() {
		trie.root.(*verkle.InternalNode).Flush(func(n verkle.VerkleNode) {
			flush <- n
		})
		close(flush)
	}()
	for n := range flush {
		value, err := n.Serialize()
		if err != nil {
			panic(err)
		}

		if err := trie.db.DiskDB().Put(nodeToDBKey(n), value); err != nil {
			return common.Hash{}, err
		}
	}

	return trie.Hash(), nil
}

// NodeIterator returns an iterator that returns nodes of the trie. Iteration
// starts at the key after the given start key.
func (trie *VerkleTrie) NodeIterator(startKey []byte) NodeIterator {
	it := &verkleNodeIterator{trie: trie}
	return it
}

// Prove constructs a Merkle proof for key. The result contains all encoded nodes
// on the path to the value at key. The value itself is also included in the last
// node and can be retrieved by verifying the proof.
//
// If the trie does not contain a value for key, the returned proof contains all
// nodes of the longest existing prefix of the key (at least the root), ending
// with the node that proves the absence of the key.
func (trie *VerkleTrie) Prove(key []byte, fromLevel uint, proofDb ethdb.KeyValueWriter) error {
	panic("not implemented")
}

func (trie *VerkleTrie) Copy(db *Database) *VerkleTrie {
	return &VerkleTrie{
		root: trie.root.Copy(),
		db:   db,
	}
}

