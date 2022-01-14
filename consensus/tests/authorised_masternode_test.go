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
	block450, err := insertBlock(blockchain, header)
	if err != nil {
		t.Fatal(err)
	}

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
	blockHeader := createBlock(params.TestXDPoSMockChainConfigWithV2Engine, currentBlock, blockNum, 1, blockCoinBase, signer, signFn)
	// it contains 3 master nodes
	// xdc0278C350152e15fa6FFC712a5A73D704Ce73E2E1
	// xdc03d9e17Ae3fF2c6712E44e25B09Ac5ee91f6c9ff
	// xdc065551F0dcAC6f00CAe11192D462db709bE3758c
	blockHeader.Validators = common.Hex2Bytes("0278c350152e15fa6ffc712a5a73d704ce73e2e103d9e17ae3ff2c6712e44e25b09ac5ee91f6c9ff065551f0dcac6f00cae11192d462db709be3758c")
	// block 11 is the first v2 block, and is treated as epoch switch block
	currentBlock, err := insertBlock(blockchain, blockHeader)
	if err != nil {
		t.Fatal(err)
	}

	// the first block will start from 1
	isAuthorisedMN := adaptor.IsAuthorisedAddress(blockchain, currentBlock.Header(), common.HexToAddress("xdc03d9e17Ae3fF2c6712E44e25B09Ac5ee91f6c9ff"))
	assert.True(t, isAuthorisedMN)
	// The third address hence not valid
	isAuthorisedMN = adaptor.IsAuthorisedAddress(blockchain, currentBlock.Header(), common.HexToAddress("xdc065551F0dcAC6f00CAe11192D462db709bE3758c"))
	assert.False(t, isAuthorisedMN)

	for blockNum = 12; blockNum < 16; blockNum++ {
		blockHeader = createBlock(params.TestXDPoSMockChainConfigWithV2Engine, currentBlock, blockNum, int64(blockNum-10), blockCoinBase, signer, signFn)
		currentBlock, err = insertBlock(blockchain, blockHeader)
		if err != nil {
			t.Fatal(err)
		}
	}
	isAuthorisedMN = adaptor.IsAuthorisedAddress(blockchain, currentBlock.Header(), common.HexToAddress("xdc065551F0dcAC6f00CAe11192D462db709bE3758c"))
	assert.True(t, isAuthorisedMN)
}
