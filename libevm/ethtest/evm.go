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

	args := &evmConstructorArgs{
		vm.BlockContext{
			CanTransfer: core.CanTransfer,
			Transfer:    core.Transfer,
		},
		vm.TxContext{},
		sdb,
		&params.ChainConfig{},
		vm.Config{},
	}
	for _, o := range opts {
		o.apply(args)
	}

	return sdb, vm.NewEVM(
		args.blockContext,
		args.txContext,
		args.stateDB,
		args.chainConfig,
		args.config,
	)
}

type evmConstructorArgs struct {
	blockContext vm.BlockContext
	txContext    vm.TxContext
	stateDB      vm.StateDB
	chainConfig  *params.ChainConfig
	config       vm.Config
}

// An EVMOption configures the EVM returned by [NewZeroEVM].
type EVMOption interface {
	apply(*evmConstructorArgs)
}

type funcOption func(*evmConstructorArgs)

var _ EVMOption = funcOption(nil)

func (f funcOption) apply(args *evmConstructorArgs) { f(args) }

// WithBlockContext overrides the default context.
func WithBlockContext(c vm.BlockContext) EVMOption {
	return funcOption(func(args *evmConstructorArgs) {
		args.blockContext = c
	})
}

// WithBlockContext overrides the default context.
func WithChainConfig(c *params.ChainConfig) EVMOption {
	return funcOption(func(args *evmConstructorArgs) {
		args.chainConfig = c
	})
}
