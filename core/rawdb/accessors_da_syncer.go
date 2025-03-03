package rawdb

import (
	"bytes"
	"math/big"

	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rlp"
)

type DAProcessedBatchMeta struct {
	BatchIndex            uint64
	L1BlockNumber         uint64
	TotalL1MessagesPopped uint64
}

// WriteDAProcessedBatchMeta writes the batch metadata of the latest processed DA batch.
func WriteDAProcessedBatchMeta(db ethdb.KeyValueWriter, daProcessedBatchMeta *DAProcessedBatchMeta) {
	value, err := rlp.EncodeToBytes(daProcessedBatchMeta)
	if err != nil {
		log.Crit("failed to RLP encode committed batch metadata", "batch index", daProcessedBatchMeta.BatchIndex, "committed batch meta", daProcessedBatchMeta, "err", err)
	}
	if err := db.Put(daSyncedL1BlockNumberKey, value); err != nil {
		log.Crit("Failed to update DAProcessedBatchMeta", "err", err)
	}
}

// ReadDAProcessedBatchMeta retrieves the batch metadata of the latest processed DA batch.
func ReadDAProcessedBatchMeta(db ethdb.Reader) *DAProcessedBatchMeta {
	data, err := db.Get(daSyncedL1BlockNumberKey)
	if err != nil && isNotFoundErr(err) {
		return nil
	}
	if err != nil {
		log.Crit("Failed to read DA synced L1 block number from database", "err", err)
	}
	if len(data) == 0 {
		return nil
	}

	// Try decoding from the newest format for future proofness, then the older one for old data.
	daProcessedBatchMeta := new(DAProcessedBatchMeta)
	if err = rlp.Decode(bytes.NewReader(data), daProcessedBatchMeta); err == nil {
		return daProcessedBatchMeta
	}

	// Before storing DAProcessedBatchMeta we used to store a single uint64 value for the L1 block number.
	l1BlockNumber := new(big.Int).SetBytes(data)
	if !l1BlockNumber.IsUint64() {
		log.Crit("Unexpected DA synced L1 block number in database", "number", l1BlockNumber)
	}

	// We can simply set only the L1BlockNumber because carrying forward the totalL1MessagesPopped is not required before EuclidV2 (CodecV7)
	// (the parentTotalL1MessagePopped is given via the parentBatchHeader).
	// Nodes need to update to the new version to be able to continue syncing after EuclidV2 (CodecV7). Therefore,
	// the only nodes that might read a uint64 value are nodes that were running L1 follower before the EuclidV2.
	return &DAProcessedBatchMeta{
		BatchIndex:            0,
		L1BlockNumber:         l1BlockNumber.Uint64(),
		TotalL1MessagesPopped: 0,
	}
}
