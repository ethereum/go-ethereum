package ethchain

// TODO Re write VM to use values instead of big integers?

import (
	"github.com/ethereum/eth-go/ethutil"
	"math/big"
)

type Callee interface {
	ReturnGas(*big.Int, *State)
	Address() []byte
}

type Reference interface {
	Callee
	ethutil.RlpEncodable
	GetMem(*big.Int) *ethutil.Value
	SetMem(*big.Int, *ethutil.Value)
}

// Basic inline closure object which implement the 'closure' interface
type Closure struct {
	callee Callee
	object Reference
	Script []byte
	State  *State

	Gas   *big.Int
	Value *big.Int

	Args []byte
}

// Create a new closure for the given data items
func NewClosure(callee Callee, object Reference, script []byte, state *State, gas, val *big.Int) *Closure {
	return &Closure{callee, object, script, state, gas, val, nil}
}

// Retuns the x element in data slice
func (c *Closure) GetMem(x *big.Int) *ethutil.Value {
	m := c.object.GetMem(x)
	if m == nil {
		return ethutil.EmptyValue()
	}

	return m
}

func (c *Closure) Get(x *big.Int) *ethutil.Value {
	return c.Gets(x, big.NewInt(1))
}

func (c *Closure) Gets(x, y *big.Int) *ethutil.Value {
	if x.Int64() > int64(len(c.Script)) || y.Int64() > int64(len(c.Script)) {
		return ethutil.NewValue(0)
	}

	partial := c.Script[x.Int64() : x.Int64()+y.Int64()]

	return ethutil.NewValue(partial)
}

func (c *Closure) SetMem(x *big.Int, val *ethutil.Value) {
	c.object.SetMem(x, val)
}

func (c *Closure) Address() []byte {
	return c.object.Address()
}

type DebugHook func(op OpCode)

func (c *Closure) Call(vm *Vm, args []byte, hook DebugHook) []byte {
	c.Args = args

	return vm.RunClosure(c, hook)
}

func (c *Closure) Return(ret []byte) []byte {
	// Return the remaining gas to the callee
	// If no callee is present return it to
	// the origin (i.e. contract or tx)
	if c.callee != nil {
		c.callee.ReturnGas(c.Gas, c.State)
	} else {
		c.object.ReturnGas(c.Gas, c.State)
		// TODO incase it's a POST contract we gotta serialise the contract again.
		// But it's not yet defined
	}

	return ret
}

// Implement the Callee interface
func (c *Closure) ReturnGas(gas *big.Int, state *State) {
	// Return the gas to the closure
	c.Gas.Add(c.Gas, gas)
}

func (c *Closure) Object() Reference {
	return c.object
}

func (c *Closure) Callee() Callee {
	return c.callee
}
