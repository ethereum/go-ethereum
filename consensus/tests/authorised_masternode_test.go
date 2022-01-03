package tests

import (
	"math/big"
	"testing"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/stretchr/testify/assert"
)

func TestIsAuthorisedMNForConsensusV1(t *testing.T) {
	/*
		V1 consensus engine
	*/
	blockchain, _, parentBlock, _ := PrepareXDCTestBlockChain(t, GAP-2, params.TestXDPoSMockChainConfig)
	// Insert first Block 449
	t.Logf("Inserting block with propose at 449...")
	blockCoinbaseA := "0xaaa0000000000000000000000000000000000449"
	tx, err := voteTX(37117, 0, acc1Addr.String())
	if err != nil {
		t.Fatal(err)
	}

	//Get from block validator error message
	merkleRoot := "46234e9cd7e85a267f7f0435b15256a794a2f6d65cc98cdbd21dcd10a01d9772"
	header := &types.Header{
		Root:       common.HexToHash(merkleRoot),
		Number:     big.NewInt(int64(449)),
		ParentHash: parentBlock.Hash(),
		Coinbase:   common.HexToAddress(blockCoinbaseA),
	}
	block449, err := insertBlockTxs(blockchain, header, []*types.Transaction{tx})
	if err != nil {
		t.Fatal(err)
	}
	parentBlock = block449

	// At block 449, we should not update signerList. we need to update it till block 450 gap block.
	// Acc3 is the default account that is on the signerList

	engine := blockchain.Engine().(*XDPoS.XDPoS)
	isAuthorisedMN := engine.IsAuthorisedAddress(block449.Header(), blockchain, acc3Addr)
	assert.True(t, isAuthorisedMN)

	isAuthorisedMN = engine.IsAuthorisedAddress(block449.Header(), blockchain, acc1Addr)
	assert.False(t, isAuthorisedMN)

	// Now, let's mine another block to trigger the GAP block signerList update
	block450CoinbaseAddress := "0xaaa0000000000000000000000000000000000450"
	merkleRoot = "46234e9cd7e85a267f7f0435b15256a794a2f6d65cc98cdbd21dcd10a01d9772"
	header = &types.Header{
		Root:       common.HexToHash(merkleRoot),
		Number:     big.NewInt(int64(450)),
		ParentHash: parentBlock.Hash(),
		Coinbase:   common.HexToAddress(block450CoinbaseAddress),
	}
	block450, err := insertBlock(blockchain, header)
	if err != nil {
		t.Fatal(err)
	}

	isAuthorisedMN = engine.IsAuthorisedAddress(block450.Header(), blockchain, acc3Addr)
	assert.False(t, isAuthorisedMN)

	isAuthorisedMN = engine.IsAuthorisedAddress(block450.Header(), blockchain, acc1Addr)
	assert.True(t, isAuthorisedMN)
}
