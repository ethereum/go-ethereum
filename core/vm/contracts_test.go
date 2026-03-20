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

package vm

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// precompiledTest defines the input/output pairs for precompiled contract tests.
type precompiledTest struct {
	Input, Expected string
	Gas             uint64
	Name            string
	NoBenchmark     bool // Benchmark primarily the worst-cases
}

// precompiledFailureTest defines the input/error pairs for precompiled
// contract failure tests.
type precompiledFailureTest struct {
	Input         string
	ExpectedError string
	Name          string
}

// allPrecompiles does not map to the actual set of precompiles, as it also contains
// repriced versions of precompiles at certain slots
var allPrecompiles = map[common.Address]PrecompiledContract{
	common.BytesToAddress([]byte{1}):    &ecrecover{},
	common.BytesToAddress([]byte{2}):    &sha256hash{},
	common.BytesToAddress([]byte{3}):    &ripemd160hash{},
	common.BytesToAddress([]byte{4}):    &dataCopy{},
	common.BytesToAddress([]byte{5}):    &bigModExp{eip2565: false},
	common.BytesToAddress([]byte{0xf5}): &bigModExp{eip2565: true},
	common.BytesToAddress([]byte{6}):    &bn256AddIstanbul{},
	common.BytesToAddress([]byte{7}):    &bn256ScalarMulIstanbul{},
	common.BytesToAddress([]byte{8}):    &bn256PairingIstanbul{},
	common.BytesToAddress([]byte{9}):    &blake2F{},
	common.BytesToAddress([]byte{0x0a}): &kzgPointEvaluation{},

	common.BytesToAddress([]byte{0x0f, 0x0a}): &bls12381G1Add{},
	common.BytesToAddress([]byte{0x0f, 0x0b}): &bls12381G1Mul{},
	common.BytesToAddress([]byte{0x0f, 0x0c}): &bls12381G1MultiExp{},
	common.BytesToAddress([]byte{0x0f, 0x0d}): &bls12381G2Add{},
	common.BytesToAddress([]byte{0x0f, 0x0e}): &bls12381G2Mul{},
	common.BytesToAddress([]byte{0x0f, 0x0f}): &bls12381G2MultiExp{},
	common.BytesToAddress([]byte{0x0f, 0x10}): &bls12381Pairing{},
	common.BytesToAddress([]byte{0x0f, 0x11}): &bls12381MapG1{},
	common.BytesToAddress([]byte{0x0f, 0x12}): &bls12381MapG2{},
}

// EIP-152 test vectors
var blake2FMalformedInputTests = []precompiledFailureTest{
	{
		Input:         "",
		ExpectedError: errBlake2FInvalidInputLength.Error(),
		Name:          "vector 0: empty input",
	},
	{
		Input:         "00000c48c9bdf267e6096a3ba7ca8485ae67bb2bf894fe72f36e3cf1361d5f3af54fa5d182e6ad7f520e511f6c3e2b8c68059b6bbd41fbabd9831f79217e1319cde05b61626300000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000300000000000000000000000000000001",
		ExpectedError: errBlake2FInvalidInputLength.Error(),
		Name:          "vector 1: less than 213 bytes input",
	},
	{
		Input:         "000000000c48c9bdf267e6096a3ba7ca8485ae67bb2bf894fe72f36e3cf1361d5f3af54fa5d182e6ad7f520e511f6c3e2b8c68059b6bbd41fbabd9831f79217e1319cde05b61626300000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000300000000000000000000000000000001",
		ExpectedError: errBlake2FInvalidInputLength.Error(),
		Name:          "vector 2: more than 213 bytes input",
	},
	{
		Input:         "0000000c48c9bdf267e6096a3ba7ca8485ae67bb2bf894fe72f36e3cf1361d5f3af54fa5d182e6ad7f520e511f6c3e2b8c68059b6bbd41fbabd9831f79217e1319cde05b61626300000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000300000000000000000000000000000002",
		ExpectedError: errBlake2FInvalidFinalFlag.Error(),
		Name:          "vector 3: malformed final block indicator flag",
	},
}

func testPrecompiled(addr string, test precompiledTest, t *testing.T) {
	p := allPrecompiles[common.HexToAddress(addr)]
	in := common.Hex2Bytes(test.Input)
	gas := p.RequiredGas(in)
	t.Run(fmt.Sprintf("%s-Gas=%d", test.Name, gas), func(t *testing.T) {
		if res, _, err := runPrecompiledContract(&EVM{}, p, AccountRef(common.Address{}), in, gas, new(uint256.Int), false); err != nil {
			t.Error(err)
		} else if common.Bytes2Hex(res) != test.Expected {
			t.Errorf("Expected %v, got %v", test.Expected, common.Bytes2Hex(res))
		}
		if expGas := test.Gas; expGas != gas {
			t.Errorf("%v: gas wrong, expected %d, got %d", test.Name, expGas, gas)
		}
		// Verify that the precompile did not touch the input buffer
		exp := common.Hex2Bytes(test.Input)
		if !bytes.Equal(in, exp) {
			t.Errorf("Precompiled %v modified input data", addr)
		}
	})
}

func testPrecompiledOOG(addr string, test precompiledTest, t *testing.T) {
	p := allPrecompiles[common.HexToAddress(addr)]
	in := common.Hex2Bytes(test.Input)
	gas := p.RequiredGas(in) - 1

	t.Run(fmt.Sprintf("%s-Gas=%d", test.Name, gas), func(t *testing.T) {
		_, _, err := runPrecompiledContract(&EVM{}, p, AccountRef(common.Address{}), in, gas, new(uint256.Int), false)
		if err.Error() != "out of gas" {
			t.Errorf("Expected error [out of gas], got [%v]", err)
		}
		// Verify that the precompile did not touch the input buffer
		exp := common.Hex2Bytes(test.Input)
		if !bytes.Equal(in, exp) {
			t.Errorf("Precompiled %v modified input data", addr)
		}
	})
}

func testPrecompiledFailure(addr string, test precompiledFailureTest, t *testing.T) {
	p := allPrecompiles[common.HexToAddress(addr)]
	in := common.Hex2Bytes(test.Input)
	gas := p.RequiredGas(in)
	t.Run(test.Name, func(t *testing.T) {
		_, _, err := runPrecompiledContract(&EVM{}, p, AccountRef(common.Address{}), in, gas, new(uint256.Int), false)
		if err.Error() != test.ExpectedError {
			t.Errorf("Expected error [%v], got [%v]", test.ExpectedError, err)
		}
		// Verify that the precompile did not touch the input buffer
		exp := common.Hex2Bytes(test.Input)
		if !bytes.Equal(in, exp) {
			t.Errorf("Precompiled %v modified input data", addr)
		}
	})
}

