// Copyright 2024 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the goevmlab library. If not, see <http://www.gnu.org/licenses/>.

package program

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	"bytes"
	"testing"
)

func TestCreate2Call(t *testing.T) {
	// Some runtime code
	runtime := New().Ops(vm.ADDRESS, vm.SELFDESTRUCT).Bytecode()
	want := common.FromHex("0x30ff")
	if !bytes.Equal(want, runtime) {
		t.Fatalf("runtime code error\nwant: %x\nhave: %x\n", want, runtime)
	}
	// A constructor returning the runtime code
	initcode := New().ReturnData(runtime).Bytecode()
	want = common.FromHex("603060005360ff60015360026000f3")
	if !bytes.Equal(want, initcode) {
		t.Fatalf("initcode error\nwant: %x\nhave: %x\n", want, initcode)
	}
	// A factory invoking the constructor
	outer := New().Create2AndCall(initcode, nil).Bytecode()
	want = common.FromHex("60606000536030600153606060025360006003536053600453606060055360ff6006536060600753600160085360536009536060600a536002600b536060600c536000600d5360f3600e536000600f60006000f560006000600060006000855af15050")
	if !bytes.Equal(want, outer) {
		t.Fatalf("factory error\nwant: %x\nhave: %x\n", want, outer)
	}
}
