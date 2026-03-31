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

// EraE file format specification.
//
// The format can be summarized with the following expression:
//
//	eraE := Version | CompressedHeader* | CompressedBody* | CompressedSlimReceipts* | TotalDifficulty* | other-entries* | Accumulator? | ComponentIndex
//
// Each basic element is its own e2store entry:
//
//	Version              = { type: 0x3265, data: nil }
//	CompressedHeader     = { type: 0x03,   data: snappyFramed(rlp(header)) }
//	CompressedBody       = { type: 0x04,   data: snappyFramed(rlp(body)) }
//	CompressedSlimReceipts = { type: 0x0a, data: snappyFramed(rlp([tx-type, post-state-or-status, cumulative-gas, logs])) }
//	TotalDifficulty      = { type: 0x06,   data: uint256 (header.total_difficulty) }
//	AccumulatorRoot      = { type: 0x07,   data: hash_tree_root(List(HeaderRecord, 8192)) }
//	ComponentIndex       = { type: 0x3267, data: component-index }
//
// Notes:
//   - TotalDifficulty is present for pre-merge and merge transition epochs.
//     For pure post-merge epochs, TotalDifficulty is omitted entirely.
//   - In merge transition epochs, post-merge blocks store the final total
//     difficulty (the TD at which the merge occurred).
//   - AccumulatorRoot is only written for pre-merge epochs.
//   - HeaderRecord is defined in the Portal Network specification.
//   - Proofs (type 0x09) are defined in the spec but not yet supported in this implementation.
//
// ComponentIndex stores relative offsets to each block's components:
//
//	component-index := starting-number | indexes | indexes | ... | component-count | count
//	indexes := header-offset | body-offset | receipts-offset | td-offset?
//
// All values are little-endian uint64.
//
// Due to the accumulator size limit of 8192, the maximum number of blocks in an
// EraE file is also 8192.

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

// Builder is used to build an EraE e2store file. It collects block entries and
// writes them to the underlying e2store.Writer.
type Builder struct {
	w *e2store.Writer

	headers  [][]byte
	hashes   []common.Hash // only pre-merge block hashes, for accumulator
	bodies   [][]byte
	receipts [][]byte
	tds      []*big.Int

	startNum    *uint64
	ttd         *big.Int     // terminal total difficulty
	last        common.Hash  // hash of last block added
	accumulator *common.Hash // accumulator root, set by Finalize (nil for post-merge)

	written uint64

	buf    *bytes.Buffer
	snappy *snappy.Writer
}

// NewBuilder returns a new Builder instance.
func NewBuilder(w io.Writer) era.Builder {
	return &Builder{
		w: e2store.NewWriter(w),
	}
}

// Add writes a block entry and its receipts into the e2store file.
func (b *Builder) Add(block *types.Block, receipts types.Receipts, td *big.Int) error {
	eh, err := rlp.EncodeToBytes(block.Header())
	if err != nil {
		return fmt.Errorf("encode header: %w", err)
	}
	eb, err := rlp.EncodeToBytes(block.Body())
	if err != nil {
		return fmt.Errorf("encode body: %w", err)
	}

	rs := make([]*types.SlimReceipt, len(receipts))
	for i, receipt := range receipts {
		rs[i] = (*types.SlimReceipt)(receipt)
	}
	er, err := rlp.EncodeToBytes(rs)
	if err != nil {
		return fmt.Errorf("encode receipts: %w", err)
	}

	return b.AddRLP(eh, eb, er, block.Number().Uint64(), block.Hash(), td, block.Difficulty())
}

