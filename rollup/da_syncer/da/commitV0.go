package da

import (
	"encoding/binary"
	"fmt"

	"github.com/scroll-tech/da-codec/encoding"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/rawdb"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rollup/da_syncer/serrors"
	"github.com/scroll-tech/go-ethereum/rollup/l1"
	"github.com/scroll-tech/go-ethereum/rollup/missing_header_fields"
)

type CommitBatchDAV0 struct {
	db ethdb.Database

	version                    encoding.CodecVersion
	batchIndex                 uint64
	parentTotalL1MessagePopped uint64
	l1MessagesPopped           int
	skippedL1MessageBitmap     []byte
	chunks                     []*encoding.DAChunkRawTx

	event *l1.CommitBatchEvent
}

func NewCommitBatchDAV0(db ethdb.Database,
	codec encoding.Codec,
	commitEvent *l1.CommitBatchEvent,
	parentBatchHeader []byte,
	chunks [][]byte,
	skippedL1MessageBitmap []byte,
) (*CommitBatchDAV0, error) {
	decodedChunks, err := codec.DecodeDAChunksRawTx(chunks)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack chunks: %d, err: %w", commitEvent.BatchIndex().Uint64(), err)
	}

	return NewCommitBatchDAV0WithChunks(db, codec.Version(), commitEvent.BatchIndex().Uint64(), parentBatchHeader, decodedChunks, skippedL1MessageBitmap, commitEvent)
}

func NewCommitBatchDAV0WithChunks(db ethdb.Database,
	version encoding.CodecVersion,
	batchIndex uint64,
	parentBatchHeader []byte,
	decodedChunks []*encoding.DAChunkRawTx,
	skippedL1MessageBitmap []byte,
	event *l1.CommitBatchEvent,
) (*CommitBatchDAV0, error) {
	parentTotalL1MessagePopped := getBatchTotalL1MessagePopped(parentBatchHeader)

	return &CommitBatchDAV0{
		db:                         db,
		version:                    version,
		batchIndex:                 batchIndex,
		parentTotalL1MessagePopped: parentTotalL1MessagePopped,
		l1MessagesPopped:           getTotalMessagesPoppedFromChunks(decodedChunks),
		skippedL1MessageBitmap:     skippedL1MessageBitmap,
		chunks:                     decodedChunks,
		event:                      event,
	}, nil
}

func NewCommitBatchDAV0Empty(event *l1.CommitBatchEvent) *CommitBatchDAV0 {
	return &CommitBatchDAV0{
		batchIndex: 0,
		event:      event,
	}
}

func (c *CommitBatchDAV0) Version() encoding.CodecVersion {
	return c.version
}

func (c *CommitBatchDAV0) Chunks() []*encoding.DAChunkRawTx {
	return c.chunks
}

func (c *CommitBatchDAV0) BlobVersionedHashes() []common.Hash {
	return nil
}

func (c *CommitBatchDAV0) Type() Type {
	return CommitBatchV0Type
}

func (c *CommitBatchDAV0) L1BlockNumber() uint64 {
	return c.event.BlockNumber()
}

func (c *CommitBatchDAV0) Event() l1.RollupEvent {
	return c.event
}

func (c *CommitBatchDAV0) BatchIndex() uint64 {
	return c.batchIndex
}

func (c *CommitBatchDAV0) CompareTo(other Entry) int {
	if c.BatchIndex() < other.BatchIndex() {
		return -1
	} else if c.BatchIndex() > other.BatchIndex() {
		return 1
	}
	return 0
}

