// Copyright 2016 The go-ethereum Authors
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

package asm

import (
	"testing"

	"encoding/hex"
	"github.com/ethereum/go-ethereum/core/vm"
)

// Tests disassembling the instructions for valid evm code
func TestForEachDisassembledInstructionValid(t *testing.T) {
	cnt := 0
	script, _ := hex.DecodeString("61000000")
	err := ForEachDisassembledInstruction(script, func(pc uint64, op vm.OpCode, args []byte) {
		cnt++
	})
	if err != nil {
		t.Errorf("Expected 2, but encountered error %v instead.", err)
	}
	if cnt != 2 {
		t.Errorf("Expected 2, but got %v instead.", cnt)
	}
}

// Tests disassembling the instructions for invalid evm code
func TestForEachDisassembledInstructionInvalid(t *testing.T) {
	cnt := 0
	script, _ := hex.DecodeString("6100")
	err := ForEachDisassembledInstruction(script, func(pc uint64, op vm.OpCode, args []byte) {
		cnt++
	})
	if err == nil {
		t.Errorf("Expected an error, but got %v instead.", cnt)
	}
}
