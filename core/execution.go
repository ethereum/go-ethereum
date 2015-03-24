package core

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
)

type Execution struct {
	env               vm.Environment
	address           *common.Address
	input             []byte
	Gas, price, value *big.Int
}

func NewExecution(env vm.Environment, address *common.Address, input []byte, gas, gasPrice, value *big.Int) *Execution {
	return &Execution{env: env, address: address, input: input, Gas: gas, price: gasPrice, value: value}
}

func (self *Execution) Call(codeAddr common.Address, caller vm.ContextRef) ([]byte, error) {
	// Retrieve the executing code
	code := self.env.State().GetCode(codeAddr)

	return self.exec(&codeAddr, code, caller)
}

func (self *Execution) exec(contextAddr *common.Address, code []byte, caller vm.ContextRef) (ret []byte, err error) {
	env := self.env
	evm := vm.NewVm(env)
	if env.Depth() == vm.MaxCallDepth {
		caller.ReturnGas(self.Gas, self.price)

		return nil, vm.DepthError{}
	}

	vsnapshot := env.State().Copy()
	if self.address == nil {
		// Generate a new address
		nonce := env.State().GetNonce(caller.Address())
		addr := crypto.CreateAddress(caller.Address(), nonce)
		self.address = &addr
		env.State().SetNonce(caller.Address(), nonce+1)
	}

	from, to := env.State().GetStateObject(caller.Address()), env.State().GetOrNewStateObject(*self.address)
	err = env.Transfer(from, to, self.value)
	if err != nil {
		env.State().Set(vsnapshot)

		caller.ReturnGas(self.Gas, self.price)

		return nil, ValueTransferErr("insufficient funds to transfer value. Req %v, has %v", self.value, from.Balance())
	}

	snapshot := env.State().Copy()
	start := time.Now()

	context := vm.NewContext(caller, to, self.value, self.Gas, self.price)
	context.SetCallCode(contextAddr, code)

	ret, err = evm.Run(context, self.input) //self.value, self.Gas, self.price, self.input)
	chainlogger.Debugf("vm took %v\n", time.Since(start))
	if err != nil {
		env.State().Set(snapshot)
	}

	return
}

func (self *Execution) Create(caller vm.ContextRef) (ret []byte, err error, account *state.StateObject) {
	ret, err = self.exec(nil, self.input, caller)
	account = self.env.State().GetStateObject(*self.address)

	return
}
