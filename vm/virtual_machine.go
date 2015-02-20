package vm

import "math/big"

type VirtualMachine interface {
	Env() Environment
	Run(me, caller ContextRef, code []byte, value, gas, price *big.Int, data []byte) ([]byte, error)
	Printf(string, ...interface{}) VirtualMachine
	Endl() VirtualMachine
}
