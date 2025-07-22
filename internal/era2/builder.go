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

package era2

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
	"github.com/ethereum/go-ethereum/internal/era/e2store"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/golang/snappy"
)

// Type constants for the e2store entries in the Era2 format.
const (
	TypeVersion            uint16 = 0x3265
	TypeCompressedHeader   uint16 = 0x08
	TypeCompressedBody     uint16 = 0x09
	TypeCompressedReceipts uint16 = 0x0a
	TypeTotalDifficulty    uint16 = 0x0b
	TypeProof              uint16 = 0x0c
	TypeAccumulatorRoot    uint16 = 0x0d
	TypeBlockIndex         uint16 = 0x3267
	MaxEraESize            int    = 8192
	headerSize             uint64 = 8
)

// Temporary buffer for writing blocks until the Finalize method is called.
type buffer struct {
	headers  [][]byte
	bodies   [][]byte
	receipts [][]byte
	proofs   [][]byte
	tds      []*big.Int
}

// The offsets holds the offsets of the different block components in the e2store file. Eventually these offsets will be used to write the index table at the end of the file.
type offsets struct {
	headers      []uint64
	bodys        []uint64
	receipts     []uint64
	proofoffsets []uint64
	tdoff        []uint64
}

// Builder is used to build an Era2 e2store file. It collects block entries and writes them to the underlying e2store.Writer.
type Builder struct {
	w   *e2store.Writer
	buf *bytes.Buffer

	buff buffer
	off  offsets

	hashes        []common.Hash
	startNum      *uint64
	written       uint64
	expectsProofs bool
}

// NewBuilder returns a new Builder instance.
func NewBuilder(w io.Writer) *Builder {
	buf := bytes.NewBuffer(nil)
	return &Builder{
		w:   e2store.NewWriter(w),
		buf: buf,
	}
}

// Add writes a block entry, its reciepts, and optionally its proofs as well into the e2store file.
func (b *Builder) Add(header types.Header, body types.Body, receipts types.Receipts, td *big.Int, proof Proof) error {
	if len(b.buff.headers) == 0 { // first block determines wether proofs are expected
		b.expectsProofs = proof != nil
	} else if b.expectsProofs && proof == nil { // every later block must follow this policy
		return fmt.Errorf("block %d missing proof: proofs required for every block", header.Number.Uint64())
	} else if !b.expectsProofs && proof != nil {
		return fmt.Errorf("unexpected proof for block %d: first block had none", header.Number.Uint64())
	}

	eh, err := rlp.EncodeToBytes(&header)
	if err != nil {
		return fmt.Errorf("encode header: %w", err)
	}
	eb, err := rlp.EncodeToBytes(&body)
	if err != nil {
		return fmt.Errorf("encode body: %w", err)
	}
	er, err := rlp.EncodeToBytes(receipts)
	if err != nil {
		return fmt.Errorf("encode receipts: %w", err)
	}

	var ep []byte
	var buffer bytes.Buffer

	if proof != nil {
		if err := proof.EncodeRLP(&buffer); err != nil {
			return fmt.Errorf("encode proof: %w", err)
		}
		ep = buffer.Bytes()
	}

	return b.AddRLP(
		eh, eb, er, ep,
		header.Number.Uint64(),
		header.Hash(), td,
	)
}

// AddRLP takes the RLP encoded block components and writes them to the underlying e2store file.
func (b *Builder) AddRLP(headerRLP []byte, bodyRLP []byte, receipts []byte, proof []byte, blockNum uint64, blockHash common.Hash, td *big.Int) error {
	if len(b.buff.headers) >= MaxEraESize {
		return fmt.Errorf("exceeds MaxEraESize %d", MaxEraESize)
	}

	b.buff.headers = append(b.buff.headers, headerRLP)
	b.buff.bodies = append(b.buff.bodies, bodyRLP)
	b.buff.receipts = append(b.buff.receipts, receipts)
	b.buff.tds = append(b.buff.tds, new(big.Int).Set(td))
	b.hashes = append(b.hashes, blockHash)
	if proof != nil {
		b.buff.proofs = append(b.buff.proofs, proof)
	}

	// Write Era2 version before writing any blocks.
	if b.startNum == nil {
		b.startNum = new(uint64)
		*b.startNum = blockNum
		if n, err := b.w.Write(TypeVersion, nil); err != nil {
			return fmt.Errorf("write version entry: %w", err)
		} else {
			b.written += uint64(n)
		}
	}
	return nil
}

