package vm

import (
	"fmt"
	"math/big"

	"github.com/ethereum/eth-go/ethstate"
	"github.com/ethereum/eth-go/ethutil"
)

type Execution struct {
	vm                VirtualMachine
	address, input    []byte
	Gas, price, value *big.Int
	object            *ethstate.StateObject
	SkipTransfer      bool
}

func NewExecution(vm VirtualMachine, address, input []byte, gas, gasPrice, value *big.Int) *Execution {
	return &Execution{vm: vm, address: address, input: input, Gas: gas, price: gasPrice, value: value}
}

func (self *Execution) Addr() []byte {
	return self.address
}

func (self *Execution) Exec(codeAddr []byte, caller ClosureRef) ([]byte, error) {
	// Retrieve the executing code
	code := self.vm.Env().State().GetCode(codeAddr)

	return self.exec(code, codeAddr, caller)
}

func (self *Execution) exec(code, caddr []byte, caller ClosureRef) (ret []byte, err error) {
	env := self.vm.Env()

	snapshot := env.State().Copy()
	defer func() {
		if IsDepthErr(err) || IsOOGErr(err) {
			env.State().Set(snapshot)
		}
	}()

	msg := env.State().Manifest().AddMessage(&ethstate.Message{
		To: self.address, From: caller.Address(),
		Input:  self.input,
		Origin: env.Origin(),
		Block:  env.BlockHash(), Timestamp: env.Time(), Coinbase: env.Coinbase(), Number: env.BlockNumber(),
		Value: self.value,
	})

	from, to := caller.Object(), env.State().GetOrNewStateObject(self.address)
	// Skipping transfer is used on testing for the initial call
	if !self.SkipTransfer {
		err = env.Transfer(from, to, self.value)
	}

	if err != nil {
		caller.ReturnGas(self.Gas, self.price)

		err = fmt.Errorf("Insufficient funds to transfer value. Req %v, has %v", self.value, from.Balance)
	} else {
		self.object = to
		// Pre-compiled contracts (address.go) 1, 2 & 3.
		naddr := ethutil.BigD(caddr).Uint64()
		if p := Precompiled[naddr]; p != nil {
			if self.Gas.Cmp(p.Gas) >= 0 {
				ret = p.Call(self.input)
				self.vm.Printf("NATIVE_FUNC(%x) => %x", naddr, ret)
			}
		} else {
			// Create a new callable closure
			c := NewClosure(msg, caller, to, code, self.Gas, self.price)
			c.exe = self

			if self.vm.Depth() == MaxCallDepth {
				c.UseGas(self.Gas)

				return c.Return(nil), DepthError{}
			}

			// Executer the closure and get the return value (if any)
			ret, _, err = c.Call(self.vm, self.input)
			msg.Output = ret
		}
	}

	return
}

func (self *Execution) Create(caller ClosureRef) (ret []byte, err error) {
	return self.exec(self.input, nil, caller)
}
