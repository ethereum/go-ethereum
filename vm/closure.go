package vm

import (
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/state"
)

type ClosureRef interface {
	ReturnGas(*big.Int, *big.Int)
	Address() []byte
	SetCode([]byte)
}

type Closure struct {
	caller  ClosureRef
	object  ClosureRef
	Code    []byte
	message *state.Message

	Gas, UsedGas, Price *big.Int

	Args []byte
}

// Create a new closure for the given data items
func NewClosure(msg *state.Message, caller ClosureRef, object ClosureRef, code []byte, gas, price *big.Int) *Closure {
	c := &Closure{message: msg, caller: caller, object: object, Code: code, Args: nil}

	// Gas should be a pointer so it can safely be reduced through the run
	// This pointer will be off the state transition
	c.Gas = gas //new(big.Int).Set(gas)
	// In most cases price and value are pointers to transaction objects
	// and we don't want the transaction's values to change.
	c.Price = new(big.Int).Set(price)
	c.UsedGas = new(big.Int)

	return c
}

func (c *Closure) GetOp(x uint64) OpCode {
	return OpCode(c.GetByte(x))
}

func (c *Closure) GetByte(x uint64) byte {
	if x < uint64(len(c.Code)) {
		return c.Code[x]
	}

	return 0
}

func (c *Closure) GetBytes(x, y int) []byte {
	return c.GetRangeValue(uint64(x), uint64(y))
}

func (c *Closure) GetRangeValue(x, size uint64) []byte {
	x = uint64(math.Min(float64(x), float64(len(c.Code))))
	y := uint64(math.Min(float64(x+size), float64(len(c.Code))))

	return ethutil.LeftPadBytes(c.Code[x:y], int(size))
}

func (c *Closure) Return(ret []byte) []byte {
	// Return the remaining gas to the caller
	c.caller.ReturnGas(c.Gas, c.Price)

	return ret
}

/*
 * Gas functions
 */
func (c *Closure) UseGas(gas *big.Int) bool {
	if c.Gas.Cmp(gas) < 0 {
		return false
	}

	// Sub the amount of gas from the remaining
	c.Gas.Sub(c.Gas, gas)
	c.UsedGas.Add(c.UsedGas, gas)

	return true
}

// Implement the caller interface
func (c *Closure) ReturnGas(gas, price *big.Int) {
	// Return the gas to the closure
	c.Gas.Add(c.Gas, gas)
	c.UsedGas.Sub(c.UsedGas, gas)
}

/*
 * Set / Get
 */
func (c *Closure) Address() []byte {
	return c.object.Address()
}

func (self *Closure) SetCode(code []byte) {
	self.Code = code
}