func benchmarkPrecompiled(addr string, test precompiledTest, bench *testing.B) {
	if test.NoBenchmark {
		return
	}
	p := allPrecompiles[common.HexToAddress(addr)]
	in := common.Hex2Bytes(test.Input)
	reqGas := p.RequiredGas(in)

	var (
		res  []byte
		err  error
		data = make([]byte, len(in))
	)

	bench.Run(fmt.Sprintf("%s-Gas=%d", test.Name, reqGas), func(bench *testing.B) {
		bench.ReportAllocs()
		start := time.Now()
		bench.ResetTimer()
		for i := 0; i < bench.N; i++ {
			copy(data, in)
			res, _, err = runPrecompiledContract(&EVM{}, p, AccountRef(common.Address{}), in, reqGas, new(uint256.Int), false)
		}
		bench.StopTimer()
		elapsed := uint64(time.Since(start))
		if elapsed < 1 {
			elapsed = 1
		}
		gasUsed := reqGas * uint64(bench.N)
		bench.ReportMetric(float64(reqGas), "gas/op")
		// Keep it as uint64, multiply 100 to get two digit float later
		mgasps := (100 * 1000 * gasUsed) / elapsed
		bench.ReportMetric(float64(mgasps)/100, "mgas/s")
		//Check if it is correct
		if err != nil {
			bench.Error(err)
			return
		}
		if common.Bytes2Hex(res) != test.Expected {
			bench.Errorf("Expected %v, got %v", test.Expected, common.Bytes2Hex(res))
			return
		}
	})
}

// mockPrecompile is a test precompile used to record the arguments it is
// executed with.
type mockPrecompile struct {
	addr             common.Address
	gas              uint64
	observedCaller   common.Address
	observedAddress  common.Address
	observedOrigin   common.Address
	observedValue    *uint256.Int
	observedInput    []byte
	observedReadOnly bool
}

func (p *mockPrecompile) Address() common.Address {
	return p.addr
}

func (p *mockPrecompile) RequiredGas(input []byte) uint64 {
	return p.gas
}

func (p *mockPrecompile) Run(evm *EVM, contract *Contract, readonly bool) ([]byte, error) {
	// Record the execution arguments observed by the test precompile.
	p.observedCaller = contract.Caller()
	p.observedAddress = contract.Address()
	if contract.Value() != nil {
		p.observedValue = new(uint256.Int).Set(contract.Value())
	}
	p.observedInput = common.CopyBytes(contract.Input)
	p.observedReadOnly = readonly
	if evm != nil {
		p.observedOrigin = evm.TxContext.Origin
	}
	return []byte{0xaa}, nil
}

type precompileContextExpectation struct {
	caller      common.Address
	origin      common.Address
	readOnly    bool
	assertValue func(*testing.T, *uint256.Int)
}

func expectNilValue(t *testing.T, observed *uint256.Int) {
	if observed != nil {
		t.Fatalf("unexpected call value: want nil got %v", observed)
	}
}

func expectZeroValue(t *testing.T, observed *uint256.Int) {
	if observed == nil || !observed.IsZero() {
		t.Fatalf("unexpected call value: want 0 got %v", observed)
	}
}

func expectExactValue(expected *uint256.Int) func(*testing.T, *uint256.Int) {
	return func(t *testing.T, observed *uint256.Int) {
		t.Helper()
		if observed == nil || observed.Cmp(expected) != 0 {
			t.Fatalf("unexpected call value: want %s got %v", expected, observed)
		}
	}
}

func assertExpectedError(t *testing.T, err error, expectedErr error) {
	t.Helper()

	switch {
	case expectedErr == nil && err == nil:
		return
	case expectedErr == nil && err != nil:
		t.Fatalf("unexpected error: %v", err)
	case expectedErr != nil && err == nil:
		t.Fatalf("unexpected error: want %v got nil", expectedErr)
	case err.Error() != expectedErr.Error():
		t.Fatalf("unexpected error: want %v got %v", expectedErr, err)
	}
}

