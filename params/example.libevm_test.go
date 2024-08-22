package params_test

import (
	"encoding/json"
	"fmt"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/params"
)

// TODO: explain why this isn't in an init()
func initFn() {
	getter = params.RegisterExtras(params.Extras[ChainConfigExtra, RulesExtra]{
		NewForRules: constructRulesExtra,
	})
}

var getter params.ExtraPayloadGetter[ChainConfigExtra, RulesExtra]

func FromChainConfig(c *params.ChainConfig) *ChainConfigExtra {
	return getter.FromChainConfig(c)
}

func FromRules(r *params.Rules) *RulesExtra {
	return getter.FromRules(r)
}

type ChainConfigExtra struct {
	MyForkTime *uint64 `json:"myForkTime"`
}

type RulesExtra struct {
	IsMyFork bool
}

func constructRulesExtra(c *params.ChainConfig, r *params.Rules, cEx *ChainConfigExtra, blockNum *big.Int, isMerge bool, timestamp uint64) *RulesExtra {
	return &RulesExtra{
		IsMyFork: cEx.MyForkTime != nil && *cEx.MyForkTime <= timestamp,
	}
}

func ExampleRegisterExtras() {
	initFn() // TODO: explain

	const forkTime = 530003640
	jsonData := fmt.Sprintf(`{
		"chainId": 1234,
		"extra": {
			"myForkTime": %d
		}
	}`, forkTime)

	// ChainConfig now unmarshals any JSON field named "extra" into a pointer to
	// the registered type, which is available via the ExtraPayload() method.
	config := new(params.ChainConfig)
	if err := json.Unmarshal([]byte(jsonData), config); err != nil {
		log.Fatal(err)
	}

	fmt.Println(config.ChainID) // original geth fields work as expected

	ccExtra := FromChainConfig(config)
	if ccExtra != nil && ccExtra.MyForkTime != nil {
		fmt.Println(*ccExtra.MyForkTime)
	}

	for _, time := range []uint64{forkTime - 1, forkTime, forkTime + 1} {
		rules := config.Rules(nil, false, time)
		rExtra := FromRules(&rules)
		if rExtra != nil {
			fmt.Printf("%+v\n", rExtra)
		}
	}

	// Output:
	// 1234
	// 530003640
	// &{IsMyFork:false}
	// &{IsMyFork:true}
	// &{IsMyFork:true}
}
