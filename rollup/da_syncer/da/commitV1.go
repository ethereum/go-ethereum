package da

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/scroll-tech/da-codec/encoding"

	"github.com/scroll-tech/go-ethereum/rollup/da_syncer/blob_client"
	"github.com/scroll-tech/go-ethereum/rollup/l1"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto/kzg4844"
	"github.com/scroll-tech/go-ethereum/ethdb"
)

type CommitBatchDAV1 struct {
	*CommitBatchDAV0
}

func NewCommitBatchDAWithBlob(ctx context.Context, db ethdb.Database,
	l1Reader *l1.Reader,
	blobClient blob_client.BlobClient,
	codec encoding.Codec,
	commitEvent *l1.CommitBatchEvent,
	parentBatchHeader []byte,
	chunks [][]byte,
	skippedL1MessageBitmap []byte,
) (*CommitBatchDAV1, error) {
	decodedChunks, err := codec.DecodeDAChunksRawTx(chunks)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack chunks: %v, err: %w", commitEvent.BatchIndex().Uint64(), err)
	}

	versionedHash, err := l1Reader.FetchTxBlobHash(commitEvent.TxHash(), commitEvent.BlockHash())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch blob hash, err: %w", err)
	}

	header, err := l1Reader.FetchBlockHeaderByNumber(commitEvent.BlockNumber())
	if err != nil {
		return nil, fmt.Errorf("failed to get header by number, err: %w", err)
	}
	blob, err := blobClient.GetBlobByVersionedHashAndBlockTime(ctx, versionedHash, header.Time)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch blob from blob client, err: %w", err)
	}
	if blob == nil {
		return nil, fmt.Errorf("unexpected, blob == nil and err != nil, batch index: %d, versionedHash: %s, blobClient: %T", commitEvent.BatchIndex().Uint64(), versionedHash.String(), blobClient)
	}

	// compute blob versioned hash and compare with one from tx
	c, err := kzg4844.BlobToCommitment(blob)
	if err != nil {
		return nil, fmt.Errorf("failed to create blob commitment: %w", err)
	}
	blobVersionedHash := common.Hash(kzg4844.CalcBlobHashV1(sha256.New(), &c))
	if blobVersionedHash != versionedHash {
		return nil, fmt.Errorf("blobVersionedHash from blob source is not equal to versionedHash from tx, correct versioned hash: %s, fetched blob hash: %s", versionedHash.String(), blobVersionedHash.String())
	}

	// decode txs from blob
	err = codec.DecodeTxsFromBlob(blob, decodedChunks)
	if err != nil {
		return nil, fmt.Errorf("failed to decode txs from blob: %w", err)
	}

	if decodedChunks == nil {
		return nil, fmt.Errorf("decodedChunks is nil after decoding")
	}

	v0, err := NewCommitBatchDAV0WithChunks(db, uint8(codec.Version()), commitEvent.BatchIndex().Uint64(), parentBatchHeader, decodedChunks, skippedL1MessageBitmap, commitEvent.BlockNumber())
	if err != nil {
		return nil, err
	}

	return &CommitBatchDAV1{v0}, nil
}

func (c *CommitBatchDAV1) Type() Type {
	return CommitBatchWithBlobType
}
