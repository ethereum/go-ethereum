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

import (
	"fmt"
	"math"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// calculateMemorySize calculates the required memory size
// it is important that this function is inlinable
func calculateMemorySize(offset, length *uint256.Int) (uint64, error) {
	memSize, overflow := calcMemSize64(offset, length)
	return memorySizeCeil(memSize, overflow)
}

// calculateMemorySizeU64 calculates the required memory size
// it is important that this function is inlinable
func calculateMemorySizeU64(offset *uint256.Int, length uint64) (uint64, error) {
	memSize, overflow := calcMemSize64WithUint(offset, length)
	return memorySizeCeil(memSize, overflow)
}

// calculateCallMemorySize calculates the required memory size for a call operation
func calculateCallMemorySize(argOffset, argSize, retOffset, retSize *uint256.Int) (uint64, error) {
	x, overflow := calcMemSize64(retOffset, retSize)
	if overflow {
		return 0, ErrGasUintOverflow
	}
	y, overflow := calcMemSize64(argOffset, argSize)
	if overflow {
		return 0, ErrGasUintOverflow
	}
	if x > y {
		return memorySizeCeil(x, false)
	}
	return memorySizeCeil(y, false)
}

func memorySizeCeil(memSize uint64, overflow bool) (uint64, error) {
	// memory is expanded in words of 32 bytes. Gas
	// is also calculated in words.
	if overflow || memSize > math.MaxUint64-31 {
		return 0, ErrGasUintOverflow
	}
	memorySize := ((memSize + 31) / 32) * 32
	return memorySize, nil
}

func opAdd(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	x, y, err := scope.Stack.pop2(1)
	if err != nil {
		return nil, err
	}
	y.Add(x, y)
	return nil, nil
}

func opSub(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	x, y, err := scope.Stack.pop2(1)
	if err != nil {
		return nil, err
	}
	y.Sub(x, y)
	return nil, nil
}

func opMul(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	x, y, err := scope.Stack.pop2(1)
	if err != nil {
		return nil, err
	}
	y.Mul(x, y)
	return nil, nil
}

func opDiv(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	x, y, err := scope.Stack.pop2(1)
	if err != nil {
		return nil, err
	}
	y.Div(x, y)
	return nil, nil
}

func opSdiv(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	x, y, err := scope.Stack.pop2(1)
	if err != nil {
		return nil, err
	}
	y.SDiv(x, y)
	return nil, nil
}

func opMod(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	x, y, err := scope.Stack.pop2(1)
	if err != nil {
		return nil, err
	}
	y.Mod(x, y)
	return nil, nil
}

func opSmod(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	x, y, err := scope.Stack.pop2(1)
	if err != nil {
		return nil, err
	}
	y.SMod(x, y)
	return nil, nil
}

func opExpEIP158(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	base, exponent, err := scope.Stack.pop2(1)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := gasExpEIP158(exponent)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}

	return opExp(base, exponent)
}

func opExpFrontier(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	base, exponent, err := scope.Stack.pop2(1)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := gasExpFrontier(exponent)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}

	return opExp(base, exponent)
}

func opExp(base, exponent *uint256.Int) ([]byte, error) {
	exponent.Exp(base, exponent)
	return nil, nil
}

func opSignExtend(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	back, num, err := scope.Stack.pop2(1)
	if err != nil {
		return nil, err
	}

	num.ExtendSign(num, back)
	return nil, nil
}

func opNot(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	x, err := scope.Stack.pop(1)
	if err != nil {
		return nil, err
	}

	x.Not(x)
	return nil, nil
}

func opLt(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	x, y, err := scope.Stack.pop2(1)
	if err != nil {
		return nil, err
	}
	if x.Lt(y) {
		y.SetOne()
	} else {
		y.Clear()
	}
	return nil, nil
}

func opGt(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	x, y, err := scope.Stack.pop2(1)
	if err != nil {
		return nil, err
	}
	if x.Gt(y) {
		y.SetOne()
	} else {
		y.Clear()
	}
	return nil, nil
}

func opSlt(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	x, y, err := scope.Stack.pop2(1)
	if err != nil {
		return nil, err
	}
	if x.Slt(y) {
		y.SetOne()
	} else {
		y.Clear()
	}
	return nil, nil
}

func opSgt(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	x, y, err := scope.Stack.pop2(1)
	if err != nil {
		return nil, err
	}
	if x.Sgt(y) {
		y.SetOne()
	} else {
		y.Clear()
	}
	return nil, nil
}

func opEq(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	x, y, err := scope.Stack.pop2(1)
	if err != nil {
		return nil, err
	}
	if x.Eq(y) {
		y.SetOne()
	} else {
		y.Clear()
	}
	return nil, nil
}

func opIszero(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	x, err := scope.Stack.pop(1)
	if err != nil {
		return nil, err
	}

	if x.IsZero() {
		x.SetOne()
	} else {
		x.Clear()
	}
	return nil, nil
}

func opAnd(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	x, y, err := scope.Stack.pop2(1)
	if err != nil {
		return nil, err
	}
	y.And(x, y)
	return nil, nil
}

func opOr(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	x, y, err := scope.Stack.pop2(1)
	if err != nil {
		return nil, err
	}
	y.Or(x, y)
	return nil, nil
}

func opXor(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	x, y, err := scope.Stack.pop2(1)
	if err != nil {
		return nil, err
	}
	y.Xor(x, y)
	return nil, nil
}

func opByte(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	th, val, err := scope.Stack.pop2(1)
	if err != nil {
		return nil, err
	}
	val.Byte(th)
	return nil, nil
}

func opAddmod(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	x, y, z, err := scope.Stack.pop3(1)
	if err != nil {
		return nil, err
	}
	z.AddMod(x, y, z)
	return nil, nil
}

func opMulmod(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	x, y, z, err := scope.Stack.pop3(1)
	if err != nil {
		return nil, err
	}
	z.MulMod(x, y, z)
	return nil, nil
}

// opSHL implements Shift Left
// The SHL instruction (shift left) pops 2 values from the stack, first arg1 and then arg2,
// and pushes on the stack arg2 shifted to the left by arg1 number of bits.
func opSHL(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	// Note, second operand is left in the stack; accumulate result into it, and no need to push it afterwards
	shift, value, err := scope.Stack.pop2(1)
	if err != nil {
		return nil, err
	}
	if shift.LtUint64(256) {
		value.Lsh(value, uint(shift.Uint64()))
	} else {
		value.Clear()
	}
	return nil, nil
}

