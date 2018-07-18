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

package core

import (
	"math"
	"testing"
)

func TestGasPoolAdd(t *testing.T) {
	var gasPool GasPool

	gasPool = math.MaxUint64 - 1
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected error, got nil")
			}
		}()
		gasPool.AddGas(2)
	}()

	gasPool = 1
	gasPool = *gasPool.AddGas(1)
	if gasPool != 2 {
		t.Errorf("expected %v, got %v", 2, gasPool)
	}
}

func TestGasPoolSub(t *testing.T) {
	var gasPool GasPool

	gasPool = 10
	err := gasPool.SubGas(11)
	if err != ErrGasLimitReached {
		t.Errorf("expected err: %v, got %v", ErrGasLimitReached, err)
	}

	gasPool = 10
	err = gasPool.SubGas(1)
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	if gasPool != 9 {
		t.Errorf("expected %v, got %v", 9, gasPool)
	}
}

func TestGasPoolGas(t *testing.T) {
	var gasPool GasPool = 100
	if gasPool.Gas() != 100 {
		t.Errorf("expected %v, got %v", 100, gasPool.Gas())
	}
}

func TestGasPoolString(t *testing.T) {
	var gasPool GasPool = 123
	if gasPool.String() != "123" {
		t.Errorf("expected %v, got %v", "123", gasPool.String())
	}
}
