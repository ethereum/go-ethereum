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

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	eth2client "github.com/attestantio/go-eth2-client"
	eth2api "github.com/attestantio/go-eth2-client/api"
	eth2http "github.com/attestantio/go-eth2-client/http"
	eth2spec "github.com/attestantio/go-eth2-client/spec"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog"
)

var (
	beaconLightClientBootstrap        = "eth/v1/beacon/light_client/bootstrap"
	beaconLightClientUpdate           = "eth/v1/beacon/light_client/updates"
	beaconLightClientOptimisticUpdate = "eth/v1/beacon/light_client/optimistic_update"
	beaconLightClientFinalityUpdate   = "eth/v1/beacon/light_client/finality_update"
)

type BeaconClient struct {
	ctx    context.Context
	url    string
	client eth2client.Service
}

func NewBeaconClient(ctx context.Context, server string) (*BeaconClient, error) {
	client, err := eth2http.New(
		ctx,
		eth2http.WithAddress(server),
		eth2http.WithLogLevel(zerolog.WarnLevel),
		eth2http.WithEnforceJSON(true),
	)
	if err != nil {
		return nil, err
	}
	return &BeaconClient{
		ctx:    ctx,
		url:    server,
		client: client,
	}, nil
}

func (c *BeaconClient) Bootstrap(root common.Hash) (*Bootstrap, error) {
	resp, err := http.Get(fmt.Sprintf("%s/%s/%s", c.url, beaconLightClientBootstrap, root.String()))
	if err != nil {
		return nil, fmt.Errorf("failed http request: %w", err)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	var bs Bootstrap
	if err := json.Unmarshal(b, &bs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal data: %v", err)
	}
	return &bs, nil
}

func (c *BeaconClient) GetRangeUpdate(start, count int) ([]*LightClientUpdate, error) {
	resp, err := http.Get(fmt.Sprintf("%s/%s?start_period=%d&count=%d", c.url, beaconLightClientUpdate, start, count))
	if err != nil {
		return nil, fmt.Errorf("failed http request: %w", err)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	var u []*LightClientUpdate
	if err := json.Unmarshal(b, &u); err != nil {
		return nil, fmt.Errorf("failed to unmarshal data: %v", err)
	}
	return u, nil

}

func (c *BeaconClient) GetOptimisticUpdate() (*LightClientUpdate, error) {
	resp, err := http.Get(fmt.Sprintf("%s/%s", c.url, beaconLightClientOptimisticUpdate))
	if err != nil {
		return nil, fmt.Errorf("failed http request: %w", err)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	var u LightClientUpdate
	if err := json.Unmarshal(b, &u); err != nil {
		return nil, fmt.Errorf("failed to unmarshal data: %v", err)
	}
	return &u, nil
}

func (c *BeaconClient) GetFinalityUpdate() (*LightClientUpdate, error) {
	resp, err := http.Get(fmt.Sprintf("%s/%s", c.url, beaconLightClientFinalityUpdate))
	if err != nil {
		return nil, fmt.Errorf("failed http request: %w", err)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	var u LightClientUpdate
	if err := json.Unmarshal(b, &u); err != nil {
		return nil, fmt.Errorf("failed to unmarshal data: %v", err)
	}
	return &u, nil

}

func (c *BeaconClient) GetBlock(root common.Hash) (*eth2spec.VersionedSignedBeaconBlock, error) {
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
