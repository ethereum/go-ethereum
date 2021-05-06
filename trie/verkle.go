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
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie/utils"
	"github.com/gballet/go-verkle"
	"github.com/protolambda/go-kzg/bls"
)

// VerkleTrie is a wrapper around VerkleNode that implements the trie.Trie
// interface so that Verkle trees can be reused verbatim.
type VerkleTrie struct {
	root verkle.VerkleNode
	db   *Database
}

//func (vt *VerkleTrie) ToDot() string {
//return verkle.ToDot(vt.root)
//}

func NewVerkleTrie(root verkle.VerkleNode, db *Database) *VerkleTrie {
	return &VerkleTrie{
		root: root,
		db:   db,
	}
}

var errInvalidProof = errors.New("invalid proof")

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

func (t *VerkleTrie) TryUpdateAccount(key []byte, acc *types.StateAccount) error {

	var err error
	if err = t.TryUpdate(utils.GetTreeKeyVersion(key), []byte{0}); err != nil {
		return fmt.Errorf("updateStateObject (%x) error: %v", key, err)
	}
	var nonce [32]byte
	binary.BigEndian.PutUint64(nonce[:], acc.Nonce)
	if err = t.TryUpdate(utils.GetTreeKeyNonce(key), nonce[:]); err != nil {
		return fmt.Errorf("updateStateObject (%x) error: %v", key, err)
	}
	if err = t.TryUpdate(utils.GetTreeKeyBalance(key), acc.Balance.Bytes()); err != nil {
		return fmt.Errorf("updateStateObject (%x) error: %v", key, err)
	}
	if err = t.TryUpdate(utils.GetTreeKeyCodeKeccak(key), acc.CodeHash); err != nil {
		return fmt.Errorf("updateStateObject (%x) error: %v", key, err)
	}

	return nil
}

// TryUpdate associates key with value in the trie. If value has length zero, any
// existing value is deleted from the trie. The value bytes must not be modified
// by the caller while they are stored in the trie. If a node was not found in the
// database, a trie.MissingNodeError is returned.
func (trie *VerkleTrie) TryUpdate(key, value []byte) error {
	return trie.root.Insert(key, value, func(h []byte) ([]byte, error) {
		return trie.db.DiskDB().Get(h)
	})
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
func (trie *VerkleTrie) Commit(onleaf LeafCallback) (common.Hash, int, error) {
	flush := make(chan verkle.VerkleNode)
	go func() {
		trie.root.(*verkle.InternalNode).Flush(func(n verkle.VerkleNode) {
			flush <- n
		})
		close(flush)
	}()
	var commitCount int
	for n := range flush {
		commitCount += 1
		value, err := n.Serialize()
		if err != nil {
			panic(err)
		}

		if err := trie.db.DiskDB().Put(nodeToDBKey(n), value); err != nil {
			return common.Hash{}, commitCount, err
		}
	}

	return trie.Hash(), commitCount, nil
}

// NodeIterator returns an iterator that returns nodes of the trie. Iteration
// starts at the key after the given start key.
func (trie *VerkleTrie) NodeIterator(startKey []byte) NodeIterator {
	return newVerkleNodeIterator(trie, nil)
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
func (trie *VerkleTrie) IsVerkle() bool {
	return true
}

type KeyValuePair struct {
	Key   []byte
	Value []byte
}

type verkleproof struct {
	D *bls.G1Point
	Y *bls.Fr
	Σ *bls.G1Point

	Cis     []*bls.G1Point
	Indices []uint
	Yis     []*bls.Fr

	Leaves []KeyValuePair
}

func (trie *VerkleTrie) ProveAndSerialize(keys [][]byte, kv map[common.Hash][]byte) ([]byte, error) {
	d, y, σ, cis, indices, yis := verkle.MakeVerkleMultiProof(trie.root, keys)
	vp := verkleproof{
		D:       d,
		Y:       y,
		Σ:       σ,
		Cis:     cis,
		Indices: indices,
		Yis:     yis,
	}
	for key, val := range kv {
		var k [32]byte
		copy(k[:], key[:])
		vp.Leaves = append(vp.Leaves, KeyValuePair{
			Key:   k[:],
			Value: val,
		})
	}
	return rlp.EncodeToBytes(vp)
}

func DeserializeAndVerifyVerkleProof(proof []byte) (*bls.G1Point, *bls.Fr, *bls.G1Point, map[common.Hash]common.Hash, error) {
	d, y, σ, cis, indices, yis, leaves, err := deserializeVerkleProof(proof)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("could not deserialize proof: %w", err)
	}
	if !verkle.VerifyVerkleProof(d, σ, y, cis, indices, yis, verkle.GetKZGConfig()) {
		return nil, nil, nil, nil, errInvalidProof
	}

	return d, y, σ, leaves, nil
}

func deserializeVerkleProof(proof []byte) (*bls.G1Point, *bls.Fr, *bls.G1Point, []*bls.G1Point, []uint, []*bls.Fr, map[common.Hash]common.Hash, error) {
	var vp verkleproof
	err := rlp.DecodeBytes(proof, &vp)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("verkle proof deserialization error: %w", err)
	}
	leaves := make(map[common.Hash]common.Hash, len(vp.Leaves))
	for _, kvp := range vp.Leaves {
		leaves[common.BytesToHash(kvp.Key)] = common.BytesToHash(kvp.Value)
	}
	return vp.D, vp.Y, vp.Σ, vp.Cis, vp.Indices, vp.Yis, leaves, nil
}

// Copy the values here so as to avoid an import cycle
const (
	PUSH1  = 0x60
	PUSH32 = 0x71
)

func ChunkifyCode(addr common.Address, code []byte) ([][32]byte, error) {
	lastOffset := byte(0)
	chunkCount := len(code) / 31
	if len(code)%31 != 0 {
		chunkCount++
	}
	chunks := make([][32]byte, chunkCount)
	for i, chunk := range chunks {
		end := 31 * (i + 1)
		if len(code) < end {
			end = len(code)
		}
		copy(chunk[1:], code[31*i:end])
		for j := lastOffset; int(j) < len(code[31*i:end]); j++ {
			if code[j] >= byte(PUSH1) && code[j] <= byte(PUSH32) {
				j += code[j] - byte(PUSH1) + 1
				lastOffset = (j + 1) % 31
			}
		}
		chunk[0] = lastOffset
	}

	return chunks, nil
}
