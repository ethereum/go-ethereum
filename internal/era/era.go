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
		if path.Ext(entry.Name()) == ".era1" {
			n := strings.Split(entry.Name(), "-")
			if len(n) != 3 {
				// invalid era1 filename, skip
				continue
			}
			if n[0] == network {
				if epoch, err := strconv.ParseUint(n[1], 10, 64); err != nil {
					return nil, fmt.Errorf("malformed era1 filename: %s", entry.Name())
				} else if epoch != next {
					return nil, fmt.Errorf("missing epoch %d", next)
				}
				next += 1
				eras = append(eras, entry.Name())
			}
		}
	}
	return eras, nil
}

// Builder is used to create Era1 archives of block data.
//
// Era1 files are themselves e2store files. For more information on this format,
// see https://github.com/status-im/nimbus-eth2/blob/stable/docs/e2store.md.
//
// The overall structure of an Era1 file follows closely the structure of an Era file
// which contains consensus Layer data (and as a byproduct, EL data after the merge).
//
// The structure can be summarized through this definition:
//
//	era1 := Version | block-tuple* | other-entries* | Accumulator | BlockIndex
//	block-tuple :=  CompressedHeader | CompressedBody | CompressedReceipts | TotalDifficulty
//
// Each basic element is its own entry:
//
//	Version            = { type: [0x65, 0x32], data: nil }
//	CompressedHeader   = { type: [0x03, 0x00], data: snappyFramed(rlp(header)) }
//	CompressedBody     = { type: [0x04, 0x00], data: snappyFramed(rlp(body)) }
//	CompressedReceipts = { type: [0x05, 0x00], data: snappyFramed(rlp(receipts)) }
//	TotalDifficulty    = { type: [0x06, 0x00], data: uint256(header.total_difficulty) }
//	Accumulator        = { type: [0x07, 0x00], data: accumulator-root }
//	BlockIndex         = { type: [0x32, 0x66], data: block-index }
//
// Accumulator is computed by constructing an SSZ list of header-records of length at most
// 8192 and then calculating the hash_tree_root of that list.
//
//	header-record := { block-hash: Bytes32, total-difficulty: Uint256 }
//	accumulator   := hash_tree_root([]header-record, 8192)
//
// BlockIndex stores relative offsets to each compressed block entry. The
// format is:
//
//	block-index := starting-number | index | index | index ... | count
//
// starting-number is the first block number in the archive. Every index is a
// defined relative to index's location in the file. The total number of block
// entries in the file is recorded in count.
//
// Due to the accumulator size limit of 8192, the maximum number of blocks in
// an Era1 batch is also 8192.
type Builder struct {
	w        *e2store.Writer
	startNum *uint64
	startTd  *big.Int
	indexes  []uint64
	hashes   []common.Hash
	tds      []*big.Int
	written  int
}

// NewBuilder returns a new Builder instance.
func NewBuilder(w io.Writer) *Builder {
	return &Builder{
		w:      e2store.NewWriter(w),
		hashes: make([]common.Hash, 0),
		tds:    make([]*big.Int, 0),
	}
}

// Add writes a compressed block entry and compressed receipts entry to the
// underlying e2store file.
func (b *Builder) Add(block *types.Block, receipts types.Receipts, td *big.Int) error {
	eh, err := rlp.EncodeToBytes(block.Header())
	if err != nil {
		return err
	}
	eb, err := rlp.EncodeToBytes(block.Body())
	if err != nil {
		return err
	}
	er, err := rlp.EncodeToBytes(receipts)
	if err != nil {
		return err
	}
	return b.AddRLP(eh, eb, er, block.NumberU64(), block.Hash(), td, block.Difficulty())
}

// AddRLP writes a compressed block entry and compressed receipts entry to the
// underlying e2store file.
func (b *Builder) AddRLP(header, body, receipts []byte, number uint64, hash common.Hash, td, difficulty *big.Int) error {
	// Write Era1 version entry before first block.
	if b.startNum == nil {
		if err := writeVersion(b.w); err != nil {
			return err
		}
		n := number
		b.startNum = &n
		b.startTd = new(big.Int).Sub(td, difficulty)
	}
	if len(b.indexes) >= MaxEra1Size {
		return fmt.Errorf("exceeds maximum batch size of %d", MaxEra1Size)
	}

	b.indexes = append(b.indexes, uint64(b.written))
	b.hashes = append(b.hashes, hash)
	b.tds = append(b.tds, td)

	// Small helper to take care snappy encoding and writing e2store entry.
	snappyWrite := func(typ uint16, in []byte) error {
		var (
			buf = bytes.NewBuffer(nil)
			s   = snappy.NewBufferedWriter(buf)
		)
		if _, err := s.Write(in); err != nil {
			return fmt.Errorf("error snappy encoding: %w", err)
		}
		if err := s.Flush(); err != nil {
			return fmt.Errorf("error flushing snappy encoding: %w", err)
		}
		n, err := b.w.Write(typ, buf.Bytes())
		b.written += n
		if err != nil {
			return fmt.Errorf("error writing e2store entry: %w", err)
		}
		return nil
	}

	// Write block data.
	if err := snappyWrite(TypeCompressedHeader, header); err != nil {
		return err
	}
	if err := snappyWrite(TypeCompressedBody, body); err != nil {
		return err
	}
	if err := snappyWrite(TypeCompressedReceipts, receipts); err != nil {
		return err
	}
	// Also write total difficulty, but don't snappy encode.
	btd := bigToBytes32(td)
	n, err := b.w.Write(TypeTotalDifficulty, btd[:])
	b.written += n
	if err != nil {
		return err
	}

	return nil
}

