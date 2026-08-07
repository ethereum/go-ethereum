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
	"encoding/binary"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/stateless"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/holiman/uint256"
)

const eip7709TestBlock = uint64(300)

var (
	eip7709TestCaller = common.Address{0xaa}
	eip7709TestSelf   = common.Address{0xbb}
	eip7709HeaderHash = common.Hash{0x11}
	eip7709StateHash  = common.Hash{0x22}
)

// bogota7709Config clones MergedTestChainConfig with Amsterdam and Bogotá active.
func bogota7709Config() *params.ChainConfig {
	config := *params.MergedTestChainConfig
	config.AmsterdamTime = new(uint64)
	config.BogotaTime = new(uint64)
	return &config
}

func eip7709TestSlot(number uint64) common.Hash {
	var slot common.Hash
	binary.BigEndian.PutUint64(slot[24:], number%params.HistoryServeWindow)
	return slot
}

func eip7709TestCode(number *uint256.Int) []byte {
	value := number.Bytes32()
	code := []byte{byte(PUSH32)}
	code = append(code, value[:]...)
	return append(code,
		byte(BLOCKHASH),
		byte(PUSH0), byte(MSTORE),
		byte(PUSH1), 32,
		byte(PUSH0), byte(RETURN),
	)
}

func new7709EVM(t *testing.T, code []byte, chainConfig *params.ChainConfig) (*EVM, *state.StateDB) {
	t.Helper()
	statedb, err := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	if err != nil {
		t.Fatal(err)
	}
	statedb.CreateAccount(eip7709TestSelf)
	statedb.SetCode(eip7709TestSelf, code, tracing.CodeChangeUnspecified)
	statedb.CreateAccount(params.HistoryStorageAddress)
	statedb.SetNonce(params.HistoryStorageAddress, 1, tracing.NonceChangeUnspecified)
	statedb.SetState(params.HistoryStorageAddress, eip7709TestSlot(eip7709TestBlock-1), eip7709StateHash)
	statedb.SetState(params.HistoryStorageAddress, eip7709TestSlot(eip7709TestBlock-256), eip7709StateHash)
	statedb.Finalise(true)

	return new7709EVMWithState(statedb, chainConfig), statedb
}

func new7709EVMWithState(statedb *state.StateDB, chainConfig *params.ChainConfig) *EVM {
	random := common.Hash{}
	context := BlockContext{
		CanTransfer: func(StateDB, common.Address, *uint256.Int) bool { return true },
		Transfer:    func(StateDB, common.Address, common.Address, *uint256.Int, *params.Rules) {},
		GetHash:     func(uint64) common.Hash { return eip7709HeaderHash },
		BlockNumber: new(big.Int).SetUint64(eip7709TestBlock),
		Random:      &random,
	}
	evm := NewEVM(context, statedb, chainConfig, Config{})
	statedb.Prepare(evm.GetRules(), eip7709TestCaller, common.Address{}, &eip7709TestSelf, nil, nil)
	evm.SetTxContext(TxContext{})
	return evm
}

// run7709 executes BLOCKHASH for number under the Bogotá ruleset.
func run7709(t *testing.T, number *uint256.Int) (common.Hash, GasBudget, *state.StateDB) {
	t.Helper()
	return run7709Code(t, eip7709TestCode(number), false)
}

func run7709Code(t *testing.T, code []byte, prewarm bool) (common.Hash, GasBudget, *state.StateDB) {
	t.Helper()
	return run7709CodeWithConfig(t, code, bogota7709Config(), prewarm)
}

func run7709CodeWithConfig(t *testing.T, code []byte, chainConfig *params.ChainConfig, prewarm bool) (common.Hash, GasBudget, *state.StateDB) {
	t.Helper()
	evm, statedb := new7709EVM(t, code, chainConfig)
	if prewarm {
		statedb.AddSlotToAccessList(params.HistoryStorageAddress, eip7709TestSlot(eip7709TestBlock-1))
	}
	ret, gas, err := evm.Call(eip7709TestCaller, eip7709TestSelf, nil, NewGasBudget(1_000_000, 0), new(uint256.Int))
	if err != nil {
		t.Fatal(err)
	}
	return common.BytesToHash(ret), gas, statedb
}

