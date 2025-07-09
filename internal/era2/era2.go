package era2

import (
	"encoding/binary"
	"fmt"
	"io"
	"math/big"
	"os"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/internal/era/e2store"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/klauspost/compress/snappy"
)

type meta struct {
	start     uint64 // start block number
	count     uint64 // number of blocks in the era
	compcount uint64 // number of properties
	filelen   int64  // length of the file in bytes
}

type BlockProofHistoricalHashesAccumulator [15]common.Hash // 15 * 32 = 480 bytes

// BlockProofHistoricalRoots – Altair / Bellatrix historical_roots branch.
type BlockProofHistoricalRoots struct {
	BeaconBlockProof    [14]common.Hash // 448
	BeaconBlockRoot     common.Hash     // 32
	ExecutionBlockProof [11]common.Hash // 352
	Slot                uint64          // 8  => 840 bytes
}

// BlockProofHistoricalSummariesCapella – Capella historical_summaries branch.
type BlockProofHistoricalSummariesCapella struct {
	BeaconBlockProof    [13]common.Hash // 416
	BeaconBlockRoot     common.Hash     // 32
	ExecutionBlockProof [11]common.Hash // 352
	Slot                uint64          // 8  => 808 bytes
}

// BlockProofHistoricalSummariesDeneb – Deneb historical_summaries branch.
type BlockProofHistoricalSummariesDeneb struct {
	BeaconBlockProof    [13]common.Hash // 416
	BeaconBlockRoot     common.Hash     // 32
	ExecutionBlockProof [12]common.Hash // 384
	Slot                uint64          // 8  => 840 bytes
}

// BlockAccumulatorRoot is the SSZ hash tree root of the Era2 block accumulator.

type ReadAtSeekCloser interface {
	io.ReaderAt
	io.Seeker
	io.Closer
}

type Era2 struct {
	f                                                 ReadAtSeekCloser
	s                                                 *e2store.Reader
	m                                                 meta // metadata for the era2 file
	mu                                                *sync.Mutex
	headeroff, bodyoff, receiptsoff, tdoff, proofsoff []uint64 // offsets for each entry type
	indstart                                          int64
	rootheader                                        uint64 // offset of the root header in the file if present
	prooftype                                         uint16
}

func Open(path string) (*Era2, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	e := &Era2{f: f, s: e2store.NewReader(f)}
	if err := e.loadIndex(); err != nil {
		f.Close()
		return nil, err
	}
	return e, nil
}

func (e *Era2) Close() error {
	if e.f == nil {
		return nil
	}
	err := e.f.Close()
	e.f = nil
	return err
}

func (e *Era2) Start() uint64 {
	return e.m.start
}

func (e *Era2) Count() uint64 {
	return e.m.count
}

func (e *Era2) GetBlockByNumber(blockNum uint64) (*types.Block, error) {
	h, err := e.getHeader(blockNum)
	if err != nil {
		return nil, err
	}
	b, err := e.getBody(blockNum)
	if err != nil {
		return nil, err
	}
	return types.NewBlockWithHeader(h).WithBody(*b), nil
}

func (e *Era2) getHeader(blockNum uint64) (*types.Header, error) {
	if blockNum < e.m.start || blockNum >= e.m.start+e.m.count {
		return nil, fmt.Errorf("block number %d out of range [%d, %d)", blockNum, e.m.start, e.m.start+e.m.count)
	}
	r, _, err := e.s.ReaderAt(TypeCompressedHeader, int64(e.headeroff[blockNum-e.m.start]))
	if err != nil {
		return nil, err
	}
	r = snappy.NewReader(r)

	var h types.Header
	return &h, rlp.Decode(r, &h)
}

func (e *Era2) getBody(blockNum uint64) (*types.Body, error) {
	if blockNum < e.m.start || blockNum >= e.m.start+e.m.count {
		return nil, fmt.Errorf("block number %d out of range [%d, %d)", blockNum, e.m.start, e.m.start+e.m.count)
	}
	r, _, err := e.s.ReaderAt(TypeCompressedBody, int64(e.bodyoff[blockNum-e.m.start]))
	if err != nil {
		return nil, err
	}
	r = snappy.NewReader(r)

	var b types.Body
	return &b, rlp.Decode(r, &b)
}

