package era

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
	"fmt"
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/internal/era/e2store"
	"github.com/golang/snappy"
)

const (
	TypeVersion            uint16 = 0x3265
	TypeCompressedHeader   uint16 = 0x08
	TypeCompressedBody     uint16 = 0x09
	TypeCompressedReceipts uint16 = 0x0a
	TypeTotalDifficulty    uint16 = 0x0b
	TypeProof              uint16 = 0x0c
	TypeAccumulator        uint16 = 0x0d
	TypeBlockIndex         uint16 = 0x3267

	MaxEraESize = 8192
)

type Builder2 struct {
	w   *e2store.Writer
	buf *bytes.Buffer
	sn  *snappy.Writer

	// buffered entries per type:
	headersRLP  [][]byte
	bodiesRLP   [][]byte
	receiptsRLP [][]byte
	proofsRLP   [][]byte
	tds         []*big.Int

	headeroffsets  []uint64
	bodyoffsets    []uint64
	receiptoffsets []uint64
	proofoffsets   []uint64
	startTd        *big.Int

	startNum     *uint64
	hashes       []common.Hash
	writtenBytes uint64
}

// NewBuilder returns a new EraE Builder writing into the given io.Writer.
func NewBuilder2(w io.Writer) *Builder2 {
	buf := bytes.NewBuffer(nil)
	return &Builder2{
		w:   e2store.NewWriter(w),
		buf: buf,
		sn:  snappy.NewBufferedWriter(buf),
	}
}

func (b *Builder) Add(block *types.Block, receipts types.Receipts, td *big.Int, proofRLP []byte) error {
	if len(b.headersRLP) >= MaxEraESize {
		return fmt.Errorf("exceeds maximum batch size of %d", MaxEraESize)
	}

	headerb, err := b.encodeHeader(block.Header())
	if err != nil {
		return fmt.Errorf("failed to encode header: %w", err)
	}
	bodyb, err := b.encodeBody(block.Body())
	if err != nil {
		return fmt.Errorf("failed to encode body: %w", err)
	}
	receiptsb, err := b.encodeReceipts(receipts)
	if err != nil {
		return fmt.Errorf("failed to encode receipts: %w", err)
	}

	tdbytes := uint256LE(td)

	if b.startNum == nil {
		start := block.NumberU64()
		b.startNum = &start
		_, err := b.w.Write(TypeVersion, nil)
		if err != nil {
			return fmt.Errorf("failed to write version entry: %w", err)
		}
		b.writtenBytes += 8
	}

	b.headersRLP = append(b.headersRLP, headerb)
	b.bodiesRLP = append(b.bodiesRLP, bodyb)
	b.receiptsRLP = append(b.receiptsRLP, receiptsb)
	b.tds = append(b.tds, tdbytes)
	b.proofsRLP = append(b.proofsRLP, proofRLP)
	b.hashes = append(b.hashes, block.Hash())
	return nil
}

func (b *Builder2) Finalize(common.Hash, error) {
	if b.startNum == nil {
		return fmt.Errorf("no blocks added, cannot finalize")
	}

	offs := snappy.Encode(b.buf, b.headersRLP)
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
