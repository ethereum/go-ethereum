package ethchain

// TODO Re write VM to use values instead of big integers?

import (
	"github.com/ethereum/eth-go/ethutil"
	"math/big"
)

type Callee interface {
	ReturnGas(*big.Int, *State)
}

type ClosureBody interface {
	Callee
	ethutil.RlpEncodable
	GetMem(int64) *ethutil.Value
}

// Basic inline closure object which implement the 'closure' interface
type Closure struct {
	callee Callee
	object ClosureBody
	State  *State

	gas *big.Int
	val *big.Int

	Args []byte
}

// Create a new closure for the given data items
func NewClosure(callee Callee, object ClosureBody, state *State, gas, val *big.Int) *Closure {
	return &Closure{callee, object, state, gas, val, nil}
}

// Retuns the x element in data slice
func (c *Closure) GetMem(x int64) *ethutil.Value {
	m := c.object.GetMem(x)
	if m == nil {
		return ethutil.EmptyValue()
	}

	return m
}

func (c *Closure) Call(vm *Vm, args []byte) []byte {
	c.Args = args

	return vm.RunClosure(c)
}

func (c *Closure) Return(ret []byte) []byte {
	// Return the remaining gas to the callee
	// If no callee is present return it to
	// the origin (i.e. contract or tx)
	if c.callee != nil {
		c.callee.ReturnGas(c.gas, c.State)
	} else {
		c.object.ReturnGas(c.gas, c.State)
		// TODO incase it's a POST contract we gotta serialise the contract again.
		// But it's not yet defined
	}

	return ret
}

// Implement the Callee interface
func (c *Closure) ReturnGas(gas *big.Int, state *State) {
	// Return the gas to the closure
	c.gas.Add(c.gas, gas)
}

func (c *Closure) GetGas() *big.Int {
	return c.gas
}
