package blob_client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto/kzg4844"
)

const (
	BeaconNodeDefaultTimeout = 15 * time.Second
)

type BeaconNodeClient struct {
	client         *http.Client
	apiEndpoint    string
	genesisTime    uint64
	secondsPerSlot uint64
}

var (
	beaconNodeGenesisEndpoint = "/eth/v1/beacon/genesis"
	beaconNodeSpecEndpoint    = "/eth/v1/config/spec"
	beaconNodeBlobEndpoint    = "/eth/v1/beacon/blobs"
)

func NewBeaconNodeClient(apiEndpoint string) (*BeaconNodeClient, error) {
	client := &http.Client{Timeout: BeaconNodeDefaultTimeout}

	// get genesis time
	genesisPath, err := url.JoinPath(apiEndpoint, beaconNodeGenesisEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to join path, err: %w", err)
	}
	resp, err := client.Get(genesisPath)
	if err != nil {
		return nil, fmt.Errorf("cannot do request, err: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("beacon node request failed with status: %s: could not read response body: %w", resp.Status, err)
		}
		bodyStr := string(body)
		return nil, fmt.Errorf("beacon node request failed, status: %s, body: %s", resp.Status, bodyStr)
	}

	var genesisResp GenesisResp
	err = json.NewDecoder(resp.Body).Decode(&genesisResp)
	if err != nil {
		return nil, fmt.Errorf("failed to decode result into struct, err: %w", err)
	}
	genesisTime, err := strconv.ParseUint(genesisResp.Data.GenesisTime, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode genesis time %s, err: %w", genesisResp.Data.GenesisTime, err)
	}

	// get seconds per slot from spec
	specPath, err := url.JoinPath(apiEndpoint, beaconNodeSpecEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to join path, err: %w", err)
	}
	resp, err = client.Get(specPath)
	if err != nil {
		return nil, fmt.Errorf("cannot do request, err: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("beacon node request failed with status: %s: could not read response body: %w", resp.Status, err)
		}
		bodyStr := string(body)
		return nil, fmt.Errorf("beacon node request failed, status: %s, body: %s", resp.Status, bodyStr)
	}

	var specResp SpecResp
	err = json.NewDecoder(resp.Body).Decode(&specResp)
	if err != nil {
		return nil, fmt.Errorf("failed to decode result into struct, err: %w", err)
	}
	secondsPerSlot, err := strconv.ParseUint(specResp.Data.SecondsPerSlot, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode seconds per slot %s, err: %w", specResp.Data.SecondsPerSlot, err)
	}
	if secondsPerSlot == 0 {
		return nil, fmt.Errorf("failed to make new BeaconNodeClient, secondsPerSlot is 0")
	}

	return &BeaconNodeClient{
		client:         client,
		apiEndpoint:    apiEndpoint,
		genesisTime:    genesisTime,
		secondsPerSlot: secondsPerSlot,
	}, nil
}

func (c *BeaconNodeClient) getBlobsPath(slot uint64, versionedHash common.Hash) (string, error) {
	basePath, err := url.JoinPath(c.apiEndpoint, beaconNodeBlobEndpoint, fmt.Sprintf("%d", slot))
	if err != nil {
		return "", fmt.Errorf("failed to join path, err: %w", err)
	}
	u, err := url.Parse(basePath)
	if err != nil {
		return "", fmt.Errorf("failed to parse path, err: %w", err)
	}
	q := u.Query()
	q.Set("versioned_hashes", versionedHash.Hex())
	u.RawQuery = q.Encode()
	queryPath := u.String()
	return queryPath, nil
}

func (c *BeaconNodeClient) GetBlobByVersionedHashAndBlockTime(ctx context.Context, versionedHash common.Hash, blockTime uint64) (*kzg4844.Blob, error) {
	slot := (blockTime - c.genesisTime) / c.secondsPerSlot

	// get blob by slot and versioned hash
	getBlobsPath, err := c.getBlobsPath(slot, versionedHash)
	if err != nil {
		return nil, fmt.Errorf("failed to create getBlobs path, err: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "GET", getBlobsPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request, err: %w", err)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot do request, err: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("beacon node request failed with status: %s: could not read response body: %w", resp.Status, err)
		}
		bodyStr := string(body)
		return nil, fmt.Errorf("beacon node request failed, status: %s, body: %s", resp.Status, bodyStr)
	}

	var blobsResp BlobsResp
	err = json.NewDecoder(resp.Body).Decode(&blobsResp)
	if err != nil {
		return nil, fmt.Errorf("failed to decode result into struct, err: %w", err)
	}

	// sanity check response length
	if len(blobsResp.Data) == 0 {
		return nil, fmt.Errorf("missing blob %v in slot %d", versionedHash, slot)
	}
	if len(blobsResp.Data) > 1 {
		return nil, fmt.Errorf("more than 1 blob returned from beacon node for slot %d, requested blob hash: %s, expected 1, got: %d", slot, versionedHash.Hex(), len(blobsResp.Data))
	}

	blobBytes := common.FromHex(blobsResp.Data[0])
	if len(blobBytes) != lenBlobBytes {
		return nil, fmt.Errorf("len of blob data is not correct, expected: %d, got: %d", lenBlobBytes, len(blobBytes))
	}

	b := kzg4844.Blob(blobBytes)
	return &b, nil
}

type GenesisResp struct {
	Data struct {
		GenesisTime string `json:"genesis_time"`
	} `json:"data"`
}

type SpecResp struct {
	Data struct {
		SecondsPerSlot string `json:"SECONDS_PER_SLOT"`
	} `json:"data"`
}

type BlobsResp struct {
	Data []string `json:"data"` // array of blobs as hex strings
}
