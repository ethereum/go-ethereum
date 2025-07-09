package types

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/scroll-tech/da-codec/encoding"
	"gorm.io/gorm"

	"github.com/scroll-tech/go-ethereum/core/types"
)

// L2Block represents a l2 block in the database.
type L2Block struct {
	db *gorm.DB `gorm:"column:-"`

	// block
	Number         uint64 `json:"number" gorm:"number"`
	Hash           string `json:"hash" gorm:"hash"`
	ParentHash     string `json:"parent_hash" gorm:"parent_hash"`
	Header         string `json:"header" gorm:"header"`
	Transactions   string `json:"transactions" gorm:"transactions"`
	WithdrawRoot   string `json:"withdraw_root" gorm:"withdraw_root"`
	StateRoot      string `json:"state_root" gorm:"state_root"`
	TxNum          uint32 `json:"tx_num" gorm:"tx_num"`
	GasUsed        uint64 `json:"gas_used" gorm:"gas_used"`
	BlockTimestamp uint64 `json:"block_timestamp" gorm:"block_timestamp"`
	RowConsumption string `json:"row_consumption" gorm:"row_consumption"`

	// chunk
	ChunkHash string `json:"chunk_hash" gorm:"chunk_hash;default:NULL"`

	// metadata
	CreatedAt time.Time      `json:"created_at" gorm:"column:created_at"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"column:deleted_at;default:NULL"`
}

// NewL2Block creates a new L2Block instance
func NewL2Block(db *gorm.DB) *L2Block {
	return &L2Block{db: db}
}

// TableName returns the name of the "l2_block" table.
func (*L2Block) TableName() string {
	return "l2_block"
}

// GetL2BlocksInRange retrieves the L2 blocks within the specified range (inclusive).
// The range is closed, i.e., it includes both start and end block numbers.
// The returned blocks are sorted in ascending order by their block number.
func (o *L2Block) GetL2BlocksInRange(ctx context.Context, startBlockNumber uint64, endBlockNumber uint64) ([]*encoding.Block, error) {
	if startBlockNumber > endBlockNumber {
		return nil, fmt.Errorf("L2Block.GetL2BlocksInRange: start block number should be less than or equal to end block number, start block: %v, end block: %v", startBlockNumber, endBlockNumber)
	}

	db := o.db.WithContext(ctx)
	db = db.Model(&L2Block{})
	db = db.Select("header")
	db = db.Where("number >= ? AND number <= ?", startBlockNumber, endBlockNumber)
	db = db.Order("number ASC")

	var l2Blocks []L2Block
	if err := db.Find(&l2Blocks).Error; err != nil {
		return nil, fmt.Errorf("L2Block.GetL2BlocksInRange error: %w, start block: %v, end block: %v", err, startBlockNumber, endBlockNumber)
	}

	// sanity check
	if uint64(len(l2Blocks)) != endBlockNumber-startBlockNumber+1 {
		return nil, fmt.Errorf("L2Block.GetL2BlocksInRange: unexpected number of results, expected: %v, got: %v", endBlockNumber-startBlockNumber+1, len(l2Blocks))
	}

	var blocks []*encoding.Block
	for _, v := range l2Blocks {
		var block encoding.Block

		block.Header = &types.Header{}
		if err := json.Unmarshal([]byte(v.Header), block.Header); err != nil {
			return nil, fmt.Errorf("L2Block.GetL2BlocksInRange error: %w, start block: %v, end block: %v", err, startBlockNumber, endBlockNumber)
		}

		blocks = append(blocks, &block)
	}

	return blocks, nil
}
