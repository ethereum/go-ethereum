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
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

var (
	self8298     = common.BytesToAddress([]byte("self"))
	source8298   = common.BytesToAddress([]byte("source"))
	template8298 = []byte{0x60, 0x2a, 0x60, 0x00, 0x52, 0x60, 0x20, 0x60, 0x00, 0xf3} // returns 42

	// initcode8298 adopts the template, then returns 0xEF-prefixed data that
	// would fail the deposit checks.
	initcode8298 = append(push20(source8298), byte(SETCODEFROM), byte(POP),
		byte(PUSH1), 0xef, byte(PUSH0), byte(MSTORE8), byte(PUSH1), 1, byte(PUSH0), byte(RETURN))
)

// newEVM8298 builds an EVM at Bogota, or at Amsterdam with the given extra EIPs.
func newEVM8298(statedb StateDB, bogota bool, extraEips ...int) *EVM {
	cfg := amsterdam8037Config()
	cfg.BogotaTime = nil
	if bogota {
		cfg.BogotaTime = new(uint64)
	}
	ctx := BlockContext{
		CanTransfer:      func(StateDB, common.Address, *uint256.Int) bool { return true },
		Transfer:         func(StateDB, common.Address, common.Address, *uint256.Int, *params.Rules) {},
		BlockNumber:      big.NewInt(0),
		Random:           &common.Hash{},
		CostPerStateByte: params.CostPerStateByte,
	}
	return NewEVM(ctx, statedb, cfg, Config{ExtraEips: extraEips})
}

// push20 pushes an address.
func push20(addr common.Address) []byte { return append([]byte{byte(PUSH20)}, addr.Bytes()...) }

// setCodeFromRet runs SETCODEFROM on source and returns its result word.
func setCodeFromRet(source common.Address) []byte {
	return append(push20(source), byte(SETCODEFROM), byte(PUSH0), byte(MSTORE), byte(PUSH1), 32, byte(PUSH0), byte(RETURN))
}

// state8298 returns a state where self runs code and source holds the template.
func state8298(code []byte) *state.StateDB {
	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	statedb.SetCode(self8298, code, tracing.CodeChangeUnspecified)
	statedb.SetCode(source8298, template8298, tracing.CodeChangeUnspecified)
	statedb.Finalise(params.Rules{IsEIP158: true})
	return statedb
}

func TestSetCodeFromGas(t *testing.T) {
	code := append(push20(source8298), byte(SETCODEFROM), byte(STOP))
	for _, tt := range []struct {
		name   string
		source []byte // code of the source, nil for none
		warm   bool
		want   uint64
	}{
		{name: "invalid cold", want: 3100},
		{name: "invalid warm", warm: true, want: 200},
		{name: "unchanged cold", source: code, want: 3200},
		{name: "unchanged warm", source: code, warm: true, want: 300},
		{name: "updated cold", source: template8298, want: 12200},
		{name: "updated warm", source: template8298, warm: true, want: 9300},
	} {
		for _, delta := range []int64{-1, 0} {
			statedb := state8298(code)
			statedb.SetCode(source8298, tt.source, tracing.CodeChangeUnspecified)
			statedb.Finalise(params.Rules{IsEIP158: true})
			if tt.warm {
				statedb.AddAddressToAccessList(source8298)
			}
			limit := uint64(int64(3+tt.want) + delta) // PUSH20 + SETCODEFROM
			_, res, err := newEVM8298(statedb, true).Call(common.Address{}, self8298, nil, NewGasBudget(limit, 0), new(uint256.Int))
			switch {
			case delta < 0 && !errors.Is(err, ErrOutOfGas):
				t.Errorf("%s: one gas short: err %v, want out of gas", tt.name, err)
			case delta == 0 && (err != nil || res.UsedExecutionGas != limit):
				t.Errorf("%s: used %d (err %v), want %d", tt.name, res.UsedExecutionGas, err, limit)
			}
		}
	}
}