// opSHR implements Logical Shift Right
// The SHR instruction (logical shift right) pops 2 values from the stack, first arg1 and then arg2,
// and pushes on the stack arg2 shifted to the right by arg1 number of bits with zero fill.
func opSHR(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	// Note, second operand is left in the stack; accumulate result into it, and no need to push it afterwards
	shift, value, err := scope.Stack.pop2(1)
	if err != nil {
		return nil, err
	}
	if shift.LtUint64(256) {
		value.Rsh(value, uint(shift.Uint64()))
	} else {
		value.Clear()
	}
	return nil, nil
}

// opSAR implements Arithmetic Shift Right
// The SAR instruction (arithmetic shift right) pops 2 values from the stack, first arg1 and then arg2,
// and pushes on the stack arg2 shifted to the right by arg1 number of bits with sign extension.
func opSAR(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	shift, value, err := scope.Stack.pop2(1)
	if err != nil {
		return nil, err
	}
	if shift.GtUint64(256) {
		if value.Sign() >= 0 {
			value.Clear()
		} else {
			// Max negative shift: all bits set
			value.SetAllOne()
		}
		return nil, nil
	}
	n := uint(shift.Uint64())
	value.SRsh(value, n)
	return nil, nil
}

func opKeccak256(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	offset, size, err := scope.Stack.pop2(1)
	if err != nil {
		return nil, err
	}
	memorySize, err := calculateMemorySize(offset, size)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := gasKeccak256(scope.Memory, memorySize, size)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}
	scope.Memory.Resize(memorySize)

	data := scope.Memory.GetPtr(offset.Uint64(), size.Uint64())
	interpreter.hasher.Reset()
	interpreter.hasher.Write(data)
	interpreter.hasher.Read(interpreter.hasherBuf[:])

	evm := interpreter.evm
	if evm.Config.EnablePreimageRecording {
		evm.StateDB.AddPreimage(interpreter.hasherBuf, data)
	}
	size.SetBytes(interpreter.hasherBuf[:])
	return nil, nil
}

func opAddress(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	return nil, scope.Stack.push(new(uint256.Int).SetBytes(scope.Contract.Address().Bytes()))
}

func opBalanceEIP4762(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	slot, err := scope.Stack.pop(1)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := gasBalance4762(interpreter.evm, scope.Contract, slot)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}

	return opBalance(interpreter, slot)
}

func opBalanceEIP2929(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	slot, err := scope.Stack.pop(1)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := gasEip2929AccountCheck(interpreter.evm, slot)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}

	return opBalance(interpreter, slot)
}

func opBalanceFrontier(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	slot, err := scope.Stack.pop(1)
	if err != nil {
		return nil, err
	}
	return opBalance(interpreter, slot)
}

func opBalance(interpreter *EVMInterpreter, slot *uint256.Int) ([]byte, error) {
	address := common.Address(slot.Bytes20())
	slot.Set(interpreter.evm.StateDB.GetBalance(address))
	return nil, nil
}

func opOrigin(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	return nil, scope.Stack.push(new(uint256.Int).SetBytes(interpreter.evm.Origin.Bytes()))
}

func opCaller(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	return nil, scope.Stack.push(new(uint256.Int).SetBytes(scope.Contract.Caller().Bytes()))
}

func opCallValue(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	return nil, scope.Stack.push(scope.Contract.value)
}

func opCallDataLoad(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	x, err := scope.Stack.pop(1)
	if err != nil {
		return nil, err
	}
	if offset, overflow := x.Uint64WithOverflow(); !overflow {
		data := getData(scope.Contract.Input, offset, 32)
		x.SetBytes(data)
	} else {
		x.Clear()
	}
	return nil, nil
}

func opCallDataSize(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	return nil, scope.Stack.push(new(uint256.Int).SetUint64(uint64(len(scope.Contract.Input))))
}

func opCallDataCopy(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	memOffset, dataOffset, length, err := scope.Stack.pop3(0)
	if err != nil {
		return nil, err
	}

	memorySize, err := calculateMemorySize(memOffset, length)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := memoryCopierGas(scope.Memory, memorySize, length)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}
	scope.Memory.Resize(memorySize)

	dataOffset64, overflow := dataOffset.Uint64WithOverflow()
	if overflow {
		dataOffset64 = math.MaxUint64
	}
	// These values are checked for overflow during gas cost calculation
	memOffset64 := memOffset.Uint64()
	length64 := length.Uint64()
	scope.Memory.Set(memOffset64, length64, getData(scope.Contract.Input, dataOffset64, length64))

	return nil, nil
}

func opReturnDataSize(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	return nil, scope.Stack.push(new(uint256.Int).SetUint64(uint64(len(interpreter.returnData))))
}

func opReturnDataCopy(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	memOffset, dataOffset, length, err := scope.Stack.pop3(0)
	if err != nil {
		return nil, err
	}
	memorySize, err := calculateMemorySize(memOffset, length)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := memoryCopierGas(scope.Memory, memorySize, length)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}
	scope.Memory.Resize(memorySize)

	offset64, overflow := dataOffset.Uint64WithOverflow()
	if overflow {
		return nil, ErrReturnDataOutOfBounds
	}
	// we can reuse dataOffset now (aliasing it for clarity)
	var end = dataOffset
	end.Add(dataOffset, length)
	end64, overflow := end.Uint64WithOverflow()
	if overflow || uint64(len(interpreter.returnData)) < end64 {
		return nil, ErrReturnDataOutOfBounds
	}
	scope.Memory.Set(memOffset.Uint64(), length.Uint64(), interpreter.returnData[offset64:end64])
	return nil, nil
}

func opExtCodeSizeEIP4762(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	slot, err := scope.Stack.pop(1)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := gasExtCodeSize4762(interpreter.evm, scope.Contract, slot)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}
	return opExtCodeSize(interpreter, slot)
}

func opExtCodeSizeEIP2929(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	slot, err := scope.Stack.pop(1)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := gasEip2929AccountCheck(interpreter.evm, slot)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}
	return opExtCodeSize(interpreter, slot)
}

func opExtCodeSizeFrontier(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	slot, err := scope.Stack.pop(1)
	if err != nil {
		return nil, err
	}
	return opExtCodeSize(interpreter, slot)
}

func opExtCodeSize(interpreter *EVMInterpreter, slot *uint256.Int) ([]byte, error) {
	slot.SetUint64(uint64(interpreter.evm.StateDB.GetCodeSize(slot.Bytes20())))
	return nil, nil
}

func opCodeSize(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	return nil, scope.Stack.push(new(uint256.Int).SetUint64(uint64(len(scope.Contract.Code))))
}