func newPrecompileTestEVM(t *testing.T, origin common.Address) *EVM {
	statedb, err := state.New(types.EmptyRootHash, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	if err != nil {
		t.Fatalf("failed to create state db: %v", err)
	}

	return NewEVM(BlockContext{
		CanTransfer: func(db StateDB, addr common.Address, amount *uint256.Int) bool {
			return db.GetBalance(addr).Cmp(amount) >= 0
		},
		Transfer: func(db StateDB, from, to common.Address, amount *uint256.Int) {
			if amount.IsZero() {
				return
			}
			db.SubBalance(from, amount, tracing.BalanceChangeTransfer)
			db.AddBalance(to, amount, tracing.BalanceChangeTransfer)
		},
		BlockNumber: new(big.Int),
	}, TxContext{Origin: origin}, statedb, params.AllEthashProtocolChanges, Config{})
}

func addBalance(evm *EVM, addr common.Address, amount *uint256.Int) {
	evm.StateDB.CreateAccount(addr)
	evm.StateDB.AddBalance(addr, amount, tracing.BalanceChangeUnspecified)
}

func assertPrecompileContext(
	t *testing.T,
	precompile *mockPrecompile,
	precompileAddr common.Address,
	input []byte,
	remainingGas uint64,
	suppliedGas uint64,
	gasCost uint64,
	expect precompileContextExpectation,
) {

	if remainingGas != suppliedGas-gasCost {
		t.Fatalf("unexpected remaining gas: want %d got %d", suppliedGas-gasCost, remainingGas)
	}
	if precompile.observedCaller != expect.caller {
		t.Fatalf("unexpected caller: want %s got %s", expect.caller, precompile.observedCaller)
	}
	if precompile.observedAddress != precompileAddr {
		t.Fatalf("unexpected contract address: want %s got %s", precompileAddr, precompile.observedAddress)
	}
	if precompile.observedOrigin != expect.origin {
		t.Fatalf("unexpected origin: want %s got %s", expect.origin, precompile.observedOrigin)
	}
	expect.assertValue(t, precompile.observedValue)
	if !bytes.Equal(precompile.observedInput, input) {
		t.Fatalf("unexpected input: want %x got %x", input, precompile.observedInput)
	}
	if precompile.observedReadOnly != expect.readOnly {
		t.Fatalf("unexpected readonly flag: want %t got %t", expect.readOnly, precompile.observedReadOnly)
	}
}

func TestEVMExecutePrecompileWithExpectedCallContexts(t *testing.T) {
	const (
		suppliedGas = uint64(100)
		gasCost     = uint64(10)
	)

	// callerAddr models msg.sender, the immediate caller seen by the precompile
	// in that call frame.
	callerAddr := common.HexToAddress("0x6001")
	// originAddr models tx.origin, the address that started the transaction.
	originAddr := common.HexToAddress("0x6002")
	// precompileAddr is the address used to register the test precompile.
	precompileAddr := common.HexToAddress("0x6003")
	// calldata is the calldata sent to the precompile.
	calldata := []byte{0x01, 0x02}

	// delegateParentAddr is used to build the outer delegatecall frame.
	delegateParentAddr := common.HexToAddress("0x5002")
	// delegateCallerAddr is used in the delegatecall case, where the
	// preserved delegate context is seen by the precompile as the caller.
	delegateCallerAddr := common.HexToAddress("0x5003")

	testCases := []struct {
		name              string
		prepareCaller     func(*EVM) ContractRef
		executePrecompile func(*EVM, *mockPrecompile, ContractRef) ([]byte, uint64, error)
		expect            precompileContextExpectation
	}{
		{
			name: "call",
			prepareCaller: func(evm *EVM) ContractRef {
				addBalance(evm, callerAddr, uint256.NewInt(1000))
				return AccountRef(callerAddr)
			},
			executePrecompile: func(evm *EVM, precompile *mockPrecompile, caller ContractRef) ([]byte, uint64, error) {
				return evm.Call(caller, precompileAddr, calldata, suppliedGas, uint256.NewInt(13))
			},
			expect: precompileContextExpectation{
				caller:      callerAddr,
				origin:      originAddr,
				readOnly:    false,
				assertValue: expectExactValue(uint256.NewInt(13)),
			},
		},
		{
			name: "staticcall",
			prepareCaller: func(evm *EVM) ContractRef {
				evm.StateDB.CreateAccount(callerAddr)
				return AccountRef(callerAddr)
			},
			executePrecompile: func(evm *EVM, precompile *mockPrecompile, caller ContractRef) ([]byte, uint64, error) {
				return evm.StaticCall(caller, precompileAddr, calldata, suppliedGas)
			},
			expect: precompileContextExpectation{
				caller:      callerAddr,
				origin:      originAddr,
				readOnly:    true,
				assertValue: expectZeroValue,
			},
		},
		{
			name: "delegatecall",
			prepareCaller: func(evm *EVM) ContractRef {
				evm.StateDB.CreateAccount(delegateCallerAddr)
				// Build a parent frame to simulate a delegatecall setup.
				// In regular contract DELEGATECALL, caller context is derived
				// from the parent frame. For precompiles, the implementation behaves
				// like CALL for caller identity: the direct caller (delegateCallerAddr)
				// is observed by the precompile, not the parent frame.
				parent := NewContract(AccountRef(callerAddr), AccountRef(delegateParentAddr), uint256.NewInt(17), suppliedGas)
				return NewContract(parent, AccountRef(delegateCallerAddr), nil, suppliedGas)
			},
			executePrecompile: func(evm *EVM, precompile *mockPrecompile, caller ContractRef) ([]byte, uint64, error) {
				return evm.DelegateCall(caller, precompileAddr, calldata, suppliedGas)
			},
			// DELEGATECALL to a precompile is read-only. It keeps the direct caller
			// which in this test is the caller passed to `evm.DelegateCall`
			// (delegateCallerAddr). This differs from regular contract DELEGATECALL
			// semantics, where caller context is derived from the parent frame
			// via AsDelegate.
			expect: precompileContextExpectation{
				caller:      delegateCallerAddr,
				origin:      originAddr,
				readOnly:    true,
				assertValue: expectNilValue,
			},
		},
		{
			name: "callcode",
			prepareCaller: func(evm *EVM) ContractRef {
				addBalance(evm, callerAddr, uint256.NewInt(1000))
				return AccountRef(callerAddr)
			},
			executePrecompile: func(evm *EVM, precompile *mockPrecompile, caller ContractRef) ([]byte, uint64, error) {
				return evm.CallCode(caller, precompileAddr, calldata, suppliedGas, uint256.NewInt(19))
			},
			// CALLCODE to a precompile is always read-only, unlike CALLCODE to
			// a regular contract.
			expect: precompileContextExpectation{
				caller:      callerAddr,
				origin:      originAddr,
				readOnly:    true,
				assertValue: expectExactValue(uint256.NewInt(19)),
			},
		},
		{
			name: "runPrecompiledContract",
			prepareCaller: func(evm *EVM) ContractRef {
				return AccountRef(callerAddr)
			},
			executePrecompile: func(evm *EVM, precompile *mockPrecompile, caller ContractRef) ([]byte, uint64, error) {
				// Execute the precompile via the low-level helper.
				return runPrecompiledContract(evm, precompile, caller, calldata, suppliedGas, uint256.NewInt(42), true)
			},
			expect: precompileContextExpectation{
				caller:      callerAddr,
				origin:      originAddr,
				readOnly:    true,
				assertValue: expectExactValue(uint256.NewInt(42)),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			evm := newPrecompileTestEVM(t, originAddr)
			precompile := &mockPrecompile{addr: precompileAddr, gas: gasCost}
			evm.WithPrecompiles(map[common.Address]PrecompiledContract{precompileAddr: precompile}, []common.Address{precompileAddr})

			caller := tc.prepareCaller(evm)
			ret, remainingGas, err := tc.executePrecompile(evm, precompile, caller)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !bytes.Equal(ret, []byte{0xaa}) {
				t.Fatalf("unexpected return data: %x", ret)
			}
			assertPrecompileContext(t, precompile, precompileAddr, calldata, remainingGas, suppliedGas, gasCost, tc.expect)
		})
	}
}

// balanceMutatingMockPrecompile is a test precompile that mutates balances.
type balanceMutatingMockPrecompile struct {
	addr   common.Address
	gas    uint64
	from   common.Address
	to     common.Address
	amount *uint256.Int
}

func (p *balanceMutatingMockPrecompile) Address() common.Address {
	return p.addr
}

func (p *balanceMutatingMockPrecompile) RequiredGas(input []byte) uint64 {
	return p.gas
}

func (p *balanceMutatingMockPrecompile) Run(evm *EVM, contract *Contract, readonly bool) ([]byte, error) {
	if readonly {
		return nil, fmt.Errorf("cannot run in read-only mode")
	}
	evm.StateDB.SubBalance(p.from, p.amount, tracing.BalanceChangeTransfer)
	evm.StateDB.AddBalance(p.to, p.amount, tracing.BalanceChangeTransfer)
	return []byte{0xaa}, nil
}

func TestEVMCallExecutesPrecompileStateMutation(t *testing.T) {
	const (
		suppliedGas = uint64(100)
		gasCost     = uint64(10)
	)

	callerAddr := common.HexToAddress("0x7001")
	originAddr := common.HexToAddress("0x7002")
	precompileAddr := common.HexToAddress("0x7003")
	recipientAddr := common.HexToAddress("0x7004")
	calldata := []byte{0x01}
	delegateParentAddr := common.HexToAddress("0x7101")
	delegateCallerAddr := common.HexToAddress("0x7102")

	initialCallerBalance := uint256.NewInt(100)
	transferAmount := uint256.NewInt(7)
	testCases := []struct {
		name              string
		prepareCaller     func(*EVM) ContractRef
		executePrecompile func(*EVM, ContractRef) ([]byte, uint64, error)
		expectedErr       error
		expectedRet       []byte
		expectedGas       uint64
		expectedCallerBal *uint256.Int
		expectedTargetBal *uint256.Int
	}{
		{
			name: "call",
			prepareCaller: func(evm *EVM) ContractRef {
				addBalance(evm, callerAddr, initialCallerBalance)
				addBalance(evm, recipientAddr, uint256.NewInt(0))
				return AccountRef(callerAddr)
			},
			executePrecompile: func(evm *EVM, caller ContractRef) ([]byte, uint64, error) {
				return evm.Call(caller, precompileAddr, calldata, suppliedGas, uint256.NewInt(0))
			},
			expectedRet:       []byte{0xaa},
			expectedGas:       suppliedGas - gasCost,
			expectedCallerBal: uint256.NewInt(93),
			expectedTargetBal: uint256.NewInt(7),
		},
		{
			name: "staticcall",
			prepareCaller: func(evm *EVM) ContractRef {
				addBalance(evm, callerAddr, initialCallerBalance)
				addBalance(evm, recipientAddr, uint256.NewInt(0))
				return AccountRef(callerAddr)
			},
			executePrecompile: func(evm *EVM, caller ContractRef) ([]byte, uint64, error) {
				return evm.StaticCall(caller, precompileAddr, calldata, suppliedGas)
			},
			expectedErr:       errors.New("cannot run in read-only mode"),
			expectedRet:       nil,
			expectedGas:       0,
			expectedCallerBal: initialCallerBalance,
			expectedTargetBal: uint256.NewInt(0),
		},
		{
			name: "delegatecall",
			prepareCaller: func(evm *EVM) ContractRef {
				addBalance(evm, callerAddr, initialCallerBalance)
				addBalance(evm, recipientAddr, uint256.NewInt(0))
				evm.StateDB.CreateAccount(delegateCallerAddr)
				parent := NewContract(AccountRef(callerAddr), AccountRef(delegateParentAddr), uint256.NewInt(0), suppliedGas)
				return NewContract(parent, AccountRef(delegateCallerAddr), nil, suppliedGas)
			},
			executePrecompile: func(evm *EVM, caller ContractRef) ([]byte, uint64, error) {
				return evm.DelegateCall(caller, precompileAddr, calldata, suppliedGas)
			},
			expectedErr:       errors.New("cannot run in read-only mode"),
			expectedRet:       nil,
			expectedGas:       0,
			expectedCallerBal: initialCallerBalance,
			expectedTargetBal: uint256.NewInt(0),
		},
		{
			name: "callcode",
			prepareCaller: func(evm *EVM) ContractRef {
				addBalance(evm, callerAddr, initialCallerBalance)
				addBalance(evm, recipientAddr, uint256.NewInt(0))
				return AccountRef(callerAddr)
			},
			executePrecompile: func(evm *EVM, caller ContractRef) ([]byte, uint64, error) {
				return evm.CallCode(caller, precompileAddr, calldata, suppliedGas, uint256.NewInt(0))
			},
			expectedErr:       errors.New("cannot run in read-only mode"),
			expectedRet:       nil,
			expectedGas:       0,
			expectedCallerBal: initialCallerBalance,
			expectedTargetBal: uint256.NewInt(0),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			evm := newPrecompileTestEVM(t, originAddr)
			precompile := &balanceMutatingMockPrecompile{
				addr:   precompileAddr,
				gas:    gasCost,
				from:   callerAddr,
				to:     recipientAddr,
				amount: transferAmount,
			}
			evm.WithPrecompiles(map[common.Address]PrecompiledContract{precompileAddr: precompile}, []common.Address{precompileAddr})

			caller := tc.prepareCaller(evm)
			ret, remainingGas, err := tc.executePrecompile(evm, caller)
			assertExpectedError(t, err, tc.expectedErr)
			if !bytes.Equal(ret, tc.expectedRet) {
				t.Fatalf("unexpected return data: want %x got %x", tc.expectedRet, ret)
			}
			if remainingGas != tc.expectedGas {
				t.Fatalf("unexpected remaining gas: want %d got %d", tc.expectedGas, remainingGas)
			}

			// Verify whether the precompile state mutation is persisted.
			callerBalance := evm.StateDB.GetBalance(callerAddr)
			if callerBalance.Cmp(tc.expectedCallerBal) != 0 {
				t.Fatalf("unexpected caller balance: want %s got %s", tc.expectedCallerBal, callerBalance)
			}
			targetBalance := evm.StateDB.GetBalance(recipientAddr)
			if targetBalance.Cmp(tc.expectedTargetBal) != 0 {
				t.Fatalf("unexpected recipient balance: want %s got %s", tc.expectedTargetBal, targetBalance)
			}
		})
	}
}

