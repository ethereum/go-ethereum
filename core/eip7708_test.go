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
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// EIP-7708 transfer-log tests. The older TestEthTransferLogs keeps its
// historical end-to-end Solidity fixture; these tests use small bytecode
// programs to pin the normative edge cases independently.

func transferLogs7708(sdb *state.StateDB) []*types.Log {
	return sdb.GetLogs(common.Hash{}, 0, common.Hash{}, 0)
}

func assertTransfer7708(t *testing.T, log *types.Log, from, to common.Address, value uint64) {
	t.Helper()
	want := types.EthTransferLog(from, to, uint256.NewInt(value))
	if log.Address != want.Address || len(log.Topics) != 3 || log.Topics[0] != want.Topics[0] ||
		log.Topics[1] != want.Topics[1] || log.Topics[2] != want.Topics[2] || string(log.Data) != string(want.Data) {
		t.Fatalf("transfer log = %+v, want %+v", log, want)
	}
}

func callCode7708(target common.Address, value byte, suffix []byte) []byte {
	return callOpcode7708(target, value, 0xf1, suffix)
}

func callOpcode7708(target common.Address, value, opcode byte, suffix []byte) []byte {
	// Optional LOG0, then CALL(gas=0xffff, to=target, value=value, empty input
	// and output), discard the success flag, and run suffix.
	code := []byte{0x60, 0x00, 0x60, 0x00, 0xa0}
	code = append(code,
		0x60, 0x00, 0x60, 0x00, 0x60, 0x00, 0x60, 0x00, // output/input sizes and offsets
		0x60, value, 0x73)
	code = append(code, target.Bytes()...)
	code = append(code, 0x61, 0xff, 0xff, opcode, 0x50)
	return append(code, suffix...)
}

func delegateCode7708(target common.Address) []byte {
	// DELEGATECALL(gas=0xffff, to=target, empty input and output), followed by
	// STOP. Value is inherited from the outer call's executing context.
	code := []byte{0x60, 0x00, 0x60, 0x00, 0x60, 0x00, 0x60, 0x00, 0x73}
	code = append(code, target.Bytes()...)
	return append(code, 0x61, 0xff, 0xff, 0xf4, 0x50, 0x00)
}

func createTx7708(value uint64, initcode []byte) *types.Transaction {
	return types.MustSignNewTx(senderKey, signer8037, &types.DynamicFeeTx{
		ChainID: cfg8037.ChainID, Nonce: 0, Value: new(big.Int).SetUint64(value),
		Gas: 500_000, GasFeeCap: new(big.Int), GasTipCap: new(big.Int), Data: initcode,
	})
}

// TestEIP7708Transactions covers ordinary transaction value transfers across
// the principal transaction encodings, including a delegated recipient.
func TestEIP7708Transactions(t *testing.T) {
	recipient := common.HexToAddress("0x7708000000000000000000000000000000000001")
	auth, _ := signAuth(t, authKeyA, delegate8037, 0)
	cases := []struct {
		name string
		tx   func() *types.Transaction
	}{
		{
			"legacy",
			func() *types.Transaction {
				return types.MustSignNewTx(senderKey, signer8037, &types.LegacyTx{
					Nonce: 0, To: &recipient, Value: big.NewInt(1), Gas: 100_000, GasPrice: new(big.Int),
				})
			},
		},
		{
			"access-list",
			func() *types.Transaction {
				return types.MustSignNewTx(senderKey, signer8037, &types.AccessListTx{
					ChainID: cfg8037.ChainID, Nonce: 0, To: &recipient, Value: big.NewInt(1), Gas: 100_000, GasPrice: new(big.Int),
				})
			},
		},
		{"dynamic", func() *types.Transaction { return callTx(0, recipient, 1, 100_000, nil) }},
		{"set-code", func() *types.Transaction {
			return setCodeTxGas(0, recipient, 1, 1_000_000, []types.SetCodeAuthorization{auth})
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sdb := mkState(senderAlloc(types.GenesisAlloc{recipient: {Balance: big.NewInt(1)}}))
			res, _, err := applyMsg(t, sdb, tc.tx())
			if err != nil || res.Err != nil {
				t.Fatalf("result=%v err=%v", res, err)
			}
			logs := transferLogs7708(sdb)
			if len(logs) != 1 {
				t.Fatalf("logs = %+v, want one", logs)
			}
			assertTransfer7708(t, logs[0], senderAddr, recipient, 1)
		})
	}

	delegated := common.HexToAddress("0x7708000000000000000000000000000000000002")
	sdb := mkState(senderAlloc(types.GenesisAlloc{
		delegated:    {Code: types.AddressToDelegation(delegate8037)},
		delegate8037: {Code: []byte{0x00}},
	}))
	res, _, err := applyMsg(t, sdb, callTx(0, delegated, 1, 100_000, nil))
	if err != nil || res.Err != nil {
		t.Fatalf("delegated result=%v err=%v", res, err)
	}
	logs := transferLogs7708(sdb)
	if len(logs) != 1 {
		t.Fatalf("delegated logs = %+v, want one", logs)
	}
	// The transfer is to the delegation account, never to its implementation.
	assertTransfer7708(t, logs[0], senderAddr, delegated, 1)
}

