package rawdb

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/scroll-tech/da-codec/encoding"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rlp"
)

// ChunkBlockRange represents the range of blocks within a chunk.
type ChunkBlockRange struct {
	StartBlockNumber uint64
	EndBlockNumber   uint64
}

// CommittedBatchMeta holds metadata for committed batches.
type CommittedBatchMeta struct {
	Version          uint8
	ChunkBlockRanges []*ChunkBlockRange

	// introduced with CodecV7
	PostL1MessageQueueHash common.Hash
}

type committedBatchMetaV0 struct {
	Version uint8
	// BlobVersionedHashes are the versioned hashes of the blobs in the batch. Currently unused. Left for compatibility.
	BlobVersionedHashes []common.Hash
	ChunkBlockRanges    []*ChunkBlockRange
}

type committedBatchMetaV7 struct {
	Version                uint8
	ChunkBlockRanges       []*ChunkBlockRange
	LastL1MessageQueueHash common.Hash
}

// FinalizedBatchMeta holds metadata for finalized batches.
type FinalizedBatchMeta struct {
	BatchHash            common.Hash
	TotalL1MessagePopped uint64 // total number of L1 messages popped before and in this batch.
	StateRoot            common.Hash
	WithdrawRoot         common.Hash
}

// WriteRollupEventSyncedL1BlockNumber stores the latest synced L1 block number related to rollup events in the database.
func WriteRollupEventSyncedL1BlockNumber(db ethdb.KeyValueWriter, l1BlockNumber uint64) {
	value := big.NewInt(0).SetUint64(l1BlockNumber).Bytes()
	if err := db.Put(rollupEventSyncedL1BlockNumberKey, value); err != nil {
		log.Crit("failed to store rollup event synced L1 block number for rollup event", "L1 block number", l1BlockNumber, "value", value, "err", err)
	}
}

// ReadRollupEventSyncedL1BlockNumber fetches the highest synced L1 block number associated with rollup events from the database.
func ReadRollupEventSyncedL1BlockNumber(db ethdb.Reader) *uint64 {
	data, err := db.Get(rollupEventSyncedL1BlockNumberKey)
	if err != nil && isNotFoundErr(err) {
		return nil
	}
	if err != nil {
		log.Crit("failed to read rollup event synced L1 block number from database", "err", err)
	}

	number := new(big.Int).SetBytes(data)
	if !number.IsUint64() {
		log.Crit("unexpected rollup event synced L1 block number in database", "data", data, "number", number)
	}

	rollupEventSyncedL1BlockNumber := number.Uint64()
	return &rollupEventSyncedL1BlockNumber
}

// WriteFinalizedBatchMeta stores the metadata of a finalized batch in the database.
func WriteFinalizedBatchMeta(db ethdb.KeyValueWriter, batchIndex uint64, finalizedBatchMeta *FinalizedBatchMeta) {
	value, err := rlp.EncodeToBytes(finalizedBatchMeta)
	if err != nil {
		log.Crit("failed to RLP encode finalized batch metadata", "batch index", batchIndex, "finalized batch meta", finalizedBatchMeta, "err", err)
	}
	if err := db.Put(batchMetaKey(batchIndex), value); err != nil {
		log.Crit("failed to store finalized batch metadata", "batch index", batchIndex, "value", value, "err", err)
	}
}

// ReadFinalizedBatchMeta fetches the metadata of a finalized batch from the database.
func ReadFinalizedBatchMeta(db ethdb.Reader, batchIndex uint64) *FinalizedBatchMeta {
	data, err := db.Get(batchMetaKey(batchIndex))
	if err != nil && isNotFoundErr(err) {
		return nil
	}
	if err != nil {
		log.Crit("failed to read finalized batch metadata from database", "batch index", batchIndex, "err", err)
	}

	fbm := new(FinalizedBatchMeta)
	if err := rlp.Decode(bytes.NewReader(data), fbm); err != nil {
		log.Crit("Invalid FinalizedBatchMeta RLP", "batch index", batchIndex, "data", data, "err", err)
	}
	return fbm
}

// WriteFinalizedL2BlockNumber stores the highest finalized L2 block number in the database.
func WriteFinalizedL2BlockNumber(db ethdb.KeyValueWriter, l2BlockNumber uint64) {
	value := big.NewInt(0).SetUint64(l2BlockNumber).Bytes()
	if err := db.Put(finalizedL2BlockNumberKey, value); err != nil {
		log.Crit("failed to store finalized L2 block number for rollup event", "L2 block number", l2BlockNumber, "value", value, "err", err)
	}
}

// ReadFinalizedL2BlockNumber fetches the highest finalized L2 block number from the database.
func ReadFinalizedL2BlockNumber(db ethdb.Reader) *uint64 {
	data, err := db.Get(finalizedL2BlockNumberKey)
	if err != nil && isNotFoundErr(err) {
		return nil
	}
	if err != nil {
		log.Crit("failed to read finalized L2 block number from database", "key", finalizedL2BlockNumberKey, "err", err)
	}

	number := new(big.Int).SetBytes(data)
	if !number.IsUint64() {
		log.Crit("unexpected finalized L2 block number in database", "data", data, "number", number)
	}

	finalizedL2BlockNumber := number.Uint64()
	return &finalizedL2BlockNumber
}

