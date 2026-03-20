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

// ChunkedCode represents a sequence of HashSize-byte chunks of code (StemSize bytes of which
// are actual code, and NodeTypeBytes byte is the pushdata offset).
type ChunkedCode []byte

// Copy the values here so as to avoid an import cycle
const (
	PUSH1  = byte(0x60)
	PUSH32 = byte(0x7f)
)

// ChunkifyCode generates the chunked version of an array representing EVM bytecode
// according to EIP-7864 specification.
func ChunkifyCode(code []byte) ChunkedCode {
	var (
		chunkOffset = 0
		chunkCount  = len(code) / StemSize
		codeOffset  = 0
	)
	if len(code)%StemSize != 0 {
		chunkCount++
	}
	chunks := make([]byte, chunkCount*HashSize)
	for i := 0; i < chunkCount; i++ {
		end := min(len(code), StemSize*(i+1))
		copy(chunks[i*HashSize+1:], code[StemSize*i:end])
		if chunkOffset > StemSize {
			chunks[i*HashSize] = StemSize
			chunkOffset = 1
			continue
		}
		chunks[HashSize*i] = byte(chunkOffset)
		chunkOffset = 0
		for ; codeOffset < end; codeOffset++ {
			if code[codeOffset] >= PUSH1 && code[codeOffset] <= PUSH32 {
				codeOffset += int(code[codeOffset] - PUSH1 + 1)
				if codeOffset+1 >= StemSize*(i+1) {
					codeOffset++
					chunkOffset = codeOffset - StemSize*(i+1)
					break
				}
			}
		}
	}
	return chunks
}

// BinaryTrie is the implementation of https://eips.ethereum.org/EIPS/eip-7864.
type BinaryTrie struct {
	store  *NodeStore
	root   NodeRef
	reader *trie.Reader
	tracer *trie.PrevalueTracer
}

// ToDot converts the binary trie to a DOT language representation. Useful for debugging.
func (t *BinaryTrie) ToDot() string {
	t.store.ComputeHash(t.root)
	return t.store.ToDot(t.root)
}

// NewBinaryTrie creates a new binary trie.
func NewBinaryTrie(root common.Hash, db database.NodeDatabase) (*BinaryTrie, error) {
	reader, err := trie.NewReader(root, common.Hash{}, db)
	if err != nil {
		return nil, err
	}
	t := &BinaryTrie{
		store:  NewNodeStore(),
		root:   EmptyRef,
		reader: reader,
		tracer: trie.NewPrevalueTracer(),
	}
	if root != types.EmptyBinaryHash && root != types.EmptyRootHash {
		blob, err := t.nodeResolver(nil, root)
		if err != nil {
			return nil, err
		}
		ref, err := t.store.DeserializeNodeWithHash(blob, 0, root)
		if err != nil {
			return nil, err
		}
		t.root = ref
	}
	return t, nil
}

