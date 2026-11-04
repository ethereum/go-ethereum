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

package params

import (
	"errors"
	"fmt"
	"math/big"
	"slices"

	"github.com/ethereum/go-ethereum/common"
)

// XShardOutboxAddress is where the cross-shard outbox precompile lives on
// QuarkChain shard chains.
// "0x000000000000000000000000000000514b43000X" is reserved for legacy quarkchain system contracts
var XShardOutboxAddress = common.HexToAddress("0x000000000000000000000000000000514b430100")

const (
	// XShardSendGas is the fixed cost of a cross-shard message. The destination
	// shard charges it out of the message's gas limit, so the limit must cover it.
	XShardSendGas uint64 = 9000

	// MaxCrossShardTxGas caps a message's destination gas budget, including
	// XShardSendGas. It matches half of QuarkChain's default 12,000,000 block gas limit.
	MaxCrossShardTxGas uint64 = 6_000_000

	// XShardGasLimitDivisor sets the cross-shard gas limit of a shard block, the
	// gas reserved for its deposits, to its gas limit over this divisor.
	XShardGasLimitDivisor uint64 = 2

	// RootCoinbaseShardID is the shard the root chain pays its coinbase to.
	RootCoinbaseShardID uint32 = 0
)

// QuarkChainConfig is the configuration of a QuarkChain shard chain. Its
// presence in the chain config enables the QuarkChain specific features.
type QuarkChainConfig struct {
	ShardID  uint32   `json:"shardId"`  // ID of this shard
	ShardIDs []uint32 `json:"shardIds"` // IDs of all shards in the network, this one included

	PoSW *PoSWConfig `json:"posw"` // Required consensus configuration for this shard.
	// RewardTaxRate is the fraction of the decayed subsidy and each miner fee
	// shared with root. Check requires a finite decimal in (0, 1) with a reduced
	// denominator no greater than 100. AfterTax returns the retained share.
	// TODO: Defer decimal JSON support to the upcoming config work; the config file
	// format and how QuarkChainConfig is constructed are not yet decided.
	RewardTaxRate *big.Rat `json:"rewardTaxRate"`
	// MaxBlocksPerRootBlock caps the number of blocks of this shard a root
	// block can confirm, as max_blocks_per_shard_in_one_root_block in
	// pyquarkchain. Zero means no cap.
	MaxBlocksPerRootBlock uint64 `json:"maxBlocksPerRootBlock,omitempty"`
}

// Equal compares all shard configuration fields by value. Two nil configs are equal.
func (c *QuarkChainConfig) Equal(other *QuarkChainConfig) bool {
	if c == nil || other == nil {
		return c == other
	}
	return c.ShardID == other.ShardID &&
		slices.Equal(c.ShardIDs, other.ShardIDs) &&
		c.PoSW.Equal(other.PoSW) &&
		configRatEqual(c.RewardTaxRate, other.RewardTaxRate) &&
		c.MaxBlocksPerRootBlock == other.MaxBlocksPerRootBlock
}

// Check validates consensus parameters and the shared reward/fee tax rule.
func (c *QuarkChainConfig) Check() error {
	if c == nil {
		return errors.New("quarkchain: missing shard configuration")
	}
	if err := c.PoSW.Check(); err != nil {
		return err
	}
	// Python's root reward reconstruction divides by the tax rate and the retained share.
	if c.RewardTaxRate == nil || c.RewardTaxRate.Sign() <= 0 || c.RewardTaxRate.Cmp(big.NewRat(1, 1)) >= 0 {
		return errors.New("quarkchain: reward tax rate must be greater than zero and less than one")
	}
	// Match pyquarkchain's reward_tax_rate denominator limit.
	if c.RewardTaxRate.Denom().Cmp(big.NewInt(100)) > 0 {
		return errors.New("quarkchain: reward tax rate denominator must not exceed 100")
	}
	if _, exact := c.RewardTaxRate.FloatPrec(); !exact {
		return errors.New("quarkchain: reward tax rate must have a finite decimal representation")
	}
	return nil
}

// AfterTax returns floor(amount * (1 - RewardTaxRate)) as a new integer.
// Apply it to each transaction fee before summing, as in pyquarkchain. amount
// must be nonnegative; Check validates the configured rate. Arbitrary-precision
// arithmetic avoids overflow when multiplying a uint256 amount by the numerator.
func (c *QuarkChainConfig) AfterTax(amount *big.Int) *big.Int {
	localRate := new(big.Rat).Sub(big.NewRat(1, 1), c.RewardTaxRate)
	local := new(big.Int).Mul(amount, localRate.Num())
	return local.Div(local, localRate.Denom())
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