func TestSetCodeFromHalts(t *testing.T) {
	code := setCodeFromRet(source8298)

	// Static context.
	statedb := state8298(code)
	if _, _, err := newEVM8298(statedb, true).StaticCall(common.Address{}, self8298, nil, NewGasBudget(100_000, 0)); err == nil {
		t.Fatal("SETCODEFROM succeeded in a static context")
	}
	// System contract as the current account.
	statedb = state8298(code)
	statedb.SetCode(params.HistoryStorageAddress, code, tracing.CodeChangeUnspecified)
	if _, _, err := newEVM8298(statedb, true).Call(common.Address{}, params.HistoryStorageAddress, nil, NewGasBudget(100_000, 0), new(uint256.Int)); err == nil {
		t.Fatal("SETCODEFROM succeeded in a system contract")
	}
	if statedb.GetCodeHash(params.HistoryStorageAddress) != crypto.Keccak256Hash(code) {
		t.Fatal("system contract code changed")
	}
	// Code of a system contract running in another account's context is fine.
	statedb = state8298(delegateCallCode(params.HistoryStorageAddress, nil))
	statedb.SetCode(params.HistoryStorageAddress, code, tracing.CodeChangeUnspecified)
	if _, _, err := newEVM8298(statedb, true).Call(common.Address{}, self8298, nil, NewGasBudget(100_000, 0), new(uint256.Int)); err != nil {
		t.Fatalf("delegated system contract code failed: %v", err)
	}
	if statedb.GetCodeHash(self8298) != crypto.Keccak256Hash(template8298) {
		t.Fatal("delegating account did not adopt the code")
	}
}

func TestSetCodeFromValidity(t *testing.T) {
	target := common.BytesToAddress([]byte("target"))
	for _, tt := range []struct {
		name    string
		code    []byte // code of the source
		nonce   uint64 // keeps a codeless source in the state
		created bool   // source created in the current transaction
		valid   bool
	}{
		{name: "template", code: template8298, valid: true},
		{name: "nonexistent"},
		{name: "no code", nonce: 1},
		{name: "0xef prefix", code: []byte{0xef, 0x00}},
		{name: "delegated", code: types.AddressToDelegation(target)},
		{name: "created in tx", code: template8298, created: true},
	} {
		statedb := state8298(setCodeFromRet(source8298))
		statedb.SetCode(source8298, tt.code, tracing.CodeChangeUnspecified)
		statedb.SetNonce(source8298, tt.nonce, tracing.NonceChangeUnspecified)
		statedb.Finalise(params.Rules{IsEIP158: true})
		if tt.created {
			statedb.CreateContract(source8298)
		}
		ret, _, err := newEVM8298(statedb, true).Call(common.Address{}, self8298, nil, NewGasBudget(100_000, 0), new(uint256.Int))
		if err != nil {
			t.Fatalf("%s: %v", tt.name, err)
		}
		valid := new(uint256.Int).SetBytes(ret).Eq(uint256.NewInt(1))
		adopted := statedb.GetCodeHash(self8298) == crypto.Keccak256Hash(tt.code)
		if valid != tt.valid || adopted != tt.valid {
			t.Errorf("%s: result %x, adopted %v, want valid %v", tt.name, ret, adopted, tt.valid)
		}
		if statedb.AddressInAccessList(target) {
			t.Errorf("%s: delegation target accessed", tt.name)
		}
	}
}

// TestSetCodeFromFrame checks that the running frame keeps its code while the
// account and later reads see the adopted code, and that high bits of the
// operand are ignored.
func TestSetCodeFromFrame(t *testing.T) {
	operand := append([]byte{byte(PUSH32)}, bytes.Repeat([]byte{0xff}, 12)...)
	operand = append(operand, source8298.Bytes()...)
	code := append(operand, byte(SETCODEFROM), byte(POP),
		byte(CODESIZE), byte(PUSH0), byte(MSTORE),
		byte(ADDRESS), byte(EXTCODESIZE), byte(PUSH1), 32, byte(MSTORE),
		byte(PUSH1), 64, byte(PUSH0), byte(RETURN))
	statedb := state8298(code)
	ret, _, err := newEVM8298(statedb, true).Call(common.Address{}, self8298, nil, NewGasBudget(100_000, 0), new(uint256.Int))
	if err != nil {
		t.Fatal(err)
	}
	if got := new(uint256.Int).SetBytes(ret[:32]).Uint64(); got != uint64(len(code)) {
		t.Errorf("CODESIZE %d, want the running code size %d", got, len(code))
	}
	if got := new(uint256.Int).SetBytes(ret[32:]).Uint64(); got != uint64(len(template8298)) {
		t.Errorf("EXTCODESIZE of self %d, want the adopted size %d", got, len(template8298))
	}
}

