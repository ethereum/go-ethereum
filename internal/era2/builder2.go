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

const (
	TypeVersion                          uint16 = 0x3265
	TypeCompressedHeader                 uint16 = 0x08
	TypeCompressedBody                   uint16 = 0x09
	TypeCompressedReceipts               uint16 = 0x0a
	TypeTotalDifficulty                  uint16 = 0x0b
	TypeProofHistoricalHashesAccumulator uint16 = 0x0c
	TypeProofHistoricalRoots             uint16 = 0x0d
	TypeProofHistoricalSummariesCapella  uint16 = 0x0e
	TypeProofHistoricalSummariesDeneb    uint16 = 0x0f
	TypeAccumulatorRoot                  uint16 = 0x1a
	TypeBlockIndex                       uint16 = 0x3267
	MaxEraESize                          int    = 8192
	headerSize                           uint64 = 8
)

type proofvar uint16

type Builder struct {
	w   *e2store.Writer
	buf *bytes.Buffer
	sn  *snappy.Writer

	// buffered entries per type:
	headersRLP  [][]byte
	bodiesRLP   [][]byte
	receiptsRLP [][]byte
	proofsRLP   [][]byte
	tds         [][]byte
	tdsint      []*big.Int
	hashes      []common.Hash

	headeroffsets  []uint64
	bodyoffsets    []uint64
	receiptoffsets []uint64
	proofoffsets   []uint64
	tdoff          []uint64
	startTd        *big.Int

	prooftype proofvar

	startNum     *uint64
	writtenBytes uint64
}

// NewBuilder returns a new EraE Builder writing into the given io.Writer.
func NewBuilder(w io.Writer) *Builder {
	buf := bytes.NewBuffer(nil)
	return &Builder{
		w:   e2store.NewWriter(w),
		buf: buf,
		sn:  snappy.NewBufferedWriter(buf),
	}
}

func (b *Builder) Add(block *types.Block, receipts types.Receipts, td *big.Int, proofBytes []byte, proofty proofvar) error {
	if len(b.headersRLP) >= MaxEraESize {
		return fmt.Errorf("exceeds MaxEraESize %d", MaxEraESize)
	}

	if proofty != 0 && len(proofBytes) == 0 {
		return fmt.Errorf("proof type %d requires proof bytes", proofty)
	}

	if len(b.headersRLP) != 0 && proofty != b.prooftype {
		return fmt.Errorf("cannot mix proof types, expected %d, got %d", b.prooftype, proofty)
	}

	hdr, err := rlp.EncodeToBytes(block.Header())
	if err != nil {
		return fmt.Errorf("error encoding block header: %w", err)
	}
	bod, err := rlp.EncodeToBytes(block.Body())
	if err != nil {
		return fmt.Errorf("error encoding block header: %w", err)
	}
	rct, err := rlp.EncodeToBytes(receipts)
	if err != nil {
		return fmt.Errorf("error encoding block header: %w", err)
	}

	b.headersRLP = append(b.headersRLP, hdr)
	b.bodiesRLP = append(b.bodiesRLP, bod)
	b.receiptsRLP = append(b.receiptsRLP, rct)
	b.tds = append(b.tds, uint256LE(td))
	b.tdsint = append(b.tdsint, new(big.Int).Set(td))
	b.hashes = append(b.hashes, block.Hash())

	if b.prooftype != 0 {
		b.proofsRLP = append(b.proofsRLP, proofBytes)
	}

	if b.startNum == nil {
		sn := block.NumberU64()
		b.startNum = &sn
		if n, err := b.w.Write(TypeVersion, nil); err != nil {
			return fmt.Errorf("error writing version entry: %w", err)
		} else {
			b.writtenBytes += uint64(n)
		}
	}
	return nil
}

