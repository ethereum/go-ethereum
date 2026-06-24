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

package core

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// TestEIP2780Intrinsic checks the intrinsic-gas decomposition.
func TestEIP2780Intrinsic(t *testing.T) {
	var (
		from = common.HexToAddress("0x1111111111111111111111111111111111111111")
		to   = common.HexToAddress("0x2222222222222222222222222222222222222222")
	)
	cases := []struct {
		name  string
		to    *common.Address
		value *uint256.Int
		want  vm.GasCosts
	}{
		{
			name:  "self-transfer",
			to:    &from,
			value: uint256.NewInt(1),
			want:  vm.GasCosts{RegularGas: params.TxBaseCost2780}, // 12,000
		},
		{
			name:  "self-transfer/zero-value",
			to:    &from,
			value: uint256.NewInt(0),
			want:  vm.GasCosts{RegularGas: params.TxBaseCost2780}, // 12,000
		},
		{
			name:  "zero-value call",
			to:    &to,
			value: uint256.NewInt(0),
			// TxBaseCost + ColdAccountAccess = 15,000
			want: vm.GasCosts{RegularGas: params.TxBaseCost2780 + params.ColdAccountAccess2780},
		},
		{
			name:  "value transfer to existing EOA",
			to:    &to,
			value: uint256.NewInt(1),
			// TxBaseCost + ColdAccountAccess + TxValueCost + TransferLogCost = 21,000
			want: vm.GasCosts{RegularGas: params.TxBaseCost2780 + params.ColdAccountAccess2780 +
				params.TxValueCost2780 + params.TransferLogCost2780},
		},
		{
			name:  "contract creation, value = 0",
			to:    nil,
			value: uint256.NewInt(0),
			// TxBaseCost + CreateAccess = 23,000 regular, plus one account creation in state.
			want: vm.GasCosts{
				RegularGas: params.TxBaseCost2780 + params.CreateAccess2780,
				StateGas:   params.AccountCreationSize * params.CostPerStateByte,
			},
		},
		{
			name:  "contract creation, value > 0",
			to:    nil,
			value: uint256.NewInt(1),
			// TxBaseCost + CreateAccess + TransferLogCost = 24,756 regular, plus account creation.
			want: vm.GasCosts{
				RegularGas: params.TxBaseCost2780 + params.CreateAccess2780 + params.TransferLogCost2780,
				StateGas:   params.AccountCreationSize * params.CostPerStateByte,
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := IntrinsicGas(nil, nil, nil, from, tc.to, tc.value, rules8037, params.CostPerStateByte)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("gas mismatch: got %+v, want %+v", got, tc.want)
			}
		})
	}
}

