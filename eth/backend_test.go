package eth

import (
	"math/big"
	"testing"

	"github.com/XinFinOrg/XDPoSChain/eth/util"
	"github.com/XinFinOrg/XDPoSChain/params"
)

func TestRewardInflation(t *testing.T) {
	for i := 0; i < 100; i++ {
		// the first 2 years
		chainReward := new(big.Int).Mul(new(big.Int).SetUint64(250), new(big.Int).SetUint64(params.Ether))
		chainReward = util.RewardInflation(nil, chainReward, uint64(i), 10)

		// 3rd year, 4th year, 5th year
		halfReward := new(big.Int).Mul(new(big.Int).SetUint64(125), new(big.Int).SetUint64(params.Ether))
		if 20 <= i && i < 50 && chainReward.Cmp(halfReward) != 0 {
			t.Error("Fail tor calculate reward inflation for 2 -> 5 years", "chainReward", chainReward)
		}

		// after 5 years
		quarterReward := new(big.Int).Mul(new(big.Int).SetUint64(62.5*1000), new(big.Int).SetUint64(params.Finney))
		if 50 <= i && chainReward.Cmp(quarterReward) != 0 {
			t.Error("Fail tor calculate reward inflation above 6 years", "chainReward", chainReward)
		}
	}
}
