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
	"fmt"
	"math/big"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/params"
	"github.com/hashicorp/golang-lru"
)

type progStatus int32

const (
	progUnknown progStatus = iota
	progCompile
	progReady
	progError

	defaultJitMaxCache int = 64
)

var (
	EnableJit   bool // Enables the JIT VM
	ForceJit    bool // Force the JIT, skip byte VM
	MaxProgSize int  // Max cache size for JIT Programs
)

var programs *lru.Cache

func init() {
	programs, _ = lru.New(defaultJitMaxCache)
}

// SetJITCacheSize recreates the program cache with the max given size. Setting
// a new cache is **not** thread safe. Use with caution.
func SetJITCacheSize(size int) {
	programs, _ = lru.New(size)
}

// GetProgram returns the program by id or nil when non-existent
func GetProgram(id common.Hash) *Program {
	if p, ok := programs.Get(id); ok {
		return p.(*Program)
	}

	return nil
}

// GenProgramStatus returns the status of the given program id
func GetProgramStatus(id common.Hash) progStatus {
	program := GetProgram(id)
	if program != nil {
		return progStatus(atomic.LoadInt32(&program.status))
	}

	return progUnknown
}

// Program is a compiled program for the JIT VM and holds all required for
// running a compiled JIT program.
type Program struct {
	Id     common.Hash // Id of the program
	status int32       // status should be accessed atomically

	contract *Contract

	instructions []instruction       // instruction set
	mapping      map[uint64]int      // real PC mapping to array indices
	destinations map[uint64]struct{} // cached jump destinations

	code []byte
}

// NewProgram returns a new JIT program
func NewProgram(code []byte) *Program {
	program := &Program{
		Id:           crypto.Sha3Hash(code),
		mapping:      make(map[uint64]int),
		destinations: make(map[uint64]struct{}),
		code:         code,
	}

	programs.Add(program.Id, program)
	return program
}

func (p *Program) addInstr(op OpCode, pc uint64, fn instrFn, data *big.Int) {
	// PUSH and DUP are a bit special. They all cost the same but we do want to have checking on stack push limit
	// PUSH is also allowed to calculate the same price for all PUSHes
	// DUP requirements are handled elsewhere (except for the stack limit check)
	baseOp := op
	if op >= PUSH1 && op <= PUSH32 {
		baseOp = PUSH1
	}
	if op >= DUP1 && op <= DUP16 {
		baseOp = DUP1
	}
	base := _baseCheck[baseOp]
	instr := instruction{op, pc, fn, data, base.gas, base.stackPop, base.stackPush}

	p.instructions = append(p.instructions, instr)
	p.mapping[pc] = len(p.instructions) - 1
}

