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
	"errors"
	"io"

	"github.com/ethereum/go-ethereum/params"
)

var (
	ErrUndefinedInstruction   = errors.New("undefined instrustion")
	ErrTruncatedImmediate     = errors.New("truncated immediate")
	ErrInvalidSectionArgument = errors.New("invalid section argument")
	ErrInvalidJumpDest        = errors.New("invalid jump destination")
	ErrConflictingStack       = errors.New("conflicting stack height")
	ErrInvalidBranchCount     = errors.New("invalid number of branches in jump table")
	ErrInvalidOutputs         = errors.New("invalid number of outputs")
	ErrInvalidMaxStackHeight  = errors.New("invalid max stack height")
	ErrInvalidCodeTermination = errors.New("invalid code termination")
	ErrUnreachableCode        = errors.New("unreachable code")
)

// validateCode validates the code parameter against the EOF v1 validity requirements.
func validateCode(code []byte, section int, metadata []*FunctionMetadata, jt *JumpTable) error {
	var (
		i = 0
		e = NewParseError
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
			return e(ErrUndefinedInstruction, i, "opcode=%s", op)
		}
		switch {
		case op >= PUSH1 && op <= PUSH32:
			size := int(op - PUSH0)
			if len(code) <= i+size {
				return e(ErrTruncatedImmediate, i, "op=%s", op)
			}
			i += size
		case op == RJUMP || op == RJUMPI:
			if len(code) <= i+2 {
				return e(ErrTruncatedImmediate, i, "op=%s", op)
			}
			if err := checkDest(code, &analysis, i+1, i+3, len(code)); err != nil {
				return err
			}
			i += 2
		case op == RJUMPV:
			if len(code) <= i+1 {
				return e(ErrTruncatedImmediate, i, "jump table size missing")
			}
			count := int(code[i+1])
			if count == 0 {
				return e(ErrInvalidBranchCount, i, "must not be 0")
			}
			if len(code) <= i+count {
				return e(ErrTruncatedImmediate, i, "jump table truncated")
			}
			for j := 0; j < count; j++ {
				if err := checkDest(code, &analysis, i+2+j*2, i+2*count+2, len(code)); err != nil {
					return err
				}
			}
			i += 1 + 2*count
		case op == CALLF:
			if i+2 >= len(code) {
				return e(ErrTruncatedImmediate, i, "op=%s", op)
			}
			arg, _ := parseUint16(code[i+1:])
			if arg >= len(metadata) {
				return e(ErrInvalidSectionArgument, i, "arg %d, last section %d", arg, len(metadata))
			}
			i += 2
		}
		i += 1
	}
	// Code sections may not "fall through" and require proper termination.
	// Therefore, the last instruction must be considered terminal.
	if !jt[op].terminal {
		return e(ErrInvalidCodeTermination, i, "ends with op %s", op)
	}
	if paths, err := validateControlFlow(code, section, metadata, jt); err != nil {
		return err
	} else if paths != count {
		// TODO(matt): return actual position of unreacable code
		return e(ErrUnreachableCode, 0, "")
	}
	return nil
}

// checkDest parses a relative offset at code[0:2] and checks if it is a valid jump destination.
func checkDest(code []byte, analysis *bitvec, imm, from, length int) error {
	if len(code) < imm+2 {
		return io.ErrUnexpectedEOF
	}
	if analysis != nil && *analysis == nil {
		*analysis = eofCodeBitmap(code)
	}
	offset := parseInt16(code[imm:])
	dest := from + offset
	if dest < 0 || dest >= length {
		return NewParseError(ErrInvalidJumpDest, imm, "relative offset out-of-bounds: offset %d, dest %d", offset, dest)
	}
	if !analysis.codeSegment(uint64(dest)) {
		return NewParseError(ErrInvalidJumpDest, imm, "relative offset into immediate value: offset %d, dest %d", offset, dest)
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
		e              = NewParseError
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
			if want, ok := heights[pos]; ok {
				if height != want {
					return 0, e(ErrConflictingStack, pos, "have %d, want %d", height, want)
				}
				// Already visited this path and stack height
				// matches.
				break
			}
			heights[pos] = height

			// Validate height for current op and update as needed.
			if jt[op].minStack > height {
				return 0, e(ErrStackUnderflow{stackLen: height, required: jt[op].minStack}, pos, "")
			}
			if jt[op].maxStack < height {
				return 0, e(ErrStackOverflow{stackLen: height, limit: jt[op].maxStack}, pos, "")
			}
			height += int(params.StackLimit) - jt[op].maxStack

			switch {
			case op == CALLF:
				arg, _ := parseUint16(code[pos+1:])
				if metadata[arg].Input > uint8(height) {
					return 0, e(ErrStackUnderflow{stackLen: height, required: int(metadata[arg].Input)}, pos, "CALLF underflow to section %d", arg)
				}
				if int(metadata[arg].Output)+height > int(params.StackLimit) {
					return 0, e(ErrStackOverflow{stackLen: int(metadata[arg].Output) + height, limit: int(params.StackLimit)}, pos, "CALLF overflow to section %d")
				}
				height -= int(metadata[arg].Input)
				height += int(metadata[arg].Output)
				pos += 3
			case op == RETF:
				if int(metadata[section].Output) != height {
					return 0, e(ErrInvalidOutputs, pos, "have %d, want %d", height, metadata[section].Output)
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
		return 0, e(ErrInvalidMaxStackHeight, 0, "at code section %d, have %d, want %d", section, maxStackHeight, metadata[section].MaxStackHeight)
	}
	return len(heights), nil
}
