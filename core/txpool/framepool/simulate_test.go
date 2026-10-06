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

package framepool

import (
	"encoding/binary"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vm/program"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

var simulationSender = common.HexToAddress("0x1234")
var simulationHelper = common.HexToAddress("0x5678")

func approveCode(p *program.Program, scope uint64) []byte {
	return p.Push(scope).Push(0).Push(0).Op(vm.APPROVE).Bytes()
}

func verifyFrame(scope uint64) types.Frame {
	return types.Frame{Mode: types.ModeVerify, Flags: scope, GasLimits: types.Limits{Execution: 40_000}, Value: new(uint256.Int)}
}

func runSimulation(t *testing.T, sender common.Address, frames []types.Frame, code map[common.Address][]byte) (*simResult, error) {
	t.Helper()
	result, _, _, err := runSimulationState(t, sender, frames, code)
	return result, err
}

func runSimulationState(t *testing.T, sender common.Address, frames []types.Frame, code map[common.Address][]byte) (*simResult, *state.StateDB, *types.Transaction, error) {
	t.Helper()
	config := *params.AllDevChainProtocolChanges
	config.AmsterdamTime = new(uint64)
	sdb, err := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	if err != nil {
		t.Fatal(err)
	}
	sdb.SetBalance(sender, uint256.NewInt(1e18), tracing.BalanceChangeUnspecified)
	for addr, code := range code {
		sdb.SetCode(addr, code, tracing.CodeChangeUnspecified)
		sdb.SetBalance(addr, uint256.NewInt(1e18), tracing.BalanceChangeUnspecified)
	}
	signatures := types.SignatureList{{Scheme: types.FrameTxSchemeSecp256k1, Signature: make([]byte, 65)}}
	if len(frames) > 1 && frames[len(frames)-1].Flags == types.ApprovePayment {
		payer := frames[len(frames)-1].ResolvedTarget(sender)
		sdb.SetBalance(payer, uint256.NewInt(1e18), tracing.BalanceChangeUnspecified)
		signatures = append(signatures, types.SignatureEntry{Scheme: types.FrameTxSchemeSecp256k1, Signer: payer.Bytes(), Signature: make([]byte, 65)})
	}
	tx := types.NewTx(&types.FrameTx{ChainID: uint256.MustFromBig(config.ChainID), Sender: sender, Frames: frames, Signatures: signatures, Fees: types.Fees{MaxFeePerGas: uint256.NewInt(1), MaxPriorityFeePerGas: new(uint256.Int), MaxFeePerBlobGas: new(uint256.Int)}})
	prefix, err := ClassifyPrefix(frames, sender, signatures)
	if err != nil {
		t.Fatal(err)
	}
	head := &types.Header{Number: big.NewInt(1), Time: 100, Difficulty: new(big.Int), BaseFee: big.NewInt(10), GasLimit: 30_000_000}
	result, err := simulate(&config, head, sdb, tx, prefix)
	return result, sdb, tx, err
}

func TestSimulationBannedOpcodes(t *testing.T) {
	for _, op := range []vm.OpCode{vm.GASPRICE, vm.BLOCKHASH, vm.COINBASE, vm.TIMESTAMP, vm.NUMBER, vm.PREVRANDAO, vm.GASLIMIT, vm.BASEFEE, vm.BLOBBASEFEE, vm.SLOTNUM, vm.INVALID, vm.SELFDESTRUCT, vm.BALANCE, vm.SELFBALANCE, vm.CREATE, vm.CREATE2, vm.SSTORE, vm.GAS} {
		t.Run(op.String(), func(t *testing.T) {
			p := program.New()
			for range 7 {
				p.Push(0)
			}
			code := approveCode(p.Op(op, vm.POP), types.ApproveExecutionAndPayment)
			_, err := runSimulation(t, simulationSender, []types.Frame{verifyFrame(types.ApproveExecutionAndPayment)}, map[common.Address][]byte{simulationSender: code})
			if !errors.Is(err, ErrTraceViolation) {
				t.Fatalf("got %v, want trace violation", err)
			}
		})
	}
}

func TestSimulationStorageAndTargets(t *testing.T) {
	slot := common.HexToHash("0x42")
	load := program.New().Push(slot).Op(vm.SLOAD, vm.POP, vm.STOP).Bytes()
	for _, tc := range []struct {
		name   string
		p      *program.Program
		helper []byte
		want   error
		slot   bool
		target common.Address
	}{
		{"sender storage", program.New().Push(slot).Op(vm.SLOAD, vm.POP), nil, nil, true, simulationHelper},
		{"foreign storage", program.New().Call(nil, simulationHelper, 0, 0, 0, 0, 0).Op(vm.POP), load, ErrTraceViolation, false, simulationHelper},
		{"delegate storage", program.New().DelegateCall(nil, simulationHelper, 0, 0, 0, 0).Op(vm.POP), load, nil, true, simulationHelper},
		{"empty call", program.New().Call(nil, simulationHelper, 0, 0, 0, 0, 0).Op(vm.POP), nil, ErrTraceViolation, false, simulationHelper},
		{"precompile", program.New().Call(nil, common.HexToAddress("0x04"), 0, 0, 0, 0, 0).Op(vm.POP), nil, nil, false, common.HexToAddress("0x04")},
		{"delegated call", program.New().Call(nil, simulationHelper, 0, 0, 0, 0, 0).Op(vm.POP), types.AddressToDelegation(simulationSender), ErrTraceViolation, false, simulationHelper},
		{"empty extcodesize", program.New().Push(simulationHelper).Op(vm.EXTCODESIZE, vm.POP), nil, ErrTraceViolation, false, simulationHelper},
		{"contract extcodehash", program.New().Push(simulationHelper).Op(vm.EXTCODEHASH, vm.POP), []byte{byte(vm.STOP)}, nil, false, simulationHelper},
		{"contract extcodecopy", program.New().Push(0).Push(0).Push(0).Push(simulationHelper).Op(vm.EXTCODECOPY), []byte{byte(vm.STOP)}, nil, false, simulationHelper},
	} {
		t.Run(tc.name, func(t *testing.T) {
			code := map[common.Address][]byte{simulationSender: approveCode(tc.p, types.ApproveExecutionAndPayment)}
			if tc.helper != nil {
				code[simulationHelper] = tc.helper
			}
			result, err := runSimulation(t, simulationSender, []types.Frame{verifyFrame(types.ApproveExecutionAndPayment)}, code)
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
			if err != nil {
				return
			}
			if tc.slot {
				if _, ok := result.dependencies.slots[slot]; !ok {
					t.Fatal("missing sender slot")
				}
			}
			if tc.name != "sender storage" && result.dependencies.accounts[tc.target]&dependencyCode == 0 {
				t.Fatal("missing helper code dependency")
			}
		})
	}
}

func TestSimulationDelegatedSender(t *testing.T) {
	result, err := runSimulation(t, simulationSender, []types.Frame{verifyFrame(types.ApproveExecutionAndPayment)}, map[common.Address][]byte{
		simulationSender: types.AddressToDelegation(simulationHelper), simulationHelper: approveCode(program.New(), types.ApproveExecutionAndPayment),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.dependencies.accounts[simulationHelper] != dependencyCode {
		t.Fatal("missing delegate code")
	}
}

func TestSimulationDeploy(t *testing.T) {
	runtime := approveCode(program.New(), types.ApproveExecutionAndPayment)
	initcode := program.New().Sstore(7, 8).ReturnViaCodeCopy(runtime).Bytes()
	factory := simulationHelper
	salt := common.HexToHash("0xab")
	sender2 := crypto.CreateAddress2(factory, salt, crypto.Keccak256(initcode))
	sender1 := crypto.CreateAddress(factory, 0)
	create := program.New().Mstore(initcode, 0).Push(len(initcode)).Push(0).Push(0).Op(vm.CREATE, vm.POP).Bytes()
	create2 := program.New().Create2(initcode, salt).Op(vm.POP).Bytes()
	for _, tc := range []struct {
		name        string
		sender      common.Address
		factoryCode []byte
		senderCode  []byte
		want        error
		nonce       bool
	}{
		{"create2", sender2, create2, nil, nil, false},
		{"create", sender1, create, nil, nil, true},
		{"foreign sstore", sender2, program.New().Sstore(0, 1).Append(create2).Bytes(), nil, ErrTraceViolation, false},
		{"value call", sender2, program.New().Call(nil, common.HexToAddress("0x04"), 1, 0, 0, 0, 0).Append(create2).Bytes(), nil, ErrTraceViolation, false},
		{"no code", simulationSender, []byte{byte(vm.STOP)}, nil, ErrTraceViolation, false},
		{"existing sender", sender2, create2, runtime, ErrTraceViolation, false},
		{"wrong created address", simulationSender, create2, nil, ErrTraceViolation, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			frames := []types.Frame{{Mode: types.ModeDefault, Target: &factory, GasLimits: types.Limits{Execution: 50_000, State: 500_000}, Value: new(uint256.Int)}, verifyFrame(types.ApproveExecutionAndPayment)}
			code := map[common.Address][]byte{factory: tc.factoryCode}
			if tc.senderCode != nil {
				code[tc.sender] = tc.senderCode
			}
			result, err := runSimulation(t, tc.sender, frames, code)
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
			if err != nil {
				return
			}
			want := dependencyCode
			if tc.nonce {
				want |= dependencyNonce
			}
			if result.dependencies.accounts[factory] != want {
				t.Fatalf("factory fields %v, want %v", result.dependencies.accounts[factory], want)
			}
		})
	}
}

func TestSimulationExpiry(t *testing.T) {
	for _, tc := range []struct {
		name     string
		code     []byte
		deadline uint64
		want     error
	}{
		{"equal", params.FrameTxExpiryVerifierCode, 100, nil},
		{"past", params.FrameTxExpiryVerifierCode, 99, core.ErrFrameTxInvalidExecution},
		{"noncanonical", []byte{byte(vm.STOP)}, 100, ErrTraceViolation},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := make([]byte, 8)
			binary.BigEndian.PutUint64(data, tc.deadline)
			target := params.FrameTxExpiryVerifier
			frames := []types.Frame{{Mode: types.ModeVerify, Target: &target, Data: data, GasLimits: types.Limits{Execution: 10_000}, Value: new(uint256.Int)}, verifyFrame(types.ApproveExecutionAndPayment)}
			result, err := runSimulation(t, simulationSender, frames, map[common.Address][]byte{target: tc.code})
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
			if err == nil && (result.expiryDeadline == nil || *result.expiryDeadline != tc.deadline || result.dependencies.accounts[target] != dependencyCode) {
				t.Fatal("missing expiry dependencies")
			}
		})
	}
}

