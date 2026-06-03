// Copyright 2026 The go-ethereum Authors
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

package core

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

type evmBlockContextChain struct {
	config *params.ChainConfig
}

func (c evmBlockContextChain) Config() *params.ChainConfig                 { return c.config }
func (c evmBlockContextChain) CurrentHeader() *types.Header                { return nil }
func (c evmBlockContextChain) GetHeader(common.Hash, uint64) *types.Header { return nil }
func (c evmBlockContextChain) GetHeaderByNumber(uint64) *types.Header      { return nil }
func (c evmBlockContextChain) GetHeaderByHash(common.Hash) *types.Header   { return nil }
func (c evmBlockContextChain) Engine() consensus.Engine                    { return nil }

func TestNewEVMBlockContextRandomRequiresPostMerge(t *testing.T) {
	author := common.HexToAddress("0x1")
	random := common.HexToHash("0x1234")
	header := &types.Header{
		Number:     big.NewInt(1),
		Difficulty: big.NewInt(0),
		MixDigest:  random,
	}

	legacy := NewEVMBlockContext(header, evmBlockContextChain{config: params.QuarkChainHistoryChainConfig}, &author)
	require.Nil(t, legacy.Random)

	merged := NewEVMBlockContext(header, evmBlockContextChain{config: params.MergedTestChainConfig}, &author)
	require.NotNil(t, merged.Random)
	require.Equal(t, random, *merged.Random)
}