func (b *Builder) Finalize() error {
	if b.startNum == nil {
		return errors.New("no blocks added, cannot finalize")
	}
	var err error
	b.headeroffsets, err = b.writeSection(TypeCompressedHeader, b.headersRLP, true)
	if err != nil {
		return fmt.Errorf("error writing compressed headers: %w", err)
	}
	b.bodyoffsets, err = b.writeSection(TypeCompressedBody, b.bodiesRLP, true)
	if err != nil {
		return fmt.Errorf("error writing compressed bodies: %w", err)
	}
	b.receiptoffsets, err = b.writeSection(TypeCompressedReceipts, b.receiptsRLP, true)
	if err != nil {
		return fmt.Errorf("error writing compressed receipts: %w", err)
	}

	if len(b.tds) > 0 {
		b.tdoff, err = b.writeSection(TypeTotalDifficulty, b.tds, false)
		if err != nil {
			return fmt.Errorf("error writing total difficulties: %w", err)
		}
	}

	if b.prooftype != 0 {
		b.proofoffsets, err = b.writeSection(uint16(b.prooftype), b.proofsRLP, true)
		if err != nil {
			return fmt.Errorf("error writing proofs: %w", err)
		}
	}

	if len(b.hashes) > 0 {
		accRoot, err := ComputeAccumulator(b.hashes, b.tdsint)
		if err != nil {
			return fmt.Errorf("error calculating accumulator root: %w", err)
		}
		if n, err := b.w.Write(TypeAccumulatorRoot, accRoot[:]); err != nil {
			return fmt.Errorf("error writing accumulator root: %w", err)
		} else {
			b.writtenBytes += uint64(n)
		}
	}

	return b.writeIndex()

}

func uint256LE(v *big.Int) []byte {
	b := v.FillBytes(make([]byte, 32))
	for i := 0; i < 16; i++ {
		b[i], b[31-i] = b[31-i], b[i]
	}
	return b
}

func decodeBigs(raw [][]byte) []*big.Int {
	out := make([]*big.Int, len(raw))
	for i, le := range raw {
		be := make([]byte, 32)
		for j := 0; j < 32; j++ {
			be[j] = le[31-j]
		}
		out[i] = new(big.Int).SetBytes(be)
	}
	return out
}

// snappyWrite is a small helper to take care snappy encoding and writing an e2store entry.
func (b *Builder) snappyWrite(typ uint16, in []byte) error {
	var (
		buf = b.buf
		s   = b.sn
	)
	buf.Reset()
	s.Reset(buf)
	if _, err := b.sn.Write(in); err != nil {
		return fmt.Errorf("error snappy encoding: %w", err)
	}
	if err := s.Flush(); err != nil {
		return fmt.Errorf("error flushing snappy encoding: %w", err)
	}
	n, err := b.w.Write(typ, b.buf.Bytes())
	b.writtenBytes += uint64(n)
	if err != nil {
		return fmt.Errorf("error writing e2store entry: %w", err)
	}
	return nil
}

func (b *Builder) writeSection(typ uint16, list [][]byte, useSnappy bool) ([]uint64, error) {
	if len(list) == 0 {
		return nil, errors.New("cannot write empty section")
	}

	buf := bytes.NewBuffer(nil)
	offs := make([]uint64, len(list))
	for i, data := range list {
		offs[i] = b.writtenBytes + headerSize + uint64(buf.Len())
		if useSnappy {
			buf.Write(snappy.Encode(nil, data))
		} else {
			buf.Write(data)
		}
	}

	if n, err := b.w.Write(typ, buf.Bytes()); err != nil {
		return nil, fmt.Errorf("error writing section %d: %w", typ, err)
	} else {
		b.writtenBytes += uint64(n)
	}

	return offs, nil
}

func (b *Builder) writeIndex() error {
	count := uint64(len(b.headeroffsets))
	compcount := uint64(3)
	if len(b.tds) > 0 {
		compcount++
	}
	if len(b.proofsRLP) > 0 {
		compcount++
	}

	idx := make([]byte, 8+count*8*compcount+16) //8 for start block, 8 per property per block, 16 for the number of properties and the number of blocks
	binary.LittleEndian.PutUint64(idx, *b.startNum)
	base := int64(b.writtenBytes)
	pos := 8
	rel := func(abs uint64) uint64 { return uint64(int64(abs) - base) }
	for i := uint64(0); i < count; i++ {
		binary.LittleEndian.PutUint64(idx[pos:], rel(b.headeroffsets[i]))
		pos += 8
		binary.LittleEndian.PutUint64(idx[pos:], rel(b.bodyoffsets[i]))
		pos += 8
		binary.LittleEndian.PutUint64(idx[pos:], rel(b.receiptoffsets[i]))
		pos += 8
		if len(b.tds) > 0 {
			binary.LittleEndian.PutUint64(idx[pos+24:], rel(b.tdoff[i]))
			pos += 8
		}
		if len(b.proofsRLP) > 0 {
			binary.LittleEndian.PutUint64(idx[pos:], rel(b.proofoffsets[i]))
			pos += 8
		}
	}

	binary.LittleEndian.PutUint64(idx[pos:], compcount)
	pos += 8
	binary.LittleEndian.PutUint64(idx[pos:], count)
	if n, err := b.w.Write(TypeBlockIndex, idx); err != nil {
		return err
	} else {
		b.writtenBytes += uint64(n)
	}
	return nil
}