// WriteLastFinalizedBatchIndex stores the last finalized batch index in the database.
func WriteLastFinalizedBatchIndex(db ethdb.KeyValueWriter, lastFinalizedBatchIndex uint64) {
	value := big.NewInt(0).SetUint64(lastFinalizedBatchIndex).Bytes()
	if err := db.Put(lastFinalizedBatchIndexKey, value); err != nil {
		log.Crit("failed to store last finalized batch index for rollup event", "batch index", lastFinalizedBatchIndex, "value", value, "err", err)
	}
}

// ReadLastFinalizedBatchIndex fetches the last finalized batch index from the database.
func ReadLastFinalizedBatchIndex(db ethdb.Reader) *uint64 {
	data, err := db.Get(lastFinalizedBatchIndexKey)
	if err != nil && isNotFoundErr(err) {
		return nil
	}
	if err != nil {
		log.Crit("failed to read last finalized batch index from database", "key", lastFinalizedBatchIndexKey, "err", err)
	}

	number := new(big.Int).SetBytes(data)
	if !number.IsUint64() {
		log.Crit("unexpected finalized batch index in database", "data", data, "number", number)
	}

	lastFinalizedBatchIndex := number.Uint64()
	return &lastFinalizedBatchIndex
}

// WriteCommittedBatchMeta stores the CommittedBatchMeta for a specific batch in the database.
func WriteCommittedBatchMeta(db ethdb.KeyValueWriter, batchIndex uint64, committedBatchMeta *CommittedBatchMeta) {
	var committedBatchMetaToStore any
	if encoding.CodecVersion(committedBatchMeta.Version) < encoding.CodecV7 {
		committedBatchMetaToStore = &committedBatchMetaV0{
			Version:          committedBatchMeta.Version,
			ChunkBlockRanges: committedBatchMeta.ChunkBlockRanges,
		}
	} else {
		committedBatchMetaToStore = &committedBatchMetaV7{
			Version:                committedBatchMeta.Version,
			ChunkBlockRanges:       committedBatchMeta.ChunkBlockRanges,
			LastL1MessageQueueHash: committedBatchMeta.PostL1MessageQueueHash,
		}
	}

	value, err := rlp.EncodeToBytes(committedBatchMetaToStore)
	if err != nil {
		log.Crit("failed to RLP encode committed batch metadata", "batch index", batchIndex, "committed batch meta", committedBatchMetaToStore, "err", err)
	}
	if err := db.Put(committedBatchMetaKey(batchIndex), value); err != nil {
		log.Crit("failed to store committed batch metadata", "batch index", batchIndex, "value", value, "err", err)
	}
}

// ReadCommittedBatchMeta fetches the CommittedBatchMeta for a specific batch from the database.
func ReadCommittedBatchMeta(db ethdb.Reader, batchIndex uint64) (*CommittedBatchMeta, error) {
	data, err := db.Get(committedBatchMetaKey(batchIndex))
	if err != nil && isNotFoundErr(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read committed batch metadata from database: batch index %d, err: %w", batchIndex, err)
	}

	// Try decoding from the newest format for future proofness, then the older one for old data.
	cbm7 := new(committedBatchMetaV7)
	if err = rlp.Decode(bytes.NewReader(data), cbm7); err == nil {
		if encoding.CodecVersion(cbm7.Version) < encoding.CodecV7 {
			return nil, fmt.Errorf("unexpected committed batch metadata version: batch index %d, version %d", batchIndex, cbm7.Version)
		}
		return &CommittedBatchMeta{
			Version:                cbm7.Version,
			ChunkBlockRanges:       cbm7.ChunkBlockRanges,
			PostL1MessageQueueHash: cbm7.LastL1MessageQueueHash,
		}, nil
	}

	cbm0 := new(committedBatchMetaV0)
	if err = rlp.Decode(bytes.NewReader(data), cbm0); err != nil {
		return nil, fmt.Errorf("failed to decode committed batch metadata: batch index %d, err: %w", batchIndex, err)
	}

	return &CommittedBatchMeta{
		Version:                cbm0.Version,
		ChunkBlockRanges:       cbm0.ChunkBlockRanges,
		PostL1MessageQueueHash: common.Hash{},
	}, nil
}

// DeleteCommittedBatchMeta removes the block ranges of all chunks associated with a specific batch from the database.
// Note: Only non-finalized batches can be reverted.
func DeleteCommittedBatchMeta(db ethdb.KeyValueWriter, batchIndex uint64) {
	if err := db.Delete(committedBatchMetaKey(batchIndex)); err != nil {
		log.Crit("failed to delete committed batch metadata", "batch index", batchIndex, "err", err)
	}
}
