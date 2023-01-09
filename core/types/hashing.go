// Copyright 2021 The go-ethereum Authors
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

package types

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/bits"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"golang.org/x/crypto/sha3"
)

// hasherPool holds LegacyKeccak256 hashers for rlpHash.
var hasherPool = sync.Pool{
	New: func() interface{} { return sha3.NewLegacyKeccak256() },
}

// encodeBufferPool holds temporary encoder buffers for DeriveSha and TX encoding.
var encodeBufferPool = sync.Pool{
	New: func() interface{} { return new(bytes.Buffer) },
}

// rlpHash encodes x and hashes the encoded bytes.
func rlpHash(x interface{}) (h common.Hash) {
	sha := hasherPool.Get().(crypto.KeccakState)
	defer hasherPool.Put(sha)
	sha.Reset()
	rlp.Encode(sha, x)
	sha.Read(h[:])
	return h
}

// prefixedRlpHash writes the prefix into the hasher before rlp-encoding x.
// It's used for typed transactions.
func prefixedRlpHash(prefix byte, x interface{}) (h common.Hash) {
	sha := hasherPool.Get().(crypto.KeccakState)
	defer hasherPool.Put(sha)
	sha.Reset()
	sha.Write([]byte{prefix})
	rlp.Encode(sha, x)
	sha.Read(h[:])
	return h
}

// TrieHasher is the tool used to calculate the hash of derivable list.
// This is internal, do not use.
type TrieHasher interface {
	Reset()
	Update([]byte, []byte)
	Hash() common.Hash
}

// DerivableList is the input to DeriveSha.
// It is implemented by the 'Transactions' and 'Receipts' types.
// This is internal, do not use these methods.
type DerivableList interface {
	Len() int
	EncodeIndex(int, *bytes.Buffer)
}

func encodeForDerive(list DerivableList, i int, buf *bytes.Buffer) []byte {
	buf.Reset()
	list.EncodeIndex(i, buf)
	// It's really unfortunate that we need to do perform this copy.
	// StackTrie holds onto the values until Hash is called, so the values
	// written to it must not alias.
	return common.CopyBytes(buf.Bytes())
}

// DeriveSha creates the tree hashes of transactions and receipts in a block header.
func DeriveSha(list DerivableList, hasher TrieHasher) common.Hash {
	hasher.Reset()

	valueBuf := encodeBufferPool.Get().(*bytes.Buffer)
	defer encodeBufferPool.Put(valueBuf)

	// StackTrie requires values to be inserted in increasing hash order, which is not the
	// order that `list` provides hashes in. This insertion sequence ensures that the
	// order is correct.
	var indexBuf []byte
	for i := 1; i < list.Len() && i <= 0x7f; i++ {
		indexBuf = rlp.AppendUint64(indexBuf[:0], uint64(i))
		value := encodeForDerive(list, i, valueBuf)
		hasher.Update(indexBuf, value)
	}
	if list.Len() > 0 {
		indexBuf = rlp.AppendUint64(indexBuf[:0], 0)
		value := encodeForDerive(list, 0, valueBuf)
		hasher.Update(indexBuf, value)
	}
	for i := 0x80; i < list.Len(); i++ {
		indexBuf = rlp.AppendUint64(indexBuf[:0], uint64(i))
		value := encodeForDerive(list, i, valueBuf)
		hasher.Update(indexBuf, value)
	}
	return hasher.Hash()
}

func nextPowerOfTwo(v uint32) uint32 {
	v--
	v |= v >> 1
	v |= v >> 2
	v |= v >> 4
	v |= v >> 8
	v |= v >> 16
	v++
	return v
}

func MerkleizeAndMix(list DerivableList) common.Hash {
	return mixinLength(Merkleize(list), list.Len())
}

func Merkleize(list DerivableList) common.Hash {
	valueBuf := encodeBufferPool.Get().(*bytes.Buffer)
	defer encodeBufferPool.Put(valueBuf)

	sha := hasherPool.Get().(crypto.KeccakState)
	defer hasherPool.Put(sha)

	size := nextPowerOfTwo(uint32(list.Len()))
	roots := make([]common.Hash, size)
	for i := 0; i < list.Len(); i++ {
		valueBuf.Reset()
		list.EncodeIndex(i, valueBuf)
		roots[i] = chunkAndHashSSZ(valueBuf.Bytes())
	}
	return merkleize(roots)
}

func merkleize(values []common.Hash) common.Hash {
	// Check if len(values) is power of two
	if len(values)&(len(values)-1) != 0 || len(values) == 0 {
		panic(fmt.Sprintf("not a power of 2: %v", len(values)))
	}
	if len(values) == 1 {
		return values[0]
	}
	round := make([]common.Hash, len(values))
	copy(round, values)
	for i := 0; i < bits.Len(uint(len(values)))-1; i++ {
		for k := 0; k < len(round)/2; k++ {
			round[k] = crypto.Keccak256Hash(round[k*2][:], round[k*2+1][:])
		}
		round = round[0 : len(round)/2]
	}
	return round[0]
}

func chunkAndHashSSZ(input []byte) common.Hash {
	size := nextPowerOfTwo(uint32(len(input)))
	roots := make([]common.Hash, size)
	for start := 0; start < len(input); start += common.HashLength {
		end := start + common.HashLength
		if end > len(input) {
			end = len(input)
		}
		copy(roots[start/common.HashLength][:], input[start:end])
	}
	return mixinLength(merkleize(roots), len(input))
}

func mixinLength(root common.Hash, length int) (h common.Hash) {
	sha := hasherPool.Get().(crypto.KeccakState)
	defer hasherPool.Put(sha)

	var buf []byte
	binary.LittleEndian.PutUint64(buf, uint64(length))

	sha.Reset()
	sha.Write(root[:])
	sha.Write(buf[:])
	sha.Read(h[:])
	return h
}

var lookup map[int]common.Hash

func init() {
	// initialize the lookup
	max := 2 ^ 30
	base := [32]byte{}
	prev := crypto.Keccak256Hash(base[:])
	lookup[1] = prev
	for i := 2; i < max; i = i << 1 {
		prev = crypto.Keccak256Hash(prev.Bytes(), prev.Bytes())
		lookup[i] = prev
	}
}

func MerkleWithLimit(list DerivableList, limit int) (h common.Hash) {
	if limit == 0 {
		panic("should not happen")
	}
	if list.Len() <= limit/2 {
		// TODO use merkleize, change to bytes based,
		return Merkleize(list)
	}
	left := MerkleWithLimit(list, limit/2)
	right := lookup[limit/2]

	sha := hasherPool.Get().(crypto.KeccakState)
	defer hasherPool.Put(sha)

	sha.Reset()
	sha.Write(left[:])
	sha.Write(right[:])
	sha.Read(h[:])
	return h
}
