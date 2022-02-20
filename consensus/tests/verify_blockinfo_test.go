package tests

import (
	"fmt"
	"testing"

	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/stretchr/testify/assert"
)

func TestShouldVerifyBlockInfo(t *testing.T) {
	// Block 901 is the first v2 block with round of 1
	blockchain, _, currentBlock, signer, signFn, _ := PrepareXDCTestBlockChainForV2Engine(t, 901, params.TestXDPoSMockChainConfig, 0)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	blockInfo := &utils.BlockInfo{
		Hash:   currentBlock.Hash(),
		Round:  utils.Round(1),
		Number: currentBlock.Number(),
	}
	err := engineV2.VerifyBlockInfo(blockchain, blockInfo)
	assert.Nil(t, err)

	// Insert another Block, but it won't trigger commit
	blockNum := 902
	blockCoinBase := fmt.Sprintf("0x111000000000000000000000000000000%03d", blockNum)
	block902 := CreateBlock(blockchain, params.TestXDPoSMockChainConfig, currentBlock, blockNum, 2, blockCoinBase, signer, signFn, nil)
	blockchain.InsertBlock(block902)

	blockInfo = &utils.BlockInfo{
		Hash:   block902.Hash(),
		Round:  utils.Round(2),
		Number: block902.Number(),
	}
	err = engineV2.VerifyBlockInfo(blockchain, blockInfo)
	assert.Nil(t, err)

	blockInfo = &utils.BlockInfo{
		Hash:   currentBlock.Hash(),
		Round:  utils.Round(2),
		Number: currentBlock.Number(),
	}
	err = engineV2.VerifyBlockInfo(blockchain, blockInfo)
	assert.NotNil(t, err)

	blockInfo = &utils.BlockInfo{
		Hash:   block902.Hash(),
		Round:  utils.Round(3),
		Number: block902.Number(),
	}
	err = engineV2.VerifyBlockInfo(blockchain, blockInfo)
	assert.NotNil(t, err)

	blockInfo = &utils.BlockInfo{
		Hash:   block902.Hash(),
		Round:  utils.Round(2),
		Number: currentBlock.Number(),
	}
	err = engineV2.VerifyBlockInfo(blockchain, blockInfo)
	assert.NotNil(t, err)
}
