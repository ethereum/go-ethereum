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

type fnId string

// optimeProgram optimises a JIT program creating segments out of program
// instructions. Currently covered are multi-pushes and static jumps
func OptimiseProgram(program *Program) {
	var load []instruction

	var (
		statsJump    = 0
		statsOldPush = 0
		statsNewPush = 0
	)

	if glog.V(logger.Debug) {
		glog.Infof("optimising %x\n", program.Id[:4])
		tstart := time.Now()
		defer func() {
			glog.Infof("optimised %x done in %v with JMP: %d PSH: %d/%d\n", program.Id[:4], time.Since(tstart), statsJump, statsNewPush, statsOldPush)
		}()
	}

	code := Parse(program.code)
	for _, test := range [][]OpCode{
		[]OpCode{PUSH, DUP, EQ, PUSH, JUMPI},
	} {
		matchCount := 0
		fmt.Printf("found %d match count on: %v\n", matchCount, test)
	}

	MatchFn(code, []OpCode{PUSH, PUSH, EXP}, func(i int) bool {
		// TODO optimise this instruction
		return true
	})

	funcTable := make(map[fnId]uint64)
	MatchFn(code, []OpCode{DUP, PUSH, EQ, PUSH, JUMPI}, func(i int) bool {
		pushOp := code[i+1]
		size := int64(program.code[pushOp.pc]) - int64(PUSH1) + 1
		funcId := fnId(getData([]byte(program.code), big.NewInt(int64(pushOp.pc+1)), big.NewInt(size)))

		pushOp = code[i+3]
		size = int64(program.code[pushOp.pc]) - int64(PUSH1) + 1
		position := common.Bytes2Big(getData([]byte(program.code), big.NewInt(int64(pushOp.pc+1)), big.NewInt(size))).Uint64()
		glog.Infof("jumpTable entry: %x => %d\n", funcId, position)

		funcTable[funcId] = position

		return true
	})

	for i := 0; i < len(program.instructions); i++ {
		instr := program.instructions[i].(instruction)

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
