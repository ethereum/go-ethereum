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
	"bytes"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

func assertionTestEVM(t *testing.T) (*EVM, *state.StateDB) {
	t.Helper()
	db, err := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	if err != nil {
		t.Fatal(err)
	}
	cfg := amsterdam8037Config()
	cfg.BogotaTime = new(uint64)
	evm := NewEVM(amsterdam8037EVM(db).Context, db, cfg, Config{})
	evm.TxContext.FrameContext = &FrameContext{Frames: []types.Frame{{Mode: types.ModePostTx}, {Mode: types.ModePostTx}}, MaxCost: uint256.NewInt(12345)}
	db.SetTxContext(common.HexToHash("0x123"), 0, 1)
	return evm, db
}

func assertionOp(evm *EVM, op OpCode, operands ...*uint256.Int) (uint256.Int, []byte, error) {
	scope := &ScopeContext{Stack: newStackForTesting(), Memory: NewMemory()}
	defer scope.Memory.Free()
	scope.Memory.Resize(128)
	for i := len(operands) - 1; i >= 0; i-- {
		scope.Stack.push(operands[i])
	}
	pc := uint64(0)
	_, err := bogotaInstructionSet[op].execute(&pc, evm, scope)
	var result uint256.Int
	if scope.Stack.len() != 0 {
		result = scope.Stack.pop()
	}
	return result, bytes.Clone(scope.Memory.Data()), err
}

func TestAssertionTables(t *testing.T) {
	evm, db := assertionTestEVM(t)
	a, b := common.HexToAddress("0x01"), common.HexToAddress("0x02")
	k1, k2 := common.HexToHash("0x03"), common.HexToHash("0x04")
	db.SetBalance(a, uint256.NewInt(10), tracing.BalanceChangeUnspecified)
	db.SetState(a, k1, k1)
	db.Finalise(params.Rules{})
	db.SetState(b, k2, k1)
	db.SetState(a, k2, k1)
	db.SetState(a, k1, k2)
	db.SetBalance(a, uint256.NewInt(8), tracing.BalanceChangeUnspecified)
	db.SetCode(b, []byte{0}, tracing.CodeChangeUnspecified)
	db.AddLog(&types.Log{Address: b, Topics: []common.Hash{k1, k2, k2}, Data: []byte{10, 11, 12}})
	db.AddLog(&types.Log{Address: a, Topics: []common.Hash{k2, k1}})
	evm.TxContext.FrameContext.Payer = &a
	cases := []struct {
		op   OpCode
		args []uint64
		want uint64
	}{
		{TXTRACE, []uint64{0, 0}, 1}, {TXTRACE, []uint64{1, 0}, 3}, {TXTRACE, []uint64{2, 0}, 1},
		{TXTRACE, []uint64{3, 0}, 1}, {TXTRACE, []uint64{4, 0}, 10}, {TXTRACE, []uint64{5, 0}, 8},
		{TXTRACE, []uint64{6, 0}, 1}, {TXTRACE, []uint64{7, 0}, 3}, {TXTRACE, []uint64{8, 0}, 3}, {TXTRACE, []uint64{9, 0}, 4},
		{TXTRACE, []uint64{6, 1}, 1}, {TXTRACE, []uint64{7, 1}, 4}, {TXTRACE, []uint64{6, 2}, 2},
		{TXTRACE, []uint64{0x0a, 0}, 2}, {TXTRACE, []uint64{0x0c, 0}, 2},
		{TXTRACE, []uint64{0x0d, 0}, 2}, {TXTRACE, []uint64{0x0e, 0}, 3},
		{TXTRACE, []uint64{0x0f, 0}, 3}, {TXTRACE, []uint64{0x10, 0}, 4}, {TXTRACE, []uint64{0x11, 0}, 4},
		{TXTRACE, []uint64{0x13, 0}, 3}, {TXTRACE, []uint64{0x14, 0}, 12345}, {TXTRACE, []uint64{0x15, 0}, 1},
		{TXDIFF, []uint64{0, 1, 3}, 3}, {TXDIFF, []uint64{1, 1, 3}, 4},
		{TXDIFF, []uint64{2, 1, 0}, 10}, {TXDIFF, []uint64{3, 1, 0}, 8},
		{TXDIFF, []uint64{6, 1, 0}, 2}, {TXDIFF, []uint64{7, 1, 1}, 1}, {TXDIFF, []uint64{7, 2, 0}, 2},
		{TXDIFF, []uint64{8, 1, 0}, 1}, {TXDIFF, []uint64{9, 1, 0}, 1}, {TXDIFF, []uint64{9, 2, 0}, 0},
		{TXDIFF, []uint64{0x0a, 1, 0}, 6}, {TXDIFF, []uint64{0x0a, 2, 0}, 12},
		{TXDIFF, []uint64{0x0b, 4, 0}, 1}, {TXDIFF, []uint64{0x0c, 4, 0}, 0}, {TXDIFF, []uint64{0x0c, 3, 0}, 1},
	}
	for _, tc := range cases {
		args := make([]*uint256.Int, len(tc.args))
		for i, v := range tc.args {
			args[i] = uint256.NewInt(v)
		}
		got, _, err := assertionOp(evm, tc.op, args...)
		if err != nil || got != *uint256.NewInt(tc.want) {
			t.Errorf("%s %v: %v, %v; want %d", tc.op, tc.args, got, err, tc.want)
		}
	}
	for _, p := range []uint64{4, 5} {
		got, _, err := assertionOp(evm, TXDIFF, uint256.NewInt(p), uint256.NewInt(99), new(uint256.Int))
		if err != nil || common.Hash(got.Bytes32()) != types.EmptyCodeHash {
			t.Fatalf("absent code hash: %x, %v", got, err)
		}
	}
	// A different frame must see fresh tables, even if the old frame read them.
	db.AddLog(&types.Log{Address: a})
	evm.TxContext.FrameContext.CurrentFrame++
	evm.depth = 2
	got, _, err := assertionOp(evm, TXTRACE, uint256.NewInt(0x0c), new(uint256.Int))
	if err != nil || got.Uint64() != 3 {
		t.Fatalf("next frame/nested call: %v, %v", got, err)
	}
	evm.TxContext.FrameContext.Payer = nil
	got, _, err = assertionOp(evm, TXTRACE, uint256.NewInt(0x15), new(uint256.Int))
	if err != nil || !got.IsZero() {
		t.Fatal("nil payer is not zero")
	}
}

