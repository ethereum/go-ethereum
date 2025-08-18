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

// The format can be summarized with the following expression:
//    eraE := Version | CompressedHeader* | CompressedBody* | CompressedReceipts* | TotalDifficulty* | Proofs* | other-entries* | Accumulator | BlockIndex
// Each basic element is its own e2store entry:

//    Version            = { type: 0x3265, data: nil }
//    CompressedHeader   = { type: 0x03,   data: snappyFramed(rlp(header)) }
//    CompressedBody     = { type: 0x04,   data: snappyFramed(rlp(body)) }
//    CompressedReceipts = { type: 0x05,   data: snappyFramed(rlp([tx-type, post-state-or-status, cumulative-gas, logs])) }
//    TotalDifficulty    = { type: 0x06,   data: uint256(header.total_difficulty) }
//    Proofs             = { type: 0x07    data: snappyFramed(rlp([BlockProofHistoricalHashesAccumulator, BlockProofHistoricalRoots, BlockProofHistoricalSummaries]))}
//    AccumulatorRoot    = { type: 0x08,   data: hash_tree_root(List(HeaderRecord, 8192)) }
//    BlockIndex         = { type: 0x3266, data: block-index }
// TotalDifficulty is little-endian encoded.

// AccumulatorRoot is only defined for epochs with pre-merge data.
// HeaderRecord is defined in the Portal Network specification[^5].

// BlockIndex stores relative offsets to each compressed block entry. The format is:

//    block-index := starting-number | index | index | index ... | count
// All values in the block index are little-endian uint64.

// starting-number is the first block number in the archive. Every index is a defined relative to index's location in the file. The total number of block entries in the file is recorded in count.

// Due to the accumulator size limit of 8192, the maximum number of blocks in an Era batch is also 8192. This is also the value of SLOTS_PER_HISTORICAL_ROOT[^6] on the Beacon chain, so it is nice to align on the value.

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

// Builder is used to build an Era2 e2store file. It collects block entries and writes them to the underlying e2store.Writer.
type Builder struct {
	w *e2store.Writer

	headers  [][]byte
	hashes   []common.Hash
	bodies   [][]byte
	receipts [][]byte
	proofs   [][]byte
	tds      []*big.Int

	startNum *uint64
	merged   bool
	written  uint64

	buf    *bytes.Buffer
	snappy *snappy.Writer
}

// NewBuilder returns a new Builder instance.
func NewBuilder(w io.Writer) era.Builder {
	return &Builder{
		w: e2store.NewWriter(w),
	}
}

// Add writes a block entry, its reciepts, and optionally its proofs as well into the e2store file.
func (b *Builder) Add(block *types.Block, receipts types.Receipts, td *big.Int, proof era.Proof) error {
	eh, err := rlp.EncodeToBytes(block.Header())
	if err != nil {
		return fmt.Errorf("encode header: %w", err)
	}
	eb, err := rlp.EncodeToBytes(block.Body())
	if err != nil {
		return fmt.Errorf("encode body: %w", err)
	}

	rs := make([]*types.ReceiptForStorage, len(receipts))
	for i, receipt := range receipts {
		rs[i] = (*types.ReceiptForStorage)(receipt)
	}
	er, err := rlp.EncodeToBytes(rs)
	if err != nil {
		return fmt.Errorf("encode receipts: %w", err)
	}

	var ep []byte
	if proof != nil {
		ep, err = rlp.EncodeToBytes(ep)
		if err != nil {
			return fmt.Errorf("encode proof: %w", err)
		}
	}
	var difficulty *big.Int
	if !b.merged {
		difficulty = block.Difficulty()
	}

	return b.AddRLP(eh, eb, er, ep, block.Number().Uint64(), block.Hash(), td, difficulty)
}

// AddRLP takes the RLP encoded block components and writes them to the underlying e2store file.
func (b *Builder) AddRLP(header []byte, body []byte, receipts []byte, proof []byte, number uint64, blockHash common.Hash, td, difficulty *big.Int) error {
	if b.startNum == nil {
		b.startNum = new(uint64)
		*b.startNum = number
		b.merged = difficulty.Sign() == 0
	}
	if len(b.headers) >= era.MaxSize {
		return fmt.Errorf("exceeds max size %d", era.MaxSize)
	}
	if !b.merged && (td == nil || difficulty == nil || difficulty.Sign() == 0) {
		return fmt.Errorf("era pre-merge, but difficulty values not supplied")
	}
	if b.merged && (td != nil || difficulty != nil) {
		return fmt.Errorf("era already merged, but given non-nil difficulty values")
	}
	if len(b.headers) != 0 && len(b.proofs) == 0 && proof != nil {
		return fmt.Errorf("unexpected proof for block %d: first block had none", number)
	}
	if len(b.headers) != 0 && len(b.proofs) != 0 && proof == nil {
		return fmt.Errorf("block %d missing proof: proofs required for every block", number)
	}

	b.headers = append(b.headers, header)
	b.bodies = append(b.bodies, body)
	b.receipts = append(b.receipts, receipts)

	if !b.merged {
		if difficulty.Sign() != 0 {
			b.hashes = append(b.hashes, blockHash)
		}
		b.tds = append(b.tds, new(big.Int).Set(td))
	}

	if proof != nil {
		b.proofs = append(b.proofs, proof)
	}
	return nil
}

