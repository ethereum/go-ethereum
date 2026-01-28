package util

import (
	"math/big"

	"github.com/ethereum/go-ethereum/consensus"
)

// RewardInflation calculates reward inflation based on block number and blocks per year
func RewardInflation(chain consensus.ChainReader, chainReward *big.Int, number uint64, blockPerYear uint64) *big.Int {
	// Check if halving should be disabled (for newer chains)
	// For now, use the halving schedule
	if blockPerYear*2 <= number && number < blockPerYear*5 {
		chainReward.Div(chainReward, new(big.Int).SetUint64(2))
	}
	if blockPerYear*5 <= number {
		chainReward.Div(chainReward, new(big.Int).SetUint64(4))
	}

	return chainReward
}

// RewardHalving computes the reward for Masternode/Protector/Observer based on epoch total reward, supply after halving is enabled, and epoch after halving is enabled
// The sequence is a geometric sequence in order to make supply be limited
func RewardHalving(epochRewardSingle *big.Int, epochRewardTotal *big.Int, halvingSupply *big.Int, epochSinceHalving uint64) *big.Int {
	rt := new(big.Float).SetInt(epochRewardTotal)
	hs := new(big.Float).SetInt(halvingSupply)
	// zero cause Quo panic so return early
	// or epoch reward > halving supply, return early
	if halvingSupply.BitLen() == 0 || epochRewardTotal.Cmp(halvingSupply) > 0 {
		return big.NewInt(0)
	}
	quo := new(big.Float).Quo(rt, hs)
	// base = 1- reward/supply
	base := new(big.Float).Sub(big.NewFloat(1), quo)
	r := new(big.Float).SetInt(epochRewardSingle)
	result := new(big.Float).Mul(r, FloatPower(base, epochSinceHalving))
	resultInt, _ := result.Int(nil)
	return resultInt
}

// FloatPower calculates base^exp for big.Float
func FloatPower(base *big.Float, exp uint64) *big.Float {
	result := big.NewFloat(1)
	for exp > 0 {
		if exp%2 == 1 {
			result.Mul(result, base)
		}
		base.Mul(base, base)
		exp >>= 1 // same as: exp = exp / 2
	}
	return result
}
