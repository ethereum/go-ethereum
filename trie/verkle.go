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

	"github.com/crate-crypto/go-ipa/bandersnatch/fr"
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

	var err error
	if err = t.TryUpdate(utils.GetTreeKeyVersion(key), []byte{0}); err != nil {
		return fmt.Errorf("updateStateObject (%x) error: %v", key, err)
	}
	var nonce [32]byte
	binary.LittleEndian.PutUint64(nonce[:], acc.Nonce)
	if err = t.TryUpdate(utils.GetTreeKeyNonce(key), nonce[:]); err != nil {
		return fmt.Errorf("updateStateObject (%x) error: %v", key, err)
	}
	var balance [32]byte
	bbytes := acc.Balance.Bytes()
	if len(bbytes) > 0 {
		for i, b := range bbytes {
			balance[len(bbytes)-i-1] = b
		}
	}
	if err = t.TryUpdate(utils.GetTreeKeyBalance(key), balance[:]); err != nil {
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
	proof, _, _, _ := verkle.MakeVerkleMultiProof(trie.root, keys, kv)
	p, kvps, err := verkle.SerializeProof(proof)
	if err != nil {
		return nil, nil, err
	}

	return p, kvps, nil
}

type set = map[string]struct{}

func hasKey(s set, key []byte) bool {
	_, ok := s[string(key)]
	return ok
}

func addKey(s set, key []byte) {
	s[string(key)] = struct{}{}
}

func DeserializeAndVerifyVerkleProof(serialized []byte) (map[common.Hash]common.Hash, error) {
	proof, cis, indices, yis, leaves, err := deserializeVerkleProof(serialized)
	if err != nil {
		return nil, fmt.Errorf("could not deserialize proof: %w", err)
	}
	cfg, err := verkle.GetConfig()
	if err != nil {
		return nil, err
	}
	if !verkle.VerifyVerkleProof(proof, cis, indices, yis, cfg) {
		return nil, errInvalidProof
	}

	return leaves, nil
}

func deserializeVerkleProof(serialized []byte) (*verkle.Proof, []*verkle.Point, []byte, []*verkle.Fr, map[common.Hash]common.Hash, error) {
	var (
		indices           []byte       // List of zis
		yis               []*verkle.Fr // List of yis
		seenIdx, seenComm set          // Mark when a zi/yi has already been seen in deserialization
		others            set          // Mark when an "other" stem has been seen
	)

	proof, err := verkle.DeserializeProof(serialized)
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("verkle proof deserialization error: %w", err)
	}

	for _, stem := range proof.PoaStems {
		addKey(others, stem)
	}

	keyvals := make(map[common.Hash]common.Hash)
	for i, key := range proof.Keys {
		keyvals[common.BytesToHash(key)] = common.BytesToHash(proof.Values[i])
	}

	if len(proof.Keys) != len(proof.Values) {
		return nil, nil, nil, nil, nil, fmt.Errorf("keys and values are of different length %d != %d", len(proof.Keys), len(proof.Values))
	}
	if len(proof.Keys) != len(proof.ExtStatus) {
		return nil, nil, nil, nil, nil, fmt.Errorf("keys and values are of different length %d != %d", len(proof.Keys), len(proof.Values))
	}

	// Rebuild the tree, creating nodes in the lexicographic order of their path
	lastcomm, lastpoa := 0, 0
	root := verkle.NewStateless()
	for i, es := range proof.ExtStatus {
		depth := es & 0x1F
		status := es >> 5
		node := root
		stem := proof.Keys[i]

		// go over the stem's bytes, in order to rebuild the internal nodes
		for j := byte(0); j < depth; j++ {
			// Recurse into the tree that is being rebuilt
			if node.Children()[stem[j]] == nil {
				node.SetChild(int(stem[j]), verkle.NewStatelessWithCommitment(proof.Cs[lastcomm]))
				lastcomm++
			}

			node = node.Children()[stem[j]].(*verkle.StatelessNode)

			// if that zi hasn't been encountered yet, add it to
			// the list of zis sorted by path.
			if !hasKey(seenIdx, stem[:j]) {
				addKey(seenIdx, stem[:j])
				indices = append(indices, stem[j])
			}

			// same thing with a yi
			if !hasKey(seenComm, stem[:j]) {
				addKey(seenComm, stem[:j])
				var yi fr.Element
				bytes := node.ComputeCommitment().Bytes()
				yi.SetBytesLE(bytes[:])
				yis = append(yis, &yi)
			}
		}

		// Reached the end, add the extension-and-suffix tree
		switch status {
		case 0:
			// missing stem, leave it as is
			break
		case 1:
			// another stem is found, build it
			node.SetStem(proof.PoaStems[lastpoa])
			lastpoa++
			break
		case 2:
			// stem is present
			node.SetStem(stem[:31])
			break
		default:
			return nil, nil, nil, nil, nil, fmt.Errorf("verkle proof deserialization error: invalid extension status %d", status)
		}

	}

	return proof, proof.Cs, indices, yis, keyvals, nil
}

// Copy the values here so as to avoid an import cycle
const (
	PUSH1  = 0x60
	PUSH32 = 0x7f
)

func ChunkifyCode(code []byte) ([][32]byte, error) {
	var (
		chunkOffset = 0 // offset in the chunk
		chunkCount  = len(code) / 31
		codeOffset  = 0 // offset in the code
	)
	if len(code)%31 != 0 {
		chunkCount++
	}
	chunks := make([][32]byte, chunkCount)
	for i := range chunks {
		// number of bytes to copy, 31 unless
		// the end of the code has been reached.
		end := 31 * (i + 1)
		if len(code) < end {
			end = len(code)
		}

		// Copy the code itself
		copy(chunks[i][1:], code[31*i:end])

		// chunk offset = taken from the
		// last chunk.
		if chunkOffset > 31 {
			// skip offset calculation if push
			// data covers the whole chunk
			chunks[i][0] = 31
			chunkOffset = 1
			continue
		}
		chunks[i][0] = byte(chunkOffset)
		chunkOffset = 0

		// Check each instruction and update the offset
		// it should be 0 unless a PUSHn overflows.
		for ; codeOffset < end; codeOffset++ {
			if code[codeOffset] >= PUSH1 && code[codeOffset] <= PUSH32 {
				codeOffset += int(code[codeOffset]) - PUSH1 + 1
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
