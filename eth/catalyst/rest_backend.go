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

package catalyst

import (
	"context"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/engineapi"
	"github.com/ethereum/go-ethereum/internal/version"
	"github.com/ethereum/go-ethereum/params/forks"
)

// restBackend adapts the JSON-RPC ConsensusAPI to the engineapi.Backend
// contract that drives the REST + SSZ Engine API (execution-apis #793). Both
// surfaces share the same underlying logic; this adapter only translates
// shapes, it holds no state of its own.
type restBackend struct {
	api *ConsensusAPI
}

// newRESTBackend wraps a ConsensusAPI as an engineapi.Backend.
func newRESTBackend(api *ConsensusAPI) engineapi.Backend {
	return &restBackend{api: api}
}

func (b *restBackend) ForkchoiceUpdated(ctx context.Context, state engine.ForkchoiceStateV1, attrs *engine.PayloadAttributes, version engine.PayloadVersion) (engine.ForkChoiceResponse, error) {
	return b.api.forkchoiceUpdated(ctx, state, attrs, version, false)
}

func (b *restBackend) NewPayload(ctx context.Context, data engine.ExecutableData, versionedHashes []common.Hash, beaconRoot *common.Hash, requests [][]byte) (engine.PayloadStatusV1, error) {
	return b.api.newPayload(ctx, data, versionedHashes, beaconRoot, requests, false)
}

func (b *restBackend) GetPayload(id engine.PayloadID, allowedForks []forks.Fork) (*engine.ExecutionPayloadEnvelope, error) {
	// full=true so the envelope carries the complete payload; versions=nil
	// because the REST router selects the fork from the Eth-Execution-Version
	// header, not the payload id's embedded version. allowedForks enforces the
	// header fork's era.
	return b.api.getPayload(id, true, nil, allowedForks)
}

func (b *restBackend) GetBlobs(hashes []common.Hash, cellProofs bool) ([]*engine.BlobAndProofV2, []*engine.BlobAndProofV1, error) {
	if cellProofs {
		v2, err := b.api.getBlobs(hashes, true)
		return v2, nil, err
	}
	v1, err := b.api.GetBlobsV1(hashes)
	return nil, v1, err
}

func (b *restBackend) BodiesByHash(hashes []common.Hash) ([]*types.Body, []uint64) {
	chain := b.api.eth.BlockChain()
	bodies := make([]*types.Body, len(hashes))
	timestamps := make([]uint64, len(hashes))
	for i, h := range hashes {
		block := chain.GetBlockByHash(h)
		if block == nil {
			continue
		}
		bodies[i] = block.Body()
		timestamps[i] = block.Time()
	}
	return bodies, timestamps
}

func (b *restBackend) BodiesByRange(from, count uint64) ([]*types.Body, []uint64) {
	chain := b.api.eth.BlockChain()
	// Truncate at head: blocks past the latest known block are omitted, never
	// padded with nil (the spec's "no trailing nulls" rule for range queries).
	head := chain.CurrentBlock().Number.Uint64()
	if from > head {
		return nil, nil
	}
	last := min(from+count-1, head)
	n := last - from + 1
	bodies := make([]*types.Body, 0, n)
	timestamps := make([]uint64, 0, n)
	for num := from; num <= last; num++ {
		block := chain.GetBlockByNumber(num)
		if block == nil {
			// In-range but pruned: keep the slot so the entry reports
			// available=false; only past-head blocks are dropped (above).
			bodies = append(bodies, nil)
			timestamps = append(timestamps, 0)
			continue
		}
		bodies = append(bodies, block.Body())
		timestamps = append(timestamps, block.Time())
	}
	return bodies, timestamps
}

func (b *restBackend) ForkFromTimestamp(ts uint64) forks.Fork {
	return b.api.config().LatestFork(ts)
}

func (b *restBackend) ClientVersion() engine.ClientVersionV1 {
	commit := make([]byte, 4)
	if vcs, ok := version.VCS(); ok {
		commit = common.FromHex(vcs.Commit)[0:4]
	}
	return engine.ClientVersionV1{
		Code:    engine.ClientCode,
		Name:    engine.ClientName,
		Version: version.WithMeta,
		Commit:  hexutil.Encode(commit),
	}
}