var errBalanceMutatingMockPrecompile = errors.New("balance-mutating precompile failed")

// balanceMutatingErrorMockPrecompile is a test precompile that mutates
// balances and then returns an error.
type balanceMutatingErrorMockPrecompile struct {
	addr   common.Address
	gas    uint64
	from   common.Address
	to     common.Address
	amount *uint256.Int
}

func (p *balanceMutatingErrorMockPrecompile) Address() common.Address {
	return p.addr
}

func (p *balanceMutatingErrorMockPrecompile) RequiredGas(input []byte) uint64 {
	return p.gas
}

func (p *balanceMutatingErrorMockPrecompile) Run(evm *EVM, contract *Contract, readonly bool) ([]byte, error) {
	// Intentionally ignore readonly so the tests can verify that all call modes
	// roll back state changes when the precompile returns an error.
	evm.StateDB.SubBalance(p.from, p.amount, tracing.BalanceChangeTransfer)
	evm.StateDB.AddBalance(p.to, p.amount, tracing.BalanceChangeTransfer)
	return nil, errBalanceMutatingMockPrecompile
}

func TestEVMCallRevertsPrecompileStateMutationOnError(t *testing.T) {
	const (
		suppliedGas = uint64(100)
		gasCost     = uint64(10)
	)

	callerAddr := common.HexToAddress("0x9001")
	originAddr := common.HexToAddress("0x9002")
	precompileAddr := common.HexToAddress("0x9003")
	recipientAddr := common.HexToAddress("0x9004")
	calldata := []byte{0x01}
	delegateParentAddr := common.HexToAddress("0x9101")
	delegateCallerAddr := common.HexToAddress("0x9102")

	initialCallerBalance := uint256.NewInt(100)
	transferAmount := uint256.NewInt(7)

	testCases := []struct {
		name              string
		prepareCaller     func(*EVM) ContractRef
		executePrecompile func(*EVM, ContractRef) ([]byte, uint64, error)
		expectedErr       error
		expectedGas       uint64
	}{
		{
			name: "call",
			prepareCaller: func(evm *EVM) ContractRef {
				addBalance(evm, callerAddr, initialCallerBalance)
				addBalance(evm, recipientAddr, uint256.NewInt(0))
				return AccountRef(callerAddr)
			},
			executePrecompile: func(evm *EVM, caller ContractRef) ([]byte, uint64, error) {
				return evm.Call(caller, precompileAddr, calldata, suppliedGas, uint256.NewInt(0))
			},
			expectedErr: errBalanceMutatingMockPrecompile,
			expectedGas: 0,
		},
		{
			name: "staticcall",
			prepareCaller: func(evm *EVM) ContractRef {
				addBalance(evm, callerAddr, initialCallerBalance)
				addBalance(evm, recipientAddr, uint256.NewInt(0))
				return AccountRef(callerAddr)
			},
			executePrecompile: func(evm *EVM, caller ContractRef) ([]byte, uint64, error) {
				return evm.StaticCall(caller, precompileAddr, calldata, suppliedGas)
			},
			expectedErr: errBalanceMutatingMockPrecompile,
			expectedGas: 0,
		},
		{
			name: "delegatecall",
			prepareCaller: func(evm *EVM) ContractRef {
				addBalance(evm, callerAddr, initialCallerBalance)
				addBalance(evm, recipientAddr, uint256.NewInt(0))
				evm.StateDB.CreateAccount(delegateCallerAddr)
				parent := NewContract(AccountRef(callerAddr), AccountRef(delegateParentAddr), uint256.NewInt(0), suppliedGas)
				return NewContract(parent, AccountRef(delegateCallerAddr), nil, suppliedGas)
			},
			executePrecompile: func(evm *EVM, caller ContractRef) ([]byte, uint64, error) {
				return evm.DelegateCall(caller, precompileAddr, calldata, suppliedGas)
			},
			expectedErr: errBalanceMutatingMockPrecompile,
			expectedGas: 0,
		},
		{
			name: "callcode",
			prepareCaller: func(evm *EVM) ContractRef {
				addBalance(evm, callerAddr, initialCallerBalance)
				addBalance(evm, recipientAddr, uint256.NewInt(0))
				return AccountRef(callerAddr)
			},
			executePrecompile: func(evm *EVM, caller ContractRef) ([]byte, uint64, error) {
				return evm.CallCode(caller, precompileAddr, calldata, suppliedGas, uint256.NewInt(0))
			},
			expectedErr: errBalanceMutatingMockPrecompile,
			expectedGas: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			evm := newPrecompileTestEVM(t, originAddr)
			precompile := &balanceMutatingErrorMockPrecompile{
				addr:   precompileAddr,
				gas:    gasCost,
				from:   callerAddr,
				to:     recipientAddr,
				amount: transferAmount,
			}
			evm.WithPrecompiles(map[common.Address]PrecompiledContract{precompileAddr: precompile}, []common.Address{precompileAddr})

			caller := tc.prepareCaller(evm)
			_, remainingGas, err := tc.executePrecompile(evm, caller)
			assertExpectedError(t, err, tc.expectedErr)
			if remainingGas != tc.expectedGas {
				t.Fatalf("unexpected remaining gas: want %d got %d", tc.expectedGas, remainingGas)
			}

			// Verify that the attempted state mutation is reverted.
			callerBalance := evm.StateDB.GetBalance(callerAddr)
			if callerBalance.Cmp(initialCallerBalance) != 0 {
				t.Fatalf("unexpected caller balance: want %s got %s", initialCallerBalance, callerBalance)
			}
			targetBalance := evm.StateDB.GetBalance(recipientAddr)
			if !targetBalance.IsZero() {
				t.Fatalf("unexpected recipient balance: want 0 got %s", targetBalance)
			}
		})
	}
}

