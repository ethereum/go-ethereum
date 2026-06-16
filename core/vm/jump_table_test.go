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
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func petersburgOnlyChainConfig() *params.ChainConfig {
	return &params.ChainConfig{
		ChainID:             big.NewInt(1),
		HomesteadBlock:      big.NewInt(0),
		DAOForkBlock:        big.NewInt(0),
		DAOForkSupport:      false,
		EIP150Block:         big.NewInt(0),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PetersburgBlock:     big.NewInt(0),
	}
}

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

func TestPetersburgOnlyInstructionSet(t *testing.T) {
	random := common.HexToHash("0xffff")
	evm := NewEVM(BlockContext{
		BlockNumber: big.NewInt(0),
		Difficulty:  big.NewInt(7),
		Random:      &random,
	}, nil, petersburgOnlyChainConfig(), Config{})
	defer evm.Release()

	require.True(t, evm.chainRules.IsPetersburg)
	require.False(t, evm.chainRules.IsMerge)
	require.True(t, evm.table[CHAINID].undefined)
	require.True(t, evm.table[BASEFEE].undefined)
	require.True(t, evm.table[PUSH0].undefined)

	stack := newStackForTesting()
	pc := uint64(0)
	_, err := evm.table[DIFFICULTY].execute(&pc, evm, &ScopeContext{Stack: stack})
	require.NoError(t, err)
	actual := stack.pop()
	require.Equal(t, uint64(7), actual.Uint64())
}
