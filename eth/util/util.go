package util

import (
	"math/big"

	"github.com/XinFinOrg/XDPoSChain/consensus"
)

func RewardInflation(chain consensus.ChainReader, chainReward *big.Int, number uint64, blockPerYear uint64) *big.Int {
	if chain != nil && chain.Config().IsTIPNoHalvingMNReward(new(big.Int).SetUint64(number)) {
		return chainReward
	}

	if blockPerYear*2 <= number && number < blockPerYear*5 {
		chainReward.Div(chainReward, new(big.Int).SetUint64(2))
	}
	if blockPerYear*5 <= number {
		chainReward.Div(chainReward, new(big.Int).SetUint64(4))
	}

	return chainReward
}
