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
	"fmt"
	"io"
)

// Below are all possible errors that can occur during validation of
// EOF containers.
var (
	ErrInvalidMagic             = errors.New("invalid magic")
	ErrUndefinedInstruction     = errors.New("undefined instruction")
	ErrTruncatedImmediate       = errors.New("truncated immediate")
	ErrInvalidSectionArgument   = errors.New("invalid section argument")
	ErrInvalidCallArgument      = errors.New("callf into non-returning section")
	ErrInvalidDataloadNArgument = errors.New("invalid dataloadN argument")
	ErrInvalidJumpDest          = errors.New("invalid jump destination")
	ErrInvalidBackwardJump      = errors.New("invalid backward jump")
	//ErrConflictingStack              = errors.New("conflicting stack height")
	//ErrInvalidBranchCount            = errors.New("invalid number of branches in jump table")
	ErrInvalidOutputs                = errors.New("invalid number of outputs")
	ErrInvalidMaxStackHeight         = errors.New("invalid max stack height")
	ErrInvalidCodeTermination        = errors.New("invalid code termination")
	ErrEOFCreateWithTruncatedSection = errors.New("eofcreate with truncated section")
	ErrOrphanedSubcontainer          = errors.New("subcontainer not referenced at all")
	ErrIncompatibleContainerKind     = errors.New("incompatible container kind")
	ErrStopAndReturnContract         = errors.New("Stop/Return and Returncontract in the same code section")
	ErrStopInInitCode                = errors.New("initcode contains a RETURN or STOP opcode")
	ErrTruncatedTopLevelContainer    = errors.New("truncated top level container")
	ErrUnreachableCode               = errors.New("unreachable code")
	ErrInvalidNonReturningFlag       = errors.New("invalid non-returning flag, bad RETF")
	ErrInvalidVersion                = errors.New("invalid version")
	ErrMissingTypeHeader             = errors.New("missing type header")
	ErrInvalidTypeSize               = errors.New("invalid type section size")
	ErrMissingCodeHeader             = errors.New("missing code header")
	ErrInvalidCodeSize               = errors.New("invalid code size")
	ErrInvalidContainerSectionSize   = errors.New("invalid container section size")
	ErrMissingDataHeader             = errors.New("missing data header")
	ErrMissingTerminator             = errors.New("missing header terminator")
	ErrTooManyInputs                 = errors.New("invalid type content, too many inputs")
	ErrTooManyOutputs                = errors.New("invalid type content, too many outputs")
	ErrInvalidSection0Type           = errors.New("invalid section 0 type, input and output should be zero and non-returning (0x80)")
	ErrTooLargeMaxStackHeight        = errors.New("invalid type content, max stack height exceeds limit")
	ErrInvalidContainerSize          = errors.New("invalid container size")
)

const (
	notRefByEither = iota
	refByReturnContract
	refByEOFCreate
)

type validationResult struct {
	visitedCode          map[int]struct{}
	visitedSubContainers map[int]int
	isInitCode           bool
	isRuntime            bool
}

