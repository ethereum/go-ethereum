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

package vm_test

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

func TestTracingStateReadsDoNotAffectBlockAccessList(t *testing.T) {
	db := state.NewDatabaseForTesting()
	statedb, err := state.New(types.EmptyRootHash, db)
	if err != nil {
		t.Fatal(err)
	}
	config := *params.MergedTestChainConfig
	config.AmsterdamTime = new(uint64)
	blockContext := vm.BlockContext{BlockNumber: new(big.Int)}
	rules := config.Rules(blockContext.BlockNumber, true, 0)
	addr := common.HexToAddress("0x1234")
	key := common.HexToHash("0x5678")
	wantBalance := uint256.NewInt(123)
	wantCode := []byte{byte(vm.PUSH1), 0x01, byte(vm.STOP)}
	wantState := common.HexToHash("0x9abc")
	statedb.SetBalance(addr, wantBalance, tracing.BalanceChangeUnspecified)
	statedb.SetNonce(addr, 7, tracing.NonceChangeUnspecified)
	statedb.SetCode(addr, wantCode, tracing.CodeChangeUnspecified)
	statedb.SetState(addr, key, wantState)
	root, err := statedb.Commit(rules, 0)
	if err != nil {
		t.Fatal(err)
	}
	statedb, err = state.New(root, db)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name  string
		state vm.StateDB
	}{
		{name: "plain", state: statedb},
		{name: "hooked", state: state.NewHookedState(statedb, new(tracing.Hooks))},
	} {
		t.Run(test.name, func(t *testing.T) {
			statedb.Prepare(rules, common.Address{}, common.Address{}, nil, nil, nil)
			evm := vm.NewEVM(blockContext, test.state, &config, vm.Config{})
			defer evm.Release()

			tracingState := evm.GetVMContext().StateDB
			if balance := tracingState.GetBalance(addr); balance.Cmp(wantBalance) != 0 {
				t.Fatalf("balance mismatch: have %v, want %v", balance, wantBalance)
			}
			if nonce := tracingState.GetNonce(addr); nonce != 7 {
				t.Fatalf("nonce mismatch: have %d, want 7", nonce)
			}
			if code := tracingState.GetCode(addr); !bytes.Equal(code, wantCode) {
				t.Fatalf("code mismatch: have %x, want %x", code, wantCode)
			}
			if codeHash := tracingState.GetCodeHash(addr); codeHash != crypto.Keccak256Hash(wantCode) {
				t.Fatalf("code hash mismatch: have %x", codeHash)
			}
			if value := tracingState.GetState(addr, key); value != wantState {
				t.Fatalf("state mismatch: have %x, want %x", value, wantState)
			}
			if !tracingState.Exist(addr) {
				t.Fatal("account does not exist")
			}

			accessList := statedb.Finalise(rules)
			if len(accessList.Accounts) != 0 {
				t.Fatalf("tracer reads changed block access list: %s", accessList.PrettyPrint())
			}
		})
	}
}

func TestSetStateDBUpdatesTracingState(t *testing.T) {
	first, err := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	if err != nil {
		t.Fatal(err)
	}
	second, err := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	if err != nil {
		t.Fatal(err)
	}
	addr := common.HexToAddress("0x1234")
	wantBalance := uint256.NewInt(123)
	second.SetBalance(addr, wantBalance, tracing.BalanceChangeUnspecified)

	evm := vm.NewEVM(vm.BlockContext{BlockNumber: new(big.Int)}, first, params.AllEthashProtocolChanges, vm.Config{})
	defer evm.Release()
	evm.SetStateDB(second)

	if balance := evm.GetVMContext().StateDB.GetBalance(addr); balance.Cmp(wantBalance) != 0 {
		t.Fatalf("balance mismatch: have %v, want %v", balance, wantBalance)
	}
}
