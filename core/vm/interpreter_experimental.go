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
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// runExperimental is an optimized interpreter loop using modern optimization
// techniques taken from reth and gevm.
// It is used when tracing is disabled and Verkle (EIP-4762) is not active, since
// the code does not deal with those cases yet.
//
// Currently there are two optimizations over the standard loop:
//
//  1. Switch dispatch: the standard interpreter calls operation.execute() via a function
//     pointer table. This is an indirect call that the CPU can't predict and the compiler won't
//     inline.
//     runExperimental replaces this with a switch statement for common
//     opcodes, with the opcode logic inlined in each case. The remaining opcodes
//     fall through to the default case which uses the standard jump table.
//     TODO: Eventualy, we can migrate over everything.
//
//  2. Gas accumulation: the standard interpreter checks and writes contract.Gas (on the heap)
//     on every opcode.
//     runExperimental instead accumulates gas in a local (register allocated) variable
//     It flushes to contract.Gas only at:
//     - Control flow boundaries (JUMP, JUMPI etc)
//     - Halt points (STOP, end of code)
//     - Error paths (stack underflow/overflow, OOG)
//     - The default fallback (before dynamic-gas opcodes like SLOAD, CALL, etc.)
//
//     This is safe because:
//     - all inlined opcodes only touch the stack, so executing a few extra ops
//     before detecting OOG has no observable side effects.
//     - the EVM spec consumes all gas on OOG errors regardless.
//     - gas is always flushed before any opcode with external effects (storage, calls, logs)
//     since those go through the default path.
//     - JUMP/JUMPI flush before jumping, so loops can't run indefinitely without a gas check.
func (evm *EVM) runExperimental(contract *Contract, stack *Stack, mem *Memory, callContext *ScopeContext, jumpTable *JumpTable, code []byte) (ret []byte, err error) {
	var (
		pc      uint64
		codeLen = uint64(len(code))
		gasUsed uint64 // accumulated gas for constant-gas opcodes
	)

	for {
		if pc >= codeLen {
			if contract.Gas < gasUsed {
				return nil, ErrOutOfGas
			}
			contract.Gas -= gasUsed
			return nil, errStopToken
		}
		op := OpCode(code[pc])

		switch op {

		case STOP:
			if contract.Gas < gasUsed {
				return nil, ErrOutOfGas
			}
			contract.Gas -= gasUsed
			return nil, errStopToken

		case ADD:
			gasUsed += GasFastestStep
			if stack.len() < 2 {
				if contract.Gas < gasUsed {
					return nil, ErrOutOfGas
				}
				contract.Gas -= gasUsed
				return nil, &ErrStackUnderflow{stackLen: stack.len(), required: 2}
			}
			x, y := stack.pop(), stack.peek()
			y.Add(&x, y)

		case SUB:
			gasUsed += GasFastestStep
			if stack.len() < 2 {
				if contract.Gas < gasUsed {
					return nil, ErrOutOfGas
				}
				contract.Gas -= gasUsed
				return nil, &ErrStackUnderflow{stackLen: stack.len(), required: 2}
			}
			x, y := stack.pop(), stack.peek()
			y.Sub(&x, y)

		case MUL:
			gasUsed += GasFastStep
			if stack.len() < 2 {
				if contract.Gas < gasUsed {
					return nil, ErrOutOfGas
				}
				contract.Gas -= gasUsed
				return nil, &ErrStackUnderflow{stackLen: stack.len(), required: 2}
			}
			x, y := stack.pop(), stack.peek()
			y.Mul(&x, y)

		case DIV:
			gasUsed += GasFastStep
			if stack.len() < 2 {
				if contract.Gas < gasUsed {
					return nil, ErrOutOfGas
				}
				contract.Gas -= gasUsed
				return nil, &ErrStackUnderflow{stackLen: stack.len(), required: 2}
			}
			x, y := stack.pop(), stack.peek()
			y.Div(&x, y)

		case SDIV:
			gasUsed += GasFastStep
			if stack.len() < 2 {
				if contract.Gas < gasUsed {
					return nil, ErrOutOfGas
				}
				contract.Gas -= gasUsed
				return nil, &ErrStackUnderflow{stackLen: stack.len(), required: 2}
			}
			x, y := stack.pop(), stack.peek()
			y.SDiv(&x, y)

		case MOD:
			gasUsed += GasFastStep
			if stack.len() < 2 {
				if contract.Gas < gasUsed {
					return nil, ErrOutOfGas
				}
				contract.Gas -= gasUsed
				return nil, &ErrStackUnderflow{stackLen: stack.len(), required: 2}
			}
			x, y := stack.pop(), stack.peek()
			y.Mod(&x, y)

		case SMOD:
			gasUsed += GasFastStep
			if stack.len() < 2 {
				if contract.Gas < gasUsed {
					return nil, ErrOutOfGas
				}
				contract.Gas -= gasUsed
				return nil, &ErrStackUnderflow{stackLen: stack.len(), required: 2}
			}
			x, y := stack.pop(), stack.peek()
			y.SMod(&x, y)

		case LT:
			gasUsed += GasFastestStep
			if stack.len() < 2 {
				if contract.Gas < gasUsed {
					return nil, ErrOutOfGas
				}
				contract.Gas -= gasUsed
				return nil, &ErrStackUnderflow{stackLen: stack.len(), required: 2}
			}
			x, y := stack.pop(), stack.peek()
			if x.Lt(y) {
				y.SetOne()
			} else {
				y.Clear()
			}

		case GT:
			gasUsed += GasFastestStep
			if stack.len() < 2 {
				if contract.Gas < gasUsed {
					return nil, ErrOutOfGas
				}
				contract.Gas -= gasUsed
				return nil, &ErrStackUnderflow{stackLen: stack.len(), required: 2}
			}
			x, y := stack.pop(), stack.peek()
			if x.Gt(y) {
				y.SetOne()
			} else {
				y.Clear()
			}

		case SLT:
			gasUsed += GasFastestStep
			if stack.len() < 2 {
				if contract.Gas < gasUsed {
					return nil, ErrOutOfGas
				}
				contract.Gas -= gasUsed
				return nil, &ErrStackUnderflow{stackLen: stack.len(), required: 2}
			}
			x, y := stack.pop(), stack.peek()
			if x.Slt(y) {
				y.SetOne()
			} else {
				y.Clear()
			}

		case SGT:
			gasUsed += GasFastestStep
			if stack.len() < 2 {
				if contract.Gas < gasUsed {
					return nil, ErrOutOfGas
				}
				contract.Gas -= gasUsed
				return nil, &ErrStackUnderflow{stackLen: stack.len(), required: 2}
			}
			x, y := stack.pop(), stack.peek()
			if x.Sgt(y) {
				y.SetOne()
			} else {
				y.Clear()
			}

		case EQ:
			gasUsed += GasFastestStep
			if stack.len() < 2 {
				if contract.Gas < gasUsed {
					return nil, ErrOutOfGas
				}
				contract.Gas -= gasUsed
				return nil, &ErrStackUnderflow{stackLen: stack.len(), required: 2}
			}
			x, y := stack.pop(), stack.peek()
			if x.Eq(y) {
				y.SetOne()
			} else {
				y.Clear()
			}

		case ISZERO:
			gasUsed += GasFastestStep
			if stack.len() < 1 {
				if contract.Gas < gasUsed {
					return nil, ErrOutOfGas
				}
				contract.Gas -= gasUsed
				return nil, &ErrStackUnderflow{stackLen: stack.len(), required: 1}
			}
			x := stack.peek()
			if x.IsZero() {
				x.SetOne()
			} else {
				x.Clear()
			}

		case AND:
			gasUsed += GasFastestStep
			if stack.len() < 2 {
				if contract.Gas < gasUsed {
					return nil, ErrOutOfGas
				}
				contract.Gas -= gasUsed
				return nil, &ErrStackUnderflow{stackLen: stack.len(), required: 2}
			}
			x, y := stack.pop(), stack.peek()
			y.And(&x, y)

		case OR:
			gasUsed += GasFastestStep
			if stack.len() < 2 {
				if contract.Gas < gasUsed {
					return nil, ErrOutOfGas
				}
				contract.Gas -= gasUsed
				return nil, &ErrStackUnderflow{stackLen: stack.len(), required: 2}
			}
			x, y := stack.pop(), stack.peek()
			y.Or(&x, y)

		case XOR:
			gasUsed += GasFastestStep
			if stack.len() < 2 {
				if contract.Gas < gasUsed {
					return nil, ErrOutOfGas
				}
				contract.Gas -= gasUsed
				return nil, &ErrStackUnderflow{stackLen: stack.len(), required: 2}
			}
			x, y := stack.pop(), stack.peek()
			y.Xor(&x, y)

		case NOT:
			gasUsed += GasFastestStep
			if stack.len() < 1 {
				if contract.Gas < gasUsed {
					return nil, ErrOutOfGas
				}
				contract.Gas -= gasUsed
				return nil, &ErrStackUnderflow{stackLen: stack.len(), required: 1}
			}
			x := stack.peek()
			x.Not(x)

		case POP:
			gasUsed += GasQuickStep
			if stack.len() < 1 {
				if contract.Gas < gasUsed {
					return nil, ErrOutOfGas
				}
				contract.Gas -= gasUsed
				return nil, &ErrStackUnderflow{stackLen: 0, required: 1}
			}
			stack.pop()

		case DUP1, DUP2, DUP3, DUP4, DUP5, DUP6, DUP7:
			gasUsed += GasFastestStep
			n := int(op - DUP1 + 1)
			sLen := stack.len()
			if sLen < n {
				if contract.Gas < gasUsed {
					return nil, ErrOutOfGas
				}
				contract.Gas -= gasUsed
				return nil, &ErrStackUnderflow{stackLen: sLen, required: n}
			}
			if sLen >= int(params.StackLimit) {
				if contract.Gas < gasUsed {
					return nil, ErrOutOfGas
				}
				contract.Gas -= gasUsed
				return nil, &ErrStackOverflow{stackLen: sLen, limit: int(params.StackLimit)}
			}
			stack.dup(n)

		case SWAP1:
			gasUsed += GasFastestStep
			if stack.len() < 2 {
				if contract.Gas < gasUsed {
					return nil, ErrOutOfGas
				}
				contract.Gas -= gasUsed
				return nil, &ErrStackUnderflow{stackLen: stack.len(), required: 2}
			}
			stack.swap1()

		case SWAP2:
			gasUsed += GasFastestStep
			if stack.len() < 3 {
				if contract.Gas < gasUsed {
					return nil, ErrOutOfGas
				}
				contract.Gas -= gasUsed
				return nil, &ErrStackUnderflow{stackLen: stack.len(), required: 3}
			}
			stack.swap2()

		case SWAP3:
			gasUsed += GasFastestStep
			if stack.len() < 4 {
				if contract.Gas < gasUsed {
					return nil, ErrOutOfGas
				}
				contract.Gas -= gasUsed
				return nil, &ErrStackUnderflow{stackLen: stack.len(), required: 4}
			}
			stack.swap3()

		case PUSH1:
			gasUsed += GasFastestStep
			if stack.len() >= int(params.StackLimit) {
				if contract.Gas < gasUsed {
					return nil, ErrOutOfGas
				}
				contract.Gas -= gasUsed
				return nil, &ErrStackOverflow{stackLen: stack.len(), limit: int(params.StackLimit)}
			}
			pc++
			var val uint256.Int
			if pc < codeLen {
				val.SetUint64(uint64(code[pc]))
			}
			stack.push(&val)

		case PUSH2:
			gasUsed += GasFastestStep
			if stack.len() >= int(params.StackLimit) {
				if contract.Gas < gasUsed {
					return nil, ErrOutOfGas
				}
				contract.Gas -= gasUsed
				return nil, &ErrStackOverflow{stackLen: stack.len(), limit: int(params.StackLimit)}
			}
			var val uint256.Int
			if pc+2 < codeLen {
				val.SetUint64(uint64(code[pc+1])<<8 | uint64(code[pc+2]))
			} else if pc+1 < codeLen {
				val.SetUint64(uint64(code[pc+1]) << 8)
			}
			stack.push(&val)
			pc += 2

		case PUSH0:
			if !evm.chainRules.IsShanghai {
				return nil, &ErrInvalidOpCode{opcode: op}
			}
			gasUsed += GasQuickStep
			if stack.len() >= int(params.StackLimit) {
				if contract.Gas < gasUsed {
					return nil, ErrOutOfGas
				}
				contract.Gas -= gasUsed
				return nil, &ErrStackOverflow{stackLen: stack.len(), limit: int(params.StackLimit)}
			}
			var zero uint256.Int
			stack.push(&zero)

		case JUMPDEST:
			gasUsed += params.JumpdestGas

		case JUMP:
			gasUsed += GasMidStep
			if contract.Gas < gasUsed {
				return nil, ErrOutOfGas
			}
			contract.Gas -= gasUsed
			gasUsed = 0

			if stack.len() < 1 {
				return nil, &ErrStackUnderflow{stackLen: stack.len(), required: 1}
			}
			pos := stack.pop()

			if evm.abort.Load() {
				return nil, errStopToken
			}
			if !contract.validJumpdest(&pos) {
				return nil, ErrInvalidJump
			}
			pc = pos.Uint64()
			continue // skip pc++

		case JUMPI:
			gasUsed += GasSlowStep
			if contract.Gas < gasUsed {
				return nil, ErrOutOfGas
			}
			contract.Gas -= gasUsed
			gasUsed = 0

			if stack.len() < 2 {
				return nil, &ErrStackUnderflow{stackLen: stack.len(), required: 2}
			}
			pos, cond := stack.pop(), stack.pop()

			if evm.abort.Load() {
				return nil, errStopToken
			}
			if !cond.IsZero() {
				if !contract.validJumpdest(&pos) {
					return nil, ErrInvalidJump
				}
				pc = pos.Uint64()
				continue // skip pc++
			}
		// Flush gas and fall back to normal dispatch
		default:
			if contract.Gas < gasUsed {
				return nil, ErrOutOfGas
			}
			contract.Gas -= gasUsed
			gasUsed = 0

			// TODO: second return (cost) is for tracing — use it when adding a tracing to runExperimental.
			operation, _, memorySize, gasErr := chargeGasOp(op, evm, contract, stack, mem, jumpTable)
			if gasErr != nil {
				return nil, gasErr
			}
			if memorySize > 0 {
				mem.Resize(memorySize)
			}
			var res []byte
			res, err = operation.execute(&pc, evm, callContext)
			if err != nil {
				return res, err
			}
		}

		pc++
	}
}
