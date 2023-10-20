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
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/trie/utils"
	"github.com/gballet/go-verkle"
	"github.com/holiman/uint256"
)

// VerkleTrie is a wrapper around VerkleNode that implements the trie.Trie
// interface so that Verkle trees can be reused verbatim.
type VerkleTrie struct {
	root       verkle.VerkleNode
	db         *Database
	pointCache *utils.PointCache
	ended      bool
	rootHash   common.Hash
	reader     *trieReader
}

func (t *VerkleTrie) ToDot() string {
	return verkle.ToDot(t.root)
}

func NewVerkleTrie(rootHash common.Hash, root verkle.VerkleNode, db *Database, pointCache *utils.PointCache, ended bool) (*VerkleTrie, error) {
	reader, err := newTrieReader(rootHash, common.Hash{}, db)
	if err != nil {
		return nil, err
	}
	return &VerkleTrie{
		root:       root,
		db:         db,
		pointCache: pointCache,
		ended:      ended,
		rootHash:   rootHash,
		reader:     reader,
	}, nil
}

func (t *VerkleTrie) FlatdbNodeResolver(path []byte) ([]byte, error) {
	return t.reader.reader.Node(t.reader.owner, path, common.Hash{})
}

var errInvalidRootType = errors.New("invalid node type for root")

// GetKey returns the sha3 preimage of a hashed key that was previously used
// to store a value.
func (t *VerkleTrie) GetKey(key []byte) []byte {
	return key
}

// GetStorage returns the value for key stored in the trie. The value bytes
// must not be modified by the caller. If a node was not found in the database,
// a trie.MissingNodeError is returned.
func (t *VerkleTrie) GetStorage(addr common.Address, key []byte) ([]byte, error) {
	pointEval := t.pointCache.GetTreeKeyHeader(addr[:])
	k := utils.GetTreeKeyStorageSlotWithEvaluatedAddress(pointEval, key)
	return t.root.Get(k, t.FlatdbNodeResolver)
}

// GetWithHashedKey returns the value, assuming that the key has already
// been hashed.
func (t *VerkleTrie) GetWithHashedKey(key []byte) ([]byte, error) {
	return t.root.Get(key, t.FlatdbNodeResolver)
}

func (t *VerkleTrie) GetAccount(addr common.Address) (*types.StateAccount, error) {
	acc := &types.StateAccount{}
	versionkey := t.pointCache.GetTreeKeyVersionCached(addr[:])
	var (
		values [][]byte
		err    error
	)
	switch t.root.(type) {
	case *verkle.InternalNode:
		values, err = t.root.(*verkle.InternalNode).GetStem(versionkey[:31], t.FlatdbNodeResolver)
	default:
		return nil, errInvalidRootType
	}
	if err != nil {
		return nil, fmt.Errorf("GetAccount (%x) error: %v", addr, err)
	}

	if values == nil {
		return nil, nil
	}
	if len(values[utils.NonceLeafKey]) > 0 {
		acc.Nonce = binary.LittleEndian.Uint64(values[utils.NonceLeafKey])
	}

	var balance [32]byte
	copy(balance[:], values[utils.BalanceLeafKey])
	for i := 0; i < len(balance)/2; i++ {
		balance[len(balance)-i-1], balance[i] = balance[i], balance[len(balance)-i-1]
	}
	acc.Balance = new(big.Int).SetBytes(balance[:])
	acc.CodeHash = values[utils.CodeKeccakLeafKey]

	return acc, nil
}

var zero [32]byte

func (t *VerkleTrie) UpdateAccount(addr common.Address, acc *types.StateAccount) error {
	var (
		err            error
		nonce, balance [32]byte
		values         = make([][]byte, verkle.NodeWidth)
		stem           = t.pointCache.GetTreeKeyVersionCached(addr[:])
	)

	// Only evaluate the polynomial once
	values[utils.VersionLeafKey] = zero[:]
	values[utils.NonceLeafKey] = nonce[:]
	values[utils.BalanceLeafKey] = balance[:]
	values[utils.CodeKeccakLeafKey] = acc.CodeHash[:]

	binary.LittleEndian.PutUint64(nonce[:], acc.Nonce)
	bbytes := acc.Balance.Bytes()
	if len(bbytes) > 0 {
		for i, b := range bbytes {
			balance[len(bbytes)-i-1] = b
		}
	}

	switch root := t.root.(type) {
	case *verkle.InternalNode:
		err = root.InsertStem(stem, values, t.FlatdbNodeResolver)
	default:
		return errInvalidRootType
	}
	if err != nil {
		return fmt.Errorf("UpdateAccount (%x) error: %v", addr, err)
	}
	// TODO figure out if the code size needs to be updated, too

	return nil
}

func (t *VerkleTrie) UpdateStem(key []byte, values [][]byte) error {
	switch root := t.root.(type) {
	case *verkle.InternalNode:
		return root.InsertStem(key, values, t.FlatdbNodeResolver)
	default:
		panic("invalid root type")
	}
}

