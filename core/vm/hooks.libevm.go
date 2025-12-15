// Copyright 2024-2025 the libevm authors.
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

package vm

import (
	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/libevm"
	"github.com/ava-labs/libevm/libevm/register"
	"github.com/ava-labs/libevm/params"
)

// RegisterHooks registers the Hooks. It is expected to be called in an `init()`
// function and MUST NOT be called more than once.
func RegisterHooks(h Hooks) {
	libevmHooks.MustRegister(h)
}

// WithTempRegisteredHooks temporarily registers `h` as if calling
// [RegisterHooks] the same type parameter. After `fn` returns, the registration
// is returned to its former state, be that none or the types originally passed
// to [RegisterHooks].
//
// This MUST NOT be used on a live chain. It is solely intended for off-chain
// consumers that require access to extras. Said consumers SHOULD NOT, however
// call this function directly. Use the libevm/temporary.WithRegisteredExtras()
// function instead as it atomically overrides all possible packages.
func WithTempRegisteredHooks(lock libevm.ExtrasLock, h Hooks, fn func() error) error {
	if err := lock.Verify(); err != nil {
		return err
	}
	return libevmHooks.TempOverride(h, fn)
}

// TestOnlyClearRegisteredHooks clears the [Hooks] previously passed to
// [RegisterHooks]. It panics if called from a non-testing call stack.
func TestOnlyClearRegisteredHooks() {
	libevmHooks.TestOnlyClear()
}

var libevmHooks register.AtMostOnce[Hooks]

// Hooks are arbitrary configuration functions to modify default VM behaviour.
// See [RegisterHooks].
type Hooks interface {
	OverrideNewEVMArgs(*NewEVMArgs) *NewEVMArgs
	OverrideEVMResetArgs(params.Rules, *EVMResetArgs) *EVMResetArgs
	Preprocessor
}

// A Preprocessor performs computation on a transaction before the
// [EVMInterpreter] is invoked and reports its gas charge for spending at the
// beginning of [EVM.Call] or [EVM.Create].
type Preprocessor interface {
	PreprocessingGasCharge(tx common.Hash) (uint64, error)
}

// NewEVMArgs are the arguments received by [NewEVM], available for override
// via [Hooks].
type NewEVMArgs struct {
	BlockContext BlockContext
	TxContext    TxContext
	StateDB      StateDB
	ChainConfig  *params.ChainConfig
	Config       Config
}

// EVMResetArgs are the arguments received by [EVM.Reset], available for
// override via [Hooks].
type EVMResetArgs struct {
	TxContext TxContext
	StateDB   StateDB
}

func overrideNewEVMArgs(
	blockCtx BlockContext,
	txCtx TxContext,
	statedb StateDB,
	chainConfig *params.ChainConfig,
	config Config,
) (BlockContext, TxContext, StateDB, *params.ChainConfig, Config) {
	if !libevmHooks.Registered() {
		return blockCtx, txCtx, statedb, chainConfig, config
	}
	args := libevmHooks.Get().OverrideNewEVMArgs(&NewEVMArgs{blockCtx, txCtx, statedb, chainConfig, config})
	return args.BlockContext, args.TxContext, args.StateDB, args.ChainConfig, args.Config
}

func (evm *EVM) overrideEVMResetArgs(txCtx TxContext, statedb StateDB) (TxContext, StateDB) {
	if !libevmHooks.Registered() {
		return txCtx, statedb
	}
	args := libevmHooks.Get().OverrideEVMResetArgs(evm.chainRules, &EVMResetArgs{txCtx, statedb})
	return args.TxContext, args.StateDB
}

// NOOPHooks implements [Hooks] such that every method is a noop.
type NOOPHooks struct{}

var _ Hooks = NOOPHooks{}

// OverrideNewEVMArgs returns the args unchanged.
func (NOOPHooks) OverrideNewEVMArgs(a *NewEVMArgs) *NewEVMArgs {
	return a
}

// OverrideEVMResetArgs returns the args unchanged.
func (NOOPHooks) OverrideEVMResetArgs(_ params.Rules, a *EVMResetArgs) *EVMResetArgs {
	return a
}

// PreprocessingGasCharge returns (0, nil).
func (NOOPHooks) PreprocessingGasCharge(common.Hash) (uint64, error) {
	return 0, nil
}
