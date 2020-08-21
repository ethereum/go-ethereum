// Copyright 2015 The go-ethereum Authors
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

package core

import (
	"fmt"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/params"
)

// GasPool tracks the amount of gas available during execution of the transactions
// in a block. The zero value is a pool with zero gas available.
type GasPool uint64

// NewLegacyGasPool returns a GasPool filled to the legacy gas limit
func NewLegacyGasPool(chainConfig *params.ChainConfig, height, gasLimit *big.Int) *GasPool {
	eip1559GasTarget := misc.CalcEIP1559GasTarget(chainConfig, height, gasLimit)
	return new(GasPool).AddGas(gasLimit.Uint64() - eip1559GasTarget.Uint64())
}

// NewEIP1559GasPool returns a GasPool filled to the EIP1559 gas limit
func NewEIP1559GasPool(chainConfig *params.ChainConfig, height, gasLimit *big.Int) *GasPool {
	// EIP1559 gas limit is slack coefficient * EIP1559GasTarget
	eip1559GasTarget := misc.CalcEIP1559GasTarget(chainConfig, height, gasLimit)
	return new(GasPool).AddGas(chainConfig.EIP1559.EIP1559SlackCoefficient * eip1559GasTarget.Uint64())
}

// AddGas makes gas available for execution.
func (gp *GasPool) AddGas(amount uint64) *GasPool {
	if uint64(*gp) > math.MaxUint64-amount {
		panic("gas pool pushed above uint64")
	}
	*(*uint64)(gp) += amount
	return gp
}

// SubGas deducts the given amount from the pool if enough gas is
// available and returns an error otherwise.
func (gp *GasPool) SubGas(amount uint64) error {
	if uint64(*gp) < amount {
		return ErrGasLimitReached
	}
	*(*uint64)(gp) -= amount
	return nil
}

// Gas returns the amount of gas remaining in the pool.
func (gp *GasPool) Gas() uint64 {
	return uint64(*gp)
}

func (gp *GasPool) String() string {
	return fmt.Sprintf("%d", *gp)
}