// runTrackingMockPrecompile is a test precompile that records whether
// Run was entered.
type runTrackingMockPrecompile struct {
	addr        common.Address
	gas         uint64
	runExecuted bool
}

func (p *runTrackingMockPrecompile) Address() common.Address {
	return p.addr
}

func (p *runTrackingMockPrecompile) RequiredGas(input []byte) uint64 {
	return p.gas
}

func (p *runTrackingMockPrecompile) Run(evm *EVM, contract *Contract, readonly bool) ([]byte, error) {
	// This mock only tracks whether Run was entered, so it intentionally ignores
	// readonly and always succeeds if execution reaches it.
	p.runExecuted = true
	return []byte{0xaa}, nil
}

func TestEVMPrecompileOutOfGas(t *testing.T) {
	const (
		// use more gas than supplied
		suppliedGas = uint64(9)
		gasCost     = uint64(10)
	)

	callerAddr := common.HexToAddress("0x8001")
	originAddr := common.HexToAddress("0x8002")
	precompileAddr := common.HexToAddress("0x8003")
	calldata := []byte{0x01}

	delegateParentAddr := common.HexToAddress("0x8101")
	delegateCallerAddr := common.HexToAddress("0x8102")

	testCases := []struct {
		name              string
		callerAddr        common.Address
		callerBalance     *uint256.Int
		prepareCaller     func(*EVM) ContractRef
		executePrecompile func(*EVM, ContractRef) ([]byte, uint64, error)
	}{
		{
			name:          "call",
			callerAddr:    callerAddr,
			callerBalance: uint256.NewInt(100),
			prepareCaller: func(evm *EVM) ContractRef {
				addBalance(evm, callerAddr, uint256.NewInt(100))
				return AccountRef(callerAddr)
			},
			executePrecompile: func(evm *EVM, caller ContractRef) ([]byte, uint64, error) {
				return evm.Call(caller, precompileAddr, calldata, suppliedGas, uint256.NewInt(0))
			},
		},
		{
			name:          "staticcall",
			callerAddr:    callerAddr,
			callerBalance: uint256.NewInt(0),
			prepareCaller: func(evm *EVM) ContractRef {
				evm.StateDB.CreateAccount(callerAddr)
				return AccountRef(callerAddr)
			},
			executePrecompile: func(evm *EVM, caller ContractRef) ([]byte, uint64, error) {
				return evm.StaticCall(caller, precompileAddr, calldata, suppliedGas)
			},
		},
		{
			name:          "delegatecall",
			callerAddr:    delegateCallerAddr,
			callerBalance: uint256.NewInt(0),
			prepareCaller: func(evm *EVM) ContractRef {
				evm.StateDB.CreateAccount(delegateCallerAddr)
				parent := NewContract(AccountRef(callerAddr), AccountRef(delegateParentAddr), uint256.NewInt(0), suppliedGas)
				return NewContract(parent, AccountRef(delegateCallerAddr), nil, suppliedGas)
			},
			executePrecompile: func(evm *EVM, caller ContractRef) ([]byte, uint64, error) {
				return evm.DelegateCall(caller, precompileAddr, calldata, suppliedGas)
			},
		},
		{
			name:          "callcode",
			callerAddr:    callerAddr,
			callerBalance: uint256.NewInt(100),
			prepareCaller: func(evm *EVM) ContractRef {
				addBalance(evm, callerAddr, uint256.NewInt(100))
				return AccountRef(callerAddr)
			},
			executePrecompile: func(evm *EVM, caller ContractRef) ([]byte, uint64, error) {
				return evm.CallCode(caller, precompileAddr, calldata, suppliedGas, uint256.NewInt(0))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			evm := newPrecompileTestEVM(t, originAddr)
			precompile := &runTrackingMockPrecompile{addr: precompileAddr, gas: gasCost}
			evm.WithPrecompiles(map[common.Address]PrecompiledContract{precompileAddr: precompile}, []common.Address{precompileAddr})

			caller := tc.prepareCaller(evm)
			_, remainingGas, err := tc.executePrecompile(evm, caller)
			if err != ErrOutOfGas {
				t.Fatalf("unexpected error: want %v got %v", ErrOutOfGas, err)
			}
			if remainingGas != 0 {
				t.Fatalf("unexpected remaining gas: want 0 got %d", remainingGas)
			}
			if precompile.runExecuted {
				t.Fatal("precompile should not execute when gas is insufficient")
			}
			callerBalance := evm.StateDB.GetBalance(tc.callerAddr)
			if callerBalance.Cmp(tc.callerBalance) != 0 {
				t.Fatalf("unexpected caller balance: want %s got %s", tc.callerBalance, callerBalance)
			}
			if evm.StateDB.Exist(precompileAddr) {
				t.Fatal("precompile account should not remain touched after out-of-gas")
			}
		})
	}
}

