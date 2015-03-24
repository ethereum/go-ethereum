package core

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
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
		op := vm.OpCode(val)

		asm = append(asm, fmt.Sprintf("%04v: %v", pc, op))

		switch op {
		case vm.PUSH1, vm.PUSH2, vm.PUSH3, vm.PUSH4, vm.PUSH5, vm.PUSH6, vm.PUSH7, vm.PUSH8,
			vm.PUSH9, vm.PUSH10, vm.PUSH11, vm.PUSH12, vm.PUSH13, vm.PUSH14, vm.PUSH15,
			vm.PUSH16, vm.PUSH17, vm.PUSH18, vm.PUSH19, vm.PUSH20, vm.PUSH21, vm.PUSH22,
			vm.PUSH23, vm.PUSH24, vm.PUSH25, vm.PUSH26, vm.PUSH27, vm.PUSH28, vm.PUSH29,
			vm.PUSH30, vm.PUSH31, vm.PUSH32:
			pc.Add(pc, common.Big1)
			a := int64(op) - int64(vm.PUSH1) + 1
			if int(pc.Int64()+a) > len(script) {
				return
			}

			data := script[pc.Int64() : pc.Int64()+a]
			if len(data) == 0 {
				data = []byte{0}
			}
			asm = append(asm, fmt.Sprintf("%04v: 0x%x", pc, data))

			pc.Add(pc, big.NewInt(a-1))
		}

		pc.Add(pc, common.Big1)
	}

	return asm
}
