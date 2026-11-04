// Copyright 2026 The quarkchain Authors
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

package miner

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// QKCBlockArgs specifies a shard candidate. The shard consensus layer selects
// Parent and PrevRootBlock and validates their ancestry before calling the miner.
// Timestamp must be later than the parent's timestamp. Difficulty and rewards
// belong to the engine; Ethereum beacon fields are fixed internally.
type QKCBlockArgs struct {
	Parent         common.Hash
	PrevRootBlock  common.Hash
	Timestamp      uint64
	FeeRecipient   common.Address
	TargetGasLimit *uint64
	NoTxs          bool // Skip pool transactions, but always include mandatory deposits.
}

// BuildQKCBlock builds one unsealed shard candidate using the configured inbox
// and transaction pool. It neither imports the block nor starts background work.
// Callers must validate root ancestry; shard.ShardChain.BuildBlock does this.
func (miner *Miner) BuildQKCBlock(ctx context.Context, args QKCBlockArgs) (*types.Block, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if miner.chainConfig.QuarkChain == nil {
		return nil, errors.New("not a QuarkChain shard")
	}
	if args.Parent == (common.Hash{}) || args.PrevRootBlock == (common.Hash{}) {
		return nil, errors.New("shard block requires parent and previous root block")
	}
	if !args.NoTxs && miner.txpool == nil {
		return nil, errors.New("transaction pool not configured")
	}
	result := miner.generateWork(ctx, &generateParams{
		parentHash:     args.Parent,
		timestamp:      args.Timestamp,
		forceTime:      true,
		coinbase:       args.FeeRecipient,
		targetGasLimit: args.TargetGasLimit,
		noTxs:          args.NoTxs,
		qkc: &types.QKCExtra{
			ShardID:       miner.chainConfig.QuarkChain.ShardID,
			PrevRootBlock: args.PrevRootBlock,
		},
	}, false)
	if result.err != nil {
		return nil, result.err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return result.block, nil
}
