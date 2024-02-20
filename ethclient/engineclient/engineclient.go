// Copyright 2021 The go-ethereum Authors
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
// GNU Lesser General Public License for more detailc.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Package engineclient provides an RPC client for engine API required functionc.
package engineclient

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Client is a wrapper around rpc.Client that implements geth-specific functionality.
//
// If you want to use the standardized Ethereum RPC functionality, use ethclient.Client instead.
type Client struct {
	*ethclient.Client
}

// New creates a client that uses the given RPC client.
func New(c *ethclient.Client) *Client {
	return &Client{c}
}

// ExchangeTransitionConfigurationV1 calls the engine_exchangeTransitionConfigurationV1
// method via JSON-RPC. This is not really needed anymore, since we are post merge,
// but it is still here for reference / completeness sake.
func (c *Client) ExchangeTransitionConfigurationV1(
	ctx context.Context,
	config engine.TransitionConfigurationV1,
) (*engine.TransitionConfigurationV1, error) {
	result := &engine.TransitionConfigurationV1{}
	if err := c.Client.Client().CallContext(
		ctx, result, "engine_exchangeTransitionConfigurationV1", config,
	); err != nil {
		return nil, err
	}
	return result, nil
}

// ExchangeCapabilities calls the engine_exchangeCapabilities method via JSON-RPC.
func (c *Client) ExchangeCapabilities(
	ctx context.Context,
	capabilities []string,
) ([]string, error) {
	result := make([]string, 0)
	if err := c.Client.Client().CallContext(
		ctx, &result, "engine_exchangeCapabilities", &capabilities,
	); err != nil {
		return nil, err
	}
	return result, nil
}

// GetClientVersionV1 calls the engine_getClientVersionV1 method via JSON-RPC.
func (c *Client) GetClientVersionV1(ctx context.Context) ([]engine.ClientVersionV1, error) {
	result := make([]engine.ClientVersionV1, 0)
	if err := c.Client.Client().CallContext(
		ctx, &result, "engine_getClientVersionV1", nil,
	); err != nil {
		return nil, err
	}
	return result, nil
}

// NewPayloadV3 calls the engine_newPayloadV3 method via JSON-RPC.
func (c *Client) NewPayloadV3(
	ctx context.Context, payload *engine.ExecutionPayloadEnvelope,
	versionedHashes common.Hash, parentBlockRoot common.Hash,
) (*engine.PayloadStatusV1, error) {
	return c.newPayloadWithArgs(ctx, CancunV3, payload, versionedHashes, parentBlockRoot)
}

// NewPayloadV2 calls the engine_newPayloadV2 method via JSON-RPC.
func (c *Client) NewPayloadV2(
	ctx context.Context, payload *engine.ExecutionPayloadEnvelope,
) (*engine.PayloadStatusV1, error) {
	return c.newPayload(ctx, ShanghaiV2, payload)
}

// NewPayloadV1 calls the engine_newPayloadV1 method via JSON-RPC.
func (c *Client) NewPayloadV1(
	ctx context.Context, payload *engine.ExecutionPayloadEnvelope,
) (*engine.PayloadStatusV1, error) {
	return c.newPayload(ctx, ParisV1, payload)
}

// newPayload is a helper function that can call an arbitrary version of the newPayload method.
func (c *Client) newPayload(
	ctx context.Context, version APIVersion, payload *engine.ExecutionPayloadEnvelope,
) (*engine.PayloadStatusV1, error) {
	result := &engine.PayloadStatusV1{}
	if err := c.Client.Client().CallContext(
		ctx, &result, fmt.Sprintf("engine_newPayloadV%d", version), payload.ExecutionPayload,
	); err != nil {
		return nil, err
	}
	return result, nil
}

// newPayloadWithArgs is a helper function that can call an arbitrary version of the newPayload method.
func (c *Client) newPayloadWithArgs(
	ctx context.Context, version APIVersion, payload *engine.ExecutionPayloadEnvelope, args ...any,
) (*engine.PayloadStatusV1, error) {
	result := &engine.PayloadStatusV1{}
	if err := c.Client.Client().CallContext(
		ctx, &result, fmt.Sprintf("engine_newPayloadV%d", version), payload.ExecutionPayload, args,
	); err != nil {
		return nil, err
	}
	return result, nil
}

// ForkchoiceUpdatedV1 calls the engine_forkchoiceUpdatedV1 method via JSON-RPC.
func (c *Client) ForkchoiceUpdatedV1(
	ctx context.Context, state *engine.ForkchoiceStateV1, attrs *engine.PayloadAttributes,
) (*ForkchoiceUpdatedResponse, error) {
	return c.forkchoiceUpdated(ctx, ParisV1, state, attrs)
}

