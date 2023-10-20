// Copyright 2022 The go-ethereum Authors
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

package vm_test

import (
	"testing"

	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/stretchr/testify/require"
)

// TestJumpTableCopy tests that deep copy is necessery to prevent modify shared jump table
func TestJumpTableCopy(t *testing.T) {
	tbl := vm.NewMergeInstructionSet()
	require.Equal(t, uint64(0), tbl[vm.SLOAD].ConstantGas)

	// a deep copy won't modify the shared jump table
	deepCopy := vm.CopyJumpTable(&tbl)
	deepCopy[vm.SLOAD].ConstantGas = 100
	require.Equal(t, uint64(100), deepCopy[vm.SLOAD].ConstantGas)
	require.Equal(t, uint64(0), tbl[vm.SLOAD].ConstantGas)
}
