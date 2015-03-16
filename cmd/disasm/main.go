package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/vm"
)

func main() {
	code, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	code = common.Hex2Bytes(string(code[:len(code)-1]))
	fmt.Printf("%x\n", code)

	for pc := uint64(0); pc < uint64(len(code)); pc++ {
		op := vm.OpCode(code[pc])
		fmt.Printf("%-5d  %v", pc, op)

		switch op {
		case vm.PUSH1, vm.PUSH2, vm.PUSH3, vm.PUSH4, vm.PUSH5, vm.PUSH6, vm.PUSH7, vm.PUSH8, vm.PUSH9, vm.PUSH10, vm.PUSH11, vm.PUSH12, vm.PUSH13, vm.PUSH14, vm.PUSH15, vm.PUSH16, vm.PUSH17, vm.PUSH18, vm.PUSH19, vm.PUSH20, vm.PUSH21, vm.PUSH22, vm.PUSH23, vm.PUSH24, vm.PUSH25, vm.PUSH26, vm.PUSH27, vm.PUSH28, vm.PUSH29, vm.PUSH30, vm.PUSH31, vm.PUSH32:
			a := uint64(op) - uint64(vm.PUSH1) + 1
			fmt.Printf("  => %x", code[pc+1:pc+1+a])

			pc += a
		}
		fmt.Println()
	}
}
