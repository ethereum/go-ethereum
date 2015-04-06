package vm

import (
	"math/big"

	"gopkg.in/fatih/set.v0"
)

type destinations struct {
	set *set.Set
}

func (d *destinations) Has(dest *big.Int) bool {
	return d.set.Has(string(dest.Bytes()))
}

func (d *destinations) Add(dest *big.Int) {
	d.set.Add(string(dest.Bytes()))
}

func analyseJumpDests(code []byte) (dests *destinations) {
	dests = &destinations{set.New()}

	for pc := uint64(0); pc < uint64(len(code)); pc++ {
		var op OpCode = OpCode(code[pc])
		switch op {
		case PUSH1, PUSH2, PUSH3, PUSH4, PUSH5, PUSH6, PUSH7, PUSH8, PUSH9, PUSH10, PUSH11, PUSH12, PUSH13, PUSH14, PUSH15, PUSH16, PUSH17, PUSH18, PUSH19, PUSH20, PUSH21, PUSH22, PUSH23, PUSH24, PUSH25, PUSH26, PUSH27, PUSH28, PUSH29, PUSH30, PUSH31, PUSH32:
			a := uint64(op) - uint64(PUSH1) + 1

			pc += a
		case JUMPDEST:
			dests.Add(big.NewInt(int64(pc)))
		}
	}
	return
}
