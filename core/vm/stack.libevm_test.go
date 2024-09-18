package vm_test

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/core/vm"
)

func TestMutableStack(t *testing.T) {
	s := &vm.Stack{}
	m := vm.MutableStack{Stack: s}

	push := func(u uint64) uint256.Int {
		u256 := uint256.NewInt(u)
		m.Push(u256)
		return *u256
	}

	require.Empty(t, s.Data(), "new stack")
	want := []uint256.Int{push(42), push(314159), push(142857)}
	require.Equalf(t, want, s.Data(), "after pushing %d values to empty stack", len(want))
	require.Equal(t, want[len(want)-1], m.Pop(), "popped value")
	require.Equal(t, want[:len(want)-1], s.Data(), "after popping a single value")
}
