package blob_client

import (
	"context"
	"errors"
	"fmt"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto/kzg4844"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rollup/da_syncer/serrors"
)

const (
	lenBlobBytes     int = 131072
	lenKZGCommitment int = 48
)

type BlobClient interface {
	GetBlobByVersionedHashAndBlockNumber(ctx context.Context, versionedHash common.Hash, blockNumber uint64) (*kzg4844.Blob, error)
}

type BlobClients struct {
	list   []BlobClient
	curPos int
}

func NewBlobClients(blobClients ...BlobClient) *BlobClients {
	return &BlobClients{
		list:   blobClients,
		curPos: 0,
	}
}

func (c *BlobClients) GetBlobByVersionedHashAndBlockNumber(ctx context.Context, versionedHash common.Hash, blockNumber uint64) (*kzg4844.Blob, error) {
	if len(c.list) == 0 {
		return nil, fmt.Errorf("BlobClients.GetBlobByVersionedHash: list of BlobClients is empty")
	}

	for i := 0; i < len(c.list); i++ {
		blob, err := c.list[c.curPos].GetBlobByVersionedHashAndBlockNumber(ctx, versionedHash, blockNumber)
		if err == nil {
			return blob, nil
		}
		c.nextPos()
		// there was an error, try the next blob client in following iteration
		log.Warn("BlobClients: failed to get blob by versioned hash from BlobClient", "err", err, "blob client pos in BlobClients", c.curPos)
	}

	// if we iterated over entire list, return a temporary error that will be handled in syncing_pipeline with a backoff and retry
	return nil, serrors.NewTemporaryError(errors.New("BlobClients.GetBlobByVersionedHash: failed to get blob by versioned hash from all BlobClients"))
}

func (c *BlobClients) nextPos() {
	c.curPos = (c.curPos + 1) % len(c.list)
}

func (c *BlobClients) AddBlobClient(blobClient BlobClient) {
	c.list = append(c.list, blobClient)
}

func (c *BlobClients) Size() int {
	return len(c.list)
}
