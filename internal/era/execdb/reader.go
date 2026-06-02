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

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/internal/era"
	"github.com/ethereum/go-ethereum/internal/era/e2store"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/klauspost/compress/snappy"
)

// Era object represents an era file that contains blocks and their components.
type Era struct {
	f era.ReadAtSeekCloser
	s *e2store.Reader
	m metadata // metadata for the Era file
}

// Filename returns a recognizable filename for an Ere file.
// The filename uses the last block hash to uniquely identify the epoch's content.
//
// Files produced by this builder do not include Proof entries, so the
// "noproofs" profile postfix is appended per the Ere spec.
func Filename(network string, epoch int, lastBlockHash common.Hash) string {
	return fmt.Sprintf("%s-%05d-%s-noproofs.ere", network, epoch, lastBlockHash.Hex()[2:10])
}

// Open accesses the era file.
func Open(path string) (*Era, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	e, err := from(f)
	if err != nil {
		f.Close()
		return nil, err
	}
	return e, nil
}

// Close closes the era file safely.
func (e *Era) Close() error {
	if e.f == nil {
		return nil
	}
	err := e.f.Close()
	e.f = nil
	return err
}

// From returns an Era backed by f.
func From(f era.ReadAtSeekCloser) (era.Era, error) {
	e, err := from(f)
	if err != nil {
		f.Close()
		return nil, err
	}
	return e, nil
}

func from(f era.ReadAtSeekCloser) (*Era, error) {
	e := &Era{f: f, s: e2store.NewReader(f)}
	if err := e.loadIndex(); err != nil {
		return nil, err
	}
	return e, nil
}

// Start retrieves the starting block number.
func (e *Era) Start() uint64 {
	return e.m.start
}

// Count retrieves the count of blocks present.
func (e *Era) Count() uint64 {
	return e.m.count
}

// Iterator returns an iterator over the era file.
func (e *Era) Iterator() (era.Iterator, error) {
	return NewIterator(e)
}

// GetBlockByNumber retrieves the block if present within the era file.
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

// GetHeader retrieves the header from the era file through the cached offset table.
func (e *Era) GetHeader(num uint64) (*types.Header, error) {
	off, err := e.headerOff(num)
	if err != nil {
		return nil, err
	}

	r, _, err := e.s.ReaderAt(era.TypeCompressedHeader, off)
	if err != nil {
		return nil, err
	}

	r = snappy.NewReader(r)
	var h types.Header
	return &h, rlp.Decode(r, &h)
}

// GetBody retrieves the body from the era file through cached offset table.
func (e *Era) GetBody(num uint64) (*types.Body, error) {
	off, err := e.bodyOff(num)
	if err != nil {
		return nil, err
	}

	r, _, err := e.s.ReaderAt(era.TypeCompressedBody, off)
	if err != nil {
		return nil, err
	}

	r = snappy.NewReader(r)
	var b types.Body
	return &b, rlp.Decode(r, &b)
}

// GetTD retrieves the td from the era file through cached offset table.
func (e *Era) GetTD(blockNum uint64) (*big.Int, error) {
	off, err := e.tdOff(blockNum)
	if err != nil {
		return nil, err
	}
	r, _, err := e.s.ReaderAt(era.TypeTotalDifficulty, off)
	if err != nil {
		return nil, err
	}
	buf, _ := io.ReadAll(r)
	slices.Reverse(buf)
	td := new(big.Int).SetBytes(buf)
	return td, nil
}

// GetRawBodyByNumber returns the RLP-encoded body for the given block number.
func (e *Era) GetRawBodyByNumber(blockNum uint64) ([]byte, error) {
	off, err := e.bodyOff(blockNum)
	if err != nil {
		return nil, err
	}
	r, _, err := e.s.ReaderAt(era.TypeCompressedBody, off)
	if err != nil {
		return nil, err
	}
	r = snappy.NewReader(r)
	return io.ReadAll(r)
}

// GetRawReceiptsByNumber returns the RLP-encoded receipts for the given block number.
func (e *Era) GetRawReceiptsByNumber(blockNum uint64) ([]byte, error) {
	off, err := e.receiptOff(blockNum)
	if err != nil {
		return nil, err
	}
	r, _, err := e.s.ReaderAt(era.TypeCompressedSlimReceipts, off)
	if err != nil {
		return nil, err
	}
	r = snappy.NewReader(r)
	return io.ReadAll(r)
}

// HasComponent reports whether the given component is recorded in the file's
// index, as detected from the on-disk e2store type tags.
func (e *Era) HasComponent(c componentType) bool {
	_, ok := e.m.layout[c]
	return ok
}

// InitialTD returns initial total difficulty before the difficulty of the
// first block of the Era is applied. Returns an error if TD is not available
// (e.g., post-merge epoch).
func (e *Era) InitialTD() (*big.Int, error) {
	// Check if TD component exists.
	if !e.HasComponent(td) {
		return nil, fmt.Errorf("total difficulty not available in this epoch")
	}

	// Get first header to read its difficulty.
	header, err := e.GetHeader(e.m.start)
	if err != nil {
		return nil, fmt.Errorf("read first header: %w", err)
	}

	// Get TD after first block using the index.
	firstTD, err := e.GetTD(e.m.start)
	if err != nil {
		return nil, fmt.Errorf("read first TD: %w", err)
	}

	// Initial TD = TD[0] - Difficulty[0]
	return new(big.Int).Sub(firstTD, header.Difficulty), nil
}

