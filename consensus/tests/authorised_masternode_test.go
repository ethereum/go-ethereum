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
	block449, err := createBlockFromHeader(blockchain, header, []*types.Transaction{tx})
	blockchain.InsertBlock(block449)
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
	block450, err := createBlockFromHeader(blockchain, header, nil)
	if err != nil {
		t.Fatal(err)
	}
	blockchain.InsertBlock(block450)

	isAuthorisedMN = engine.IsAuthorisedAddress(blockchain, block450.Header(), acc3Addr)
	assert.False(t, isAuthorisedMN)

	isAuthorisedMN = engine.IsAuthorisedAddress(blockchain, block450.Header(), acc1Addr)
	assert.True(t, isAuthorisedMN)
}

func TestIsAuthorisedMNForConsensusV2(t *testing.T) {
	// we skip test for v1 since it's hard to make a real genesis block
	blockchain, _, currentBlock, signer, signFn, _ := PrepareXDCTestBlockChainForV2Engine(t, 10, params.TestXDPoSMockChainConfigWithV2Engine, 0)
	adaptor := blockchain.Engine().(*XDPoS.XDPoS)
	blockNum := 11
	blockCoinBase := "0x111000000000000000000000000000000123"
	currentBlock = CreateBlock(blockchain, params.TestXDPoSMockChainConfigWithV2Engine, currentBlock, blockNum, 1, blockCoinBase, signer, signFn)
	blockchain.InsertBlock(currentBlock)

	// As long as the address is in the master node list, they are all valid
	isAuthorisedMN := adaptor.IsAuthorisedAddress(blockchain, currentBlock.Header(), common.HexToAddress("xdc0D3ab14BBaD3D99F4203bd7a11aCB94882050E7e"))
	assert.True(t, isAuthorisedMN)

	isAuthorisedMN = adaptor.IsAuthorisedAddress(blockchain, currentBlock.Header(), common.HexToAddress("xdc71562b71999873DB5b286dF957af199Ec94617F7"))
	assert.True(t, isAuthorisedMN)

	isAuthorisedMN = adaptor.IsAuthorisedAddress(blockchain, currentBlock.Header(), common.HexToAddress("xdcbanana"))
	assert.False(t, isAuthorisedMN)
}

func TestIsYourTurnConsensusV2(t *testing.T) {
	// we skip test for v1 since it's hard to make a real genesis block
	blockchain, _, currentBlock, signer, signFn, _ := PrepareXDCTestBlockChainForV2Engine(t, 10, params.TestXDPoSMockChainConfigWithV2Engine, 0)

	adaptor := blockchain.Engine().(*XDPoS.XDPoS)
	blockNum := 11
	blockCoinBase := "0x111000000000000000000000000000000123"
	currentBlock = CreateBlock(blockchain, params.TestXDPoSMockChainConfigWithV2Engine, currentBlock, blockNum, 1, blockCoinBase, signer, signFn)
	blockchain.InsertBlock(currentBlock)

	// The first address is valid
	isYourTurn, err := adaptor.YourTurn(blockchain, currentBlock.Header(), common.HexToAddress("xdc703c4b2bD70c169f5717101CaeE543299Fc946C7"))
	assert.Nil(t, err)
	assert.True(t, isYourTurn)

	// The second and third address are not valid
	isYourTurn, err = adaptor.YourTurn(blockchain, currentBlock.Header(), common.HexToAddress("xdc0D3ab14BBaD3D99F4203bd7a11aCB94882050E7e"))
	assert.Nil(t, err)
	assert.False(t, isYourTurn)
	isYourTurn, err = adaptor.YourTurn(blockchain, currentBlock.Header(), common.HexToAddress("xdc71562b71999873DB5b286dF957af199Ec94617F7"))
	assert.Nil(t, err)
	assert.False(t, isYourTurn)

	// We continue to grow the chain which will increase the round number
	blockNum = 12
	currentBlock = CreateBlock(blockchain, params.TestXDPoSMockChainConfigWithV2Engine, currentBlock, blockNum, int64(blockNum-10), blockCoinBase, signer, signFn)
	blockchain.InsertBlock(currentBlock)

	adaptor.EngineV2.SetNewRoundFaker(1, false)
	isYourTurn, _ = adaptor.YourTurn(blockchain, currentBlock.Header(), common.HexToAddress("xdc703c4b2bD70c169f5717101CaeE543299Fc946C7"))
	assert.False(t, isYourTurn)

	isYourTurn, _ = adaptor.YourTurn(blockchain, currentBlock.Header(), common.HexToAddress("xdc0D3ab14BBaD3D99F4203bd7a11aCB94882050E7e"))
	assert.True(t, isYourTurn)

	isYourTurn, _ = adaptor.YourTurn(blockchain, currentBlock.Header(), common.HexToAddress("xdc71562b71999873DB5b286dF957af199Ec94617F7"))
	assert.False(t, isYourTurn)

}
