package engine_v2_tests

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/XinFinOrg/XDPoSChain/rpc"
	"github.com/stretchr/testify/assert"
)

func TestGetMissedRoundsInEpochByBlockNumOnlyForV2Consensus(t *testing.T) {
	_, bc, _, _, _ := PrepareXDCTestBlockChainWith128Candidates(t, 1802, params.TestXDPoSMockChainConfig)

	engine := bc.BlockChain().Engine().(*XDPoS.XDPoS)
	blockNum := rpc.BlockNumber(123)

	data, err := engine.APIs(bc.BlockChain())[0].Service.(*XDPoS.API).GetMissedRoundsInEpochByBlockNum(&blockNum)

	assert.EqualError(t, err, "not supported in the v1 consensus")
	assert.Nil(t, data)
}

func TestGetMissedRoundsInEpochByBlockNumReturnEmptyForV2(t *testing.T) {
	_, bc, cb, _, _ := PrepareXDCTestBlockChainWith128Candidates(t, 1802, params.TestXDPoSMockChainConfig)

	engine := bc.BlockChain().Engine().(*XDPoS.XDPoS)
	blockNum := rpc.BlockNumber(cb.NumberU64())

	data, err := engine.APIs(bc.BlockChain())[0].Service.(*XDPoS.API).GetMissedRoundsInEpochByBlockNum(&blockNum)

	assert.Nil(t, err)
	assert.Equal(t, types.Round(900), data.EpochRound)
	assert.Equal(t, big.NewInt(1800), data.EpochBlockNumber)
	assert.Equal(t, 0, len(data.MissedRounds))

	blockNum = rpc.BlockNumber(1800)

	data, err = engine.APIs(bc.BlockChain())[0].Service.(*XDPoS.API).GetMissedRoundsInEpochByBlockNum(&blockNum)

	assert.Nil(t, err)
	assert.Equal(t, types.Round(900), data.EpochRound)
	assert.Equal(t, big.NewInt(1800), data.EpochBlockNumber)
	assert.Equal(t, 0, len(data.MissedRounds))

	blockNum = rpc.BlockNumber(1801)

	data, err = engine.APIs(bc.BlockChain())[0].Service.(*XDPoS.API).GetMissedRoundsInEpochByBlockNum(&blockNum)

	assert.Nil(t, err)
	assert.Equal(t, types.Round(900), data.EpochRound)
	assert.Equal(t, big.NewInt(1800), data.EpochBlockNumber)
	assert.Equal(t, 0, len(data.MissedRounds))
}

func TestGetMissedRoundsInEpochByBlockNumReturnEmptyForV2FistEpoch(t *testing.T) {
	_, bc, _, _, _ := PrepareXDCTestBlockChainWith128Candidates(t, 1802, params.TestXDPoSMockChainConfig)

	engine := bc.BlockChain().Engine().(*XDPoS.XDPoS)
	blockNum := rpc.BlockNumber(901)

	data, err := engine.APIs(bc.BlockChain())[0].Service.(*XDPoS.API).GetMissedRoundsInEpochByBlockNum(&blockNum)

	assert.Nil(t, err)
	assert.Equal(t, types.Round(1), data.EpochRound)
	assert.Equal(t, big.NewInt(901), data.EpochBlockNumber)
	assert.Equal(t, 0, len(data.MissedRounds))
}

