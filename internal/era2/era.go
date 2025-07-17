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

import (
	"encoding/binary"
	"fmt"
	"io"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/internal/era/e2store"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/klauspost/compress/snappy"
)

type metadata struct {
	start      uint64 // start block number
	count      uint64 // number of blocks in the era
	components uint64 // number of properties
	length     int64  // length of the file in bytes
}

const (
	ProofNone variant = iota
	proofHHA
	proofRoots
	proofCapella
	proofDeneb
)

const (
	compHeader   = 0
	compBody     = 1
	compReceipts = 2
	compTD       = 3
	compProof    = 4
)

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

type Proof interface {
	EncodeRLP(w io.Writer) error
	DecodeRlP(s *rlp.Stream) error
	Variant() variant
}

type hhaAlias BlockProofHistoricalHashesAccumulator // alias ⇒ no EncodeRLP method

func (p *BlockProofHistoricalHashesAccumulator) EncodeRLP(w io.Writer) error {
	payload := []interface{}{uint16(proofHHA), hhaAlias(*p)}
	return rlp.Encode(w, payload)
}

func (p *BlockProofHistoricalHashesAccumulator) Variant() variant { return proofHHA }

type rootsAlias BlockProofHistoricalRoots

func (p *BlockProofHistoricalRoots) EncodeRLP(w io.Writer) error {
	payload := []interface{}{uint16(proofRoots), rootsAlias(*p)}
	return rlp.Encode(w, payload)
}

func (*BlockProofHistoricalRoots) Variant() variant { return proofRoots }

type capellaAlias BlockProofHistoricalSummariesCapella

func (p *BlockProofHistoricalSummariesCapella) EncodeRLP(w io.Writer) error {
	payload := []interface{}{uint16(proofCapella), capellaAlias(*p)}
	return rlp.Encode(w, payload)
}

func (*BlockProofHistoricalSummariesCapella) Variant() variant { return proofCapella }

type denebAlias BlockProofHistoricalSummariesDeneb

func (p *BlockProofHistoricalSummariesDeneb) EncodeRLP(w io.Writer) error {
	payload := []interface{}{uint16(proofDeneb), denebAlias(*p)}
	return rlp.Encode(w, payload)
}

func (*BlockProofHistoricalSummariesDeneb) Variant() variant { return proofDeneb }

func variantOf(p Proof) variant {
	if p == nil {
		return ProofNone
	}
	return p.Variant()
}

type ReadAtSeekCloser interface {
	io.ReaderAt
	io.Seeker
	io.Closer
}

type Era struct {
	f          ReadAtSeekCloser
	s          *e2store.Reader
	m          metadata // metadata for the Era file
	indstart   int64
	rootheader uint64 // offset of the root header in the file if present
}

// Opens era file
func Open(path string) (*Era, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	e := &Era{f: f, s: e2store.NewReader(f)}
	if err := e.loadIndex(); err != nil {
		f.Close()
		return nil, err
	}
	return e, nil
}

// Closes era file
func (e *Era) Close() error {
	if e.f == nil {
		return nil
	}
	err := e.f.Close()
	e.f = nil
	return err
}

// retrieves starting block number
func (e *Era) Start() uint64 {
	return e.m.start
}

// retrieves count of blocks present
func (e *Era) Count() uint64 {
	return e.m.count
}

// retrieves the block if present within the era file
func (e *Era) GetBlockByNumber(blockNum uint64) (*types.Block, error) {
	h, err := e.GetHeader(blockNum)
	if err != nil {
		return nil, err
	}
	b, err := e.GetBody(blockNum)
	if err != nil {
		return nil, err
	}
	return types.NewBlockWithHeader(h).WithBody(*b), nil
}

// retrieves header from era file through the cached offset table
func (e *Era) GetHeader(num uint64) (*types.Header, error) {
	off, err := e.headerOff(num)
	if err != nil {
		return nil, err
	}

	r, _, err := e.s.ReaderAt(TypeCompressedHeader, int64(off))
	if err != nil {
		return nil, err
	}

	r = snappy.NewReader(r)
	var h types.Header
	return &h, rlp.Decode(r, &h)
}

// retrieves body from era file through cached offset table
func (e *Era) GetBody(num uint64) (*types.Body, error) {
	off, err := e.bodyOff(num)
	if err != nil {
		return nil, err
	}

	r, _, err := e.s.ReaderAt(TypeCompressedBody, int64(off))
	if err != nil {
		return nil, err
	}

	r = snappy.NewReader(r)
	var b types.Body
	return &b, rlp.Decode(r, &b)
}

// retrieves td from era file through cached offset table
func (e *Era) getTD(blockNum uint64) (*big.Int, error) {
	off, err := e.tdOff(blockNum)
	if err != nil {
		return nil, err
	}
	r, _, err := e.s.ReaderAt(TypeTotalDifficulty, int64(off))
	if err != nil {
		return nil, err
	}
	buf, _ := io.ReadAll(r)
	td := new(big.Int).SetBytes(reverseOrder(buf))
	return td, nil
}

// retrieves the raw body frame in bytes of a specific block
func (e *Era) GetRawBodyFrameByNumber(blockNum uint64) ([]byte, error) {
	off, err := e.bodyOff(blockNum)
	if err != nil {
		return nil, err
	}
	r, _, err := e.s.ReaderAt(TypeCompressedBody, int64(off))
	if err != nil {
		return nil, err
	}
	return io.ReadAll(r)
}

// retrieves the raw receipts frame in bytes of a specific block
func (e *Era) GetRawReceiptsFrameByNumber(blockNum uint64) ([]byte, error) {
	off, err := e.rcptOff(blockNum)
	if err != nil {
		return nil, err
	}
	r, _, err := e.s.ReaderAt(TypeCompressedReceipts, int64(off))
	if err != nil {
		return nil, err
	}
	return io.ReadAll(r)
}