// TestEIP2780Gas checks every "Transaction reference case" in
// the EIP-2780 specification end-to-end, asserting the two-dimensional charge
// (intrinsic + top-level + execution) recorded in the block gas pool.
func TestEIP2780Gas(t *testing.T) {
	const (
		cold     = params.ColdAccountAccess2780
		base     = params.TxBaseCost2780
		valueCst = params.TxValueCost2780 + params.TransferLogCost2780
	)
	var (
		existingEOA  = common.HexToAddress("0xe0a0000000000000000000000000000000000001")
		stopContract = common.HexToAddress("0xc0de000000000000000000000000000000000001")
		delegated    = common.HexToAddress("0xde1e000000000000000000000000000000000001")
		emptyTarget  = common.HexToAddress("0x7a76000000000000000000000000000000000001") // never allocated
		freshEOA     = common.HexToAddress("0xbeef000000000000000000000000000000000001") // never allocated
	)
	// Shared world: a funded EOA, a STOP contract and an account delegated to a
	// non-existent (codeless) target. The delegation target is intentionally
	// absent so resolving it executes no code.
	base7702 := types.GenesisAlloc{
		existingEOA:  {Balance: big.NewInt(1)},
		stopContract: {Code: []byte{0x00}}, // STOP
		delegated:    {Code: types.AddressToDelegation(emptyTarget)},
	}
	// valueCreateTx builds a contract-creation transaction carrying value.
	valueCreateTx := func(value int64) *types.Transaction {
		return types.MustSignNewTx(senderKey, signer8037, &types.DynamicFeeTx{
			ChainID: cfg8037.ChainID, Nonce: 0, To: nil, Value: big.NewInt(value),
			Gas: 300_000, GasFeeCap: big.NewInt(0), GasTipCap: big.NewInt(0),
		})
	}

	cases := []struct {
		name                   string
		tx                     *types.Transaction
		wantRegular, wantState uint64
	}{
		// case 1: ETH transfer to self.
		{"self-transfer", callTx(0, senderAddr, 1, 100_000, nil), base, 0},
		// case 2: no-transfer to an existing EOA.
		{"zero-value/eoa", callTx(0, existingEOA, 0, 100_000, nil), base + cold, 0},
		// case 3: no-transfer to a contract.
		{"zero-value/contract", callTx(0, stopContract, 0, 100_000, nil), base + cold, 0},
		// case 4: ETH transfer to an existing EOA.
		{"value/eoa", callTx(0, existingEOA, 1, 100_000, nil), base + cold + valueCst, 0},
		// case 5: ETH transfer to a contract.
		{"value/contract", callTx(0, stopContract, 1, 100_000, nil), base + cold + valueCst, 0},
		// case 6: no-transfer to a 7702-delegated account.
		{"zero-value/delegated", callTx(0, delegated, 0, 100_000, nil), base + 2*cold, 0},
		// case 7: ETH transfer to a 7702-delegated account (no new-account charge).
		{"value/delegated", callTx(0, delegated, 1, 100_000, nil), base + 2*cold + valueCst, 0},
		// case 8: ETH transfer creating a new account.
		{"value/new-account", callTx(0, freshEOA, 1, 300_000, nil), base + cold + valueCst, newAccountState},
		// case 9: contract-creation transaction, value = 0.
		{"create/zero-value", createTx(0, 300_000, nil), base + params.CreateAccess2780, newAccountState},
		// case 10: contract-creation transaction, value > 0.
		{"create/value", valueCreateTx(1), base + params.CreateAccess2780 + params.TransferLogCost2780, newAccountState},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, gp, err := applyMsg(t, mkState(senderAlloc(base7702)), tc.tx)
			if err != nil {
				t.Fatalf("consensus error: %v", err)
			}
			if res.Err != nil {
				t.Fatalf("execution failed: %v", res.Err)
			}
			if gp.cumulativeRegular != tc.wantRegular {
				t.Errorf("regular gas = %d, want %d", gp.cumulativeRegular, tc.wantRegular)
			}
			if gp.cumulativeState != tc.wantState {
				t.Errorf("state gas = %d, want %d", gp.cumulativeState, tc.wantState)
			}
		})
	}
}

// TestEIP2780NewAccountFunded verifies that a value transfer creating a new
// account both materializes and funds the recipient.
func TestEIP2780NewAccountFunded(t *testing.T) {
	fresh := common.HexToAddress("0xbeef000000000000000000000000000000000002")
	sdb := mkState(senderAlloc(nil))
	if _, _, err := applyMsg(t, sdb, callTx(0, fresh, 1, 300_000, nil)); err != nil {
		t.Fatal(err)
	}
	if !sdb.Exist(fresh) || sdb.GetBalance(fresh).Cmp(uint256.NewInt(1)) != 0 {
		t.Fatalf("recipient not funded: exist=%v balance=%v", sdb.Exist(fresh), sdb.GetBalance(fresh))
	}
}

// TestEIP2780InsufficientGasForCallCharge verifies that a value transfer
// creating a new account is rejected when the gas limit only covers the 21,000
// intrinsic base and not the additional new-account state gas charged before the
// call executes.
func TestEIP2780InsufficientGasForCallCharge(t *testing.T) {
	fresh := common.HexToAddress("0xbeef000000000000000000000000000000000003")
	sdb := mkState(senderAlloc(nil))
	_, _, err := applyMsg(t, sdb, callTx(0, fresh, 1, 21_000, nil))
	if !errors.Is(err, ErrEIP2780CallCharge) {
		t.Fatalf("expected ErrEIP2780CallCharge, got %v", err)
	}
	if sdb.Exist(fresh) {
		t.Fatal("recipient should not be created when the call charge cannot be paid")
	}
}
