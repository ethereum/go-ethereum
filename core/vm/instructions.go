// Copyright 2014 The go-ethereum Authors
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
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

type programInstruction interface {
	Do(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack)
}

type instrFn func(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack)

// Do executes the function. This implements programInstruction
func (fn instrFn) Do(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	fn(instr, pc, env, contract, memory, stack)
}

type instruction struct {
	op   OpCode
	pc   uint64
	fn   instrFn
	data *big.Int

	gas   *big.Int
	spop  int
	spush int
}

func opStaticJump(instr instruction, pc *uint64, ret *big.Int, env Environment, contract *Contract, memory *Memory, stack *stack) {
	ret.Set(instr.data)
}

func opAdd(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	x, y := stack.pop(), stack.pop()
	stack.push(U256(x.Add(x, y)))
}

func opSub(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	x, y := stack.pop(), stack.pop()
	stack.push(U256(x.Sub(x, y)))
}

func opMul(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	x, y := stack.pop(), stack.pop()
	stack.push(U256(x.Mul(x, y)))
}

func opDiv(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	x, y := stack.pop(), stack.pop()
	if y.Cmp(common.Big0) != 0 {
		stack.push(U256(x.Div(x, y)))
	} else {
		stack.push(new(big.Int))
	}
}

func opSdiv(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	x, y := S256(stack.pop()), S256(stack.pop())
	if y.Cmp(common.Big0) == 0 {
		stack.push(new(big.Int))
		return
	} else {
		n := new(big.Int)
		if new(big.Int).Mul(x, y).Cmp(common.Big0) < 0 {
			n.SetInt64(-1)
		} else {
			n.SetInt64(1)
		}

		res := x.Div(x.Abs(x), y.Abs(y))
		res.Mul(res, n)

		stack.push(U256(res))
	}
}

func opMod(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	x, y := stack.pop(), stack.pop()
	if y.Cmp(common.Big0) == 0 {
		stack.push(new(big.Int))
	} else {
		stack.push(U256(x.Mod(x, y)))
	}
}

func opSmod(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	x, y := S256(stack.pop()), S256(stack.pop())

	if y.Cmp(common.Big0) == 0 {
		stack.push(new(big.Int))
	} else {
		n := new(big.Int)
		if x.Cmp(common.Big0) < 0 {
			n.SetInt64(-1)
		} else {
			n.SetInt64(1)
		}

		res := x.Mod(x.Abs(x), y.Abs(y))
		res.Mul(res, n)

		stack.push(U256(res))
	}
}

func opExp(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	x, y := stack.pop(), stack.pop()
	stack.push(U256(x.Exp(x, y, Pow256)))
}

func opSignExtend(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	back := stack.pop()
	if back.Cmp(big.NewInt(31)) < 0 {
		bit := uint(back.Uint64()*8 + 7)
		num := stack.pop()
		mask := back.Lsh(common.Big1, bit)
		mask.Sub(mask, common.Big1)
		if common.BitTest(num, int(bit)) {
			num.Or(num, mask.Not(mask))
		} else {
			num.And(num, mask)
		}

		stack.push(U256(num))
	}
}

func opNot(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	x := stack.pop()
	stack.push(U256(x.Not(x)))
}

func opLt(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	x, y := stack.pop(), stack.pop()
	if x.Cmp(y) < 0 {
		stack.push(big.NewInt(1))
	} else {
		stack.push(new(big.Int))
	}
}

func opGt(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	x, y := stack.pop(), stack.pop()
	if x.Cmp(y) > 0 {
		stack.push(big.NewInt(1))
	} else {
		stack.push(new(big.Int))
	}
}

func opSlt(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	x, y := S256(stack.pop()), S256(stack.pop())
	if x.Cmp(S256(y)) < 0 {
		stack.push(big.NewInt(1))
	} else {
		stack.push(new(big.Int))
	}
}

func opSgt(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	x, y := S256(stack.pop()), S256(stack.pop())
	if x.Cmp(y) > 0 {
		stack.push(big.NewInt(1))
	} else {
		stack.push(new(big.Int))
	}
}

func opEq(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	x, y := stack.pop(), stack.pop()
	if x.Cmp(y) == 0 {
		stack.push(big.NewInt(1))
	} else {
		stack.push(new(big.Int))
	}
}

func opIszero(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	x := stack.pop()
	if x.Cmp(common.Big0) > 0 {
		stack.push(new(big.Int))
	} else {
		stack.push(big.NewInt(1))
	}
}

