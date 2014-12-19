package core

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/state"
	"github.com/ethereum/go-ethereum/vm"
)

type Execution struct {
	env               vm.Environment
	address, input    []byte
	Gas, price, value *big.Int
	object            *state.StateObject
	SkipTransfer      bool
}

func NewExecution(env vm.Environment, address, input []byte, gas, gasPrice, value *big.Int) *Execution {
	return &Execution{env: env, address: address, input: input, Gas: gas, price: gasPrice, value: value}
}

func (self *Execution) Addr() []byte {
	return self.address
}

func (self *Execution) Call(codeAddr []byte, caller vm.ClosureRef) ([]byte, error) {
	// Retrieve the executing code
	code := self.env.State().GetCode(codeAddr)

	return self.exec(code, codeAddr, caller)
}

func (self *Execution) exec(code, contextAddr []byte, caller vm.ClosureRef) (ret []byte, err error) {
	env := self.env
	evm := vm.New(env, vm.DebugVmTy)

	chainlogger.Debugf("pre state %x\n", env.State().Root())

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
	defer func() {
		env.State().Set(snapshot)
		chainlogger.Debugf("post state %x\n", env.State().Root())
	}()

	self.object = to
	ret, err = evm.Run(to, caller, code, self.value, self.Gas, self.price, self.input)

	return
}

func (self *Execution) Create(caller vm.ClosureRef) (ret []byte, err error, account *state.StateObject) {
	ret, err = self.exec(self.input, nil, caller)
	account = self.env.State().GetStateObject(self.address)

	return
}
