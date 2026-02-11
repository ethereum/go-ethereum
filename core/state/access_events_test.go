// Copyright 2021 The go-ethereum Authors
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

package state

import (
	"math"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

var (
	testAddr  [20]byte
	testAddr2 [20]byte
)

func init() {
	for i := byte(0); i < 20; i++ {
		testAddr[i] = i
		testAddr[2] = 2 * i
	}
}

func TestAccountHeaderGas(t *testing.T) {
	ae := NewAccessEvents()

	// Check cold read cost
	gas := ae.BasicDataGas(testAddr, false, math.MaxUint64, false)
	if want := params.WitnessBranchReadCost + params.WitnessChunkReadCost; gas != want {
		t.Fatalf("incorrect gas computed, got %d, want %d", gas, want)
	}

	// Check warm read cost
	gas = ae.BasicDataGas(testAddr, false, math.MaxUint64, false)
	if gas != 0 {
		t.Fatalf("incorrect gas computed, got %d, want %d", gas, 0)
	}

	// Check cold read costs in the same group no longer incur the branch read cost
	gas = ae.CodeHashGas(testAddr, false, math.MaxUint64, false)
	if gas != params.WitnessChunkReadCost {
		t.Fatalf("incorrect gas computed, got %d, want %d", gas, params.WitnessChunkReadCost)
	}

	// Check cold write cost
	gas = ae.BasicDataGas(testAddr, true, math.MaxUint64, false)
	if want := params.WitnessBranchWriteCost + params.WitnessChunkWriteCost; gas != want {
		t.Fatalf("incorrect gas computed, got %d, want %d", gas, want)
	}

	// Check warm write cost
	gas = ae.BasicDataGas(testAddr, true, math.MaxUint64, false)
	if gas != 0 {
		t.Fatalf("incorrect gas computed, got %d, want %d", gas, 0)
	}

	// Check a write without a read charges both read and write costs
	gas = ae.BasicDataGas(testAddr2, true, math.MaxUint64, false)
	if want := params.WitnessBranchReadCost + params.WitnessBranchWriteCost + params.WitnessChunkWriteCost + params.WitnessChunkReadCost; gas != want {
		t.Fatalf("incorrect gas computed, got %d, want %d", gas, want)
	}

	// Check that a write followed by a read charges nothing
	gas = ae.BasicDataGas(testAddr2, false, math.MaxUint64, false)
	if gas != 0 {
		t.Fatalf("incorrect gas computed, got %d, want %d", gas, 0)
	}

	// Check that reading a slot from the account header only charges the
	// chunk read cost.
	gas = ae.SlotGas(testAddr, common.Hash{}, false, math.MaxUint64, false)
	if gas != params.WitnessChunkReadCost {
		t.Fatalf("incorrect gas computed, got %d, want %d", gas, params.WitnessChunkReadCost)
	}
}

// TestContractCreateInitGas checks that the gas cost of contract creation is correctly
// calculated.
func TestContractCreateInitGas(t *testing.T) {
	ae := NewAccessEvents()

	var testAddr [20]byte
	for i := byte(0); i < 20; i++ {
		testAddr[i] = i
	}

	// Check cold read cost, without a value
	gas, _ := ae.ContractCreateInitGas(testAddr, math.MaxUint64)
	if want := params.WitnessBranchWriteCost + params.WitnessBranchReadCost + 2*params.WitnessChunkWriteCost + 2*params.WitnessChunkReadCost; gas != want {
		t.Fatalf("incorrect gas computed, got %d, want %d", gas, want)
	}

	// Check warm read cost
	gas, _ = ae.ContractCreateInitGas(testAddr, math.MaxUint64)
	if gas != 0 {
		t.Fatalf("incorrect gas computed, got %d, want %d", gas, 0)
	}
}

// TestWarmCostOOG checks that BasicDataGas and CodeHashGas return the full
// warm cost even when the available gas is insufficient. This is a regression
// test: previously these functions returned availableGas instead of the warm
// cost, which prevented the EVM interpreter from detecting out-of-gas.
func TestWarmCostOOG(t *testing.T) {
	ae := NewAccessEvents()
	insufficientGas := params.WarmStorageReadCostEIP2929 - 1

	// Warm up the address
	ae.BasicDataGas(testAddr, false, math.MaxUint64, false)
	ae.CodeHashGas(testAddr, false, math.MaxUint64, false)

	// BasicDataGas must return the full warm cost, not the available gas
	gas := ae.BasicDataGas(testAddr, false, insufficientGas, true)
	if gas != params.WarmStorageReadCostEIP2929 {
		t.Fatalf("BasicDataGas: got %d, want %d", gas, params.WarmStorageReadCostEIP2929)
	}

	// CodeHashGas must return the full warm cost, not the available gas
	gas = ae.CodeHashGas(testAddr, false, insufficientGas, true)
	if gas != params.WarmStorageReadCostEIP2929 {
		t.Fatalf("CodeHashGas: got %d, want %d", gas, params.WarmStorageReadCostEIP2929)
	}
}

// TestMessageCallGas checks that the gas cost of message calls is correctly
// calculated.
func TestMessageCallGas(t *testing.T) {
	ae := NewAccessEvents()

	// Check cold read cost, without a value
	gas := ae.MessageCallGas(testAddr, math.MaxUint64)
	if want := params.WitnessBranchReadCost + params.WitnessChunkReadCost; gas != want {
		t.Fatalf("incorrect gas computed, got %d, want %d", gas, want)
	}

	// Check that reading the basic data and code hash of the same account does not incur the branch read cost
	gas = ae.BasicDataGas(testAddr, false, math.MaxUint64, false)
	if gas != 0 {
		t.Fatalf("incorrect gas computed, got %d, want %d", gas, 0)
	}
	gas = ae.CodeHashGas(testAddr, false, math.MaxUint64, false)
	if gas != params.WitnessChunkReadCost {
		t.Fatalf("incorrect gas computed, got %d, want %d", gas, params.WitnessChunkReadCost)
	}

	// Check warm read cost
	gas = ae.MessageCallGas(testAddr, math.MaxUint64)
	if gas != params.WarmStorageReadCostEIP2929 {
		t.Fatalf("incorrect gas computed, got %d, want %d", gas, params.WarmStorageReadCostEIP2929)
	}
}
