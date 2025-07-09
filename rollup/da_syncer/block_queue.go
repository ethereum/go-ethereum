package da_syncer

import (
	"context"
	"fmt"

	"github.com/scroll-tech/go-ethereum/core/rawdb"
	"github.com/scroll-tech/go-ethereum/rollup/da_syncer/da"
	"github.com/scroll-tech/go-ethereum/rollup/missing_header_fields"
)

// BlockQueue is a pipeline stage that reads batches from BatchQueue, extracts all da.PartialBlock from it and
// provides them to the next stage one-by-one.
type BlockQueue struct {
	batchQueue                 *BatchQueue
	blocks                     []*da.PartialBlock
	missingHeaderFieldsManager *missing_header_fields.Manager
}

func NewBlockQueue(batchQueue *BatchQueue, missingHeaderFieldsManager *missing_header_fields.Manager) *BlockQueue {
	return &BlockQueue{
		batchQueue:                 batchQueue,
		blocks:                     make([]*da.PartialBlock, 0),
		missingHeaderFieldsManager: missingHeaderFieldsManager,
	}
}

func (bq *BlockQueue) NextBlock(ctx context.Context) (*da.PartialBlock, error) {
	for len(bq.blocks) == 0 {
		err := bq.getBlocksFromBatch(ctx)
		if err != nil {
			return nil, err
		}
	}
	block := bq.blocks[0]
	bq.blocks = bq.blocks[1:]
	return block, nil
}

func (bq *BlockQueue) getBlocksFromBatch(ctx context.Context) error {
	entryWithBlocks, err := bq.batchQueue.NextBatch(ctx)
	if err != nil {
		return err
	}

	bq.blocks, err = entryWithBlocks.Blocks(bq.missingHeaderFieldsManager)
	if err != nil {
		return fmt.Errorf("failed to get blocks from entry: %w", err)
	}

	return nil
}

func (bq *BlockQueue) Reset(lastProcessedBatchMeta *rawdb.DAProcessedBatchMeta) {
	bq.blocks = make([]*da.PartialBlock, 0)
	bq.batchQueue.Reset(lastProcessedBatchMeta)
}