// CompileProgram compiles the given program and return an error when it fails
func CompileProgram(program *Program) (err error) {
	if progStatus(atomic.LoadInt32(&program.status)) == progCompile {
		return nil
	}
	atomic.StoreInt32(&program.status, int32(progCompile))
	defer func() {
		if err != nil {
			atomic.StoreInt32(&program.status, int32(progError))
		} else {
			atomic.StoreInt32(&program.status, int32(progReady))
		}
	}()
	if glog.V(logger.Debug) {
		glog.Infof("compiling %x\n", program.Id[:4])
		tstart := time.Now()
		defer func() {
			glog.Infof("compiled  %x instrc: %d time: %v\n", program.Id[:4], len(program.instructions), time.Since(tstart))
		}()
	}

	// loop thru the opcodes and "compile" in to instructions
	for pc := uint64(0); pc < uint64(len(program.code)); pc++ {
		switch op := OpCode(program.code[pc]); op {
		case ADD:
			program.addInstr(op, pc, opAdd, nil)
		case SUB:
			program.addInstr(op, pc, opSub, nil)
		case MUL:
			program.addInstr(op, pc, opMul, nil)
		case DIV:
			program.addInstr(op, pc, opDiv, nil)
		case SDIV:
			program.addInstr(op, pc, opSdiv, nil)
		case MOD:
			program.addInstr(op, pc, opMod, nil)
		case SMOD:
			program.addInstr(op, pc, opSmod, nil)
		case EXP:
			program.addInstr(op, pc, opExp, nil)
		case SIGNEXTEND:
			program.addInstr(op, pc, opSignExtend, nil)
		case NOT:
			program.addInstr(op, pc, opNot, nil)
		case LT:
			program.addInstr(op, pc, opLt, nil)
		case GT:
			program.addInstr(op, pc, opGt, nil)
		case SLT:
			program.addInstr(op, pc, opSlt, nil)
		case SGT:
			program.addInstr(op, pc, opSgt, nil)
		case EQ:
			program.addInstr(op, pc, opEq, nil)
		case ISZERO:
			program.addInstr(op, pc, opIszero, nil)
		case AND:
			program.addInstr(op, pc, opAnd, nil)
		case OR:
			program.addInstr(op, pc, opOr, nil)
		case XOR:
			program.addInstr(op, pc, opXor, nil)
		case BYTE:
			program.addInstr(op, pc, opByte, nil)
		case ADDMOD:
			program.addInstr(op, pc, opAddmod, nil)
		case MULMOD:
			program.addInstr(op, pc, opMulmod, nil)
		case SHA3:
			program.addInstr(op, pc, opSha3, nil)
		case ADDRESS:
			program.addInstr(op, pc, opAddress, nil)
		case BALANCE:
			program.addInstr(op, pc, opBalance, nil)
		case ORIGIN:
			program.addInstr(op, pc, opOrigin, nil)
		case CALLER:
			program.addInstr(op, pc, opCaller, nil)
		case CALLVALUE:
			program.addInstr(op, pc, opCallValue, nil)
		case CALLDATALOAD:
			program.addInstr(op, pc, opCalldataLoad, nil)
		case CALLDATASIZE:
			program.addInstr(op, pc, opCalldataSize, nil)
		case CALLDATACOPY:
			program.addInstr(op, pc, opCalldataCopy, nil)
		case CODESIZE:
			program.addInstr(op, pc, opCodeSize, nil)
		case EXTCODESIZE:
			program.addInstr(op, pc, opExtCodeSize, nil)
		case CODECOPY:
			program.addInstr(op, pc, opCodeCopy, nil)
		case EXTCODECOPY:
			program.addInstr(op, pc, opExtCodeCopy, nil)
		case GASPRICE:
			program.addInstr(op, pc, opGasprice, nil)
		case BLOCKHASH:
			program.addInstr(op, pc, opBlockhash, nil)
		case COINBASE:
			program.addInstr(op, pc, opCoinbase, nil)
		case TIMESTAMP:
			program.addInstr(op, pc, opTimestamp, nil)
		case NUMBER:
			program.addInstr(op, pc, opNumber, nil)
		case DIFFICULTY:
			program.addInstr(op, pc, opDifficulty, nil)
		case GASLIMIT:
			program.addInstr(op, pc, opGasLimit, nil)
		case PUSH1, PUSH2, PUSH3, PUSH4, PUSH5, PUSH6, PUSH7, PUSH8, PUSH9, PUSH10, PUSH11, PUSH12, PUSH13, PUSH14, PUSH15, PUSH16, PUSH17, PUSH18, PUSH19, PUSH20, PUSH21, PUSH22, PUSH23, PUSH24, PUSH25, PUSH26, PUSH27, PUSH28, PUSH29, PUSH30, PUSH31, PUSH32:
			size := uint64(op - PUSH1 + 1)
			bytes := getData([]byte(program.code), new(big.Int).SetUint64(pc+1), new(big.Int).SetUint64(size))

			program.addInstr(op, pc, opPush, common.Bytes2Big(bytes))

			pc += size

		case POP:
			program.addInstr(op, pc, opPop, nil)
		case DUP1, DUP2, DUP3, DUP4, DUP5, DUP6, DUP7, DUP8, DUP9, DUP10, DUP11, DUP12, DUP13, DUP14, DUP15, DUP16:
			program.addInstr(op, pc, opDup, big.NewInt(int64(op-DUP1+1)))
		case SWAP1, SWAP2, SWAP3, SWAP4, SWAP5, SWAP6, SWAP7, SWAP8, SWAP9, SWAP10, SWAP11, SWAP12, SWAP13, SWAP14, SWAP15, SWAP16:
			program.addInstr(op, pc, opSwap, big.NewInt(int64(op-SWAP1+2)))
		case LOG0, LOG1, LOG2, LOG3, LOG4:
			program.addInstr(op, pc, opLog, big.NewInt(int64(op-LOG0)))
		case MLOAD:
			program.addInstr(op, pc, opMload, nil)
		case MSTORE:
			program.addInstr(op, pc, opMstore, nil)
		case MSTORE8:
			program.addInstr(op, pc, opMstore8, nil)
		case SLOAD:
			program.addInstr(op, pc, opSload, nil)
		case SSTORE:
			program.addInstr(op, pc, opSstore, nil)
		case JUMP:
			program.addInstr(op, pc, opJump, nil)
		case JUMPI:
			program.addInstr(op, pc, opJumpi, nil)
		case JUMPDEST:
			program.addInstr(op, pc, opJumpdest, nil)
			program.destinations[pc] = struct{}{}
		case PC:
			program.addInstr(op, pc, opPc, big.NewInt(int64(pc)))
		case MSIZE:
			program.addInstr(op, pc, opMsize, nil)
		case GAS:
			program.addInstr(op, pc, opGas, nil)
		case CREATE:
			program.addInstr(op, pc, opCreate, nil)
		case CALL:
			program.addInstr(op, pc, opCall, nil)
		case CALLCODE:
			program.addInstr(op, pc, opCallCode, nil)
		case RETURN:
			program.addInstr(op, pc, opReturn, nil)
		case SUICIDE:
			program.addInstr(op, pc, opSuicide, nil)
		case STOP: // Stop the contract
			program.addInstr(op, pc, opStop, nil)
		default:
			program.addInstr(op, pc, nil, nil)
		}
	}

	return nil
}

