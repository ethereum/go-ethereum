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

func validateCode(code []byte, section int, metadata []*FunctionMetadata, jt *JumpTable) error {
	var (
		i        = 0
		count    = 0
		op       OpCode
		analysis *bitvec
	)
	for i < len(code) {
		count += 1
		op = OpCode(code[i])
		if jt[op].undefined {
			return fmt.Errorf("use of undefined opcode %s", op)
		}
		switch {
		case op >= PUSH1 && op <= PUSH32:
			// Verify that push data is not truncated.
			size := int(op - PUSH0)
			if i+size >= len(code) {
				return fmt.Errorf("truncated operand")
			}
			i += size
		case op == RJUMP || op == RJUMPI:
			if analysis == nil {
				tmp := eofCodeBitmap(code)
				analysis = &tmp
			}
			// Verify that the relative jump offset points to a
			// destination in-bounds.
			if err := checkDest(code[i+1:], *analysis, i+3, len(code)); err != nil {
				return err
			}
			i += 2
		case op == RJUMPV:
			if analysis == nil {
				tmp := eofCodeBitmap(code)
				analysis = &tmp
			}
			// Verify each branch in the jump table points to a
			// destination in-bounds.
			if i+1 >= len(code) {
				return fmt.Errorf("truncated jump table operand")
			}
			count := int(code[i+1])
			if count == 0 {
				return fmt.Errorf("rjumpv branch count must not be 0")
			}
			for j := 0; j < count; j++ {
				if err := checkDest(code[i+2+j*2:], *analysis, i+2*count+2, len(code)); err != nil {
					return err
				}
			}
			i += 1 + 2*count
		case op == CALLF || op == JUMPF:
			if i+2 >= len(code) {
				return fmt.Errorf("truncated operand")
			}
			arg := parseUint16(code[i+1:])
			if arg >= len(metadata) {
				return fmt.Errorf("code section out-of-bounds (want: %d, have: %d)", arg, len(metadata))
			}
			if op == JUMPF {
				if metadata[section].Output < metadata[arg].Output {
					return fmt.Errorf("jumpf to section with more outputs")
				}
			}
			i += 2
		}
		i += 1
	}
	if !jt[op].terminal {
		return fmt.Errorf("code section ends with non-terminal instruction")
	}
	if max, paths, err := validateControlFlow(code, section, metadata, jt); err != nil {
		return err
	} else if paths != count {
		return fmt.Errorf("unreachable code")
	} else if max != int(metadata[section].MaxStackHeight) {
		return fmt.Errorf("computed max stack height for code section %d does not match expect (want: %d, got: %d)", section, metadata[section].MaxStackHeight, max)
	}
	return nil
}

func checkDest(code []byte, analysis bitvec, idx, length int) error {
	if len(code) < 2 {
		return fmt.Errorf("truncated operand")
	}
	offset := parseInt16(code)
	dest := idx + int(offset)
	if dest < 0 || dest >= length {
		return fmt.Errorf("relative offset out-of-bounds: %d", dest)
	}
	if !analysis.codeSegment(uint64(dest)) {
		return fmt.Errorf("relative offset into immediate operand: %d", dest)
	}
	return nil
}

func validateControlFlow(code []byte, section int, metadata []*FunctionMetadata, jt *JumpTable) (int, int, error) {
	type item struct {
		pos    int
		height int
	}
	var (
		heights        = make(map[int]int)
		worklist       = []item{{0, int(metadata[section].Input)}}
		maxStackHeight = 0
	)
	for 0 < len(worklist) {
		var (
			idx    = len(worklist) - 1
			pos    = worklist[idx].pos
			height = worklist[idx].height
		)
		worklist = worklist[:idx]
		for pos < len(code) {
			op := OpCode(code[pos])

			// Check if pos has already be visited; if so, the stack heights should be the same.
			if exp, ok := heights[pos]; ok {
				if height != exp {
					return 0, 0, fmt.Errorf("stack height mismatch for different paths")
				}
				// Already visited this path and stack height
				// matches.
				break
			}
			heights[pos] = height

			switch {
			case op == CALLF:
				arg := parseUint16(code[pos+1:])
				if metadata[arg].Input < uint8(height) {
					return 0, 0, fmt.Errorf("stack underflow")
				}
				if int(metadata[arg].Output)+height > int(params.StackLimit) {
					return 0, 0, fmt.Errorf("stack overflow")
				}
			case op == RJUMP:
				arg := parseUint16(code[pos+1:])
				pos += 3 + arg
			case op == RJUMPI:
				arg := parseUint16(code[pos+1:])
				worklist = append(worklist, item{pos: pos + 3 + arg, height: height})
				pos += 3
			case op == RJUMPV:
				count := int(code[pos+1])
				for i := 0; i < count; i++ {
					arg := parseUint16(code[pos+2+2*i:])
					worklist = append(worklist, item{pos: pos + arg, height: height})
				}
				pos += 2 + 2*count
			default:
				if jt[op].minStack > height {
					return 0, 0, fmt.Errorf("stack underflow")
				}
				if jt[op].maxStack < height {
					return 0, 0, fmt.Errorf("stack overflow")
				}
				height += int(params.StackLimit) - jt[op].maxStack
				if op >= PUSH1 && op <= PUSH32 {
					pos += 1 + int(op-PUSH0)
				} else {
					// No immediate.
					pos += 1
				}
			}
			maxStackHeight = max(maxStackHeight, height)
		}
	}
	return maxStackHeight, len(heights), nil
}

func max(a, b int) int {
	if a < b {
		return b
	}
	return a
}