// TestSetCodeFromInitcode checks that a creation adopting code skips the
// deposit, reports the adopted code and leaves the return buffer empty.
func TestSetCodeFromInitcode(t *testing.T) {
	statedb := state8298(nil)
	ret, addr, _, err := newEVM8298(statedb, true).Create(self8298, initcode8298, NewGasBudget(1_000_000, 1_000_000), new(uint256.Int))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(ret, template8298) || !bytes.Equal(statedb.GetCode(addr), template8298) {
		t.Fatalf("created code %x (output %x), want %x", statedb.GetCode(addr), ret, template8298)
	}
	// A CREATE that adopts leaves RETURNDATASIZE at zero.
	creator := deployCode(initcode8298, false, 0)
	creator = append(creator[:len(creator)-1], // drop the final STOP
		byte(RETURNDATASIZE), byte(PUSH0), byte(MSTORE), byte(PUSH1), 32, byte(PUSH0), byte(RETURN))
	statedb = state8298(creator)
	ret, _, err = newEVM8298(statedb, true).Call(common.Address{}, self8298, nil, NewGasBudget(1_000_000, 1_000_000), new(uint256.Int))
	if err != nil || len(ret) != 32 || new(uint256.Int).SetBytes(ret).Sign() != 0 {
		t.Fatalf("RETURNDATASIZE after CREATE: %x (err %v), want 0", ret, err)
	}
	if code := statedb.GetCode(crypto.CreateAddress(self8298, 0)); !bytes.Equal(code, template8298) {
		t.Fatalf("CREATE deployed %x, want %x", code, template8298)
	}
	// A failed adoption leaves the normal deposit: the returned code is installed.
	missing := common.BytesToAddress([]byte("missing"))
	deposit := append(push20(missing), byte(SETCODEFROM), byte(POP),
		byte(PUSH1), 0x60, byte(PUSH0), byte(MSTORE8), byte(PUSH1), 1, byte(PUSH0), byte(RETURN))
	statedb = state8298(nil)
	if _, addr, _, err := newEVM8298(statedb, true).Create(self8298, deposit, NewGasBudget(1_000_000, 1_000_000), new(uint256.Int)); err != nil || !bytes.Equal(statedb.GetCode(addr), []byte{0x60}) {
		t.Fatalf("deposit after a failed adoption: code %x, err %v", statedb.GetCode(addr), err)
	}
}

// TestSetCodeFromActivation checks that SETCODEFROM, with its deposit guard, is
// undefined before Bogota unless EIP-8298 is enabled explicitly.
func TestSetCodeFromActivation(t *testing.T) {
	for _, tt := range []struct {
		name   string
		bogota bool
		extra  []int
		ok     bool
	}{
		{"amsterdam", false, nil, false},
		{"amsterdam+8298", false, []int{8298}, true},
		{"bogota", true, nil, true},
	} {
		statedb := state8298(nil)
		_, addr, _, err := newEVM8298(statedb, tt.bogota, tt.extra...).Create(self8298, initcode8298, NewGasBudget(1_000_000, 1_000_000), new(uint256.Int))
		if adopted := err == nil && bytes.Equal(statedb.GetCode(addr), template8298); adopted != tt.ok {
			t.Errorf("%s: code %x, err %v, want adoption %v", tt.name, statedb.GetCode(addr), err, tt.ok)
		}
	}
}
