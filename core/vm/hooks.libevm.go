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

package vm

import "github.com/ethereum/go-ethereum/params"

// RegisterHooks registers the Hooks. It is expected to be called in an `init()`
// function and MUST NOT be called more than once.
func RegisterHooks(h Hooks) {
	if libevmHooks != nil {
		panic("already registered")
	}
	libevmHooks = h
}

var libevmHooks Hooks

// Hooks are arbitrary configuration functions to modify default VM behaviour.
// See [RegisterHooks].
type Hooks interface {
	OverrideNewEVMArgs(*NewEVMArgs) *NewEVMArgs
	OverrideEVMResetArgs(params.Rules, *EVMResetArgs) *EVMResetArgs
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
	if libevmHooks == nil {
		return blockCtx, txCtx, statedb, chainConfig, config
	}
	args := libevmHooks.OverrideNewEVMArgs(&NewEVMArgs{blockCtx, txCtx, statedb, chainConfig, config})
	return args.BlockContext, args.TxContext, args.StateDB, args.ChainConfig, args.Config
}

func (evm *EVM) overrideEVMResetArgs(txCtx TxContext, statedb StateDB) (TxContext, StateDB) {
	if libevmHooks == nil {
		return txCtx, statedb
	}
	args := libevmHooks.OverrideEVMResetArgs(evm.chainRules, &EVMResetArgs{txCtx, statedb})
	return args.TxContext, args.StateDB
}
