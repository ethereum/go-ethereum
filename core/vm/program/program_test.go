// Copyright 2024 The go-ethereum Authors
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

package program

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
)

func TestPush(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected string
	}{
		// native ints
		{0, "6000"},
		{0xfff, "610fff"},
		{nil, "6000"},
		{uint8(1), "6001"},
		{uint16(1), "6001"},
		{uint32(1), "6001"},
		{uint64(1), "6001"},
		// bigints
		{big.NewInt(0), "6000"},
		{big.NewInt(1), "6001"},
		{big.NewInt(0xfff), "610fff"},
		// uint256
		{uint256.NewInt(1), "6001"},
		{uint256.Int{1, 0, 0, 0}, "6001"},
		// Addresses
		{common.HexToAddress("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"), "73deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"},
		{&common.Address{}, "6000"},
	}
	for i, tc := range tests {
		have := New().Push(tc.input).Hex()
		if have != tc.expected {
			t.Errorf("test %d: got %v expected %v", i, have, tc.expected)
		}
	}
}

func TestCall(t *testing.T) {
	{ // Nil gas
		have := New().Call(nil, common.HexToAddress("0x1337"), big.NewInt(1), 1, 2, 3, 4).Hex()
		want := "600460036002600160016113375af1"
		if have != want {
			t.Errorf("have %v want %v", have, want)
		}
	}
	{ // Non nil gas
		have := New().Call(uint256.NewInt(0xffff), common.HexToAddress("0x1337"), big.NewInt(1), 1, 2, 3, 4).Hex()
		want := "6004600360026001600161133761fffff1"
		if have != want {
			t.Errorf("have %v want %v", have, want)
		}
	}
}

func TestMstore(t *testing.T) {
	{
		have := New().Mstore(common.FromHex("0xaabb"), 0).Hex()
		want := "60aa60005360bb600153"
		if have != want {
			t.Errorf("have %v want %v", have, want)
		}
	}
	{ // store at offset
		have := New().Mstore(common.FromHex("0xaabb"), 3).Hex()
		want := "60aa60035360bb600453"
		if have != want {
			t.Errorf("have %v want %v", have, want)
		}
	}
	{ // 34 bytes
		data := common.FromHex("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF" +
			"FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF" +
			"FFFF")

		have := New().Mstore(data, 0).Hex()
		want := "7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff60005260ff60205360ff602153"
		if have != want {
			t.Errorf("have %v want %v", have, want)
		}
	}
}

func TestMemToStorage(t *testing.T) {
	have := New().MemToStorage(0, 33, 1).Hex()
	want := "600051600155602051600255"
	if have != want {
		t.Errorf("have %v want %v", have, want)
	}
}

func TestSstore(t *testing.T) {
	have := New().Sstore(0x1337, []byte("1234")).Hex()
	want := "633132333461133755"
	if have != want {
		t.Errorf("have %v want %v", have, want)
	}
}

func TestReturnData(t *testing.T) {
	{
		have := New().ReturnData([]byte{0xFF}).Hex()
		want := "60ff60005360016000f3"
		if have != want {
			t.Errorf("have %v want %v", have, want)
		}
	}
	{
		// 32 bytes
		data := common.FromHex("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF")
		have := New().ReturnData(data).Hex()
		want := "7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff60005260206000f3"
		if have != want {
			t.Errorf("have %v want %v", have, want)
		}
	}
	{ // ReturnViaCodeCopy
		data := common.FromHex("0x6001")
		have := New().Append([]byte{0x5b, 0x5b, 0x5b}).ReturnViaCodeCopy(data).Hex()
		want := "5b5b5b600261001060003960026000f36001"
		if have != want {
			t.Errorf("have %v want %v", have, want)
		}
	}
	{ // ReturnViaCodeCopy larger code
		data := common.FromHex("7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff60005260206000f3")
		have := New().Append([]byte{0x5b, 0x5b, 0x5b}).ReturnViaCodeCopy(data).Hex()
		want := "5b5b5b602961001060003960296000f37fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff60005260206000f3"
		if have != want {
			t.Errorf("have %v want %v", have, want)
		}
	}
}

func TestCreateAndCall(t *testing.T) {
	// A constructor that stores a slot
	ctor := New().Sstore(0, big.NewInt(5))

	// A runtime bytecode which reads the slot and returns
	deployed := New()
	deployed.Push(0).Op(vm.SLOAD) // [value] in stack
	deployed.Push(0)              // [value, 0]
	deployed.Op(vm.MSTORE)
	deployed.Return(0, 32)

	// Pack them
	ctor.ReturnData(deployed.Bytes())
	// Verify constructor + runtime code
	{
		want := "6005600055606060005360006001536054600253606060035360006004536052600553606060065360206007536060600853600060095360f3600a53600b6000f3"
		if got := ctor.Hex(); got != want {
			t.Fatalf("1: got %v expected %v", got, want)
		}
	}
}

