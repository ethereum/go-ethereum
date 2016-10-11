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
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

type (
	fnId   string
	fnInfo struct {
		gas uint64
		pos uint64
	}
	jumpTableInstr map[fnId]fnInfo
	arithSeg       struct {
		value   *big.Int
		gas     uint64
		pcRange uint64
	}
)

func (as arithSeg) do(program *Program, pc *uint64, env *Environment, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	// Use the calculated gas. When insufficient gas is present, use all gas and return an
	// Out Of Gas error
	if !contract.UseGas(as.gas) {
		return nil, OutOfGasError
	}

	stack.push(new(big.Int).Set(as.value))

	*pc += as.pcRange
	return nil, nil
}
func (as arithSeg) halts() bool    { return false }
func (as arithSeg) Op() OpCode     { return OPTIMISED }
func (as arithSeg) String() string { return fmt.Sprintf("ARITH_SEG(%v: %v)", EXP, as.value) }

func (jti jumpTableInstr) do(program *Program, pc *uint64, env *Environment, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	fnId := fnId(stack.data[len(stack.data)-1].Bytes())
	fn, ok := jti[fnId]
	glog.V(logger.Debug).Infof("function call: 0x%x (p=%d exist=%v)", fnId, fn.pos, ok)
	*pc = fn.pos

	return nil, nil
}
func (jumpTableInstr) halts() bool        { return false }
func (jumpTableInstr) Op() OpCode         { return OPTIMISED }
func (jti jumpTableInstr) String() string { return fmt.Sprintf("FUNC_TABLE(%d)", len(jti)) }

// optimeProgram optimises a JIT program creating segments out of program
// instructions. Currently covered are multi-pushes and static jumps
func OptimiseProgram(program *Program) {
	if glog.V(logger.Debug) {
		glog.Infof("optimising %x\n", program.Id[:4])
		tstart := time.Now()
		defer func() {
			glog.Infof("optimised %x done in %v\n", program.Id[:4], time.Since(tstart))
		}()
	}

	code := Parse(program.code)
	MatchFn(code, []OpCode{PUSH, PUSH, EXP}, func(i int) bool {
		var (
			instr    arithSeg
			instrPos uint64
		)

		pushOp := code[i]
		instrPos = pushOp.pc

		size := int64(program.code[pushOp.pc]) - int64(PUSH1) + 1
		instr.pcRange += uint64(size) + 1 // size + push instruction
		exponent := getData([]byte(program.code), big.NewInt(int64(pushOp.pc+1)), big.NewInt(size))

		pushOp = code[i+1]
		size = int64(program.code[pushOp.pc]) - int64(PUSH1) + 1
		instr.pcRange += uint64(size) + 1 // size + push instruction
		base := getData([]byte(program.code), big.NewInt(int64(pushOp.pc+1)), big.NewInt(size))

		//instr.value = math.Exp(common.Bytes2Big(base), common.Bytes2Big(exponent))
		instr.value = new(big.Int).Exp(common.Bytes2Big(base), common.Bytes2Big(exponent), Pow256)
		instr.gas = GasFastestStep64*2 + GasSlowStep64 + uint64(len(exponent))*ExpByteGas64

		instr.pcRange++
		instr.pcRange--
		instr.pcRange--

		program.instructions[program.mapping[instrPos]] = instr

		return true
	})

	funcTable, jumpStart := make(jumpTableInstr), uint64(0)
	MatchFn(code, []OpCode{PUSH, DUP, EQ, PUSH, JUMPI}, func(i int) bool {
		pushOp := code[i]
		jumpStart = pushOp.pc

		size := int64(program.code[pushOp.pc]) - int64(PUSH1) + 1
		funcId := fnId(getData([]byte(program.code), big.NewInt(int64(pushOp.pc+1)), big.NewInt(size)))

		pushOp = code[i+3]
		size = int64(program.code[pushOp.pc]) - int64(PUSH1) + 1
		position := common.Bytes2Big(getData([]byte(program.code), big.NewInt(int64(pushOp.pc+1)), big.NewInt(size))).Uint64()
		glog.V(logger.Debug).Infof("jumpTable start : %x => %d - %d\n", funcId, position, jumpStart)

		// TODO set the right amount of gas
		funcTable[funcId] = fnInfo{pos: program.mapping[position], gas: 0}

		return true
	})

	MatchFn(code, []OpCode{DUP, PUSH, EQ, PUSH, JUMPI}, func(i int) bool {
		pushOp := code[i+1]
		size := int64(program.code[pushOp.pc]) - int64(PUSH1) + 1
		funcId := fnId(getData([]byte(program.code), big.NewInt(int64(pushOp.pc+1)), big.NewInt(size)))

		pushOp = code[i+3]
		size = int64(program.code[pushOp.pc]) - int64(PUSH1) + 1
		position := common.Bytes2Big(getData([]byte(program.code), big.NewInt(int64(pushOp.pc+1)), big.NewInt(size))).Uint64()
		glog.V(logger.Debug).Infof("jumpTable entry: %x => %d\n", funcId, position)

		// TODO set the rigth amount of gas
		funcTable[funcId] = fnInfo{pos: program.mapping[position], gas: 0}

		return true
	})
	if len(funcTable) > 0 {
		program.instructions[program.mapping[jumpStart]] = funcTable
	}

	var (
		load         []instruction
		statsJump    = 0
		statsOldPush = 0
		statsNewPush = 0
	)
	for i := 0; i < len(program.instructions); i++ {
		instr, ok := program.instructions[i].(instruction)
		if !ok {
			continue
		}

		switch {
		case instr.op.IsPush():
			load = append(load, instr)
			statsOldPush++
		case instr.op.IsStaticJump():
			if len(load) == 0 {
				continue
			}
			// if the push load is greater than 1, finalise that
			// segment first
			if len(load) > 2 {
				seg, size := makePushSeg(load[:len(load)-1])
				program.instructions[i-size-1] = seg
				statsNewPush++
			}
			// create a segment consisting of a pre determined
			// jump, destination and validity.
			seg := makeStaticJumpSeg(load[len(load)-1].data, program)
			program.instructions[i-1] = seg
			statsJump++

			load = nil
		default:
			// create a new N pushes segment
			if len(load) > 1 {
				seg, size := makePushSeg(load)
				program.instructions[i-size] = seg
				statsNewPush++
			}
			load = nil
		}
	}
	glog.V(logger.Debug).Infof("optimised %d pushes as %d pushes\n", statsOldPush, statsNewPush)
	glog.V(logger.Debug).Infof("optimised %d static jumps\n", statsJump)
}

// makePushSeg creates a new push segment from N amount of push instructions
func makePushSeg(instrs []instruction) (pushSeg, int) {
	var (
		data []*big.Int
		gas  uint64
	)

	for _, instr := range instrs {
		data = append(data, instr.data)
		gas += instr.gas
	}

	return pushSeg{data, gas}, len(instrs)
}

// makeStaticJumpSeg creates a new static jump segment from a predefined
// destination (PUSH, JUMP).
func makeStaticJumpSeg(to *big.Int, program *Program) jumpSeg {
	gas := _baseCheck[PUSH1].gas + _baseCheck[JUMP].gas

	contract := &Contract{Code: program.code}
	pos, err := jump(program.mapping, program.destinations, contract, to)
	return jumpSeg{pos, err, gas}
}
