// Copyright 2015 The go-ethereum Authors
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

import "testing"

type matchTest struct {
	input   []OpCode
	match   []OpCode
	matches int
}

func TestMatchFn(t *testing.T) {
	tests := []matchTest{
		matchTest{
			[]OpCode{PUSH1, PUSH1, MSTORE, JUMP},
			[]OpCode{PUSH1, MSTORE},
			1,
		},
		matchTest{
			[]OpCode{PUSH1, PUSH1, MSTORE, JUMP},
			[]OpCode{PUSH1, MSTORE, PUSH1},
			0,
		},
		matchTest{
			[]OpCode{},
			[]OpCode{PUSH1},
			0,
		},
	}

	for i, test := range tests {
		var matchCount int
		MatchFn(test.input, test.match, func(i int) bool {
			matchCount++
			return true
		})
		if matchCount != test.matches {
			t.Errorf("match count failed on test[%d]: expected %d matches, got %d", i, test.matches, matchCount)
		}
	}
}

type parseTest struct {
	base   OpCode
	size   int
	output OpCode
}

func TestParser(t *testing.T) {
	tests := []parseTest{
		parseTest{PUSH1, 32, PUSH},
		parseTest{DUP1, 16, DUP},
		parseTest{SWAP1, 16, SWAP},
		parseTest{MSTORE, 1, MSTORE},
	}

	for _, test := range tests {
		for i := 0; i < test.size; i++ {
			code := append([]byte{byte(byte(test.base) + byte(i))}, make([]byte, i+1)...)
			output := Parse(code)
			if len(output) == 0 {
				t.Fatal("empty output")
			}
			if output[0] != test.output {
				t.Error("%v failed: expected %v but got %v", test.base+OpCode(i), output[0])
			}
		}
	}
}
