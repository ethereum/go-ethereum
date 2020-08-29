// Copyright 2017 The go-ethereum Authors
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

package misc

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

// TestCalcEIP1559GasTarget tests that CalEIP1559GasTarget()returns the correct value
func TestCalcEIP1559GasTarget(t *testing.T) {
	testConditions := []struct {
		// Test inputs
		config             *params.ChainConfig
		eip1559activation  *big.Int
		transitionDuration uint64
		height             *big.Int
		gasLimit           *big.Int
		// Expected result
		eip1559GasTarget *big.Int
	}{
		{
			params.EIP1559ChainConfig,
			big.NewInt(1000),
			1000,
			big.NewInt(1),
			big.NewInt(100000),
			big.NewInt(0),
		},
		{
			params.EIP1559ChainConfig,
			big.NewInt(1000),
			1000,
			big.NewInt(999),
			big.NewInt(100000),
			big.NewInt(0),
		},
		{
			params.EIP1559ChainConfig,
			big.NewInt(1000),
			1000,
			big.NewInt(1000),
			big.NewInt(100000),
			big.NewInt(50000),
		},
		{
			params.EIP1559ChainConfig,
			big.NewInt(1000),
			1000,
			big.NewInt(1001),
			big.NewInt(100000),
			big.NewInt(50050),
		},
		{
			params.EIP1559ChainConfig,
			big.NewInt(1000),
			1000,
			big.NewInt(1500),
			big.NewInt(100000),
			big.NewInt(75000),
		},
		{
			params.EIP1559ChainConfig,
			big.NewInt(1000),
			1000,
			big.NewInt(1999),
			big.NewInt(100000),
			big.NewInt(99950),
		},
		{
			params.EIP1559ChainConfig,
			big.NewInt(1000),
			1000,
			big.NewInt(2000),
			big.NewInt(100000),
			big.NewInt(100000),
		},
		{
			params.EIP1559ChainConfig,
			big.NewInt(1000),
			1000,
			big.NewInt(2001),
			big.NewInt(100000),
			big.NewInt(100000),
		},
	}
	for i, test := range testConditions {
		config := *test.config
		config.EIP1559Block = test.eip1559activation
		config.EIP1559.MigrationBlockDuration = test.transitionDuration
		config.EIP1559FinalizedBlock = new(big.Int).Add(config.EIP1559Block, new(big.Int).SetUint64(config.EIP1559.MigrationBlockDuration))
		gasTarget := CalcEIP1559GasTarget(&config, test.height, test.gasLimit)
		if gasTarget.Cmp(test.eip1559GasTarget) != 0 {
			t.Errorf("test %d expected EIP1559GasTarget %d got %d", i+1, test.eip1559GasTarget.Uint64(), gasTarget.Uint64())
		}
	}
}

