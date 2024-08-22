package extraparams

import (
	"math/big"

	"github.com/ethereum/go-ethereum/params"
)

var getter params.ExtraPayloadGetter[ChainConfigExtra, RulesExtra]

func init() {
	getter = params.RegisterExtras(params.Extras[ChainConfigExtra, RulesExtra]{
		NewForRules: constructRulesExtra,
	})
}

type ChainConfigExtra struct {
	MyFeatureTime *uint64
}

type RulesExtra struct {
	IsMyFeature bool
}

func constructRulesExtra(c *params.ChainConfig, r *params.Rules, cEx *ChainConfigExtra, blockNum *big.Int, isMerge bool, timestamp uint64) *RulesExtra {
	return &RulesExtra{
		IsMyFeature: isMerge && cEx.MyFeatureTime != nil && *cEx.MyFeatureTime < timestamp,
	}
}

func FromChainConfig(c *params.ChainConfig) *ChainConfigExtra {
	return getter.FromChainConfig(c)
}

func FromRules(r *params.Rules) *RulesExtra {
	return getter.FromRules(r)
}
