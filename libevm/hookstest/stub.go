// Package hookstest provides test doubles for testing subsets of libevm hooks.
package hookstest

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/libevm"
	"github.com/ethereum/go-ethereum/params"
)

// A Stub is a test double for [params.ChainConfigHooks] and
// [params.RulesHooks]. Each of the fields, if non-nil, back their respective
// hook methods, which otherwise fall back to the default behaviour.
type Stub struct {
	PrecompileOverrides     map[common.Address]libevm.PrecompiledContract
	CanExecuteTransactionFn func(common.Address, *common.Address, libevm.StateReader) error
	CanCreateContractFn     func(*libevm.AddressContext, libevm.StateReader) error
}

// RegisterForRules clears any registered [params.Extras] and then registers s
// as [params.RulesHooks], which are themselves cleared by the
// [testing.TB.Cleanup] routine.
func (s *Stub) RegisterForRules(tb testing.TB) {
	params.TestOnlyClearRegisteredExtras()
	params.RegisterExtras(params.Extras[params.NOOPHooks, Stub]{
		NewRules: func(_ *params.ChainConfig, _ *params.Rules, _ *params.NOOPHooks, blockNum *big.Int, isMerge bool, timestamp uint64) *Stub {
			return s
		},
	})
	tb.Cleanup(params.TestOnlyClearRegisteredExtras)
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
