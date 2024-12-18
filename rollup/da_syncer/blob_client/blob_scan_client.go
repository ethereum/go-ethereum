package blob_client

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/crypto/kzg4844"
)

type BlobScanClient struct {
	client      *http.Client
	apiEndpoint string
}

func NewBlobScanClient(apiEndpoint string) *BlobScanClient {
	return &BlobScanClient{
		client:      http.DefaultClient,
		apiEndpoint: apiEndpoint,
	}
}

func (c *BlobScanClient) GetBlobByVersionedHashAndBlockNumber(ctx context.Context, versionedHash common.Hash, blockNumber uint64) (*kzg4844.Blob, error) {
	// blobscan api docs https://api.blobscan.com/#/blobs/blob-getByBlobId
	path, err := url.JoinPath(c.apiEndpoint, versionedHash.String())
	if err != nil {
		return nil, fmt.Errorf("failed to join path, err: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot create request, err: %w", err)
	}
	req.Header.Set("accept", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot do request, err: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("no blob with versioned hash : %s", versionedHash.String())
		}
		var res ErrorRespBlobScan
		err = json.NewDecoder(resp.Body).Decode(&res)
		if err != nil {
			return nil, fmt.Errorf("failed to decode result into struct, err: %w", err)
		}
		return nil, fmt.Errorf("error while fetching blob, message: %s, code: %s, versioned hash: %s", res.Message, res.Code, versionedHash.String())
	}
	var result BlobRespBlobScan

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("failed to decode result into struct, err: %w", err)
	}
	blobBytes, err := hex.DecodeString(result.Data[2:])
	if err != nil {
		return nil, fmt.Errorf("failed to decode data to bytes, err: %w", err)
	}
	if len(blobBytes) != lenBlobBytes {
		return nil, fmt.Errorf("len of blob data is not correct, expected: %d, got: %d", lenBlobBytes, len(blobBytes))
	}
	blob := kzg4844.Blob(blobBytes)

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

type BlobRespBlobScan struct {
	Data string `json:"data"`
}

type ErrorRespBlobScan struct {
	Message string `json:"message"`
	Code    string `json:"code"`
}