// Finalize computes the accumulator and block index values, then writes the
// corresponding e2store entries.
func (b *Builder) Finalize() (common.Hash, error) {
	if b.startNum == nil {
		return common.Hash{}, fmt.Errorf("finalize called on empty builder")
	}
	// Compute accumulator root and write entry.
	root, err := ComputeAccumulator(b.hashes, b.tds)
	if err != nil {
		return common.Hash{}, fmt.Errorf("error calculating accumulator root: %w", err)
	}
	n, err := b.w.Write(TypeAccumulator, root[:])
	b.written += n
	if err != nil {
		return common.Hash{}, fmt.Errorf("error writing accumulator: %w", err)
	}
	// Get beginning of index entry to calculate block relative offset.
	base := int64(b.written + (3 * 8)) // skip e2store header (type, length) and start block

	// Construct block index. Detailed format described in Builder
	// documentation, but it is essentially encoded as:
	// "start | index | index | ... | count"
	var (
		count = len(b.indexes)
		index = make([]byte, 16+count*8)
	)
	binary.LittleEndian.PutUint64(index, *b.startNum)
	// Each offset is relative from the position it is encoded in the
	// index. This means that even if the same block was to be included in
	// the index twice (this would be invalid anyways), the relative offset
	// would be different. The idea with this is that after reading a
	// relative offset, the corresponding block can be quickly read by
	// performing a seek relative to the current position.
	for i, offset := range b.indexes {
		relative := int64(offset) - (base + int64(i)*8)
		binary.LittleEndian.PutUint64(index[8+i*8:], uint64(relative))
	}
	binary.LittleEndian.PutUint64(index[8+count*8:], uint64(count))

	// Finally, write the block index entry.
	if _, err := b.w.Write(TypeBlockIndex, index); err != nil {
		return common.Hash{}, fmt.Errorf("unable to write block index: %w", err)
	}

	return root, nil
}

// writeVersion writes a version entry to e2store.
func writeVersion(w *e2store.Writer) error {
	_, err := w.Write(TypeVersion, nil)
	return err
}

// Reader reads an Era1 archive.
// See Builder documentation for a detailed explanation of the Era1 format.
type Reader struct {
	r        io.ReadSeeker
	offset   uint64
	metadata metadata
}

// NewReader returns a new Reader instance.
func NewReader(r io.ReadSeeker) (*Reader, error) {
	m, err := readMetadata(r)
	if err != nil {
		return nil, err
	}
	return &Reader{r, m.start, m}, nil
}

// readOffset reads a specific block's offset from the block index. The value n
// is the absolute block number desired. It is normalized against the index's
// start block.
func (r *Reader) readOffset(n uint64) (int64, error) {
	// Seek to the encoding of the block's offset.
	var (
		firstIndex  = -8 - int64(r.metadata.count)*8 // size of count - index entries
		indexOffset = int64(n-r.metadata.start) * 8  // desired index * size of indexes
	)
	if _, err := r.r.Seek(firstIndex+indexOffset, io.SeekEnd); err != nil {
		return 0, err
	}
	// Read the block's offset.
	var offset int64
	if err := binary.Read(r.r, binary.LittleEndian, &offset); err != nil {
		return 0, err
	}
	return offset, nil
}

// Read reads one (block, receipts) tuple from an Era1 archive.
func (r *Reader) Read() (*types.Block, types.Receipts, error) {
	block, receipts, err := r.ReadBlockAndReceipts(r.offset)
	if err != nil {
		return nil, nil, err
	}
	r.offset += 1
	return block, receipts, nil
}

