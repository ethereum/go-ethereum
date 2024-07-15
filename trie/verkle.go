// Copyright 2023 go-ethereum Authors
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
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/trie/utils"
	"github.com/ethereum/go-ethereum/triedb/database"
	"github.com/ethereum/go-verkle"
	"github.com/holiman/uint256"
)

var (
	zero               [32]byte
	errInvalidRootType = errors.New("invalid node type for root")
)

// VerkleTrie is a wrapper around VerkleNode that implements the trie.Trie
// interface so that Verkle trees can be reused verbatim.
type VerkleTrie struct {
	root   verkle.VerkleNode
	cache  *utils.PointCache
	reader *trieReader
}

// NewVerkleTrie constructs a verkle tree based on the specified root hash.
func NewVerkleTrie(root common.Hash, db database.Database, cache *utils.PointCache) (*VerkleTrie, error) {
	reader, err := newTrieReader(root, common.Hash{}, db)
	if err != nil {
		return nil, err
	}
	// Parse the root verkle node if it's not empty.
	node := verkle.New()
	if root != types.EmptyVerkleHash && root != types.EmptyRootHash {
		blob, err := reader.node(nil, common.Hash{})
		if err != nil {
			return nil, err
		}
		node, err = verkle.ParseNode(blob, 0)
		if err != nil {
			return nil, err
		}
	}
	return &VerkleTrie{
		root:   node,
		cache:  cache,
		reader: reader,
	}, nil
}

// GetKey returns the sha3 preimage of a hashed key that was previously used
// to store a value.
func (t *VerkleTrie) GetKey(key []byte) []byte {
	return key
}

// GetAccount implements state.Trie, retrieving the account with the specified
// account address. If the specified account is not in the verkle tree, nil will
// be returned. If the tree is corrupted, an error will be returned.
func (t *VerkleTrie) GetAccount(addr common.Address) (*types.StateAccount, error) {
	var (
		acc    = &types.StateAccount{}
		values [][]byte
		err    error
	)
	switch n := t.root.(type) {
	case *verkle.InternalNode:
		values, err = n.GetValuesAtStem(t.cache.GetStem(addr[:]), t.nodeResolver)
		if err != nil {
			return nil, fmt.Errorf("GetAccount (%x) error: %v", addr, err)
		}
	default:
		return nil, errInvalidRootType
	}
	if values == nil {
		return nil, nil
	}
	// Decode nonce in little-endian
	if len(values[utils.NonceLeafKey]) > 0 {
		acc.Nonce = binary.LittleEndian.Uint64(values[utils.NonceLeafKey])
	}
	// Decode balance in little-endian
	var balance [32]byte
	copy(balance[:], values[utils.BalanceLeafKey])
	for i := 0; i < len(balance)/2; i++ {
		balance[len(balance)-i-1], balance[i] = balance[i], balance[len(balance)-i-1]
	}
	acc.Balance = new(uint256.Int).SetBytes32(balance[:])

	// Decode codehash
	acc.CodeHash = values[utils.CodeKeccakLeafKey]

	// TODO account.Root is leave as empty. How should we handle the legacy account?
	return acc, nil
}

// GetStorage implements state.Trie, retrieving the storage slot with the specified
// account address and storage key. If the specified slot is not in the verkle tree,
// nil will be returned. If the tree is corrupted, an error will be returned.
func (t *VerkleTrie) GetStorage(addr common.Address, key []byte) ([]byte, error) {
	k := utils.StorageSlotKeyWithEvaluatedAddress(t.cache.Get(addr.Bytes()), key)
	val, err := t.root.Get(k, t.nodeResolver)
	if err != nil {
		return nil, err
	}
	return common.TrimLeftZeroes(val), nil
}

// UpdateAccount implements state.Trie, writing the provided account into the tree.
// If the tree is corrupted, an error will be returned.
func (t *VerkleTrie) UpdateAccount(addr common.Address, acc *types.StateAccount) error {
	var (
		err            error
		nonce, balance [32]byte
		values         = make([][]byte, verkle.NodeWidth)
	)
	values[utils.VersionLeafKey] = zero[:]
	values[utils.CodeKeccakLeafKey] = acc.CodeHash[:]

	// Encode nonce in little-endian
	binary.LittleEndian.PutUint64(nonce[:], acc.Nonce)
	values[utils.NonceLeafKey] = nonce[:]

	// Encode balance in little-endian
	bytes := acc.Balance.Bytes()
	for i, b := range bytes {
		balance[len(bytes)-i-1] = b
	}
	values[utils.BalanceLeafKey] = balance[:]

	switch n := t.root.(type) {
	case *verkle.InternalNode:
		err = n.InsertValuesAtStem(t.cache.GetStem(addr[:]), values, t.nodeResolver)
		if err != nil {
			return fmt.Errorf("UpdateAccount (%x) error: %v", addr, err)
		}
	default:
		return errInvalidRootType
	}
	// TODO figure out if the code size needs to be updated, too
	return nil
}

