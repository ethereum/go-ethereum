package vm

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestJumpTableCopy tests that deep copy is necessery to prevent modify shared jump table
func TestJumpTableCopy(t *testing.T) {
	tbl := newMergeInstructionSet()
	require.Equal(t, uint64(0), tbl[SLOAD].constantGas)

	// a deep copy won't modify the shared jump table
	deepCopy := copyJumpTable(&tbl)
	deepCopy[SLOAD].constantGas = 100
	require.Equal(t, uint64(100), deepCopy[SLOAD].constantGas)
	require.Equal(t, uint64(0), tbl[SLOAD].constantGas)

	// but a shallow copy will modify the shared table
	shallowCopy := tbl
	shallowCopy[SLOAD].constantGas = 100
	require.Equal(t, uint64(100), shallowCopy[SLOAD].constantGas)
	require.Equal(t, uint64(100), tbl[SLOAD].constantGas)
}
