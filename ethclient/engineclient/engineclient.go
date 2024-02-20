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
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Package engineclient provides an RPC client for engine API required functions.
package engineclient

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

// Client is a wrapper around rpc.Client that implements geth-specific functionality.
//
// If you want to use the standardized Ethereum RPC functionality, use ethclient.Client instead.
type Client struct {
	c *rpc.Client
}

// New creates a client that uses the given RPC client.
func New(c *rpc.Client) *Client {
	return &Client{c}
}

// PayloadIDBytes defines a custom type for Payload IDs used by the engine API
// client with proper JSON Marshal and Unmarshal methods to hex.
type PayloadIDBytes [8]byte

// MarshalJSON --
func (b PayloadIDBytes) MarshalJSON() ([]byte, error) {
	return json.Marshal(hexutil.Bytes(b[:]))
}

// ForkchoiceUpdatedResponse is the response kind received by the
// engine_forkchoiceUpdatedV1 endpoint.
type ForkchoiceUpdatedResponse struct {
	Status          *engine.PayloadStatusV1 `json:"payloadStatus"`
	PayloadId       *PayloadIDBytes         `json:"payloadId"`
	ValidationError string                  `json:"validationError"`
}

// NewPayloadV3 calls the engine_newPayloadV3 method via JSON-RPC.
func (s *Client) NewPayloadV3(
	ctx context.Context, payload *engine.ExecutionPayloadEnvelope,
	versionedHashes []common.Hash, parentBlockRoot *common.Hash,
) (*engine.PayloadStatusV1, error) {
	return s.newPayload(ctx, "engine_newPayloadV3", payload, versionedHashes, parentBlockRoot)
}

// NewPayloadV2 calls the engine_newPayloadV2 method via JSON-RPC.
func (s *Client) NewPayloadV2(
	ctx context.Context, payload *engine.ExecutionPayloadEnvelope,
) (*engine.PayloadStatusV1, error) {
	return s.newPayload(ctx, "engine_newPayloadV2", payload)
}

// NewPayloadV1 calls the engine_newPayloadV1 method via JSON-RPC.
func (s *Client) NewPayloadV1(
	ctx context.Context, payload *engine.ExecutionPayloadEnvelope,
) (*engine.PayloadStatusV1, error) {
	return s.newPayload(ctx, "engine_newPayloadV1", payload)
}

func (s *Client) newPayload(
	ctx context.Context, method string, payload *engine.ExecutionPayloadEnvelope, args ...any,
) (*engine.PayloadStatusV1, error) {
	result := &engine.PayloadStatusV1{}
	if err := s.c.CallContext(
		ctx, result, method, payload, args,
	); err != nil {
		return nil, err
	}
	return result, nil
}

// ForkchoiceUpdatedV3 calls the engine_forkchoiceUpdatedV3 method via JSON-RPC.
func (s *Client) ForkchoiceUpdatedV3(
	ctx context.Context, state *engine.ForkchoiceStateV1, attrs *engine.PayloadAttributes,
) (*ForkchoiceUpdatedResponse, error) {
	return s.forkchoiceUpdated(ctx, "engine_forkchoiceUpdatedV3", state, attrs)
}

// ForkchoiceUpdatedV2 calls the engine_forkchoiceUpdatedV2 method via JSON-RPC.
func (s *Client) ForkchoiceUpdatedV2(
	ctx context.Context, state *engine.ForkchoiceStateV1, attrs *engine.PayloadAttributes,
) (*ForkchoiceUpdatedResponse, error) {
	return s.forkchoiceUpdated(ctx, "engine_forkchoiceUpdatedV2", state, attrs)
}

// ForkchoiceUpdatedV1 calls the engine_forkchoiceUpdatedV1 method via JSON-RPC.
func (s *Client) ForkchoiceUpdatedV1(
	ctx context.Context, state *engine.ForkchoiceStateV1, attrs *engine.PayloadAttributes,
) (*ForkchoiceUpdatedResponse, error) {
	return s.forkchoiceUpdated(ctx, "engine_forkchoiceUpdatedV1", state, attrs)
}

// forkchoiceUpdateCall is a helper function to call to any version of the forkchoiceUpdated
// method.
func (s *Client) forkchoiceUpdated(
	ctx context.Context, method string, state *engine.ForkchoiceStateV1, attrs any,
) (*ForkchoiceUpdatedResponse, error) {
	result := &ForkchoiceUpdatedResponse{}

	if err := s.c.CallContext(
		ctx, result, method, state, attrs,
	); err != nil {
		return nil, err
	}

	if result.Status == nil {
		return nil, errors.New("got nil status in" + method)
	} else if result.ValidationError != "" {
		return nil, errors.New(result.ValidationError)
	}

	return result, nil
}

// GetPayloadV3 calls the engine_getPayloadV3 method via JSON-RPC.
func (s *Client) GetPayloadV3(
	ctx context.Context, payloadID PayloadIDBytes,
) (*engine.ExecutionPayloadEnvelope, error) {
	return s.getPayload(ctx, "engine_getPayloadV3", payloadID)
}

// GetPayloadV2 calls the engine_getPayloadV3 method via JSON-RPC.
func (s *Client) GetPayloadV2(
	ctx context.Context, payloadID PayloadIDBytes,
) (*engine.ExecutionPayloadEnvelope, error) {
	return s.getPayload(ctx, "engine_getPayloadV2", payloadID)
}

// GetPayloadV3 calls the engine_getPayloadV3 method via JSON-RPC.
func (s *Client) GetPayloadV1(
	ctx context.Context, payloadID PayloadIDBytes,
) (*engine.ExecutionPayloadEnvelope, error) {
	return s.getPayload(ctx, "engine_getPayloadV1", payloadID)
}

func (s *Client) getPayload(ctx context.Context, method string, payloadID PayloadIDBytes) (*engine.ExecutionPayloadEnvelope, error) {
	result := &engine.ExecutionPayloadEnvelope{}
	if err := s.c.CallContext(
		ctx, result, method, payloadID,
	); err != nil {
		return nil, err
	}
	return result, nil
}