// validateCode validates the code parameter against the EOF v1 validity requirements.
func validateCode(code []byte, section int, container *Container, jt *JumpTable, isInitCode bool) (*validationResult, error) {
	var (
		i = 0
		// Tracks the number of actual instructions in the code (e.g.
		// non-immediate values). This is used at the end to determine
		// if each instruction is reachable.
		count                = 0
		op                   OpCode
		analysis             bitvec
		visitedCode          = make(map[int]struct{})
		visitedSubcontainers = make(map[int]int)
		hasReturnContract    bool
		hasStop              bool
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
			return nil, fmt.Errorf("%w: op %s, pos %d", ErrUndefinedInstruction, op, i)
		}
		size := int(immediates[op])
		if size != 0 && len(code) <= i+size {
			return nil, fmt.Errorf("%w: op %s, pos %d", ErrTruncatedImmediate, op, i)
		}
		switch op {
		case RJUMP, RJUMPI:
			if err := checkDest(code, &analysis, i+1, i+3, len(code)); err != nil {
				return nil, err
			}
		case RJUMPV:
			max_size := int(code[i+1])
			length := max_size + 1
			if len(code) <= i+length {
				return nil, fmt.Errorf("%w: jump table truncated, op %s, pos %d", ErrTruncatedImmediate, op, i)
			}
			offset := i + 2
			for j := 0; j < length; j++ {
				if err := checkDest(code, &analysis, offset+j*2, offset+(length*2), len(code)); err != nil {
					return nil, err
				}
			}
			i += 2 * max_size
		case CALLF:
			arg, _ := parseUint16(code[i+1:])
			if arg >= len(container.types) {
				return nil, fmt.Errorf("%w: arg %d, last %d, pos %d", ErrInvalidSectionArgument, arg, len(container.types), i)
			}
			if container.types[arg].outputs == 0x80 {
				return nil, fmt.Errorf("%w: section %v", ErrInvalidCallArgument, arg)
			}
			visitedCode[arg] = struct{}{}
		case JUMPF:
			arg, _ := parseUint16(code[i+1:])
			if arg >= len(container.types) {
				return nil, fmt.Errorf("%w: arg %d, last %d, pos %d", ErrInvalidSectionArgument, arg, len(container.types), i)
			}
			if container.types[arg].outputs != 0x80 && container.types[arg].outputs > container.types[section].outputs {
				return nil, fmt.Errorf("%w: arg %d, last %d, pos %d", ErrInvalidOutputs, arg, len(container.types), i)
			}
			visitedCode[arg] = struct{}{}
		case DATALOADN:
			arg, _ := parseUint16(code[i+1:])
			// TODO why are we checking this? We should just pad
			if arg+32 > len(container.data) {
				return nil, fmt.Errorf("%w: arg %d, last %d, pos %d", ErrInvalidDataloadNArgument, arg, len(container.data), i)
			}
		case RETURNCONTRACT:
			if !isInitCode {
				return nil, ErrIncompatibleContainerKind
			}
			arg := int(code[i+1])
			if arg >= len(container.sections) {
				return nil, fmt.Errorf("%w: arg %d, last %d, pos %d", ErrUnreachableCode, arg, len(container.sections), i)
			}
			// We need to store per subcontainer how it was referenced
			if v, ok := visitedSubcontainers[arg]; ok && v != refByReturnContract {
				return nil, fmt.Errorf("section already referenced, arg :%d", arg)
			}
			if hasStop {
				return nil, ErrStopAndReturnContract
			}
			hasReturnContract = true
			visitedSubcontainers[arg] = refByReturnContract
		case EOFCREATE:
			arg := int(code[i+1])
			if arg >= len(container.sections) {
				return nil, fmt.Errorf("%w: arg %d, last %d, pos %d", ErrUnreachableCode, arg, len(container.sections), i)
			}
			if ct := container.sections[arg]; len(ct.data) != ct.dataSize {
				return nil, fmt.Errorf("%w: container %d, have %d, claimed %d, pos %d", ErrEOFCreateWithTruncatedSection, arg, len(ct.data), ct.dataSize, i)
			}
			if _, ok := visitedSubcontainers[arg]; ok {
				return nil, fmt.Errorf("section already referenced, arg :%d", arg)
			}
			// We need to store per subcontainer how it was referenced
			if v, ok := visitedSubcontainers[arg]; ok && v != refByEOFCreate {
				return nil, fmt.Errorf("section already referenced, arg :%d", arg)
			}
			visitedSubcontainers[arg] = refByEOFCreate
		case STOP, RETURN:
			if isInitCode {
				return nil, ErrStopInInitCode
			}
			if hasReturnContract {
				return nil, ErrStopAndReturnContract
			}
			hasStop = true
		}
		i += size + 1
	}
	// Code sections may not "fall through" and require proper termination.
	// Therefore, the last instruction must be considered terminal or RJUMP.
	if !terminals[op] && op != RJUMP {
		return nil, fmt.Errorf("%w: end with %s, pos %d", ErrInvalidCodeTermination, op, i)
	}
	if paths, err := validateControlFlow(code, section, container.types, jt); err != nil {
		return nil, err
	} else if paths != count {
		// TODO(matt): return actual position of unreachable code
		return nil, ErrUnreachableCode
	}
	return &validationResult{
		visitedCode:          visitedCode,
		visitedSubContainers: visitedSubcontainers,
		isInitCode:           hasReturnContract,
		isRuntime:            hasStop,
	}, nil
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
		return fmt.Errorf("%w: out-of-bounds offset: offset %d, dest %d, pos %d", ErrInvalidJumpDest, offset, dest, imm)
	}
	if !analysis.codeSegment(uint64(dest)) {
		return fmt.Errorf("%w: offset into immediate: offset %d, dest %d, pos %d", ErrInvalidJumpDest, offset, dest, imm)
	}
	return nil
}
