package da

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/scroll-tech/da-codec/encoding"

	"github.com/scroll-tech/go-ethereum/core/rawdb"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rollup/da_syncer/blob_client"
	"github.com/scroll-tech/go-ethereum/rollup/da_syncer/serrors"
	"github.com/scroll-tech/go-ethereum/rollup/l1"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto/kzg4844"
	"github.com/scroll-tech/go-ethereum/ethdb"
)

type CommitBatchDAV7 struct {
	version               encoding.CodecVersion
	batchIndex            uint64
	initialL1MessageIndex uint64
	blocks                []encoding.DABlock
	transactions          []types.Transactions
	l1Txs                 []types.Transactions
	versionedHashes       []common.Hash

	event *l1.CommitBatchEvent
}

func NewCommitBatchDAV7(ctx context.Context, db ethdb.Database,
	blobClient blob_client.BlobClient,
	codec encoding.Codec,
	commitEvent *l1.CommitBatchEvent,
	blobHash common.Hash,
	parentBatchHash common.Hash,
	l1BlockTime uint64,
) (*CommitBatchDAV7, error) {
	calculatedBatch, err := codec.NewDABatchFromParams(commitEvent.BatchIndex().Uint64(), blobHash, parentBatchHash)
	if err != nil {
		return nil, fmt.Errorf("failed to create new DA batch from params, batch index: %d, err: %w", commitEvent.BatchIndex().Uint64(), err)
	}

	if calculatedBatch.Hash() != commitEvent.BatchHash() {
		return nil, fmt.Errorf("calculated batch hash is not equal to the one from commit event: %s, calculated hash: %s", commitEvent.BatchHash().Hex(), calculatedBatch.Hash().Hex())
	}

	blob, err := blobClient.GetBlobByVersionedHashAndBlockTime(ctx, blobHash, l1BlockTime)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch blob from blob client, err: %w", err)
	}
	if blob == nil {
		return nil, fmt.Errorf("unexpected, blob == nil and err != nil, batch index: %d, versionedHash: %s, blobClient: %T", commitEvent.BatchIndex().Uint64(), blobHash.Hex(), blobClient)
	}

	// compute blob versioned hash and compare with one from tx
	c, err := kzg4844.BlobToCommitment(blob)
	if err != nil {
		return nil, fmt.Errorf("failed to create blob commitment: %w", err)
	}
	blobVersionedHash := common.Hash(kzg4844.CalcBlobHashV1(sha256.New(), &c))
	if blobVersionedHash != blobHash {
		return nil, fmt.Errorf("blobVersionedHash from blob source is not equal to versionedHash from tx, correct versioned hash: %s, fetched blob hash: %s", blobHash.Hex(), blobVersionedHash.Hex())
	}

	blobPayload, err := codec.DecodeBlob(blob)
	if err != nil {
		return nil, fmt.Errorf("failed to decode blob: %w", err)
	}

	l1Txs, err := getL1MessagesV7(db, blobPayload.Blocks(), blobPayload.InitialL1MessageIndex())
	if err != nil {
		return nil, fmt.Errorf("failed to get L1 messages for v7 batch %d: %w", commitEvent.BatchIndex().Uint64(), err)
	}

	return &CommitBatchDAV7{
		version:               codec.Version(),
		batchIndex:            commitEvent.BatchIndex().Uint64(),
		initialL1MessageIndex: blobPayload.InitialL1MessageIndex(),
		blocks:                blobPayload.Blocks(),
		transactions:          blobPayload.Transactions(),
		l1Txs:                 l1Txs,
		versionedHashes:       []common.Hash{blobVersionedHash},
		event:                 commitEvent,
	}, nil
}

func (c *CommitBatchDAV7) Type() Type {
	return CommitBatchWithBlobType
}

func (c *CommitBatchDAV7) BlobVersionedHashes() []common.Hash {
	return c.versionedHashes
}

func (c *CommitBatchDAV7) BatchIndex() uint64 {
	return c.batchIndex
}

func (c *CommitBatchDAV7) L1BlockNumber() uint64 {
	return c.event.BlockNumber()
}

func (c *CommitBatchDAV7) CompareTo(other Entry) int {
	if c.BatchIndex() < other.BatchIndex() {
		return -1
	} else if c.BatchIndex() > other.BatchIndex() {
		return 1
	}
	return 0
}

func (c *CommitBatchDAV7) Event() l1.RollupEvent {
	return c.event
}

func (c *CommitBatchDAV7) Blocks() []*PartialBlock {
	var blocks []*PartialBlock

	for i, daBlock := range c.blocks {
		// create txs
		txs := make(types.Transactions, 0, daBlock.NumTransactions())

		// insert L1 messages
		txs = append(txs, c.l1Txs[i]...)

		// insert L2 txs
		txs = append(txs, c.transactions[i]...)

		block := NewPartialBlock(
			&PartialHeader{
				Number:     daBlock.Number(),
				Time:       daBlock.Timestamp(),
				BaseFee:    daBlock.BaseFee(),
				GasLimit:   daBlock.GasLimit(),
				Difficulty: 1,        // difficulty is enforced to be 1
				ExtraData:  []byte{}, // extra data is enforced to be empty or at least excluded from the block hash
			},
			txs)
		blocks = append(blocks, block)
	}

	return blocks
}

func (c *CommitBatchDAV7) Version() encoding.CodecVersion {
	return c.version
}

func (c *CommitBatchDAV7) Chunks() []*encoding.DAChunkRawTx {
	return []*encoding.DAChunkRawTx{
		{
			Blocks:       c.blocks,
			Transactions: c.transactions,
		},
	}
}

func getL1MessagesV7(db ethdb.Database, blocks []encoding.DABlock, initialL1MessageIndex uint64) ([]types.Transactions, error) {
	allTxs := make([]types.Transactions, 0, len(blocks))

	messageIndex := initialL1MessageIndex
	totalL1Messages := 0
	for _, block := range blocks {
		var txsPerBlock types.Transactions
		for i := messageIndex; i < messageIndex+uint64(block.NumL1Messages()); i++ {
			l1Tx := rawdb.ReadL1Message(db, i)
			if l1Tx == nil {
				log.Info("L1 message not yet available", "index", i)
				// message not yet available
				// we return serrors.EOFError as this will be handled in the syncing pipeline with a backoff and retry
				return nil, serrors.EOFError
			}

			txsPerBlock = append(txsPerBlock, types.NewTx(l1Tx))
		}

		totalL1Messages += int(block.NumL1Messages())
		messageIndex += uint64(block.NumL1Messages())
		allTxs = append(allTxs, txsPerBlock)
	}

	if messageIndex != initialL1MessageIndex+uint64(totalL1Messages) {
		return nil, fmt.Errorf("unexpected message index: %d, expected: %d", messageIndex, initialL1MessageIndex+uint64(totalL1Messages))
	}

	return allTxs, nil
}
