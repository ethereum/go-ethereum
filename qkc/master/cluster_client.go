// Copyright 2026-2027 The quarkchain Authors
// This file is part of the quarkchain library.
//
// The quarkchain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The quarkchain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the quarkchain library. If not, see <http://www.gnu.org/licenses/>.

// Package master defines the cluster client used by master business logic.
package master

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	qkctypes "github.com/ethereum/go-ethereum/qkc/types"
	"github.com/holiman/uint256"
)

// ClusterClient is implemented by the component that owns the connections to
// chains. Master business logic holds this interface instead of those
// connections and does not construct wire messages.
type ClusterClient interface {
	// BroadcastRootBlock sends block to every ShardNode in the cluster and waits
	// for all acknowledgements. A send failure, a nonzero remote ErrorCode
	// (returned as *wire.RemoteError), or a ShardNode without a connection is
	// returned as an error, so a disconnected chain never silently misses the
	// root block. If only some hosts acknowledge the block, the caller must
	// retain it as uncommitted and retry the complete broadcast.
	BroadcastRootBlock(ctx context.Context, block *qkctypes.RootBlock) error

	// GetUnconfirmedHeaderList collects unconfirmed shard headers from the cluster,
	// grouped by decimal chainID strings, for constructing a root block. Each shard's headers are
	// ordered by increasing height. Master filters out headers not yet reported to
	// it, as in pyquarkchain. A transport or RPC failure, a nonzero remote ErrorCode
	// (returned as *wire.RemoteError), or a missing connection is returned as an
	// error rather than silently omitting headers.
	GetUnconfirmedHeaderList(ctx context.Context) (map[string][]*types.Header, error)

	// GetRootChainStakes asks chainID for the stakes recorded for address at
	// minorBlockHash. The result is the decoded business response after a
	// completed RPC. err reports a transport or RPC failure, or a nonzero remote
	// ErrorCode as *wire.RemoteError.
	GetRootChainStakes(ctx context.Context, chainID *big.Int, address common.Address, minorBlockHash common.Hash) (*RootChainStakesResult, error)
}

// RootChainStakesResult is the business representation of a chain's staking
// response, separate from wire.GetRootChainStakesResponse. Stakes is decoded
// from the wire uint256 representation. A nonzero remote ErrorCode is returned
// as an error instead of being stored here.
type RootChainStakesResult struct {
	Stakes *uint256.Int
	Signer common.Address
}