type offsets struct {
	headers  []uint64
	bodies   []uint64
	receipts []uint64
	tds      []uint64
	proofs   []uint64
}

// Finalize writes all collected block entries to the e2store file and returns the accumulator root hash.
// It also writes the index table at the end of the file, which contains offsets to each block entry.
func (b *Builder) Finalize() (common.Hash, error) {
	if b.startNum == nil {
		return common.Hash{}, errors.New("no blocks added, cannot finalize")
	}
	// Write Era2 version before writing any blocks.
	if n, err := b.w.Write(era.TypeVersion, nil); err != nil {
		return common.Hash{}, fmt.Errorf("write version entry: %w", err)
	} else {
		b.written += uint64(n)
	}

	// Convert int values to byte-level LE representation.
	var tds [][]byte
	for _, td := range b.tds {
		tds = append(tds, uint256LE(td))
	}

	// Create snappy writer.
	b.buf = bytes.NewBuffer(nil)
	b.snappy = snappy.NewBufferedWriter(b.buf)

	var o offsets
	for _, section := range []struct {
		typ        uint16
		data       [][]byte
		compressed bool
		offsets    *[]uint64
	}{
		{era.TypeCompressedHeader, b.headers, true, &o.headers},
		{era.TypeCompressedBody, b.bodies, true, &o.bodies},
		{era.TypeCompressedSlimReceipts, b.receipts, true, &o.receipts},
		{era.TypeTotalDifficulty, tds, false, &o.tds},
		{era.TypeProof, b.proofs, true, &o.proofs},
	} {
		for _, data := range section.data {
			*section.offsets = append(*section.offsets, b.written)
			if section.compressed {
				// Write snappy compressed data.
				if err := b.snappyWrite(section.typ, data); err != nil {
					return common.Hash{}, err
				}
			} else {
				// Directly write uncompressed data.
				n, err := b.w.Write(section.typ, data)
				if err != nil {
					return common.Hash{}, err
				}
				b.written += uint64(n)
			}
		}
	}

	// Compute and write accumlator root only when the first block the epoch is
	// pre-merge, otherwise omit.
	var accRoot common.Hash
	if !b.merged {
		var err error
		accRoot, err = era.ComputeAccumulator(b.hashes, b.tds[:len(b.hashes)])
		if err != nil {
			return common.Hash{}, fmt.Errorf("compute accumulator: %w", err)
		}
		if n, err := b.w.Write(era.TypeAccumulator, accRoot[:]); err != nil {
			return common.Hash{}, fmt.Errorf("write accumulator: %w", err)
		} else {
			b.written += uint64(n)
		}
	}

	// TODO: accumulator root should only be returned when it's relevant
	// (pre-merge)?
	return accRoot, b.writeIndex(&o)
}

// uin256LE writes 32 byte big integers to little endian.
func uint256LE(v *big.Int) []byte {
	b := v.FillBytes(make([]byte, 32))
	for i := 0; i < 16; i++ {
		b[i], b[31-i] = b[31-i], b[i]
	}
	return b
}

// SnappyWrite compresses the input data using snappy and writes it to the e2store file.
func (b *Builder) snappyWrite(typ uint16, in []byte) error {
	b.buf.Reset()
	b.snappy.Reset(b.buf)
	if _, err := b.snappy.Write(in); err != nil {
		return fmt.Errorf("error snappy encoding: %w", err)
	}
	if err := b.snappy.Flush(); err != nil {
		return fmt.Errorf("error flushing snappy encoding: %w", err)
	}
	n, err := b.w.Write(typ, b.buf.Bytes())
	b.written += uint64(n)
	if err != nil {
		return fmt.Errorf("error writing e2store entry: %w", err)
	}
	return nil
}

// writeIndex takes all the offset table and writes it to the file. The index table contains all offsets to specific block entries
func (b *Builder) writeIndex(o *offsets) error {
	count := uint64(len(o.headers))
	componentCount := uint64(3)
	if len(o.tds) > 0 {
		componentCount++
	}
	if len(o.proofs) > 0 {
		componentCount++
	}

	index := make([]byte, 8+count*8*componentCount+16) // 8 for start block, 8 per property per block, 16 for the number of properties and the number of blocks
	binary.LittleEndian.PutUint64(index, *b.startNum)
	base := int64(b.written)
	rel := func(abs uint64) uint64 { return uint64(int64(abs) - base) }
	for i := uint64(0); i < count; i++ {
		basePosition := 8 + i*componentCount*8

		binary.LittleEndian.PutUint64(index[basePosition:], rel(o.headers[i]))
		binary.LittleEndian.PutUint64(index[basePosition+8:], rel(o.bodies[i]))
		binary.LittleEndian.PutUint64(index[basePosition+16:], rel(o.receipts[i]))

		pos := uint64(24)
		if len(o.tds) > 0 {
			binary.LittleEndian.PutUint64(index[basePosition+pos:], rel(o.tds[i]))
			pos += 8
		}
		if len(o.proofs) > 0 {
			binary.LittleEndian.PutUint64(index[basePosition+pos:], rel(o.proofs[i]))
		}
	}
	end := 8 + count*componentCount*8

	binary.LittleEndian.PutUint64(index[end+0:], componentCount)
	binary.LittleEndian.PutUint64(index[end+8:], count)
	if n, err := b.w.Write(era.TypeComponentIndex, index); err != nil {
		return err
	} else {
		b.written += uint64(n)
	}
	return nil
}
