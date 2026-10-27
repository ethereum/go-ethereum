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
	"github.com/ethereum/go-ethereum/crypto"
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
	evm.TxContext.FrameContext = &FrameContext{Frames: []types.Frame{{Mode: types.ModeAssert}, {Mode: types.ModeAssert}}, MaxCost: uint256.NewInt(12345)}
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
	db.SetNonce(a, 5, tracing.NonceChangeUnspecified)
	db.SetCode(b, []byte{0}, tracing.CodeChangeUnspecified)
	db.AddLog(&types.Log{Address: b, Topics: []common.Hash{k1, k2, k2}, Data: []byte{10, 11, 12}})
	db.AddLog(&types.Log{Address: a, Topics: []common.Hash{k2, k1}})
	evm.TxContext.FrameContext.Payer = &a
	cases := []struct {
		param, index uint64
		want         uint64
	}{
		// Table counts: a's balance, a's two slots and b's slot, b's code,
		// a's nonce, and the two events.
		{0x00, 0, 1}, {0x01, 0, 3}, {0x02, 0, 1}, {0x03, 0, 1}, {0x11, 0, 2},
		// Balances: address, before, after.
		{0x04, 0, 1}, {0x05, 0, 10}, {0x06, 0, 8},
		// Slots, sorted by address then key: address, before, after, key.
		{0x07, 0, 1}, {0x08, 0, 3}, {0x09, 0, 4}, {0x0a, 0, 3},
		{0x07, 1, 1}, {0x08, 1, 0}, {0x09, 1, 3}, {0x0a, 1, 4},
		{0x07, 2, 2}, {0x0a, 2, 4},
		// Codes and nonces: address first.
		{0x0b, 0, 2}, {0x0e, 0, 1}, {0x0f, 0, 0}, {0x10, 0, 5},
		// Events in emission order.
		{0x12, 0, 2}, {0x13, 0, 3}, {0x14, 0, 3}, {0x15, 0, 4}, {0x16, 0, 4}, {0x18, 0, 3},
		{0x12, 1, 1}, {0x13, 1, 2}, {0x14, 1, 4}, {0x15, 1, 3}, {0x18, 1, 0},
		// Gas payment.
		{0x19, 0, 12345}, {0x1a, 0, 1},
	}
	for _, tc := range cases {
		got, _, err := assertionOp(evm, TXTRACE, uint256.NewInt(tc.param), uint256.NewInt(tc.index))
		if err != nil || got != *uint256.NewInt(tc.want) {
			t.Errorf("TXTRACE %#x %d: %v, %v; want %d", tc.param, tc.index, got, err, tc.want)
		}
	}
	for param, want := range map[uint64]common.Hash{0x0c: types.EmptyCodeHash, 0x0d: crypto.Keccak256Hash([]byte{0})} {
		got, _, err := assertionOp(evm, TXTRACE, uint256.NewInt(param), new(uint256.Int))
		if err != nil || common.Hash(got.Bytes32()) != want {
			t.Errorf("TXTRACE %#x: %x, %v; want %x", param, got.Bytes32(), err, want)
		}
	}
	// A different frame must see fresh tables, even if the old frame read them.
	db.AddLog(&types.Log{Address: a})
	evm.TxContext.FrameContext.CurrentFrame++
	evm.depth = 2
	got, _, err := assertionOp(evm, TXTRACE, uint256.NewInt(0x11), new(uint256.Int))
	if err != nil || got.Uint64() != 3 {
		t.Fatalf("next frame/nested call: %v, %v", got, err)
	}
	evm.TxContext.FrameContext.Payer = nil
	got, _, err = assertionOp(evm, TXTRACE, uint256.NewInt(0x1a), new(uint256.Int))
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
		{TXTRACE, []*uint256.Int{uint256.NewInt(0x1b), zero}},
		{TXTRACE, []*uint256.Int{huge, zero}},
		{TXTRACE, []*uint256.Int{zero, one}},
		{TXTRACE, []*uint256.Int{uint256.NewInt(0x19), one}},
		{TXTRACE, []*uint256.Int{uint256.NewInt(0x04), zero}},
		{TXTRACE, []*uint256.Int{uint256.NewInt(0x12), huge}},
		{TXTRACE, []*uint256.Int{uint256.NewInt(0x12), one}},
		{TXTRACE, []*uint256.Int{uint256.NewInt(0x14), zero}},
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
	for _, mode := range []uint64{types.ModeDefault, types.ModeVerify, types.ModeSender, types.ModeAssert + 1} {
		evm.TxContext.FrameContext.Frames[0].Mode = mode
		for _, op := range []OpCode{TXTRACE, EVENTDATACOPY} {
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

// assertionReads counts live state reads, which the assertion instructions
// must never perform: they read only the transaction's own diff and logs.
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

// TestAssertionGasPrecedesContext checks that the instructions charge their
// gas before validating their context and operands.
func TestAssertionGasPrecedesContext(t *testing.T) {
	for _, op := range []OpCode{TXTRACE, EVENTDATACOPY} {
		for _, gas := range []uint64{0, 10000} {
			evm, db := assertionTestEVM(t)
			reads := &assertionReads{StateDB: db}
			evm.StateDB = reads
			evm.TxContext.FrameContext = nil
			code := []byte{byte(PUSH0), byte(PUSH0)}
			opCost := params.WarmAccountAccessAmsterdam
			if op == EVENTDATACOPY {
				code = append(code, byte(PUSH0), byte(PUSH0))
				opCost = GasFastestStep
			}
			code = append(code, byte(op))
			budget := gas
			if gas == 0 {
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
	for _, op := range []OpCode{TXTRACE, EVENTDATACOPY} {
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
