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

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/params"
)

type evmArgOverrider struct {
	newEVMchainID int64

	gotResetChainID  *big.Int
	resetTxContextTo TxContext
	resetStateDBTo   StateDB
}

func (o *evmArgOverrider) OverrideNewEVMArgs(args *NewEVMArgs) *NewEVMArgs {
	args.ChainConfig = &params.ChainConfig{ChainID: big.NewInt(o.newEVMchainID)}
	return args
}

func (o *evmArgOverrider) OverrideEVMResetArgs(r params.Rules, _ *EVMResetArgs) *EVMResetArgs {
	o.gotResetChainID = r.ChainID
	return &EVMResetArgs{
		TxContext: o.resetTxContextTo,
		StateDB:   o.resetStateDBTo,
	}
}

func (o *evmArgOverrider) register(t *testing.T) {
	t.Helper()
	libevmHooks = nil
	RegisterHooks(o)
	t.Cleanup(func() {
		libevmHooks = nil
	})
}

func TestOverrideNewEVMArgs(t *testing.T) {
	// The overrideNewEVMArgs function accepts and returns all arguments to
	// NewEVM(), in order. Here we lock in our assumption of that order. If this
	// breaks then all functionality overriding the args MUST be updated.
	var _ func(BlockContext, TxContext, StateDB, *params.ChainConfig, Config) *EVM = NewEVM

	const chainID = 13579
	hooks := evmArgOverrider{newEVMchainID: chainID}
	hooks.register(t)

	evm := NewEVM(BlockContext{}, TxContext{}, nil, nil, Config{})
	got := evm.ChainConfig().ChainID
	require.Equalf(t, big.NewInt(chainID), got, "%T.ChainConfig().ChainID set by NewEVM() hook", evm)
}

func TestOverrideEVMResetArgs(t *testing.T) {
	// Equivalent to rationale for TestOverrideNewEVMArgs above.
	var _ func(TxContext, StateDB) = (*EVM)(nil).Reset

	const (
		chainID  = 0xc0ffee
		gasPrice = 1357924680
	)
	hooks := &evmArgOverrider{
		newEVMchainID: chainID,
		resetTxContextTo: TxContext{
			GasPrice: big.NewInt(gasPrice),
		},
	}
	hooks.register(t)

	evm := NewEVM(BlockContext{}, TxContext{}, nil, nil, Config{})
	evm.Reset(TxContext{}, nil)
	assert.Equalf(t, big.NewInt(chainID), hooks.gotResetChainID, "%T.ChainID passed to Reset() hook", params.Rules{})
	assert.Equalf(t, big.NewInt(gasPrice), evm.GasPrice, "%T.GasPrice set by Reset() hook", evm)
}