func opCodeCopyEIP4762(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	memOffset, codeOffset, length, err := scope.Stack.pop3(0)
	if err != nil {
		return nil, err
	}
	memorySize, err := calculateMemorySize(memOffset, length)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := gasCodeCopyEip4762(interpreter.evm, scope.Contract, scope.Memory, memorySize, codeOffset, length)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}
	scope.Memory.Resize(memorySize)
	return opCodeCopy(scope, memOffset, codeOffset, length)
}

func opCodeCopyFrontier(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	memOffset, codeOffset, length, err := scope.Stack.pop3(0)
	if err != nil {
		return nil, err
	}
	memorySize, err := calculateMemorySize(memOffset, length)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := memoryCopierGas(scope.Memory, memorySize, length)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}
	scope.Memory.Resize(memorySize)
	return opCodeCopy(scope, memOffset, codeOffset, length)
}

func opCodeCopy(scope *ScopeContext, memOffset, codeOffset, length *uint256.Int) ([]byte, error) {
	uint64CodeOffset, overflow := codeOffset.Uint64WithOverflow()
	if overflow {
		uint64CodeOffset = math.MaxUint64
	}

	codeCopy := getData(scope.Contract.Code, uint64CodeOffset, length.Uint64())
	scope.Memory.Set(memOffset.Uint64(), length.Uint64(), codeCopy)
	return nil, nil
}

func opExtCodeCopyEIP2929(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	a, memOffset, codeOffset, length, err := scope.Stack.pop4(0)
	if err != nil {
		return nil, err
	}
	memorySize, err := calculateMemorySize(memOffset, length)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := gasExtCodeCopyEIP2929(interpreter.evm, scope.Memory, memorySize, a, length)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}
	scope.Memory.Resize(memorySize)
	return opExtCodeCopy(interpreter, scope, a, memOffset, codeOffset, length)
}

func opExtCodeCopyFrontier(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	a, memOffset, codeOffset, length, err := scope.Stack.pop4(0)
	if err != nil {
		return nil, err
	}
	memorySize, err := calculateMemorySize(memOffset, length)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := memoryCopierGas(scope.Memory, memorySize, length)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}
	scope.Memory.Resize(memorySize)
	return opExtCodeCopy(interpreter, scope, a, memOffset, codeOffset, length)
}

func opExtCodeCopy(interpreter *EVMInterpreter, scope *ScopeContext, a, memOffset, codeOffset, length *uint256.Int) ([]byte, error) {
	uint64CodeOffset, overflow := codeOffset.Uint64WithOverflow()
	if overflow {
		uint64CodeOffset = math.MaxUint64
	}
	addr := common.Address(a.Bytes20())
	code := interpreter.evm.StateDB.GetCode(addr)
	codeCopy := getData(code, uint64CodeOffset, length.Uint64())
	scope.Memory.Set(memOffset.Uint64(), length.Uint64(), codeCopy)

	return nil, nil
}

func opExtCodeHashEIP4762(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	slot, err := scope.Stack.pop(1)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := gasExtCodeHash4762(interpreter.evm, scope.Contract, slot)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}
	return opExtCodeHash(interpreter, slot)
}

func opExtCodeHashEIP2929(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	slot, err := scope.Stack.pop(1)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := gasEip2929AccountCheck(interpreter.evm, slot)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}
	return opExtCodeHash(interpreter, slot)
}

func opExtCodeHashConstantinople(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	slot, err := scope.Stack.pop(1)
	if err != nil {
		return nil, err
	}
	return opExtCodeHash(interpreter, slot)
}

// opExtCodeHash returns the code hash of a specified account.
// There are several cases when the function is called, while we can relay everything
// to `state.GetCodeHash` function to ensure the correctness.
//
//  1. Caller tries to get the code hash of a normal contract account, state
//     should return the relative code hash and set it as the result.
//
//  2. Caller tries to get the code hash of a non-existent account, state should
//     return common.Hash{} and zero will be set as the result.
//
//  3. Caller tries to get the code hash for an account without contract code, state
//     should return emptyCodeHash(0xc5d246...) as the result.
//
//  4. Caller tries to get the code hash of a precompiled account, the result should be
//     zero or emptyCodeHash.
//
// It is worth noting that in order to avoid unnecessary create and clean, all precompile
// accounts on mainnet have been transferred 1 wei, so the return here should be
// emptyCodeHash. If the precompile account is not transferred any amount on a private or
// customized chain, the return value will be zero.
//
//  5. Caller tries to get the code hash for an account which is marked as self-destructed
//     in the current transaction, the code hash of this account should be returned.
//
//  6. Caller tries to get the code hash for an account which is marked as deleted, this
//     account should be regarded as a non-existent account and zero should be returned.
func opExtCodeHash(interpreter *EVMInterpreter, slot *uint256.Int) ([]byte, error) {
	address := common.Address(slot.Bytes20())
	if interpreter.evm.StateDB.Empty(address) {
		slot.Clear()
	} else {
		slot.SetBytes(interpreter.evm.StateDB.GetCodeHash(address).Bytes())
	}
	return nil, nil
}

func opGasprice(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	v, _ := uint256.FromBig(interpreter.evm.GasPrice)
	return nil, scope.Stack.push(v)
}

func opBlockhash(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	num, err := scope.Stack.pop(1)
	if err != nil {
		return nil, err
	}
	num64, overflow := num.Uint64WithOverflow()
	if overflow {
		num.Clear()
		return nil, nil
	}

	var upper, lower uint64
	upper = interpreter.evm.Context.BlockNumber.Uint64()
	if upper < 257 {
		lower = 0
	} else {
		lower = upper - 256
	}
	if num64 >= lower && num64 < upper {
		res := interpreter.evm.Context.GetHash(num64)
		if witness := interpreter.evm.StateDB.Witness(); witness != nil {
			witness.AddBlockHash(num64)
		}
		if interpreter.evm.Config.Tracer.OnBlockHashRead != nil {
			interpreter.evm.Config.Tracer.OnBlockHashRead(num64, res)
		}
		num.SetBytes(res[:])
	} else {
		num.Clear()
	}
	return nil, nil
}

func opCoinbase(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	return nil, scope.Stack.push(new(uint256.Int).SetBytes(interpreter.evm.Context.Coinbase.Bytes()))
}

func opTimestamp(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	return nil, scope.Stack.push(new(uint256.Int).SetUint64(interpreter.evm.Context.Time))
}

func opNumber(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	v, _ := uint256.FromBig(interpreter.evm.Context.BlockNumber)
	return nil, scope.Stack.push(v)
}

func opDifficulty(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	v, _ := uint256.FromBig(interpreter.evm.Context.Difficulty)
	return nil, scope.Stack.push(v)
}

func opRandom(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	v := new(uint256.Int).SetBytes(interpreter.evm.Context.Random.Bytes())
	return nil, scope.Stack.push(v)
}

