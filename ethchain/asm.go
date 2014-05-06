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

		asm = append(asm, fmt.Sprintf("%v", op))

		switch op {
		case oPUSH32: // Push PC+1 on to the stack
			pc.Add(pc, ethutil.Big1)
			data := script[pc.Int64() : pc.Int64()+32]
			val := ethutil.BigD(data)

			var b []byte
			if val.Int64() == 0 {
				b = []byte{0}
			} else {
				b = val.Bytes()
			}

			asm = append(asm, fmt.Sprintf("0x%x", b))

			pc.Add(pc, big.NewInt(31))
		}

		pc.Add(pc, ethutil.Big1)
	}

	return
}
