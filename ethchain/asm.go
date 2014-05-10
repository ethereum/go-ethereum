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
		case oPUSH1, oPUSH2, oPUSH3, oPUSH4, oPUSH5, oPUSH6, oPUSH7, oPUSH8, oPUSH9, oPUSH10, oPUSH11, oPUSH12, oPUSH13, oPUSH14, oPUSH15, oPUSH16, oPUSH17, oPUSH18, oPUSH19, oPUSH20, oPUSH21, oPUSH22, oPUSH23, oPUSH24, oPUSH25, oPUSH26, oPUSH27, oPUSH28, oPUSH29, oPUSH30, oPUSH31, oPUSH32:
			pc.Add(pc, ethutil.Big1)
			a := int64(op) - int64(oPUSH1) + 1
			data := script[pc.Int64() : pc.Int64()+a]
			val := ethutil.BigD(data)

			var b []byte
			if val.Int64() == 0 {
				b = []byte{0}
			} else {
				b = val.Bytes()
			}

			asm = append(asm, fmt.Sprintf("0x%x", b))

			pc.Add(pc, big.NewInt(a-1))
		}

		pc.Add(pc, ethutil.Big1)
	}

	return
}
