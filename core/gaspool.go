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
)

// GasPool tracks the amount of gas available during execution of the transactions
// in a block. The zero value is a pool with zero gas available.
type GasPool struct {
	gas, dataGas uint64
}

// AddGas makes gas available for execution.
func (gp *GasPool) AddGas(amount uint64) *GasPool {
	if gp.gas > math.MaxUint64-amount {
		panic("gas pool pushed above uint64")
	}
	gp.gas += amount
	return gp
}

// SubGas deducts the given amount from the pool if enough gas is
// available and returns an error otherwise.
func (gp *GasPool) SubGas(amount uint64) error {
	if gp.gas < amount {
		return ErrGasLimitReached
	}
	gp.gas -= amount
	return nil
}

// Gas returns the amount of gas remaining in the pool.
func (gp *GasPool) Gas() uint64 {
	return gp.gas
}

// AddDataGas makes data gas available for execution.
func (gp *GasPool) AddDataGas(amount uint64) *GasPool {
	if gp.dataGas > math.MaxUint64-amount {
		panic("data gas pool pushed above uint64")
	}
	gp.dataGas += amount
	return gp
}

// SubDataGas deducts the given amount from the pool if enough data gas is available and returns an
// error otherwise.
func (gp *GasPool) SubDataGas(amount uint64) error {
	if gp.dataGas < amount {
		return ErrDataGasLimitReached
	}
	gp.dataGas -= amount
	return nil
}

// DataGas returns the amount of data gas remaining in the pool.
func (gp *GasPool) DataGas() uint64 {
	return gp.dataGas
}

func (gp *GasPool) String() string {
	return fmt.Sprintf("gas: %d, data_gas: %d", gp.gas, gp.dataGas)
}