// AddRLP takes the RLP encoded block components and writes them to the underlying e2store file.
// The builder automatically handles transition epochs where both pre and post-merge blocks exist.
func (b *Builder) AddRLP(header, body, receipts []byte, number uint64, blockHash common.Hash, td, difficulty *big.Int) error {
	if len(b.headers) >= era.MaxSize {
		return fmt.Errorf("exceeds max size %d", era.MaxSize)
	}
	// Set starting block number on first add.
	if b.startNum == nil {
		b.startNum = new(uint64)
		*b.startNum = number
	}

	if difficulty == nil {
		return fmt.Errorf("invalid block: difficulty is nil")
	}
	hasDifficulty := difficulty.Sign() > 0
	// Expect td to be nil for post-merge blocks
	// and non-nil for pre-merge blocks.
	if hasDifficulty != (td != nil) {
		return fmt.Errorf("TD and difficulty mismatch: expected both nil or both non-nil")
	}
	// After the merge, difficulty must be nil.
	post := (b.tds == nil && len(b.headers) > 0) || b.ttd != nil
	if post && hasDifficulty {
		return fmt.Errorf("post-merge epoch: cannot accept total difficulty for block %d", number)
	}

	// If this marks the start of the transition, record final total
	// difficulty value.
	if b.ttd == nil && len(b.tds) > 0 && !hasDifficulty {
		b.ttd = new(big.Int).Set(b.tds[len(b.tds)-1])
	}

	// Record block data.
	b.headers = append(b.headers, header)
	b.bodies = append(b.bodies, body)
	b.receipts = append(b.receipts, receipts)
	b.last = blockHash

	// Conditionally write the total difficulty and block hashes.
	//   - Pre-merge: store total difficulty and block hashes.
	//   - Transition: only store total difficulty.
	//   - Post-merge: store neither.
	if hasDifficulty {
		b.hashes = append(b.hashes, blockHash)
		b.tds = append(b.tds, new(big.Int).Set(td))
	} else if b.ttd != nil {
		b.tds = append(b.tds, new(big.Int).Set(b.ttd))
	} else {
		// Post-merge: no TD or block hashes stored.
	}

	return nil
}

// Accumulator returns the accumulator root after Finalize has been called.
// Returns nil for post-merge epochs where no accumulator exists.
func (b *Builder) Accumulator() *common.Hash {
	return b.accumulator
}

type offsets struct {
	headers  []uint64
	bodies   []uint64
	receipts []uint64
	tds      []uint64
}

// Finalize writes all collected block entries to the e2store file.
// For pre-merge or transition epochs, the accumulator root is computed over
// pre-merge blocks and written. For pure post-merge epochs, no accumulator
// is written. Always returns the last block hash as the epoch identifier.
func (b *Builder) Finalize() (common.Hash, error) {
	if b.startNum == nil {
		return common.Hash{}, errors.New("no blocks added, cannot finalize")
	}
	// Write version before writing any blocks.
	if n, err := b.w.Write(era.TypeVersion, nil); err != nil {
		return common.Hash{}, fmt.Errorf("write version entry: %w", err)
	} else {
		b.written += uint64(n)
	}

	// Convert TD values to byte-level LE representation.
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

	// Compute and write accumulator root only for epochs that started pre-merge.
	// The accumulator is computed over only the pre-merge blocks (b.hashes).
	// Pure post-merge epochs have no accumulator.
	if len(b.tds) > 0 {
		accRoot, err := era.ComputeAccumulator(b.hashes, b.tds[:len(b.hashes)])
		if err != nil {
			return common.Hash{}, fmt.Errorf("compute accumulator: %w", err)
		}
		if n, err := b.w.Write(era.TypeAccumulator, accRoot[:]); err != nil {
			return common.Hash{}, fmt.Errorf("write accumulator: %w", err)
		} else {
			b.written += uint64(n)
		}
		b.accumulator = &accRoot
		if err := b.writeIndex(&o); err != nil {
			return common.Hash{}, err
		}
		return b.last, nil
	}

	// Pure post-merge epoch: no accumulator.
	if err := b.writeIndex(&o); err != nil {
		return common.Hash{}, err
	}
	return b.last, nil
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

// writeIndex writes the component index to the file.
func (b *Builder) writeIndex(o *offsets) error {
	count := len(o.headers)

	// Post-merge, we only index headers, bodies, and receipts. Pre-merge, we also
	// need to index the total difficulties.
	componentCount := 3
	if len(o.tds) > 0 {
		componentCount++
	}

	// Offsets are stored relative to the index position (negative, stored as uint64).
	base := int64(b.written)
	rel := func(abs uint64) uint64 { return uint64(int64(abs) - base) }

	var buf bytes.Buffer
	write := func(v uint64) { binary.Write(&buf, binary.LittleEndian, v) }

	write(*b.startNum)
	for i := range o.headers {
		write(rel(o.headers[i]))
		write(rel(o.bodies[i]))
		write(rel(o.receipts[i]))
		if len(o.tds) > 0 {
			write(rel(o.tds[i]))
		}
	}
	write(uint64(componentCount))
	write(uint64(count))

	n, err := b.w.Write(era.TypeComponentIndex, buf.Bytes())
	b.written += uint64(n)
	return err
}
