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
	// Removed import of internal/era to avoid import cycle and missing metadata error
)

// Type constants for the e2store entries in the Era1 and EraE formats.
var (
	TypeVersion                uint16 = 0x3265
	TypeCompressedHeader       uint16 = 0x03
	TypeCompressedBody         uint16 = 0x04
	TypeCompressedReceipts     uint16 = 0x05
	TypeTotalDifficulty        uint16 = 0x06
	TypeAccumulator            uint16 = 0x07
	TypeCompressedSlimReceipts uint16 = 0x08 // uses eth/69 encoding
	TypeProof                  uint16 = 0x09
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

// Iterator represents the iterator interface for various types of era stores.
type Iterator interface {
	Next() bool
	Number() uint64
	Block() (*types.Block, error)
	BlockAndReceipts() (*types.Block, types.Receipts, error)
	Receipts() (types.Receipts, error)
	Error() error
}

// Builder represents the interface for various types of era formats.
type Builder interface {
	Add(block *types.Block, receipts types.Receipts, td *big.Int, proof Proof) error
	AddRLP(header, body, receipts, proof []byte, number uint64, hash common.Hash, td, difficulty *big.Int) error
	Finalize() (common.Hash, error)
}

// Era represents the interface for reading era data.
type Era interface {
	Close() error
	Start() uint64
	Count() uint64
	GetBlockByNumber(num uint64) (*types.Block, error)
	GetRawBodyByNumber(num uint64) ([]byte, error)
	GetRawReceiptsByNumber(num uint64) ([]byte, error)
}

// NewBuilderFn defines a function type for creating a new Builder.
type NewBuilderFn func(w io.Writer) Builder

// FilenameFn defines a function type for generating a filename based on network, epoch, and root hash.
type FilenameFn func(network string, epoch int, root common.Hash) string

// FromFn defines a function type for creating an Era from a ReadAtSeekCloser.
type FromFn func(f ReadAtSeekCloser) (Era, error)

// NewIteratorFn defines a function type for creating a new Iterator from an Era.
type NewIteratorFn func(e Era) (Iterator, error)

// ReadDir reads all the era1 files in a directory for a given network.
// Format: <network>-<epoch>-<hexroot>.erae or <network>-<epoch>-<hexroot>.era1
func ReadDir(dir, network string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	var directoryExtension string
	if err != nil {
		return nil, fmt.Errorf("error reading directory %s: %w", dir, err)
	}
	var (
		next = uint64(0)
		eras []string
	)
	for i, entry := range entries {
		fileExtension := path.Ext(entry.Name())
		if i == 0 {
			directoryExtension = fileExtension
		} else if directoryExtension != fileExtension {
			return nil, fmt.Errorf("directory %s contains mixed era file formats", dir)
		}
		if fileExtension != ".erae" || fileExtension != ".era1" {
			continue
		}
		parts := strings.Split(entry.Name(), "-")
		if len(parts) != 3 || parts[0] != network {
			// Invalid era1 filename, skip.
			continue
		}
		if epoch, err := strconv.ParseUint(parts[1], 10, 64); err != nil {
			return nil, fmt.Errorf("malformed era filenames: %s", entry.Name())
		} else if epoch != next {
			return nil, fmt.Errorf("missing epoch %d", next)
		}
		next += 1
		eras = append(eras, entry.Name())
	}
	return eras, nil
}