// UpdateStorage implements state.Trie, writing the provided storage slot into
// the tree. If the tree is corrupted, an error will be returned.
func (t *VerkleTrie) UpdateStorage(address common.Address, key, value []byte) error {
	// Left padding the slot value to 32 bytes.
	var v [32]byte
	if len(value) >= 32 {
		copy(v[:], value[:32])
	} else {
		copy(v[32-len(value):], value[:])
	}
	k := utils.StorageSlotKeyWithEvaluatedAddress(t.cache.Get(address.Bytes()), key)
	return t.root.Insert(k, v[:], t.nodeResolver)
}

// DeleteAccount implements state.Trie, deleting the specified account from the
// trie. If the account was not existent in the trie, no error will be returned.
// If the trie is corrupted, an error will be returned.
func (t *VerkleTrie) DeleteAccount(addr common.Address) error {
	var (
		err    error
		values = make([][]byte, verkle.NodeWidth)
	)
	for i := 0; i < verkle.NodeWidth; i++ {
		values[i] = zero[:]
	}
	switch n := t.root.(type) {
	case *verkle.InternalNode:
		err = n.InsertValuesAtStem(t.cache.GetStem(addr.Bytes()), values, t.nodeResolver)
		if err != nil {
			return fmt.Errorf("DeleteAccount (%x) error: %v", addr, err)
		}
	default:
		return errInvalidRootType
	}
	return nil
}

// RollBackAccount removes the account info + code from the tree, unlike DeleteAccount
// that will overwrite it with 0s. The first 64 storage slots are also removed.
func (t *VerkleTrie) RollBackAccount(addr common.Address) error {
	var (
		evaluatedAddr = t.cache.Get(addr.Bytes())
		codeSizeKey   = utils.CodeSizeKeyWithEvaluatedAddress(evaluatedAddr)
	)
	codeSizeBytes, err := t.root.Get(codeSizeKey, t.nodeResolver)
	if err != nil {
		return fmt.Errorf("rollback: error finding code size: %w", err)
	}
	if len(codeSizeBytes) == 0 {
		return errors.New("rollback: code size is not existent")
	}
	codeSize := binary.LittleEndian.Uint64(codeSizeBytes)

	// Delete the account header + first 64 slots + first 128 code chunks
	key := common.CopyBytes(codeSizeKey)
	for i := 0; i < verkle.NodeWidth; i++ {
		key[31] = byte(i)

		// this is a workaround to avoid deleting nil leaves, the lib needs to be
		// fixed to be able to handle that
		v, err := t.root.Get(key, t.nodeResolver)
		if err != nil {
			return fmt.Errorf("error rolling back account header: %w", err)
		}
		if len(v) == 0 {
			continue
		}
		_, err = t.root.Delete(key, t.nodeResolver)
		if err != nil {
			return fmt.Errorf("error rolling back account header: %w", err)
		}
	}
	// Delete all further code
	for i, chunknr := uint64(32*128), uint64(128); i < codeSize; i, chunknr = i+32, chunknr+1 {
		// evaluate group key at the start of a new group
		groupOffset := (chunknr + 128) % 256
		if groupOffset == 0 {
			key = utils.CodeChunkKeyWithEvaluatedAddress(evaluatedAddr, uint256.NewInt(chunknr))
		}
		key[31] = byte(groupOffset)
		_, err = t.root.Delete(key[:], t.nodeResolver)
		if err != nil {
			return fmt.Errorf("error deleting code chunk (addr=%x) error: %w", addr[:], err)
		}
	}
	return nil
}

