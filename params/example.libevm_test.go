// In practice, everything in this file except for the Example() function SHOULD
// be a standalone package, typically called `extraparams`. As long as this new
// package is imported anywhere, its init() function will register the "extra"
// types, which can be accessed via [extraparams.FromChainConfig] and/or
// [extraparams.FromRules]. In all other respects, the [params.ChainConfig] and
// [params.Rules] types will act as expected.
//
// The Example() function demonstrates how the `extraparams` package might be
// used from elsewhere.
package params_test

import (
	"encoding/json"
	"fmt"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/params"
)

// In practice this would be a regular init() function but nuances around the
// testing of this package require it to be called in the Example().
func initFn() {
	// This registration makes *all* [params.ChainConfig] and [params.Rules]
	// instances respect the payload types. They do not need to be modified to
	// know about `extraparams`.
	getter = params.RegisterExtras(params.Extras[ChainConfigExtra, RulesExtra]{
		NewRules: constructRulesExtra,
	})
}

var getter params.ExtraPayloadGetter[ChainConfigExtra, RulesExtra]

// constructRulesExtra acts as an adjunct to the [params.ChainConfig.Rules]
// method. Its primary purpose is to construct the extra payload for the
// [params.Rules] but it MAY also modify the [params.Rules].
func constructRulesExtra(c *params.ChainConfig, r *params.Rules, cEx *ChainConfigExtra, blockNum *big.Int, isMerge bool, timestamp uint64) *RulesExtra {
	return &RulesExtra{
		IsMyFork: cEx.MyForkTime != nil && *cEx.MyForkTime <= timestamp,
	}
}

// ChainConfigExtra can be any struct. Here it just mirrors a common pattern in
// the standard [params.ChainConfig] struct.
type ChainConfigExtra struct {
	MyForkTime *uint64 `json:"myForkTime"`
}

// RulesExtra can be any struct. It too mirrors a common pattern in
// [params.Rules].
type RulesExtra struct {
	IsMyFork bool
}

// FromChainConfig returns the extra payload carried by the ChainConfig.
func FromChainConfig(c *params.ChainConfig) *ChainConfigExtra {
	return getter.FromChainConfig(c)
}

// FromRules returns the extra payload carried by the Rules.
func FromRules(r *params.Rules) *RulesExtra {
	return getter.FromRules(r)
}

// This example demonstrates how the rest of this file would be used from a
// *different* package.
func ExampleExtraPayloadGetter() {
	initFn() // Outside of an example this is unnecessary as the function will be a regular init().

	const forkTime = 530003640
	jsonData := fmt.Sprintf(`{
		"chainId": 1234,
		"extra": {
			"myForkTime": %d
		}
	}`, forkTime)

	// Because [params.RegisterExtras] has been called, unmarshalling a JSON
	// field of "extra" into a [params.ChainConfig] will populate a new value of
	// the registered type. This can be accessed with the [FromChainConfig]
	// function.
	config := new(params.ChainConfig)
	if err := json.Unmarshal([]byte(jsonData), config); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Chain ID", config.ChainID) // original geth fields work as expected

	ccExtra := FromChainConfig(config) // extraparams.FromChainConfig() in practice
	if ccExtra != nil && ccExtra.MyForkTime != nil {
		fmt.Println("Fork time", *ccExtra.MyForkTime)
	}

	for _, time := range []uint64{forkTime - 1, forkTime, forkTime + 1} {
		rules := config.Rules(nil, false, time)
		rExtra := FromRules(&rules) // extraparams.FromRules() in practice
		if rExtra != nil {
			fmt.Printf("%+v\n", rExtra)
		}
	}

	// Output:
	// Chain ID 1234
	// Fork time 530003640
	// &{IsMyFork:false}
	// &{IsMyFork:true}
	// &{IsMyFork:true}
}