func TestEVMPrecompileInsufficientBalance(t *testing.T) {
	const (
		suppliedGas = uint64(100)
		gasCost     = uint64(10)
	)

	callerAddr := common.HexToAddress("0xa001")
	originAddr := common.HexToAddress("0xa002")
	precompileAddr := common.HexToAddress("0xa003")
	calldata := []byte{0x01}
	delegateParentAddr := common.HexToAddress("0xa101")
	delegateCallerAddr := common.HexToAddress("0xa102")

	// Balance lower than call value.
	initialCallerBalance := uint256.NewInt(5)
	callValue := uint256.NewInt(13)

	testCases := []struct {
		name              string
		prepareCaller     func(*EVM) ContractRef
		executePrecompile func(*EVM, ContractRef) ([]byte, uint64, error)
		expectedErr       error
		expectedGas       uint64
		expectedRun       bool
		expectedCallerBal *uint256.Int
	}{
		{
			name: "call",
			prepareCaller: func(evm *EVM) ContractRef {
				addBalance(evm, callerAddr, initialCallerBalance)
				return AccountRef(callerAddr)
			},
			executePrecompile: func(evm *EVM, caller ContractRef) ([]byte, uint64, error) {
				return evm.Call(caller, precompileAddr, calldata, suppliedGas, callValue)
			},
			expectedErr:       ErrInsufficientBalance,
			expectedGas:       suppliedGas,
			expectedRun:       false,
			expectedCallerBal: initialCallerBalance,
		},
		{
			name: "callcode",
			prepareCaller: func(evm *EVM) ContractRef {
				addBalance(evm, callerAddr, initialCallerBalance)
				return AccountRef(callerAddr)
			},
			executePrecompile: func(evm *EVM, caller ContractRef) ([]byte, uint64, error) {
				return evm.CallCode(caller, precompileAddr, calldata, suppliedGas, callValue)
			},
			expectedErr:       ErrInsufficientBalance,
			expectedGas:       suppliedGas,
			expectedRun:       false,
			expectedCallerBal: initialCallerBalance,
		},
		{
			name: "staticcall",
			prepareCaller: func(evm *EVM) ContractRef {
				evm.StateDB.CreateAccount(callerAddr)
				return AccountRef(callerAddr)
			},
			executePrecompile: func(evm *EVM, caller ContractRef) ([]byte, uint64, error) {
				return evm.StaticCall(caller, precompileAddr, calldata, suppliedGas)
			},
			expectedGas:       suppliedGas - gasCost,
			expectedRun:       true,
			expectedCallerBal: uint256.NewInt(0),
		},
		{
			name: "delegatecall",
			prepareCaller: func(evm *EVM) ContractRef {
				evm.StateDB.CreateAccount(delegateCallerAddr)
				parent := NewContract(AccountRef(callerAddr), AccountRef(delegateParentAddr), uint256.NewInt(0), suppliedGas)
				return NewContract(parent, AccountRef(delegateCallerAddr), nil, suppliedGas)
			},
			executePrecompile: func(evm *EVM, caller ContractRef) ([]byte, uint64, error) {
				return evm.DelegateCall(caller, precompileAddr, calldata, suppliedGas)
			},
			expectedGas:       suppliedGas - gasCost,
			expectedRun:       true,
			expectedCallerBal: uint256.NewInt(0),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			evm := newPrecompileTestEVM(t, originAddr)
			precompile := &runTrackingMockPrecompile{addr: precompileAddr, gas: gasCost}
			evm.WithPrecompiles(map[common.Address]PrecompiledContract{precompileAddr: precompile}, []common.Address{precompileAddr})

			caller := tc.prepareCaller(evm)
			_, remainingGas, err := tc.executePrecompile(evm, caller)
			assertExpectedError(t, err, tc.expectedErr)
			if remainingGas != tc.expectedGas {
				t.Fatalf("unexpected remaining gas: want %d got %d", tc.expectedGas, remainingGas)
			}
			if precompile.runExecuted != tc.expectedRun {
				t.Fatalf("unexpected run flag: want %t got %t", tc.expectedRun, precompile.runExecuted)
			}
			callerBalance := evm.StateDB.GetBalance(callerAddr)
			if callerBalance.Cmp(tc.expectedCallerBal) != 0 {
				t.Fatalf("unexpected caller balance: want %s got %s", tc.expectedCallerBal, callerBalance)
			}
		})
	}
}

func TestEVMPrecompileCallDepthExceeded(t *testing.T) {
	const suppliedGas = uint64(100)

	callerAddr := common.HexToAddress("0xb001")
	originAddr := common.HexToAddress("0xb002")
	precompileAddr := common.HexToAddress("0xb003")
	calldata := []byte{0x01}

	delegateParentAddr := common.HexToAddress("0xb101")
	delegateCallerAddr := common.HexToAddress("0xb102")

	testCases := []struct {
		name              string
		prepareCaller     func(*EVM) ContractRef
		executePrecompile func(*EVM, ContractRef) ([]byte, uint64, error)
	}{
		{
			name: "call",
			prepareCaller: func(evm *EVM) ContractRef {
				addBalance(evm, callerAddr, uint256.NewInt(100))
				return AccountRef(callerAddr)
			},
			executePrecompile: func(evm *EVM, caller ContractRef) ([]byte, uint64, error) {
				return evm.Call(caller, precompileAddr, calldata, suppliedGas, uint256.NewInt(0))
			},
		},
		{
			name: "staticcall",
			prepareCaller: func(evm *EVM) ContractRef {
				evm.StateDB.CreateAccount(callerAddr)
				return AccountRef(callerAddr)
			},
			executePrecompile: func(evm *EVM, caller ContractRef) ([]byte, uint64, error) {
				return evm.StaticCall(caller, precompileAddr, calldata, suppliedGas)
			},
		},
		{
			name: "delegatecall",
			prepareCaller: func(evm *EVM) ContractRef {
				evm.StateDB.CreateAccount(delegateCallerAddr)
				parent := NewContract(AccountRef(callerAddr), AccountRef(delegateParentAddr), uint256.NewInt(0), suppliedGas)
				return NewContract(parent, AccountRef(delegateCallerAddr), nil, suppliedGas)
			},
			executePrecompile: func(evm *EVM, caller ContractRef) ([]byte, uint64, error) {
				return evm.DelegateCall(caller, precompileAddr, calldata, suppliedGas)
			},
		},
		{
			name: "callcode",
			prepareCaller: func(evm *EVM) ContractRef {
				addBalance(evm, callerAddr, uint256.NewInt(100))
				return AccountRef(callerAddr)
			},
			executePrecompile: func(evm *EVM, caller ContractRef) ([]byte, uint64, error) {
				return evm.CallCode(caller, precompileAddr, calldata, suppliedGas, uint256.NewInt(0))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			evm := newPrecompileTestEVM(t, originAddr)
			evm.depth = int(params.CallCreateDepth) + 1
			precompile := &runTrackingMockPrecompile{addr: precompileAddr, gas: 1}
			evm.WithPrecompiles(map[common.Address]PrecompiledContract{precompileAddr: precompile}, []common.Address{precompileAddr})

			caller := tc.prepareCaller(evm)
			_, remainingGas, err := tc.executePrecompile(evm, caller)
			if err != ErrDepth {
				t.Fatalf("unexpected error: want %v got %v", ErrDepth, err)
			}
			if remainingGas != suppliedGas {
				t.Fatalf("unexpected remaining gas: want %d got %d", suppliedGas, remainingGas)
			}
			if precompile.runExecuted {
				t.Fatal("precompile should not execute when call depth is exceeded")
			}
		})
	}
}