// ForkchoiceUpdatedV2 calls the engine_forkchoiceUpdatedV2 method via JSON-RPC.
func (c *Client) ForkchoiceUpdatedV2(
	ctx context.Context, state *engine.ForkchoiceStateV1, attrs *engine.PayloadAttributes,
) (*ForkchoiceUpdatedResponse, error) {
	return c.forkchoiceUpdated(ctx, ShanghaiV2, state, attrs)
}

// ForkchoiceUpdatedV3 calls the engine_forkchoiceUpdatedV3 method via JSON-RPC.
func (c *Client) ForkchoiceUpdatedV3(
	ctx context.Context, state *engine.ForkchoiceStateV1, attrs *engine.PayloadAttributes,
) (*ForkchoiceUpdatedResponse, error) {
	return c.forkchoiceUpdated(ctx, CancunV3, state, attrs)
}

// forkchoiceUpdateCall is a helper function to call to any version of the forkchoiceUpdated
// method.
func (c *Client) forkchoiceUpdated(
	ctx context.Context, version APIVersion, state *engine.ForkchoiceStateV1, attrs any,
) (*ForkchoiceUpdatedResponse, error) {
	result := &ForkchoiceUpdatedResponse{}

	if err := c.Client.Client().CallContext(
		ctx, result, fmt.Sprintf("engine_forkchoiceUpdatedV%d", version), state, attrs,
	); err != nil {
		return nil, err
	}

	if result.Status == nil {
		return nil, fmt.Errorf("got nil status in engine_forkchoiceUpdatedV%d", version)
	} else if result.ValidationError != "" {
		return nil, errors.New(result.ValidationError)
	}

	return result, nil
}

// GetPayloadV3 calls the engine_getPayloadV3 method via JSON-RPC.
func (c *Client) GetPayloadV1(
	ctx context.Context, payloadID *engine.PayloadID,
) (*engine.ExecutionPayloadEnvelope, error) {
	return c.getPayload(ctx, ParisV1, payloadID)
}

// GetPayloadV2 calls the engine_getPayloadV3 method via JSON-RPC.
func (c *Client) GetPayloadV2(
	ctx context.Context, payloadID *engine.PayloadID,
) (*engine.ExecutionPayloadEnvelope, error) {
	return c.getPayload(ctx, ShanghaiV2, payloadID)
}

// GetPayloadV3 calls the engine_getPayloadV3 method via JSON-RPC.
func (c *Client) GetPayloadV3(
	ctx context.Context, payloadID *engine.PayloadID,
) (*engine.ExecutionPayloadEnvelope, error) {
	return c.getPayload(ctx, CancunV3, payloadID)
}

// getPayload is a helper function that can call an arbitrary version of the getPayload method.
func (c *Client) getPayload(ctx context.Context, version APIVersion, payloadID *engine.PayloadID) (*engine.ExecutionPayloadEnvelope, error) {
	result := &engine.ExecutionPayloadEnvelope{}
	if err := c.Client.Client().CallContext(
		ctx, result, fmt.Sprintf("engine_getPayloadV%d", version), payloadID,
	); err != nil {
		return nil, err
	}
	return result, nil
}

// GetPayloadBodiesByHashV1 calls the engine_getPayloadBodiesByHashV1 method via JSON-RPC.
func (c *Client) GetPayloadBodiesByHashV1(
	ctx context.Context,
	hashes []common.Hash,
) ([]*engine.ExecutionPayloadBodyV1, error) {
	result := make([]*engine.ExecutionPayloadBodyV1, 0)
	if err := c.Client.Client().CallContext(
		ctx, &result, "engine_getPayloadBodiesByHashV1", &hashes,
	); err != nil {
		return nil, err
	}
	return result, nil
}

// GetPayloadBodiesByRangeV1 calls the engine_getPayloadBodiesByRangeV1 method via JSON-RPC.
func (c *Client) GetPayloadBodiesByRangeV1(
	ctx context.Context, start, count hexutil.Uint64,
) ([]*engine.ExecutionPayloadBodyV1, error) {
	result := make([]*engine.ExecutionPayloadBodyV1, 0)
	if err := c.Client.Client().CallContext(
		ctx, &result, "engine_getPayloadBodiesByRangeV1", start, count,
	); err != nil {
		return nil, err
	}
	return result, nil
}