func TestGetMissedRoundsInEpochByBlockNum(t *testing.T) {
	blockchain, bc, currentBlock, signer, signFn := PrepareXDCTestBlockChainWith128Candidates(t, 1802, params.TestXDPoSMockChainConfig)
	chainConfig := params.TestXDPoSMockChainConfig
	engine := bc.BlockChain().Engine().(*XDPoS.XDPoS)
	blockCoinBase := signer.Hex()

	startingBlockNum := currentBlock.Number().Int64() + 1
	// Skipped the round
	roundNumber := startingBlockNum - chainConfig.XDPoS.V2.SwitchBlock.Int64() + 2
	block := CreateBlock(blockchain, chainConfig, currentBlock, int(startingBlockNum), roundNumber, blockCoinBase, signer, signFn, nil, nil, "b345a8560bd51926803dd17677c9f0751193914a851a4ec13063d6bf50220b53")
	err := blockchain.InsertBlock(block)
	if err != nil {
		t.Fatal(err)
	}

	// Update Signer as there is no previous signer assigned
	blockchain.UpdateM1()
	if err != nil {
		t.Fatal(err)
	}

	blockNum := rpc.BlockNumber(1803)

	data, err := engine.APIs(bc.BlockChain())[0].Service.(*XDPoS.API).GetMissedRoundsInEpochByBlockNum(&blockNum)

	assert.Nil(t, err)
	assert.Equal(t, types.Round(900), data.EpochRound)
	assert.Equal(t, big.NewInt(1800), data.EpochBlockNumber)
	assert.Equal(t, 2, len(data.MissedRounds))
	assert.NotEmpty(t, data.MissedRounds[0].Miner)
	assert.Equal(t, data.MissedRounds[0].Round, types.Round(903))
	assert.Equal(t, data.MissedRounds[0].CurrentBlockNum, big.NewInt(1803))
	assert.Equal(t, data.MissedRounds[0].ParentBlockNum, big.NewInt(1802))
	assert.NotEmpty(t, data.MissedRounds[1].Miner)
	assert.Equal(t, data.MissedRounds[1].Round, types.Round(904))
	assert.Equal(t, data.MissedRounds[0].CurrentBlockNum, big.NewInt(1803))
	assert.Equal(t, data.MissedRounds[0].ParentBlockNum, big.NewInt(1802))

	assert.NotEqual(t, data.MissedRounds[0].Miner, data.MissedRounds[1].Miner)
}