func (c *CommitBatchDAV0) Blocks(manager *missing_header_fields.Manager) ([]*PartialBlock, error) {
	l1Txs, err := getL1Messages(c.db, c.parentTotalL1MessagePopped, c.skippedL1MessageBitmap, c.l1MessagesPopped)
	if err != nil {
		return nil, fmt.Errorf("failed to get L1 messages for v0 batch %d: %w", c.batchIndex, err)
	}

	var blocks []*PartialBlock
	l1TxPointer := 0

	curL1TxIndex := c.parentTotalL1MessagePopped
	for _, chunk := range c.chunks {
		for blockIndex, daBlock := range chunk.Blocks {
			// create txs
			txs := make(types.Transactions, 0, daBlock.NumTransactions())
			// insert l1 msgs
			for l1TxPointer < len(l1Txs) && l1Txs[l1TxPointer].QueueIndex < curL1TxIndex+uint64(daBlock.NumL1Messages()) {
				l1Tx := types.NewTx(l1Txs[l1TxPointer])
				txs = append(txs, l1Tx)
				l1TxPointer++
			}
			curL1TxIndex += uint64(daBlock.NumL1Messages())

			// insert l2 txs
			txs = append(txs, chunk.Transactions[blockIndex]...)

			difficulty, stateRoot, coinbase, nonce, extraData, err := manager.GetMissingHeaderFields(daBlock.Number())
			if err != nil {
				return nil, fmt.Errorf("failed to get missing header fields for block %d: %w", daBlock.Number(), err)
			}

			block := NewPartialBlock(
				&PartialHeader{
					Number:     daBlock.Number(),
					Time:       daBlock.Timestamp(),
					BaseFee:    daBlock.BaseFee(),
					GasLimit:   daBlock.GasLimit(),
					Difficulty: difficulty,
					ExtraData:  extraData,
					StateRoot:  stateRoot,
					Coinbase:   coinbase,
					Nonce:      nonce,
				},
				txs)
			blocks = append(blocks, block)
		}
	}

	return blocks, nil
}

func (c *CommitBatchDAV0) SetParentTotalL1MessagePopped(totalL1MessagePopped uint64) {
	// we ignore setting parentTotalL1MessagePopped from outside as it is calculated from parent batch header for V0 batches
}

func (c *CommitBatchDAV0) TotalL1MessagesPopped() uint64 {
	return c.parentTotalL1MessagePopped + uint64(c.l1MessagesPopped)
}

func (c *CommitBatchDAV0) L1MessagesPoppedInBatch() uint64 {
	return uint64(c.l1MessagesPopped)
}

func getTotalMessagesPoppedFromChunks(decodedChunks []*encoding.DAChunkRawTx) int {
	totalL1MessagePopped := 0
	for _, chunk := range decodedChunks {
		for _, block := range chunk.Blocks {
			totalL1MessagePopped += int(block.NumL1Messages())
		}
	}
	return totalL1MessagePopped
}

func getL1Messages(db ethdb.Database, parentTotalL1MessagePopped uint64, skippedBitmap []byte, totalL1MessagePopped int) ([]*types.L1MessageTx, error) {
	var txs []*types.L1MessageTx

	decodedSkippedBitmap, err := encoding.DecodeBitmap(skippedBitmap, totalL1MessagePopped)
	if err != nil {
		return nil, fmt.Errorf("failed to decode skipped message bitmap: err: %w", err)
	}

	// get all necessary l1 messages without skipped
	currentIndex := parentTotalL1MessagePopped
	for index := 0; index < totalL1MessagePopped; index++ {
		if encoding.IsL1MessageSkipped(decodedSkippedBitmap, currentIndex-parentTotalL1MessagePopped) {
			currentIndex++
			continue
		}
		l1Tx := rawdb.ReadL1Message(db, currentIndex)
		if l1Tx == nil {
			log.Info("L1 message not yet available", "index", currentIndex)
			// message not yet available
			// we return serrors.EOFError as this will be handled in the syncing pipeline with a backoff and retry
			return nil, serrors.EOFError
		}
		txs = append(txs, l1Tx)
		currentIndex++
	}

	return txs, nil
}

func getBatchTotalL1MessagePopped(data []byte) uint64 {
	// total l1 message popped stored in bytes from 17 to 24, accordingly to codec spec
	return binary.BigEndian.Uint64(data[17:25])
}
