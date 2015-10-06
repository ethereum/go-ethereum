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
