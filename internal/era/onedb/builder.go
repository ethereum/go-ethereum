// Copyright 2024 The go-ethereum Authors
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

package onedb

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/internal/era"
	"github.com/ethereum/go-ethereum/internal/era/e2store"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/golang/snappy"
)

// Builder is used to create Era1 archives of block data.
//
// Era1 files are themselves e2store files. For more information on this format,
// see https://github.com/status-im/nimbus-eth2/blob/stable/docs/e2store.md.
//
// The overall structure of an Era1 file follows closely the structure of an Era file
// which contains consensus Layer data (and as a byproduct, EL data after the merge).
//
// The structure can be summarized through this definition:
//
//	era1 := Version | block-tuple* | other-entries* | Accumulator | BlockIndex
//	block-tuple :=  CompressedHeader | CompressedBody | CompressedReceipts | TotalDifficulty
//
// Each basic element is its own entry:
//
//	Version            = { type: [0x65, 0x32], data: nil }
//	CompressedHeader   = { type: [0x03, 0x00], data: snappyFramed(rlp(header)) }
//	CompressedBody     = { type: [0x04, 0x00], data: snappyFramed(rlp(body)) }
//	CompressedReceipts = { type: [0x05, 0x00], data: snappyFramed(rlp(receipts)) }
//	TotalDifficulty    = { type: [0x06, 0x00], data: uint256(header.total_difficulty) }
//	AccumulatorRoot    = { type: [0x07, 0x00], data: accumulator-root }
//	BlockIndex         = { type: [0x32, 0x66], data: block-index }
//
// Accumulator is computed by constructing an SSZ list of header-records of length at most
// 8192 and then calculating the hash_tree_root of that list.
//
//	header-record := { block-hash: Bytes32, total-difficulty: Uint256 }
//	accumulator   := hash_tree_root([]header-record, 8192)
//
// BlockIndex stores relative offsets to each compressed block entry. The
// format is:
//
//	block-index := starting-number | index | index | index ... | count
//
// starting-number is the first block number in the archive. Every index is a
// defined relative to beginning of the record. The total number of block
// entries in the file is recorded with count.
//
// Due to the accumulator size limit of 8192, the maximum number of blocks in
// an Era1 batch is also 8192.
type Builder struct {
	w           *e2store.Writer
	startNum    *uint64
	startTd     *big.Int
	indexes     []uint64
	hashes      []common.Hash
	tds         []*big.Int
	accumulator *common.Hash // accumulator root, set by Finalize

	written int

	buf    *bytes.Buffer
	snappy *snappy.Writer
}

// NewBuilder returns a new Builder instance.
func NewBuilder(w io.Writer) era.Builder {
	buf := bytes.NewBuffer(nil)
	return &Builder{
		w:      e2store.NewWriter(w),
		buf:    buf,
		snappy: snappy.NewBufferedWriter(buf),
	}
}

// Add writes a compressed block entry and compressed receipts entry to the
// underlying e2store file.
func (b *Builder) Add(block *types.Block, receipts types.Receipts, td *big.Int) error {
	eh, err := rlp.EncodeToBytes(block.Header())
	if err != nil {
		return err
	}
	eb, err := rlp.EncodeToBytes(block.Body())
	if err != nil {
		return err
	}
	er, err := rlp.EncodeToBytes(receipts)
	if err != nil {
		return err
	}
	return b.AddRLP(eh, eb, er, block.NumberU64(), block.Hash(), td, block.Difficulty())
}

// AddRLP writes a compressed block entry and compressed receipts entry to the
// underlying e2store file.
func (b *Builder) AddRLP(header, body, receipts []byte, number uint64, hash common.Hash, td, difficulty *big.Int) error {
	// Write Era1 version entry before first block.
	if b.startNum == nil {
		n, err := b.w.Write(era.TypeVersion, nil)
		if err != nil {
			return err
		}
		startNum := number
		b.startNum = &startNum
		b.startTd = new(big.Int).Sub(td, difficulty)
		b.written += n
	}
	if len(b.indexes) >= era.MaxSize {
		return fmt.Errorf("exceeds maximum batch size of %d", era.MaxSize)
	}

	b.indexes = append(b.indexes, uint64(b.written))
	b.hashes = append(b.hashes, hash)
	b.tds = append(b.tds, td)

	// Write block data.
	if err := b.snappyWrite(era.TypeCompressedHeader, header); err != nil {
		return err
	}
	if err := b.snappyWrite(era.TypeCompressedBody, body); err != nil {
		return err
	}
	if err := b.snappyWrite(era.TypeCompressedReceipts, receipts); err != nil {
		return err
	}

	// Also write total difficulty, but don't snappy encode.
	btd := era.BigToBytes32(td)
	n, err := b.w.Write(era.TypeTotalDifficulty, btd[:])
	b.written += n
	if err != nil {
		return err
	}

	return nil
}

// Finalize computes the accumulator and block index values, then writes the
// corresponding e2store entries. Era1 always has an accumulator, so this
// always returns a valid hash.
func (b *Builder) Finalize() (common.Hash, error) {
	if b.startNum == nil {
		return common.Hash{}, errors.New("finalize called on empty builder")
	}
	// Compute accumulator root and write entry.
	root, err := era.ComputeAccumulator(b.hashes, b.tds)
	if err != nil {
		return common.Hash{}, fmt.Errorf("error calculating accumulator root: %w", err)
	}
	n, err := b.w.Write(era.TypeAccumulator, root[:])
	b.written += n
	if err != nil {
		return common.Hash{}, fmt.Errorf("error writing accumulator: %w", err)
	}
	b.accumulator = &root

	// Get beginning of index entry to calculate block relative offset.
	base := int64(b.written)

	// Construct block index. Detailed format described in Builder
	// documentation, but it is essentially encoded as:
	// "start | index | index | ... | count"
	var (
		count = len(b.indexes)
		index = make([]byte, 16+count*8)
	)
	binary.LittleEndian.PutUint64(index, *b.startNum)
	// Each offset is relative from the position it is encoded in the
	// index. This means that even if the same block was to be included in
	// the index twice (this would be invalid anyways), the relative offset
	// would be different. The idea with this is that after reading a
	// relative offset, the corresponding block can be quickly read by
	// performing a seek relative to the current position.
	for i, offset := range b.indexes {
		relative := int64(offset) - base
		binary.LittleEndian.PutUint64(index[8+i*8:], uint64(relative))
	}
	binary.LittleEndian.PutUint64(index[8+count*8:], uint64(count))

	// Finally, write the block index entry.
	if _, err := b.w.Write(era.TypeBlockIndex, index); err != nil {
		return common.Hash{}, fmt.Errorf("unable to write block index: %w", err)
	}

	return root, nil
}

// Accumulator returns the accumulator root after Finalize has been called.
// For Era1, this always returns a non-nil value since all blocks are pre-merge.
func (b *Builder) Accumulator() *common.Hash {
	return b.accumulator
}

// snappyWrite is a small helper to take care snappy encoding and writing an e2store entry.
func (b *Builder) snappyWrite(typ uint16, in []byte) error {
	var (
		buf = b.buf
		s   = b.snappy
	)
	buf.Reset()
	s.Reset(buf)
	if _, err := b.snappy.Write(in); err != nil {
		return fmt.Errorf("error snappy encoding: %w", err)
	}
	if err := s.Flush(); err != nil {
		return fmt.Errorf("error flushing snappy encoding: %w", err)
	}
	n, err := b.w.Write(typ, b.buf.Bytes())
	b.written += n
	if err != nil {
		return fmt.Errorf("error writing e2store entry: %w", err)
	}
	return nil
}