// Benchmarks the sample inputs from the ECRECOVER precompile.
func BenchmarkPrecompiledEcrecover(bench *testing.B) {
	t := precompiledTest{
		Input:    "38d18acb67d25c8bb9942764b62f18e17054f66a817bd4295423adf9ed98873e000000000000000000000000000000000000000000000000000000000000001b38d18acb67d25c8bb9942764b62f18e17054f66a817bd4295423adf9ed98873e789d1dd423d25f0772d2748d60f7e4b81bb14d086eba8e8e8efb6dcff8a4ae02",
		Expected: "000000000000000000000000ceaccac640adf55b2028469bd36ba501f28b699d",
		Name:     "",
	}
	benchmarkPrecompiled("01", t, bench)
}

// Benchmarks the sample inputs from the SHA256 precompile.
func BenchmarkPrecompiledSha256(bench *testing.B) {
	t := precompiledTest{
		Input:    "38d18acb67d25c8bb9942764b62f18e17054f66a817bd4295423adf9ed98873e000000000000000000000000000000000000000000000000000000000000001b38d18acb67d25c8bb9942764b62f18e17054f66a817bd4295423adf9ed98873e789d1dd423d25f0772d2748d60f7e4b81bb14d086eba8e8e8efb6dcff8a4ae02",
		Expected: "811c7003375852fabd0d362e40e68607a12bdabae61a7d068fe5fdd1dbbf2a5d",
		Name:     "128",
	}
	benchmarkPrecompiled("02", t, bench)
}

// Benchmarks the sample inputs from the RIPEMD precompile.
func BenchmarkPrecompiledRipeMD(bench *testing.B) {
	t := precompiledTest{
		Input:    "38d18acb67d25c8bb9942764b62f18e17054f66a817bd4295423adf9ed98873e000000000000000000000000000000000000000000000000000000000000001b38d18acb67d25c8bb9942764b62f18e17054f66a817bd4295423adf9ed98873e789d1dd423d25f0772d2748d60f7e4b81bb14d086eba8e8e8efb6dcff8a4ae02",
		Expected: "0000000000000000000000009215b8d9882ff46f0dfde6684d78e831467f65e6",
		Name:     "128",
	}
	benchmarkPrecompiled("03", t, bench)
}

// Benchmarks the sample inputs from the identity precompile.
func BenchmarkPrecompiledIdentity(bench *testing.B) {
	t := precompiledTest{
		Input:    "38d18acb67d25c8bb9942764b62f18e17054f66a817bd4295423adf9ed98873e000000000000000000000000000000000000000000000000000000000000001b38d18acb67d25c8bb9942764b62f18e17054f66a817bd4295423adf9ed98873e789d1dd423d25f0772d2748d60f7e4b81bb14d086eba8e8e8efb6dcff8a4ae02",
		Expected: "38d18acb67d25c8bb9942764b62f18e17054f66a817bd4295423adf9ed98873e000000000000000000000000000000000000000000000000000000000000001b38d18acb67d25c8bb9942764b62f18e17054f66a817bd4295423adf9ed98873e789d1dd423d25f0772d2748d60f7e4b81bb14d086eba8e8e8efb6dcff8a4ae02",
		Name:     "128",
	}
	benchmarkPrecompiled("04", t, bench)
}

// Tests the sample inputs from the ModExp EIP 198.
func TestPrecompiledModExp(t *testing.T)      { testJson("modexp", "05", t) }
func BenchmarkPrecompiledModExp(b *testing.B) { benchJson("modexp", "05", b) }

func TestPrecompiledModExpEip2565(t *testing.T)      { testJson("modexp_eip2565", "f5", t) }
func BenchmarkPrecompiledModExpEip2565(b *testing.B) { benchJson("modexp_eip2565", "f5", b) }

// Tests the sample inputs from the elliptic curve addition EIP 213.
func TestPrecompiledBn256Add(t *testing.T)      { testJson("bn256Add", "06", t) }
func BenchmarkPrecompiledBn256Add(b *testing.B) { benchJson("bn256Add", "06", b) }

// Tests OOG
func TestPrecompiledModExpOOG(t *testing.T) {
	modexpTests, err := loadJson("modexp")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range modexpTests {
		testPrecompiledOOG("05", test, t)
	}
}

// Tests the sample inputs from the elliptic curve scalar multiplication EIP 213.
func TestPrecompiledBn256ScalarMul(t *testing.T)      { testJson("bn256ScalarMul", "07", t) }
func BenchmarkPrecompiledBn256ScalarMul(b *testing.B) { benchJson("bn256ScalarMul", "07", b) }

// Tests the sample inputs from the elliptic curve pairing check EIP 197.
func TestPrecompiledBn256Pairing(t *testing.T)      { testJson("bn256Pairing", "08", t) }
func BenchmarkPrecompiledBn256Pairing(b *testing.B) { benchJson("bn256Pairing", "08", b) }

func TestPrecompiledBlake2F(t *testing.T)      { testJson("blake2F", "09", t) }
func BenchmarkPrecompiledBlake2F(b *testing.B) { benchJson("blake2F", "09", b) }

func TestPrecompileBlake2FMalformedInput(t *testing.T) {
	for _, test := range blake2FMalformedInputTests {
		testPrecompiledFailure("09", test, t)
	}
}

func TestPrecompiledEcrecover(t *testing.T) { testJson("ecRecover", "01", t) }

func testJson(name, addr string, t *testing.T) {
	tests, err := loadJson(name)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range tests {
		testPrecompiled(addr, test, t)
	}
}

func testJsonFail(name, addr string, t *testing.T) {
	tests, err := loadJsonFail(name)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range tests {
		testPrecompiledFailure(addr, test, t)
	}
}

func benchJson(name, addr string, b *testing.B) {
	tests, err := loadJson(name)
	if err != nil {
		b.Fatal(err)
	}
	for _, test := range tests {
		benchmarkPrecompiled(addr, test, b)
	}
}

func TestPrecompiledBLS12381G1Add(t *testing.T)      { testJson("blsG1Add", "f0a", t) }
func TestPrecompiledBLS12381G1Mul(t *testing.T)      { testJson("blsG1Mul", "f0b", t) }
func TestPrecompiledBLS12381G1MultiExp(t *testing.T) { testJson("blsG1MultiExp", "f0c", t) }
func TestPrecompiledBLS12381G2Add(t *testing.T)      { testJson("blsG2Add", "f0d", t) }
func TestPrecompiledBLS12381G2Mul(t *testing.T)      { testJson("blsG2Mul", "f0e", t) }
func TestPrecompiledBLS12381G2MultiExp(t *testing.T) { testJson("blsG2MultiExp", "f0f", t) }
func TestPrecompiledBLS12381Pairing(t *testing.T)    { testJson("blsPairing", "f10", t) }
func TestPrecompiledBLS12381MapG1(t *testing.T)      { testJson("blsMapG1", "f11", t) }
func TestPrecompiledBLS12381MapG2(t *testing.T)      { testJson("blsMapG2", "f12", t) }

