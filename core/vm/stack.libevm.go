package vm

import "github.com/holiman/uint256"

// A MutableStack embeds a Stack to expose unexported mutation methods.
type MutableStack struct {
	*Stack
}

// Push pushes a value to the stack.
func (s MutableStack) Push(d *uint256.Int) { s.Stack.push(d) }

// Pop pops a value from the stack.
func (s MutableStack) Pop() uint256.Int { return s.Stack.pop() }
