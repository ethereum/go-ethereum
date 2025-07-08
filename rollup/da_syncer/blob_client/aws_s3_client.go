package blob_client

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/crypto/kzg4844"
)

const (
	AwsS3DefaultTimeout = 15 * time.Second
)

type AwsS3Client struct {
	client      *http.Client
	apiEndpoint string
}

func NewAwsS3Client(apiEndpoint string) *AwsS3Client {
	return &AwsS3Client{
		apiEndpoint: apiEndpoint,
		client:      &http.Client{Timeout: AwsS3DefaultTimeout},
	}
}

func (c *AwsS3Client) GetBlobByVersionedHashAndBlockTime(ctx context.Context, versionedHash common.Hash, blockTime uint64) (*kzg4844.Blob, error) {
	// Scroll mainnet blob data AWS S3 endpoint:  https://scroll-mainnet-blob-data.s3.us-west-2.amazonaws.com/
	// Scroll sepolia blob data AWS S3 endpoint:  https://scroll-sepolia-blob-data.s3.us-west-2.amazonaws.com/
	path, err := url.JoinPath(c.apiEndpoint, versionedHash.String())
	if err != nil {
		return nil, fmt.Errorf("failed to join path, err: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot create request, err: %w", err)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot do request, err: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("aws s3 request failed with status: %s: could not read response body: %w", resp.Status, err)
		}
		bodyStr := string(body)
		return nil, fmt.Errorf("aws s3 request failed, status: %s, body: %s", resp.Status, bodyStr)
	}

	var blob kzg4844.Blob
	buf := blob[:]
	if n, err := io.ReadFull(resp.Body, buf); err != nil {
		if err == io.ErrUnexpectedEOF || err == io.EOF {
			return nil, fmt.Errorf("blob data too short: got %d bytes", n)
		}
		return nil, fmt.Errorf("failed to read blob data: %w", err)
	}

	// sanity check that retrieved blob matches versioned hash
	commitment, err := kzg4844.BlobToCommitment(&blob)
	if err != nil {
		return nil, fmt.Errorf("failed to convert blob to commitment, err: %w", err)
	}

	blobVersionedHash := kzg4844.CalcBlobHashV1(sha256.New(), &commitment)
	if blobVersionedHash != versionedHash {
		return nil, fmt.Errorf("blob versioned hash mismatch, expected: %s, got: %s", versionedHash.String(), hexutil.Encode(blobVersionedHash[:]))
	}

	return &blob, nil
}