// UpdateStorage associates key with value in the trie. If value has length zero,
// any existing value is deleted from the trie. The value bytes must not be modified
// by the caller while they are stored in the trie. If a node was not found in the
// database, a trie.MissingNodeError is returned.
func (t *VerkleTrie) UpdateStorage(address common.Address, key, value []byte) error {
	k := utils.GetTreeKeyStorageSlotWithEvaluatedAddress(t.pointCache.GetTreeKeyHeader(address[:]), key)
	var v [32]byte
	if len(value) >= 32 {
		copy(v[:], value[:32])
	} else {
		copy(v[32-len(value):], value[:])
	}
	return t.root.Insert(k, v[:], t.FlatdbNodeResolver)
}

func (t *VerkleTrie) DeleteAccount(addr common.Address) error {
	var (
		err    error
		values = make([][]byte, verkle.NodeWidth)
		stem   = t.pointCache.GetTreeKeyVersionCached(addr[:])
	)

	for i := 0; i < verkle.NodeWidth; i++ {
		values[i] = zero[:]
	}

	switch root := t.root.(type) {
	case *verkle.InternalNode:
		err = root.InsertStem(stem, values, t.FlatdbNodeResolver)
	default:
		return errInvalidRootType
	}
	if err != nil {
		return fmt.Errorf("DeleteAccount (%x) error: %v", addr, err)
	}
	// TODO figure out if the code size needs to be updated, too

	return nil
}

// DeleteStorage removes any existing value for key from the trie. If a node was
// not found in the database, a trie.MissingNodeError is returned.
func (t *VerkleTrie) DeleteStorage(addr common.Address, key []byte) error {
	pointEval := t.pointCache.GetTreeKeyHeader(addr[:])
	k := utils.GetTreeKeyStorageSlotWithEvaluatedAddress(pointEval, key)
	var zero [32]byte
	return t.root.Insert(k, zero[:], t.FlatdbNodeResolver)
}

// Hash returns the root hash of the trie. It does not write to the database and
// can be used even if the trie doesn't have one.
func (t *VerkleTrie) Hash() common.Hash {
	return t.root.Commit().Bytes()
}

func nodeToDBKey(n verkle.VerkleNode) []byte {
	ret := n.Commitment().Bytes()
	return ret[:]
}

// Commit writes all nodes to the trie's memory database, tracking the internal
// and external (for account tries) references.
func (t *VerkleTrie) Commit(_ bool) (common.Hash, *trienode.NodeSet, error) {
	root, ok := t.root.(*verkle.InternalNode)
	if !ok {
		return common.Hash{}, nil, errors.New("unexpected root node type")
	}
	nodes, err := root.BatchSerialize()
	if err != nil {
		return common.Hash{}, nil, fmt.Errorf("serializing tree nodes: %s", err)
	}

	nodeset := trienode.NewNodeSet(common.Hash{})
	for _, node := range nodes {
		// hash parameter is not used in pathdb
		nodeset.AddNode(node.Path, trienode.New(common.Hash{}, node.SerializedBytes))
	}

	// Serialize root commitment form
	t.rootHash = t.Hash()
	return t.rootHash, nodeset, nil
}

// NodeIterator returns an iterator that returns nodes of the trie. Iteration
// starts at the key after the given start key.
func (t *VerkleTrie) NodeIterator(startKey []byte) (NodeIterator, error) {
	return newVerkleNodeIterator(t, nil)
}

// Prove constructs a Merkle proof for key. The result contains all encoded nodes
// on the path to the value at key. The value itself is also included in the last
// node and can be retrieved by verifying the proof.
//
// If the trie does not contain a value for key, the returned proof contains all
// nodes of the longest existing prefix of the key (at least the root), ending
// with the node that proves the absence of the key.
func (t *VerkleTrie) Prove(key []byte, proofDb ethdb.KeyValueWriter) error {
	panic("not implemented")
}

func (t *VerkleTrie) Copy() *VerkleTrie {
	return &VerkleTrie{
		root:       t.root.Copy(),
		db:         t.db,
		pointCache: t.pointCache,
		reader:     t.reader,
	}
}

// IsVerkle indicates if the trie is a Verkle trie.
func (t *VerkleTrie) IsVerkle() bool {
	return true
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
func ChunkifyCode(code []byte) ChunkedCode {
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

	return chunks
}

func (t *VerkleTrie) UpdateContractCode(addr common.Address, codeHash common.Hash, code []byte) error {
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
			key = utils.GetTreeKeyCodeChunkWithEvaluatedAddress(t.pointCache.GetTreeKeyHeader(addr[:]), uint256.NewInt(chunknr))
		}
		values[groupOffset] = chunks[i : i+32]

		// Reuse the calculated key to also update the code size.
		if i == 0 {
			cs := make([]byte, 32)
			binary.LittleEndian.PutUint64(cs, uint64(len(code)))
			values[utils.CodeSizeLeafKey] = cs
		}

		if groupOffset == 255 || len(chunks)-i <= 32 {
			err = t.UpdateStem(key[:31], values)

			if err != nil {
				return fmt.Errorf("UpdateContractCode (addr=%x) error: %w", addr[:], err)
			}
		}
	}
	return nil
}