// Finalize writes all collected block entries to the e2store file and returns the accumulator root hash.
// It also writes the index table at the end of the file, which contains offsets to each block entry.
func (b *Builder) Finalize() (common.Hash, error) {
	if b.startNum == nil {
		return common.Hash{}, errors.New("no blocks added, cannot finalize")
	}
	for _, data := range b.buff.headers {
		off, err := b.addEntry(TypeCompressedHeader, data, true)
		if err != nil {
			return common.Hash{}, fmt.Errorf("headers: %w", err)
		}
		b.off.headers = append(b.off.headers, off)
	}

	for _, data := range b.buff.bodies {
		off, err := b.addEntry(TypeCompressedBody, data, true)
		if err != nil {
			return common.Hash{}, fmt.Errorf("bodies: %w", err)
		}
		b.off.bodys = append(b.off.bodys, off)
	}

	for _, data := range b.buff.receipts {
		off, err := b.addEntry(TypeCompressedReceipts, data, true)
		if err != nil {
			return common.Hash{}, fmt.Errorf("receipts: %w", err)
		}
		b.off.receipts = append(b.off.receipts, off)
	}

	if len(b.buff.tds) > 0 {
		for _, data := range b.buff.tds {
			littleEndian := uint256LE(data)
			off, err := b.addEntry(TypeTotalDifficulty, littleEndian, false)
			if err != nil {
				return common.Hash{}, fmt.Errorf("total-difficulty: %w", err)
			}
			b.off.tdoff = append(b.off.tdoff, off)
		}
	}

	if len(b.buff.proofs) > 0 {
		for _, data := range b.buff.proofs {
			off, err := b.addEntry(TypeProof, data, true)
			if err != nil {
				return common.Hash{}, fmt.Errorf("proofs: %w", err)
			}
			b.off.proofoffsets = append(b.off.proofoffsets, off)
		}
	}

	var accRoot common.Hash
	if len(b.hashes) > 0 {
		var err error
		accRoot, err = ComputeAccumulator(b.hashes, b.buff.tds)
		if err != nil {
			return common.Hash{}, fmt.Errorf("compute accumulator: %w", err)
		}
		if n, err := b.w.Write(TypeAccumulatorRoot, accRoot[:]); err != nil {
			return common.Hash{}, fmt.Errorf("write accumulator: %w", err)
		} else {
			b.written += uint64(n)
		}
	}

	return accRoot, b.writeIndex()
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
	var buf = b.buf
	snappy := snappy.NewBufferedWriter(b.buf)
	buf.Reset()
	snappy.Reset(buf)
	if _, err := snappy.Write(in); err != nil {
		return fmt.Errorf("error snappy encoding: %w", err)
	}
	if err := snappy.Flush(); err != nil {
		return fmt.Errorf("error flushing snappy encoding: %w", err)
	}
	n, err := b.w.Write(typ, b.buf.Bytes())
	b.written += uint64(n)
	if err != nil {
		return fmt.Errorf("error writing e2store entry: %w", err)
	}
	return nil
}

// addEntry takes the e2store object and writes it into the file.
func (b *Builder) addEntry(typ uint16, payload []byte, compressed bool) (uint64, error) {
	offset := b.written
	var err error
	if compressed {
		if err = b.snappyWrite(typ, payload); err != nil {
			return 0, err
		}
	} else {
		var n int
		if n, err = b.w.Write(typ, payload); err != nil {
			return 0, err
		}
		b.written += uint64(n)
	}
	return offset, nil
}

// writeIndex takes all the offset table and writes it to the file. The index table contains all offsets to specific block entries
func (b *Builder) writeIndex() error {
	count := uint64(len(b.off.headers))
	componentCount := uint64(3)
	if len(b.buff.tds) > 0 {
		componentCount++
	}
	if len(b.buff.proofs) > 0 {
		componentCount++
	}

	index := make([]byte, 8+count*8*componentCount+16) // 8 for start block, 8 per property per block, 16 for the number of properties and the number of blocks
	binary.LittleEndian.PutUint64(index, *b.startNum)
	base := int64(b.written)
	rel := func(abs uint64) uint64 { return uint64(int64(abs) - base) }
	for i := uint64(0); i < count; i++ {
		basePosition := 8 + i*componentCount*8

		binary.LittleEndian.PutUint64(index[basePosition:], rel(b.off.headers[i]))
		binary.LittleEndian.PutUint64(index[basePosition+8:], rel(b.off.bodys[i]))
		binary.LittleEndian.PutUint64(index[basePosition+16:], rel(b.off.receipts[i]))

		pos := uint64(24)
		if len(b.buff.tds) > 0 {
			binary.LittleEndian.PutUint64(index[basePosition+pos:], rel(b.off.tdoff[i]))
			pos += 8
		}
		if len(b.buff.proofs) > 0 {
			binary.LittleEndian.PutUint64(index[basePosition+pos:], rel(b.off.proofoffsets[i]))
		}
	}
	end := 8 + count*componentCount*8

	binary.LittleEndian.PutUint64(index[end+0:], componentCount)
	binary.LittleEndian.PutUint64(index[end+8:], count)
	if n, err := b.w.Write(TypeBlockIndex, index); err != nil {
		return err
	} else {
		b.written += uint64(n)
	}
	return nil
}