// nodeResolver is a node resolver that reads nodes from the flatdb.
func (t *BinaryTrie) nodeResolver(path []byte, hash common.Hash) ([]byte, error) {
	if hash == (common.Hash{}) {
		return nil, nil
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
	return t.store.Get(t.root, key, t.nodeResolver)
}

// GetAccount returns the account information for the given address.
func (t *BinaryTrie) GetAccount(addr common.Address) (*types.StateAccount, error) {
	var (
		values [][]byte
		err    error
		acc    = &types.StateAccount{}
		key    = GetBinaryTreeKey(addr, zero[:])
	)
	switch t.root.Kind() {
	case KindInternal:
		values, err = t.store.GetValuesAtStem(t.root, key[:StemSize], t.nodeResolver)
	case KindStem:
		sn := t.store.getStem(t.root.Index())
		values = sn.Values
	case KindEmpty:
		return nil, nil
	default:
		return nil, errInvalidRootType
	}
	if err != nil {
		return nil, fmt.Errorf("GetAccount (%x) error: %v", addr, err)
	}

	emptyAccount := true
	for i := 0; values != nil && i <= CodeHashLeafKey && emptyAccount; i++ {
		emptyAccount = emptyAccount && values[i] == nil
	}
	if emptyAccount {
		return nil, nil
	}

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

// GetStorage returns the value for key stored in the trie.
func (t *BinaryTrie) GetStorage(addr common.Address, key []byte) ([]byte, error) {
	return t.store.Get(t.root, GetBinaryTreeKeyStorageSlot(addr, key), t.nodeResolver)
}

// UpdateAccount updates the account information for the given address.
func (t *BinaryTrie) UpdateAccount(addr common.Address, acc *types.StateAccount, codeLen int) error {
	var (
		err       error
		basicData [HashSize]byte
		values    = make([][]byte, StemNodeWidth)
		stem      = GetBinaryTreeKey(addr, zero[:])
	)
	binary.BigEndian.PutUint32(basicData[BasicDataCodeSizeOffset-1:], uint32(codeLen))
	binary.BigEndian.PutUint64(basicData[BasicDataNonceOffset:], acc.Nonce)

	balanceBytes := acc.Balance.Bytes()
	if len(balanceBytes) > 16 {
		balanceBytes = balanceBytes[16:]
	}
	copy(basicData[HashSize-len(balanceBytes):], balanceBytes[:])
	values[BasicDataLeafKey] = basicData[:]
	values[CodeHashLeafKey] = acc.CodeHash[:]

	t.root, err = t.store.InsertValuesAtStem(t.root, stem, values, t.nodeResolver, 0)
	return err
}

// UpdateStem updates the values for the given stem key.
func (t *BinaryTrie) UpdateStem(key []byte, values [][]byte) error {
	var err error
	t.root, err = t.store.InsertValuesAtStem(t.root, key, values, t.nodeResolver, 0)
	return err
}

// UpdateStorage associates key with value in the trie.
func (t *BinaryTrie) UpdateStorage(address common.Address, key, value []byte) error {
	k := GetBinaryTreeKeyStorageSlot(address, key)
	var v [HashSize]byte
	if len(value) >= HashSize {
		copy(v[:], value[:HashSize])
	} else {
		copy(v[HashSize-len(value):], value[:])
	}
	root, err := t.store.Insert(t.root, k, v[:], t.nodeResolver, 0)
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

// DeleteStorage removes any existing value for key from the trie.
func (t *BinaryTrie) DeleteStorage(addr common.Address, key []byte) error {
	k := GetBinaryTreeKeyStorageSlot(addr, key)
	var zero [HashSize]byte
	root, err := t.store.Insert(t.root, k, zero[:], t.nodeResolver, 0)
	if err != nil {
		return fmt.Errorf("DeleteStorage (%x) error: %v", addr, err)
	}
	t.root = root
	return nil
}

// Hash returns the root hash of the trie.
func (t *BinaryTrie) Hash() common.Hash {
	return t.store.ComputeHash(t.root)
}

// Commit writes all nodes to the trie's memory database.
func (t *BinaryTrie) Commit(_ bool) (common.Hash, *trienode.NodeSet) {
	nodeset := trienode.NewNodeSet(common.Hash{})
	err := t.store.CollectNodes(t.root, nil, func(path []byte, ref NodeRef) {
		serialized := t.store.SerializeNode(ref)
		nodeset.AddNode(path, trienode.NewNodeWithPrev(t.store.ComputeHash(ref), serialized, t.tracer.Get(path)))
	})
	if err != nil {
		panic(fmt.Errorf("CollectNodes failed: %v", err))
	}
	return t.Hash(), nodeset
}

// NodeIterator returns an iterator that returns nodes of the trie.
func (t *BinaryTrie) NodeIterator(startKey []byte) (trie.NodeIterator, error) {
	return newBinaryNodeIterator(t, nil)
}

// Prove constructs a Merkle proof for key.
func (t *BinaryTrie) Prove(key []byte, proofDb ethdb.KeyValueWriter) error {
	panic("not implemented")
}

// Copy creates a deep copy of the trie.
func (t *BinaryTrie) Copy() *BinaryTrie {
	return &BinaryTrie{
		store:  t.store.Copy(),
		root:   t.root,
		reader: t.reader,
		tracer: t.tracer.Copy(),
	}
}

// IsVerkle returns true if the trie is a Verkle tree.
func (t *BinaryTrie) IsVerkle() bool {
	return true
}

// UpdateContractCode updates the contract code into the trie.
func (t *BinaryTrie) UpdateContractCode(addr common.Address, codeHash common.Hash, code []byte) error {
	var (
		chunks = ChunkifyCode(code)
		values [][]byte
		key    []byte
		err    error
	)
	for i, chunknr := 0, uint64(0); i < len(chunks); i, chunknr = i+HashSize, chunknr+1 {
		groupOffset := (chunknr + 128) % StemNodeWidth
		if groupOffset == 0 || chunknr == 0 {
			values = make([][]byte, StemNodeWidth)
			var offset [HashSize]byte
			binary.BigEndian.PutUint64(offset[24:], chunknr+128)
			key = GetBinaryTreeKey(addr, offset[:])
		}
		values[groupOffset] = chunks[i : i+HashSize]

		if groupOffset == StemNodeWidth-1 || len(chunks)-i <= HashSize {
			err = t.UpdateStem(key[:StemSize], values)
			if err != nil {
				return fmt.Errorf("UpdateContractCode (addr=%x) error: %w", addr[:], err)
			}
		}
	}
	return nil
}

// PrefetchAccount attempts to resolve specific accounts from the database.
func (t *BinaryTrie) PrefetchAccount(addresses []common.Address) error {
	for _, addr := range addresses {
		if _, err := t.GetAccount(addr); err != nil {
			return err
		}
	}
	return nil
}

// PrefetchStorage attempts to resolve specific storage slots from the database.
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
	return t.tracer.Values()
}
