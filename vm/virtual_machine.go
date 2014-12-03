package vm

import "math/big"

type VirtualMachine interface {
	Env() Environment
	Run(me, caller ClosureRef, code []byte, value, gas, price *big.Int, data []byte) ([]byte, error)
	Depth() int
	Printf(string, ...interface{}) VirtualMachine
	Endl() VirtualMachine
}