func opGasLimit(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	return nil, scope.Stack.push(new(uint256.Int).SetUint64(interpreter.evm.Context.GasLimit))
}

func opPop(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	_, err := scope.Stack.pop(0)
	return nil, err
}

func opMload(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	v, err := scope.Stack.pop(1)
	if err != nil {
		return nil, err
	}
	memorySize, err := calculateMemorySizeU64(v, 32)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := memoryGasCost(scope.Memory, memorySize)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}
	scope.Memory.Resize(memorySize)

	offset := v.Uint64()
	v.SetBytes(scope.Memory.GetPtr(offset, 32))
	return nil, nil
}

func opMstore(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	mStart, val, err := scope.Stack.pop2(0)
	if err != nil {
		return nil, err
	}
	memorySize, err := calculateMemorySizeU64(mStart, 32)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := memoryGasCost(scope.Memory, memorySize)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}
	scope.Memory.Resize(memorySize)

	scope.Memory.Set32(mStart.Uint64(), val)
	return nil, nil
}

func opMstore8(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	off, val, err := scope.Stack.pop2(0)
	if err != nil {
		return nil, err
	}

	memorySize, err := calculateMemorySizeU64(off, 1)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := memoryGasCost(scope.Memory, memorySize)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}
	scope.Memory.Resize(memorySize)

	scope.Memory.store[off.Uint64()] = byte(val.Uint64())
	return nil, nil
}

func opSLoadEIP4762(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	loc, err := scope.Stack.pop(1)
	if err != nil {
		return nil, err
	}

	dynamicCost, err := gasSLoad4762(interpreter.evm, scope.Contract, loc)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}
	return opSload(interpreter, scope, loc)
}

func opSLoadEIP2929(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	loc, err := scope.Stack.pop(1)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := gasSLoadEIP2929(interpreter.evm, scope.Contract, loc)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}
	return opSload(interpreter, scope, loc)
}

func opSLoadFrontier(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	loc, err := scope.Stack.pop(1)
	if err != nil {
		return nil, err
	}
	return opSload(interpreter, scope, loc)
}

func opSload(interpreter *EVMInterpreter, scope *ScopeContext, loc *uint256.Int) ([]byte, error) {
	hash := common.Hash(loc.Bytes32())
	val := interpreter.evm.StateDB.GetState(scope.Contract.Address(), hash)
	loc.SetBytes(val.Bytes())
	return nil, nil
}

func opSstoreEIP4762(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	loc, val, err := scope.Stack.pop2(0)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := gasSStore4762(interpreter.evm, scope.Contract, loc)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}

	return opSstore(interpreter, scope, loc, val)
}

func opSstoreEIP3529(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	loc, val, err := scope.Stack.pop2(0)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := gasSStoreWithClearingRefund(interpreter.evm, scope.Contract, loc, val, params.SstoreClearsScheduleRefundEIP3529)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}

	return opSstore(interpreter, scope, loc, val)
}

func opSstoreEIP2200(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	loc, val, err := scope.Stack.pop2(0)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := gasSStoreEIP2200(interpreter.evm, scope.Contract, loc, val)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}

	return opSstore(interpreter, scope, loc, val)
}

func opSstoreEIP2929(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	loc, val, err := scope.Stack.pop2(0)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := gasSStoreWithClearingRefund(interpreter.evm, scope.Contract, loc, val, params.SstoreClearsScheduleRefundEIP2200)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}

	return opSstore(interpreter, scope, loc, val)
}

func opSstoreFrontier(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	loc, val, err := scope.Stack.pop2(0)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := gasSStore(interpreter.evm, scope.Contract, loc, val)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}

	return opSstore(interpreter, scope, loc, val)
}

func opSstore(interpreter *EVMInterpreter, scope *ScopeContext, loc, val *uint256.Int) ([]byte, error) {
	if interpreter.readOnly {
		return nil, ErrWriteProtection
	}
	interpreter.evm.StateDB.SetState(scope.Contract.Address(), loc.Bytes32(), val.Bytes32())
	return nil, nil
}

func opJump(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	if interpreter.evm.abort.Load() {
		return nil, errStopToken
	}
	pos, err := scope.Stack.pop(0)
	if err != nil {
		return nil, err
	}
	if !scope.Contract.validJumpdest(pos) {
		return nil, ErrInvalidJump
	}
	*pc = pos.Uint64() - 1 // pc will be increased by the interpreter loop
	return nil, nil
}

func opJumpi(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	if interpreter.evm.abort.Load() {
		return nil, errStopToken
	}
	pos, cond, err := scope.Stack.pop2(0)
	if err != nil {
		return nil, err
	}
	if !cond.IsZero() {
		if !scope.Contract.validJumpdest(pos) {
			return nil, ErrInvalidJump
		}
		*pc = pos.Uint64() - 1 // pc will be increased by the interpreter loop
	}
	return nil, nil
}

func opJumpdest(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	return nil, nil
}

func opPc(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	return nil, scope.Stack.push(new(uint256.Int).SetUint64(*pc))
}

func opMsize(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	return nil, scope.Stack.push(new(uint256.Int).SetUint64(uint64(scope.Memory.Len())))
}

func opGas(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	return nil, scope.Stack.push(new(uint256.Int).SetUint64(scope.Contract.Gas))
}

func opSwap1(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	return nil, scope.Stack.swap1()
}

func opSwap2(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	return nil, scope.Stack.swap2()
}

func opSwap3(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	return nil, scope.Stack.swap3()
}

func opSwap4(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	return nil, scope.Stack.swap4()
}

func opSwap5(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	return nil, scope.Stack.swap5()
}

func opSwap6(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	return nil, scope.Stack.swap6()
}

func opSwap7(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	return nil, scope.Stack.swap7()
}

func opSwap8(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	return nil, scope.Stack.swap8()
}

func opSwap9(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	return nil, scope.Stack.swap9()
}

func opSwap10(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	return nil, scope.Stack.swap10()
}

func opSwap11(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	return nil, scope.Stack.swap11()
}

func opSwap12(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	return nil, scope.Stack.swap12()
}

func opSwap13(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	return nil, scope.Stack.swap13()
}

func opSwap14(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	return nil, scope.Stack.swap14()
}

func opSwap15(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	return nil, scope.Stack.swap15()
}

func opSwap16(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	return nil, scope.Stack.swap16()
}

func opCreateEIP3860(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	value, offset, size, err := scope.Stack.pop3(0)
	if err != nil {
		return nil, err
	}
	memorySize, err := calculateMemorySize(offset, size)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := gasCreateEip3860(scope.Memory, memorySize, size)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}
	scope.Memory.Resize(memorySize)

	return opCreate(interpreter, scope, value, offset, size)
}