// TestEIP7708Special covers recipients that are frequently warmed or handled
// specially by the EVM, and checks that fee recipients do not produce logs.
func TestEIP7708Special(t *testing.T) {
	precompile := common.BytesToAddress([]byte{4})
	system := params.SystemAddress
	coinbase := common.HexToAddress("0x7708000000000000000000000000000000000003")
	cases := []struct {
		name     string
		to       common.Address
		coinbase bool
	}{
		{"precompile", precompile, false},
		{"system", system, false},
		{"coinbase", coinbase, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sdb := mkState(senderAlloc(nil))
			tx := callTx(0, tc.to, 1, 300_000, nil)
			var (
				res *ExecutionResult
				err error
			)
			if tc.coinbase {
				res, _, err = applyMsgCoinbase(t, sdb, tx, coinbase)
			} else {
				res, _, err = applyMsg(t, sdb, tx)
			}
			if err != nil || res.Err != nil {
				t.Fatalf("result=%v err=%v", res, err)
			}
			logs := transferLogs7708(sdb)
			if len(logs) != 1 {
				t.Fatalf("logs = %+v, want one transfer only", logs)
			}
			assertTransfer7708(t, logs[0], senderAddr, tc.to, 1)
		})
	}
}

// TestEIP7708Calls checks ordering, attribution, and rollback for value CALLs.
func TestEIP7708Calls(t *testing.T) {
	caller := common.HexToAddress("0x7708000000000000000000000000000000000010")
	callee := common.HexToAddress("0x7708000000000000000000000000000000000011")
	reverter := common.HexToAddress("0x7708000000000000000000000000000000000012")

	t.Run("order", func(t *testing.T) {
		sdb := mkState(senderAlloc(types.GenesisAlloc{
			caller: {Code: callCode7708(callee, 3, []byte{0x00})},
			callee: {Code: []byte{0x00}},
		}))
		res, _, err := applyMsg(t, sdb, callTx(0, caller, 10, 300_000, nil))
		if err != nil || res.Err != nil {
			t.Fatalf("result=%v err=%v", res, err)
		}
		logs := transferLogs7708(sdb)
		if len(logs) != 3 {
			t.Fatalf("logs = %+v, want tx transfer, LOG0, inner transfer", logs)
		}
		assertTransfer7708(t, logs[0], senderAddr, caller, 10)

		// Ordinary log emitted by contract
		if logs[1].Address != caller || len(logs[1].Topics) != 0 || len(logs[1].Data) != 0 {
			t.Fatalf("ordinary log = %+v, want caller LOG0", logs[1])
		}
		assertTransfer7708(t, logs[2], caller, callee, 3)
	})

	t.Run("inner-revert", func(t *testing.T) {
		sdb := mkState(senderAlloc(types.GenesisAlloc{
			caller:   {Code: callCode7708(reverter, 3, []byte{0x00})},
			reverter: {Code: []byte{0x60, 0x00, 0x60, 0x00, 0xfd}},
		}))
		res, _, err := applyMsg(t, sdb, callTx(0, caller, 10, 300_000, nil))
		if err != nil || res.Err != nil {
			t.Fatalf("result=%v err=%v", res, err)
		}
		logs := transferLogs7708(sdb)
		if len(logs) != 2 { // top-level transfer and the caller's LOG0
			t.Fatalf("logs = %+v, want no rolled-back inner transfer", logs)
		}
		assertTransfer7708(t, logs[0], senderAddr, caller, 10)
	})

	t.Run("outer-revert", func(t *testing.T) {
		sdb := mkState(senderAlloc(types.GenesisAlloc{
			caller: {Code: callCode7708(callee, 3, []byte{0x60, 0x00, 0x60, 0x00, 0xfd})},
			callee: {Code: []byte{0x00}},
		}))
		res, _, err := applyMsg(t, sdb, callTx(0, caller, 10, 300_000, nil))
		if err != nil || res.Err != vm.ErrExecutionReverted {
			t.Fatalf("result=%v err=%v, want execution revert", res, err)
		}
		if logs := transferLogs7708(sdb); len(logs) != 0 {
			t.Fatalf("reverted transaction retained logs: %+v", logs)
		}
	})

	t.Run("callcode", func(t *testing.T) {
		sdb := mkState(senderAlloc(types.GenesisAlloc{
			caller: {Code: callOpcode7708(callee, 3, 0xf2, []byte{0x00})},
			callee: {Code: []byte{0x00}},
		}))
		res, _, err := applyMsg(t, sdb, callTx(0, caller, 10, 300_000, nil))
		if err != nil || res.Err != nil {
			t.Fatalf("result=%v err=%v", res, err)
		}
		logs := transferLogs7708(sdb)
		if len(logs) != 2 {
			t.Fatalf("logs = %+v, want only top-level transfer and LOG0", logs)
		}
		assertTransfer7708(t, logs[0], senderAddr, caller, 10)
	})

	t.Run("delegatecall", func(t *testing.T) {
		implementation := common.HexToAddress("0x7708000000000000000000000000000000000013")
		sdb := mkState(senderAlloc(types.GenesisAlloc{
			caller:         {Code: delegateCode7708(implementation)},
			implementation: {Code: callCode7708(callee, 3, []byte{0x00})},
			callee:         {Code: []byte{0x00}},
		}))
		res, _, err := applyMsg(t, sdb, callTx(0, caller, 10, 300_000, nil))
		if err != nil || res.Err != nil {
			t.Fatalf("result=%v err=%v", res, err)
		}
		logs := transferLogs7708(sdb)
		if len(logs) != 3 {
			t.Fatalf("logs = %+v, want top-level transfer, delegated LOG0, inner transfer", logs)
		}
		assertTransfer7708(t, logs[0], senderAddr, caller, 10)
		if logs[1].Address != caller {
			t.Fatalf("delegated LOG0 address = %s, want executing context %s", logs[1].Address, caller)
		}
		assertTransfer7708(t, logs[2], caller, callee, 3)
	})
}

