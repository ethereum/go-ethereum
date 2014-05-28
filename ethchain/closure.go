package ethchain

// TODO Re write VM to use values instead of big integers?

import (
	"github.com/ethereum/eth-go/ethutil"
	"math/big"
)

type ClosureRef interface {
	ReturnGas(*big.Int, *big.Int, *State)
	Address() []byte
	GetMem(*big.Int) *ethutil.Value
	SetStorage(*big.Int, *ethutil.Value)
	N() *big.Int
}

// Basic inline closure object which implement the 'closure' interface
type Closure struct {
	callee ClosureRef
	object *StateObject
	Script []byte
	State  *State

	Gas, UsedGas, Price *big.Int

	Args []byte
}

// Create a new closure for the given data items
func NewClosure(callee ClosureRef, object *StateObject, script []byte, state *State, gas, price *big.Int) *Closure {
	c := &Closure{callee: callee, object: object, Script: script, State: state, Args: nil}

	// In most cases gas, price and value are pointers to transaction objects
	// and we don't want the transaction's values to change.
	c.Gas = new(big.Int).Set(gas)
	c.Price = new(big.Int).Set(price)
	c.UsedGas = new(big.Int)

	return c
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
	if x.Int64() >= int64(len(c.Script)) || y.Int64() >= int64(len(c.Script)) {
		return ethutil.NewValue(0)
	}

	partial := c.Script[x.Int64() : x.Int64()+y.Int64()]

	return ethutil.NewValue(partial)
}

func (c *Closure) SetStorage(x *big.Int, val *ethutil.Value) {
	c.object.SetStorage(x, val)
}

func (c *Closure) Address() []byte {
	return c.object.Address()
}

type DebugHook func(step int, op OpCode, mem *Memory, stack *Stack, stateObject *StateObject) bool

func (c *Closure) Call(vm *Vm, args []byte, hook DebugHook) ([]byte, *big.Int, error) {
	c.Args = args

	ret, err := vm.RunClosure(c, hook)

	return ret, c.UsedGas, err
}

func (c *Closure) Return(ret []byte) []byte {
	// Return the remaining gas to the callee
	// If no callee is present return it to
	// the origin (i.e. contract or tx)
	if c.callee != nil {
		c.callee.ReturnGas(c.Gas, c.Price, c.State)
	} else {
		c.object.ReturnGas(c.Gas, c.Price, c.State)
	}

	return ret
}

func (c *Closure) UseGas(gas *big.Int) bool {
	if c.Gas.Cmp(gas) < 0 {
		return false
	}

	// Sub the amount of gas from the remaining
	c.Gas.Sub(c.Gas, gas)
	c.UsedGas.Add(c.UsedGas, gas)

	return true
}

// Implement the Callee interface
func (c *Closure) ReturnGas(gas, price *big.Int, state *State) {
	// Return the gas to the closure
	c.Gas.Add(c.Gas, gas)
	c.UsedGas.Sub(c.UsedGas, gas)
}

func (c *Closure) Object() *StateObject {
	return c.object
}

func (c *Closure) Callee() ClosureRef {
	return c.callee
}

func (c *Closure) N() *big.Int {
	return c.object.N()
}