func (e *Era2) getTD(blockNum uint64) (*big.Int, error) {
	if blockNum < e.m.start || blockNum >= e.m.start+e.m.count {
		return nil, fmt.Errorf("block number %d out of range [%d, %d)",
			blockNum, e.m.start, e.m.start+e.m.count)
	}
	if len(e.tdoff) == 0 {
		return nil, fmt.Errorf("total-difficulty section not present")
	}

	r, _, err := e.s.ReaderAt(TypeTotalDifficulty, int64(e.tdoff[blockNum-e.m.start]))
	if err != nil {
		return nil, err
	}
	buf, _ := io.ReadAll(r)
	td := new(big.Int).SetBytes(reverseOrder(buf))
	return td, nil
}

func (e *Era2) GetRawBodyFrameByNumber(blockNum uint64) ([]byte, error) {
	if blockNum < e.m.start || blockNum >= e.m.start+e.m.count {
		return nil, fmt.Errorf("block number %d out of range [%d, %d)", blockNum, e.m.start, e.m.start+e.m.count)
	}
	r, _, err := e.s.ReaderAt(TypeCompressedBody, int64(e.bodyoff[blockNum-e.m.start]))
	if err != nil {
		return nil, err
	}
	return io.ReadAll(r)
}

func (e *Era2) GetRawReceiptsFrameByNumber(blockNum uint64) ([]byte, error) {
	if blockNum < e.m.start || blockNum >= e.m.start+e.m.count {
		return nil, fmt.Errorf("block number %d out of range [%d, %d)", blockNum, e.m.start, e.m.start+e.m.count)
	}
	r, _, err := e.s.ReaderAt(TypeCompressedReceipts, int64(e.receiptsoff[blockNum-e.m.start]))
	if err != nil {
		return nil, err
	}
	return io.ReadAll(r)
}

func (e *Era2) GetRawProofFrameByNumber(blockNum uint64) ([]byte, error) {
	if len(e.proofsoff) == 0 {
		return nil, fmt.Errorf("proofs section not present")
	}
	if blockNum < e.m.start || blockNum >= e.m.start+e.m.count {
		return nil, fmt.Errorf("block number %d out of range [%d, %d)", blockNum, e.m.start, e.m.start+e.m.count)
	}
	r, _, err := e.s.ReaderAt(e.prooftype, int64(e.proofsoff[blockNum-e.m.start]))
	if err != nil {
		return nil, err
	}
	return io.ReadAll(r)
}

func (e *Era2) rawPayload(abs uint64) ([]byte, error) {
	sr := io.NewSectionReader(e.f, int64(abs), e.indstart-int64(abs))
	return io.ReadAll(sr)
}

func (e *Era2) snappyPayload(abs uint64) (io.Reader, error) {
	sr := io.NewSectionReader(e.f, int64(abs), e.indstart-int64(abs))
	return snappy.NewReader(sr), nil
}