// TestEIP7708Create checks successful CREATE/CREATE2 endowments and that
// failing initcode rolls the transfer log back with the failed creation.
func TestEIP7708Create(t *testing.T) {
	t.Run("transaction", func(t *testing.T) {
		sdb := mkState(senderAlloc(nil))
		res, _, err := applyMsg(t, sdb, createTx7708(5, []byte{0x00}))
		if err != nil || res.Err != nil {
			t.Fatalf("result=%v err=%v", res, err)
		}
		logs := transferLogs7708(sdb)
		if len(logs) != 1 {
			t.Fatalf("logs = %+v, want one", logs)
		}
		assertTransfer7708(t, logs[0], senderAddr, crypto.CreateAddress(senderAddr, 0), 5)
	})

	t.Run("revert", func(t *testing.T) {
		sdb := mkState(senderAlloc(nil))
		res, _, err := applyMsg(t, sdb, createTx7708(5, []byte{0x60, 0x00, 0x60, 0x00, 0xfd}))
		if err != nil || res.Err != vm.ErrExecutionReverted {
			t.Fatalf("result=%v err=%v, want execution revert", res, err)
		}
		if logs := transferLogs7708(sdb); len(logs) != 0 {
			t.Fatalf("failed create retained logs: %+v", logs)
		}
	})

	t.Run("create2", func(t *testing.T) {
		sdb := mkState(senderAlloc(nil))
		evm := amsterdamCoreEVM(sdb)
		_, created, _, err := evm.Create2(senderAddr, []byte{0x00}, vm.NewGasBudget(500_000, 0), uint256.NewInt(5), new(uint256.Int))
		if err != nil {
			t.Fatal(err)
		}
		logs := transferLogs7708(sdb)
		if len(logs) != 1 {
			t.Fatalf("logs = %+v, want one", logs)
		}
		assertTransfer7708(t, logs[0], senderAddr, created, 5)
	})
}