func TestAssertionHaltsAndCopy(t *testing.T) {
	evm, db := assertionTestEVM(t)
	db.AddLog(&types.Log{Data: []byte{10, 11, 12}})
	zero, one := new(uint256.Int), uint256.NewInt(1)
	huge := new(uint256.Int).Lsh(one, 64)
	max := new(uint256.Int).Not(zero)
	cases := []struct {
		op   OpCode
		args []*uint256.Int
	}{
		{TXTRACE, []*uint256.Int{uint256.NewInt(0x16), zero}},
		{TXTRACE, []*uint256.Int{huge, zero}},
		{TXTRACE, []*uint256.Int{zero, one}},
		{TXTRACE, []*uint256.Int{uint256.NewInt(0x0d), huge}},
		{TXTRACE, []*uint256.Int{uint256.NewInt(0x0d), one}},
		{TXTRACE, []*uint256.Int{uint256.NewInt(0x0f), zero}},
		{TXDIFF, []*uint256.Int{uint256.NewInt(13), zero, zero}},
		{TXDIFF, []*uint256.Int{huge, zero, zero}},
		{TXDIFF, []*uint256.Int{uint256.NewInt(2), zero, one}},
		{TXDIFF, []*uint256.Int{uint256.NewInt(7), zero, huge}},
		{TXDIFF, []*uint256.Int{uint256.NewInt(9), zero, one}},
		{TXDIFF, []*uint256.Int{uint256.NewInt(0x0c), zero, zero}},
		{EVENTDATACOPY, []*uint256.Int{huge, zero, zero, zero}},
		{EVENTDATACOPY, []*uint256.Int{zero, zero, uint256.NewInt(4), zero}},
		{EVENTDATACOPY, []*uint256.Int{zero, zero, huge, zero}},
		{EVENTDATACOPY, []*uint256.Int{zero, zero, max, one}},
		{EVENTDATACOPY, []*uint256.Int{zero, zero, one, max}},
		{EVENTDATACOPY, []*uint256.Int{zero, zero, one, uint256.NewInt(3)}},
	}
	for _, tc := range cases {
		if _, _, err := assertionOp(evm, tc.op, tc.args...); !errors.Is(err, errInvalidAssertion) {
			t.Errorf("%s %v: %v", tc.op, tc.args, err)
		}
	}
	_, mem, err := assertionOp(evm, EVENTDATACOPY, zero, uint256.NewInt(31), one, uint256.NewInt(2))
	if err != nil || !bytes.Equal(mem[31:33], []byte{11, 12}) {
		t.Fatalf("copy: %x, %v", mem[31:33], err)
	}
	if _, _, err := assertionOp(evm, EVENTDATACOPY, zero, max, uint256.NewInt(3), zero); err != nil {
		t.Fatalf("zero length at end: %v", err)
	}
	for _, mode := range []uint64{types.ModeDefault, types.ModeVerify, types.ModeSender, types.ModePostTx + 1} {
		evm.TxContext.FrameContext.Frames[0].Mode = mode
		for _, op := range []OpCode{TXTRACE, TXDIFF, EVENTDATACOPY} {
			if _, _, err := assertionOp(evm, op, zero, zero, zero, zero); !errors.Is(err, errInvalidAssertion) {
				t.Fatalf("mode %d %s: %v", mode, op, err)
			}
		}
	}
	evm.TxContext.FrameContext = nil
	if _, _, err := assertionOp(evm, TXTRACE, zero, zero); !errors.Is(err, errInvalidAssertion) {
		t.Fatalf("legacy context: %v", err)
	}
}

