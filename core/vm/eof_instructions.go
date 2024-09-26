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
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/common/math"
)

// opRjump implements the RJUMP opcode.
func opRjump(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	var (
		code   = scope.Contract.CodeAt(scope.CodeSection)
		offset = parseInt16(code[*pc+1:])
	)
	// move pc past op and operand (+3), add relative offset, subtract 1 to
	// account for interpreter loop.
	*pc = uint64(int64(*pc+3) + int64(offset) - 1)
	return nil, nil
}

// opRjumpi implements the RJUMPI opcode
func opRjumpi(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	condition := scope.Stack.pop()
	if condition.BitLen() == 0 {
		// Not branching, just skip over immediate argument.
		*pc += 2
		return nil, nil
	}
	return opRjump(pc, interpreter, scope)
}

// opRjumpv implements the RJUMPV opcode
func opRjumpv(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	var (
		code     = scope.Contract.CodeAt(scope.CodeSection)
		maxIndex = uint64(code[*pc+1]) + 1
		idx      = scope.Stack.pop()
	)
	if idx, overflow := idx.Uint64WithOverflow(); overflow || idx >= maxIndex {
		// Index out-of-bounds, don't branch, just skip over immediate
		// argument.
		*pc += 1 + maxIndex*2
		return nil, nil
	}
	offset := parseInt16(code[*pc+2+2*idx.Uint64():])
	// move pc past op and count byte (2), move past count number of 16bit offsets (count*2), add relative offset, subtract 1 to
	// account for interpreter loop.
	*pc = uint64(int64(*pc+2+maxIndex*2) + int64(offset) - 1)
	return nil, nil
}

// opCallf implements the CALLF opcode
func opCallf(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	var (
		code = scope.Contract.CodeAt(scope.CodeSection)
		idx  = binary.BigEndian.Uint16(code[*pc+1:])
		typ  = scope.Contract.Container.types[idx]
	)
	if scope.Stack.len()+int(typ.maxStackHeight)-int(typ.inputs) > 1024 {
		return nil, fmt.Errorf("stack overflow")
	}
	if scope.ReturnStack.Len() > 1024 {
		return nil, fmt.Errorf("return stack overflow")
	}
	retCtx := &ReturnContext{
		Section:     scope.CodeSection,
		Pc:          *pc + 3,
		StackHeight: scope.Stack.len() - int(typ.inputs),
	}
	scope.ReturnStack = append(scope.ReturnStack, retCtx)
	scope.CodeSection = uint64(idx)
	*pc = uint64(math.MaxUint64)
	return nil, nil
}

// opRetf implements the RETF opcode
func opRetf(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	retCtx := scope.ReturnStack.Pop()
	scope.CodeSection = retCtx.Section
	*pc = retCtx.Pc - 1

	// If returning from top frame, exit cleanly.
	if scope.ReturnStack.Len() == 0 {
		return nil, errStopToken
	}
	return nil, nil
}

// opJumpf implements the JUMPF opcode
func opJumpf(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	panic("not implemented")
}

// opEOFCreate implements the EOFCREATE opcode
func opEOFCreate(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	panic("not implemented")
}

// opReturnContract implements the RETURNCONTRACT opcode
func opReturnContract(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	panic("not implemented")
}

// opDataLoad implements the DATALOAD opcode
func opDataLoad(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	panic("not implemented")
}

// opDataLoadN implements the DATALOADN opcode
func opDataLoadN(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	panic("not implemented")
}

// opDataSize implements the DATASIZE opcode
func opDataSize(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	panic("not implemented")
}

// opDataCopy implements the DATACOPY opcode
func opDataCopy(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	panic("not implemented")
}

// opDupN implements the DUPN opcode
func opDupN(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	var (
		code  = scope.Contract.CodeAt(scope.CodeSection)
		index = int(code[*pc+1]) + 1
	)
	scope.Stack.dup(index)
	*pc += 1 // move past immediate
	return nil, nil
}

// opSwapN implements the SWAPN opcode
func opSwapN(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	var (
		code  = scope.Contract.CodeAt(scope.CodeSection)
		index = int(code[*pc+1]) + 1
	)
	scope.Stack.swap(index + 1)
	*pc += 1 // move past immediate
	return nil, nil
}

// opExchange implements the EXCHANGE opcode
func opExchange(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	var (
		code  = scope.Contract.CodeAt(scope.CodeSection)
		index = int(code[*pc+1])
		n     = (index >> 4) + 1
		m     = (index & 0x0F) + 1
	)
	scope.Stack.swapN(n+1, n+m+1)
	*pc += 1 // move past immediate
	return nil, nil
}

// opReturnDataLoad implements the RETURNDATALOAD opcode
func opReturnDataLoad(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	panic("not implemented")
}

// opExtCall implements the EOFCREATE opcode
func opExtCall(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	panic("not implemented")
}

// opExtDelegateCall implements the EXTDELEGATECALL opcode
func opExtDelegateCall(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	panic("not implemented")
}

// opExtStaticCall implements the EXTSTATICCALL opcode
func opExtStaticCall(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	panic("not implemented")
}
