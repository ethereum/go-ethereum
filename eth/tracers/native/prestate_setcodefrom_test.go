// Copyright 2026 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package native_test

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// TestPrestateTracerSetCodeFrom checks that the source account read by
// SETCODEFROM (EIP-8298) is part of the prestate.
func TestPrestateTracerSetCodeFrom(t *testing.T) {
	var (
		source = common.HexToAddress("0x1111111111111111111111111111111111111111")
		to     = common.HexToAddress("0x2222222222222222222222222222222222222222")
	)
	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	statedb.SetCode(source, []byte{0x60, 0x00}, tracing.CodeChangeUnspecified)

	tracer, err := tracers.DefaultDirectory.New("prestateTracer", &tracers.Context{}, nil, params.MergedTestChainConfig)
	require.NoError(t, err)
	env := &tracing.VMContext{StateDB: statedb, BlockNumber: new(big.Int)}
	tracer.OnTxStart(env, types.NewTx(&types.LegacyTx{To: &to}), common.Address{})
	stack := []uint256.Int{*new(uint256.Int).SetBytes(source.Bytes())}
	tracer.OnOpcode(0, byte(vm.SETCODEFROM), 0, 0, &mockOpContext{stack: stack}, nil, 1, nil)

	res, err := tracer.GetResult()
	require.NoError(t, err)
	var pre map[common.Address]json.RawMessage
	require.NoError(t, json.Unmarshal(res, &pre))
	require.Contains(t, pre, source)
}
