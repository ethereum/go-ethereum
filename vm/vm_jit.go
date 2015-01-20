package vm

import "math/big"

type JitVm struct {
	env    Environment
	backup *Vm
}

func NewJitVm(env Environment) *JitVm {
	backupVm := New(env)
	return &JitVm{env: env, backup: backupVm}
}

func (self *JitVm) Run(me, caller ContextRef, code []byte, value, gas, price *big.Int, callData []byte) (ret []byte, err error) {
	return self.backup.Run(me, caller, code, value, gas, price, callData)
}

func (self *JitVm) Printf(format string, v ...interface{}) VirtualMachine {
	return self.backup.Printf(format, v)
}

func (self *JitVm) Endl() VirtualMachine {
	return self.backup.Endl()
}

func (self *JitVm) Env() Environment {
	return self.env
}

//go is nice
