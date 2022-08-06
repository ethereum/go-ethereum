package engine_v1_tests

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
	blockchain, _, parentBlock, signer, signFn := PrepareXDCTestBlockChain(t, GAP-2, params.TestXDPoSMockChainConfig)
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
	block449, err := createBlockFromHeader(blockchain, header, []*types.Transaction{tx}, signer, signFn, blockchain.Config())
	assert.Nil(t, err)
	err = blockchain.InsertBlock(block449)
	assert.Nil(t, err)
	if err != nil {
		t.Fatal(err)
	}
	parentBlock = block449

	// At block 449, we should not update signerList. we need to update it till block 450 gap block.
	// Acc3 is the default account that is on the signerList

	engine := blockchain.Engine().(*XDPoS.XDPoS)
	isAuthorisedMN := engine.IsAuthorisedAddress(blockchain, block449.Header(), acc3Addr)
	assert.True(t, isAuthorisedMN)

	isAuthorisedMN = engine.IsAuthorisedAddress(blockchain, block449.Header(), acc1Addr)
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
	block450, err := createBlockFromHeader(blockchain, header, nil, signer, signFn, blockchain.Config())
	if err != nil {
		t.Fatal(err)
	}
	err = blockchain.InsertBlock(block450)
	assert.Nil(t, err)

	isAuthorisedMN = engine.IsAuthorisedAddress(blockchain, block450.Header(), acc3Addr)
	assert.False(t, isAuthorisedMN)

	isAuthorisedMN = engine.IsAuthorisedAddress(blockchain, block450.Header(), acc1Addr)
	assert.True(t, isAuthorisedMN)
}
