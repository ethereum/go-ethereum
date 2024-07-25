package rawdb

import (
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
)

func TestWriteRollupEventSyncedL1BlockNumber(t *testing.T) {
	blockNumbers := []uint64{
		1,
		1 << 2,
		1 << 8,
		1 << 16,
		1 << 32,
	}

	db := NewMemoryDatabase()

	// read non-existing value
	if got := ReadRollupEventSyncedL1BlockNumber(db); got != nil {
		t.Fatal("Expected 0 for non-existing value", "got", *got)
	}

	for _, num := range blockNumbers {
		WriteRollupEventSyncedL1BlockNumber(db, num)
		got := ReadRollupEventSyncedL1BlockNumber(db)

		if *got != num {
			t.Fatal("Block number mismatch", "expected", num, "got", got)
		}
	}
}

func TestFinalizedL2BlockNumber(t *testing.T) {
	blockNumbers := []uint64{
		1,
		1 << 2,
		1 << 8,
		1 << 16,
		1 << 32,
	}

	db := NewMemoryDatabase()

	// read non-existing value
	if got := ReadFinalizedL2BlockNumber(db); got != nil {
		t.Fatal("Expected 0 for non-existing value", "got", *got)
	}

	for _, num := range blockNumbers {
		WriteFinalizedL2BlockNumber(db, num)
		got := ReadFinalizedL2BlockNumber(db)

		if *got != num {
			t.Fatal("Block number mismatch", "expected", num, "got", got)
		}
	}
}

func TestFinalizedBatchMeta(t *testing.T) {
	batches := []*FinalizedBatchMeta{
		{
			BatchHash:            common.BytesToHash([]byte("batch1")),
			TotalL1MessagePopped: 123,
			StateRoot:            common.BytesToHash([]byte("stateRoot1")),
			WithdrawRoot:         common.BytesToHash([]byte("withdrawRoot1")),
		},
		{
			BatchHash:            common.BytesToHash([]byte("batch2")),
			TotalL1MessagePopped: 456,
			StateRoot:            common.BytesToHash([]byte("stateRoot2")),
			WithdrawRoot:         common.BytesToHash([]byte("withdrawRoot2")),
		},
		{
			BatchHash:            common.BytesToHash([]byte("batch3")),
			TotalL1MessagePopped: 789,
			StateRoot:            common.BytesToHash([]byte("stateRoot3")),
			WithdrawRoot:         common.BytesToHash([]byte("withdrawRoot3")),
		},
	}

	db := NewMemoryDatabase()

	for i, batch := range batches {
		batchIndex := uint64(i)
		WriteFinalizedBatchMeta(db, batchIndex, batch)
	}

	for i, batch := range batches {
		batchIndex := uint64(i)
		readBatch := ReadFinalizedBatchMeta(db, batchIndex)
		if readBatch == nil {
			t.Fatal("Failed to read batch from database")
		}
		if readBatch.BatchHash != batch.BatchHash || readBatch.TotalL1MessagePopped != batch.TotalL1MessagePopped ||
			readBatch.StateRoot != batch.StateRoot || readBatch.WithdrawRoot != batch.WithdrawRoot {
			t.Fatal("Mismatch in read batch", "expected", batch, "got", readBatch)
		}
	}

	// over-write
	newBatch := &FinalizedBatchMeta{
		BatchHash:            common.BytesToHash([]byte("newBatch")),
		TotalL1MessagePopped: 999,
		StateRoot:            common.BytesToHash([]byte("newStateRoot")),
		WithdrawRoot:         common.BytesToHash([]byte("newWithdrawRoot")),
	}
	WriteFinalizedBatchMeta(db, 0, newBatch) // over-writing the batch with index 0
	readBatch := ReadFinalizedBatchMeta(db, 0)
	if readBatch.BatchHash != newBatch.BatchHash || readBatch.TotalL1MessagePopped != newBatch.TotalL1MessagePopped ||
		readBatch.StateRoot != newBatch.StateRoot || readBatch.WithdrawRoot != newBatch.WithdrawRoot {
		t.Fatal("Mismatch after over-writing batch", "expected", newBatch, "got", readBatch)
	}

	// read non-existing value
	nonExistingIndex := uint64(len(batches) + 1)
	readBatch = ReadFinalizedBatchMeta(db, nonExistingIndex)
	if readBatch != nil {
		t.Fatal("Expected nil for non-existing value", "got", readBatch)
	}
}

func TestBatchChunkRanges(t *testing.T) {
	chunks := [][]*ChunkBlockRange{
		{
			{StartBlockNumber: 1, EndBlockNumber: 100},
			{StartBlockNumber: 101, EndBlockNumber: 200},
		},
		{
			{StartBlockNumber: 201, EndBlockNumber: 300},
			{StartBlockNumber: 301, EndBlockNumber: 400},
		},
		{
			{StartBlockNumber: 401, EndBlockNumber: 500},
		},
	}

	db := NewMemoryDatabase()

	for i, chunkRange := range chunks {
		batchIndex := uint64(i)
		WriteBatchChunkRanges(db, batchIndex, chunkRange)
	}

	for i, chunkRange := range chunks {
		batchIndex := uint64(i)
		readChunkRange := ReadBatchChunkRanges(db, batchIndex)
		if len(readChunkRange) != len(chunkRange) {
			t.Fatal("Mismatch in number of chunk ranges", "expected", len(chunkRange), "got", len(readChunkRange))
		}

		for j, cr := range readChunkRange {
			if cr.StartBlockNumber != chunkRange[j].StartBlockNumber || cr.EndBlockNumber != chunkRange[j].EndBlockNumber {
				t.Fatal("Mismatch in chunk range", "batch index", batchIndex, "expected", chunkRange[j], "got", cr)
			}
		}
	}

	// over-write
	newRange := []*ChunkBlockRange{{StartBlockNumber: 1001, EndBlockNumber: 1100}}
	WriteBatchChunkRanges(db, 0, newRange)
	readChunkRange := ReadBatchChunkRanges(db, 0)
	if len(readChunkRange) != 1 || readChunkRange[0].StartBlockNumber != 1001 || readChunkRange[0].EndBlockNumber != 1100 {
		t.Fatal("Over-write failed for chunk range", "expected", newRange, "got", readChunkRange)
	}

	// read non-existing value
	if readChunkRange = ReadBatchChunkRanges(db, uint64(len(chunks)+1)); readChunkRange != nil {
		t.Fatal("Expected nil for non-existing value", "got", readChunkRange)
	}

	// delete: revert batch
	for i := range chunks {
		batchIndex := uint64(i)
		DeleteBatchChunkRanges(db, batchIndex)

		readChunkRange := ReadBatchChunkRanges(db, batchIndex)
		if readChunkRange != nil {
			t.Fatal("Chunk range was not deleted", "batch index", batchIndex)
		}
	}

	// delete non-existing value: ensure the delete operation handles non-existing values without errors.
	DeleteBatchChunkRanges(db, uint64(len(chunks)+1))
}