// assertionReads checks the live-getter boundary separately from table reads.
type assertionReads struct {
	StateDB
	reads int
}

func (s *assertionReads) GetBalance(a common.Address) *uint256.Int {
	s.reads++
	return s.StateDB.GetBalance(a)
}
func (s *assertionReads) GetCodeHash(a common.Address) common.Hash {
	s.reads++
	return s.StateDB.GetCodeHash(a)
}
func (s *assertionReads) GetState(a common.Address, k common.Hash) common.Hash {
	s.reads++
	return s.StateDB.GetState(a, k)
}

func TestAssertionGasAndAccess(t *testing.T) {
	for _, p := range []uint64{0, 1, 2, 3, 4, 5, 6, 8, 10, 11, 13} {
		t.Run(uint256.NewInt(p).String(), func(t *testing.T) {
			evm, db := assertionTestEVM(t)
			reads := &assertionReads{StateDB: db}
			evm.StateDB = reads
			address := common.HexToAddress("0x22")
			scope := &ScopeContext{Stack: newStackForTesting(), Memory: NewMemory(), Contract: NewContract(common.Address{}, common.Address{}, new(uint256.Int), NewGasBudget(100000, 0), nil)}
			defer scope.Memory.Free()
			scope.Stack.push(new(uint256.Int))
			scope.Stack.push(uint256.NewInt(0x22))
			scope.Stack.push(uint256.NewInt(p))
			snapshot := db.Snapshot()
			gas, err := gasTxDiff(evm, scope.Contract, scope.Stack, scope.Memory, 0)
			want := params.WarmAccountAccessAmsterdam
			if p < 2 {
				want = params.ColdStorageAccessAmsterdam
			} else if p < 6 {
				want = params.ColdAccountAccessAmsterdam
			}
			if err != nil || gas.ExecutionGas != want || reads.reads != 0 {
				t.Fatalf("cold cost %v, %v; reads %d", gas, err, reads.reads)
			}
			account, slot := db.SlotInAccessList(address, common.Hash{})
			if account != (p >= 2 && p < 6) || slot != (p < 2) {
				t.Fatalf("warmth: %v %v", account, slot)
			}
			gas, _ = gasTxDiff(evm, scope.Contract, scope.Stack, scope.Memory, 0)
			if gas.ExecutionGas != params.WarmAccountAccessAmsterdam {
				t.Fatalf("warm cost: %v", gas)
			}
			pc := uint64(0)
			_, err = opTxDiff(&pc, evm, scope)
			if p <= 12 && err != nil {
				t.Fatal(err)
			}
			wantReads := 0
			if p < 6 {
				wantReads = 1
			}
			if reads.reads != wantReads {
				t.Fatalf("live reads: %d", reads.reads)
			}
			db.RevertToSnapshot(snapshot)
			if a, s := db.SlotInAccessList(address, common.Hash{}); a || s {
				t.Fatal("warmth survived revert")
			}
		})
	}
	// Interpreter-level checks ensure gas precedes invalid context/operands.
	for _, op := range []OpCode{TXTRACE, TXDIFF, EVENTDATACOPY} {
		for _, gas := range []uint64{0, 10000} {
			evm, db := assertionTestEVM(t)
			reads := &assertionReads{StateDB: db}
			evm.StateDB = reads
			evm.TxContext.FrameContext = nil
			code := []byte{byte(PUSH0), byte(PUSH0)}
			if op != TXTRACE {
				code = append(code, byte(PUSH0))
			}
			if op == EVENTDATACOPY {
				code = append(code, byte(PUSH0))
			}
			code = append(code, byte(op))
			budget := gas
			if gas == 0 {
				opCost := params.WarmAccountAccessAmsterdam
				if op == TXDIFF {
					opCost = params.ColdStorageAccessAmsterdam
				} else if op == EVENTDATACOPY {
					opCost = GasFastestStep
				}
				budget = uint64(len(code)-1)*GasQuickStep + opCost - 1
			}
			contract := NewContract(common.Address{}, common.Address{}, new(uint256.Int), NewGasBudget(budget, 0), nil)
			contract.Code = code
			_, err := evm.Run(contract, nil, true)
			want := errInvalidAssertion
			if gas == 0 {
				want = ErrOutOfGas
			}
			if !errors.Is(err, want) || reads.reads != 0 {
				t.Fatalf("%s gas %d: %v reads %d", op, gas, err, reads.reads)
			}
		}
	}
}

