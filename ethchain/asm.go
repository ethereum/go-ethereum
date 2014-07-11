package ethchain

import (
	"fmt"
	"github.com/ethereum/eth-go/ethutil"
	"math/big"
)

func Disassemble(script []byte) (asm []string) {
	pc := new(big.Int)
	for {
		if pc.Cmp(big.NewInt(int64(len(script)))) >= 0 {
			return
		}

		// Get the memory location of pc
		val := script[pc.Int64()]
		// Get the opcode (it must be an opcode!)
		op := OpCode(val)

		asm = append(asm, fmt.Sprintf("%04v: %v", pc, op))

		switch op {
		case PUSH1, PUSH2, PUSH3, PUSH4, PUSH5, PUSH6, PUSH7, PUSH8, PUSH9, PUSH10, PUSH11, PUSH12, PUSH13, PUSH14, PUSH15, PUSH16, PUSH17, PUSH18, PUSH19, PUSH20, PUSH21, PUSH22, PUSH23, PUSH24, PUSH25, PUSH26, PUSH27, PUSH28, PUSH29, PUSH30, PUSH31, PUSH32:
			pc.Add(pc, ethutil.Big1)
			a := int64(op) - int64(PUSH1) + 1
			if int(pc.Int64()+a) > len(script) {
				return nil
			}

			data := script[pc.Int64() : pc.Int64()+a]
			if len(data) == 0 {
				data = []byte{0}
			}
			asm = append(asm, fmt.Sprintf("%04v: 0x%x", pc, data))

			pc.Add(pc, big.NewInt(a-1))
		}

		pc.Add(pc, ethutil.Big1)
	}

	return
}
