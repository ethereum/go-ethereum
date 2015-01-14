// +build !evmjit

package vm

import "fmt"

func NewJitVm(env Environment) VirtualMachine {
	fmt.Printf("Warning! EVM JIT not enabled.\n")
	return NewDebugVm(env)
}