// Accumulator reads the accumulator entry if present. Only pre-merge and
// merge-transition Ere files contain one.
func (e *Era) Accumulator() (common.Hash, error) {
	entry, err := e.s.Find(era.TypeAccumulator)
	if err != nil {
		return common.Hash{}, err
	}
	return common.BytesToHash(entry.Value), nil
}

// loadIndex loads in the index table trailer (start, count, component-count)
// and then derives the component→slot layout from the on-disk type tags.
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
	_, err = e.f.ReadAt(b[:8], tlvstart+8)
	if err != nil {
		return err
	}
	e.m.start = binary.LittleEndian.Uint64(b[:8])

	layout, err := e.detectLayout()
	if err != nil {
		return err
	}
	e.m.layout = layout
	return nil
}

// detectLayout reads the e2store type tag at each component slot of the first
// block and builds a componentType→slot map, so components are looked up by tag
// rather than by a fixed position.
func (e *Era) detectLayout() (map[componentType]int, error) {
	if e.m.count == 0 {
		return nil, errors.New("Ere file contains no blocks")
	}
	tagToComponent := map[uint16]componentType{
		era.TypeCompressedHeader:       header,
		era.TypeCompressedBody:         body,
		era.TypeCompressedSlimReceipts: receipts,
		era.TypeTotalDifficulty:        td,
		era.TypeProof:                  proof,
	}
	layout := make(map[componentType]int, e.m.components)
	for slot := 0; slot < int(e.m.components); slot++ {
		off, err := e.slotOffset(0, slot)
		if err != nil {
			return nil, fmt.Errorf("read slot %d offset: %w", slot, err)
		}
		typ, _, err := e.s.ReadMetadataAt(off)
		if err != nil {
			return nil, fmt.Errorf("read slot %d type tag: %w", slot, err)
		}
		comp, ok := tagToComponent[typ]
		if !ok {
			return nil, fmt.Errorf("unknown e2store type 0x%04x at index slot %d", typ, slot)
		}
		if existing, dup := layout[comp]; dup {
			return nil, fmt.Errorf("duplicate component %d at slots %d and %d", comp, existing, slot)
		}
		layout[comp] = slot
	}
	if _, ok := layout[header]; !ok {
		return nil, errors.New("Ere index has no header component")
	}
	if _, ok := layout[body]; !ok {
		return nil, errors.New("Ere index has no body component")
	}
	return layout, nil
}

// slotOffset returns the absolute file offset of the entry at the given slot
// of the given block index (0 = first block in file).
func (e *Era) slotOffset(blockIdx uint64, slot int) (int64, error) {
	payloadlen := 8 + 8*e.m.count*e.m.components + 16
	indstart := e.m.length - int64(payloadlen) - 8

	rec := blockIdx*e.m.components + uint64(slot)
	pos := indstart + 8 + 8 + int64(rec*8)

	var buf [8]byte
	if _, err := e.f.ReadAt(buf[:], pos); err != nil {
		return 0, err
	}
	rel := binary.LittleEndian.Uint64(buf[:])
	return int64(rel) + indstart, nil
}

// headerOff, bodyOff, receiptOff, and tdOff return the offsets of the respective components for a given block number.
func (e *Era) headerOff(num uint64) (int64, error)  { return e.indexOffset(num, header) }
func (e *Era) bodyOff(num uint64) (int64, error)    { return e.indexOffset(num, body) }
func (e *Era) receiptOff(num uint64) (int64, error) { return e.indexOffset(num, receipts) }
func (e *Era) tdOff(num uint64) (int64, error)      { return e.indexOffset(num, td) }

// indexOffset calculates offset to a certain component for a block number
// within a file.
func (e *Era) indexOffset(n uint64, component componentType) (int64, error) {
	if n < e.m.start || n >= e.m.start+e.m.count {
		return 0, fmt.Errorf("block %d out of range [%d,%d)", n, e.m.start, e.m.start+e.m.count)
	}
	slot, ok := e.m.layout[component]
	if !ok {
		return 0, fmt.Errorf("component %d not present in this Ere file", component)
	}
	return e.slotOffset(n-e.m.start, slot)
}

// metadata contains the information about the era file that is written into the file.
type metadata struct {
	start      uint64                // start block number
	count      uint64                // number of blocks in the era
	components uint64                // number of slots per block in the index
	layout     map[componentType]int // component → slot index, derived from on-disk type tags
	length     int64                 // length of the file in bytes
}

// componentType identifies a kind of per-block entry (header, body, etc.).
type componentType int

// td and proof are independently optional per the Ere spec.
const (
	header componentType = iota
	body
	receipts
	td
	proof
)