// TestEIP7708Selfdestruct verifies the EIP-8246-compatible SELFDESTRUCT
// cases: different beneficiaries transfer and log, self beneficiaries do not.
func TestEIP7708Selfdestruct(t *testing.T) {
	contract := common.HexToAddress("0x7708000000000000000000000000000000000020")
	beneficiary := common.HexToAddress("0x7708000000000000000000000000000000000021")
	code := append([]byte{0x73}, beneficiary.Bytes()...)
	code = append(code, 0xff)
	sdb := mkState(senderAlloc(types.GenesisAlloc{contract: {Code: code}}))
	res, _, err := applyMsg(t, sdb, callTx(0, contract, 7, 300_000, nil))
	if err != nil || res.Err != nil {
		t.Fatalf("result=%v err=%v", res, err)
	}
	logs := transferLogs7708(sdb)
	if len(logs) != 2 {
		t.Fatalf("logs = %+v, want transaction and selfdestruct transfers", logs)
	}
	assertTransfer7708(t, logs[0], senderAddr, contract, 7)
	assertTransfer7708(t, logs[1], contract, beneficiary, 7)

	selfCode := append([]byte{0x73}, contract.Bytes()...)
	selfCode = append(selfCode, 0xff)
	sdb = mkState(senderAlloc(types.GenesisAlloc{contract: {Code: selfCode}}))
	res, _, err = applyMsg(t, sdb, callTx(0, contract, 7, 300_000, nil))
	if err != nil || res.Err != nil {
		t.Fatalf("self result=%v err=%v", res, err)
	}
	logs = transferLogs7708(sdb)
	if len(logs) != 1 {
		t.Fatalf("selfdestruct-to-self logs = %+v, want top-level transfer only", logs)
	}
	assertTransfer7708(t, logs[0], senderAddr, contract, 7)
}

// TestEIP7708Negative and TestEIP7708Transition pin the no-log cases and the
// activation guard independently of transaction construction.
func TestEIP7708Negative(t *testing.T) {
	recipient := common.HexToAddress("0x7708000000000000000000000000000000000030")
	for _, tc := range []struct {
		name  string
		to    common.Address
		value int64
	}{
		{"zero", recipient, 0},
		{"self", senderAddr, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sdb := mkState(senderAlloc(nil))
			res, _, err := applyMsg(t, sdb, callTx(0, tc.to, tc.value, 100_000, nil))
			if err != nil || res.Err != nil {
				t.Fatalf("result=%v err=%v", res, err)
			}
			if logs := transferLogs7708(sdb); len(logs) != 0 {
				t.Fatalf("unexpected logs: %+v", logs)
			}
		})
	}
}

func TestEIP7708Transition(t *testing.T) {
	from := common.HexToAddress("0x7708000000000000000000000000000000000040")
	to := common.HexToAddress("0x7708000000000000000000000000000000000041")
	sdb := mkState(types.GenesisAlloc{from: {Balance: big.NewInt(2)}})
	Transfer(sdb, from, to, uint256.NewInt(1), &params.Rules{})
	if logs := transferLogs7708(sdb); len(logs) != 0 {
		t.Fatalf("pre-Amsterdam logs = %+v, want none", logs)
	}
	Transfer(sdb, from, to, uint256.NewInt(1), &rules8037)
	logs := transferLogs7708(sdb)
	if len(logs) != 1 {
		t.Fatalf("Amsterdam logs = %+v, want one", logs)
	}
	assertTransfer7708(t, logs[0], from, to, 1)
}

// TestEIP7708Fees ensures a priority-fee credit to coinbase remains outside
// EIP-7708: the transaction value transfer is the receipt's only log.
func TestEIP7708Fees(t *testing.T) {
	env := newBALTestEnv(nil)
	coinbase := common.HexToAddress("0x7708000000000000000000000000000000000050")
	recipient := common.HexToAddress("0x7708000000000000000000000000000000000051")
	engine := beacon.New(ethash.NewFaker())
	_, _, receipts := GenerateChainWithGenesis(env.gspec, engine, 1, func(_ int, g *BlockGen) {
		g.SetCoinbase(coinbase)
		g.AddTx(env.tx(0, &recipient, big.NewInt(1), txGasNewAccount, 1, nil))
	})
	logs := receipts[0][0].Logs
	if len(logs) != 1 {
		t.Fatalf("logs = %+v, want transaction transfer only", logs)
	}
	assertTransfer7708(t, logs[0], env.from, recipient, 1)
}
