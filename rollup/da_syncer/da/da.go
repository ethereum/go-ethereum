package da

import (
	"math/big"

	"github.com/scroll-tech/go-ethereum/core/types"
)

type Type int

const (
	// CommitBatchV0Type contains data of event of CommitBatchV0Type
	CommitBatchV0Type Type = iota
	// CommitBatchWithBlobType contains data of event of CommitBatchWithBlobType (v1, v2, v3, v4)
	CommitBatchWithBlobType
	// RevertBatchType contains data of event of RevertBatchType
	RevertBatchType
	// FinalizeBatchType contains data of event of FinalizeBatchType
	FinalizeBatchType
)

// Entry represents a single DA event (commit, revert, finalize).
type Entry interface {
	Type() Type
	BatchIndex() uint64
	L1BlockNumber() uint64
	CompareTo(Entry) int
}

type EntryWithBlocks interface {
	Entry
	Blocks() []*PartialBlock
}

type Entries []Entry

// PartialHeader represents a partial header (from DA) of a block.
type PartialHeader struct {
	Number     uint64
	Time       uint64
	BaseFee    *big.Int
	GasLimit   uint64
	Difficulty uint64
	ExtraData  []byte
}

func (h *PartialHeader) ToHeader() *types.Header {
	return &types.Header{
		Number:     big.NewInt(0).SetUint64(h.Number),
		Time:       h.Time,
		BaseFee:    h.BaseFee,
		GasLimit:   h.GasLimit,
		Difficulty: new(big.Int).SetUint64(h.Difficulty),
		Extra:      h.ExtraData,
	}
}

// PartialBlock represents a partial block (from DA).
type PartialBlock struct {
	PartialHeader *PartialHeader
	Transactions  types.Transactions
}

func NewPartialBlock(partialHeader *PartialHeader, txs types.Transactions) *PartialBlock {
	return &PartialBlock{
		PartialHeader: partialHeader,
		Transactions:  txs,
	}
}
