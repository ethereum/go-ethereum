// Copyright 2026-2027, QuarkChain.

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
