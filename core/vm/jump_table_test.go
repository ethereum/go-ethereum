// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package vm

import (
	"testing"

	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

// TestJumpTableCopy tests that deep copy is necessary to prevent modify shared jump table
func TestJumpTableCopy(t *testing.T) {
	tbl := newMergeInstructionSet()
	require.Equal(t, uint64(0), tbl[SLOAD].constantGas)

	// a deep copy won't modify the shared jump table
	deepCopy := copyJumpTable(&tbl)
	deepCopy[SLOAD].constantGas = 100
	require.Equal(t, uint64(100), deepCopy[SLOAD].constantGas)
	require.Equal(t, uint64(0), tbl[SLOAD].constantGas)
}

func TestLookupInstructionSetAmsterdam(t *testing.T) {
	jt, err := LookupInstructionSet(params.Rules{
		IsMerge:     true,
		IsShanghai:  true,
		IsCancun:    true,
		IsPrague:    true,
		IsOsaka:     true,
		IsAmsterdam: true,
	})
	require.NoError(t, err)

	require.True(t, jt[SLOTNUM].HasCost())
	require.True(t, jt[DUPN].HasCost())
	require.True(t, jt[SWAPN].HasCost())
	require.True(t, jt[EXCHANGE].HasCost())
}

func TestAmsterdamOpcodeActivation(t *testing.T) {
	amsterdam := newAmsterdamInstructionSet()
	osaka := newOsakaInstructionSet()

	require.True(t, amsterdam[SLOTNUM].HasCost())
	require.True(t, amsterdam[DUPN].HasCost())
	require.True(t, amsterdam[SWAPN].HasCost())
	require.True(t, amsterdam[EXCHANGE].HasCost())

	require.False(t, osaka[SLOTNUM].HasCost())
	require.False(t, osaka[DUPN].HasCost())
	require.False(t, osaka[SWAPN].HasCost())
	require.False(t, osaka[EXCHANGE].HasCost())
}
