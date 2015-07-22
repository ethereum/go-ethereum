// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// go-ethereum is free software: you can redistribute it and/or modify
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
	"os"
	"unicode"

	"github.com/ethereum/go-ethereum/common"
)

func StdErrFormat(logs []StructLog) {
	fmt.Fprintf(os.Stderr, "VM STAT %d OPs\n", len(logs))
	for _, log := range logs {
		fmt.Fprintf(os.Stderr, "PC %08d: %s GAS: %v COST: %v", log.Pc, log.Op, log.Gas, log.GasCost)
		if log.Err != nil {
			fmt.Fprintf(os.Stderr, " ERROR: %v", log.Err)
		}
		fmt.Fprintf(os.Stderr, "\n")

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
				} else if unicode.IsPrint(rune(r)) {
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
