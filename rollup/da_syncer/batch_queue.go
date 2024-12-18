package da_syncer

import (
	"context"
	"fmt"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/rawdb"
	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/rollup/da_syncer/da"
)

// BatchQueue is a pipeline stage that reads all batch events from DAQueue and provides only finalized batches to the next stage.
type BatchQueue struct {
	DAQueue                 *DAQueue
	db                      ethdb.Database
	lastFinalizedBatchIndex uint64
	batches                 *common.Heap[da.Entry]
	batchesMap              *common.ShrinkingMap[uint64, *common.HeapElement[da.Entry]]
}

func NewBatchQueue(DAQueue *DAQueue, db ethdb.Database) *BatchQueue {
	return &BatchQueue{
		DAQueue:                 DAQueue,
		db:                      db,
		lastFinalizedBatchIndex: 0,
		batches:                 common.NewHeap[da.Entry](),
		batchesMap:              common.NewShrinkingMap[uint64, *common.HeapElement[da.Entry]](1000),
	}
}

// NextBatch finds next finalized batch and returns data, that was committed in that batch
func (bq *BatchQueue) NextBatch(ctx context.Context) (da.Entry, error) {
	if batch := bq.getFinalizedBatch(); batch != nil {
		return batch, nil
	}

	for {
		daEntry, err := bq.DAQueue.NextDA(ctx)
		if err != nil {
			return nil, err
		}
		switch daEntry.Type() {
		case da.CommitBatchV0Type, da.CommitBatchWithBlobType:
			bq.addBatch(daEntry)
		case da.RevertBatchType:
			bq.deleteBatch(daEntry)
		case da.FinalizeBatchType:
			if daEntry.BatchIndex() > bq.lastFinalizedBatchIndex {
				bq.lastFinalizedBatchIndex = daEntry.BatchIndex()
			}

			if batch := bq.getFinalizedBatch(); batch != nil {
				return batch, nil
			}
		default:
			return nil, fmt.Errorf("unexpected type of daEntry: %T", daEntry)
		}
	}
}

// getFinalizedBatch returns next finalized batch if there is available
func (bq *BatchQueue) getFinalizedBatch() da.Entry {
	if bq.batches.Len() == 0 {
		return nil
	}

	batch := bq.batches.Peek().Value()
	if batch.BatchIndex() <= bq.lastFinalizedBatchIndex {
		bq.deleteBatch(batch)
		return batch
	} else {
		return nil
	}
}

func (bq *BatchQueue) addBatch(batch da.Entry) {
	heapElement := bq.batches.Push(batch)
	bq.batchesMap.Set(batch.BatchIndex(), heapElement)
}

// deleteBatch deletes data committed in the batch from map, because this batch is reverted or finalized
// updates DASyncedL1BlockNumber
func (bq *BatchQueue) deleteBatch(batch da.Entry) {
	batchHeapElement, exists := bq.batchesMap.Get(batch.BatchIndex())
	if !exists {
		return
	}

	bq.batchesMap.Delete(batch.BatchIndex())
	bq.batches.Remove(batchHeapElement)

	// we store here min height of currently loaded batches to be able to start syncing from the same place in case of restart
	// TODO: we should store this information when the batch is done being processed to avoid inconsistencies
	rawdb.WriteDASyncedL1BlockNumber(bq.db, batch.L1BlockNumber()-1)
}

func (bq *BatchQueue) Reset(height uint64) {
	bq.batches.Clear()
	bq.batchesMap.Clear()
	bq.lastFinalizedBatchIndex = 0
	bq.DAQueue.Reset(height)
}