func opCreateFrontier(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	value, offset, size, err := scope.Stack.pop3(0)
	if err != nil {
		return nil, err
	}
	memorySize, err := calculateMemorySize(offset, size)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := memoryGasCost(scope.Memory, memorySize)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}
	scope.Memory.Resize(memorySize)

	return opCreate(interpreter, scope, value, offset, size)
}

func opCreate(interpreter *EVMInterpreter, scope *ScopeContext, value, offset, size *uint256.Int) ([]byte, error) {
	if interpreter.readOnly {
		return nil, ErrWriteProtection
	}

	var (
		input = scope.Memory.GetCopy(offset.Uint64(), size.Uint64())
		gas   = scope.Contract.Gas
	)
	if interpreter.evm.chainRules.IsEIP150 {
		gas -= gas / 64
	}

	// reuse size int for stackvalue
	stackvalue := size

	scope.Contract.UseGas(gas, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallContractCreation)

	res, addr, returnGas, suberr := interpreter.evm.Create(scope.Contract.Address(), input, gas, value)
	// Push item on the stack based on the returned error. If the ruleset is
	// homestead we must check for CodeStoreOutOfGasError (homestead only
	// rule) and treat as an error, if the ruleset is frontier we must
	// ignore this error and pretend the operation was successful.
	if interpreter.evm.chainRules.IsHomestead && suberr == ErrCodeStoreOutOfGas {
		stackvalue.Clear()
	} else if suberr != nil && suberr != ErrCodeStoreOutOfGas {
		stackvalue.Clear()
	} else {
		stackvalue.SetBytes(addr.Bytes())
	}
	scope.Stack.push(stackvalue)

	scope.Contract.RefundGas(returnGas, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallLeftOverRefunded)

	if suberr == ErrExecutionReverted {
		interpreter.returnData = res // set REVERT data to return data buffer
		return res, nil
	}
	interpreter.returnData = nil // clear dirty return data buffer
	return nil, nil
}

func opCreate2EIP3860(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	endowment, offset, size, salt, err := scope.Stack.pop4(0)
	if err != nil {
		return nil, err
	}
	memorySize, err := calculateMemorySize(offset, size)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := gasCreate2Eip3860(scope.Memory, memorySize, size)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}
	scope.Memory.Resize(memorySize)

	return opCreate2(interpreter, scope, endowment, offset, size, salt)
}

func opCreate2Constantinople(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	endowment, offset, size, salt, err := scope.Stack.pop4(0)
	if err != nil {
		return nil, err
	}
	memorySize, err := calculateMemorySize(offset, size)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := gasCreate2(scope.Memory, memorySize, size)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}
	scope.Memory.Resize(memorySize)

	return opCreate2(interpreter, scope, endowment, offset, size, salt)
}

func opCreate2(interpreter *EVMInterpreter, scope *ScopeContext, endowment, offset, size, salt *uint256.Int) ([]byte, error) {
	if interpreter.readOnly {
		return nil, ErrWriteProtection
	}
	var (
		input = scope.Memory.GetCopy(offset.Uint64(), size.Uint64())
		gas   = scope.Contract.Gas
	)

	// Apply EIP150
	gas -= gas / 64
	scope.Contract.UseGas(gas, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallContractCreation2)
	// reuse size int for stackvalue
	stackvalue := size
	res, addr, returnGas, suberr := interpreter.evm.Create2(scope.Contract.Address(), input, gas,
		endowment, salt)
	// Push item on the stack based on the returned error.
	if suberr != nil {
		stackvalue.Clear()
	} else {
		stackvalue.SetBytes(addr.Bytes())
	}
	scope.Stack.push(stackvalue)
	scope.Contract.RefundGas(returnGas, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallLeftOverRefunded)

	if suberr == ErrExecutionReverted {
		interpreter.returnData = res // set REVERT data to return data buffer
		return res, nil
	}
	interpreter.returnData = nil // clear dirty return data buffer
	return nil, nil
}

func opCallEIP4762(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	gas, addr, value, inOffset, inSize, retOffset, retSize, err := scope.Stack.pop7(0)
	if err != nil {
		return nil, err
	}
	memorySize, err := calculateCallMemorySize(inOffset, inSize, retOffset, retSize)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := gasCallEIP4762(interpreter.evm, scope.Contract, func() (uint64, error) {
		return gasCall(interpreter.evm, scope.Contract, scope.Memory, memorySize, gas, addr, value)
	}, addr, value, true)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}
	scope.Memory.Resize(memorySize)

	return opCall(interpreter, scope, gas, addr, value, inOffset, inSize, retOffset, retSize)
}

func opCallEIP7702(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	gas, addr, value, inOffset, inSize, retOffset, retSize, err := scope.Stack.pop7(0)
	if err != nil {
		return nil, err
	}
	memorySize, err := calculateCallMemorySize(inOffset, inSize, retOffset, retSize)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := gasCallEIP7702(interpreter.evm, scope.Contract, addr, func() (uint64, error) {
		return gasCall(interpreter.evm, scope.Contract, scope.Memory, memorySize, gas, addr, value)
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}
	scope.Memory.Resize(memorySize)

	return opCall(interpreter, scope, gas, addr, value, inOffset, inSize, retOffset, retSize)
}

func opCallEIP2929(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	gas, addr, value, inOffset, inSize, retOffset, retSize, err := scope.Stack.pop7(0)
	if err != nil {
		return nil, err
	}
	memorySize, err := calculateCallMemorySize(inOffset, inSize, retOffset, retSize)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := gasCallEIP2929(interpreter.evm, scope.Contract, addr, func() (uint64, error) {
		return gasCall(interpreter.evm, scope.Contract, scope.Memory, memorySize, gas, addr, value)
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}
	scope.Memory.Resize(memorySize)

	return opCall(interpreter, scope, gas, addr, value, inOffset, inSize, retOffset, retSize)
}

func opCallFrontier(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	gas, addr, value, inOffset, inSize, retOffset, retSize, err := scope.Stack.pop7(0)
	if err != nil {
		return nil, err
	}
	memorySize, err := calculateCallMemorySize(inOffset, inSize, retOffset, retSize)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := gasCall(interpreter.evm, scope.Contract, scope.Memory, memorySize, gas, addr, value)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}
	scope.Memory.Resize(memorySize)

	return opCall(interpreter, scope, gas, addr, value, inOffset, inSize, retOffset, retSize)
}

func opCall(interpreter *EVMInterpreter, scope *ScopeContext,
	temp, addr, value, inOffset, inSize, retOffset, retSize *uint256.Int,
) ([]byte, error) {
	stack := scope.Stack
	gas := interpreter.evm.callGasTemp
	// Pop other call parameters.
	toAddr := common.Address(addr.Bytes20())
	// Get the arguments from the memory.
	args := scope.Memory.GetPtr(inOffset.Uint64(), inSize.Uint64())

	if interpreter.readOnly && !value.IsZero() {
		return nil, ErrWriteProtection
	}
	if !value.IsZero() {
		gas += params.CallStipend
	}
	ret, returnGas, err := interpreter.evm.Call(scope.Contract.Address(), toAddr, args, gas, value)

	if err != nil {
		temp.Clear()
	} else {
		temp.SetOne()
	}
	if err == nil || err == ErrExecutionReverted {
		scope.Memory.Set(retOffset.Uint64(), retSize.Uint64(), ret)
	}
	stack.push(temp)

	scope.Contract.RefundGas(returnGas, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallLeftOverRefunded)

	interpreter.returnData = ret
	return ret, nil
}

func opCallCodeEIP4762(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	gas, addr, value, inOffset, inSize, retOffset, retSize, err := scope.Stack.pop7(0)
	if err != nil {
		return nil, err
	}
	memorySize, err := calculateCallMemorySize(inOffset, inSize, retOffset, retSize)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := gasCallEIP4762(interpreter.evm, scope.Contract, func() (uint64, error) {
		return gasCallCode(interpreter.evm, scope.Contract, scope.Memory, memorySize, gas, value)
	}, addr, value, false)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}
	scope.Memory.Resize(memorySize)

	return opCallCode(interpreter, scope, gas, addr, value, inOffset, inSize, retOffset, retSize)
}

