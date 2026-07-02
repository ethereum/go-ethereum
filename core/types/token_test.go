// Copyright 2024 The go-ethereum Authors
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

package types

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenBalancesListFormat(t *testing.T) {
	tb := NewEmptyTokenBalances()
	tb.SetValue(uint256.NewInt(1000), DefaultTokenID)
	tb.SetValue(uint256.NewInt(500), 100)

	data, err := tb.SerializeToBytes()
	require.NoError(t, err)
	assert.Equal(t, byte(0x00), data[0], "list format prefix")

	tb2, err := NewTokenBalancesFromBytes(data)
	require.NoError(t, err)
	assert.Equal(t, uint256.NewInt(1000), tb2.GetTokenBalance(DefaultTokenID))
	assert.Equal(t, uint256.NewInt(500), tb2.GetTokenBalance(100))
}

func TestTokenBalancesIsBlank(t *testing.T) {
	tb := NewEmptyTokenBalances()
	assert.True(t, tb.IsBlank())
	tb.SetValue(uint256.NewInt(0), DefaultTokenID)
	assert.True(t, tb.IsBlank(), "zero balance is blank")
	tb.SetValue(uint256.NewInt(1), DefaultTokenID)
	assert.False(t, tb.IsBlank())
}

// TestTokenBalancesTrieFormatUnsupported verifies that serializing more than
// TokenTrieThreshold (16) non-zero token balances returns an error, since the
// trie format (0x01) is not implemented.
func TestTokenBalancesTrieFormatUnsupported(t *testing.T) {
	tb := NewEmptyTokenBalances()
	for i := uint64(1); i <= 17; i++ {
		tb.SetValue(uint256.NewInt(i*100), i)
	}
	_, err := tb.SerializeToBytes()
	require.Error(t, err, "expected error for >16 token balances (trie format unsupported)")
}

// TestTokenBalancesFromBytesTrieFormatUnsupported verifies that deserializing
// a 0x01-prefixed (trie format) byte slice returns an error.
func TestTokenBalancesFromBytesTrieFormatUnsupported(t *testing.T) {
	data := make([]byte, 33)
	data[0] = 0x01 // trie format prefix
	_, err := NewTokenBalancesFromBytes(data)
	require.Error(t, err, "expected error for trie format (0x01) deserialization")
}

func TestTokenBalancesCopy(t *testing.T) {
	tb := NewTokenBalancesWithMap(map[uint64]*uint256.Int{
		DefaultTokenID: uint256.NewInt(1e18),
		100:   uint256.NewInt(500),
	})
	cp := tb.Copy()
	cp.SetValue(uint256.NewInt(0), DefaultTokenID)
	assert.Equal(t, uint256.NewInt(1e18), tb.GetTokenBalance(DefaultTokenID), "original unaffected")
}
