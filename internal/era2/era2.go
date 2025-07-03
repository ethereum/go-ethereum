package era

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
	f   ReadAtSeekCloser
	s   *e2store.Reader
	m   meta // metadata for the era2 file
	mu  *sync.Mutex
	buf [8]byte // buffer reading entry offset

	headeroff, bodyoff, receiptsoff, tdoff, proofsoff []uint64 // offsets for each entry type
	indstart                                          int64
}

func OpenEra(filename string) (*Era2, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	e := &Era2{f: f, s: e2store.NewReader(f), mu: new(sync.Mutex)}
	if err := e.loadIndex(); err != nil {
		f.Close()
		return nil, err
	}
	return e, nil
}

func (e *Era2) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.f == nil {
		return nil
	}
	err := e.f.Close()
	e.f = nil
	return err
}

func (e *Era2) Start() uint64 {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.m.start
}

func (e *Era2) Count() uint64 {
	e.mu.Lock()
	defer e.mu.Unlock()
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
	e.mu.Lock()
	defer e.mu.Unlock()
	if blockNum < e.m.start || blockNum >= e.m.start+e.m.count {
		return nil, fmt.Errorf("block number %d out of range [%d, %d)", blockNum, e.m.start, e.m.start+e.m.count)
	}
	r, err := e.snappyPayload(e.headeroff[blockNum-e.m.start])
	if err != nil {
		return nil, fmt.Errorf("error reading header for block %d: %w", blockNum, err)
	}
	var h types.Header
	return &h, rlp.Decode(r, &h)
}

func (e *Era2) getBody(blockNum uint64) (*types.Body, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if blockNum < e.m.start || blockNum >= e.m.start+e.m.count {
		return nil, fmt.Errorf("block number %d out of range [%d, %d)", blockNum, e.m.start, e.m.start+e.m.count)
	}
	r, err := e.snappyPayload(e.bodyoff[blockNum-e.m.start])
	if err != nil {
		return nil, fmt.Errorf("error reading body for block %d: %w", blockNum, err)
	}
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

	start := e.tdoff[blockNum-e.m.start]
	buf := make([]byte, 32)

	if _, err := e.f.ReadAt(buf, int64(start)); err != nil {
		return nil, fmt.Errorf("error reading total difficulty for block %d: %w",
			blockNum, err)
	}
	td := new(big.Int).SetBytes(reverseOrder(buf))
	return td, nil
}

func (e *Era2) GetRawBodyFrameByNumber(n uint64) ([]byte, error) {
	if n < e.m.start || n >= e.m.start+e.m.count {
		return nil, fmt.Errorf("block number %d out of range [%d, %d)", n, e.m.start, e.m.start+e.m.count)
	}
	start := e.bodyoff[n-e.m.start]
	end := e.nextBoundary(n, e.bodyoff, e.receiptsoff) // receipts section is the next safest fallback
	length := end - start
	sr := io.NewSectionReader(e.f, int64(start), int64(length))
	return io.ReadAll(sr)
}

func (e *Era2) GetRawReceiptsFrameByNumber(n uint64) ([]byte, error) {
	if n < e.m.start || n >= e.m.start+e.m.count {
		return nil, fmt.Errorf("block number %d out of range [%d, %d)", n, e.m.start, e.m.start+e.m.count)
	}
	start := e.receiptsoff[n-e.m.start]
	end := e.nextBoundary(n, e.receiptsoff, e.tdoff) // TD sec. is next fallback
	length := end - start
	sr := io.NewSectionReader(e.f, int64(start), int64(length))
	return io.ReadAll(sr)
}

func (e *Era2) rawPayload(abs uint64) ([]byte, error) {
	sr := io.NewSectionReader(e.f, int64(abs), e.indstart-int64(abs))
	return io.ReadAll(sr)
}

func (e *Era2) snappyPayload(abs uint64) (io.Reader, error) {
	sr := io.NewSectionReader(e.f, int64(abs), e.indstart-int64(abs))
	return snappy.NewReader(sr), nil
}

func (e *Era2) nextBoundary(idx uint64, sameSec []uint64, nextSec []uint64) uint64 {
	local := idx - e.m.start
	// next frame in the same section
	if local+1 < uint64(len(sameSec)) {
		return sameSec[local+1]
	}
	// first frame of the NEXT section (if present)
	if len(nextSec) > 0 {
		return nextSec[0]
	}
	// otherwise clamp to start of BlockIndex
	return uint64(e.indstart)
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
	return nil
}

func reverseOrder32(b []byte) []byte {
	for i := 0; i < 16; i++ {
		b[i], b[32-i-1] = b[32-i-1], b[i]
	}
	return b
}
