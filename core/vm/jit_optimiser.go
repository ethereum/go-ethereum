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
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

// optimeProgram optimises a JIT program creating segments out of program
// instructions. Currently covered are multi-pushes and static jumps
func optimiseProgram(program *Program) {
	var load []instruction

	var (
		statsJump = 0
		statsPush = 0
	)

	if glog.V(logger.Debug) {
		glog.Infof("optimising %x\n", program.Id[:4])
		tstart := time.Now()
		defer func() {
			glog.Infof("optimised %x done in %v with JMP: %d PSH: %d\n", program.Id[:4], time.Since(tstart), statsJump, statsPush)
		}()
	}

	/*
		code := Parse(program.code)
		for _, test := range [][]OpCode{
			[]OpCode{PUSH, PUSH, ADD},
			[]OpCode{PUSH, PUSH, SUB},
			[]OpCode{PUSH, PUSH, MUL},
			[]OpCode{PUSH, PUSH, DIV},
		} {
			matchCount := 0
			MatchFn(code, test, func(i int) bool {
				matchCount++
				return true
			})
			fmt.Printf("found %d match count on: %v\n", matchCount, test)
		}
	*/

	for i := 0; i < len(program.instructions); i++ {
		instr := program.instructions[i].(instruction)

		switch {
		case instr.op.IsPush():
			load = append(load, instr)
		case instr.op.IsStaticJump():
			if len(load) == 0 {
				continue
			}
			// if the push load is greater than 1, finalise that
			// segment first
			if len(load) > 2 {
				seg, size := makePushSeg(load[:len(load)-1])
				program.instructions[i-size-1] = seg
				statsPush++
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
				statsPush++
			}
			load = nil
		}
	}
}

// makePushSeg creates a new push segment from N amount of push instructions
func makePushSeg(instrs []instruction) (pushSeg, int) {
	var (
		data []*big.Int
		gas  = new(big.Int)
	)

	for _, instr := range instrs {
		data = append(data, instr.data)
		gas.Add(gas, instr.gas)
	}

	return pushSeg{data, gas}, len(instrs)
}

// makeStaticJumpSeg creates a new static jump segment from a predefined
// destination (PUSH, JUMP).
func makeStaticJumpSeg(to *big.Int, program *Program) jumpSeg {
	gas := new(big.Int)
	gas.Add(gas, _baseCheck[PUSH1].gas)
	gas.Add(gas, _baseCheck[JUMP].gas)

	contract := &Contract{Code: program.code}
	pos, err := jump(program.mapping, program.destinations, contract, to)
	return jumpSeg{pos, err, gas}
}
