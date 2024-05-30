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
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie/utils"
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
	ae := NewAccessEvents(utils.NewPointCache(1024))

	// Check cold read cost
	gas := ae.VersionGas(testAddr, false)
	if gas != params.WitnessBranchReadCost+params.WitnessChunkReadCost {
		t.Fatalf("incorrect gas computed, got %d, want %d", gas, params.WitnessBranchReadCost+params.WitnessChunkReadCost)
	}

	// Check warm read cost
	gas = ae.VersionGas(testAddr, false)
	if gas != 0 {
		t.Fatalf("incorrect gas computed, got %d, want %d", gas, 0)
	}

	// Check cold read costs in the same group no longer incur the branch read cost
	gas = ae.BalanceGas(testAddr, false)
	if gas != params.WitnessChunkReadCost {
		t.Fatalf("incorrect gas computed, got %d, want %d", gas, params.WitnessChunkReadCost)
	}
	gas = ae.NonceGas(testAddr, false)
	if gas != params.WitnessChunkReadCost {
		t.Fatalf("incorrect gas computed, got %d, want %d", gas, params.WitnessChunkReadCost)
	}
	gas = ae.CodeSizeGas(testAddr, false)
	if gas != params.WitnessChunkReadCost {
		t.Fatalf("incorrect gas computed, got %d, want %d", gas, params.WitnessChunkReadCost)
	}
	gas = ae.CodeHashGas(testAddr, false)
	if gas != params.WitnessChunkReadCost {
		t.Fatalf("incorrect gas computed, got %d, want %d", gas, params.WitnessChunkReadCost)
	}

	// Check cold write cost
	gas = ae.VersionGas(testAddr, true)
	if gas != params.WitnessBranchWriteCost+params.WitnessChunkWriteCost {
		t.Fatalf("incorrect gas computed, got %d, want %d", gas, params.WitnessBranchReadCost+params.WitnessBranchWriteCost)
	}

	// Check warm write cost
	gas = ae.VersionGas(testAddr, true)
	if gas != 0 {
		t.Fatalf("incorrect gas computed, got %d, want %d", gas, 0)
	}

	// Check a write without a read charges both read and write costs
	gas = ae.BalanceGas(testAddr2, true)
	if gas != params.WitnessBranchReadCost+params.WitnessBranchWriteCost+params.WitnessChunkWriteCost+params.WitnessChunkReadCost {
		t.Fatalf("incorrect gas computed, got %d, want %d", gas, params.WitnessBranchReadCost+params.WitnessBranchWriteCost+params.WitnessChunkWriteCost+params.WitnessChunkReadCost)
	}

	// Check that a write followed by a read charges nothing
	gas = ae.BalanceGas(testAddr2, false)
	if gas != 0 {
		t.Fatalf("incorrect gas computed, got %d, want %d", gas, 0)
	}

	// Check that reading a slot from the account header only charges the
	// chunk read cost.
	gas = ae.SlotGas(testAddr, common.Hash{}, false)
	if gas != params.WitnessChunkReadCost {
		t.Fatalf("incorrect gas computed, got %d, want %d", gas, params.WitnessChunkReadCost)
	}
}

// TestContractCreateInitGas checks that the gas cost of contract creation is correctly
// calculated.
func TestContractCreateInitGas(t *testing.T) {
	ae := NewAccessEvents(utils.NewPointCache(1024))

	var testAddr [20]byte
	for i := byte(0); i < 20; i++ {
		testAddr[i] = i
	}

	// Check cold read cost, without a value
	gas := ae.ContractCreateInitGas(testAddr, false)
	if gas != params.WitnessBranchWriteCost+params.WitnessBranchReadCost+params.WitnessChunkWriteCost*2+params.WitnessChunkReadCost*2 {
		t.Fatalf("incorrect gas computed, got %d, want %d", gas, params.WitnessBranchWriteCost+params.WitnessBranchReadCost+params.WitnessChunkWriteCost*3)
	}

	// Check warm read cost
	gas = ae.ContractCreateInitGas(testAddr, false)
	if gas != 0 {
		t.Fatalf("incorrect gas computed, got %d, want %d", gas, 0)
	}
}

// TestMessageCallGas checks that the gas cost of message calls is correctly
// calculated.
func TestMessageCallGas(t *testing.T) {
	ae := NewAccessEvents(utils.NewPointCache(1024))

	// Check cold read cost, without a value
	gas := ae.MessageCallGas(testAddr)
	if gas != params.WitnessBranchReadCost+params.WitnessChunkReadCost*2 {
		t.Fatalf("incorrect gas computed, got %d, want %d", gas, params.WitnessBranchReadCost+params.WitnessChunkReadCost*2)
	}

	// Check that reading the version and code size of the same account does not incur the branch read cost
	gas = ae.VersionGas(testAddr, false)
	if gas != 0 {
		t.Fatalf("incorrect gas computed, got %d, want %d", gas, 0)
	}
	gas = ae.CodeSizeGas(testAddr, false)
	if gas != 0 {
		t.Fatalf("incorrect gas computed, got %d, want %d", gas, 0)
	}

	// Check warm read cost
	gas = ae.MessageCallGas(testAddr)
	if gas != 0 {
		t.Fatalf("incorrect gas computed, got %d, want %d", gas, 0)
	}
}