func opCallCodeEIP7702(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	gas, addr, value, inOffset, inSize, retOffset, retSize, err := scope.Stack.pop7(0)
	if err != nil {
		return nil, err
	}
	memorySize, err := calculateCallMemorySize(inOffset, inSize, retOffset, retSize)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := gasCallEIP7702(interpreter.evm, scope.Contract, addr, func() (uint64, error) {
		return gasCallCode(interpreter.evm, scope.Contract, scope.Memory, memorySize, gas, value)
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}
	scope.Memory.Resize(memorySize)

	return opCallCode(interpreter, scope, gas, addr, value, inOffset, inSize, retOffset, retSize)
}

func opCallCodeEIP2929(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	gas, addr, value, inOffset, inSize, retOffset, retSize, err := scope.Stack.pop7(0)
	if err != nil {
		return nil, err
	}
	memorySize, err := calculateCallMemorySize(inOffset, inSize, retOffset, retSize)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := gasCallEIP2929(interpreter.evm, scope.Contract, addr, func() (uint64, error) {
		return gasCallCode(interpreter.evm, scope.Contract, scope.Memory, memorySize, gas, value)
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}
	scope.Memory.Resize(memorySize)

	return opCallCode(interpreter, scope, gas, addr, value, inOffset, inSize, retOffset, retSize)
}

func opCallCodeFrontier(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	gas, addr, value, inOffset, inSize, retOffset, retSize, err := scope.Stack.pop7(0)
	if err != nil {
		return nil, err
	}
	memorySize, err := calculateCallMemorySize(inOffset, inSize, retOffset, retSize)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := gasCallCode(interpreter.evm, scope.Contract, scope.Memory, memorySize, gas, value)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}
	scope.Memory.Resize(memorySize)

	return opCallCode(interpreter, scope, gas, addr, value, inOffset, inSize, retOffset, retSize)
}

func opCallCode(interpreter *EVMInterpreter, scope *ScopeContext,
	temp, addr, value, inOffset, inSize, retOffset, retSize *uint256.Int,
) ([]byte, error) {
	gas := interpreter.evm.callGasTemp
	// Pop other call parameters.
	toAddr := common.Address(addr.Bytes20())
	// Get arguments from the memory.
	args := scope.Memory.GetPtr(inOffset.Uint64(), inSize.Uint64())

	if !value.IsZero() {
		gas += params.CallStipend
	}

	ret, returnGas, err := interpreter.evm.CallCode(scope.Contract.Address(), toAddr, args, gas, value)
	if err != nil {
		temp.Clear()
	} else {
		temp.SetOne()
	}
	if err == nil || err == ErrExecutionReverted {
		scope.Memory.Set(retOffset.Uint64(), retSize.Uint64(), ret)
	}
	scope.Stack.push(temp)

	scope.Contract.RefundGas(returnGas, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallLeftOverRefunded)

	interpreter.returnData = ret
	return ret, nil
}

func opDelegateCallEIP7702(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	gas, addr, inOffset, inSize, retOffset, retSize, err := scope.Stack.pop6(0)
	if err != nil {
		return nil, err
	}
	memorySize, err := calculateCallMemorySize(inOffset, inSize, retOffset, retSize)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := gasCallEIP7702(interpreter.evm, scope.Contract, addr, func() (uint64, error) {
		return gasDelegateCall(interpreter.evm, scope.Contract, scope.Memory, memorySize, gas)
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}
	scope.Memory.Resize(memorySize)

	return opDelegateCall(interpreter, scope, gas, addr, inOffset, inSize, retOffset, retSize)
}

func opDelegateCallEIP2929(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	gas, addr, inOffset, inSize, retOffset, retSize, err := scope.Stack.pop6(0)
	if err != nil {
		return nil, err
	}
	memorySize, err := calculateCallMemorySize(inOffset, inSize, retOffset, retSize)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := gasCallEIP2929(interpreter.evm, scope.Contract, addr, func() (uint64, error) {
		return gasDelegateCall(interpreter.evm, scope.Contract, scope.Memory, memorySize, gas)
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}
	scope.Memory.Resize(memorySize)

	return opDelegateCall(interpreter, scope, gas, addr, inOffset, inSize, retOffset, retSize)
}

func opDelegateCallEIP4762(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	gas, addr, inOffset, inSize, retOffset, retSize, err := scope.Stack.pop6(0)
	if err != nil {
		return nil, err
	}
	memorySize, err := calculateCallMemorySize(inOffset, inSize, retOffset, retSize)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := gasCallEIP4762(interpreter.evm, scope.Contract, func() (uint64, error) {
		return gasDelegateCall(interpreter.evm, scope.Contract, scope.Memory, memorySize, gas)
	}, addr, nil, false)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}
	scope.Memory.Resize(memorySize)

	return opDelegateCall(interpreter, scope, gas, addr, inOffset, inSize, retOffset, retSize)
}

func opDelegateCall(interpreter *EVMInterpreter, scope *ScopeContext,
	temp, addr, inOffset, inSize, retOffset, retSize *uint256.Int,
) ([]byte, error) {
	stack := scope.Stack
	gas := interpreter.evm.callGasTemp
	toAddr := common.Address(addr.Bytes20())
	// Get arguments from the memory.
	args := scope.Memory.GetPtr(inOffset.Uint64(), inSize.Uint64())

	ret, returnGas, err := interpreter.evm.DelegateCall(scope.Contract.Caller(), scope.Contract.Address(), toAddr, args, gas, scope.Contract.value)
	if err != nil {
		temp.Clear()
	} else {
		temp.SetOne()
	}
	if err == nil || err == ErrExecutionReverted {
		scope.Memory.Set(retOffset.Uint64(), retSize.Uint64(), ret)
	}
	stack.push(temp)

	scope.Contract.RefundGas(returnGas, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallLeftOverRefunded)

	interpreter.returnData = ret
	return ret, nil
}

func opDelegateCallHomestead(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	gas, addr, inOffset, inSize, retOffset, retSize, err := scope.Stack.pop6(0)
	if err != nil {
		return nil, err
	}
	memorySize, err := calculateCallMemorySize(inOffset, inSize, retOffset, retSize)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := gasDelegateCall(interpreter.evm, scope.Contract, scope.Memory, memorySize, gas)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}
	scope.Memory.Resize(memorySize)

	return opDelegateCall(interpreter, scope, gas, addr, inOffset, inSize, retOffset, retSize)
}

func opStaticCallEIP7702(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	gas, addr, inOffset, inSize, retOffset, retSize, err := scope.Stack.pop6(0)
	if err != nil {
		return nil, err
	}
	memorySize, err := calculateCallMemorySize(inOffset, inSize, retOffset, retSize)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := gasCallEIP7702(interpreter.evm, scope.Contract, addr, func() (uint64, error) {
		return gasStaticCall(interpreter.evm, scope.Contract, scope.Memory, memorySize, gas)
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}
	scope.Memory.Resize(memorySize)

	return opStaticCall(interpreter, scope, gas, addr, inOffset, inSize, retOffset, retSize)
}

func opStaticCallByzantium(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	gas, addr, inOffset, inSize, retOffset, retSize, err := scope.Stack.pop6(0)
	if err != nil {
		return nil, err
	}
	memorySize, err := calculateCallMemorySize(inOffset, inSize, retOffset, retSize)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := gasStaticCall(interpreter.evm, scope.Contract, scope.Memory, memorySize, gas)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}
	scope.Memory.Resize(memorySize)

	return opStaticCall(interpreter, scope, gas, addr, inOffset, inSize, retOffset, retSize)
}

func opStaticCallEIP2929(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	gas, addr, inOffset, inSize, retOffset, retSize, err := scope.Stack.pop6(0)
	if err != nil {
		return nil, err
	}
	memorySize, err := calculateCallMemorySize(inOffset, inSize, retOffset, retSize)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := gasCallEIP2929(interpreter.evm, scope.Contract, addr, func() (uint64, error) {
		return gasStaticCall(interpreter.evm, scope.Contract, scope.Memory, memorySize, gas)
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}
	scope.Memory.Resize(memorySize)

	return opStaticCall(interpreter, scope, gas, addr, inOffset, inSize, retOffset, retSize)
}

func opStaticCallEIP4762(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	gas, addr, inOffset, inSize, retOffset, retSize, err := scope.Stack.pop6(0)
	if err != nil {
		return nil, err
	}
	memorySize, err := calculateCallMemorySize(inOffset, inSize, retOffset, retSize)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := gasCallEIP4762(interpreter.evm, scope.Contract, func() (uint64, error) {
		return gasStaticCall(interpreter.evm, scope.Contract, scope.Memory, memorySize, gas)
	}, addr, nil, false)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}
	scope.Memory.Resize(memorySize)

	return opStaticCall(interpreter, scope, gas, addr, inOffset, inSize, retOffset, retSize)
}

func opStaticCall(interpreter *EVMInterpreter, scope *ScopeContext,
	temp, addr, inOffset, inSize, retOffset, retSize *uint256.Int,
) ([]byte, error) {
	// We use it as a temporary value
	gas := interpreter.evm.callGasTemp
	// Pop other call parameters.
	toAddr := common.Address(addr.Bytes20())
	// Get arguments from the memory.
	args := scope.Memory.GetPtr(inOffset.Uint64(), inSize.Uint64())

	ret, returnGas, err := interpreter.evm.StaticCall(scope.Contract.Address(), toAddr, args, gas)
	if err != nil {
		temp.Clear()
	} else {
		temp.SetOne()
	}
	if err == nil || err == ErrExecutionReverted {
		scope.Memory.Set(retOffset.Uint64(), retSize.Uint64(), ret)
	}
	scope.Stack.push(temp)

	scope.Contract.RefundGas(returnGas, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallLeftOverRefunded)

	interpreter.returnData = ret
	return ret, nil
}

func opReturn(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	offset, size, err := scope.Stack.pop2(0)
	if err != nil {
		return nil, err
	}
	memorySize, err := calculateMemorySize(offset, size)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := memoryGasCost(scope.Memory, memorySize)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}
	scope.Memory.Resize(memorySize)

	ret := scope.Memory.GetCopy(offset.Uint64(), size.Uint64())

	return ret, errStopToken
}

