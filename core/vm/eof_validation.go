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

import (
	"errors"
	"fmt"
	"io"
)

// Below are all possible errors that can occur during validation of
// EOF containers.
var (
	errInvalidMagic                  = errors.New("invalid magic")
	errUndefinedInstruction          = errors.New("undefined instruction")
	errTruncatedImmediate            = errors.New("truncated immediate")
	errInvalidSectionArgument        = errors.New("invalid section argument")
	errInvalidCallArgument           = errors.New("callf into non-returning section")
	errInvalidDataloadNArgument      = errors.New("invalid dataloadN argument")
	errInvalidJumpDest               = errors.New("invalid jump destination")
	errInvalidBackwardJump           = errors.New("invalid backward jump")
	errInvalidOutputs                = errors.New("invalid number of outputs")
	errInvalidMaxStackHeight         = errors.New("invalid max stack height")
	errInvalidCodeTermination        = errors.New("invalid code termination")
	errEOFCreateWithTruncatedSection = errors.New("eofcreate with truncated section")
	errOrphanedSubcontainer          = errors.New("subcontainer not referenced at all")
	errIncompatibleContainerKind     = errors.New("incompatible container kind")
	errStopAndReturnContract         = errors.New("Stop/Return and Returncontract in the same code section")
	errStopInInitCode                = errors.New("initcode contains a RETURN or STOP opcode")
	errTruncatedTopLevelContainer    = errors.New("truncated top level container")
	errUnreachableCode               = errors.New("unreachable code")
	errInvalidNonReturningFlag       = errors.New("invalid non-returning flag, bad RETF")
	errInvalidVersion                = errors.New("invalid version")
	errMissingTypeHeader             = errors.New("missing type header")
	errInvalidTypeSize               = errors.New("invalid type section size")
	errMissingCodeHeader             = errors.New("missing code header")
	errInvalidCodeSize               = errors.New("invalid code size")
	errInvalidContainerSectionSize   = errors.New("invalid container section size")
	errMissingDataHeader             = errors.New("missing data header")
	errMissingTerminator             = errors.New("missing header terminator")
	errTooManyInputs                 = errors.New("invalid type content, too many inputs")
	errTooManyOutputs                = errors.New("invalid type content, too many outputs")
	errInvalidSection0Type           = errors.New("invalid section 0 type, input and output should be zero and non-returning (0x80)")
	errTooLargeMaxStackHeight        = errors.New("invalid type content, max stack height exceeds limit")
	errInvalidContainerSize          = errors.New("invalid container size")
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
		visitedCode          map[int]struct{}
		visitedSubcontainers map[int]int
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
			return nil, fmt.Errorf("%w: op %s, pos %d", errUndefinedInstruction, op, i)
		}
		size := int(immediates[op])
		if size != 0 && len(code) <= i+size {
			return nil, fmt.Errorf("%w: op %s, pos %d", errTruncatedImmediate, op, i)
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
				return nil, fmt.Errorf("%w: jump table truncated, op %s, pos %d", errTruncatedImmediate, op, i)
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
				return nil, fmt.Errorf("%w: arg %d, last %d, pos %d", errInvalidSectionArgument, arg, len(container.types), i)
			}
			if container.types[arg].outputs == 0x80 {
				return nil, fmt.Errorf("%w: section %v", errInvalidCallArgument, arg)
			}
			if visitedCode == nil {
				visitedCode = make(map[int]struct{})
			}
			visitedCode[arg] = struct{}{}
		case JUMPF:
			arg, _ := parseUint16(code[i+1:])
			if arg >= len(container.types) {
				return nil, fmt.Errorf("%w: arg %d, last %d, pos %d", errInvalidSectionArgument, arg, len(container.types), i)
			}
			if container.types[arg].outputs != 0x80 && container.types[arg].outputs > container.types[section].outputs {
				return nil, fmt.Errorf("%w: arg %d, last %d, pos %d", errInvalidOutputs, arg, len(container.types), i)
			}
			if visitedCode == nil {
				visitedCode = make(map[int]struct{})
			}
			visitedCode[arg] = struct{}{}
		case DATALOADN:
			arg, _ := parseUint16(code[i+1:])
			// TODO why are we checking this? We should just pad
			if arg+32 > len(container.data) {
				return nil, fmt.Errorf("%w: arg %d, last %d, pos %d", errInvalidDataloadNArgument, arg, len(container.data), i)
			}
		case RETURNCONTRACT:
			if !isInitCode {
				return nil, errIncompatibleContainerKind
			}
			arg := int(code[i+1])
			if arg >= len(container.subContainers) {
				return nil, fmt.Errorf("%w: arg %d, last %d, pos %d", errUnreachableCode, arg, len(container.subContainers), i)
			}
			if visitedSubcontainers == nil {
				visitedSubcontainers = make(map[int]int)
			}
			// We need to store per subcontainer how it was referenced
			if v, ok := visitedSubcontainers[arg]; ok && v != refByReturnContract {
				return nil, fmt.Errorf("section already referenced, arg :%d", arg)
			}
			if hasStop {
				return nil, errStopAndReturnContract
			}
			hasReturnContract = true
			visitedSubcontainers[arg] = refByReturnContract
		case EOFCREATE:
			arg := int(code[i+1])
			if arg >= len(container.subContainers) {
				return nil, fmt.Errorf("%w: arg %d, last %d, pos %d", errUnreachableCode, arg, len(container.subContainers), i)
			}
			if ct := container.subContainers[arg]; len(ct.data) != ct.dataSize {
				return nil, fmt.Errorf("%w: container %d, have %d, claimed %d, pos %d", errEOFCreateWithTruncatedSection, arg, len(ct.data), ct.dataSize, i)
			}
			if visitedSubcontainers == nil {
				visitedSubcontainers = make(map[int]int)
			}
			// We need to store per subcontainer how it was referenced
			if v, ok := visitedSubcontainers[arg]; ok && v != refByEOFCreate {
				return nil, fmt.Errorf("section already referenced, arg :%d", arg)
			}
			visitedSubcontainers[arg] = refByEOFCreate
		case STOP, RETURN:
			if isInitCode {
				return nil, errStopInInitCode
			}
			if hasReturnContract {
				return nil, errStopAndReturnContract
			}
			hasStop = true
		}
		i += size + 1
	}
	// Code sections may not "fall through" and require proper termination.
	// Therefore, the last instruction must be considered terminal or RJUMP.
	if !terminals[op] && op != RJUMP {
		return nil, fmt.Errorf("%w: end with %s, pos %d", errInvalidCodeTermination, op, i)
	}
	if paths, err := validateControlFlow(code, section, container.types, jt); err != nil {
		return nil, err
	} else if paths != count {
		// TODO(matt): return actual position of unreachable code
		return nil, errUnreachableCode
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
		return fmt.Errorf("%w: out-of-bounds offset: offset %d, dest %d, pos %d", errInvalidJumpDest, offset, dest, imm)
	}
	if !analysis.codeSegment(uint64(dest)) {
		return fmt.Errorf("%w: offset into immediate: offset %d, dest %d, pos %d", errInvalidJumpDest, offset, dest, imm)
	}
	return nil
}

//// disasm is a helper utility to show a sequence of comma-separated operations,
//// with immediates shown inline,
//// e.g: PUSH1(0x00),EOFCREATE(0x00),
//func disasm(code []byte) string {
//	var ops []string
//	for i := 0; i < len(code); i++ {
//		var op string
//		if args := immediates[code[i]]; args > 0 {
//			op = fmt.Sprintf("%v(%#x)", OpCode(code[i]).String(), code[i+1:i+1+int(args)])
//			i += int(args)
//		} else {
//			op = OpCode(code[i]).String()
//		}
//		ops = append(ops, op)
//	}
//	return strings.Join(ops, ",")
//}
