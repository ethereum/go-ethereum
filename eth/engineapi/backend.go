// Copyright 2026 The go-ethereum Authors
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

// Package engineapi implements the REST + SSZ Engine API (execution-apis #793).
//
// The package is deliberately decoupled from eth/catalyst: the JSON-RPC engine
// surface and this REST surface share the same backing logic through the
// Backend interface defined here, but neither imports the other. The catalyst
// package supplies a Backend impl at node-startup time.
package engineapi

import (
	"context"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params/forks"
)

// Backend is the contract a host node implements to drive the REST handlers.
type Backend interface {
	// ForkchoiceUpdated applies a forkchoice update and optionally starts
	// a payload build. version selects the JSON-RPC PayloadVersion the
	// catalyst-layer expects.
	ForkchoiceUpdated(ctx context.Context, state engine.ForkchoiceStateV1, attrs *engine.PayloadAttributes, version engine.PayloadVersion) (engine.ForkChoiceResponse, error)

	// NewPayload validates and imports a new payload.
	NewPayload(ctx context.Context, data engine.ExecutableData, versionedHashes []common.Hash, beaconRoot *common.Hash, requests [][]byte) (engine.PayloadStatusV1, error)

	// GetPayload returns a previously-built payload. allowedForks filters
	// to the fork the Eth-Execution-Version header selected; an empty slice
	// disables the check.
	GetPayload(id engine.PayloadID, allowedForks []forks.Fork) (*engine.ExecutionPayloadEnvelope, error)

	// GetBlobs returns blob bundle entries indexed by versioned hash.
	// cellProofs=false matches /blobs/v1 semantics (single proof per blob);
	// true matches /blobs/v2,v3 (cell proofs).
	GetBlobs(hashes []common.Hash, cellProofs bool) ([]*engine.BlobAndProofV2, []*engine.BlobAndProofV1, error)

	// BodiesByHash returns block bodies for the given hashes. Order matches
	// the input; an entry is nil for unknown/pruned blocks. The second slice
	// returns the timestamp of each known block (zero for unknown) so the
	// router can enforce the header fork's era window.
	BodiesByHash(hashes []common.Hash) ([]*types.Body, []uint64)

	// BodiesByRange returns block bodies for [from, from+count), but the
	// result MUST be truncated at the latest known block: blocks past head are
	// omitted, not padded with nil. The spec's "no trailing nulls" rule means a
	// range response has length min(count, head-from+1) for from <= head, and
	// is empty for from > head. In-range-but-pruned blocks are still returned as
	// nil entries (-> available=false); only past-head blocks are dropped.
	BodiesByRange(from, count uint64) ([]*types.Body, []uint64)

	// ForkFromTimestamp returns the fork active at timestamp ts.
	ForkFromTimestamp(ts uint64) forks.Fork

	// ClientVersion returns the EL client identity for /identity responses.
	ClientVersion() engine.ClientVersionV1
}
