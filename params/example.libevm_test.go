// Copyright 2024 the libevm authors.
//
// The libevm additions to go-ethereum are free software: you can redistribute
// them and/or modify them under the terms of the GNU Lesser General Public License
// as published by the Free Software Foundation, either version 3 of the License,
// or (at your option) any later version.
//
// The libevm additions are distributed in the hope that they will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Lesser
// General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see
// <http://www.gnu.org/licenses/>.

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
	"errors"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/core/vm"
	"github.com/ava-labs/libevm/libevm"
	"github.com/ava-labs/libevm/params"
)

// In practice this would be a regular init() function but nuances around the
// testing of this package require it to be called in the Example().
func initFn() {
	params.TestOnlyClearRegisteredExtras() // not necessary outside of the example
	// This registration makes *all* [params.ChainConfig] and [params.Rules]
	// instances respect the payload types. They do not need to be modified to
	// know about `extraparams`.
	payloads = params.RegisterExtras(params.Extras[ChainConfigExtra, RulesExtra]{
		NewRules: constructRulesExtra,
	})
}

var payloads params.ExtraPayloads[ChainConfigExtra, RulesExtra]

// constructRulesExtra acts as an adjunct to the [params.ChainConfig.Rules]
// method. Its primary purpose is to construct the extra payload for the
// [params.Rules] but it MAY also modify the [params.Rules].
func constructRulesExtra(c *params.ChainConfig, r *params.Rules, cEx ChainConfigExtra, blockNum *big.Int, isMerge bool, timestamp uint64) RulesExtra {
	return RulesExtra{
		IsMyFork:  cEx.MyForkTime != nil && *cEx.MyForkTime <= timestamp,
		timestamp: timestamp,
	}
}

// ChainConfigExtra can be any struct. Here it just mirrors a common pattern in
// the standard [params.ChainConfig] struct.
type ChainConfigExtra struct {
	MyForkTime *uint64 `json:"myForkTime"`

	// (Optional) If not all hooks are desirable then embedding a [NOOPHooks]
	// allows the type to satisfy the [ChainConfigHooks] interface, resulting in
	// default Ethereum behaviour.
	params.NOOPHooks
}

// RulesExtra can be any struct. It too mirrors a common pattern in
// [params.Rules].
type RulesExtra struct {
	IsMyFork  bool
	timestamp uint64

	params.NOOPHooks
}

// FromChainConfig returns the extra payload carried by the ChainConfig.
func FromChainConfig(c *params.ChainConfig) ChainConfigExtra {
	return payloads.FromChainConfig(c)
}

// FromRules returns the extra payload carried by the Rules.
func FromRules(r *params.Rules) RulesExtra {
	return payloads.FromRules(r)
}

// myForkPrecompiledContracts is analogous to the vm.PrecompiledContracts<Fork>
// maps. Note [RulesExtra.PrecompileOverride] treatment of nil values here.
var myForkPrecompiledContracts = map[common.Address]vm.PrecompiledContract{
	//...
	common.BytesToAddress([]byte{0x2}): nil, // i.e disabled
	//...
}

// PrecompileOverride implements the required [params.RuleHooks] method.
func (r RulesExtra) PrecompileOverride(addr common.Address) (_ libevm.PrecompiledContract, override bool) {
	if !r.IsMyFork {
		return nil, false
	}
	p, ok := myForkPrecompiledContracts[addr]
	// The returned boolean indicates whether or not [vm.EVMInterpreter] MUST
	// override the address, not what it returns as its own `isPrecompile`
	// boolean.
	//
	// Therefore returning `nil, true` here indicates that the precompile will
	// be disabled. Returning `false` here indicates that the default precompile
	// behaviour will be exhibited.
	//
	// The same pattern can alternatively be implemented with an explicit
	// `disabledPrecompiles` set to make the behaviour clearer.
	return p, ok
}

// CanCreateContract implements the required [params.RuleHooks] method. Access
// to state allows it to be configured on-chain however this is an optional
// implementation detail.
func (r RulesExtra) CanCreateContract(_ *libevm.AddressContext, gas uint64, _ libevm.StateReader) (uint64, error) {
	if time.Unix(int64(r.timestamp), 0).UTC().Day() != int(time.Tuesday) { //nolint:gosec // G115 timestamp won't overflow int64 for millions of years so this is someone else's problem
		// Consumes all remaining gas.
		return 0, errors.New("uh oh")
	}
	return gas, nil
}

// This example demonstrates how the rest of this file would be used from a
// *different* package.
func ExampleExtraPayloads() {
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
	if ccExtra.MyForkTime != nil {
		fmt.Println("Fork time", *ccExtra.MyForkTime)
	}

	for _, time := range []uint64{forkTime - 1, forkTime, forkTime + 1} {
		rules := config.Rules(nil, false, time)
		rExtra := FromRules(&rules) // extraparams.FromRules() in practice
		fmt.Printf("IsMyFork at %v: %t\n", rExtra.timestamp, rExtra.IsMyFork)
	}

	// Output:
	// Chain ID 1234
	// Fork time 530003640
	// IsMyFork at 530003639: false
	// IsMyFork at 530003640: true
	// IsMyFork at 530003641: true
}