// RunProgram runs the program given the enviroment and contract and returns an
// error if the execution failed (non-consensus)
func RunProgram(program *Program, env Environment, contract *Contract, input []byte) ([]byte, error) {
	return runProgram(program, 0, NewMemory(), newstack(), env, contract, input)
}

func runProgram(program *Program, pcstart uint64, mem *Memory, stack *stack, env Environment, contract *Contract, input []byte) ([]byte, error) {
	contract.Input = input

	var (
		caller         = contract.caller
		statedb        = env.Db()
		pc         int = program.mapping[pcstart]
		instrCount     = 0

		jump = func(to *big.Int) error {
			if !validDest(program.destinations, to) {
				nop := contract.GetOp(to.Uint64())
				return fmt.Errorf("invalid jump destination (%v) %v", nop, to)
			}

			pc = program.mapping[to.Uint64()]

			return nil
		}
	)

	if glog.V(logger.Debug) {
		glog.Infof("running JIT program %x\n", program.Id[:4])
		tstart := time.Now()
		defer func() {
			glog.Infof("JIT program %x done. time: %v instrc: %v\n", program.Id[:4], time.Since(tstart), instrCount)
		}()
	}

	for pc < len(program.instructions) {
		instrCount++

		instr := program.instructions[pc]

		// calculate the new memory size and gas price for the current executing opcode
		newMemSize, cost, err := jitCalculateGasAndSize(env, contract, caller, instr, statedb, mem, stack)
		if err != nil {
			return nil, err
		}

		// Use the calculated gas. When insufficient gas is present, use all gas and return an
		// Out Of Gas error
		if !contract.UseGas(cost) {
			return nil, OutOfGasError
		}
		// Resize the memory calculated previously
		mem.Resize(newMemSize.Uint64())

		// These opcodes return an argument and are thefor handled
		// differently from the rest of the opcodes
		switch instr.op {
		case JUMP:
			if err := jump(stack.pop()); err != nil {
				return nil, err
			}
			continue
		case JUMPI:
			pos, cond := stack.pop(), stack.pop()

			if cond.Cmp(common.BigTrue) >= 0 {
				if err := jump(pos); err != nil {
					return nil, err
				}
				continue
			}
		case RETURN:
			offset, size := stack.pop(), stack.pop()
			ret := mem.GetPtr(offset.Int64(), size.Int64())

			return contract.Return(ret), nil
		case SUICIDE:
			instr.fn(instr, nil, env, contract, mem, stack)

			return contract.Return(nil), nil
		case STOP:
			return contract.Return(nil), nil
		default:
			if instr.fn == nil {
				return nil, fmt.Errorf("Invalid opcode %x", instr.op)
			}

			instr.fn(instr, nil, env, contract, mem, stack)
		}

		pc++
	}

	contract.Input = nil

	return contract.Return(nil), nil
}