func opAnd(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	x, y := stack.pop(), stack.pop()
	stack.push(x.And(x, y))
}
func opOr(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	x, y := stack.pop(), stack.pop()
	stack.push(x.Or(x, y))
}
func opXor(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	x, y := stack.pop(), stack.pop()
	stack.push(x.Xor(x, y))
}
func opByte(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	th, val := stack.pop(), stack.pop()
	if th.Cmp(big.NewInt(32)) < 0 {
		byte := big.NewInt(int64(common.LeftPadBytes(val.Bytes(), 32)[th.Int64()]))
		stack.push(byte)
	} else {
		stack.push(new(big.Int))
	}
}
func opAddmod(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	x, y, z := stack.pop(), stack.pop(), stack.pop()
	if z.Cmp(Zero) > 0 {
		add := x.Add(x, y)
		add.Mod(add, z)
		stack.push(U256(add))
	} else {
		stack.push(new(big.Int))
	}
}
func opMulmod(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	x, y, z := stack.pop(), stack.pop(), stack.pop()
	if z.Cmp(Zero) > 0 {
		mul := x.Mul(x, y)
		mul.Mod(mul, z)
		stack.push(U256(mul))
	} else {
		stack.push(new(big.Int))
	}
}

func opSha3(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	offset, size := stack.pop(), stack.pop()
	hash := crypto.Sha3(memory.Get(offset.Int64(), size.Int64()))

	stack.push(common.BytesToBig(hash))
}

func opAddress(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	stack.push(common.Bytes2Big(contract.Address().Bytes()))
}

func opBalance(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	addr := common.BigToAddress(stack.pop())
	balance := env.Db().GetBalance(addr)

	stack.push(new(big.Int).Set(balance))
}

func opOrigin(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	stack.push(env.Origin().Big())
}

func opCaller(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	stack.push(common.Bytes2Big(contract.caller.Address().Bytes()))
}

func opCallValue(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	stack.push(new(big.Int).Set(contract.value))
}

func opCalldataLoad(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	stack.push(common.Bytes2Big(getData(contract.Input, stack.pop(), common.Big32)))
}

func opCalldataSize(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	stack.push(big.NewInt(int64(len(contract.Input))))
}

func opCalldataCopy(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	var (
		mOff = stack.pop()
		cOff = stack.pop()
		l    = stack.pop()
	)
	memory.Set(mOff.Uint64(), l.Uint64(), getData(contract.Input, cOff, l))
}

func opExtCodeSize(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	addr := common.BigToAddress(stack.pop())
	l := big.NewInt(int64(len(env.Db().GetCode(addr))))
	stack.push(l)
}

func opCodeSize(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	l := big.NewInt(int64(len(contract.Code)))
	stack.push(l)
}

func opCodeCopy(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	var (
		mOff = stack.pop()
		cOff = stack.pop()
		l    = stack.pop()
	)
	codeCopy := getData(contract.Code, cOff, l)

	memory.Set(mOff.Uint64(), l.Uint64(), codeCopy)
}

func opExtCodeCopy(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	var (
		addr = common.BigToAddress(stack.pop())
		mOff = stack.pop()
		cOff = stack.pop()
		l    = stack.pop()
	)
	codeCopy := getData(env.Db().GetCode(addr), cOff, l)

	memory.Set(mOff.Uint64(), l.Uint64(), codeCopy)
}

func opGasprice(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	stack.push(new(big.Int).Set(contract.Price))
}

func opBlockhash(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	num := stack.pop()

	n := new(big.Int).Sub(env.BlockNumber(), common.Big257)
	if num.Cmp(n) > 0 && num.Cmp(env.BlockNumber()) < 0 {
		stack.push(env.GetHash(num.Uint64()).Big())
	} else {
		stack.push(new(big.Int))
	}
}

func opCoinbase(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	stack.push(env.Coinbase().Big())
}

func opTimestamp(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	stack.push(U256(new(big.Int).Set(env.Time())))
}

func opNumber(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	stack.push(U256(new(big.Int).Set(env.BlockNumber())))
}

func opDifficulty(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	stack.push(U256(new(big.Int).Set(env.Difficulty())))
}

func opGasLimit(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	stack.push(U256(new(big.Int).Set(env.GasLimit())))
}

func opPop(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	stack.pop()
}

func opPush(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	stack.push(new(big.Int).Set(instr.data))
}

func opDup(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	stack.dup(int(instr.data.Int64()))
}

func opSwap(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	stack.swap(int(instr.data.Int64()))
}

func opLog(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	n := int(instr.data.Int64())
	topics := make([]common.Hash, n)
	mStart, mSize := stack.pop(), stack.pop()
	for i := 0; i < n; i++ {
		topics[i] = common.BigToHash(stack.pop())
	}

	d := memory.Get(mStart.Int64(), mSize.Int64())
	log := NewLog(contract.Address(), topics, d, env.BlockNumber().Uint64())
	env.AddLog(log)
}

func opMload(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	offset := stack.pop()
	val := common.BigD(memory.Get(offset.Int64(), 32))
	stack.push(val)
}

func opMstore(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	// pop value of the stack
	mStart, val := stack.pop(), stack.pop()
	memory.Set(mStart.Uint64(), 32, common.BigToBytes(val, 256))
}

