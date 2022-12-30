// Copyright 2022 The go-ethereum Authors
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
	"fmt"
	"testing"
)

func TestValidateCode(t *testing.T) {
	for i, test := range []struct {
		code     []byte
		section  int
		metadata []*FunctionMetadata
		err      error
	}{
		{
			code: []byte{
				byte(CALLER),
				byte(POP),
				byte(STOP),
			},
			section:  0,
			metadata: []*FunctionMetadata{{Input: 0, Output: 0, MaxStackHeight: 1}},
		},
		{
			code: []byte{
				byte(CALLER),
				byte(POP),
			},
			section:  0,
			metadata: []*FunctionMetadata{{Input: 0, Output: 0, MaxStackHeight: 1}},
			err:      fmt.Errorf("code section ends with non-terminal instruction"),
		},
		{
			code: []byte{
				byte(RJUMP),
				byte(0x00),
				byte(0x01),
				byte(CALLER),
				byte(STOP),
			},
			section:  0,
			metadata: []*FunctionMetadata{{Input: 0, Output: 0, MaxStackHeight: 1}},
			err:      fmt.Errorf("unreachable code"),
		},
		{
			code: []byte{
				byte(PUSH1),
				byte(0x42),
				byte(ADD),
				byte(STOP),
			},
			section:  0,
			metadata: []*FunctionMetadata{{Input: 0, Output: 0, MaxStackHeight: 1}},
			err:      fmt.Errorf("stack underflow"),
		},
		{
			code: []byte{
				byte(PUSH1),
				byte(0x42),
				byte(POP),
				byte(STOP),
			},
			section:  0,
			metadata: []*FunctionMetadata{{Input: 0, Output: 0, MaxStackHeight: 2}},
			err:      fmt.Errorf("computed max stack height for code section 0 does not match expect (want: 2, got: 1)"),
		},
		{
			code: []byte{
				byte(PUSH0),
				byte(RJUMPI),
				byte(0x00),
				byte(0x01),
				byte(PUSH1),
				byte(0x42), // jumps to here
				byte(POP),
				byte(STOP),
			},
			section:  0,
			metadata: []*FunctionMetadata{{Input: 0, Output: 0, MaxStackHeight: 1}},
			err:      fmt.Errorf("relative offset into immediate operand: 5"),
		},
		{
			code: []byte{
				byte(PUSH0),
				byte(RJUMPV),
				byte(0x02),
				byte(0x00),
				byte(0x01),
				byte(0x00),
				byte(0x02),
				byte(PUSH1),
				byte(0x42), // jumps to here
				byte(POP),  // and here
				byte(STOP),
			},
			section:  0,
			metadata: []*FunctionMetadata{{Input: 0, Output: 0, MaxStackHeight: 1}},
			err:      fmt.Errorf("relative offset into immediate operand: 8"),
		},
		{
			code: []byte{
				byte(PUSH0),
				byte(RJUMPV),
				byte(0x00),
				byte(STOP),
			},
			section:  0,
			metadata: []*FunctionMetadata{{Input: 0, Output: 0, MaxStackHeight: 1}},
			err:      fmt.Errorf("rjumpv branch count must not be 0"),
		},
		{
			code: []byte{
				byte(JUMPF),
				byte(0x00),
				byte(0x01),
			},
			section:  0,
			metadata: []*FunctionMetadata{{Input: 0, Output: 0, MaxStackHeight: 1}, {Input: 0, Output: 1, MaxStackHeight: 1}},
			err:      fmt.Errorf("jumpf to section with more outputs"),
		},
		{
			code: []byte{
				byte(RJUMP), 0x00, 0x03,
				byte(JUMPDEST),
				byte(JUMPDEST),
				byte(RETURN),
				byte(PUSH1), 20,
				byte(PUSH1), 39,
				byte(PUSH1), 0x00,
				byte(CODECOPY),
				byte(PUSH1), 20,
				byte(PUSH1), 0x00,
				byte(RJUMP), 0xff, 0xef,
			},
			section:  0,
			metadata: []*FunctionMetadata{{Input: 0, Output: 0, MaxStackHeight: 3}},
		},
		{
			code: []byte{
				byte(PUSH1), 1,
				byte(RJUMPI), 0x00, 0x03,
				byte(JUMPDEST),
				byte(JUMPDEST),
				byte(STOP),
				byte(PUSH1), 20,
				byte(PUSH1), 39,
				byte(PUSH1), 0x00,
				byte(CODECOPY),
				byte(PUSH1), 20,
				byte(PUSH1), 0x00,
				byte(RETURN),
			},
			section:  0,
			metadata: []*FunctionMetadata{{Input: 0, Output: 0, MaxStackHeight: 3}},
		},
		{
			code: []byte{
				byte(PUSH1), 1,
				byte(RJUMPV), 0x02, 0x00, 0x03, 0xff, 0xf8,
				byte(JUMPDEST),
				byte(JUMPDEST),
				byte(STOP),
				byte(PUSH1), 20,
				byte(PUSH1), 39,
				byte(PUSH1), 0x00,
				byte(CODECOPY),
				byte(PUSH1), 20,
				byte(PUSH1), 0x00,
				byte(RETURN),
			},
			section:  0,
			metadata: []*FunctionMetadata{{Input: 0, Output: 0, MaxStackHeight: 3}},
		},
	} {
		err := validateCode(test.code, test.section, test.metadata, &shanghaiEOFInstructionSet)
		if (err != nil && test.err == nil) || (err == nil && test.err != nil) || (err != nil && test.err != nil && err.Error() != test.err.Error()) {
			t.Errorf("test %d: unexpected error (want: %v, got: %v)", i, test.err, err)
		}
	}

}
