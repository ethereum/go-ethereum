// Copyright 2019 The go-ethereum Authors
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

// EnableEIP enables the given EIP on the config.
// This operation writes in-place, and callers need to ensure that the globally
// defined jump tables are not polluted.
func EnableEIP(eipNum int, jt *JumpTable) error {
	switch eipNum {
	case 2200:
		enable2200(jt)
	case 1884:
		enable1884(jt)
	case 1344:
		enable1344(jt)
	case 2315:
		enable2315(jt)
	default:
		return fmt.Errorf("undefined eip %d", eipNum)
	}
	return nil
}

// enable1884 applies EIP-1884 to the given jump table:
// - Increase cost of BALANCE to 700
// - Increase cost of EXTCODEHASH to 700
// - Increase cost of SLOAD to 800
// - Define SELFBALANCE, with cost GasFastStep (5)
func enable1884(jt *JumpTable) {
	// Gas cost changes
	jt[SLOAD].constantGas = params.SloadGasEIP1884
	jt[BALANCE].constantGas = params.BalanceGasEIP1884
	jt[EXTCODEHASH].constantGas = params.ExtcodeHashGasEIP1884

	// New opcode
	jt[SELFBALANCE] = operation{
		execute:     opSelfBalance,
		constantGas: GasFastStep,
		minStack:    minStack(0, 1),
		maxStack:    maxStack(0, 1),
		valid:       true,
	}
}

func opSelfBalance(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	balance := interpreter.intPool.get().Set(interpreter.evm.StateDB.GetBalance(callContext.contract.Address()))
	callContext.stack.push(balance)
	return nil, nil
}

// enable1344 applies EIP-1344 (ChainID Opcode)
// - Adds an opcode that returns the current chain’s EIP-155 unique identifier
func enable1344(jt *JumpTable) {
	// New opcode
	jt[CHAINID] = operation{
		execute:     opChainID,
		constantGas: GasQuickStep,
		minStack:    minStack(0, 1),
		maxStack:    maxStack(0, 1),
		valid:       true,
	}
}

// opChainID implements CHAINID opcode
func opChainID(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	chainId := interpreter.intPool.get().Set(interpreter.evm.chainConfig.ChainID)
	callContext.stack.push(chainId)
	return nil, nil
}

// enable2200 applies EIP-2200 (Rebalance net-metered SSTORE)
func enable2200(jt *JumpTable) {
	jt[SLOAD].constantGas = params.SloadGasEIP2200
	jt[SSTORE].dynamicGas = gasSStoreEIP2200
}

// enable2315 applies EIP-2315 (Simple Subroutines)
// - Adds opcodes that jump to and return from subroutines
func enable2315(jt *JumpTable) {
	// New opcode
	jt[BEGINSUB] = operation{
		execute:     opBeginSub,
		constantGas: GasQuickStep,
		minStack:    minStack(0, 0),
		maxStack:    maxStack(0, 0),
		valid:       true,
	}
	// New opcode
	jt[JUMPSUB] = operation{
		execute:     opJumpSub,
		constantGas: GasSlowStep,
		minStack:    minStack(1, 0),
		maxStack:    maxStack(1, 0),
		jumps:       true,
		valid:       true,
	}
	// New opcode
	jt[RETURNSUB] = operation{
		execute:     opReturnSub,
		constantGas: GasFastStep,
		minStack:    minStack(0, 0),
		maxStack:    maxStack(0, 0),
		valid:       true,
		jumps:       true,
	}
	// redefine opcode
	jt[JUMP] = operation{
		execute:     opJumpEip2315,
		constantGas: GasMidStep,
		minStack:    minStack(1, 0),
		maxStack:    maxStack(1, 0),
		jumps:       true,
		valid:       true,
	}
	jt[JUMPI] = operation{
		execute:     opJumpiEip2315,
		constantGas: GasSlowStep,
		minStack:    minStack(2, 0),
		maxStack:    maxStack(2, 0),
		jumps:       true,
		valid:       true,
	}

}

// opJumpEip2315 implements JUMP when restricted subroutines are active
func opJumpEip2315(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	pos := callContext.stack.pop()
	if !callContext.contract.validJumpdest(pos) {
		return nil, ErrInvalidJump
	}
	// A this point, we know that
	// 1. The destination is _code_
	// 2. The destination is JUMPDEST
	// Remains to find out if destination is within the same subroutine
	cur := callContext.rstack.currentSubroutine()
	dest := pos.Uint64()
	if !callContext.contract.isSameSubroutine(cur, dest) {
		return nil, ErrJumpAcrossRoutine
	}
	*pc = dest

	interpreter.intPool.putOne(pos)
	return nil, nil
}

// opJumpiEip2315 implements JUMPI when restricted subroutines are active
func opJumpiEip2315(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	pos, cond := callContext.stack.pop(), callContext.stack.pop()
	if cond.Sign() != 0 {
		if !callContext.contract.validJumpdest(pos) {
			return nil, ErrInvalidJump
		}
		cur := callContext.rstack.currentSubroutine()
		dest := pos.Uint64()
		if !callContext.contract.isSameSubroutine(cur, dest) {
			return nil, ErrJumpAcrossRoutine
		}
		*pc = dest
	} else {
		*pc++
	}

	interpreter.intPool.put(pos, cond)
	return nil, nil
}
