package core

import (
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/state"
	"github.com/ethereum/go-ethereum/vm"
)

type Execution struct {
	env               vm.Environment
	address, input    []byte
	Gas, price, value *big.Int
	SkipTransfer      bool
}

func NewExecution(env vm.Environment, address, input []byte, gas, gasPrice, value *big.Int) *Execution {
	return &Execution{env: env, address: address, input: input, Gas: gas, price: gasPrice, value: value}
}

func (self *Execution) Addr() []byte {
	return self.address
}

func (self *Execution) Call(codeAddr []byte, caller vm.ContextRef) ([]byte, error) {
	// Retrieve the executing code
	code := self.env.State().GetCode(codeAddr)

	return self.exec(code, codeAddr, caller)
}

func (self *Execution) exec(code, contextAddr []byte, caller vm.ContextRef) (ret []byte, err error) {
	env := self.env
	evm := vm.New(env, vm.DebugVmTy)

	if env.Depth() == vm.MaxCallDepth {
		// Consume all gas (by not returning it) and return a depth error
		return nil, vm.DepthError{}
	}

	from, to := env.State().GetStateObject(caller.Address()), env.State().GetOrNewStateObject(self.address)
	// Skipping transfer is used on testing for the initial call
	if !self.SkipTransfer {
		err = env.Transfer(from, to, self.value)
		if err != nil {
			caller.ReturnGas(self.Gas, self.price)

			err = fmt.Errorf("Insufficient funds to transfer value. Req %v, has %v", self.value, from.Balance)
			return
		}
	}

	snapshot := env.State().Copy()
	start := time.Now()
	ret, err = evm.Run(to, caller, code, self.value, self.Gas, self.price, self.input)
	if err != nil {
		env.State().Set(snapshot)
	}
	chainlogger.Debugf("vm took %v\n", time.Since(start))

	return
}

func (self *Execution) Create(caller vm.ContextRef) (ret []byte, err error, account *state.StateObject) {
	ret, err = self.exec(self.input, nil, caller)
	account = self.env.State().GetStateObject(self.address)

	return
}
