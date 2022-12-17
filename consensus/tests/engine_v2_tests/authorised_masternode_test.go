package engine_v2_tests

import (
	"math/big"
	"testing"
	"time"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/stretchr/testify/assert"
)

func TestIsAuthorisedMNForConsensusV2(t *testing.T) {
	// we skip test for v1 since it's hard to make a real genesis block
	blockchain, _, currentBlock, signer, signFn, _ := PrepareXDCTestBlockChainForV2Engine(t, 901, params.TestXDPoSMockChainConfig, nil)
	adaptor := blockchain.Engine().(*XDPoS.XDPoS)
	blockNum := 902
	blockCoinBase := "0x111000000000000000000000000000000123"
	currentBlock = CreateBlock(blockchain, params.TestXDPoSMockChainConfig, currentBlock, blockNum, 2, blockCoinBase, signer, signFn, nil, nil)
	err := blockchain.InsertBlock(currentBlock)
	assert.Nil(t, err)
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
	blockchain, _, currentBlock, signer, signFn, _ := PrepareXDCTestBlockChainForV2Engine(t, 900, params.TestXDPoSMockChainConfig, nil)
	minePeriod := params.TestV2Configs[0].MinePeriod
	adaptor := blockchain.Engine().(*XDPoS.XDPoS)
	blockNum := 901
	blockCoinBase := "0x111000000000000000000000000000000123"
	currentBlock = CreateBlock(blockchain, params.TestXDPoSMockChainConfig, currentBlock, blockNum, 1, blockCoinBase, signer, signFn, nil, nil)
	currentBlockHeader := currentBlock.Header()
	currentBlockHeader.Time = big.NewInt(time.Now().Unix())
	err := blockchain.InsertBlock(currentBlock)
	assert.Nil(t, err)
	// Less then Mine Period
	isYourTurn, err := adaptor.YourTurn(blockchain, currentBlockHeader, common.HexToAddress("xdc0D3ab14BBaD3D99F4203bd7a11aCB94882050E7e"))
	assert.Nil(t, err)
	assert.False(t, isYourTurn)

	time.Sleep(time.Duration(minePeriod) * time.Second)
	// The second address is valid as the round starting from 1
	isYourTurn, err = adaptor.YourTurn(blockchain, currentBlockHeader, common.HexToAddress("xdc0D3ab14BBaD3D99F4203bd7a11aCB94882050E7e"))
	assert.Nil(t, err)
	assert.True(t, isYourTurn)

	// The first and third address are not valid
	isYourTurn, err = adaptor.YourTurn(blockchain, currentBlock.Header(), common.HexToAddress("xdc703c4b2bD70c169f5717101CaeE543299Fc946C7"))
	assert.Nil(t, err)
	assert.False(t, isYourTurn)
	isYourTurn, err = adaptor.YourTurn(blockchain, currentBlock.Header(), common.HexToAddress("xdc71562b71999873DB5b286dF957af199Ec94617F7"))
	assert.Nil(t, err)
	assert.False(t, isYourTurn)

	// We continue to grow the chain which will increase the round number
	blockNum = 902
	currentBlock = CreateBlock(blockchain, params.TestXDPoSMockChainConfig, currentBlock, blockNum, int64(blockNum-900), blockCoinBase, signer, signFn, nil, nil)
	err = blockchain.InsertBlock(currentBlock)
	assert.Nil(t, err)
	time.Sleep(time.Duration(minePeriod) * time.Second)

	adaptor.EngineV2.SetNewRoundFaker(blockchain, 2, false)
	isYourTurn, _ = adaptor.YourTurn(blockchain, currentBlock.Header(), common.HexToAddress("xdc0D3ab14BBaD3D99F4203bd7a11aCB94882050E7e"))
	assert.False(t, isYourTurn)

	isYourTurn, _ = adaptor.YourTurn(blockchain, currentBlock.Header(), common.HexToAddress("xdc71562b71999873DB5b286dF957af199Ec94617F7"))
	assert.True(t, isYourTurn)

	isYourTurn, _ = adaptor.YourTurn(blockchain, currentBlock.Header(), common.HexToAddress("xdc5F74529C0338546f82389402a01c31fB52c6f434"))
	assert.False(t, isYourTurn)

}

func TestIsYourTurnConsensusV2CrossConfig(t *testing.T) {
	// we skip test for v1 since it's hard to make a real genesis block
	blockchain, _, currentBlock, signer, signFn, _ := PrepareXDCTestBlockChainForV2Engine(t, 909, params.TestXDPoSMockChainConfig, nil)
	firstMinePeriod := blockchain.Config().XDPoS.V2.CurrentConfig.MinePeriod

	adaptor := blockchain.Engine().(*XDPoS.XDPoS)
	blockNum := 910 // 910 is new config switch block
	blockCoinBase := "0x111000000000000000000000000000000123"
	currentBlock = CreateBlock(blockchain, params.TestXDPoSMockChainConfig, currentBlock, blockNum, 10, blockCoinBase, signer, signFn, nil, nil)
	currentBlockHeader := currentBlock.Header()
	currentBlockHeader.Time = big.NewInt(time.Now().Unix())
	err := blockchain.InsertBlock(currentBlock)
	assert.Nil(t, err)
	// after first mine period
	time.Sleep(time.Duration(firstMinePeriod) * time.Second)
	isYourTurn, err := adaptor.YourTurn(blockchain, currentBlockHeader, common.HexToAddress("xdc0D3ab14BBaD3D99F4203bd7a11aCB94882050E7e"))
	assert.Nil(t, err)
	assert.False(t, isYourTurn)

	adaptor.UpdateParams(currentBlockHeader) // it will be triggered automatically on the real code by other process

	// after new mine period
	secondMinePeriod := blockchain.Config().XDPoS.V2.CurrentConfig.MinePeriod

	time.Sleep(time.Duration(secondMinePeriod-firstMinePeriod) * time.Second)
	isYourTurn, err = adaptor.YourTurn(blockchain, currentBlockHeader, common.HexToAddress("xdc0D3ab14BBaD3D99F4203bd7a11aCB94882050E7e"))
	assert.Nil(t, err)
	assert.True(t, isYourTurn)
}
