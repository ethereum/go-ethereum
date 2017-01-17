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
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

func opAdd(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x, y := stack.pop(), stack.pop()
	stack.push(U256(x.Add(x, y)))
	return nil, nil
}

func opSub(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x, y := stack.pop(), stack.pop()
	stack.push(U256(x.Sub(x, y)))
	return nil, nil
}

func opMul(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x, y := stack.pop(), stack.pop()
	stack.push(U256(x.Mul(x, y)))
	return nil, nil
}

func opDiv(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x, y := stack.pop(), stack.pop()
	if y.Cmp(common.Big0) != 0 {
		stack.push(U256(x.Div(x, y)))
	} else {
		stack.push(new(big.Int))
	}
	return nil, nil
}

func opSdiv(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x, y := S256(stack.pop()), S256(stack.pop())
	if y.Cmp(common.Big0) == 0 {
		stack.push(new(big.Int))
		return nil, nil
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
	return nil, nil
}

func opMod(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x, y := stack.pop(), stack.pop()
	if y.Cmp(common.Big0) == 0 {
		stack.push(new(big.Int))
	} else {
		stack.push(U256(x.Mod(x, y)))
	}
	return nil, nil
}

func opSmod(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
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
	return nil, nil
}

func opExp(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	base, exponent := stack.pop(), stack.pop()
	stack.push(math.Exp(base, exponent))
	return nil, nil
}

func opSignExtend(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
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
	return nil, nil
}

func opNot(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x := stack.pop()
	stack.push(U256(x.Not(x)))
	return nil, nil
}

func opLt(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x, y := stack.pop(), stack.pop()
	if x.Cmp(y) < 0 {
		stack.push(big.NewInt(1))
	} else {
		stack.push(new(big.Int))
	}
	return nil, nil
}

func opGt(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x, y := stack.pop(), stack.pop()
	if x.Cmp(y) > 0 {
		stack.push(big.NewInt(1))
	} else {
		stack.push(new(big.Int))
	}
	return nil, nil
}

func opSlt(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x, y := S256(stack.pop()), S256(stack.pop())
	if x.Cmp(S256(y)) < 0 {
		stack.push(big.NewInt(1))
	} else {
		stack.push(new(big.Int))
	}
	return nil, nil
}

func opSgt(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x, y := S256(stack.pop()), S256(stack.pop())
	if x.Cmp(y) > 0 {
		stack.push(big.NewInt(1))
	} else {
		stack.push(new(big.Int))
	}
	return nil, nil
}

func opEq(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x, y := stack.pop(), stack.pop()
	if x.Cmp(y) == 0 {
		stack.push(big.NewInt(1))
	} else {
		stack.push(new(big.Int))
	}
	return nil, nil
}

func opIszero(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x := stack.pop()
	if x.Cmp(common.Big0) > 0 {
		stack.push(new(big.Int))
	} else {
		stack.push(big.NewInt(1))
	}
	return nil, nil
}

func opAnd(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x, y := stack.pop(), stack.pop()
	stack.push(x.And(x, y))
	return nil, nil
}
func opOr(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x, y := stack.pop(), stack.pop()
	stack.push(x.Or(x, y))
	return nil, nil
}
func opXor(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x, y := stack.pop(), stack.pop()
	stack.push(x.Xor(x, y))
	return nil, nil
}
func opByte(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	th, val := stack.pop(), stack.pop()
	if th.Cmp(big.NewInt(32)) < 0 {
		byte := big.NewInt(int64(common.LeftPadBytes(val.Bytes(), 32)[th.Int64()]))
		stack.push(byte)
	} else {
		stack.push(new(big.Int))
	}
	return nil, nil
}
func opAddmod(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x, y, z := stack.pop(), stack.pop(), stack.pop()
	if z.Cmp(Zero) > 0 {
		add := x.Add(x, y)
		add.Mod(add, z)
		stack.push(U256(add))
	} else {
		stack.push(new(big.Int))
	}
	return nil, nil
}
func opMulmod(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x, y, z := stack.pop(), stack.pop(), stack.pop()
	if z.Cmp(Zero) > 0 {
		mul := x.Mul(x, y)
		mul.Mod(mul, z)
		stack.push(U256(mul))
	} else {
		stack.push(new(big.Int))
	}
	return nil, nil
}

func opSha3(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	offset, size := stack.pop(), stack.pop()
	data := memory.Get(offset.Int64(), size.Int64())
	hash := crypto.Keccak256(data)

	if env.vmConfig.EnablePreimageRecording {
		env.StateDB.AddPreimage(common.BytesToHash(hash), data)
	}

	stack.push(common.BytesToBig(hash))
	return nil, nil
}

func opAddress(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	stack.push(common.Bytes2Big(contract.Address().Bytes()))
	return nil, nil
}

func opBalance(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	addr := common.BigToAddress(stack.pop())
	balance := env.StateDB.GetBalance(addr)

	stack.push(new(big.Int).Set(balance))
	return nil, nil
}

func opOrigin(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	stack.push(env.Origin.Big())
	return nil, nil
}

func opCaller(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	stack.push(contract.Caller().Big())
	return nil, nil
}

func opCallValue(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	stack.push(new(big.Int).Set(contract.value))
	return nil, nil
}

func opCalldataLoad(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	stack.push(common.Bytes2Big(getData(contract.Input, stack.pop(), common.Big32)))
	return nil, nil
}

func opCalldataSize(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	stack.push(big.NewInt(int64(len(contract.Input))))
	return nil, nil
}

func opCalldataCopy(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	var (
		mOff = stack.pop()
		cOff = stack.pop()
		l    = stack.pop()
	)
	memory.Set(mOff.Uint64(), l.Uint64(), getData(contract.Input, cOff, l))
	return nil, nil
}

func opExtCodeSize(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	addr := common.BigToAddress(stack.pop())
	l := big.NewInt(int64(env.StateDB.GetCodeSize(addr)))
	stack.push(l)
	return nil, nil
}

func opCodeSize(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	l := big.NewInt(int64(len(contract.Code)))
	stack.push(l)
	return nil, nil
}

func opCodeCopy(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	var (
		mOff = stack.pop()
		cOff = stack.pop()
		l    = stack.pop()
	)
	codeCopy := getData(contract.Code, cOff, l)

	memory.Set(mOff.Uint64(), l.Uint64(), codeCopy)
	return nil, nil
}

func opExtCodeCopy(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	var (
		addr = common.BigToAddress(stack.pop())
		mOff = stack.pop()
		cOff = stack.pop()
		l    = stack.pop()
	)
	codeCopy := getData(env.StateDB.GetCode(addr), cOff, l)

	memory.Set(mOff.Uint64(), l.Uint64(), codeCopy)
	return nil, nil
}

func opGasprice(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	stack.push(new(big.Int).Set(env.GasPrice))
	return nil, nil
}

func opBlockhash(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	num := stack.pop()

	n := new(big.Int).Sub(env.BlockNumber, common.Big257)
	if num.Cmp(n) > 0 && num.Cmp(env.BlockNumber) < 0 {
		stack.push(env.GetHash(num.Uint64()).Big())
	} else {
		stack.push(new(big.Int))
	}
	return nil, nil
}

func opCoinbase(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	stack.push(env.Coinbase.Big())
	return nil, nil
}

func opTimestamp(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	stack.push(U256(new(big.Int).Set(env.Time)))
	return nil, nil
}

func opNumber(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	stack.push(U256(new(big.Int).Set(env.BlockNumber)))
	return nil, nil
}

func opDifficulty(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	stack.push(U256(new(big.Int).Set(env.Difficulty)))
	return nil, nil
}

func opGasLimit(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	stack.push(U256(new(big.Int).Set(env.GasLimit)))
	return nil, nil
}

func opPop(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	stack.pop()
	return nil, nil
}

func opMload(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	offset := stack.pop()
	val := common.BigD(memory.Get(offset.Int64(), 32))
	stack.push(val)
	return nil, nil
}

func opMstore(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	// pop value of the stack
	mStart, val := stack.pop(), stack.pop()
	memory.Set(mStart.Uint64(), 32, common.BigToBytes(val, 256))
	return nil, nil
}

func opMstore8(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	off, val := stack.pop().Int64(), stack.pop().Int64()
	memory.store[off] = byte(val & 0xff)
	return nil, nil
}

func opSload(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	loc := common.BigToHash(stack.pop())
	val := env.StateDB.GetState(contract.Address(), loc).Big()
	stack.push(val)
	return nil, nil
}

func opSstore(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	loc := common.BigToHash(stack.pop())
	val := stack.pop()
	env.StateDB.SetState(contract.Address(), loc, common.BigToHash(val))
	return nil, nil
}

func opJump(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	pos := stack.pop()
	if !contract.jumpdests.has(contract.CodeHash, contract.Code, pos) {
		nop := contract.GetOp(pos.Uint64())
		return nil, fmt.Errorf("invalid jump destination (%v) %v", nop, pos)
	}
	*pc = pos.Uint64()
	return nil, nil
}
func opJumpi(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	pos, cond := stack.pop(), stack.pop()
	if cond.Cmp(common.BigTrue) >= 0 {
		if !contract.jumpdests.has(contract.CodeHash, contract.Code, pos) {
			nop := contract.GetOp(pos.Uint64())
			return nil, fmt.Errorf("invalid jump destination (%v) %v", nop, pos)
		}
		*pc = pos.Uint64()
	} else {
		*pc++
	}
	return nil, nil
}
func opJumpdest(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	return nil, nil
}

func opPc(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	stack.push(new(big.Int).SetUint64(*pc))
	return nil, nil
}

func opMsize(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	stack.push(big.NewInt(int64(memory.Len())))
	return nil, nil
}

func opGas(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	stack.push(new(big.Int).Set(contract.Gas))
	return nil, nil
}

func opCreate(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	var (
		value        = stack.pop()
		offset, size = stack.pop(), stack.pop()
		input        = memory.Get(offset.Int64(), size.Int64())
		gas          = new(big.Int).Set(contract.Gas)
	)
	if env.ChainConfig().IsEIP150(env.BlockNumber) {
		gas.Div(gas, n64)
		gas = gas.Sub(contract.Gas, gas)
	}

	contract.UseGas(gas)
	_, addr, suberr := env.Create(contract, input, gas, value)
	// Push item on the stack based on the returned error. If the ruleset is
	// homestead we must check for CodeStoreOutOfGasError (homestead only
	// rule) and treat as an error, if the ruleset is frontier we must
	// ignore this error and pretend the operation was successful.
	if env.ChainConfig().IsHomestead(env.BlockNumber) && suberr == ErrCodeStoreOutOfGas {
		stack.push(new(big.Int))
	} else if suberr != nil && suberr != ErrCodeStoreOutOfGas {
		stack.push(new(big.Int))
	} else {
		stack.push(addr.Big())
	}
	return nil, nil
}

func opCall(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
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

	ret, err := env.Call(contract, address, args, gas, value)

	if err != nil {
		stack.push(new(big.Int))

	} else {
		stack.push(big.NewInt(1))

		memory.Set(retOffset.Uint64(), retSize.Uint64(), ret)
	}
	return nil, nil
}

func opCallCode(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
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

	ret, err := env.CallCode(contract, address, args, gas, value)

	if err != nil {
		stack.push(new(big.Int))

	} else {
		stack.push(big.NewInt(1))

		memory.Set(retOffset.Uint64(), retSize.Uint64(), ret)
	}
	return nil, nil
}

func opDelegateCall(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	// if not homestead return an error. DELEGATECALL is not supported
	// during pre-homestead.
	if !env.ChainConfig().IsHomestead(env.BlockNumber) {
		return nil, fmt.Errorf("invalid opcode %x", DELEGATECALL)
	}

	gas, to, inOffset, inSize, outOffset, outSize := stack.pop(), stack.pop(), stack.pop(), stack.pop(), stack.pop(), stack.pop()

	toAddr := common.BigToAddress(to)
	args := memory.Get(inOffset.Int64(), inSize.Int64())
	ret, err := env.DelegateCall(contract, toAddr, args, gas)
	if err != nil {
		stack.push(new(big.Int))
	} else {
		stack.push(big.NewInt(1))
		memory.Set(outOffset.Uint64(), outSize.Uint64(), ret)
	}
	return nil, nil
}

func opReturn(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	offset, size := stack.pop(), stack.pop()
	ret := memory.GetPtr(offset.Int64(), size.Int64())

	return ret, nil
}

func opStop(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	return nil, nil
}

func opSuicide(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	balance := env.StateDB.GetBalance(contract.Address())
	env.StateDB.AddBalance(common.BigToAddress(stack.pop()), balance)

	env.StateDB.Suicide(contract.Address())

	return nil, nil
}

// following functions are used by the instruction jump  table

// make log instruction function
func makeLog(size int) executionFunc {
	return func(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
		topics := make([]common.Hash, size)
		mStart, mSize := stack.pop(), stack.pop()
		for i := 0; i < size; i++ {
			topics[i] = common.BigToHash(stack.pop())
		}

		d := memory.Get(mStart.Int64(), mSize.Int64())
		env.StateDB.AddLog(&types.Log{
			Address: contract.Address(),
			Topics:  topics,
			Data:    d,
			// This is a non-consensus field, but assigned here because
			// core/state doesn't know the current block number.
			BlockNumber: env.BlockNumber.Uint64(),
		})
		return nil, nil
	}
}

// make push instruction function
func makePush(size uint64, bsize *big.Int) executionFunc {
	return func(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
		byts := getData(contract.Code, new(big.Int).SetUint64(*pc+1), bsize)
		stack.push(common.Bytes2Big(byts))
		*pc += size
		return nil, nil
	}
}

// make push instruction function
func makeDup(size int64) executionFunc {
	return func(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
		stack.dup(int(size))
		return nil, nil
	}
}

// make swap instruction function
func makeSwap(size int64) executionFunc {
	// switch n + 1 otherwise n would be swapped with n
	size += 1
	return func(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
		stack.swap(int(size))
		return nil, nil
	}
}
