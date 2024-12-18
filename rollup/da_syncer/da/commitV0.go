package da

import (
	"encoding/binary"
	"fmt"

	"github.com/scroll-tech/da-codec/encoding"

	"github.com/scroll-tech/go-ethereum/core/rawdb"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/rollup/da_syncer/serrors"
)

type CommitBatchDAV0 struct {
	version                    uint8
	batchIndex                 uint64
	parentTotalL1MessagePopped uint64
	skippedL1MessageBitmap     []byte
	chunks                     []*encoding.DAChunkRawTx
	l1Txs                      []*types.L1MessageTx

	l1BlockNumber uint64
}

func NewCommitBatchDAV0(db ethdb.Database,
	codec encoding.Codec,
	version uint8,
	batchIndex uint64,
	parentBatchHeader []byte,
	chunks [][]byte,
	skippedL1MessageBitmap []byte,
	l1BlockNumber uint64,
) (*CommitBatchDAV0, error) {
	decodedChunks, err := codec.DecodeDAChunksRawTx(chunks)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack chunks: %d, err: %w", batchIndex, err)
	}

	return NewCommitBatchDAV0WithChunks(db, version, batchIndex, parentBatchHeader, decodedChunks, skippedL1MessageBitmap, l1BlockNumber)
}

func NewCommitBatchDAV0WithChunks(db ethdb.Database,
	version uint8,
	batchIndex uint64,
	parentBatchHeader []byte,
	decodedChunks []*encoding.DAChunkRawTx,
	skippedL1MessageBitmap []byte,
	l1BlockNumber uint64,
) (*CommitBatchDAV0, error) {
	parentTotalL1MessagePopped := getBatchTotalL1MessagePopped(parentBatchHeader)
	l1Txs, err := getL1Messages(db, parentTotalL1MessagePopped, skippedL1MessageBitmap, getTotalMessagesPoppedFromChunks(decodedChunks))
	if err != nil {
		return nil, fmt.Errorf("failed to get L1 messages for v0 batch %d: %w", batchIndex, err)
	}

	return &CommitBatchDAV0{
		version:                    version,
		batchIndex:                 batchIndex,
		parentTotalL1MessagePopped: parentTotalL1MessagePopped,
		skippedL1MessageBitmap:     skippedL1MessageBitmap,
		chunks:                     decodedChunks,
		l1Txs:                      l1Txs,
		l1BlockNumber:              l1BlockNumber,
	}, nil
}

func NewCommitBatchDAV0Empty() *CommitBatchDAV0 {
	return &CommitBatchDAV0{
		batchIndex: 0,
	}
}

func (c *CommitBatchDAV0) Type() Type {
	return CommitBatchV0Type
}

func (c *CommitBatchDAV0) L1BlockNumber() uint64 {
	return c.l1BlockNumber
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

func (c *CommitBatchDAV0) Blocks() []*PartialBlock {
	var blocks []*PartialBlock
	l1TxPointer := 0

	curL1TxIndex := c.parentTotalL1MessagePopped
	for _, chunk := range c.chunks {
		for blockId, daBlock := range chunk.Blocks {
			// create txs
			txs := make(types.Transactions, 0, daBlock.NumTransactions())
			// insert l1 msgs
			for l1TxPointer < len(c.l1Txs) && c.l1Txs[l1TxPointer].QueueIndex < curL1TxIndex+uint64(daBlock.NumL1Messages()) {
				l1Tx := types.NewTx(c.l1Txs[l1TxPointer])
				txs = append(txs, l1Tx)
				l1TxPointer++
			}
			curL1TxIndex += uint64(daBlock.NumL1Messages())

			// insert l2 txs
			txs = append(txs, chunk.Transactions[blockId]...)

			block := NewPartialBlock(
				&PartialHeader{
					Number:     daBlock.Number(),
					Time:       daBlock.Timestamp(),
					BaseFee:    daBlock.BaseFee(),
					GasLimit:   daBlock.GasLimit(),
					Difficulty: 10,                             // TODO: replace with real difficulty
					ExtraData:  []byte{1, 2, 3, 4, 5, 6, 7, 8}, // TODO: replace with real extra data
				},
				txs)
			blocks = append(blocks, block)
		}
	}

	return blocks
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
