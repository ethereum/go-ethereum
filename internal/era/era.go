package era

import (
	"io"
	"math/big"

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
	Receipts() (types.Receipts, error)
	Error() error
}

// Builder represents the interface for various types of era formats.
type Builder interface {
	Add(block *types.Block, receipts types.Receipts, td *big.Int, proof []byte) error
	Finalize() (common.Hash, error)
}