func (e *Era2) loadIndex() error {
	var err error
	e.m.filelen, err = e.f.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}

	b := make([]byte, 16)
	if _, err = e.f.ReadAt(b, e.m.filelen-16); err != nil {
		return err
	}
	e.m.compcount = binary.LittleEndian.Uint64(b[0:8])
	e.m.count = binary.LittleEndian.Uint64(b[8:16])

	payloadlen := 8 + 8*e.m.count*e.m.compcount + 16 // 8 for start block, 8 per property per block, 16 for the number of properties and the number of blocks
	tlvstart := e.m.filelen - int64(payloadlen) - 8
	e.indstart = tlvstart

	_, err = e.f.ReadAt(b[:8], tlvstart+8)
	if err != nil {
		return err
	}

	e.m.start = binary.LittleEndian.Uint64(b[:8])
	num := int(e.m.compcount)

	totaloffsets := num * int(e.m.count)
	offBytes := make([]byte, totaloffsets*8)
	offsetArea := tlvstart + 8 + 8 // 8 for the header size, 8 for the start number
	_, err = e.f.ReadAt(offBytes, offsetArea)
	if err != nil {
		return err
	}

	toAbs := func(i int) uint64 {
		rel := binary.LittleEndian.Uint64(offBytes[i*8 : i*8+8])
		return uint64(int64(rel) + e.indstart)
	}

	e.headeroff = make([]uint64, e.m.count)
	e.bodyoff = make([]uint64, e.m.count)
	e.receiptsoff = make([]uint64, e.m.count)
	if num > 3 {
		e.tdoff = make([]uint64, e.m.count)
	}
	if num > 4 {
		e.proofsoff = make([]uint64, e.m.count)
	}
	for i := uint64(0); i < e.m.count; i++ {
		base := int(i * uint64(num))
		e.headeroff[i] = toAbs(base)
		e.bodyoff[i] = toAbs(base + 1)
		e.receiptsoff[i] = toAbs(base + 2)
		if num > 3 {
			e.tdoff[i] = toAbs(base + 3)
		}
		if num > 4 {
			e.proofsoff[i] = toAbs(base + 4)
		}
	}

	if len(e.proofsoff) > 0 {
		typ, _, perr := e.s.ReadMetadataAt(int64(e.proofsoff[0]))
		if perr != nil {
			return fmt.Errorf("read proof header: %w", perr)
		}
		e.prooftype = typ
	}

	var off int64 = 0 // start at byte-0 of file

	for off < e.indstart { // never enter the Block-Index TLV
		typ, length, err := e.s.ReadMetadataAt(off)
		if err != nil {
			return fmt.Errorf("error reading TLV at offset %d: %w", off, err)
		}
		if typ == TypeAccumulatorRoot {
			e.rootheader = uint64(off) // remember absolute header offset
			break                      // found
		}
		off += 8 + int64(length) // headersize plus length to jump to next TLV header
	}
	return nil
}

func (e *Era2) BatchRange(first, count uint64, wantHdr, wantBody, wantRec, wantPrf bool) (hdrs []*types.Header, bods []*types.Body, recs []types.Receipts, prfs [][]byte, err error) {
	if count == 0 {
		err = fmt.Errorf("count must be > 0")
		return
	}
	if first < e.m.start || first+count > e.m.start+e.m.count {
		err = fmt.Errorf("range [%d,%d) out of bounds", first, first+count)
		return
	}

	idx := first - e.m.start
	if wantHdr {
		hdrs = make([]*types.Header, count)
	}
	if wantBody {
		bods = make([]*types.Body, count)
	}
	if wantRec {
		recs = make([]types.Receipts, count)
	}
	if wantPrf {
		prfs = make([][]byte, count)
	}

	for i := uint64(0); i < count; i++ {
		id := idx + i

		if wantHdr {
			r, _, er := e.s.ReaderAt(TypeCompressedHeader, int64(e.headeroff[id]))
			if er != nil {
				err = er
				return
			}
			if er = rlp.Decode(snappy.NewReader(r), &hdrs[i]); er != nil {
				err = er
				return
			}
		}
		if wantBody {
			r, _, er := e.s.ReaderAt(TypeCompressedBody, int64(e.bodyoff[id]))
			if er != nil {
				err = er
				return
			}
			if er = rlp.Decode(snappy.NewReader(r), &bods[i]); er != nil {
				err = er
				return
			}
		}
		if wantRec {
			r, _, er := e.s.ReaderAt(TypeCompressedReceipts, int64(e.receiptsoff[id]))
			if er != nil {
				err = er
				return
			}
			if er = rlp.Decode(snappy.NewReader(r), &recs[i]); er != nil {
				err = er
				return
			}
		}
		if wantPrf {
			if len(e.proofsoff) == 0 {
				err = fmt.Errorf("proofs section not present")
				return
			}
			r, _, er := e.s.ReaderAt(e.prooftype, int64(e.proofsoff[id])) // type already checked when writing
			if er != nil {
				err = er
				return
			}
			prfs[i], _ = io.ReadAll(r)
		}
	}
	return
}

func reverseOrder32(b []byte) []byte {
	for i := 0; i < 16; i++ {
		b[i], b[32-i-1] = b[32-i-1], b[i]
	}
	return b
}
