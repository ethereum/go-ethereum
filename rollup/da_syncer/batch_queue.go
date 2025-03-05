package da_syncer

import (
	"context"
	"fmt"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/rawdb"
	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rollup/da_syncer/da"
	"github.com/scroll-tech/go-ethereum/rollup/l1"
)

// BatchQueue is a pipeline stage that reads all batch events from DAQueue and provides only finalized batches to the next stage.
type BatchQueue struct {
	DAQueue                 *DAQueue
	db                      ethdb.Database
	lastFinalizedBatchIndex uint64
	batches                 *common.Heap[da.Entry]
	batchesMap              *common.ShrinkingMap[uint64, *common.HeapElement[da.Entry]]

	previousBatch *rawdb.DAProcessedBatchMeta
}

func NewBatchQueue(DAQueue *DAQueue, db ethdb.Database, lastProcessedBatch *rawdb.DAProcessedBatchMeta) *BatchQueue {
	return &BatchQueue{
		DAQueue:                 DAQueue,
		db:                      db,
		lastFinalizedBatchIndex: lastProcessedBatch.BatchIndex,
		batches:                 common.NewHeap[da.Entry](),
		batchesMap:              common.NewShrinkingMap[uint64, *common.HeapElement[da.Entry]](1000),
		previousBatch:           lastProcessedBatch,
	}
}

// NextBatch finds next finalized batch and returns data, that was committed in that batch
func (bq *BatchQueue) NextBatch(ctx context.Context) (da.EntryWithBlocks, error) {
	if batch := bq.nextFinalizedBatch(); batch != nil {
		return batch, nil
	}

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		daEntry, err := bq.DAQueue.NextDA(ctx)
		if err != nil {
			return nil, err
		}
		switch daEntry.Type() {
		case da.CommitBatchV0Type, da.CommitBatchWithBlobType:
			bq.addBatch(daEntry)
		case da.RevertBatchType:
			if err = bq.handleRevertEvent(daEntry.Event()); err != nil {
				return nil, fmt.Errorf("failed to handle revert event: %w", err)
			}
		case da.FinalizeBatchType:
			if daEntry.BatchIndex() > bq.lastFinalizedBatchIndex {
				bq.lastFinalizedBatchIndex = daEntry.BatchIndex()
			}

			if batch := bq.nextFinalizedBatch(); batch != nil {
				return batch, nil
			}
		default:
			return nil, fmt.Errorf("unexpected type of daEntry: %T", daEntry)
		}
	}
}

// nextFinalizedBatch returns next finalized batch if there is available
func (bq *BatchQueue) nextFinalizedBatch() da.EntryWithBlocks {
	if bq.batches.Len() == 0 {
		return nil
	}

	batch := bq.batches.Peek().Value()
	// we process all batches smaller or equal to the last finalized batch index -> this reflects bundles of multiple batches
	// where we only receive the finalize event for the last batch of the bundle.
	if batch.BatchIndex() <= bq.lastFinalizedBatchIndex {
		return bq.processAndDeleteBatch(batch)
	} else {
		return nil
	}
}

func (bq *BatchQueue) addBatch(batch da.Entry) {
	heapElement := bq.batches.Push(batch)
	bq.batchesMap.Set(batch.BatchIndex(), heapElement)
}

func (bq *BatchQueue) handleRevertEvent(event l1.RollupEvent) error {
	switch event.Type() {
	case l1.RevertEventV0Type:
		revertBatch, ok := event.(*l1.RevertBatchEventV0)
		if !ok {
			return fmt.Errorf("unexpected type of revert event: %T, expected RevertEventV0Type", event)
		}

		log.Info("reverting batch due to RevertEventV0Type", "batchIndex", revertBatch.BatchIndex())

		bq.deleteBatch(revertBatch.BatchIndex().Uint64())
	case l1.RevertEventV7Type:
		revertBatch, ok := event.(*l1.RevertBatchEventV7)
		if !ok {
			return fmt.Errorf("unexpected type of revert event: %T, expected RevertEventV7Type", event)
		}

		// delete all batches from revertBatch.StartBatchIndex (inclusive) to revertBatch.FinishBatchIndex (inclusive)
		for i := revertBatch.StartBatchIndex().Uint64(); i <= revertBatch.FinishBatchIndex().Uint64(); i++ {
			log.Info("reverting batch due to RevertEventV7Type", "batchIndex", i)
			bq.deleteBatch(i)
		}
	default:
		return fmt.Errorf("unexpected type of revert event: %T", event)
	}

	return nil
}

func (bq *BatchQueue) deleteBatch(batchIndex uint64) (deleted bool) {
	batchHeapElement, exists := bq.batchesMap.Get(batchIndex)
	if !exists {
		return false
	}

	bq.batchesMap.Delete(batchIndex)
	bq.batches.Remove(batchHeapElement)

	return true
}

// processAndDeleteBatch processes a batch and deletes the batch from map. Stores the syncing progress on disk.
func (bq *BatchQueue) processAndDeleteBatch(batch da.Entry) da.EntryWithBlocks {
	if !bq.deleteBatch(batch.BatchIndex()) {
		return nil
	}

	entryWithBlocks, ok := batch.(da.EntryWithBlocks)
	if !ok {
		// this should only happen if we delete a reverted batch
		return nil
	}

	// sanity check that the next batch is the one we expect. If not, we skip the batch.
	if bq.previousBatch.BatchIndex > 0 && bq.previousBatch.BatchIndex+1 != entryWithBlocks.BatchIndex() {
		log.Info("BatchQueue: skipping batch ", "currentBatch", entryWithBlocks.BatchIndex(), "previousBatch", bq.previousBatch.BatchIndex)
		return nil
	}

	// carry forward the total L1 messages popped from the previous batch
	entryWithBlocks.SetParentTotalL1MessagePopped(bq.previousBatch.TotalL1MessagesPopped)

	// we store the previous batch as it has been completely processed which we know because the next batch is requested within the pipeline.
	// In case of a restart or crash we can continue from the last processed batch (and its metadata).
	rawdb.WriteDAProcessedBatchMeta(bq.db, bq.previousBatch)

	log.Debug("processing batch", "batchIndex", entryWithBlocks.BatchIndex(), "L1BlockNumber", entryWithBlocks.L1BlockNumber(), "totalL1MessagesPopped", entryWithBlocks.TotalL1MessagesPopped(), "previousBatch", bq.previousBatch.BatchIndex, "previousL1BlockNumber", bq.previousBatch.L1BlockNumber, "previous TotalL1MessagesPopped", bq.previousBatch.TotalL1MessagesPopped)

	bq.previousBatch = &rawdb.DAProcessedBatchMeta{
		L1BlockNumber:         entryWithBlocks.L1BlockNumber(),
		BatchIndex:            entryWithBlocks.BatchIndex(),
		TotalL1MessagesPopped: entryWithBlocks.TotalL1MessagesPopped(),
	}

	return entryWithBlocks
}

func (bq *BatchQueue) Reset(lastProcessedBatchMeta *rawdb.DAProcessedBatchMeta) {
	bq.batches.Clear()
	bq.batchesMap.Clear()
	bq.lastFinalizedBatchIndex = lastProcessedBatchMeta.BatchIndex
	bq.previousBatch = lastProcessedBatchMeta
	bq.DAQueue.Reset(lastProcessedBatchMeta)
}
