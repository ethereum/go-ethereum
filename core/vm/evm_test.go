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

package vm

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

func amsterdamTestChainConfig() *params.ChainConfig {
	cfg := *params.MergedTestChainConfig
	cfg.AmsterdamTime = new(uint64)
	return &cfg
}

func TestCreateAccountCreationStateGasOOGReverts(t *testing.T) {
	statedb, err := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	if err != nil {
		t.Fatal(err)
	}
	caller := common.HexToAddress("0x1111")
	statedb.CreateAccount(caller)

	cfg := amsterdamTestChainConfig()
	random := common.Hash{}
	evm := NewEVM(BlockContext{
		CanTransfer:      func(StateDB, common.Address, *uint256.Int) bool { return true },
		Transfer:         func(StateDB, common.Address, common.Address, *uint256.Int, *params.Rules) {},
		BlockNumber:      big.NewInt(0),
		Time:             0,
		Random:           &random,
		CostPerStateByte: params.CostPerStateByte,
	}, statedb, cfg, Config{})
	evm.depth = 1 // nested CREATE; account-creation state gas is charged here, not in intrinsic gas

	contractAddr := crypto.CreateAddress(caller, 0)
	if statedb.Exist(contractAddr) {
		t.Fatal("contract address should not exist before create")
	}

	creationCost := uint64(params.AccountCreationSize * params.CostPerStateByte)
	gas := NewGasBudget(creationCost-1, 0)

	_, addr, _, err := evm.Create(caller, nil, gas, new(uint256.Int))
	if !errors.Is(err, ErrOutOfGas) {
		t.Fatalf("expected ErrOutOfGas, got %v (addr %v)", err, addr)
	}
	if statedb.Exist(contractAddr) {
		t.Fatal("contract account was not rolled back after account-creation state gas OOG")
	}
	if statedb.GetNonce(caller) != 1 {
		t.Fatalf("caller nonce should be incremented, got %d", statedb.GetNonce(caller))
	}
}