func opRevert(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	offset, size, err := scope.Stack.pop2(0)
	if err != nil {
		return nil, err
	}
	memorySize, err := calculateMemorySize(offset, size)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := memoryGasCost(scope.Memory, memorySize)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}
	scope.Memory.Resize(memorySize)

	ret := scope.Memory.GetCopy(offset.Uint64(), size.Uint64())

	interpreter.returnData = ret
	return ret, ErrExecutionReverted
}

func opUndefined(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	return nil, &ErrInvalidOpCode{opcode: OpCode(scope.Contract.Code[*pc])}
}

func opStop(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	return nil, errStopToken
}

func opSelfdestructEIP6780(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	beneficiary, err := scope.Stack.pop(0)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := gasSelfdestructEIP(interpreter.evm, scope.Contract, beneficiary, false)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}

	return opSelfdestruct6780(interpreter, scope, beneficiary)
}

func opSelfdestructEIP3529(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	beneficiary, err := scope.Stack.pop(0)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := gasSelfdestructEIP(interpreter.evm, scope.Contract, beneficiary, false)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}

	return opSelfdestruct(interpreter, scope, beneficiary)
}

func opSelfdestructEIP2929(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	beneficiary, err := scope.Stack.pop(0)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := gasSelfdestructEIP(interpreter.evm, scope.Contract, beneficiary, true)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}

	return opSelfdestruct(interpreter, scope, beneficiary)
}