// retrieves the raw proof frame in bytes of a specific block proof
func (e *Era) GetRawProofFrameByNumber(blockNum uint64) ([]byte, error) {
	off, err := e.proofOff(blockNum)
	if err != nil {
		return nil, err
	}
	r, _, err := e.s.ReaderAt(TypeProof, int64(off))
	if err != nil {
		return nil, err
	}
	return io.ReadAll(r)
}

// loads in the index table containing all offsets and caches it
func (e *Era) loadIndex() error {
	var err error
	e.m.length, err = e.f.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}

	b := make([]byte, 16)
	if _, err = e.f.ReadAt(b, e.m.length-16); err != nil {
		return err
	}
	e.m.components = binary.LittleEndian.Uint64(b[0:8])
	e.m.count = binary.LittleEndian.Uint64(b[8:16])

	payloadlen := 8 + 8*e.m.count*e.m.components + 16 // 8 for start block, 8 per property per block, 16 for the number of properties and the number of blocks
	tlvstart := e.m.length - int64(payloadlen) - 8
	e.indstart = tlvstart

	_, err = e.f.ReadAt(b[:8], tlvstart+8)
	if err != nil {
		return err
	}

	e.m.start = binary.LittleEndian.Uint64(b[:8])
	num := int(e.m.components)

	totaloffsets := num * int(e.m.count)
	offBytes := make([]byte, totaloffsets*8)
	offsetArea := tlvstart + 8 + 8 // 8 for the header size, 8 for the start number
	_, err = e.f.ReadAt(offBytes, offsetArea)
	if err != nil {
		return err
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

// Getter methods to calculate offset of a specific component in the file.
func (e *Era) headerOff(num uint64) (uint64, error) { return e.indexOffset(num, compHeader) }
func (e *Era) bodyOff(num uint64) (uint64, error)   { return e.indexOffset(num, compBody) }
func (e *Era) rcptOff(num uint64) (uint64, error)   { return e.indexOffset(num, compReceipts) }
func (e *Era) tdOff(num uint64) (uint64, error)     { return e.indexOffset(num, compTD) }
func (e *Era) proofOff(num uint64) (uint64, error)  { return e.indexOffset(num, compProof) }

// calculates offset to a certain component for a block number within a file.
func (e *Era) indexOffset(n uint64, comp int) (uint64, error) {
	if n < e.m.start || n >= e.m.start+e.m.count {
		return 0, fmt.Errorf("block %d out of range [%d,%d)", n, e.m.start, e.m.start+e.m.count)
	}
	if comp >= int(e.m.components) {
		return 0, fmt.Errorf("component %d not present", comp)
	}

	rec := (n-e.m.start)*e.m.components + uint64(comp)
	pos := e.indstart + 8 + 8 + int64(rec*8)

	var buf [8]byte
	if _, err := e.f.ReadAt(buf[:], pos); err != nil {
		return 0, err
	}
	rel := binary.LittleEndian.Uint64(buf[:])
	return uint64(int64(rel) + e.indstart), nil
}

// GetHeaders returns RLP-decoded headers.
func (e *Era) GetHeaders(first, count uint64) ([]*types.Header, error) {
	if count == 0 {
		return nil, fmt.Errorf("count must be > 0")
	}
	if first < e.m.start || first+count > e.m.start+e.m.count {
		return nil, fmt.Errorf("range [%d,%d) out of bounds", first, first+count)
	}

	out := make([]*types.Header, count)
	for i := uint64(0); i < count; i++ {
		n := first + i
		off, err := e.headerOff(n)
		if err != nil {
			return nil, err
		}
		r, _, err := e.s.ReaderAt(TypeCompressedHeader, int64(off))
		if err != nil {
			return nil, err
		}
		var h types.Header
		if err := rlp.Decode(snappy.NewReader(r), &h); err != nil {
			return nil, err
		}
		out[i] = &h
	}
	return out, nil
}

// GetBodies returns RLP-decoded bodies.
func (e *Era) GetBodies(first, count uint64) ([]*types.Body, error) {
	if count == 0 {
		return nil, fmt.Errorf("count must be > 0")
	}
	if first < e.m.start || first+count > e.m.start+e.m.count {
		return nil, fmt.Errorf("range [%d,%d) out of bounds", first, first+count)
	}

	out := make([]*types.Body, count)
	for i := uint64(0); i < count; i++ {
		n := first + i
		off, err := e.bodyOff(n)
		if err != nil {
			return nil, err
		}
		r, _, err := e.s.ReaderAt(TypeCompressedBody, int64(off))
		if err != nil {
			return nil, err
		}
		var b types.Body
		if err := rlp.Decode(snappy.NewReader(r), &b); err != nil {
			return nil, err
		}
		out[i] = &b
	}
	return out, nil
}

// GetReceipts returns RLP-decoded receipts.
func (e *Era) GetReceipts(first, count uint64) ([]types.Receipts, error) {
	if count == 0 {
		return nil, fmt.Errorf("count must be > 0")
	}
	if first < e.m.start || first+count > e.m.start+e.m.count {
		return nil, fmt.Errorf("range [%d,%d) out of bounds", first, first+count)
	}

	out := make([]types.Receipts, count)
	for i := uint64(0); i < count; i++ {
		n := first + i
		off, err := e.rcptOff(n)
		if err != nil {
			return nil, err
		}
		r, _, err := e.s.ReaderAt(TypeCompressedReceipts, int64(off))
		if err != nil {
			return nil, err
		}
		var rc types.Receipts
		if err := rlp.Decode(snappy.NewReader(r), &rc); err != nil {
			return nil, err
		}
		out[i] = rc
	}
	return out, nil
}
