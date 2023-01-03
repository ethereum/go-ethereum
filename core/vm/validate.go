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

	"github.com/ethereum/go-ethereum/params"
)

// validateCode validates the code parameter against the EOF v1 validity requirements.
func validateCode(code []byte, section int, metadata []*FunctionMetadata, jt *JumpTable) error {
	var (
		i = 0
		// Tracks the number of actual instructions in the code (e.g.
		// non-immediate values). This is used at the end to determine
		// if each instruction is reachable.
		count    = 0
		op       OpCode
		analysis bitvec
	)
	// This loop visits every single instruction and verifies:
	// * if the instruction is valid for the given jump table.
	// * if the instruction has an immediate value, it is not truncated.
	// * if performing a relative jump, all jump destinations are valid.
	// * if changing code sections, the new code section index is valid and
	//   will not cause a stack overflow.
	for i < len(code) {
		count++
		op = OpCode(code[i])
		if jt[op].undefined {
			return fmt.Errorf("use of undefined opcode %s", op)
		}
		switch {
		case op >= PUSH1 && op <= PUSH32:
			size := int(op - PUSH0)
			if len(code) <= i+size {
				return fmt.Errorf("truncated operand")
			}
			i += size
		case op == RJUMP || op == RJUMPI:
			if len(code) <= i+2 {
				return fmt.Errorf("truncated rjump* operand")
			}
			if err := checkDest(code, &analysis, i+1, i+3, len(code)); err != nil {
				return err
			}
			i += 2
		case op == RJUMPV:
			if len(code) <= i+1 {
				return fmt.Errorf("truncated jump table operand")
			}
			count := int(code[i+1])
			if count == 0 {
				return fmt.Errorf("rjumpv branch count must not be 0")
			}
			if len(code) <= i+count {
				return fmt.Errorf("truncated jump table operand")
			}
			for j := 0; j < count; j++ {
				if err := checkDest(code, &analysis, i+2+j*2, i+2*count+2, len(code)); err != nil {
					return err
				}
			}
			i += 1 + 2*count
		case op == CALLF:
			if i+2 >= len(code) {
				return fmt.Errorf("truncated operand")
			}
			arg, _ := parseUint16(code[i+1:])
			if arg >= len(metadata) {
				return fmt.Errorf("code section out-of-bounds (want: %d, have: %d)", arg, len(metadata))
			}
			i += 2
		}
		i += 1
	}
	// Code sections may not "fall through" and require proper termination.
	// Therefore, the last instruction must be considered terminal.
	if !jt[op].terminal {
		return fmt.Errorf("code section ends with non-terminal instruction")
	}
	if paths, err := validateControlFlow(code, section, metadata, jt); err != nil {
		return err
	} else if paths != count {
		return fmt.Errorf("unreachable code")
	}
	return nil
}

// checkDest parses a relative offset at code[0:2] and checks if it is a valid jump destination.
func checkDest(code []byte, analysis *bitvec, imm, from, length int) error {
	if len(code) < imm+2 {
		return fmt.Errorf("truncated operand")
	}
	if analysis != nil && *analysis == nil {
		*analysis = eofCodeBitmap(code)
	}
	dest := from + parseInt16(code[imm:])
	if dest < 0 || dest >= length {
		return fmt.Errorf("relative offset out-of-bounds: %d", dest)
	}
	if !analysis.codeSegment(uint64(dest)) {
		return fmt.Errorf("relative offset into immediate operand: %d", dest)
	}
	return nil
}

// validateControlFlow iterates through all possible branches the provided code
// value and determines if it is valid per EOF v1.
func validateControlFlow(code []byte, section int, metadata []*FunctionMetadata, jt *JumpTable) (int, error) {
	type item struct {
		pos    int
		height int
	}
	var (
		heights        = make(map[int]int)
		worklist       = []item{{0, int(metadata[section].Input)}}
		maxStackHeight = int(metadata[section].Input)
	)
	for 0 < len(worklist) {
		var (
			idx    = len(worklist) - 1
			pos    = worklist[idx].pos
			height = worklist[idx].height
		)
		worklist = worklist[:idx]
	outer:
		for pos < len(code) {
			op := OpCode(code[pos])

			// Check if pos has already be visited; if so, the stack heights should be the same.
			if exp, ok := heights[pos]; ok {
				if height != exp {
					return 0, fmt.Errorf("stack height mismatch for different paths")
				}
				// Already visited this path and stack height
				// matches.
				break
			}
			heights[pos] = height

			// Validate height for current op and update as needed.
			if jt[op].minStack > height {
				return 0, fmt.Errorf("stack underflow")
			}
			if jt[op].maxStack < height {
				return 0, fmt.Errorf("stack overflow")
			}
			height += int(params.StackLimit) - jt[op].maxStack

			switch {
			case op == CALLF:
				arg, _ := parseUint16(code[pos+1:])
				if metadata[arg].Input > uint8(height) {
					return 0, fmt.Errorf("stack underflow")
				}
				if int(metadata[arg].Output)+height > int(params.StackLimit) {
					return 0, fmt.Errorf("stack overflow")
				}
				height -= int(metadata[arg].Input)
				height += int(metadata[arg].Output)
				pos += 3
			case op == RETF:
				if int(metadata[section].Output) != height {
					return 0, fmt.Errorf("wrong number of outputs (want: %d, got: %d)", metadata[section].Output, height)
				}
				break outer
			case op == RJUMP:
				arg := parseInt16(code[pos+1:])
				pos += 3 + arg
			case op == RJUMPI:
				arg := parseInt16(code[pos+1:])
				worklist = append(worklist, item{pos: pos + 3 + arg, height: height})
				pos += 3
			case op == RJUMPV:
				count := int(code[pos+1])
				for i := 0; i < count; i++ {
					arg := parseInt16(code[pos+2+2*i:])
					worklist = append(worklist, item{pos: pos + 2 + 2*count + arg, height: height})
				}
				pos += 2 + 2*count
			default:
				if op >= PUSH1 && op <= PUSH32 {
					pos += 1 + int(op-PUSH0)
				} else if jt[op].terminal {
					break outer
				} else {
					// Simple op, no operand.
					pos += 1
				}
			}
			maxStackHeight = max(maxStackHeight, height)
		}
	}
	if maxStackHeight != int(metadata[section].MaxStackHeight) {
		return 0, fmt.Errorf("computed max stack height for code section %d does not match expected (want: %d, got: %d)", section, metadata[section].MaxStackHeight, maxStackHeight)
	}
	return len(heights), nil
}
