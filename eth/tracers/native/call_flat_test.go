// Copyright 2024 The go-ethereum Authors
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
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func TestCallFlatStop(t *testing.T) {
	tracer, err := tracers.DefaultDirectory.New("flatCallTracer", &tracers.Context{}, nil, params.MainnetChainConfig)
	require.NoError(t, err)

	// this error should be returned by GetResult
	stopError := errors.New("stop error")

	// simulate a transaction
	tx := types.NewTx(&types.LegacyTx{
		Nonce:    0,
		To:       &common.Address{},
		Value:    big.NewInt(0),
		Gas:      0,
		GasPrice: big.NewInt(0),
		Data:     nil,
	})

	tracer.OnTxStart(&tracing.VMContext{}, tx, common.Address{})

	tracer.OnEnter(0, byte(vm.CALL), common.Address{}, common.Address{}, nil, 0, big.NewInt(0))

	// stop before the transaction is finished
	tracer.Stop(stopError)

	tracer.OnTxEnd(&types.Receipt{GasUsed: 0}, nil)

	// check that the error is returned by GetResult
	_, tracerError := tracer.GetResult()
	require.Equal(t, stopError, tracerError)
}
