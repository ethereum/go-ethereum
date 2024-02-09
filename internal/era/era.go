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
	"encoding/binary"
	"fmt"
	"io"
	"math/big"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"

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

type ReadAtSeekCloser interface {
	io.ReaderAt
	io.Seeker
	io.Closer
}

// Era reads and Era1 file.
type Era struct {
	f   ReadAtSeekCloser // backing era1 file
	s   *e2store.Reader  // e2store reader over f
	m   metadata         // start, count, length info
	mu  *sync.Mutex      // lock for buf
	buf [8]byte          // buffer reading entry offsets
}

// From returns an Era backed by f.
func From(f ReadAtSeekCloser) (*Era, error) {
	m, err := readMetadata(f)
	if err != nil {
		return nil, err
	}
	return &Era{
		f:  f,
		s:  e2store.NewReader(f),
		m:  m,
		mu: new(sync.Mutex),
	}, nil
}

// Open returns an Era backed by the given filename.
func Open(filename string) (*Era, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	return From(f)
}

func (e *Era) Close() error {
	return e.f.Close()
}

func (e *Era) GetBlockByNumber(num uint64) (*types.Block, error) {
	if e.m.start > num || e.m.start+e.m.count <= num {
		return nil, fmt.Errorf("out-of-bounds")
	}
	off, err := e.readOffset(num)
	if err != nil {
		return nil, err
	}
	r, n, err := newSnappyReader(e.s, TypeCompressedHeader, off)
	if err != nil {
		return nil, err
	}
	var header types.Header
	if err := rlp.Decode(r, &header); err != nil {
		return nil, err
	}
	off += n
	r, _, err = newSnappyReader(e.s, TypeCompressedBody, off)
	if err != nil {
		return nil, err
	}
	var body types.Body
	if err := rlp.Decode(r, &body); err != nil {
		return nil, err
	}
	return types.NewBlockWithHeader(&header).WithBody(body.Transactions, body.Uncles), nil
}

// Accumulator reads the accumulator entry in the Era1 file.
func (e *Era) Accumulator() (common.Hash, error) {
	entry, err := e.s.Find(TypeAccumulator)
	if err != nil {
		return common.Hash{}, err
	}
	return common.BytesToHash(entry.Value), nil
}

// InitialTD returns initial total difficulty before the difficulty of the
// first block of the Era1 is applied.
func (e *Era) InitialTD() (*big.Int, error) {
	var (
		r      io.Reader
		header types.Header
		rawTd  []byte
		n      int64
		off    int64
		err    error
	)

	// Read first header.
	if off, err = e.readOffset(e.m.start); err != nil {
		return nil, err
	}
	if r, n, err = newSnappyReader(e.s, TypeCompressedHeader, off); err != nil {
		return nil, err
	}
	if err := rlp.Decode(r, &header); err != nil {
		return nil, err
	}
	off += n

	// Skip over next two records.
	for i := 0; i < 2; i++ {
		length, err := e.s.LengthAt(off)
		if err != nil {
			return nil, err
		}
		off += length
	}

	// Read total difficulty after first block.
	if r, _, err = e.s.ReaderAt(TypeTotalDifficulty, off); err != nil {
		return nil, err
	}
	rawTd, err = io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	td := new(big.Int).SetBytes(reverseOrder(rawTd))
	return td.Sub(td, header.Difficulty), nil
}

// Start returns the listed start block.
func (e *Era) Start() uint64 {
	return e.m.start
}

// Count returns the total number of blocks in the Era1.
func (e *Era) Count() uint64 {
	return e.m.count
}

// readOffset reads a specific block's offset from the block index. The value n
// is the absolute block number desired.
func (e *Era) readOffset(n uint64) (int64, error) {
	var (
		blockIndexRecordOffset = e.m.length - 24 - int64(e.m.count)*8 // skips start, count, and header
		firstIndex             = blockIndexRecordOffset + 16          // first index after header / start-num
		indexOffset            = int64(n-e.m.start) * 8               // desired index * size of indexes
		offOffset              = firstIndex + indexOffset             // offset of block offset
	)
	e.mu.Lock()
	defer e.mu.Unlock()
	clearBuffer(e.buf[:])
	if _, err := e.f.ReadAt(e.buf[:], offOffset); err != nil {
		return 0, err
	}
	// Since the block offset is relative from the start of the block index record
	// we need to add the record offset to it's offset to get the block's absolute
	// offset.
	return blockIndexRecordOffset + int64(binary.LittleEndian.Uint64(e.buf[:])), nil
}

// newReader returns a snappy.Reader for the e2store entry value at off.
func newSnappyReader(e *e2store.Reader, expectedType uint16, off int64) (io.Reader, int64, error) {
	r, n, err := e.ReaderAt(expectedType, off)
	if err != nil {
		return nil, 0, err
	}
	return snappy.NewReader(r), int64(n), err
}

// clearBuffer zeroes out the buffer.
func clearBuffer(buf []byte) {
	for i := 0; i < len(buf); i++ {
		buf[i] = 0
	}
}

// metadata wraps the metadata in the block index.
type metadata struct {
	start  uint64
	count  uint64
	length int64
}

// readMetadata reads the metadata stored in an Era1 file's block index.
func readMetadata(f ReadAtSeekCloser) (m metadata, err error) {
	// Determine length of reader.
	if m.length, err = f.Seek(0, io.SeekEnd); err != nil {
		return
	}
	b := make([]byte, 16)
	// Read count. It's the last 8 bytes of the file.
	if _, err = f.ReadAt(b[:8], m.length-8); err != nil {
		return
	}
	m.count = binary.LittleEndian.Uint64(b)
	// Read start. It's at the offset -sizeof(m.count) -
	// count*sizeof(indexEntry) - sizeof(m.start)
	if _, err = f.ReadAt(b[8:], m.length-16-int64(m.count*8)); err != nil {
		return
	}
	m.start = binary.LittleEndian.Uint64(b[8:])
	return
}
