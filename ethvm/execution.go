package ethvm

import (
	"fmt"
	"math/big"

	"github.com/ethereum/eth-go/ethstate"
	"github.com/ethereum/eth-go/ethutil"
)

type Execution struct {
	vm                VirtualMachine
	closure           *Closure
	address, input    []byte
	gas, price, value *big.Int
	object            *ethstate.StateObject
}

func NewExecution(vm VirtualMachine, address, input []byte, gas, gasPrice, value *big.Int) *Execution {
	return &Execution{vm: vm, address: address, input: input, gas: gas, price: gasPrice, value: value}
}

func (self *Execution) Addr() []byte {
	return self.address
}

func (self *Execution) Exec(codeAddr []byte, caller ClosureRef) (ret []byte, err error) {
	env := self.vm.Env()

	snapshot := env.State().Copy()

	msg := env.State().Manifest().AddMessage(&ethstate.Message{
		To: self.address, From: caller.Address(),
		Input:  self.input,
		Origin: env.Origin(),
		Block:  env.BlockHash(), Timestamp: env.Time(), Coinbase: env.Coinbase(), Number: env.BlockNumber(),
		Value: self.value,
	})

	object := caller.Object()
	if object.Balance.Cmp(self.value) < 0 {
		caller.ReturnGas(self.gas, self.price)

		err = fmt.Errorf("Insufficient funds to transfer value. Req %v, has %v", self.value, object.Balance)
	} else {
		stateObject := env.State().GetOrNewStateObject(self.address)
		self.object = stateObject

		caller.Object().SubAmount(self.value)
		stateObject.AddAmount(self.value)

		// Precompiled contracts (address.go) 1, 2 & 3.
		naddr := ethutil.BigD(codeAddr).Uint64()
		if p := Precompiled[naddr]; p != nil {
			if self.gas.Cmp(p.Gas) >= 0 {
				ret = p.Call(self.input)
				self.vm.Printf("NATIVE_FUNC(%x) => %x", naddr, ret)
			}
		} else {
			if self.vm.Depth() == MaxCallDepth {
				return nil, fmt.Errorf("Max call depth exceeded (%d)", MaxCallDepth)
			}

			// Retrieve the executing code
			code := env.State().GetCode(codeAddr)

			// Create a new callable closure
			c := NewClosure(msg, caller, stateObject, code, self.gas, self.price)
			c.exe = self
			// Executer the closure and get the return value (if any)
			ret, _, err = c.Call(self.vm, self.input)

			msg.Output = ret
		}
	}

	if err != nil {
		env.State().Set(snapshot)
	}

	return
}
