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
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"slices"
	"strings"

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

// Open accesses the era file. The path is used to parse the profile postfix
// (per the Ere spec filename convention); files written with the "noreceipts"
// profile are rejected because the positional index reader assumes receipts
// are present.
func Open(path string) (*Era, error) {
	if err := checkProfile(filepath.Base(path)); err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	e := &Era{f: f, s: e2store.NewReader(f)}
	if err := e.loadIndex(); err != nil {
		f.Close()
		return nil, err
	}
	if err := e.checkComponents(); err != nil {
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

// From returns an Era backed by f. Since no filename is available, the profile
// cannot be inspected; the component count is still validated against the
// supported layouts (header, body, receipts, [td]).
func From(f era.ReadAtSeekCloser) (era.Era, error) {
	e := &Era{f: f, s: e2store.NewReader(f)}
	if err := e.loadIndex(); err != nil {
		f.Close()
		return nil, err
	}
	if err := e.checkComponents(); err != nil {
		f.Close()
		return nil, err
	}
	return e, nil
}

// checkProfile inspects the profile postfix(es) in an Ere filename and rejects
// any combination this reader can't safely decode. The reader maps components
// by fixed positions (header, body, receipts, td?, proof?), so a file written
// with the "noreceipts" profile would silently shift TD into the receipts slot.
//
// The Ere format itself does not require a particular filename, so this check
// is permissive about non-conforming names: validation only kicks in when a
// profile postfix is actually present.
func checkProfile(name string) error {
	name = strings.TrimSuffix(name, ".ere")
	parts := strings.Split(name, "-")
	if len(parts) <= 3 {
		return nil // no profile postfix to validate
	}
	for _, p := range parts[3:] {
		if p == "noreceipts" {
			return fmt.Errorf("Ere file %q uses the noreceipts profile, which is not supported", name)
		}
	}
	return nil
}

// checkComponents verifies the file's component count matches what this reader
// supports. The reader assumes the fixed positional layout
// (header, body, receipts, td?, proof?), and the builder in this package only
// produces files with 3 (post-merge) or 4 (pre-merge / transition) components.
// Files with 2 (noreceipts) or 5 (proofs present) components are rejected.
func (e *Era) checkComponents() error {
	if e.m.components < 3 || e.m.components > 4 {
		return fmt.Errorf("unsupported Ere component count %d (reader expects header, body, receipts, and optional total difficulty)", e.m.components)
	}
	return nil
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

// InitialTD returns initial total difficulty before the difficulty of the
// first block of the Era is applied. Returns an error if TD is not available
// (e.g., post-merge epoch).
func (e *Era) InitialTD() (*big.Int, error) {
	// Check if TD component exists.
	if int(td) >= int(e.m.components) {
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

// loadIndex loads in the index table containing all offsets and caches it.
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
	return nil
}

// headerOff, bodyOff, receiptOff, and tdOff return the offsets of the respective components for a given block number.
func (e *Era) headerOff(num uint64) (int64, error)  { return e.indexOffset(num, header) }
func (e *Era) bodyOff(num uint64) (int64, error)    { return e.indexOffset(num, body) }
func (e *Era) receiptOff(num uint64) (int64, error) { return e.indexOffset(num, receipts) }
func (e *Era) tdOff(num uint64) (int64, error)      { return e.indexOffset(num, td) }

// indexOffset calculates offset to a certain component for a block number within a file.
func (e *Era) indexOffset(n uint64, component componentType) (int64, error) {
	if n < e.m.start || n >= e.m.start+e.m.count {
		return 0, fmt.Errorf("block %d out of range [%d,%d)", n, e.m.start, e.m.start+e.m.count)
	}
	if int(component) >= int(e.m.components) {
		return 0, fmt.Errorf("component %d not present", component)
	}

	payloadlen := 8 + 8*e.m.count*e.m.components + 16 // 8 for start block, 8 per property per block, 16 for the number of properties and the number of blocks
	indstart := e.m.length - int64(payloadlen) - 8

	rec := (n-e.m.start)*e.m.components + uint64(component)
	pos := indstart + 8 + 8 + int64(rec*8)

	var buf [8]byte
	if _, err := e.f.ReadAt(buf[:], pos); err != nil {
		return 0, err
	}
	rel := binary.LittleEndian.Uint64(buf[:])
	return int64(rel) + indstart, nil
}

// metadata contains the information about the era file that is written into the file.
type metadata struct {
	start      uint64 // start block number
	count      uint64 // number of blocks in the era
	components uint64 // number of properties
	length     int64  // length of the file in bytes
}

// componentType represents the integer form of a specific type that can be present in the era file.
type componentType int

// header, body, receipts, td, and proof are the different types of components
// that can be present in the era file. The Ere spec defines receipts, td, and
// proof as independently optional, but this reader maps components to their
// position in the index using this fixed enum. That positional mapping is only
// safe as long as receipts are present (no "noreceipts" profile) — Open() and
// From() enforce this via checkProfile and checkComponents.
const (
	header componentType = iota
	body
	receipts
	td
	proof
)
