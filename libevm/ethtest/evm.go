// Package ethtest provides utility functions for use in testing
// Ethereum-related functionality.
package ethtest

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

// NewZeroEVM returns a new EVM backed by a [rawdb.NewMemoryDatabase]; all other
// arguments to [vm.NewEVM] are the zero values of their respective types,
// except for the use of [core.CanTransfer] and [core.Transfer] instead of nil
// functions.
func NewZeroEVM(tb testing.TB, opts ...EVMOption) (*state.StateDB, *vm.EVM) {
	tb.Helper()

	sdb, err := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	require.NoError(tb, err, "state.New()")

	vm := vm.NewEVM(
		vm.BlockContext{
			CanTransfer: core.CanTransfer,
			Transfer:    core.Transfer,
		},
		vm.TxContext{},
		sdb,
		&params.ChainConfig{},
		vm.Config{},
	)
	for _, o := range opts {
		o.apply(vm)
	}

	return sdb, vm
}

// An EVMOption configures the EVM returned by [NewZeroEVM].
type EVMOption interface {
	apply(*vm.EVM)
}

type funcOption func(*vm.EVM)

var _ EVMOption = funcOption(nil)

func (f funcOption) apply(vm *vm.EVM) { f(vm) }

// WithBlockContext overrides the default context.
func WithBlockContext(c vm.BlockContext) EVMOption {
	return funcOption(func(vm *vm.EVM) {
		vm.Context = c
	})
}
