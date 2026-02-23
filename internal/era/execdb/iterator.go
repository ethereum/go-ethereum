// Copyright 2025 The go-ethereum Authors
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

package execdb

import (
	"errors"
	"io"
	"math/big"
	"slices"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/internal/era"
	"github.com/ethereum/go-ethereum/internal/era/e2store"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/klauspost/compress/snappy"
)

type Iterator struct {
	inner *RawIterator
	block *types.Block // cache for decoded block
}

// NewIterator returns a header/body/receipt iterator over the archive.
// Call Next immediately to position on the first block.
func NewIterator(e era.Era) (era.Iterator, error) {
	inner, err := NewRawIterator(e.(*Era))
	if err != nil {
		return nil, err
	}
	return &Iterator{inner: inner}, nil
}

// Next advances to the next block entry.
func (it *Iterator) Next() bool {
	it.block = nil
	return it.inner.Next()
}

// Number is the number of the block currently loaded.
func (it *Iterator) Number() uint64 { return it.inner.next - 1 }

// Error returns any iteration error (EOF is reported as nil, identical
// to the Era‑1 iterator behaviour).
func (it *Iterator) Error() error { return it.inner.Error() }

// Block decodes the current header+body into a *types.Block.
func (it *Iterator) Block() (*types.Block, error) {
	if it.block != nil {
		return it.block, nil
	}
	if it.inner.Header == nil || it.inner.Body == nil {
		return nil, errors.New("header and body must be non‑nil")
	}
	var (
		h types.Header
		b types.Body
	)
	if err := rlp.Decode(it.inner.Header, &h); err != nil {
		return nil, err
	}
	if err := rlp.Decode(it.inner.Body, &b); err != nil {
		return nil, err
	}
	it.block = types.NewBlockWithHeader(&h).WithBody(b)
	return it.block, nil
}

// Receipts decodes receipts for the current block.
func (it *Iterator) Receipts() (types.Receipts, error) {
	block, err := it.Block()
	if err != nil {
		return nil, err
	}
	if it.inner.Receipts == nil {
		return nil, errors.New("receipts must be non‑nil")
	}
	var rs []*types.SlimReceipt
	if err := rlp.Decode(it.inner.Receipts, &rs); err != nil {
		return nil, err
	}
	if len(rs) != len(block.Transactions()) {
		return nil, errors.New("number of txs does not match receipts")
	}
	receipts := make([]*types.Receipt, len(rs))
	for i, receipt := range rs {
		receipts[i] = (*types.Receipt)(receipt)
		receipts[i].Bloom = types.CreateBloom(receipts[i])
	}
	return receipts, nil
}

// BlockAndReceipts is a convenience wrapper.
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

// TotalDifficulty returns the TD at the current position (if present).
func (it *Iterator) TotalDifficulty() (*big.Int, error) {
	if it.inner.TotalDifficulty == nil {
		return nil, errors.New("total‑difficulty stream is nil")
	}
	tdBytes, err := io.ReadAll(it.inner.TotalDifficulty)
	if err != nil {
		return nil, err
	}
	slices.Reverse(tdBytes)
	return new(big.Int).SetBytes(tdBytes), nil
}

// -----------------------------------------------------------------------------
// Low‑level iterator (raw TLV/offset handling, no decoding)
// -----------------------------------------------------------------------------

type RawIterator struct {
	e    *Era
	next uint64 // next block to pull
	err  error

	Header          io.Reader
	Body            io.Reader
	Receipts        io.Reader
	TotalDifficulty io.Reader // nil when archive omits TD
}

// NewRawIterator creates an iterator positioned *before* the first block.
func NewRawIterator(e *Era) (*RawIterator, error) {
	return &RawIterator{e: e, next: e.m.start}, nil
}

// Next loads the next block’s components; returns false on EOF or error.
func (it *RawIterator) Next() bool {
	it.err = nil // clear previous error

	if it.next >= it.e.m.start+it.e.m.count {
		it.clear()
		return false
	}

	headerOffset, err := it.e.headerOff(it.next)
	if err != nil {
		it.setErr(err)
		return false
	}
	it.Header, _, err = newSnappyReader(it.e.s, era.TypeCompressedHeader, headerOffset)
	if err != nil {
		it.setErr(err)
		return false
	}

	bodyOffset, err := it.e.bodyOff(it.next)
	if err != nil {
		it.setErr(err)
		return false
	}
	it.Body, _, err = newSnappyReader(it.e.s, era.TypeCompressedBody, bodyOffset)
	if err != nil {
		it.setErr(err)
		return false
	}

	receiptsOffset, err := it.e.receiptOff(it.next)
	if err != nil {
		it.setErr(err)
		return false
	}
	it.Receipts, _, err = newSnappyReader(it.e.s, era.TypeCompressedSlimReceipts, receiptsOffset)
	if err != nil {
		it.setErr(err)
		return false
	}

	// Check if TD component is present in this file (pre-merge or merge-transition epoch).
	if int(td) < int(it.e.m.components) {
		tdOffset, err := it.e.tdOff(it.next)
		if err != nil {
			it.setErr(err)
			return false
		}
		it.TotalDifficulty, _, err = it.e.s.ReaderAt(era.TypeTotalDifficulty, tdOffset)
		if err != nil {
			it.setErr(err)
			return false
		}
	} else {
		it.TotalDifficulty = nil
	}

	it.next++
	return true
}

func (it *RawIterator) Number() uint64 { return it.next - 1 }

func (it *RawIterator) Error() error {
	if it.err == io.EOF {
		return nil
	}
	return it.err
}

func (it *RawIterator) setErr(err error) {
	it.err = err
	it.clear()
}

func (it *RawIterator) clear() {
	it.Header, it.Body, it.Receipts, it.TotalDifficulty = nil, nil, nil, nil
}

// newSnappyReader behaves like era.newSnappyReader: returns a snappy.Reader
// plus the length of the underlying TLV payload so callers can advance offsets.
func newSnappyReader(r *e2store.Reader, typ uint16, off int64) (io.Reader, int64, error) {
	raw, n, err := r.ReaderAt(typ, off)
	if err != nil {
		return nil, 0, err
	}
	return snappy.NewReader(raw), int64(n), nil
}