func TestEIP7709BlockHashSource(t *testing.T) {
	number := new(uint256.Int).SetUint64(eip7709TestBlock - 1)

	legacy, _, _ := run7709CodeWithConfig(t, eip7709TestCode(number), params.MergedTestChainConfig, false)
	if legacy != eip7709HeaderHash {
		t.Fatalf("legacy BLOCKHASH returned %v, want header hash %v", legacy, eip7709HeaderHash)
	}
	fromState, _, _ := run7709(t, number)
	if fromState != eip7709StateHash {
		t.Fatalf("EIP-7709 BLOCKHASH returned %v, want state hash %v", fromState, eip7709StateHash)
	}
}

func TestEIP7709BlockHashBounds(t *testing.T) {
	overflow := uint256.MustFromBig(new(big.Int).Lsh(big.NewInt(1), 64))
	tests := []struct {
		name   string
		number *uint256.Int
		want   common.Hash
		warm   bool
	}{
		{"newest", new(uint256.Int).SetUint64(eip7709TestBlock - 1), eip7709StateHash, true},
		{"oldest", new(uint256.Int).SetUint64(eip7709TestBlock - 256), eip7709StateHash, true},
		{"current", new(uint256.Int).SetUint64(eip7709TestBlock), common.Hash{}, false},
		{"too old", new(uint256.Int).SetUint64(eip7709TestBlock - 257), common.Hash{}, false},
		{"uint64 overflow", overflow, common.Hash{}, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, _, statedb := run7709(t, test.number)
			if got != test.want {
				t.Fatalf("BLOCKHASH returned %v, want %v", got, test.want)
			}
			if test.number.IsUint64() {
				_, warm := statedb.SlotInAccessList(params.HistoryStorageAddress, eip7709TestSlot(test.number.Uint64()))
				if warm != test.warm {
					t.Fatalf("history slot warm = %v, want %v", warm, test.warm)
				}
			}
		})
	}
}

func TestEIP7709BlockHashGas(t *testing.T) {
	const opcodeBase = GasFastestStep + GasExtStep + GasQuickStep + GasFastestStep + GasFastestStep + GasQuickStep + params.MemoryGas
	number := new(uint256.Int).SetUint64(eip7709TestBlock - 1)

	_, cold, _ := run7709(t, number)
	if want := opcodeBase + params.ColdStorageAccessAmsterdam; cold.UsedExecutionGas != want {
		t.Fatalf("cold BLOCKHASH gas = %d, want %d", cold.UsedExecutionGas, want)
	}
	_, warm, _ := run7709Code(t, eip7709TestCode(number), true)
	if want := opcodeBase + params.WarmStorageReadCostEIP2929; warm.UsedExecutionGas != want {
		t.Fatalf("prewarmed BLOCKHASH gas = %d, want %d", warm.UsedExecutionGas, want)
	}
	_, invalid, _ := run7709(t, new(uint256.Int).SetUint64(eip7709TestBlock))
	if invalid.UsedExecutionGas != opcodeBase {
		t.Fatalf("invalid BLOCKHASH gas = %d, want %d", invalid.UsedExecutionGas, opcodeBase)
	}
}

func TestEIP7709RepeatedBlockHashIsWarm(t *testing.T) {
	number := new(uint256.Int).SetUint64(eip7709TestBlock - 1)
	value := number.Bytes32()
	code := []byte{byte(PUSH32)}
	code = append(code, value[:]...)
	code = append(code, byte(BLOCKHASH), byte(POP), byte(PUSH32))
	code = append(code, value[:]...)
	code = append(code,
		byte(BLOCKHASH),
		byte(PUSH0), byte(MSTORE),
		byte(PUSH1), 32,
		byte(PUSH0), byte(RETURN),
	)
	got, gas, _ := run7709Code(t, code, false)
	if got != eip7709StateHash {
		t.Fatalf("BLOCKHASH returned %v, want %v", got, eip7709StateHash)
	}
	const opcodeBase = 2*GasFastestStep + 2*GasExtStep + GasQuickStep + GasQuickStep + GasFastestStep + GasFastestStep + GasQuickStep + params.MemoryGas
	want := opcodeBase + params.ColdStorageAccessAmsterdam + params.WarmStorageReadCostEIP2929
	if gas.UsedExecutionGas != want {
		t.Fatalf("repeated BLOCKHASH gas = %d, want %d", gas.UsedExecutionGas, want)
	}
}

