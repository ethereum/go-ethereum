package core

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/state"
	"github.com/ethereum/go-ethereum/vm"
)

type Execution struct {
	vm                vm.VirtualMachine
	address, input    []byte
	Gas, price, value *big.Int
	object            *state.StateObject
	SkipTransfer      bool
}

func NewExecution(vm vm.VirtualMachine, address, input []byte, gas, gasPrice, value *big.Int) *Execution {
	return &Execution{vm: vm, address: address, input: input, Gas: gas, price: gasPrice, value: value}
}

func (self *Execution) Addr() []byte {
	return self.address
}

func (self *Execution) Call(codeAddr []byte, caller vm.ClosureRef) ([]byte, error) {
	// Retrieve the executing code
	code := self.vm.Env().State().GetCode(codeAddr)

	return self.exec(code, codeAddr, caller)
}

func (self *Execution) exec(code, caddr []byte, caller vm.ClosureRef) (ret []byte, err error) {
	env := self.vm.Env()
	chainlogger.Debugf("pre state %x\n", env.State().Root())

	from, to := env.State().GetStateObject(caller.Address()), env.State().GetOrNewStateObject(self.address)
	// Skipping transfer is used on testing for the initial call
	if !self.SkipTransfer {
		err = env.Transfer(from, to, self.value)
	}

	snapshot := env.State().Copy()
	defer func() {
		if vm.IsDepthErr(err) || vm.IsOOGErr(err) {
			env.State().Set(snapshot)
		}
		chainlogger.Debugf("post state %x\n", env.State().Root())
	}()

	if err != nil {
		caller.ReturnGas(self.Gas, self.price)

		err = fmt.Errorf("Insufficient funds to transfer value. Req %v, has %v", self.value, from.Balance)
	} else {
		self.object = to
		// Pre-compiled contracts (address.go) 1, 2 & 3.
		naddr := ethutil.BigD(caddr).Uint64()
		if p := vm.Precompiled[naddr]; p != nil {
			if self.Gas.Cmp(p.Gas(len(self.input))) >= 0 {
				ret = p.Call(self.input)
				self.vm.Printf("NATIVE_FUNC(%x) => %x", naddr, ret)
				self.vm.Endl()
			}
		} else {
			ret, err = self.vm.Run(to, caller, code, self.value, self.Gas, self.price, self.input)
		}
	}

	return
}

func (self *Execution) Create(caller vm.ClosureRef) (ret []byte, err error, account *state.StateObject) {
	ret, err = self.exec(self.input, nil, caller)
	account = self.vm.Env().State().GetStateObject(self.address)

	return
}