// validDest checks if the given distination is a valid one given the
// destination table of the program
func validDest(dests map[uint64]struct{}, dest *big.Int) bool {
	// PC cannot go beyond len(code) and certainly can't be bigger than 64bits.
	// Don't bother checking for JUMPDEST in that case.
	if dest.Cmp(bigMaxUint64) > 0 {
		return false
	}
	_, ok := dests[dest.Uint64()]
	return ok
}

// jitCalculateGasAndSize calculates the required given the opcode and stack items calculates the new memorysize for
// the operation. This does not reduce gas or resizes the memory.
func jitCalculateGasAndSize(env Environment, contract *Contract, caller ContractRef, instr instruction, statedb Database, mem *Memory, stack *stack) (*big.Int, *big.Int, error) {
	var (
		gas                 = new(big.Int)
		newMemSize *big.Int = new(big.Int)
	)
	err := jitBaseCheck(instr, stack, gas)
	if err != nil {
		return nil, nil, err
	}

	// stack Check, memory resize & gas phase
	switch op := instr.op; op {
	case SWAP1, SWAP2, SWAP3, SWAP4, SWAP5, SWAP6, SWAP7, SWAP8, SWAP9, SWAP10, SWAP11, SWAP12, SWAP13, SWAP14, SWAP15, SWAP16:
		n := int(op - SWAP1 + 2)
		err := stack.require(n)
		if err != nil {
			return nil, nil, err
		}
		gas.Set(GasFastestStep)
	case DUP1, DUP2, DUP3, DUP4, DUP5, DUP6, DUP7, DUP8, DUP9, DUP10, DUP11, DUP12, DUP13, DUP14, DUP15, DUP16:
		n := int(op - DUP1 + 1)
		err := stack.require(n)
		if err != nil {
			return nil, nil, err
		}
		gas.Set(GasFastestStep)
	case LOG0, LOG1, LOG2, LOG3, LOG4:
		n := int(op - LOG0)
		err := stack.require(n + 2)
		if err != nil {
			return nil, nil, err
		}

		mSize, mStart := stack.data[stack.len()-2], stack.data[stack.len()-1]

		add := new(big.Int)
		gas.Add(gas, params.LogGas)
		gas.Add(gas, add.Mul(big.NewInt(int64(n)), params.LogTopicGas))
		gas.Add(gas, add.Mul(mSize, params.LogDataGas))

		newMemSize = calcMemSize(mStart, mSize)
	case EXP:
		gas.Add(gas, new(big.Int).Mul(big.NewInt(int64(len(stack.data[stack.len()-2].Bytes()))), params.ExpByteGas))
	case SSTORE:
		err := stack.require(2)
		if err != nil {
			return nil, nil, err
		}

		var g *big.Int
		y, x := stack.data[stack.len()-2], stack.data[stack.len()-1]
		val := statedb.GetState(contract.Address(), common.BigToHash(x))

		// This checks for 3 scenario's and calculates gas accordingly
		// 1. From a zero-value address to a non-zero value         (NEW VALUE)
		// 2. From a non-zero value address to a zero-value address (DELETE)
		// 3. From a nen-zero to a non-zero                         (CHANGE)
		if common.EmptyHash(val) && !common.EmptyHash(common.BigToHash(y)) {
			g = params.SstoreSetGas
		} else if !common.EmptyHash(val) && common.EmptyHash(common.BigToHash(y)) {
			statedb.AddRefund(params.SstoreRefundGas)

			g = params.SstoreClearGas
		} else {
			g = params.SstoreClearGas
		}
		gas.Set(g)
	case SUICIDE:
		if !statedb.IsDeleted(contract.Address()) {
			statedb.AddRefund(params.SuicideRefundGas)
		}
	case MLOAD:
		newMemSize = calcMemSize(stack.peek(), u256(32))
	case MSTORE8:
		newMemSize = calcMemSize(stack.peek(), u256(1))
	case MSTORE:
		newMemSize = calcMemSize(stack.peek(), u256(32))
	case RETURN:
		newMemSize = calcMemSize(stack.peek(), stack.data[stack.len()-2])
	case SHA3:
		newMemSize = calcMemSize(stack.peek(), stack.data[stack.len()-2])

		words := toWordSize(stack.data[stack.len()-2])
		gas.Add(gas, words.Mul(words, params.Sha3WordGas))
	case CALLDATACOPY:
		newMemSize = calcMemSize(stack.peek(), stack.data[stack.len()-3])

		words := toWordSize(stack.data[stack.len()-3])
		gas.Add(gas, words.Mul(words, params.CopyGas))
	case CODECOPY:
		newMemSize = calcMemSize(stack.peek(), stack.data[stack.len()-3])

		words := toWordSize(stack.data[stack.len()-3])
		gas.Add(gas, words.Mul(words, params.CopyGas))
	case EXTCODECOPY:
		newMemSize = calcMemSize(stack.data[stack.len()-2], stack.data[stack.len()-4])

		words := toWordSize(stack.data[stack.len()-4])
		gas.Add(gas, words.Mul(words, params.CopyGas))

	case CREATE:
		newMemSize = calcMemSize(stack.data[stack.len()-2], stack.data[stack.len()-3])
	case CALL, CALLCODE:
		gas.Add(gas, stack.data[stack.len()-1])

		if op == CALL {
			//if env.Db().GetStateObject(common.BigToAddress(stack.data[stack.len()-2])) == nil {
			if !env.Db().Exist(common.BigToAddress(stack.data[stack.len()-2])) {
				gas.Add(gas, params.CallNewAccountGas)
			}
		}

		if len(stack.data[stack.len()-3].Bytes()) > 0 {
			gas.Add(gas, params.CallValueTransferGas)
		}

		x := calcMemSize(stack.data[stack.len()-6], stack.data[stack.len()-7])
		y := calcMemSize(stack.data[stack.len()-4], stack.data[stack.len()-5])

		newMemSize = common.BigMax(x, y)
	}
	quadMemGas(mem, newMemSize, gas)

	return newMemSize, gas, nil
}

// jitBaseCheck is the same as baseCheck except it doesn't do the look up in the
// gas table. This is done during compilation instead.
func jitBaseCheck(instr instruction, stack *stack, gas *big.Int) error {
	err := stack.require(instr.spop)
	if err != nil {
		return err
	}

	if instr.spush > 0 && stack.len()-instr.spop+instr.spush > int(params.StackLimit.Int64()) {
		return fmt.Errorf("stack limit reached %d (%d)", stack.len(), params.StackLimit.Int64())
	}

	// nil on gas means no base calculation
	if instr.gas == nil {
		return nil
	}

	gas.Add(gas, instr.gas)

	return nil
}
