package vm

import (
	"fmt"
	"os"
	"unicode/utf8"

	"github.com/ethereum/go-ethereum/common"
)

func StdErrFormat(logs []StructLog) {
	fmt.Fprintf(os.Stderr, "VM Stats %d ops\n", len(logs))
	for _, log := range logs {
		fmt.Fprintf(os.Stderr, "PC %08d: %s GAS: %v COST: %v\n", log.Pc, log.Op, log.Gas, log.GasCost)
		fmt.Fprintln(os.Stderr, "STACK =", len(log.Stack))

		for i := len(log.Stack) - 1; i >= 0; i-- {
			fmt.Fprintf(os.Stderr, "%04d: %x\n", len(log.Stack)-i-1, common.LeftPadBytes(log.Stack[i].Bytes(), 32))
		}

		const maxMem = 10
		addr := 0
		fmt.Fprintln(os.Stderr, "MEM =", len(log.Memory))
		for i := 0; i+16 <= len(log.Memory) && addr < maxMem; i += 16 {
			data := log.Memory[i : i+16]
			str := fmt.Sprintf("%04d: % x  ", addr*16, data)
			for _, r := range data {
				if r == 0 {
					str += "."
				} else if utf8.ValidRune(rune(r)) {
					str += fmt.Sprintf("%s", string(r))
				} else {
					str += "?"
				}
			}
			addr++
			fmt.Fprintln(os.Stderr, str)
		}

		fmt.Fprintln(os.Stderr, "STORAGE =", len(log.Storage))
		for h, item := range log.Storage {
			fmt.Fprintf(os.Stderr, "%x: %x\n", h, common.LeftPadBytes(item, 32))
		}
		fmt.Fprintln(os.Stderr)
	}
}
