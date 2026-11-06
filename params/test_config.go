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

import "math/big"

// NewTestQuarkChainConfig returns a fresh chain configuration for tests using
// fake consensus, with staking disabled and zero subsidy. The first chain receives
// root rewards. If chainIDs is omitted, chain 10001 is first and the local chain is included.
func NewTestQuarkChainConfig(chainID int64, chainIDs ...int64) *ChainConfig {
	if len(chainIDs) == 0 {
		chainIDs = []int64{10001}
		if chainID != 10001 {
			chainIDs = append(chainIDs, chainID)
		}
	}
	config := *MergedTestChainConfig
	config.ChainID = big.NewInt(chainID)
	config.QuarkChain = &QuarkChainConfig{
		MaxBlocksPerRootBlock: 100,
		PoSW: &PoSWConfig{
			AdjustmentCutoff: 7, AdjustmentFactor: 512, MinimumDifficulty: big.NewInt(1),
			CoinbaseAmount: big.NewInt(0), EpochInterval: 3153600, DecayFactor: big.NewRat(22, 25), RewardTaxRate: big.NewRat(1, 2),
		},
	}
	for _, id := range chainIDs {
		config.QuarkChain.ChainIDs = append(config.QuarkChain.ChainIDs, big.NewInt(id))
	}
	return &config
}