// ReadHeader reads the header number n RLP.
func (r *Reader) ReadHeaderRLP(n uint64) ([]byte, error) {
	// Determine if the request can served by current the Era1 file, e.g. n
	// must be within the range of blocks specified in the block index
	// metadata.
	if n < r.metadata.start || r.metadata.start+r.metadata.count < n {
		return nil, fmt.Errorf("request out-of-bounds: want %d, start: %d, count: %d", n, r.metadata.start, r.metadata.count)
	}
	// Read the specified block's offset from the block index.
	offset, err := r.readOffset(n)
	if err != nil {
		return nil, fmt.Errorf("error reading block offset: %w", err)
	}
	if _, err := r.r.Seek(offset, io.SeekCurrent); err != nil {
		return nil, err
	}
	// Read header.
	entry, err := e2store.NewReader(r.r).Read()
	if err != nil {
		return nil, err
	}
	if entry.Type != TypeCompressedHeader {
		return nil, fmt.Errorf("expected header entry, got %x", entry.Type)
	}
	return io.ReadAll(snappy.NewReader(bytes.NewReader(entry.Value)))
}

// ReadBodyRLP reads the block body number n RLP.
func (r *Reader) ReadBodyRLP(n uint64) ([]byte, error) {
	// Orient cursor.
	_, err := r.ReadHeaderRLP(n)
	if err != nil {
		return nil, err
	}
	// Read body.
	entry, err := e2store.NewReader(r.r).Read()
	if err != nil {
		return nil, err
	}
	if entry.Type != TypeCompressedBody {
		return nil, fmt.Errorf("expected body entry, got %x", entry.Type)
	}
	return io.ReadAll(snappy.NewReader(bytes.NewReader(entry.Value)))
}

// ReadReceiptsRLP reads the receipts RLP associated with number n.
func (r *Reader) ReadReceiptsRLP(n uint64) ([]byte, error) {
	// Orient cursor.
	_, err := r.ReadBodyRLP(n)
	if err != nil {
		return nil, err
	}
	// Read receipts.
	entry, err := e2store.NewReader(r.r).Read()
	if err != nil {
		return nil, err
	}
	if entry.Type != TypeCompressedReceipts {
		return nil, fmt.Errorf("expected receipts entry, got %x", entry.Type)
	}
	return io.ReadAll(snappy.NewReader(bytes.NewReader(entry.Value)))
}

// ReadTotalDifficulty reads the total difficulty of block number n.
func (r *Reader) ReadTotalDifficulty(n uint64) (*big.Int, error) {
	// Orient cursor.
	_, err := r.ReadReceiptsRLP(n)
	if err != nil {
		return nil, err
	}
	// Read totaly difficulty.
	entry, err := e2store.NewReader(r.r).Read()
	if err != nil {
		return nil, err
	}
	if entry.Type != TypeTotalDifficulty {
		return nil, fmt.Errorf("expected total difficulty entry, got %x", entry.Type)
	}
	return new(big.Int).SetBytes(reverseOrder(entry.Value)), nil
}

// ReadHeader reads the header number n.
func (r *Reader) ReadHeader(n uint64) (*types.Header, error) {
	h, err := r.ReadHeaderRLP(n)
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
	b, err := r.ReadBodyRLP(n)
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
	rr, err := r.ReadReceiptsRLP(n)
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
	_, err := r.seek(0, io.SeekStart)
	if err != nil {
		return common.Hash{}, err
	}
	entry, err := e2store.NewReader(r.r).Find(TypeAccumulator)
	if err != nil {
		return common.Hash{}, err
	}
	return common.BytesToHash(entry.Value), nil
}

// InitialTD returns initial total difficulty before the difficulty of the
// first block of the Era1 is applied.
func (r *Reader) InitialTD() (*big.Int, error) {
	_, err := r.seek(0, io.SeekStart)
	if err != nil {
		return nil, err
	}
	h, err := r.ReadHeader(r.Start())
	if err != nil {
		return nil, err
	}
	// Above seek also sets reader so next TD entry will be for this block.
	entry, err := e2store.NewReader(r.r).Find(TypeTotalDifficulty)
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

// seek is a shorthand method for calling seek on the inner reader.
func (r *Reader) seek(offset int64, whence int) (int64, error) {
	return r.r.Seek(offset, whence)
}

// metadata wraps the metadata in the block index.
type metadata struct {
	start, count uint64
}

// readMetadata reads the metadata stored in an Era1 file's block index.
func readMetadata(r io.ReadSeeker) (m metadata, err error) {
	// Seek to count value. It's the last 8 bytes of the file.
	if _, err = r.Seek(-8, io.SeekEnd); err != nil {
		return
	}
	// Read count.
	if err = binary.Read(r, binary.LittleEndian, &m.count); err != nil {
		return
	}
	// Seek to start value. It's at the offset -sizeof(m.count) -
	// count*sizeof(indexEntry) - sizeof(m.start)
	if _, err = r.Seek(-16-int64(m.count)*8, io.SeekEnd); err != nil {
		return
	}
	// Read start.
	if err = binary.Read(r, binary.LittleEndian, &m.start); err != nil {
		return
	}
	return
}
