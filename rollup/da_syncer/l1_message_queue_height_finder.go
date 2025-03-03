package da_syncer

import (
	"context"
	"fmt"

	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/rollup/da_syncer/blob_client"
	"github.com/scroll-tech/go-ethereum/rollup/da_syncer/da"
	"github.com/scroll-tech/go-ethereum/rollup/l1"
)

type L1MessageQueueHeightFinder struct {
	ctx                context.Context
	calldataBlobSource *da.CalldataBlobSource
	l1Reader           *l1.Reader
}

func NewL1MessageQueueHeightFinder(ctx context.Context, l1height uint64, l1Reader *l1.Reader, blobClient blob_client.BlobClient, db ethdb.Database) (*L1MessageQueueHeightFinder, error) {
	calldataBlobSource, err := da.NewCalldataBlobSource(ctx, l1height, l1Reader, blobClient, db)
	if err != nil {
		return nil, fmt.Errorf("failed to create calldata blob source: %w", err)
	}

	return &L1MessageQueueHeightFinder{
		ctx:                ctx,
		calldataBlobSource: calldataBlobSource,
		l1Reader:           l1Reader,
	}, nil
}

// TotalL1MessagesPoppedBefore finds the total L1 messages popped (L1 message queue height) before target batch.
// It does so by:
// 1. find bundle in which target batch was finalized
// 2. fetch the tx of the bundle to get the height of the L1 message queue after the bundle
// 3. with this information we can calculate the L1 message count for each batch from last finalized bundle to the target batch.
func (f *L1MessageQueueHeightFinder) TotalL1MessagesPoppedBefore(targetBatch uint64) (uint64, error) {
	batches := make(map[uint64]da.EntryWithBlocks)

	finalizedBundle, err := f.findFinalizedBundle(targetBatch, batches)
	if err != nil {
		return 0, fmt.Errorf("failed to find the bundle in which the target batch was finalized")
	}

	// 2. fetch tx of the bundle to get the TotalL1MessagesPoppedOverall after the bundle and the first batch within the bundle.
	args, err := f.l1Reader.FetchFinalizeTxDataPostEuclidV2(finalizedBundle.Event().(*l1.FinalizeBatchEvent))
	if err != nil {
		return 0, fmt.Errorf("failed to fetch finalize tx data: %w", err)
	}

	// 3. with this information we can calculate the L1 message queue height for target batch: for each batch from last finalized batch to the target batch subtract L1 messages popped in the batch from L1 message queue height
	lastBatchInBundle := finalizedBundle.BatchIndex()

	var l1MessageQueueHeight uint64
	// totalL1MessagesPoppedOverall is the number of messages processed after the bundle -> subtract 1 to get the last message in the bundle
	if args.TotalL1MessagesPoppedOverall.Uint64() > 0 {
		l1MessageQueueHeight = args.TotalL1MessagesPoppedOverall.Uint64() - 1
	}

	for i := lastBatchInBundle; i >= targetBatch; i-- {
		batch, ok := batches[i]
		if !ok {
			return 0, fmt.Errorf("batch %d not found", i)
		}

		if batch.L1MessagesPoppedInBatch() > l1MessageQueueHeight {
			return 0, fmt.Errorf("L1 message queue height is less than L1 messages popped in batch %d (%d < %d)", i, l1MessageQueueHeight, batch.L1MessagesPoppedInBatch())
		}
		l1MessageQueueHeight -= batch.L1MessagesPoppedInBatch()
	}

	return l1MessageQueueHeight, nil
}

func (f *L1MessageQueueHeightFinder) findFinalizedBundle(targetBatch uint64, batches map[uint64]da.EntryWithBlocks) (*da.FinalizeBatch, error) {
	for {
		// 1. find bundle in which target batch was finalized
		daEntries, err := f.calldataBlobSource.NextData()
		if err != nil {
			return nil, fmt.Errorf("failed to get next data: %w", err)
		}

		for _, daEntry := range daEntries {
			switch daEntry.Type() {
			case da.CommitBatchV0Type, da.CommitBatchWithBlobType:
				daEntryWithBlocks, ok := daEntry.(da.EntryWithBlocks)
				if !ok {
					return nil, fmt.Errorf("unexpected type of daEntry: %T, expected EntryWithBlocks", daEntry)
				}

				// save the batch for later use
				batches[daEntry.BatchIndex()] = daEntryWithBlocks
			case da.RevertBatchType:
				if err = f.handleRevertEvent(batches, daEntry.Event()); err != nil {
					return nil, fmt.Errorf("failed to handle revert event: %w", err)
				}
			case da.FinalizeBatchType:
				// the finalized event is triggered only for the last batch in the bundle:
				// we found the bundle in which the target batch was finalized
				if daEntry.BatchIndex() >= targetBatch {
					return daEntry.(*da.FinalizeBatch), nil
				}

			default:
				return nil, fmt.Errorf("unexpected type of daEntry: %T", daEntry)
			}
		}
	}

}

func (f *L1MessageQueueHeightFinder) handleRevertEvent(batches map[uint64]da.EntryWithBlocks, event l1.RollupEvent) error {
	switch event.Type() {
	case l1.RevertEventV0Type:
		revertBatch, ok := event.(*l1.RevertBatchEventV0)
		if !ok {
			return fmt.Errorf("unexpected type of revert event: %T, expected RevertEventV0Type", event)
		}

		delete(batches, revertBatch.BatchIndex().Uint64())
	case l1.RevertEventV7Type:
		revertBatch, ok := event.(*l1.RevertBatchEventV7)
		if !ok {
			return fmt.Errorf("unexpected type of revert event: %T, expected RevertEventV7Type", event)
		}

		// delete all batches from revertBatch.StartBatchIndex (inclusive) to revertBatch.FinishBatchIndex (inclusive)
		for i := revertBatch.StartBatchIndex().Uint64(); i <= revertBatch.FinishBatchIndex().Uint64(); i++ {
			delete(batches, i)
		}
	default:
		return fmt.Errorf("unexpected type of revert event: %T", event)
	}

	return nil
}
