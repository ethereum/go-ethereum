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
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/holiman/uint256"
)

// RewardConfig holds the configuration for block rewards
type RewardConfig struct {
	// BlockReward is the base block reward in wei
	BlockReward *big.Int
	// MasterPercent is the percentage of reward going to masternodes (90%)
	MasterPercent int64
	// VoterPercent is the percentage of reward going to voters (0% in current implementation)
	VoterPercent int64
	// FoundationPercent is the percentage of reward going to foundation (10%)
	FoundationPercent int64
	// FoundationWallet is the address receiving foundation rewards
	FoundationWallet common.Address
}

// DefaultRewardConfig returns the default XDC reward configuration
// Block reward: 5000 XDC per epoch (distributed among masternodes)
// Note: The actual calculation is more complex in production
func DefaultRewardConfig(foundationWallet common.Address) *RewardConfig {
	// 5000 XDC = 5000 * 10^18 wei
	blockReward := new(big.Int).Mul(big.NewInt(5000), big.NewInt(1e18))

	return &RewardConfig{
		BlockReward:       blockReward,
		MasterPercent:     RewardMasterPercent,     // 90
		VoterPercent:      RewardVoterPercent,      // 0
		FoundationPercent: RewardFoundationPercent, // 10
		FoundationWallet:  foundationWallet,
	}
}

// CalculateRewards calculates the rewards for masternodes at checkpoint blocks
// This implements the XDC reward distribution mechanism:
// - 90% to masternodes who signed blocks
// - 10% to foundation wallet
// - 0% to voters (handled separately by voter contract)
func CalculateRewards(
	chain consensus.ChainHeaderReader,
	state *state.StateDB,
	header *types.Header,
	config *RewardConfig,
	signers []common.Address,
	signCount map[common.Address]int64,
) (map[string]interface{}, error) {
	rewards := make(map[string]interface{})
	number := header.Number.Uint64()

	if config == nil || config.BlockReward == nil || config.BlockReward.Sign() <= 0 {
		log.Debug("No reward configured", "number", number)
		return rewards, nil
	}

	// Calculate total signs in this epoch
	var totalSigns int64
	for _, count := range signCount {
		totalSigns += count
	}

	if totalSigns == 0 {
		log.Warn("No signatures found for reward calculation", "number", number)
		return rewards, nil
	}

	// Calculate masternode portion (90%)
	masternodeReward := new(big.Int).Mul(config.BlockReward, big.NewInt(config.MasterPercent))
	masternodeReward.Div(masternodeReward, big.NewInt(100))

	// Calculate foundation portion (10%)
	foundationReward := new(big.Int).Mul(config.BlockReward, big.NewInt(config.FoundationPercent))
	foundationReward.Div(foundationReward, big.NewInt(100))

	// Distribute rewards to masternodes based on their signing activity
	masternodeRewards := make(map[common.Address]*big.Int)
	totalDistributed := big.NewInt(0)

	for addr, count := range signCount {
		if count > 0 {
			// Reward proportional to number of blocks signed
			reward := new(big.Int).Mul(masternodeReward, big.NewInt(count))
			reward.Div(reward, big.NewInt(totalSigns))

			if reward.Sign() > 0 {
				masternodeRewards[addr] = reward
				totalDistributed.Add(totalDistributed, reward)

				// Add reward to state (convert big.Int to uint256.Int)
				rewardU256, _ := uint256.FromBig(reward)
				state.AddBalance(addr, rewardU256, tracing.BalanceIncreaseRewardMineBlock)
				log.Debug("Masternode reward",
					"address", addr.Hex(),
					"reward", reward.String(),
					"signs", count,
					"block", number)
			}
		}
	}

	// Send foundation reward
	if foundationReward.Sign() > 0 && config.FoundationWallet != (common.Address{}) {
		foundationRewardU256, _ := uint256.FromBig(foundationReward)
		state.AddBalance(config.FoundationWallet, foundationRewardU256, tracing.BalanceIncreaseRewardMineBlock)
		log.Debug("Foundation reward",
			"address", config.FoundationWallet.Hex(),
			"reward", foundationReward.String(),
			"block", number)
	}

	// Build rewards map for logging/storage
	rewards["block"] = number
	rewards["totalReward"] = config.BlockReward.String()
	rewards["masternodeReward"] = masternodeReward.String()
	rewards["foundationReward"] = foundationReward.String()
	rewards["totalSigns"] = totalSigns

	masternodeRewardStrings := make(map[string]string)
	for addr, reward := range masternodeRewards {
		masternodeRewardStrings[addr.Hex()] = reward.String()
	}
	rewards["masternodes"] = masternodeRewardStrings

	log.Info("Rewards distributed",
		"block", number,
		"totalReward", config.BlockReward.String(),
		"masternodes", len(masternodeRewards),
		"foundation", foundationReward.String())

	return rewards, nil
}

// CreateDefaultHookReward creates a default reward hook function
// This can be used to set up the HookReward function in the XDPoS engine
func (c *XDPoS) CreateDefaultHookReward() func(chain consensus.ChainHeaderReader, state *state.StateDB, header *types.Header) (map[string]interface{}, error) {
	return func(chain consensus.ChainHeaderReader, state *state.StateDB, header *types.Header) (map[string]interface{}, error) {
		number := header.Number.Uint64()
		epoch := c.config.Epoch

		// Get the foundation wallet from config
		foundationWallet := common.Address{}
		if c.config.FoudationWalletAddr != (common.Address{}) {
			foundationWallet = c.config.FoudationWalletAddr
		}

		rewardConfig := DefaultRewardConfig(foundationWallet)

		// Get masternodes for this epoch
		masternodes := c.GetMasternodes(chain, header)
		if len(masternodes) == 0 {
			log.Warn("No masternodes found for reward calculation", "number", number)
			return make(map[string]interface{}), nil
		}

		// Count signatures in the epoch
		// In a full implementation, this would scan the blocks in the epoch
		// and count how many times each masternode signed
		signCount := make(map[common.Address]int64)

		startBlock := number - (number % epoch)
		if startBlock == 0 {
			startBlock = 1
		}

		// Simplified: give each masternode equal weight
		// In production, this should count actual block signatures
		for _, mn := range masternodes {
			signCount[mn] = 1
		}

		// Scan recent blocks to count actual signatures
		for blockNum := startBlock; blockNum < number; blockNum++ {
			blockHeader := chain.GetHeaderByNumber(blockNum)
			if blockHeader != nil {
				signer, err := c.RecoverSigner(blockHeader)
				if err == nil {
					signCount[signer]++
				}
			}
		}

		return CalculateRewards(chain, state, header, rewardConfig, masternodes, signCount)
	}
}
