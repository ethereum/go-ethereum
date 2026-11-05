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
)

// ConsensusType identifies the PoW algorithm selected for a QKC root or shard chain.
type ConsensusType string

const (
	ConsensusEthash   ConsensusType = "POW_ETHASH"
	ConsensusSimulate ConsensusType = "POW_SIMULATE"
)

// Check rejects unknown algorithms and NONE, since every configured chain is active.
func (c ConsensusType) Check() error {
	switch c {
	case ConsensusEthash, ConsensusSimulate:
		return nil
	default:
		return fmt.Errorf("invalid ConsensusType %q: want POW_ETHASH, POW_SIMULATE", c)
	}
}

// QuarkChainConfig is the configuration of a QuarkChain shard chain. Its
// presence in the chain config enables the QuarkChain specific features.
type QuarkChainConfig struct {
	// ChainIDs lists the EVM chain IDs of all shards, this one included.
	// The first chain receives root rewards and hosts the root staking contract.
	// Network loading derives this list; it cannot be configured separately in TOML.
	ChainIDs []*big.Int `json:"chainIds" toml:"-"`

	ConsensusType ConsensusType `json:"consensusType,omitempty"`
	PoSW          *PoSWConfig   `json:"posw"` // Required consensus configuration for this shard.
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
	return slices.EqualFunc(c.ChainIDs, other.ChainIDs, configBlockEqual) &&
		c.ConsensusType == other.ConsensusType &&
		c.PoSW.Equal(other.PoSW) &&
		c.MaxBlocksPerRootBlock == other.MaxBlocksPerRootBlock
}

// String implements the stringer interface.
func (c QuarkChainConfig) String() string {
	return fmt.Sprintf("quarkchain(chains: %v)", c.ChainIDs)
}

// IsXShardDestination reports whether this shard can send a cross-shard
// message to the given shard.
func (c *ChainConfig) IsXShardDestination(chainID *big.Int) bool {
	return c != nil && c.QuarkChain != nil && chainID != nil && chainID.Sign() > 0 &&
		!configBlockEqual(chainID, c.ChainID) &&
		slices.ContainsFunc(c.QuarkChain.ChainIDs, func(id *big.Int) bool { return configBlockEqual(id, chainID) })
}

// IsRootCoinbaseChain reports whether this chain receives root rewards and hosts root staking.
// These are assigned to the first chain in QuarkChain.ChainIDs.
func (c *ChainConfig) IsRootCoinbaseChain() bool {
	return c != nil && c.QuarkChain != nil && c.ChainID != nil && len(c.QuarkChain.ChainIDs) > 0 &&
		configBlockEqual(c.ChainID, c.QuarkChain.ChainIDs[0])
}

// CheckQuarkChain validates the shard identity, network membership and consensus settings.
// Shard IDs use the same positive uint256 values as transaction signatures and the EVM.
func (c *ChainConfig) CheckQuarkChain() error {
	if c.QuarkChain == nil {
		return nil
	}
	validID := func(id *big.Int) bool { return id != nil && id.Sign() > 0 && id.BitLen() <= 256 }
	if !validID(c.ChainID) {
		return fmt.Errorf("invalid QuarkChain chain ID: %v", c.ChainID)
	}
	q := c.QuarkChain
	if len(q.ChainIDs) == 0 {
		return fmt.Errorf("chain IDs must not be empty")
	}
	seen := make(map[string]bool, len(q.ChainIDs))
	for _, id := range q.ChainIDs {
		if !validID(id) {
			return fmt.Errorf("invalid member chain ID: %v", id)
		}
		if seen[id.String()] {
			return fmt.Errorf("duplicate chain ID: %v", id)
		}
		seen[id.String()] = true
	}
	if !seen[c.ChainID.String()] {
		return fmt.Errorf("chain IDs must include local chain %v", c.ChainID)
	}
	if q.ConsensusType != "" {
		if err := q.ConsensusType.Check(); err != nil {
			return err
		}
	}
	return q.PoSW.Check()
}
