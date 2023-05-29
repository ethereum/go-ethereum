// Copyright 2023 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package era

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math/big"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/internal/era/e2store"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/golang/snappy"
)

var (
	TypeVersion            uint16 = 0x3265
	TypeCompressedHeader   uint16 = 0x03
	TypeCompressedBody     uint16 = 0x04
	TypeCompressedReceipts uint16 = 0x05
	TypeTotalDifficulty    uint16 = 0x06
	TypeAccumulator        uint16 = 0x07
	TypeBlockIndex         uint16 = 0x3266

	MaxEra1Size = 8192
)

// Filename returns a recognizable Era1-formatted file name for the specified
// epoch and network.
func Filename(network string, epoch int, root common.Hash) string {
	return fmt.Sprintf("%s-%05d-%s.era1", network, epoch, root.Hex()[2:10])
}

// ReadDir reads all the era1 files in a directory for a given network.
// Format: <network>-<epoch>-<hexroot>.era1
func ReadDir(dir, network string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("error reading directory %s: %w", dir, err)
	}
	var (
		next = uint64(0)
		eras []string
	)
	for _, entry := range entries {
		if path.Ext(entry.Name()) != ".era1" {
			continue
		}
		parts := strings.Split(entry.Name(), "-")
		if len(parts) != 3 || parts[0] != network {
			// invalid era1 filename, skip
			continue
		}
		if epoch, err := strconv.ParseUint(parts[1], 10, 64); err != nil {
			return nil, fmt.Errorf("malformed era1 filename: %s", entry.Name())
		} else if epoch != next {
			return nil, fmt.Errorf("missing epoch %d", next)
		}
		next += 1
		eras = append(eras, entry.Name())
	}
	return eras, nil
}

// Reader reads an Era1 archive.
// See Builder documentation for a detailed explanation of the Era1 format.
type Reader struct {
	r io.ReaderAt
	e *e2store.Reader

	buf      [8]byte  // buffer reading entry offsets
	next     uint64   // next block to read
	length   int64    // total length of r
	metadata metadata // start, count info
}

type ReadAtSeeker interface {
	io.ReaderAt
	io.Seeker
}

// NewReader returns a new Reader instance.
func NewReader(r ReadAtSeeker) (*Reader, error) {
	length, err := r.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, err
	}
	m, err := readMetadata(r, length)
	if err != nil {
		return nil, err
	}
	return &Reader{
		r:        r,
		e:        e2store.NewReader(r),
		next:     m.start,
		length:   length,
		metadata: m,
	}, nil
}

// readOffset reads a specific block's offset from the block index. The value n
// is the absolute block number desired.
func (r *Reader) readOffset(n uint64) (int64, error) {
	var (
		firstIndex  = -8 - int64(r.metadata.count)*8      // size of count - index entries
		indexOffset = int64(n-r.metadata.start) * 8       // desired index * size of indexes
		offOffset   = r.length + firstIndex + indexOffset // offset of block offset
	)
	r.clearBuffer()
	if _, err := r.r.ReadAt(r.buf[:], offOffset); err != nil {
		return 0, err
	}
	// Since the block offset is relative from its location + size of index
	// value (8), we need to add it to it's offset to get the block's
	// absolute offset.
	return offOffset + 8 + int64(binary.LittleEndian.Uint64(r.buf[:])), nil
}

// Read reads one (block, receipts) tuple from an Era1 archive.
func (r *Reader) Read() (*types.Block, types.Receipts, error) {
	block, receipts, err := r.ReadBlockAndReceipts(r.next)
	if err != nil {
		return nil, nil, err
	}
	r.next += 1
	return block, receipts, nil
}

// readBlob reads an entry of data.
func (r *Reader) readEntry(n uint64, skip int) (*e2store.Entry, error) {
	if n < r.metadata.start || r.metadata.start+r.metadata.count < n {
		return nil, fmt.Errorf("request out-of-bounds: want %d, start: %d, count: %d", n, r.metadata.start, r.metadata.count)
	}
	// Read the specified block's offset from the block index.
	off, err := r.readOffset(n)
	if err != nil {
		return nil, fmt.Errorf("error reading block offset: %w", err)
	}
	// Skip to the requested entry.
	for i := 0; i < skip; i++ {
		if length, err := r.e.LengthAt(off); err != nil {
			return nil, err
		} else {
			off += length
		}
	}
	// Read entry.
	var entry e2store.Entry
	if _, err := r.e.ReadAt(&entry, off); err != nil {
		return nil, err
	}
	return &entry, nil
}

// readHeaderRLP reads the header number n RLP.
func (r *Reader) readHeaderRLP(n uint64) ([]byte, error) {
	e, err := r.readEntry(n, 0)
	if err != nil {
		return nil, err
	}
	if e.Type != TypeCompressedHeader {
		return nil, fmt.Errorf("expected header entry, got %x", e.Type)
	}
	return io.ReadAll(snappy.NewReader(bytes.NewReader(e.Value)))
}