func TestAssertionForkActivation(t *testing.T) {
	for _, op := range []OpCode{TXTRACE, TXDIFF, EVENTDATACOPY} {
		evm, _ := assertionTestEVM(t)
		cfg := amsterdam8037Config()
		old := NewEVM(evm.Context, evm.StateDB, cfg, Config{})
		contract := NewContract(common.Address{}, common.Address{}, new(uint256.Int), NewGasBudget(10000, 0), nil)
		contract.Code = []byte{byte(op)}
		_, err := old.Run(contract, nil, true)
		var invalid *ErrInvalidOpCode
		if !errors.As(err, &invalid) {
			t.Fatalf("pre-Bogota %s: %v", op, err)
		}
	}
}

func TestAssertionCopyGas(t *testing.T) {
	evm, db := assertionTestEVM(t)
	db.AddLog(&types.Log{Data: make([]byte, 64)})
	contract := NewContract(common.Address{}, common.Address{}, new(uint256.Int), NewGasBudget(1000, 0), nil)
	contract.Code = []byte{byte(PUSH1), 33, byte(PUSH0), byte(PUSH1), 31, byte(PUSH0), byte(EVENTDATACOPY)}
	if _, err := evm.Run(contract, nil, true); err != nil {
		t.Fatal(err)
	}
	// PUSH operands (10), copy base (3), two copied words (6), two memory words (6).
	if contract.Gas.UsedExecutionGas != 25 {
		t.Fatalf("EVENTDATACOPY gas = %d, want 25", contract.Gas.UsedExecutionGas)
	}
}

func TestAssertionReservedInputsDoNotReadState(t *testing.T) {
	evm, db := assertionTestEVM(t)
	reads := &assertionReads{StateDB: db}
	evm.StateDB = reads
	for p := uint64(2); p <= 5; p++ {
		if _, _, err := assertionOp(evm, TXDIFF, uint256.NewInt(p), uint256.NewInt(42), uint256.NewInt(1)); !errors.Is(err, errInvalidAssertion) {
			t.Fatalf("reserved input, param %d: %v", p, err)
		}
	}
	if reads.reads != 0 {
		t.Fatalf("reserved inputs recorded %d state reads", reads.reads)
	}
}
