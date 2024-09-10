package params

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/libevm"
)

// ChainConfigHooks are required for all types registered as [Extras] for
// [ChainConfig] payloads.
type ChainConfigHooks interface{}

// TODO(arr4n): given the choice of whether a hook should be defined on a
// ChainConfig or on the Rules, what are the guiding principles? A ChainConfig
// carries the most general information while Rules benefit from "knowing" the
// block number and timestamp. I am leaning towards the default choice being
// on Rules (as it's trivial to copy information from ChainConfig to Rules in
// [Extras.NewRules]) unless the call site only has access to a ChainConfig.

// RulesHooks are required for all types registered as [Extras] for [Rules]
// payloads.
type RulesHooks interface {
	RulesAllowlistHooks
	// PrecompileOverride signals whether or not the EVM interpreter MUST
	// override its treatment of the address when deciding if it is a
	// precompiled contract. If PrecompileOverride returns `true` then the
	// interpreter will treat the address as a precompile i.f.f the
	// [PrecompiledContract] is non-nil. If it returns `false` then the default
	// precompile behaviour is honoured.
	PrecompileOverride(common.Address) (_ libevm.PrecompiledContract, override bool)
}

// RulesAllowlistHooks are a subset of [RulesHooks] that gate actions, signalled
// by returning a nil (allowed) or non-nil (blocked) error.
type RulesAllowlistHooks interface {
	CanCreateContract(*libevm.AddressContext, libevm.StateReader) error
	CanExecuteTransaction(from common.Address, to *common.Address, _ libevm.StateReader) error
}

// Hooks returns the hooks registered with [RegisterExtras], or [NOOPHooks] if
// none were registered.
func (c *ChainConfig) Hooks() ChainConfigHooks {
	if e := registeredExtras; e != nil {
		return e.getter.hooksFromChainConfig(c)
	}
	return NOOPHooks{}
}

// Hooks returns the hooks registered with [RegisterExtras], or [NOOPHooks] if
// none were registered.
func (r *Rules) Hooks() RulesHooks {
	if e := registeredExtras; e != nil {
		return e.getter.hooksFromRules(r)
	}
	return NOOPHooks{}
}

// NOOPHooks implements both [ChainConfigHooks] and [RulesHooks] such that every
// hook is a no-op. This allows it to be returned instead of a nil interface,
// which would otherwise require every usage site to perform a nil check. It can
// also be embedded in structs that only wish to implement a sub-set of hooks.
// Use of a NOOPHooks is equivalent to default Ethereum behaviour.
type NOOPHooks struct{}

var _ interface {
	ChainConfigHooks
	RulesHooks
} = NOOPHooks{}

// CanExecuteTransaction allows all (otherwise valid) transactions.
func (NOOPHooks) CanExecuteTransaction(_ common.Address, _ *common.Address, _ libevm.StateReader) error {
	return nil
}

// CanCreateContract allows all (otherwise valid) contract deployment.
func (NOOPHooks) CanCreateContract(*libevm.AddressContext, libevm.StateReader) error {
	return nil
}

// PrecompileOverride instructs the EVM interpreter to use the default
// precompile behaviour.
func (NOOPHooks) PrecompileOverride(common.Address) (libevm.PrecompiledContract, bool) {
	return nil, false
}