func TestGetEpochNumbersBetween(t *testing.T) {
	_, bc, _, _, _ := PrepareXDCTestBlockChainWith128Candidates(t, 1802, params.TestXDPoSMockChainConfig)

	engine := bc.BlockChain().Engine().(*XDPoS.XDPoS)

	begin := rpc.BlockNumber(1800)
	end := rpc.BlockNumber(1802)
	numbers, err := engine.APIs(bc.BlockChain())[0].Service.(*XDPoS.API).GetEpochNumbersBetween(&begin, &end)

	assert.True(t, reflect.DeepEqual([]uint64{1800}, numbers))
	assert.Nil(t, err)

	begin = rpc.BlockNumber(1799)
	end = rpc.BlockNumber(1802)
	numbers, err = engine.APIs(bc.BlockChain())[0].Service.(*XDPoS.API).GetEpochNumbersBetween(&begin, &end)

	assert.True(t, reflect.DeepEqual([]uint64{1800}, numbers))
	assert.Nil(t, err)

	begin = rpc.BlockNumber(1799)
	end = rpc.BlockNumber(1802)
	numbers, err = engine.APIs(bc.BlockChain())[0].Service.(*XDPoS.API).GetEpochNumbersBetween(&begin, &end)

	assert.True(t, reflect.DeepEqual([]uint64{1800}, numbers))
	assert.Nil(t, err)

	begin = rpc.BlockNumber(901)
	end = rpc.BlockNumber(1802)
	numbers, err = engine.APIs(bc.BlockChain())[0].Service.(*XDPoS.API).GetEpochNumbersBetween(&begin, &end)

	assert.True(t, reflect.DeepEqual([]uint64{901, 1800}, numbers))
	assert.Nil(t, err)

	// 900 is V1, not V2, so error
	begin = rpc.BlockNumber(900)
	end = rpc.BlockNumber(1802)
	numbers, err = engine.APIs(bc.BlockChain())[0].Service.(*XDPoS.API).GetEpochNumbersBetween(&begin, &end)

	assert.Nil(t, numbers)
	assert.EqualError(t, err, "not supported in the v1 consensus")

	// 1803 not exist
	begin = rpc.BlockNumber(901)
	end = rpc.BlockNumber(1803)
	numbers, err = engine.APIs(bc.BlockChain())[0].Service.(*XDPoS.API).GetEpochNumbersBetween(&begin, &end)

	assert.Nil(t, numbers)
	assert.EqualError(t, err, "illegal end block number")

	// 1803 not exist
	begin = rpc.BlockNumber(1803)
	end = rpc.BlockNumber(1803)
	numbers, err = engine.APIs(bc.BlockChain())[0].Service.(*XDPoS.API).GetEpochNumbersBetween(&begin, &end)

	assert.Nil(t, numbers)
	assert.EqualError(t, err, "illegal begin block number")
}
func TestGetBlockByEpochNumber(t *testing.T) {
	blockchain, _, currentBlock, signer, signFn := PrepareXDCTestBlockChainWithPenaltyForV2Engine(t, 1802, params.TestXDPoSMockChainConfig)

	blockCoinBase := "0x111000000000000000000000000000000123"
	largeRound := int64(1802)
	newBlock := CreateBlock(blockchain, params.TestXDPoSMockChainConfig, currentBlock, int(currentBlock.NumberU64())+1, largeRound, blockCoinBase, signer, signFn, nil, nil, currentBlock.Header().Root.Hex())
	err := blockchain.InsertBlock(newBlock)
	assert.Nil(t, err)
	largeRound2 := int64(3603)
	newBlock2 := CreateBlock(blockchain, params.TestXDPoSMockChainConfig, newBlock, int(newBlock.NumberU64())+1, largeRound2, blockCoinBase, signer, signFn, nil, nil, newBlock.Header().Root.Hex())
	err = blockchain.InsertBlock(newBlock2)
	assert.Nil(t, err)

	// block num, round, epoch is as follows
	// 900,0,1 (v2 switch block, not v2 epoch switch block)
	// 901,1,1 (1st epoch switch block)
	// 902,2,1
	// ...
	// 1800,900,2 (2nd epoch switch block)
	// 1801,901,2
	// 1802,902,2
	// 1803,1802,3 (epoch switch)
	// epoch 4 has no block
	// 1804,3603,5 (epoch switch)
	engine := blockchain.Engine().(*XDPoS.XDPoS)

	// init the snapshot, otherwise getEpochSwitchInfo would return error
	checkpointHeader := blockchain.GetHeaderByNumber(blockchain.Config().XDPoS.V2.SwitchBlock.Uint64() + 1)
	err = engine.Initial(blockchain, checkpointHeader)
	assert.Nil(t, err)

	info, err := engine.APIs(blockchain)[0].Service.(*XDPoS.API).GetBlockInfoByEpochNum(0)
	assert.NotNil(t, err)
	assert.Nil(t, info)

	info, err = engine.APIs(blockchain)[0].Service.(*XDPoS.API).GetBlockInfoByEpochNum(1)
	assert.Equal(t, info.EpochRound, types.Round(1))
	assert.Nil(t, err)

	info, err = engine.APIs(blockchain)[0].Service.(*XDPoS.API).GetBlockInfoByEpochNum(2)
	assert.Equal(t, info.EpochRound, types.Round(900))
	assert.Nil(t, err)

	info, err = engine.APIs(blockchain)[0].Service.(*XDPoS.API).GetBlockInfoByEpochNum(3)
	assert.Equal(t, info.EpochRound, types.Round(largeRound))
	assert.Nil(t, err)

	info, err = engine.APIs(blockchain)[0].Service.(*XDPoS.API).GetBlockInfoByEpochNum(4)
	assert.NotNil(t, err)
	assert.Nil(t, info)

	info, err = engine.APIs(blockchain)[0].Service.(*XDPoS.API).GetBlockInfoByEpochNum(5)
	assert.Equal(t, info.EpochRound, types.Round(largeRound2))
	assert.Nil(t, err)

	info, err = engine.APIs(blockchain)[0].Service.(*XDPoS.API).GetBlockInfoByEpochNum(6)
	assert.NotNil(t, err)
	assert.Nil(t, info)
}
