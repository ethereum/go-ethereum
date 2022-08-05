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
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/trie/utils"
	"github.com/gballet/go-verkle"
)

// VerkleTrie is a wrapper around VerkleNode that implements the trie.Trie
// interface so that Verkle trees can be reused verbatim.
type VerkleTrie struct {
	root verkle.VerkleNode
	db   *Database
}

func (vt *VerkleTrie) ToDot() string {
	return verkle.ToDot(vt.root)
}

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
	var (
		err                                error
		nonce, balance                     [32]byte
		balancekey, cskey, ckkey, noncekey [32]byte
	)

	// Only evaluate the polynomial once
	versionkey := utils.GetTreeKeyVersion(key[:])
	copy(balancekey[:], versionkey)
	balancekey[31] = utils.BalanceLeafKey
	copy(noncekey[:], versionkey)
	noncekey[31] = utils.NonceLeafKey
	copy(cskey[:], versionkey)
	cskey[31] = utils.CodeSizeLeafKey
	copy(ckkey[:], versionkey)
	ckkey[31] = utils.CodeKeccakLeafKey

	if err = t.TryUpdate(versionkey, []byte{0}); err != nil {
		return fmt.Errorf("updateStateObject (%x) error: %v", key, err)
	}
	binary.LittleEndian.PutUint64(nonce[:], acc.Nonce)
	if err = t.TryUpdate(noncekey[:], nonce[:]); err != nil {
		return fmt.Errorf("updateStateObject (%x) error: %v", key, err)
	}
	bbytes := acc.Balance.Bytes()
	if len(bbytes) > 0 {
		for i, b := range bbytes {
			balance[len(bbytes)-i-1] = b
		}
	}
	if err = t.TryUpdate(balancekey[:], balance[:]); err != nil {
		return fmt.Errorf("updateStateObject (%x) error: %v", key, err)
	}
	if err = t.TryUpdate(ckkey[:], acc.CodeHash); err != nil {
		return fmt.Errorf("updateStateObject (%x) error: %v", key, err)
	}
	// TODO figure out if the code size needs to be updated, too

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
	return trie.root.Delete(key, func(h []byte) ([]byte, error) {
		return trie.db.DiskDB().Get(h)
	})
}

// Hash returns the root hash of the trie. It does not write to the database and
// can be used even if the trie doesn't have one.
func (trie *VerkleTrie) Hash() common.Hash {
	return trie.root.ComputeCommitment().Bytes()
}

func nodeToDBKey(n verkle.VerkleNode) []byte {
	ret := n.ComputeCommitment().Bytes()
	return ret[:]
}

