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

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/params"
)

type chainIDOverrider struct {
	chainID int64
}

func (o chainIDOverrider) OverrideNewEVMArgs(args *NewEVMArgs) *NewEVMArgs {
	args.ChainConfig = &params.ChainConfig{ChainID: big.NewInt(o.chainID)}
	return args
}

func TestOverrideNewEVMArgs(t *testing.T) {
	// The overrideNewEVMArgs function accepts and returns all arguments to
	// NewEVM(), in order. Here we lock in our assumption of that order. If this
	// breaks then all functionality overriding the args MUST be updated.
	var _ func(BlockContext, TxContext, StateDB, *params.ChainConfig, Config) *EVM = NewEVM

	const chainID = 13579
	libevmHooks = nil
	RegisterHooks(chainIDOverrider{chainID: chainID})
	defer func() { libevmHooks = nil }()

	got := NewEVM(BlockContext{}, TxContext{}, nil, nil, Config{}).ChainConfig().ChainID
	require.Equal(t, big.NewInt(chainID), got)
}