func TestSimulationPayerDependencies(t *testing.T) {
	for _, tc := range []struct {
		name    string
		sponsor bool
		coded   bool
	}{
		{"self relay", false, false}, {"default sponsor", true, false}, {"coded paymaster", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			frames := []types.Frame{verifyFrame(types.ApproveExecutionAndPayment)}
			code := make(map[common.Address][]byte)
			payer := simulationSender
			if tc.sponsor {
				payer = simulationHelper
				frames[0] = verifyFrame(types.ApproveExecution)
				pay := verifyFrame(types.ApprovePayment)
				pay.Target = &payer
				frames = append(frames, pay)
				if tc.coded {
					code[payer] = approveCode(program.New(), types.ApprovePayment)
				}
			} else {
				code[payer] = approveCode(program.New(), types.ApproveExecutionAndPayment)
			}
			result, sdb, tx, err := runSimulationState(t, simulationSender, frames, code)
			if err != nil {
				t.Fatal(err)
			}
			if result.payer != payer || result.codedPaymaster != tc.coded {
				t.Fatalf("payer=%v coded=%v", result.payer, result.codedPaymaster)
			}
			if result.payerBalance.Cmp(uint256.NewInt(1e18)) != 0 {
				t.Fatalf("head payer balance %v, want %v", result.payerBalance, uint256.NewInt(1e18))
			}
			maxCost := new(uint256.Int).Mul(uint256.NewInt(tx.Gas()), uint256.MustFromBig(tx.GasFeeCap()))
			wantBalance := new(uint256.Int).Sub(uint256.NewInt(1e18), maxCost)
			if sdb.GetBalance(payer).Cmp(wantBalance) != 0 {
				t.Fatalf("simulation payer balance %v, want %v", sdb.GetBalance(payer), wantBalance)
			}
			if result.dependencies.accounts[simulationSender] != dependencyNonce|dependencyBalance|dependencyCode {
				t.Fatal("missing sender fields")
			}
			if result.dependencies.accounts[payer]&(dependencyBalance|dependencyCode) != dependencyBalance|dependencyCode {
				t.Fatal("missing payer fields")
			}
			if payer != simulationSender && result.dependencies.accounts[payer]&dependencyNonce != 0 {
				t.Fatal("unnecessary payer nonce dependency")
			}
		})
	}
}

func TestSimulationRequiredApproval(t *testing.T) {
	for _, code := range [][]byte{{byte(vm.STOP)}, program.New().Push(0).Push(0).Op(vm.REVERT).Bytes(), approveCode(program.New(), types.ApproveExecution)} {
		_, err := runSimulation(t, simulationSender, []types.Frame{verifyFrame(types.ApproveExecutionAndPayment)}, map[common.Address][]byte{simulationSender: code})
		if !errors.Is(err, core.ErrFrameTxInvalidExecution) {
			t.Fatalf("got %v, want invalid execution", err)
		}
	}
}