// Commit writes all nodes to the trie's memory database, tracking the internal
// and external (for account tries) references.
func (trie *VerkleTrie) Commit(onleaf LeafCallback) (common.Hash, int, error) {
	flush := make(chan verkle.VerkleNode)
	go func() {
		trie.root.(*verkle.InternalNode).Flush(func(n verkle.VerkleNode) {
			if onleaf != nil {
				if leaf, isLeaf := n.(*verkle.LeafNode); isLeaf {
					for i := 0; i < verkle.NodeWidth; i++ {
						if leaf.Value(i) != nil {
							comm := n.ComputeCommitment().Bytes()
							onleaf(nil, nil, leaf.Value(i), common.BytesToHash(comm[:]))
						}
					}
				}
			}
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

func (trie *VerkleTrie) ProveAndSerialize(keys [][]byte, kv map[string][]byte) ([]byte, []verkle.KeyValuePair, error) {
	proof, _, _, _, err := verkle.MakeVerkleMultiProof(trie.root, keys, kv)
	if err != nil {
		return nil, nil, err
	}

	p, kvps, err := verkle.SerializeProof(proof)
	if err != nil {
		return nil, nil, err
	}

	return p, kvps, nil
}

type set = map[string]struct{}

func addKey(s set, key []byte) {
	s[string(key)] = struct{}{}
}

func DeserializeAndVerifyVerkleProof(serialized []byte, rootC *verkle.Point, keyvals []verkle.KeyValuePair) error {
	proof, cis, indices, yis, err := deserializeVerkleProof(serialized, rootC, keyvals)
	if err != nil {
		return fmt.Errorf("could not deserialize proof: %w", err)
	}
	cfg, err := verkle.GetConfig()
	if err != nil {
		return fmt.Errorf("could not get configuration %w", err)
	}
	if !verkle.VerifyVerkleProof(proof, cis, indices, yis, cfg) {
		return errInvalidProof
	}

	return nil
}

func deserializeVerkleProof(serialized []byte, rootC *verkle.Point, keyvals []verkle.KeyValuePair) (*verkle.Proof, []*verkle.Point, []byte, []*verkle.Fr, error) {
	var others set = set{} // Mark when an "other" stem has been seen

	proof, err := verkle.DeserializeProof(serialized, keyvals)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("verkle proof deserialization error: %w", err)
	}

	for _, stem := range proof.PoaStems {
		addKey(others, stem)
	}

	if len(proof.Keys) != len(proof.Values) {
		return nil, nil, nil, nil, fmt.Errorf("keys and values are of different length %d != %d", len(proof.Keys), len(proof.Values))
	}

	tree, err := verkle.TreeFromProof(proof, rootC)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("error rebuilding the tree from proof: %w", err)
	}
	for _, kv := range keyvals {
		val, err := tree.Get(kv.Key, nil)
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("could not find key %x in tree rebuilt from proof: %w", kv.Key, err)
		}

		if !bytes.Equal(val, kv.Value) {
			return nil, nil, nil, nil, fmt.Errorf("could not find correct value at %x in tree rebuilt from proof: %x != %x", kv.Key, val, kv.Value)
		}
	}

	pe, _, _ := tree.GetProofItems(proof.Keys)

	return proof, pe.Cis, pe.Zis, pe.Yis, nil
}

// ChunkedCode represents a sequence of 32-bytes chunks of code (31 bytes of which
// are actual code, and 1 byte is the pushdata offset).
type ChunkedCode []byte

// Copy the values here so as to avoid an import cycle
const (
	PUSH1  = byte(0x60)
	PUSH3  = byte(0x62)
	PUSH4  = byte(0x63)
	PUSH7  = byte(0x66)
	PUSH21 = byte(0x74)
	PUSH30 = byte(0x7d)
	PUSH32 = byte(0x7f)
)

// ChunkifyCode generates the chunked version of an array representing EVM bytecode
func ChunkifyCode(code []byte) (ChunkedCode, error) {
	var (
		chunkOffset = 0 // offset in the chunk
		chunkCount  = len(code) / 31
		codeOffset  = 0 // offset in the code
	)
	if len(code)%31 != 0 {
		chunkCount++
	}
	chunks := make([]byte, chunkCount*32)
	for i := 0; i < chunkCount; i++ {
		// number of bytes to copy, 31 unless
		// the end of the code has been reached.
		end := 31 * (i + 1)
		if len(code) < end {
			end = len(code)
		}

		// Copy the code itself
		copy(chunks[i*32+1:], code[31*i:end])

		// chunk offset = taken from the
		// last chunk.
		if chunkOffset > 31 {
			// skip offset calculation if push
			// data covers the whole chunk
			chunks[i*32] = 31
			chunkOffset = 1
			continue
		}
		chunks[32*i] = byte(chunkOffset)
		chunkOffset = 0

		// Check each instruction and update the offset
		// it should be 0 unless a PUSHn overflows.
		for ; codeOffset < end; codeOffset++ {
			if code[codeOffset] >= PUSH1 && code[codeOffset] <= PUSH32 {
				codeOffset += int(code[codeOffset] - PUSH1 + 1)
				if codeOffset+1 >= 31*(i+1) {
					codeOffset++
					chunkOffset = codeOffset - 31*(i+1)
					break
				}
			}
		}
	}

	return chunks, nil
}
