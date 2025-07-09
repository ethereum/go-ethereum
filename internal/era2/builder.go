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

type proofvar uint16

const (
	ProofNone proofvar = iota
	ProofHHA
	ProofRoots
	ProofCapella
	ProofDeneb
)

// Proof bundles variant + compressed bytes.
type Proof struct {
	Variant proofvar
	Data    []byte
}

type Builder struct {
	w   *e2store.Writer
	buf *bytes.Buffer
	sn  *snappy.Writer

	headers  [][]byte
	bodies   [][]byte
	receipts [][]byte
	proofs   [][]byte
	tds      [][]byte
	tdsint   []*big.Int
	hashes   []common.Hash

	headeroffsets  []uint64
	bodyoffsets    []uint64
	receiptoffsets []uint64
	proofoffsets   []uint64
	tdoff          []uint64

	prooftype proofvar

	startNum     *uint64
	writtenBytes uint64
}

// NewBuilder returns a new Builder instance.
func NewBuilder(w io.Writer) *Builder {
	buf := bytes.NewBuffer(nil)
	return &Builder{
		w:   e2store.NewWriter(w),
		buf: buf,
		sn:  snappy.NewBufferedWriter(buf),
	}
}

// Add writes a block entry, its reciepts, and optionally its proofs as well into the e2store file.
func (b *Builder) Add(header types.Header, body types.Body, receipts types.Receipts, td *big.Int, proof *Proof) error {
	var pv proofvar = ProofNone
	var pData []byte
	if proof != nil {
		pv, pData = proof.Variant, proof.Data
		if pv == ProofNone || len(pData) == 0 {
			return fmt.Errorf("invalid proof: variant=%d len=%d", pv, len(pData))
		}
	}

	if len(b.headers) == 0 {
		b.prooftype = pv
	} else if pv != b.prooftype {
		return fmt.Errorf("cannot mix proof variants: first=%d now=%d", b.prooftype, pv)
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
	if pv != ProofNone {
		ep, err = rlp.EncodeToBytes([]interface{}{uint16(pv), proof.Data})
		if err != nil {
			return fmt.Errorf("encode proof: %w", err)
		}
	}

	return b.AddRLP(eh, eb, er, ep, header.Number.Uint64(), header.Hash(), td)
}

// AddRLP takes the RLP encoded block components and writes them to the underlying e2store file
func (b *Builder) AddRLP(headerRLP []byte, bodyRLP []byte, receipts []byte, proof []byte, blockNum uint64, blockHash common.Hash, td *big.Int) error {
	if len(b.headers) >= MaxEraESize {
		return fmt.Errorf("exceeds MaxEraESize %d", MaxEraESize)
	}

	b.headers = append(b.headers, headerRLP)
	b.bodies = append(b.bodies, bodyRLP)
	b.receipts = append(b.receipts, receipts)
	b.tds = append(b.tds, uint256LE(td))
	b.tdsint = append(b.tdsint, new(big.Int).Set(td))
	b.hashes = append(b.hashes, blockHash)
	if proof != nil {
		b.proofs = append(b.proofs, proof)
	}

	//Write Era2 version before writing any blocks.
	if b.startNum == nil {
		b.startNum = new(uint64)
		*b.startNum = blockNum
		if n, err := b.w.Write(TypeVersion, nil); err != nil {
			return fmt.Errorf("write version entry: %w", err)
		} else {
			b.writtenBytes += uint64(n)
		}
	}
	return nil
}

// Finalize flushes all entries present in cache to the underlying e2store file.
func (b *Builder) Finalize() (common.Hash, error) {
	if b.startNum == nil {
		return common.Hash{}, errors.New("no blocks added, cannot finalize")
	}
	var err error
	if err := b.flushKind(TypeCompressedHeader, b.headers, true, &b.headeroffsets); err != nil {
		return common.Hash{}, err
	}
	if err := b.flushKind(TypeCompressedBody, b.bodies, true, &b.bodyoffsets); err != nil {
		return common.Hash{}, err
	}
	if err := b.flushKind(TypeCompressedReceipts, b.receipts, true, &b.receiptoffsets); err != nil {
		return common.Hash{}, err
	}

	if len(b.tds) > 0 {
		if err := b.flushKind(TypeTotalDifficulty, b.tds, false, &b.tdoff); err != nil {
			return common.Hash{}, err
		}
	}
	if b.prooftype != ProofNone {
		if err := b.flushKind(TypeProof, b.proofs, true, &b.proofoffsets); err != nil {
			return common.Hash{}, err
		}
	}

	var accRoot common.Hash
	if len(b.hashes) > 0 {
		accRoot, err = ComputeAccumulator(b.hashes, b.tdsint)
		if err != nil {
			return common.Hash{}, fmt.Errorf("error calculating accumulator root: %w", err)
		}
		if n, err := b.w.Write(TypeAccumulatorRoot, accRoot[:]); err != nil {
			return common.Hash{}, fmt.Errorf("error writing accumulator root: %w", err)
		} else {
			b.writtenBytes += uint64(n)
		}
	}
	return accRoot, b.writeIndex()
}

// Writes 32 byte big integers to little endian
func uint256LE(v *big.Int) []byte {
	b := v.FillBytes(make([]byte, 32))
	for i := 0; i < 16; i++ {
		b[i], b[31-i] = b[31-i], b[i]
	}
	return b
}

// Compresses into snappy encoding
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

// Add entry takes the e2store object and writes it into the file
func (b *Builder) addEntry(typ uint16, payload []byte, snappyIt bool) (uint64, error) {
	offset := b.writtenBytes
	var err error
	if snappyIt {
		if err = b.snappyWrite(typ, payload); err != nil {
			return 0, err
		}
	} else {
		var n int
		if n, err = b.w.Write(typ, payload); err != nil {
			return 0, err
		}
		b.writtenBytes += uint64(n)
	}
	return offset, nil
}

// flush kind takes all entries of the cached component and flushes it to the file
func (b *Builder) flushKind(typ uint16, list [][]byte, useSnappy bool, dst *[]uint64) error {
	for _, data := range list {
		off, err := b.addEntry(typ, data, useSnappy)
		if err != nil {
			return fmt.Errorf("entry type %d: %w", typ, err)
		}
		*dst = append(*dst, off)
	}
	return nil
}

// write index takes all the offset table and writes it to the file
// the index table contains all offsets to specific block entries
func (b *Builder) writeIndex() error {
	count := uint64(len(b.headeroffsets))
	compcount := uint64(3)
	if len(b.tds) > 0 {
		compcount++
	}
	if len(b.proofs) > 0 {
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
			binary.LittleEndian.PutUint64(idx[pos:], rel(b.tdoff[i]))
			pos += 8
		}
		if len(b.proofs) > 0 {
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
