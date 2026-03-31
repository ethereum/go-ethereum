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

package era

import (
	"fmt"
	"io"
	"math/big"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Type constants for the e2store entries in the Era1 and EraE formats.
var (
	TypeVersion                uint16 = 0x3265
	TypeCompressedHeader       uint16 = 0x03
	TypeCompressedBody         uint16 = 0x04
	TypeCompressedReceipts     uint16 = 0x05
	TypeTotalDifficulty        uint16 = 0x06
	TypeAccumulator            uint16 = 0x07
	TypeCompressedSlimReceipts uint16 = 0x0a // uses eth/69 encoding
	TypeProof                  uint16 = 0x0b
	TypeBlockIndex             uint16 = 0x3266
	TypeComponentIndex         uint16 = 0x3267

	MaxSize = 8192
	// headerSize uint64 = 8
)

type ReadAtSeekCloser interface {
	io.ReaderAt
	io.Seeker
	io.Closer
}

// Iterator provides sequential access to blocks in an era file.
type Iterator interface {
	// Next advances to the next block. Returns true if a block is available,
	// false when iteration is complete or an error occurred.
	Next() bool

	// Number returns the block number of the current block.
	Number() uint64

	// Block returns the current block.
	Block() (*types.Block, error)

	// BlockAndReceipts returns the current block and its receipts.
	BlockAndReceipts() (*types.Block, types.Receipts, error)

	// Receipts returns the receipts for the current block.
	Receipts() (types.Receipts, error)

	// Error returns any error encountered during iteration.
	Error() error
}

// Builder constructs era files from blocks and receipts.
//
// Builders handle three epoch types automatically:
//   - Pre-merge: all blocks have difficulty > 0, TD is stored for each block
//   - Transition: starts pre-merge, ends post-merge; TD stored for all blocks
//   - Post-merge: all blocks have difficulty == 0, no TD stored
type Builder interface {
	// Add appends a block and its receipts to the era file.
	// For pre-merge blocks, td must be provided.
	// For post-merge blocks, td should be nil.
	Add(block *types.Block, receipts types.Receipts, td *big.Int) error

	// AddRLP appends RLP-encoded block components to the era file.
	// For pre-merge blocks, td and difficulty must be provided.
	// For post-merge blocks, td and difficulty should be nil.
	AddRLP(header, body, receipts []byte, number uint64, hash common.Hash, td, difficulty *big.Int) error

	// Finalize writes all collected entries and returns the epoch identifier.
	// For Era1 (onedb): returns the accumulator root.
	// For EraE (execdb): returns the last block hash.
	Finalize() (common.Hash, error)

	// Accumulator returns the accumulator root after Finalize has been called.
	// Returns nil for post-merge epochs where no accumulator exists.
	Accumulator() *common.Hash
}

// Era represents the interface for reading era data.
type Era interface {
	Close() error
	Start() uint64
	Count() uint64
	Iterator() (Iterator, error)
	GetBlockByNumber(num uint64) (*types.Block, error)
	GetRawBodyByNumber(num uint64) ([]byte, error)
	GetRawReceiptsByNumber(num uint64) ([]byte, error)
	InitialTD() (*big.Int, error)
	Accumulator() (common.Hash, error)
}

// ReadDir reads all the era files in a directory for a given network.
// Format: <network>-<epoch>-<hexroot>.erae or <network>-<epoch>-<hexroot>.era1
func ReadDir(dir, network string) ([]string, error) {
	entries, err := os.ReadDir(dir)

	if err != nil {
		return nil, fmt.Errorf("error reading directory %s: %w", dir, err)
	}
	var (
		next    = uint64(0)
		eras    []string
		dirType string
	)
	for _, entry := range entries {
		ext := path.Ext(entry.Name())
		if ext != ".erae" && ext != ".era1" {
			continue
		}
		if dirType == "" {
			dirType = ext
		}
		parts := strings.Split(entry.Name(), "-")
		if len(parts) != 3 || parts[0] != network {
			// Invalid era filename, skip.
			continue
		}
		if epoch, err := strconv.ParseUint(parts[1], 10, 64); err != nil {
			return nil, fmt.Errorf("malformed era filenames: %s", entry.Name())
		} else if epoch != next {
			return nil, fmt.Errorf("missing epoch %d", next)
		}
		if dirType != ext {
			return nil, fmt.Errorf("directory %s contains mixed era file formats: want %s, have %s", dir, dirType, ext)
		}
		next += 1
		eras = append(eras, entry.Name())
	}
	return eras, nil
}
