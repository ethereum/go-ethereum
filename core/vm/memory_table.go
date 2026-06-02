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

func memoryKeccak256(stack *Stack) (uint64, bool) {
	return calcMemSize64(stack.back(0), stack.back(1))
}

func memoryCallDataCopy(stack *Stack) (uint64, bool) {
	return calcMemSize64(stack.back(0), stack.back(2))
}

func memoryReturnDataCopy(stack *Stack) (uint64, bool) {
	return calcMemSize64(stack.back(0), stack.back(2))
}

func memoryCodeCopy(stack *Stack) (uint64, bool) {
	return calcMemSize64(stack.back(0), stack.back(2))
}

func memoryExtCodeCopy(stack *Stack) (uint64, bool) {
	return calcMemSize64(stack.back(1), stack.back(3))
}

func memoryMLoad(stack *Stack) (uint64, bool) {
	return calcMemSize64WithUint(stack.back(0), 32)
}

func memoryMStore8(stack *Stack) (uint64, bool) {
	return calcMemSize64WithUint(stack.back(0), 1)
}

func memoryMStore(stack *Stack) (uint64, bool) {
	return calcMemSize64WithUint(stack.back(0), 32)
}

func memoryMcopy(stack *Stack) (uint64, bool) {
	mStart := stack.back(0) // stack[0]: dest
	if stack.back(1).Gt(mStart) {
		mStart = stack.back(1) // stack[1]: source
	}
	return calcMemSize64(mStart, stack.back(2)) // stack[2]: length
}

func memoryCreate(stack *Stack) (uint64, bool) {
	return calcMemSize64(stack.back(1), stack.back(2))
}

func memoryCreate2(stack *Stack) (uint64, bool) {
	return calcMemSize64(stack.back(1), stack.back(2))
}

func memoryCall(stack *Stack) (uint64, bool) {
	x, overflow := calcMemSize64(stack.back(5), stack.back(6))
	if overflow {
		return 0, true
	}
	y, overflow := calcMemSize64(stack.back(3), stack.back(4))
	if overflow {
		return 0, true
	}
	if x > y {
		return x, false
	}
	return y, false
}

func memoryDelegateCall(stack *Stack) (uint64, bool) {
	x, overflow := calcMemSize64(stack.back(4), stack.back(5))
	if overflow {
		return 0, true
	}
	y, overflow := calcMemSize64(stack.back(2), stack.back(3))
	if overflow {
		return 0, true
	}
	if x > y {
		return x, false
	}
	return y, false
}

func memoryStaticCall(stack *Stack) (uint64, bool) {
	x, overflow := calcMemSize64(stack.back(4), stack.back(5))
	if overflow {
		return 0, true
	}
	y, overflow := calcMemSize64(stack.back(2), stack.back(3))
	if overflow {
		return 0, true
	}
	if x > y {
		return x, false
	}
	return y, false
}

func memoryReturn(stack *Stack) (uint64, bool) {
	return calcMemSize64(stack.back(0), stack.back(1))
}

func memoryRevert(stack *Stack) (uint64, bool) {
	return calcMemSize64(stack.back(0), stack.back(1))
}

func memoryLog(stack *Stack) (uint64, bool) {
	return calcMemSize64(stack.back(0), stack.back(1))
}
