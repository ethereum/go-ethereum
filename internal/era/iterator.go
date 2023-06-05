// Copyright 2023 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package era

import (
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

// Iterator wraps RawIterator and returns decoded Era1 entries.
type Iterator struct {
	inner *RawIterator
}

// NewIterator returns a new Iterator instance.
func NewIterator(e *Era) (*Iterator, error) {
	inner, err := NewRawIterator(e)
	if err != nil {
		return nil, err
	}
	return &Iterator{inner}, nil
}

// Next moves the iterator to the next block entry.
func (it *Iterator) Next() bool {
	return it.inner.Next()
}

// Number returns the current number block the iterator will return.
func (it *Iterator) Number() uint64 {
	return it.inner.next - 1
}

// Error returns the error status of the iterator.
func (it *Iterator) Error() error {
	return it.inner.Error()
}

// Block returns the block for the iterator's current position.
func (it *Iterator) Block() (*types.Block, error) {
	var (
		header types.Header
		body   types.Body
	)
	if err := rlp.Decode(it.inner.Header, &header); err != nil {
		return nil, err
	}
	if err := rlp.Decode(it.inner.Body, &body); err != nil {
		return nil, err
	}
	return types.NewBlockWithHeader(&header).WithBody(body.Transactions, body.Uncles), nil
}

// Receipts returns the receipts for the iterator's current position.
func (it *Iterator) Receipts() (types.Receipts, error) {
	var receipts types.Receipts
	err := rlp.Decode(it.inner.Receipts, &receipts)
	return receipts, err
}

// BlockAndReceipts returns the block and receipts for the iterator's current
// position.
func (it *Iterator) BlockAndReceipts() (*types.Block, types.Receipts, error) {
	b, err := it.Block()
	if err != nil {
		return nil, nil, err
	}
	r, err := it.Receipts()
	if err != nil {
		return nil, nil, err
	}
	return b, r, nil
}

// TotalDifficulty returns the total difficulty for the iterator's current
// position.
func (it *Iterator) TotalDifficulty() (*big.Int, error) {
	var td []byte
	if err := rlp.Decode(it.inner.TotalDifficulty, td); err != nil {
		return nil, err
	}
	return new(big.Int).SetBytes(reverseOrder(td)), nil
}

// RawIterator reads an RLP-encode Era1 entries.
type RawIterator struct {
	e    *Era   // backing Era1
	next uint64 // next block to read
	err  error  // last error

	Header          io.Reader
	Body            io.Reader
	Receipts        io.Reader
	TotalDifficulty io.Reader
}

// NewRawIterator returns a new Iterator instance.
func NewRawIterator(e *Era) (*RawIterator, error) {
	return &RawIterator{
		e:    e,
		next: e.m.start,
	}, nil
}

// Next moves the iterator to the next block entry.
func (it *RawIterator) Next() bool {
	if it.e.m.start+it.e.m.count <= it.next {
		return false
	}
	off, err := it.e.readOffset(it.next)
	if err == io.EOF {
		it.err = err
		return false
	}
	var n int
	if it.Header, n, it.err = newSnappyReader(it.e.s, off); it.err != nil {
		return true
	}
	off += int64(n)
	if it.Body, n, it.err = newSnappyReader(it.e.s, off); it.err != nil {
		return true
	}
	off += int64(n)
	if it.Receipts, n, it.err = newSnappyReader(it.e.s, off); it.err != nil {
		return true
	}
	off += int64(n)
	if it.TotalDifficulty, _, it.err = newReader(it.e.s, off); it.err != nil {
		return true
	}
	it.next += 1
	return true
}

// Number returns the current number block the iterator will return.
func (it *RawIterator) Number() uint64 {
	return it.next - 1
}

// Error returns the error status of the iterator.
func (it *RawIterator) Error() error {
	if it.err == io.EOF {
		return nil
	}
	return it.err
}
