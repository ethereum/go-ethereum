// Package hookstest provides test doubles and convenience wrappers for testing
// libevm hooks.
package hookstest

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/libevm"
	"github.com/ethereum/go-ethereum/params"
)

// Register clears any registered [params.Extras] and then registers `extras`
// for the liftime of the current test, clearing them via tb's
// [testing.TB.Cleanup].
func Register[C params.ChainConfigHooks, R params.RulesHooks](tb testing.TB, extras params.Extras[C, R]) {
	params.TestOnlyClearRegisteredExtras()
	tb.Cleanup(params.TestOnlyClearRegisteredExtras)
	params.RegisterExtras(extras)
}

// A Stub is a test double for [params.ChainConfigHooks] and
// [params.RulesHooks]. Each of the fields, if non-nil, back their respective
// hook methods, which otherwise fall back to the default behaviour.
type Stub struct {
	PrecompileOverrides     map[common.Address]libevm.PrecompiledContract
	CanExecuteTransactionFn func(common.Address, *common.Address, libevm.StateReader) error
	CanCreateContractFn     func(*libevm.AddressContext, libevm.StateReader) error
}

// Register is a convenience wrapper for registering s as both the
// [params.ChainConfigHooks] and [params.RulesHooks] via [Register].
func (s *Stub) Register(tb testing.TB) {
	Register(tb, params.Extras[Stub, Stub]{
		NewRules: func(_ *params.ChainConfig, _ *params.Rules, _ *Stub, blockNum *big.Int, isMerge bool, timestamp uint64) *Stub {
			return s
		},
	})
}

func (s Stub) PrecompileOverride(a common.Address) (libevm.PrecompiledContract, bool) {
	if len(s.PrecompileOverrides) == 0 {
		return nil, false
	}
	p, ok := s.PrecompileOverrides[a]
	return p, ok
}

func (s Stub) CanExecuteTransaction(from common.Address, to *common.Address, sr libevm.StateReader) error {
	if f := s.CanExecuteTransactionFn; f != nil {
		return f(from, to, sr)
	}
	return nil
}

func (s Stub) CanCreateContract(cc *libevm.AddressContext, sr libevm.StateReader) error {
	if f := s.CanCreateContractFn; f != nil {
		return f(cc, sr)
	}
	return nil
}

var _ interface {
	params.ChainConfigHooks
	params.RulesHooks
} = Stub{}
