// Copyright 2023 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package beaclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	eth2client "github.com/attestantio/go-eth2-client"
	eth2api "github.com/attestantio/go-eth2-client/api"
	eth2http "github.com/attestantio/go-eth2-client/http"
	eth2spec "github.com/attestantio/go-eth2-client/spec"
	"github.com/ethereum/go-ethereum/beacon/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog"
)

var (
	beaconLightClientBootstrap        = "eth/v1/beacon/light_client/bootstrap"
	beaconLightClientUpdate           = "eth/v1/beacon/light_client/updates"
	beaconLightClientOptimisticUpdate = "eth/v1/beacon/light_client/optimistic_update"
	beaconLightClientFinalityUpdate   = "eth/v1/beacon/light_client/finality_update"
)

// Client is a wrapper around the attestantio/go-eth2-client beacon api client.
type Client struct {
	ctx          context.Context
	url          string
	http         *http.Client
	extraHeaders map[string]string
	client       eth2client.Service
}

// NewClient creates a Client for the given server URL.
func NewClient(ctx context.Context, server string, headers []string) (*Client, error) {
	// Parse additional headers.
	extraHeaders := make(map[string]string)
	for _, h := range headers {
		s := strings.Split(h, ":")
		if len(s) != 2 {
			return nil, fmt.Errorf("malformed extra header: %s", h)
		}
		extraHeaders[s[0]] = s[1]
	}
	client, err := eth2http.New(
		ctx,
		eth2http.WithAddress(server),
		eth2http.WithLogLevel(zerolog.WarnLevel),
		eth2http.WithEnforceJSON(true),
		eth2http.WithExtraHeaders(extraHeaders),
	)
	if err != nil {
		return nil, err
	}
	return &Client{
		ctx:          ctx,
		url:          server,
		http:         &http.Client{},
		extraHeaders: extraHeaders,
		client:       client,
	}, nil
}

// Bootstrap retrieves a bootstrap object associated with the given root from
// the beacon server.
func (c *Client) Bootstrap(root common.Hash) (*types.Bootstrap, error) {
	var bs types.Bootstrap
	if err := c.fetch(fmt.Sprintf("%s/%s/%s", c.url, beaconLightClientBootstrap, root.String()), &bs); err != nil {
		return nil, err
	}
	return &bs, nil
}

// GetRangeUpdate retrieves a range update for the desired periods. The updates
// include the next sync committee and finalized header from the period.
func (c *Client) GetRangeUpdate(start, count int) ([]*types.LightClientUpdate, error) {
	var u []*types.LightClientUpdate
	if err := c.fetch(fmt.Sprintf("%s/%s?start_period=%d&count=%d", c.url, beaconLightClientUpdate, start, count), &u); err != nil {
		return nil, err
	}
	return u, nil
}

// GetOptimisticUpdate retrieves the latest available optimistic update from the
// beacon api server.
func (c *Client) GetOptimisticUpdate() (*types.LightClientUpdate, error) {
	var u types.LightClientUpdate
	if err := c.fetch(fmt.Sprintf("%s/%s", c.url, beaconLightClientOptimisticUpdate), &u); err != nil {
		return nil, err
	}
	return &u, nil
}

// GetFinalityUpdate retrieves the latest available finality update from the
// beacon api server.
func (c *Client) GetFinalityUpdate() (*types.LightClientUpdate, error) {
	var u types.LightClientUpdate
	if err := c.fetch(fmt.Sprintf("%s/%s", c.url, beaconLightClientFinalityUpdate), &u); err != nil {
		return nil, err
	}
	return &u, nil
}

// GetBlock retrieves the full beacon block associated with the given root.
func (c *Client) GetBlock(root common.Hash) (*eth2spec.VersionedSignedBeaconBlock, error) {
	provider, ok := c.client.(eth2client.SignedBeaconBlockProvider)
	if !ok {
		return nil, fmt.Errorf("beacon server does not support retrieving blocks")
	}
	resp, err := provider.SignedBeaconBlock(c.ctx, &eth2api.SignedBeaconBlockOpts{Block: root.String()})
	if err != nil {
		return nil, fmt.Errorf("failed http request: %w", err)
	}
	return resp.Data, nil
}

func (c *Client) fetch(url string, val any) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	for k, v := range c.extraHeaders {
		req.Header.Set(k, v)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("failed http request: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed http request: status %d", resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}
	if err := json.Unmarshal(b, val); err != nil {
		return fmt.Errorf("failed to unmarshal data: %v", err)
	}
	return nil
}
