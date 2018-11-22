package eth

import (
	"github.com/ethereum/go-ethereum/params"
	"math/big"
	"testing"
)

func TestRewardInflation(t *testing.T) {
	for i := 0; i < 100; i++ {
		chainReward := new(big.Int).Mul(new(big.Int).SetUint64(250), new(big.Int).SetUint64(params.Ether))
		chainReward = rewardInflation(chainReward, uint64(i), 10)

		halfReward := new(big.Int).Mul(new(big.Int).SetUint64(125), new(big.Int).SetUint64(params.Ether))
		if 20 <= i && i < 60 && chainReward.Cmp(halfReward) != 0 {
			t.Error("Fail tor calculate reward inflation for 2 -> 5 years", "chainReward", chainReward)
		}

		quarterReward := new(big.Int).Mul(new(big.Int).SetUint64(62.5*1000), new(big.Int).SetUint64(params.Finney))
		if 60 <= i && chainReward.Cmp(quarterReward) != 0 {
			t.Error("Fail tor calculate reward inflation above 6 years", "chainReward", chainReward)
		}
	}
}
