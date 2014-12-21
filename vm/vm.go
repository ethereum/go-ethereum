package vm

import "math/big"

// BIG FAT WARNING. THIS VM IS NOT YET IS USE!
// I want to get all VM tests pass first before updating this VM

type Vm struct {
	env   Environment
	err   error
	depth int
}

func New(env Environment, typ Type) VirtualMachine {
	switch typ {
	case DebugVmTy:
		return NewDebugVm(env)
	default:
		return &Vm{env: env}
	}
}

func (self *Vm) Run(me, caller ClosureRef, code []byte, value, gas, price *big.Int, data []byte) (ret []byte, err error) {
	panic("not implemented")
}

func (self *Vm) Env() Environment {
	return self.env
}

func (self *Vm) Depth() int {
	return self.depth
}

func (self *Vm) Printf(format string, v ...interface{}) VirtualMachine { return self }
func (self *Vm) Endl() VirtualMachine                                  { return self }