func opSelfdestructFrontier(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	beneficiary, err := scope.Stack.pop(0)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := gasSelfdestruct(interpreter.evm, scope.Contract, beneficiary)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}

	return opSelfdestruct(interpreter, scope, beneficiary)
}

func opSelfdestruct(interpreter *EVMInterpreter, scope *ScopeContext, beneficiary *uint256.Int) ([]byte, error) {
	if interpreter.readOnly {
		return nil, ErrWriteProtection
	}

	balance := interpreter.evm.StateDB.GetBalance(scope.Contract.Address())
	interpreter.evm.StateDB.AddBalance(beneficiary.Bytes20(), balance, tracing.BalanceIncreaseSelfdestruct)
	interpreter.evm.StateDB.SelfDestruct(scope.Contract.Address())
	if interpreter.evm.Config.Tracer.OnEnter != nil {
		interpreter.evm.Config.Tracer.OnEnter(interpreter.evm.depth, byte(SELFDESTRUCT), scope.Contract.Address(), beneficiary.Bytes20(), []byte{}, 0, balance.ToBig())
	}
	if interpreter.evm.Config.Tracer.OnExit != nil {
		interpreter.evm.Config.Tracer.OnExit(interpreter.evm.depth, []byte{}, 0, nil, false)
	}
	return nil, errStopToken
}

func opSelfdestructEIP4762(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	beneficiary, err := scope.Stack.pop(0)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := gasSelfdestructEIP4762(interpreter.evm, scope.Contract, beneficiary)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}

	return opSelfdestruct6780(interpreter, scope, beneficiary)
}

func opSelfdestruct6780(interpreter *EVMInterpreter, scope *ScopeContext, beneficiary *uint256.Int) ([]byte, error) {
	if interpreter.readOnly {
		return nil, ErrWriteProtection
	}
	balance := interpreter.evm.StateDB.GetBalance(scope.Contract.Address())
	interpreter.evm.StateDB.SubBalance(scope.Contract.Address(), balance, tracing.BalanceDecreaseSelfdestruct)
	interpreter.evm.StateDB.AddBalance(beneficiary.Bytes20(), balance, tracing.BalanceIncreaseSelfdestruct)
	interpreter.evm.StateDB.SelfDestruct6780(scope.Contract.Address())
	if interpreter.evm.Config.Tracer.OnEnter != nil {
		interpreter.evm.Config.Tracer.OnEnter(interpreter.evm.depth, byte(SELFDESTRUCT), scope.Contract.Address(), beneficiary.Bytes20(), []byte{}, 0, balance.ToBig())
	}
	if interpreter.evm.Config.Tracer.OnExit != nil {
		interpreter.evm.Config.Tracer.OnExit(interpreter.evm.depth, []byte{}, 0, nil, false)
	}
	return nil, errStopToken
}

// following functions are used by the instruction jump  table

// make log instruction function
func makeLog(size uint64) executionFunc {
	return func(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
		if interpreter.readOnly {
			return nil, ErrWriteProtection
		}
		mStart, mSize, err := scope.Stack.pop2(0)
		if err != nil {
			return nil, err
		}
		topics := make([]common.Hash, size)
		for i := uint64(0); i < size; i++ {
			addr, err := scope.Stack.pop(0)
			if err != nil {
				return nil, err
			}
			topics[i] = addr.Bytes32()
		}

		memorySize, err := calculateMemorySize(mStart, mSize)
		if err != nil {
			return nil, err
		}
		dynamicCost, err := gasLog(scope.Memory, memorySize, size, mSize)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
		}
		if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
			return nil, ErrOutOfGas
		}
		scope.Memory.Resize(memorySize)

		d := scope.Memory.GetCopy(mStart.Uint64(), mSize.Uint64())
		interpreter.evm.StateDB.AddLog(&types.Log{
			Address: scope.Contract.Address(),
			Topics:  topics,
			Data:    d,
			// This is a non-consensus field, but assigned here because
			// core/state doesn't know the current block number.
			BlockNumber: interpreter.evm.Context.BlockNumber.Uint64(),
		})

		return nil, nil
	}
}

// opPush1 is a specialized version of pushN
func opPush1(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	var (
		codeLen = uint64(len(scope.Contract.Code))
		integer = new(uint256.Int)
	)
	*pc += 1
	if *pc < codeLen {
		return nil, scope.Stack.push(integer.SetUint64(uint64(scope.Contract.Code[*pc])))
	}
	return nil, scope.Stack.push(integer.Clear())
}

// opPush2 is a specialized version of pushN
func opPush2(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	var (
		codeLen = uint64(len(scope.Contract.Code))
		integer = new(uint256.Int)
	)
	var err error
	if *pc+2 < codeLen {
		err = scope.Stack.push(integer.SetBytes2(scope.Contract.Code[*pc+1 : *pc+3]))
	} else if *pc+1 < codeLen {
		err = scope.Stack.push(integer.SetUint64(uint64(scope.Contract.Code[*pc+1]) << 8))
	} else {
		err = scope.Stack.push(integer.Clear())
	}
	if err != nil {
		return nil, err
	}
	*pc += 2
	return nil, nil
}

// make push instruction function
func makePush(size uint64, pushByteSize int) executionFunc {
	return func(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
		var (
			codeLen = len(scope.Contract.Code)
			start   = min(codeLen, int(*pc+1))
			end     = min(codeLen, start+pushByteSize)
		)
		a := new(uint256.Int).SetBytes(scope.Contract.Code[start:end])

		// Missing bytes: pushByteSize - len(pushData)
		if missing := pushByteSize - (end - start); missing > 0 {
			a.Lsh(a, uint(8*missing))
		}
		*pc += size
		return nil, scope.Stack.push(a)
	}
}

func opDup1(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	return nil, scope.Stack.dup1()
}

func opDup2(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	return nil, scope.Stack.dup2()
}

func opDup3(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	return nil, scope.Stack.dup3()
}

func opDup4(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	return nil, scope.Stack.dup4()
}

func opDup5(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	return nil, scope.Stack.dup5()
}

func opDup6(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	return nil, scope.Stack.dup6()
}

func opDup7(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	return nil, scope.Stack.dup7()
}

func opDup8(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	return nil, scope.Stack.dup8()
}

func opDup9(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	return nil, scope.Stack.dup9()
}

func opDup10(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	return nil, scope.Stack.dup10()
}

func opDup11(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	return nil, scope.Stack.dup11()
}

func opDup12(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	return nil, scope.Stack.dup12()
}

func opDup13(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	return nil, scope.Stack.dup13()
}

func opDup14(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	return nil, scope.Stack.dup14()
}

func opDup15(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	return nil, scope.Stack.dup15()
}

func opDup16(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	return nil, scope.Stack.dup16()
}