func TestCreate2Call(t *testing.T) {
	// Some runtime code
	runtime := New().Op(vm.ADDRESS, vm.SELFDESTRUCT).Bytes()
	want := common.FromHex("0x30ff")
	if !bytes.Equal(want, runtime) {
		t.Fatalf("runtime code error\nwant: %x\nhave: %x\n", want, runtime)
	}
	// A constructor returning the runtime code
	initcode := New().ReturnData(runtime).Bytes()
	want = common.FromHex("603060005360ff60015360026000f3")
	if !bytes.Equal(want, initcode) {
		t.Fatalf("initcode error\nwant: %x\nhave: %x\n", want, initcode)
	}
	// A factory invoking the constructor
	outer := New().Create2ThenCall(initcode, nil).Bytes()
	want = common.FromHex("60606000536030600153606060025360006003536053600453606060055360ff6006536060600753600160085360536009536060600a536002600b536060600c536000600d5360f3600e536000600f60006000f560006000600060006000855af15050")
	if !bytes.Equal(want, outer) {
		t.Fatalf("factory error\nwant: %x\nhave: %x\n", want, outer)
	}
}

func TestGenerator(t *testing.T) {
	for i, tc := range []struct {
		want   []byte
		haveFn func() []byte
	}{
		{ // CREATE
			want: []byte{
				// Store initcode in memory at 0x00 (5 bytes left-padded to 32 bytes)
				byte(vm.PUSH5),
				// Init code: PUSH1 0, PUSH1 0, RETURN (3 steps)
				byte(vm.PUSH1), 0, byte(vm.PUSH1), 0, byte(vm.RETURN),
				byte(vm.PUSH1), 0,
				byte(vm.MSTORE),
				// length, offset, value
				byte(vm.PUSH1), 5, byte(vm.PUSH1), 27, byte(vm.PUSH1), 0,
				byte(vm.CREATE),
				byte(vm.POP),
			},
			haveFn: func() []byte {
				initcode := New().Return(0, 0).Bytes()
				return New().MstoreSmall(initcode, 0).
					Push(len(initcode)).      // length
					Push(32 - len(initcode)). // offset
					Push(0).                  // value
					Op(vm.CREATE).
					Op(vm.POP).Bytes()
			},
		},
		{ // CREATE2
			want: []byte{
				// Store initcode in memory at 0x00 (5 bytes left-padded to 32 bytes)
				byte(vm.PUSH5),
				// Init code: PUSH1 0, PUSH1 0, RETURN (3 steps)
				byte(vm.PUSH1), 0, byte(vm.PUSH1), 0, byte(vm.RETURN),
				byte(vm.PUSH1), 0,
				byte(vm.MSTORE),
				// salt, length, offset, value
				byte(vm.PUSH1), 1, byte(vm.PUSH1), 5, byte(vm.PUSH1), 27, byte(vm.PUSH1), 0,
				byte(vm.CREATE2),
				byte(vm.POP),
			},
			haveFn: func() []byte {
				initcode := New().Return(0, 0).Bytes()
				return New().MstoreSmall(initcode, 0).
					Push(1).                  // salt
					Push(len(initcode)).      // length
					Push(32 - len(initcode)). // offset
					Push(0).                  // value
					Op(vm.CREATE2).
					Op(vm.POP).Bytes()
			},
		},
		{ // CALL
			want: []byte{
				// outsize, outoffset, insize, inoffset
				byte(vm.PUSH1), 0, byte(vm.DUP1), byte(vm.DUP1), byte(vm.DUP1),
				byte(vm.DUP1),        // value
				byte(vm.PUSH1), 0xbb, //address
				byte(vm.GAS), // gas
				byte(vm.CALL),
				byte(vm.POP),
			},
			haveFn: func() []byte {
				return New().Call(nil, 0xbb, 0, 0, 0, 0, 0).Op(vm.POP).Bytes()
			},
		},
		{ // CALLCODE
			want: []byte{
				// outsize, outoffset, insize, inoffset
				byte(vm.PUSH1), 0, byte(vm.DUP1), byte(vm.DUP1), byte(vm.DUP1),
				byte(vm.PUSH1), 0, // value
				byte(vm.PUSH1), 0xcc, //address
				byte(vm.GAS), // gas
				byte(vm.CALLCODE),
				byte(vm.POP),
			},
			haveFn: func() []byte {
				return New().CallCode(nil, 0xcc, 0, 0, 0, 0, 0).Op(vm.POP).Bytes()
			},
		},
		{ // STATICCALL
			want: []byte{
				// outsize, outoffset, insize, inoffset
				byte(vm.PUSH1), 0, byte(vm.DUP1), byte(vm.DUP1), byte(vm.DUP1),
				byte(vm.PUSH1), 0xdd, //address
				byte(vm.GAS), // gas
				byte(vm.STATICCALL),
				byte(vm.POP),
			},
			haveFn: func() []byte {
				return New().StaticCall(nil, 0xdd, 0, 0, 0, 0).Op(vm.POP).Bytes()
			},
		},
		{ // DELEGATECALL
			want: []byte{
				// outsize, outoffset, insize, inoffset
				byte(vm.PUSH1), 0, byte(vm.DUP1), byte(vm.DUP1), byte(vm.DUP1),
				byte(vm.PUSH1), 0xee, //address
				byte(vm.GAS), // gas
				byte(vm.DELEGATECALL),
				byte(vm.POP),
			},
			haveFn: func() []byte {
				return New().DelegateCall(nil, 0xee, 0, 0, 0, 0).Op(vm.POP).Bytes()
			},
		},
	} {
		if have := tc.haveFn(); !bytes.Equal(have, tc.want) {
			t.Fatalf("test %d error\nhave: %x\nwant: %x\n", i, have, tc.want)
		}
	}
}
