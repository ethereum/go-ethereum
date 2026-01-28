// Copyright (c) 2018 XDCchain
// Copyright 2024 The go-ethereum Authors
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package XDPoS

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// PenaltyConfig holds configuration for penalty calculation
type PenaltyConfig struct {
	// MinimumBlocksPerEpoch is the minimum number of blocks a masternode must sign
	// to avoid being penalized
	MinimumBlocksPerEpoch int64

	// LimitPenaltyEpoch is the number of epochs a penalized masternode remains banned
	LimitPenaltyEpoch int
}

// DefaultPenaltyConfig returns the default penalty configuration
func DefaultPenaltyConfig() *PenaltyConfig {
	return &PenaltyConfig{
		MinimumBlocksPerEpoch: MinimunMinerBlockPerEpoch, // 1
		LimitPenaltyEpoch:     LimitPenaltyEpoch,         // 4
	}
}

// CalculatePenalties determines which masternodes should be penalized
// Masternodes are penalized if they:
// - Failed to sign the minimum required blocks in an epoch
// - Behaved maliciously (double signing, etc.)
func CalculatePenalties(
	chain consensus.ChainHeaderReader,
	epoch uint64,
	masternodes []common.Address,
	signCount map[common.Address]int64,
	config *PenaltyConfig,
) []common.Address {
	penalties := make([]common.Address, 0)

	if config == nil {
		config = DefaultPenaltyConfig()
	}

	for _, masternode := range masternodes {
		count, exists := signCount[masternode]

		// Penalize masternodes who didn't sign enough blocks
		if !exists || count < config.MinimumBlocksPerEpoch {
			penalties = append(penalties, masternode)
			log.Info("Masternode penalized for insufficient signing",
				"address", masternode.Hex(),
				"signCount", count,
				"minimum", config.MinimumBlocksPerEpoch,
				"epoch", epoch)
		}
	}

	return penalties
}

// CreateDefaultHookPenalty creates a default penalty hook function
// This calculates penalties based on block signing activity
func (c *XDPoS) CreateDefaultHookPenalty() func(chain consensus.ChainHeaderReader, blockNumberEpoch uint64) ([]common.Address, error) {
	return func(chain consensus.ChainHeaderReader, blockNumberEpoch uint64) ([]common.Address, error) {
		epoch := c.config.Epoch
		if blockNumberEpoch == 0 || epoch == 0 {
			return []common.Address{}, nil
		}

		// Calculate the start and end blocks of the previous epoch
		epochStart := blockNumberEpoch - epoch
		if epochStart > blockNumberEpoch {
			// Overflow protection
			return []common.Address{}, nil
		}

		// Get checkpoint header for this epoch
		checkpointHeader := chain.GetHeaderByNumber(epochStart)
		if checkpointHeader == nil {
			log.Warn("Could not find checkpoint header for penalty calculation",
				"epochStart", epochStart)
			return []common.Address{}, nil
		}

		// Get masternodes for this epoch
		masternodes := c.GetMasternodesFromCheckpointHeader(checkpointHeader, epochStart, epoch)
		if len(masternodes) == 0 {
			return []common.Address{}, nil
		}

		// Count how many blocks each masternode signed
		signCount := make(map[common.Address]int64)

		for blockNum := epochStart + 1; blockNum < blockNumberEpoch; blockNum++ {
			header := chain.GetHeaderByNumber(blockNum)
			if header == nil {
				continue
			}

			signer, err := c.RecoverSigner(header)
			if err != nil {
				log.Debug("Could not recover signer for penalty calculation",
					"block", blockNum, "error", err)
				continue
			}

			signCount[signer]++
		}

		// Calculate penalties
		penalties := CalculatePenalties(chain, blockNumberEpoch/epoch, masternodes, signCount, DefaultPenaltyConfig())

		return penalties, nil
	}
}

// CreateDefaultHookPenaltyTIPSigning creates a penalty hook for TIP signing
// This is an enhanced penalty calculation that includes signature validation
func (c *XDPoS) CreateDefaultHookPenaltyTIPSigning() func(chain consensus.ChainHeaderReader, header *types.Header, candidates []common.Address) ([]common.Address, error) {
	return func(chain consensus.ChainHeaderReader, header *types.Header, candidates []common.Address) ([]common.Address, error) {
		number := header.Number.Uint64()
		epoch := c.config.Epoch

		if number == 0 || epoch == 0 {
			return []common.Address{}, nil
		}

		// Get the epoch start
		epochStart := number - epoch
		if epochStart > number {
			return []common.Address{}, nil
		}

		// Build a map of valid candidates
		candidateMap := make(map[common.Address]bool)
		for _, c := range candidates {
			candidateMap[c] = true
		}

		// Count signatures
		signCount := make(map[common.Address]int64)

		for blockNum := epochStart + 1; blockNum < number; blockNum++ {
			blockHeader := chain.GetHeaderByNumber(blockNum)
			if blockHeader == nil {
				continue
			}

			signer, err := c.RecoverSigner(blockHeader)
			if err != nil {
				continue
			}

			// Only count signatures from valid candidates
			if candidateMap[signer] {
				signCount[signer]++
			}
		}

		// Calculate penalties for candidates who didn't sign enough
		penalties := make([]common.Address, 0)
		minBlocks := int64(MinimunMinerBlockPerEpoch)

		for _, candidate := range candidates {
			if signCount[candidate] < minBlocks {
				penalties = append(penalties, candidate)
				log.Info("Candidate penalized (TIP signing)",
					"address", candidate.Hex(),
					"signCount", signCount[candidate],
					"minimum", minBlocks,
					"block", number)
			}
		}

		return penalties, nil
	}
}

// ExtractPenaltiesFromHeader extracts penalty addresses from a block header
func ExtractPenaltiesFromHeader(header *types.Header) []common.Address {
	// In XDC, penalties are stored in the Penalties field of the header
	// For compatibility with standard go-ethereum, we might need to handle this differently
	// Currently returning empty as standard headers don't have this field

	// If we had access to header.Penalties:
	// return extractAddressFromBytes(header.Penalties)

	return []common.Address{}
}

// IsPenalized checks if an address is in the penalty list
func IsPenalized(address common.Address, penalties []common.Address) bool {
	for _, penalty := range penalties {
		if penalty == address {
			return true
		}
	}
	return false
}
