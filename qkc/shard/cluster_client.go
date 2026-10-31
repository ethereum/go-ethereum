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

package shard

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
)

// ClusterClient is implemented by the component that owns the cluster
// connections. A ShardBackend holds this narrow interface instead of the connections
// or wire-message encoders.
type ClusterClient interface {
	// BroadcastXShardTxList delivers depositTxList from minorBlockHash to the
	// hosts serving chainID. An empty transaction list still advances that
	// chain's inbox cursor. A send failure, a nonzero remote ErrorCode
	// (returned as *wire.RemoteError), or no connection for chainID is returned
	// as an error. If only some hosts acknowledge the request, the caller must
	// not report the block header to Master and must retry delivery to chainID.
	BroadcastXShardTxList(ctx context.Context, chainID uint32, minorBlockHash common.Hash, depositTxList []*types.DepositTx) error

	// SendMinorBlockHeaderToMaster reports an accepted shard block to Master.
	// It returns nil after a completed RPC whose remote ErrorCode is zero. A
	// nonzero Master ErrorCode is returned as *wire.RemoteError; a transport or
	// RPC failure is returned as is.
	SendMinorBlockHeaderToMaster(ctx context.Context, blockHeader *types.Header, coinbaseAmount *uint256.Int) error
}