// DeleteStorage implements state.Trie, deleting the specified storage slot from
// the trie. If the storage slot was not existent in the trie, no error will be
// returned. If the trie is corrupted, an error will be returned.
func (t *VerkleTrie) DeleteStorage(addr common.Address, key []byte) error {
	var zero [32]byte
	k := utils.StorageSlotKeyWithEvaluatedAddress(t.cache.Get(addr.Bytes()), key)
	return t.root.Insert(k, zero[:], t.nodeResolver)
}

// Hash returns the root hash of the tree. It does not write to the database and
// can be used even if the tree doesn't have one.
func (t *VerkleTrie) Hash() common.Hash {
	return t.root.Commit().Bytes()
}

// Commit writes all nodes to the tree's memory database.
func (t *VerkleTrie) Commit(_ bool) (common.Hash, *trienode.NodeSet) {
	root := t.root.(*verkle.InternalNode)
	nodes, err := root.BatchSerialize()
	if err != nil {
		// Error return from this function indicates error in the code logic
		// of BatchSerialize, and we fail catastrophically if this is the case.
		panic(fmt.Errorf("BatchSerialize failed: %v", err))
	}
	nodeset := trienode.NewNodeSet(common.Hash{})
	for _, node := range nodes {
		// Hash parameter is not used in pathdb
		nodeset.AddNode(node.Path, trienode.New(common.Hash{}, node.SerializedBytes))
	}
	// Serialize root commitment form
	return t.Hash(), nodeset
}

// NodeIterator implements state.Trie, returning an iterator that returns
// nodes of the trie. Iteration starts at the key after the given start key.
//
// TODO(gballet, rjl493456442) implement it.
func (t *VerkleTrie) NodeIterator(startKey []byte) (NodeIterator, error) {
	panic("not implemented")
}

// Prove implements state.Trie, constructing a Merkle proof for key. The result
// contains all encoded nodes on the path to the value at key. The value itself
// is also included in the last node and can be retrieved by verifying the proof.
//
// If the trie does not contain a value for key, the returned proof contains all
// nodes of the longest existing prefix of the key (at least the root), ending
// with the node that proves the absence of the key.
//
// TODO(gballet, rjl493456442) implement it.
func (t *VerkleTrie) Prove(key []byte, proofDb ethdb.KeyValueWriter) error {
	panic("not implemented")
}

// Copy returns a deep-copied verkle tree.
func (t *VerkleTrie) Copy() *VerkleTrie {
	return &VerkleTrie{
		root:   t.root.Copy(),
		cache:  t.cache,
		reader: t.reader,
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
		// number of bytes to copy, 31 unless the end of the code has been reached.
		end := 31 * (i + 1)
		if len(code) < end {
			end = len(code)
		}
		copy(chunks[i*32+1:], code[31*i:end]) // copy the code itself

		// chunk offset = taken from the last chunk.
		if chunkOffset > 31 {
			// skip offset calculation if push data covers the whole chunk
			chunks[i*32] = 31
			chunkOffset = 1
			continue
		}
		chunks[32*i] = byte(chunkOffset)
		chunkOffset = 0

		// Check each instruction and update the offset it should be 0 unless
		// a PUSH-N overflows.
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

// UpdateContractCode implements state.Trie, writing the provided contract code
// into the trie.
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
			key = utils.CodeChunkKeyWithEvaluatedAddress(t.cache.Get(addr.Bytes()), uint256.NewInt(chunknr))
		}
		values[groupOffset] = chunks[i : i+32]

		// Reuse the calculated key to also update the code size.
		if i == 0 {
			cs := make([]byte, 32)
			binary.LittleEndian.PutUint64(cs, uint64(len(code)))
			values[utils.CodeSizeLeafKey] = cs
		}
		if groupOffset == 255 || len(chunks)-i <= 32 {
			switch root := t.root.(type) {
			case *verkle.InternalNode:
				err = root.InsertValuesAtStem(key[:31], values, t.nodeResolver)
				if err != nil {
					return fmt.Errorf("UpdateContractCode (addr=%x) error: %w", addr[:], err)
				}
			default:
				return errInvalidRootType
			}
		}
	}
	return nil
}

func (t *VerkleTrie) ToDot() string {
	return verkle.ToDot(t.root)
}

func (t *VerkleTrie) nodeResolver(path []byte) ([]byte, error) {
	return t.reader.node(path, common.Hash{})
}

// Witness returns a set containing all trie nodes that have been accessed.
func (t *VerkleTrie) Witness() map[string]struct{} {
	panic("not implemented")
}
