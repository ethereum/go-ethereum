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
	"fmt"
	"math"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

// hasherPool holds LegacyKeccak256 buffer for rlpHash.
var hasherPool = sync.Pool{
	New: func() interface{} { return crypto.NewKeccakState() },
}

// encodeBufferPool holds temporary encoder buffers for DeriveSha and TX encoding.
var encodeBufferPool = sync.Pool{
	New: func() interface{} { return new(bytes.Buffer) },
}

// getPooledBuffer retrieves a buffer from the pool and creates a byte slice of the
// requested size from it.
//
// The caller should return the *bytes.Buffer object back into encodeBufferPool after use!
// The returned byte slice must not be used after returning the buffer.
func getPooledBuffer(size uint64) ([]byte, *bytes.Buffer, error) {
	if size > math.MaxInt {
		return nil, nil, fmt.Errorf("can't get buffer of size %d", size)
	}
	buf := encodeBufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	buf.Grow(int(size))
	b := buf.Bytes()[:int(size)]
	return b, buf, nil
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

// ListHasher defines the interface for computing the hash of a derivable list.
type ListHasher interface {
	// Reset clears the internal state of the hasher, preparing it for reuse.
	Reset()

	// Update inserts the given key-value pair into the hasher.
	// The implementation must copy the provided slices, allowing the caller
	// to safely modify them after the call returns.
	Update(key []byte, value []byte) error

	// Hash computes and returns the final hash of all inserted key-value pairs.
	Hash() common.Hash
}

// DerivableList is the input to DeriveSha.
// It is implemented by the 'Transactions' and 'Receipts' types.
// This is internal, do not use these methods.
type DerivableList interface {
	Len() int
	EncodeIndex(int, *bytes.Buffer)
}

// encodeForDerive encodes the element in the list at the position i into the buffer.
func encodeForDerive(list DerivableList, i int, buf *bytes.Buffer) []byte {
	buf.Reset()
	list.EncodeIndex(i, buf)
	return buf.Bytes()
}

// DeriveSha creates the tree hashes of transactions, receipts, and withdrawals in a block header.
func DeriveSha(list DerivableList, hasher ListHasher) common.Hash {
	hasher.Reset()

	// Allocate a buffer for value encoding. As the hasher is claimed that all
	// supplied key value pairs will be copied by hasher and safe to reuse the
	// encoding buffer.
	valueBuf := encodeBufferPool.Get().(*bytes.Buffer)
	defer encodeBufferPool.Put(valueBuf)

	// StackTrie requires values to be inserted in increasing hash order, which is not the
	// order that `list` provides hashes in. This insertion sequence ensures that the
	// order is correct.
	//
	// The error returned by hasher is omitted because hasher will produce an incorrect
	// hash in case any error occurs.
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

// ListHashStream computes the hash DeriveSha computes, for a list that is
// filled in one element at a time. The block processor uses it to hash the
// receipts of a block while the remaining transactions are still executing.
type ListHashStream struct {
	hasher   ListHasher
	valueBuf *bytes.Buffer
	indexBuf []byte
	first    []byte // encoding of the element at index zero, inserted last
	count    int
}

// NewListHashStream creates a stream feeding the given hasher.
func NewListHashStream(hasher ListHasher) *ListHashStream {
	hasher.Reset()
	return &ListHashStream{
		hasher:   hasher,
		valueBuf: new(bytes.Buffer),
	}
}

// Update inserts the next element of the list into the hasher. The list only
// has to hold the element the stream is up to.
func (s *ListHashStream) Update(list DerivableList) {
	i := s.count
	s.count++

	// StackTrie wants the keys in increasing hash order, which puts the element
	// at index zero after the one at index 0x7f. Keep its encoding aside until
	// the list gets there, the same insertion sequence DeriveSha uses.
	if i == 0 {
		s.first = append(s.first[:0], encodeForDerive(list, 0, s.valueBuf)...)
		return
	}
	s.insert(list, i)
	if i == 0x7f {
		s.insertFirst()
	}
}

// Hash inserts whatever the stream kept aside and returns the hash of
// everything it was given.
func (s *ListHashStream) Hash() common.Hash {
	if s.count > 0 && s.count <= 0x7f {
		s.insertFirst()
	}
	return s.hasher.Hash()
}

// insert encodes the element at index i and updates the hasher with it. The
// hasher error is dropped for the same reason DeriveSha drops it, a failing
// hasher produces a wrong hash anyway.
func (s *ListHashStream) insert(list DerivableList, i int) {
	s.indexBuf = rlp.AppendUint64(s.indexBuf[:0], uint64(i))
	s.hasher.Update(s.indexBuf, encodeForDerive(list, i, s.valueBuf))
}

// insertFirst inserts the element kept aside at index zero.
func (s *ListHashStream) insertFirst() {
	s.indexBuf = rlp.AppendUint64(s.indexBuf[:0], 0)
	s.hasher.Update(s.indexBuf, s.first)
}
