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

package asm

import (
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
)

// Tests disassembling instructions
func TestInstructionIterator(t *testing.T) {
	for i, tc := range []struct {
		code       string
		legacyWant string
		eofWant    string
	}{
		{"", "", ""}, // empty
		{"6100", `err: incomplete instruction at 0`, `err: incomplete instruction at 0`},
		{"61000000", `
00000: PUSH2 0x0000
00003: STOP`, `
00000: PUSH2 0x0000
00003: STOP`},
		{"5F00", `
00000: PUSH0
00001: STOP`, `
00000: PUSH0
00001: STOP`},
		{"d1aabb00", `00000: DATALOADN
00001: opcode 0xaa not defined
00002: opcode 0xbb not defined
00003: STOP`, `
00000: DATALOADN 0xaabb
00003: STOP`}, // DATALOADN(aabb),STOP
		{"d1aa", `
00000: DATALOADN
00001: opcode 0xaa not defined`, "err: incomplete instruction at 0\n"}, // DATALOADN(aa) invalid
		{"e20211223344556600", `
00000: RJUMPV
00001: MUL
00002: GT
00003: opcode 0x22 not defined
00004: CALLER
00005: DIFFICULTY
00006: SSTORE
err: incomplete instruction at 7`, `
00000: RJUMPV 0x02112233445566
00008: STOP`}, // RJUMPV( 6 bytes), STOP

	} {
		var (
			code, _ = hex.DecodeString(tc.code)
			legacy  = strings.TrimSpace(disassembly(NewInstructionIterator(code)))
			eof     = strings.TrimSpace(disassembly(NewEOFInstructionIterator(code)))
		)
		if want := strings.TrimSpace(tc.legacyWant); legacy != want {
			t.Errorf("test %d: wrong (legacy) output. have:\n%q\nwant:\n%q\n", i, legacy, want)
		}
		if want := strings.TrimSpace(tc.eofWant); eof != want {
			t.Errorf("test %d: wrong (eof) output. have:\n%q\nwant:\n%q\n", i, eof, want)
		}
	}
}

func disassembly(it *instructionIterator) string {
	var out = new(strings.Builder)
	for it.Next() {
		if it.Arg() != nil && 0 < len(it.Arg()) {
			fmt.Fprintf(out, "%05x: %v %#x\n", it.PC(), it.Op(), it.Arg())
		} else {
			fmt.Fprintf(out, "%05x: %v\n", it.PC(), it.Op())
		}
	}
	if err := it.Error(); err != nil {
		fmt.Fprintf(out, "err: %v\n", err)
	}
	return out.String()
}
