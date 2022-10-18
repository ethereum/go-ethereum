// Copyright 2017 The go-ethereum Authors
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

func memoryKeccak256(stack *Stack, scope *ScopeContext) (error, uint64, bool) {
	return calcMemSize64(stack.Back(0), stack.Back(1))
}

func memoryCallDataCopy(stack *Stack, scope *ScopeContext) (error, uint64, bool) {
	return calcMemSize64(stack.Back(0), stack.Back(2))
}

func memoryReturnDataCopy(stack *Stack, scope *ScopeContext) (error, uint64, bool) {
	return calcMemSize64(stack.Back(0), stack.Back(2))
}

func memoryCodeCopy(stack *Stack, scope *ScopeContext) (error, uint64, bool) {
	return calcMemSize64(stack.Back(0), stack.Back(2))
}

func memoryExtCodeCopy(stack *Stack, scope *ScopeContext) (error, uint64, bool) {
	return calcMemSize64(stack.Back(1), stack.Back(3))
}

func memoryMLoad(stack *Stack, scope *ScopeContext) (error, uint64, bool) {
	return calcMemSize64WithUint(stack.Back(0), 32)
}

func memoryMStore8(stack *Stack, scope *ScopeContext) (error, uint64, bool) {
	return calcMemSize64WithUint(stack.Back(0), 1)
}

func memoryMStore(stack *Stack, scope *ScopeContext) (error, uint64, bool) {
	return calcMemSize64WithUint(stack.Back(0), 32)
}

func memoryCreate(stack *Stack, scope *ScopeContext) (error, uint64, bool) {
	return calcMemSize64(stack.Back(1), stack.Back(2))
}

func memoryCreate2(stack *Stack, scope *ScopeContext) (error, uint64, bool) {
	return calcMemSize64(stack.Back(1), stack.Back(2))
}

func memoryCall(stack *Stack, scope *ScopeContext) (error, uint64, bool) {
	_, x, overflow := calcMemSize64(stack.Back(5), stack.Back(6))
	if overflow {
		return nil, 0, true
	}
	_, y, overflow := calcMemSize64(stack.Back(3), stack.Back(4))
	if overflow {
		return nil, 0, true
	}
	if x > y {
		return nil, x, false
	}
	return nil, y, false
}
func memoryDelegateCall(stack *Stack, scope *ScopeContext) (error, uint64, bool) {
	_, x, overflow := calcMemSize64(stack.Back(4), stack.Back(5))
	if overflow {
		return nil, 0, true
	}
	_, y, overflow := calcMemSize64(stack.Back(2), stack.Back(3))
	if overflow {
		return nil, 0, true
	}
	if x > y {
		return nil, x, false
	}
	return nil, y, false
}

func memoryStaticCall(stack *Stack, scope *ScopeContext) (error, uint64, bool) {
	_, x, overflow := calcMemSize64(stack.Back(4), stack.Back(5))
	if overflow {
		return nil, 0, true
	}
	_, y, overflow := calcMemSize64(stack.Back(2), stack.Back(3))
	if overflow {
		return nil, 0, true
	}
	if x > y {
		return nil, x, false
	}
	return nil, y, false
}

func memoryReturn(stack *Stack, scope *ScopeContext) (error, uint64, bool) {
	return calcMemSize64(stack.Back(0), stack.Back(1))
}

func memoryRevert(stack *Stack, scope *ScopeContext) (error, uint64, bool) {
	return calcMemSize64(stack.Back(0), stack.Back(1))
}

func memoryLog(stack *Stack, scope *ScopeContext) (error, uint64, bool) {
	return calcMemSize64(stack.Back(0), stack.Back(1))
}

/*
func memoryEVMMAXArith(stack *Stack, scope *ScopeContext) (error, uint64, bool) {
	if scope.EVMMAXField == nil {
		return ErrOutOfGas, 0, false
	}
	elemSize := uint64(scope.EVMMAXField.NumLimbs) * 8

	out_offset := byte(params_offsets[0] >> 16)
	x_offset := byte(params_offsets[0] >> 8)
	y_offset := byte(params_offsets[0])
	max_offset := uint64(max(max(out_offset, x_offset), y_offset)) * elemSize

	return nil, max_offset + elemSize, false
}

func memoryToMontX(stack *Stack, scope *ScopeContext) (error, uint64, bool) {
	if scope.EVMMAXField == nil {
		return ErrOutOfGas, 0, false
	}
	params_offsets := scope.Stack.peek()
	elemSize := uint64(scope.EVMMAXField.NumLimbs) * 8

	out_offset := byte(params_offsets[0] >> 16)
	input_offset := byte(params_offsets[0] >> 8)
	max_offset := uint64(max(out_offset, input_offset)) * elemSize

	return nil, max_offset + elemSize, false
}
*/