// readBodyRLP reads the block body number n RLP.
func (r *Reader) readBodyRLP(n uint64) ([]byte, error) {
	e, err := r.readEntry(n, 1)
	if err != nil {
		return nil, err
	}
	if e.Type != TypeCompressedBody {
		return nil, fmt.Errorf("expected body entry, got %x", e.Type)
	}
	return io.ReadAll(snappy.NewReader(bytes.NewReader(e.Value)))
}

// readReceiptsRLP reads the receipts RLP associated with number n.
func (r *Reader) readReceiptsRLP(n uint64) ([]byte, error) {
	e, err := r.readEntry(n, 2)
	if err != nil {
		return nil, err
	}
	if e.Type != TypeCompressedReceipts {
		return nil, fmt.Errorf("expected receipts entry, got %x", e.Type)
	}
	return io.ReadAll(snappy.NewReader(bytes.NewReader(e.Value)))
}

// readTotalDifficulty reads the total difficulty of block number n.
func (r *Reader) readTotalDifficulty(n uint64) (*big.Int, error) {
	e, err := r.readEntry(n, 3)
	if err != nil {
		return nil, err
	}
	if e.Type != TypeTotalDifficulty {
		return nil, fmt.Errorf("expected TD entry, got %x", e.Type)
	}
	return new(big.Int).SetBytes(reverseOrder(e.Value)), nil
}

// ReadHeader reads the header number n.
func (r *Reader) ReadHeader(n uint64) (*types.Header, error) {
	h, err := r.readHeaderRLP(n)
	if err != nil {
		return nil, err
	}
	var header types.Header
	if err := rlp.DecodeBytes(h, &header); err != nil {
		return nil, err
	}
	return &header, nil
}

// ReadBlock reads the block number n.
func (r *Reader) ReadBlock(n uint64) (*types.Block, error) {
	header, err := r.ReadHeader(n)
	if err != nil {
		return nil, err
	}
	b, err := r.readBodyRLP(n)
	if err != nil {
		return nil, err
	}
	var body types.Body
	if err := rlp.DecodeBytes(b, &body); err != nil {
		return nil, err
	}
	return types.NewBlockWithHeader(header).WithBody(body.Transactions, body.Uncles), nil
}

// ReadBlockAndReceipts reads the block number n and associated receipts.
func (r *Reader) ReadBlockAndReceipts(n uint64) (*types.Block, types.Receipts, error) {
	// Read block.
	block, err := r.ReadBlock(n)
	if err != nil {
		return nil, nil, err
	}
	// Read receipts.
	rr, err := r.readReceiptsRLP(n)
	if err != nil {
		return nil, nil, err
	}
	// Decode receipts.
	var receipts types.Receipts
	if err := rlp.DecodeBytes(rr, &receipts); err != nil {
		return nil, nil, err
	}
	return block, receipts, err
}

// Accumulator reads the accumulator entry in the Era1 file.
func (r *Reader) Accumulator() (common.Hash, error) {
	entry, err := r.e.Find(TypeAccumulator)
	if err != nil {
		return common.Hash{}, err
	}
	return common.BytesToHash(entry.Value), nil
}

// InitialTD returns initial total difficulty before the difficulty of the
// first block of the Era1 is applied.
func (r *Reader) InitialTD() (*big.Int, error) {
	h, err := r.ReadHeader(r.Start())
	if err != nil {
		return nil, err
	}
	// Above seek also sets reader so next TD entry will be for this block.
	entry, err := r.e.Find(TypeTotalDifficulty)
	if err != nil {
		return nil, err
	}
	td := new(big.Int).SetBytes(reverseOrder(entry.Value))
	return td.Sub(td, h.Difficulty), nil
}

// Start returns the listed start block.
func (r *Reader) Start() uint64 {
	return r.metadata.start
}

// Count returns the total number of blocks in the Era1.
func (r *Reader) Count() uint64 {
	return r.metadata.count
}

// clearBuffer zeroes out the buffer.
func (r *Reader) clearBuffer() {
	for i := 0; i < len(r.buf); i++ {
		r.buf[i] = 0
	}
}

// metadata wraps the metadata in the block index.
type metadata struct {
	start, count uint64
}

// readMetadata reads the metadata stored in an Era1 file's block index.
func readMetadata(r io.ReaderAt, length int64) (m metadata, err error) {
	b := make([]byte, 16)
	// Read count. It's the last 8 bytes of the file.
	if _, err = r.ReadAt(b[:8], length-8); err != nil {
		return
	}
	m.count = binary.LittleEndian.Uint64(b)
	// Read start. It's at the offset -sizeof(m.count) -
	// count*sizeof(indexEntry) - sizeof(m.start)
	if _, err = r.ReadAt(b[8:], length-16-int64(m.count*8)); err != nil {
		return
	}
	m.start = binary.LittleEndian.Uint64(b[8:])
	return
}
