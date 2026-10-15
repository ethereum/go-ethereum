// Copyright 2026 The quarkchain Authors
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

package params

import (
	"fmt"
	"slices"

	"github.com/ethereum/go-ethereum/common"
)

// XShardOutboxAddress is where the cross-shard outbox precompile lives on
// QuarkChain shard chains.
// "0x000000000000000000000000000000514b43000X" is reserved for legacy quarkchain system contracts
var XShardOutboxAddress = common.HexToAddress("0x000000000000000000000000000000514b430100")

const (
	// XShardSendGas is the fixed cost of a cross-shard message. The destination
	// shard charges it with the calldata cost of the message out of the message's
	// gas limit, so the limit must cover both.
	XShardSendGas uint64 = 9000

	// MaxCrossShardTxGas caps a message's destination gas budget, including
	// XShardSendGas. It matches half of QuarkChain's default 12,000,000 block gas limit.
	MaxCrossShardTxGas uint64 = 6_000_000
)

// QuarkChainConfig is the configuration of a QuarkChain shard chain. Its
// presence in the chain config enables the QuarkChain specific features.
type QuarkChainConfig struct {
	ShardID  uint32   `json:"shardId"`  // ID of this shard
	ShardIDs []uint32 `json:"shardIds"` // IDs of all shards in the network, this one included
}

// String implements the stringer interface.
func (c QuarkChainConfig) String() string {
	return fmt.Sprintf("quarkchain(shard: %d, shards: %v)", c.ShardID, c.ShardIDs)
}

// IsXShardDestination reports whether this shard can send a cross-shard
// message to the given shard.
func (c *QuarkChainConfig) IsXShardDestination(shardID uint32) bool {
	return c != nil && shardID != c.ShardID && slices.Contains(c.ShardIDs, shardID)
}