// TestCalcBaseFee tests that CalcBaseFee()returns the correct value
func TestCalcBaseFee(t *testing.T) {
	testConditions := []struct {
		// Test inputs
		config             *params.ChainConfig
		eip1559activation  *big.Int
		transitionDuration uint64
		parentHeight       *big.Int
		parentBaseFee      *big.Int
		parentGasLimit     uint64
		parentGasUsed      uint64
		// Expected result
		baseFee *big.Int
	}{
		{
			params.EIP1559ChainConfig,
			big.NewInt(1000),
			1000,
			big.NewInt(1),
			big.NewInt(1000000000),
			1000000,
			10000000,
			nil,
		},
		{
			params.EIP1559ChainConfig,
			big.NewInt(1000),
			1000,
			big.NewInt(999),
			big.NewInt(1000000000),
			1000000,
			10000000,
			new(big.Int).SetUint64(params.EIP1559ChainConfig.EIP1559.InitialBaseFee),
		},
		{
			params.EIP1559ChainConfig,
			big.NewInt(1000),
			1000,
			big.NewInt(1000),
			big.NewInt(1000000000),
			1000000,
			10000000,
			big.NewInt(1125000000),
		},
		{
			params.EIP1559ChainConfig,
			big.NewInt(1000),
			1000,
			big.NewInt(2000), // past finalization parentGasLimit is the EIP1559GasTarget
			big.NewInt(1000000000),
			500000,
			10000000,
			big.NewInt(1125000000),
		},
		{
			params.EIP1559ChainConfig,
			big.NewInt(1000),
			1000,
			big.NewInt(2000),
			big.NewInt(1000000000),
			1000000,
			10000000,
			big.NewInt(1125000000),
		},
		{
			params.EIP1559ChainConfig,
			big.NewInt(1000),
			1000,
			big.NewInt(2000),
			big.NewInt(1000000000),
			6000000,
			10000000,
			big.NewInt(1083333333),
		},
		{
			params.EIP1559ChainConfig,
			big.NewInt(1000),
			1000,
			big.NewInt(2000),
			big.NewInt(1000000000),
			7000000,
			10000000,
			big.NewInt(1053571428),
		},
		{
			params.EIP1559ChainConfig,
			big.NewInt(1000),
			1000,
			big.NewInt(2000),
			big.NewInt(1000000000),
			8000000,
			10000000,
			big.NewInt(1031250000),
		},
		{
			params.EIP1559ChainConfig,
			big.NewInt(1000),
			1000,
			big.NewInt(2000),
			big.NewInt(1000000000),
			9000000,
			10000000,
			big.NewInt(1013888888),
		},
		{
			params.EIP1559ChainConfig,
			big.NewInt(1000),
			1000,
			big.NewInt(2000),
			big.NewInt(1000000000),
			10000000,
			10000000,
			big.NewInt(999999999), // baseFee diff is -1 when usage == target
		},
		{
			params.EIP1559ChainConfig,
			big.NewInt(1000),
			1000,
			big.NewInt(2000),
			big.NewInt(1000000000),
			11000000,
			10000000,
			big.NewInt(988636363),
		},
		{
			params.EIP1559ChainConfig,
			big.NewInt(1000),
			1000,
			big.NewInt(2000),
			big.NewInt(900000000),
			1000000,
			10000000,
			big.NewInt(1012500000),
		},
		{
			params.EIP1559ChainConfig,
			big.NewInt(1000),
			1000,
			big.NewInt(2000),
			big.NewInt(1100000000),
			1000000,
			10000000,
			big.NewInt(1237500000),
		},
		{
			params.EIP1559ChainConfig,
			big.NewInt(1000),
			1000,
			big.NewInt(2000),
			big.NewInt(1200000000),
			1000000,
			10000000,
			big.NewInt(1350000000),
		},
		{
			params.EIP1559ChainConfig,
			big.NewInt(1000),
			1000,
			big.NewInt(2000),
			big.NewInt(1000000000),
			10000000,
			9000000,
			big.NewInt(987500000),
		},
		{
			params.EIP1559ChainConfig,
			big.NewInt(1000),
			1000,
			big.NewInt(2000),
			big.NewInt(1000000000),
			10000000,
			11000000,
			big.NewInt(1012500000),
		},
		{
			params.EIP1559ChainConfig,
			big.NewInt(1000),
			1000,
			big.NewInt(2000),
			big.NewInt(1000000000),
			10000000,
			12000000,
			big.NewInt(1025000000),
		},
		{
			params.EIP1559ChainConfig,
			big.NewInt(1000),
			1000,
			big.NewInt(2000),
			big.NewInt(0),
			1,
			1000000000000000,
			big.NewInt(1),
		},

		// Low parent baseFee tests
		{
			params.EIP1559ChainConfig,
			big.NewInt(1000),
			1000,
			big.NewInt(2000),
			big.NewInt(1),
			1,
			8,
			big.NewInt(2),
		},
		{
			params.EIP1559ChainConfig,
			big.NewInt(1000),
			1000,
			big.NewInt(2000),
			big.NewInt(1),
			1,
			9,
			big.NewInt(2),
		},
		{
			params.EIP1559ChainConfig,
			big.NewInt(1000),
			1000,
			big.NewInt(2000),
			big.NewInt(1),
			100000000,
			899999999,
			big.NewInt(2),
		},
		{
			params.EIP1559ChainConfig,
			big.NewInt(1000),
			1000,
			big.NewInt(2000),
			big.NewInt(1),
			100000000,
			900000000,
			big.NewInt(2),
		},
		{
			params.EIP1559ChainConfig,
			big.NewInt(1000),
			1000,
			big.NewInt(2000),
			big.NewInt(2),
			1,
			4,
			big.NewInt(3),
		},
		{
			params.EIP1559ChainConfig,
			big.NewInt(1000),
			1000,
			big.NewInt(2000),
			big.NewInt(2),
			1,
			5,
			big.NewInt(3),
		},
		{
			params.EIP1559ChainConfig,
			big.NewInt(1000),
			1000,
			big.NewInt(2000),
			big.NewInt(2),
			100000000,
			499999999,
			big.NewInt(3),
		},
		{
			params.EIP1559ChainConfig,
			big.NewInt(1000),
			1000,
			big.NewInt(2000),
			big.NewInt(2),
			100000000,
			500000000,
			big.NewInt(3),
		},
	}
	for i, test := range testConditions {
		config := *test.config
		config.EIP1559Block = test.eip1559activation
		config.EIP1559.MigrationBlockDuration = test.transitionDuration
		config.EIP1559FinalizedBlock = new(big.Int).Add(config.EIP1559Block, new(big.Int).SetUint64(config.EIP1559.MigrationBlockDuration))
		parent := &types.Header{
			GasLimit: test.parentGasLimit,
			GasUsed:  test.parentGasUsed,
			Number:   test.parentHeight,
			BaseFee:  test.parentBaseFee,
		}
		gasTarget := CalcBaseFee(&config, parent)
		if gasTarget != nil {
			if test.baseFee != nil && gasTarget.Cmp(test.baseFee) != 0 {
				t.Errorf("test %d expected BaseFee %d got %d", i+1, test.baseFee.Uint64(), gasTarget.Uint64())
			}
			if test.baseFee == nil {
				t.Errorf("test %d expected nil BaseFee got %d", i+1, gasTarget.Uint64())
			}
		} else if test.baseFee != nil {
			t.Errorf("test %d expected BaseFee %d got nil", i+1, test.baseFee.Uint64())
		}
	}
}
