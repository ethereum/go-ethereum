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

package simulated

import (
	"context"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

// Tests that the simulator starts with the initial gas limit in the genesis block,
// and that it keeps the same target value.
func TestWithBlockGasLimitOption(t *testing.T) {
	// Construct a simulator, targeting a different gas limit
	sim := NewBackend(types.GenesisAlloc{}, WithBlockGasLimit(12_345_678))
	defer sim.Close()

	client := sim.Client()
	genesis, err := client.BlockByNumber(context.Background(), big.NewInt(0))
	if err != nil {
		t.Fatalf("failed to retrieve genesis block: %v", err)
	}
	if genesis.GasLimit() != 12_345_678 {
		t.Errorf("genesis gas limit mismatch: have %v, want %v", genesis.GasLimit(), 12_345_678)
	}
	// Produce a number of blocks and verify the locked in gas target
	sim.Commit()
	head, err := client.BlockByNumber(context.Background(), big.NewInt(1))
	if err != nil {
		t.Fatalf("failed to retrieve head block: %v", err)
	}
	if head.GasLimit() != 12_345_678 {
		t.Errorf("head gas limit mismatch: have %v, want %v", head.GasLimit(), 12_345_678)
	}
}

// Tests that the simulator honors the RPC call caps set by the options.
func TestWithCallGasLimitOption(t *testing.T) {
	// Construct a simulator, targeting a different gas limit
	sim := NewBackend(types.GenesisAlloc{
		testAddr: {Balance: big.NewInt(10000000000000000)},
	}, WithCallGasLimit(params.TxGas-1))
	defer sim.Close()

	client := sim.Client()
	_, err := client.CallContract(context.Background(), ethereum.CallMsg{
		From: testAddr,
		To:   &testAddr,
		Gas:  21000,
	}, nil)
	if !strings.Contains(err.Error(), core.ErrIntrinsicGas.Error()) {
		t.Fatalf("error mismatch: have %v, want %v", err, core.ErrIntrinsicGas)
	}
}