func TestEIP7709BogotaBlockAccessList(t *testing.T) {
	number := new(uint256.Int).SetUint64(eip7709TestBlock - 1)
	_, _, statedb := run7709(t, number)
	accesses := statedb.Finalise(true)
	account := accesses.Accounts[params.HistoryStorageAddress]
	if account == nil {
		t.Fatal("history storage account missing from block access list")
	}
	if _, ok := account.StorageReads[eip7709TestSlot(eip7709TestBlock-1)]; !ok {
		t.Fatal("history slot missing from block access list")
	}
}

func TestEIP7709StatelessWitness(t *testing.T) {
	number := new(uint256.Int).SetUint64(eip7709TestBlock - 1)
	code := eip7709TestCode(number)
	db := state.NewDatabaseForTesting()
	initial, err := state.New(types.EmptyRootHash, db)
	if err != nil {
		t.Fatal(err)
	}
	initial.CreateAccount(eip7709TestSelf)
	initial.SetCode(eip7709TestSelf, code, tracing.CodeChangeUnspecified)
	initial.CreateAccount(params.HistoryStorageAddress)
	initial.SetNonce(params.HistoryStorageAddress, 1, tracing.NonceChangeUnspecified)
	initial.SetState(params.HistoryStorageAddress, eip7709TestSlot(eip7709TestBlock-1), eip7709StateHash)
	root, err := initial.Commit(0, true, false)
	if err != nil {
		t.Fatal(err)
	}
	fullState, err := state.New(root, db)
	if err != nil {
		t.Fatal(err)
	}
	witness := &stateless.Witness{
		Headers: []*types.Header{{Number: new(big.Int).SetUint64(eip7709TestBlock - 1), Difficulty: new(big.Int), Root: root}},
		Codes:   make(map[string]struct{}),
		State:   make(map[string]struct{}),
	}
	fullState.StartPrefetcher("eip7709", witness)
	evm := new7709EVMWithState(fullState, bogota7709Config())
	ret, _, err := evm.Call(eip7709TestCaller, eip7709TestSelf, nil, NewGasBudget(1_000_000, 0), new(uint256.Int))
	if err != nil {
		t.Fatal(err)
	}
	if got := common.BytesToHash(ret); got != eip7709StateHash {
		t.Fatalf("full-state BLOCKHASH returned %v, want %v", got, eip7709StateHash)
	}
	fullState.IntermediateRoot(true)
	fullState.StopPrefetcher()
	if len(witness.State) == 0 {
		t.Fatal("BLOCKHASH did not add the history storage proof to the witness")
	}
	if len(witness.Headers) != 1 {
		t.Fatalf("witness contains %d headers, want only the parent header", len(witness.Headers))
	}

	memdb := witness.MakeHashDB()
	witnessState, err := state.New(root, state.NewDatabase(triedb.NewDatabase(memdb, triedb.HashDefaults), state.NewCodeDB(memdb)))
	if err != nil {
		t.Fatal(err)
	}
	replay := new7709EVMWithState(witnessState, bogota7709Config())
	ret, _, err = replay.Call(eip7709TestCaller, eip7709TestSelf, nil, NewGasBudget(1_000_000, 0), new(uint256.Int))
	if err != nil {
		t.Fatal(err)
	}
	if got := common.BytesToHash(ret); got != eip7709StateHash {
		t.Fatalf("stateless BLOCKHASH returned %v, want %v", got, eip7709StateHash)
	}
	if err := witnessState.Error(); err != nil {
		t.Fatalf("stateless replay used data absent from the witness: %v", err)
	}
}
