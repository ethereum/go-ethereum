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

package vm

// immediate denotes how many immediate bytes an operation uses. This information
// is not required during runtime, only during EOF-validation, so is not
// places into the op-struct in the instruction table.
// Note: the immediates is fork-agnostic, and assumes that validity of opcodes at
// the given time is performed elsewhere.
var immediates [256]uint8

// terminals denotes whether instructions can be the final opcode in a code section.
// Note: the terminals is fork-agnostic, and assumes that validity of opcodes at
// the given time is performed elsewhere.
var terminals [256]bool

func init() {
	// The legacy pushes
	for i := uint8(1); i < 33; i++ {
		immediates[int(PUSH0)+int(i)] = i
	}
	// And new eof opcodes.
	immediates[DATALOADN] = 2
	immediates[RJUMP] = 2
	immediates[RJUMPI] = 2
	immediates[RJUMPV] = 3
	immediates[CALLF] = 2
	immediates[JUMPF] = 2
	immediates[DUPN] = 1
	immediates[SWAPN] = 1
	immediates[EXCHANGE] = 1
	immediates[EOFCREATE] = 1
	immediates[RETURNCONTRACT] = 1

	// Define the terminals.
	terminals[STOP] = true
	terminals[RETF] = true
	terminals[JUMPF] = true
	terminals[RETURNCONTRACT] = true
	terminals[RETURN] = true
	terminals[REVERT] = true
	terminals[INVALID] = true
}

// Immediates returns the number bytes of immediates (argument not from
// stack but from code) a given opcode has.
// OBS:
//   - This function assumes EOF instruction-set. It cannot be upon in
//     a. pre-EOF code
//     b. post-EOF but legacy code
//   - RJUMPV is unique as it has a variable sized operand. The total size is
//     determined by the count byte which immediately follows RJUMPV. This method
//     will return '3' for RJUMPV, which is the minimum.
func Immediates(op OpCode) int {
	return int(immediates[op])
}