func opMstore8(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	off, val := stack.pop().Int64(), stack.pop().Int64()
	memory.store[off] = byte(val & 0xff)
}

func opSload(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	loc := common.BigToHash(stack.pop())
	val := env.Db().GetState(contract.Address(), loc).Big()
	stack.push(val)
}

func opSstore(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	loc := common.BigToHash(stack.pop())
	val := stack.pop()

	env.Db().SetState(contract.Address(), loc, common.BigToHash(val))
}

func opJump(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
}
func opJumpi(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
}
func opJumpdest(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
}

func opPc(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	stack.push(new(big.Int).Set(instr.data))
}

func opMsize(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	stack.push(big.NewInt(int64(memory.Len())))
}

func opGas(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	stack.push(new(big.Int).Set(contract.Gas))
}

func opCreate(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	var (
		value        = stack.pop()
		offset, size = stack.pop(), stack.pop()
		input        = memory.Get(offset.Int64(), size.Int64())
		gas          = new(big.Int).Set(contract.Gas)
		addr         common.Address
		ret          []byte
		suberr       error
	)

	contract.UseGas(contract.Gas)
	ret, addr, suberr = env.Create(contract, input, gas, contract.Price, value)
	if suberr != nil {
		stack.push(new(big.Int))
	} else {
		// gas < len(ret) * Createinstr.dataGas == NO_CODE
		dataGas := big.NewInt(int64(len(ret)))
		dataGas.Mul(dataGas, params.CreateDataGas)
		if contract.UseGas(dataGas) {
			env.Db().SetCode(addr, ret)
		}

		stack.push(addr.Big())

	}
}

func opCall(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	gas := stack.pop()
	// pop gas and value of the stack.
	addr, value := stack.pop(), stack.pop()
	value = U256(value)
	// pop input size and offset
	inOffset, inSize := stack.pop(), stack.pop()
	// pop return size and offset
	retOffset, retSize := stack.pop(), stack.pop()

	address := common.BigToAddress(addr)

	// Get the arguments from the memory
	args := memory.Get(inOffset.Int64(), inSize.Int64())

	if len(value.Bytes()) > 0 {
		gas.Add(gas, params.CallStipend)
	}

	ret, err := env.Call(contract, address, args, gas, contract.Price, value)

	if err != nil {
		stack.push(new(big.Int))

	} else {
		stack.push(big.NewInt(1))

		memory.Set(retOffset.Uint64(), retSize.Uint64(), ret)
	}
}

func opCallCode(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	gas := stack.pop()
	// pop gas and value of the stack.
	addr, value := stack.pop(), stack.pop()
	value = U256(value)
	// pop input size and offset
	inOffset, inSize := stack.pop(), stack.pop()
	// pop return size and offset
	retOffset, retSize := stack.pop(), stack.pop()

	address := common.BigToAddress(addr)

	// Get the arguments from the memory
	args := memory.Get(inOffset.Int64(), inSize.Int64())

	if len(value.Bytes()) > 0 {
		gas.Add(gas, params.CallStipend)
	}

	ret, err := env.CallCode(contract, address, args, gas, contract.Price, value)

	if err != nil {
		stack.push(new(big.Int))

	} else {
		stack.push(big.NewInt(1))

		memory.Set(retOffset.Uint64(), retSize.Uint64(), ret)
	}
}

func opReturn(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
}
func opStop(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
}

func opSuicide(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
	//receiver := env.Db().GetOrNewStateObject(common.BigToAddress(stack.pop()))
	//receiver.AddBalance(balance)
	balance := env.Db().GetBalance(contract.Address())
	env.Db().AddBalance(common.BigToAddress(stack.pop()), balance)

	env.Db().Delete(contract.Address())
}

// following functions are used by the instruction jump  table

// make log instruction function
func makeLog(size int) instrFn {
	return func(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
		topics := make([]common.Hash, size)
		mStart, mSize := stack.pop(), stack.pop()
		for i := 0; i < size; i++ {
			topics[i] = common.BigToHash(stack.pop())
		}

		d := memory.Get(mStart.Int64(), mSize.Int64())
		log := NewLog(contract.Address(), topics, d, env.BlockNumber().Uint64())
		env.AddLog(log)
	}
}

// make push instruction function
func makePush(size uint64, bsize *big.Int) instrFn {
	return func(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
		byts := getData(contract.Code, new(big.Int).SetUint64(*pc+1), bsize)
		stack.push(common.Bytes2Big(byts))
		*pc += size
	}
}

// make push instruction function
func makeDup(size int64) instrFn {
	return func(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
		stack.dup(int(size))
	}
}

// make swap instruction function
func makeSwap(size int64) instrFn {
	// switch n + 1 otherwise n would be swapped with n
	size += 1
	return func(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) {
		stack.swap(int(size))
	}
}
