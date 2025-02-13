package rawdb

import (
	"testing"

	"github.com/stretchr/testify/require"

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
		t.Fatal("Expected nil for non-existing value", "got", *got)
	}

	for _, num := range blockNumbers {
		WriteFinalizedL2BlockNumber(db, num)
		got := ReadFinalizedL2BlockNumber(db)

		if *got != num {
			t.Fatal("Block number mismatch", "expected", num, "got", got)
		}
	}
}

func TestLastFinalizedBatchIndex(t *testing.T) {
	batchIndxes := []uint64{
		1,
		1 << 2,
		1 << 8,
		1 << 16,
		1 << 32,
	}

	db := NewMemoryDatabase()

	// read non-existing value
	if got := ReadLastFinalizedBatchIndex(db); got != nil {
		t.Fatal("Expected nil for non-existing value", "got", *got)
	}

	for _, num := range batchIndxes {
		WriteLastFinalizedBatchIndex(db, num)
		got := ReadLastFinalizedBatchIndex(db)

		if *got != num {
			t.Fatal("Batch index mismatch", "expected", num, "got", got)
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

func TestWriteReadDeleteCommittedBatchMeta(t *testing.T) {
	db := NewMemoryDatabase()

	testCases := []struct {
		batchIndex uint64
		meta       *CommittedBatchMeta
	}{
		{
			batchIndex: 0,
			meta: &CommittedBatchMeta{
				Version:          0,
				ChunkBlockRanges: []*ChunkBlockRange{},
			},
		},
		{
			batchIndex: 1,
			meta: &CommittedBatchMeta{
				Version:          1,
				ChunkBlockRanges: []*ChunkBlockRange{{StartBlockNumber: 0, EndBlockNumber: 10}},
			},
		},
		{
			batchIndex: 1,
			meta: &CommittedBatchMeta{
				Version:          2,
				ChunkBlockRanges: []*ChunkBlockRange{{StartBlockNumber: 0, EndBlockNumber: 10}},
			},
		},
		{
			batchIndex: 1,
			meta: &CommittedBatchMeta{
				Version:                7,
				ChunkBlockRanges:       []*ChunkBlockRange{{StartBlockNumber: 0, EndBlockNumber: 10}},
				LastL1MessageQueueHash: common.Hash{1, 2, 3, 4, 5, 6, 7},
			},
		},
		{
			batchIndex: 255,
			meta: &CommittedBatchMeta{
				Version:                255,
				ChunkBlockRanges:       []*ChunkBlockRange{{StartBlockNumber: 0, EndBlockNumber: 10}, {StartBlockNumber: 11, EndBlockNumber: 20}},
				LastL1MessageQueueHash: common.Hash{255},
			},
		},
	}

	for _, tc := range testCases {
		WriteCommittedBatchMeta(db, tc.batchIndex, tc.meta)
		got, err := ReadCommittedBatchMeta(db, tc.batchIndex)
		require.NoError(t, err)
		require.NotNil(t, got)

		if !compareCommittedBatchMeta(tc.meta, got) {
			t.Fatalf("CommittedBatchMeta mismatch for batch index %d, expected %+v, got %+v", tc.batchIndex, tc.meta, got)
		}
	}

	// reading a non-existing value
	got, err := ReadCommittedBatchMeta(db, 256)
	require.NoError(t, err)
	if got != nil {
		t.Fatalf("Expected nil for non-existing value, got %+v", got)
	}

	// delete: revert batch
	for _, tc := range testCases {
		DeleteCommittedBatchMeta(db, tc.batchIndex)

		readChunkRange, err := ReadCommittedBatchMeta(db, tc.batchIndex)
		require.NoError(t, err)
		require.Nil(t, readChunkRange, "Committed batch metadata was not deleted", "batch index", tc.batchIndex)
	}

	// delete non-existing value: ensure the delete operation handles non-existing values without errors.
	DeleteCommittedBatchMeta(db, 256)
}

func TestOverwriteCommittedBatchMeta(t *testing.T) {
	db := NewMemoryDatabase()

	batchIndex := uint64(42)
	initialMeta := &CommittedBatchMeta{
		Version:          1,
		ChunkBlockRanges: []*ChunkBlockRange{{StartBlockNumber: 0, EndBlockNumber: 10}},
	}
	newMeta := &CommittedBatchMeta{
		Version:                255,
		ChunkBlockRanges:       []*ChunkBlockRange{{StartBlockNumber: 0, EndBlockNumber: 20}, {StartBlockNumber: 21, EndBlockNumber: 30}},
		LastL1MessageQueueHash: common.Hash{255},
	}

	// write initial meta
	WriteCommittedBatchMeta(db, batchIndex, initialMeta)
	got, err := ReadCommittedBatchMeta(db, batchIndex)
	require.NoError(t, err)

	if !compareCommittedBatchMeta(initialMeta, got) {
		t.Fatalf("Initial write failed, expected %+v, got %+v", initialMeta, got)
	}

	// overwrite with new meta
	WriteCommittedBatchMeta(db, batchIndex, newMeta)
	got, err = ReadCommittedBatchMeta(db, batchIndex)
	require.NoError(t, err)

	if !compareCommittedBatchMeta(newMeta, got) {
		t.Fatalf("Overwrite failed, expected %+v, got %+v", newMeta, got)
	}

	// read non-existing batch index
	nonExistingIndex := uint64(999)
	got, err = ReadCommittedBatchMeta(db, nonExistingIndex)
	require.NoError(t, err)

	if got != nil {
		t.Fatalf("Expected nil for non-existing batch index, got %+v", got)
	}
}

func compareCommittedBatchMeta(a, b *CommittedBatchMeta) bool {
	if a.Version != b.Version {
		return false
	}

	if len(a.ChunkBlockRanges) != len(b.ChunkBlockRanges) {
		return false
	}
	for i := range a.ChunkBlockRanges {
		if a.ChunkBlockRanges[i].StartBlockNumber != b.ChunkBlockRanges[i].StartBlockNumber || a.ChunkBlockRanges[i].EndBlockNumber != b.ChunkBlockRanges[i].EndBlockNumber {
			return false
		}
	}

	return a.LastL1MessageQueueHash == b.LastL1MessageQueueHash
}