func TestPrecompiledPointEvaluation(t *testing.T) { testJson("pointEvaluation", "0a", t) }

func BenchmarkPrecompiledPointEvaluation(b *testing.B) { benchJson("pointEvaluation", "0a", b) }

func BenchmarkPrecompiledBLS12381G1Add(b *testing.B)      { benchJson("blsG1Add", "f0a", b) }
func BenchmarkPrecompiledBLS12381G1Mul(b *testing.B)      { benchJson("blsG1Mul", "f0b", b) }
func BenchmarkPrecompiledBLS12381G1MultiExp(b *testing.B) { benchJson("blsG1MultiExp", "f0c", b) }
func BenchmarkPrecompiledBLS12381G2Add(b *testing.B)      { benchJson("blsG2Add", "f0d", b) }
func BenchmarkPrecompiledBLS12381G2Mul(b *testing.B)      { benchJson("blsG2Mul", "f0e", b) }
func BenchmarkPrecompiledBLS12381G2MultiExp(b *testing.B) { benchJson("blsG2MultiExp", "f0f", b) }
func BenchmarkPrecompiledBLS12381Pairing(b *testing.B)    { benchJson("blsPairing", "f10", b) }
func BenchmarkPrecompiledBLS12381MapG1(b *testing.B)      { benchJson("blsMapG1", "f11", b) }
func BenchmarkPrecompiledBLS12381MapG2(b *testing.B)      { benchJson("blsMapG2", "f12", b) }

// Failure tests
func TestPrecompiledBLS12381G1AddFail(t *testing.T)      { testJsonFail("blsG1Add", "f0a", t) }
func TestPrecompiledBLS12381G1MulFail(t *testing.T)      { testJsonFail("blsG1Mul", "f0b", t) }
func TestPrecompiledBLS12381G1MultiExpFail(t *testing.T) { testJsonFail("blsG1MultiExp", "f0c", t) }
func TestPrecompiledBLS12381G2AddFail(t *testing.T)      { testJsonFail("blsG2Add", "f0d", t) }
func TestPrecompiledBLS12381G2MulFail(t *testing.T)      { testJsonFail("blsG2Mul", "f0e", t) }
func TestPrecompiledBLS12381G2MultiExpFail(t *testing.T) { testJsonFail("blsG2MultiExp", "f0f", t) }
func TestPrecompiledBLS12381PairingFail(t *testing.T)    { testJsonFail("blsPairing", "f10", t) }
func TestPrecompiledBLS12381MapG1Fail(t *testing.T)      { testJsonFail("blsMapG1", "f11", t) }
func TestPrecompiledBLS12381MapG2Fail(t *testing.T)      { testJsonFail("blsMapG2", "f12", t) }

func loadJson(name string) ([]precompiledTest, error) {
	data, err := os.ReadFile(fmt.Sprintf("testdata/precompiles/%v.json", name))
	if err != nil {
		return nil, err
	}
	var testcases []precompiledTest
	err = json.Unmarshal(data, &testcases)
	return testcases, err
}

func loadJsonFail(name string) ([]precompiledFailureTest, error) {
	data, err := os.ReadFile(fmt.Sprintf("testdata/precompiles/fail-%v.json", name))
	if err != nil {
		return nil, err
	}
	var testcases []precompiledFailureTest
	err = json.Unmarshal(data, &testcases)
	return testcases, err
}

// BenchmarkPrecompiledBLS12381G1MultiExpWorstCase benchmarks the worst case we could find that still fits a gaslimit of 10MGas.
func BenchmarkPrecompiledBLS12381G1MultiExpWorstCase(b *testing.B) {
	task := "0000000000000000000000000000000008d8c4a16fb9d8800cce987c0eadbb6b3b005c213d44ecb5adeed713bae79d606041406df26169c35df63cf972c94be1" +
		"0000000000000000000000000000000011bc8afe71676e6730702a46ef817060249cd06cd82e6981085012ff6d013aa4470ba3a2c71e13ef653e1e223d1ccfe9" +
		"FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"
	input := task
	for i := 0; i < 4787; i++ {
		input = input + task
	}
	testcase := precompiledTest{
		Input:       input,
		Expected:    "0000000000000000000000000000000005a6310ea6f2a598023ae48819afc292b4dfcb40aabad24a0c2cb6c19769465691859eeb2a764342a810c5038d700f18000000000000000000000000000000001268ac944437d15923dc0aec00daa9250252e43e4b35ec7a19d01f0d6cd27f6e139d80dae16ba1c79cc7f57055a93ff5",
		Name:        "WorstCaseG1",
		NoBenchmark: false,
	}
	benchmarkPrecompiled("f0c", testcase, b)
}

// BenchmarkPrecompiledBLS12381G2MultiExpWorstCase benchmarks the worst case we could find that still fits a gaslimit of 10MGas.
func BenchmarkPrecompiledBLS12381G2MultiExpWorstCase(b *testing.B) {
	task := "000000000000000000000000000000000d4f09acd5f362e0a516d4c13c5e2f504d9bd49fdfb6d8b7a7ab35a02c391c8112b03270d5d9eefe9b659dd27601d18f" +
		"000000000000000000000000000000000fd489cb75945f3b5ebb1c0e326d59602934c8f78fe9294a8877e7aeb95de5addde0cb7ab53674df8b2cfbb036b30b99" +
		"00000000000000000000000000000000055dbc4eca768714e098bbe9c71cf54b40f51c26e95808ee79225a87fb6fa1415178db47f02d856fea56a752d185f86b" +
		"000000000000000000000000000000001239b7640f416eb6e921fe47f7501d504fadc190d9cf4e89ae2b717276739a2f4ee9f637c35e23c480df029fd8d247c7" +
		"FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"
	input := task
	for i := 0; i < 1040; i++ {
		input = input + task
	}

	testcase := precompiledTest{
		Input:       input,
		Expected:    "0000000000000000000000000000000018f5ea0c8b086095cfe23f6bb1d90d45de929292006dba8cdedd6d3203af3c6bbfd592e93ecb2b2c81004961fdcbb46c00000000000000000000000000000000076873199175664f1b6493a43c02234f49dc66f077d3007823e0343ad92e30bd7dc209013435ca9f197aca44d88e9dac000000000000000000000000000000000e6f07f4b23b511eac1e2682a0fc224c15d80e122a3e222d00a41fab15eba645a700b9ae84f331ae4ed873678e2e6c9b000000000000000000000000000000000bcb4849e460612aaed79617255fd30c03f51cf03d2ed4163ca810c13e1954b1e8663157b957a601829bb272a4e6c7b8",
		Name:        "WorstCaseG2",
		NoBenchmark: false,
	}
	benchmarkPrecompiled("f0f", testcase, b)
}
