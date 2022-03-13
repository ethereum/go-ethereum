package tests

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/eth/hooks"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/stretchr/testify/assert"
)

func TestHookPenaltyV2Mining(t *testing.T) {
	config := params.TestXDPoSMockChainConfig
	blockchain, _, _, _, _, _ := PrepareXDCTestBlockChainForV2Engine(t, int(config.XDPoS.Epoch)*7, config, 0)
	adaptor := blockchain.Engine().(*XDPoS.XDPoS)
	hooks.AttachConsensusV2Hooks(adaptor, blockchain, config)
	assert.NotNil(t, adaptor.EngineV2.HookPenalty)
	var extraField utils.ExtraFields_v2
	// 901 is the first v2 block
	header901 := blockchain.GetHeaderByNumber(config.XDPoS.Epoch + 1)
	err := utils.DecodeBytesExtraFields(header901.Extra, &extraField)
	assert.Nil(t, err)
	masternodes := adaptor.GetMasternodesFromCheckpointHeader(header901)
	assert.Equal(t, 4, len(masternodes))
	header6300 := blockchain.GetHeaderByNumber(config.XDPoS.Epoch * 7)
	penalty, err := adaptor.EngineV2.HookPenalty(blockchain, big.NewInt(int64(config.XDPoS.Epoch*7)), header6300.ParentHash, masternodes)
	assert.Nil(t, err)
	// miner (coinbase) is not in penalty. all others are in penalty
	assert.Equal(t, 3, len(penalty))
	assert.True(t, reflect.DeepEqual([]common.Address{header901.Coinbase}, common.RemoveItemFromArray(masternodes, penalty)))
	// set adaptor round/qc to that of 6299
	err = utils.DecodeBytesExtraFields(header6300.Extra, &extraField)
	assert.Nil(t, err)
	err = adaptor.EngineV2.ProcessQCFaker(blockchain, extraField.QuorumCert)
	assert.Nil(t, err)
	headerMining := &types.Header{
		ParentHash: header6300.ParentHash,
		Number:     header6300.Number,
		GasLimit:   params.TargetGasLimit,
		Time:       header6300.Time,
	}
	err = adaptor.Prepare(blockchain, headerMining)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(headerMining.Penalties)/common.AddressLength)
	// 20 candidates (set by PrepareXDCTestBlockChainForV2Engine) - 3 penalty = 17
	assert.Equal(t, 17, len(headerMining.Validators)/common.AddressLength)
}

func TestHookPenaltyV2Comeback(t *testing.T) {
	config := params.TestXDPoSMockChainConfig
	blockchain, _, _, signer, signFn := PrepareXDCTestBlockChainWithPenaltyForV2Engine(t, int(config.XDPoS.Epoch)*7, config)
	adaptor := blockchain.Engine().(*XDPoS.XDPoS)
	hooks.AttachConsensusV2Hooks(adaptor, blockchain, config)
	assert.NotNil(t, adaptor.EngineV2.HookPenalty)
	var extraField utils.ExtraFields_v2
	// 901 is the first v2 block
	header901 := blockchain.GetHeaderByNumber(config.XDPoS.Epoch + 1)
	err := utils.DecodeBytesExtraFields(header901.Extra, &extraField)
	assert.Nil(t, err)
	masternodes := adaptor.GetMasternodesFromCheckpointHeader(header901)
	assert.Equal(t, 4, len(masternodes))
	header6300 := blockchain.GetHeaderByNumber(config.XDPoS.Epoch * 7)
	penalty, err := adaptor.EngineV2.HookPenalty(blockchain, big.NewInt(int64(config.XDPoS.Epoch*7)), header6300.ParentHash, masternodes)
	assert.Nil(t, err)
	// miner (coinbase) is in comeback. so all addresses are in penalty
	assert.Equal(t, 4, len(penalty))
	header6285 := blockchain.GetHeaderByNumber(config.XDPoS.Epoch*7 - common.MergeSignRange)
	// forcely insert signing tx into cache, to cancel comeback. since no comeback, penalty is 3
	tx, err := signingTxWithSignerFn(header6285, 0, signer, signFn)
	assert.Nil(t, err)
	adaptor.CacheSigningTxs(header6285.Hash(), []*types.Transaction{tx})
	penalty, err = adaptor.EngineV2.HookPenalty(blockchain, big.NewInt(int64(config.XDPoS.Epoch*7)), header6300.ParentHash, masternodes)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(penalty))
}

func TestHookPenaltyV2Jump(t *testing.T) {
	config := params.TestXDPoSMockChainConfig
	end := int(config.XDPoS.Epoch)*7 - common.MergeSignRange
	blockchain, _, _, _, _ := PrepareXDCTestBlockChainWithPenaltyForV2Engine(t, int(config.XDPoS.Epoch)*7, config)
	adaptor := blockchain.Engine().(*XDPoS.XDPoS)
	hooks.AttachConsensusV2Hooks(adaptor, blockchain, config)
	assert.NotNil(t, adaptor.EngineV2.HookPenalty)
	var extraField utils.ExtraFields_v2
	// 901 is the first v2 block
	header901 := blockchain.GetHeaderByNumber(config.XDPoS.Epoch + 1)
	err := utils.DecodeBytesExtraFields(header901.Extra, &extraField)
	assert.Nil(t, err)
	masternodes := adaptor.GetMasternodesFromCheckpointHeader(header901)
	assert.Equal(t, 4, len(masternodes))
	header6285 := blockchain.GetHeaderByNumber(uint64(end))
	adaptor.EngineV2.SetNewRoundFaker(blockchain, utils.Round(config.XDPoS.Epoch*7), false)
	// round 6285-6300 miss blocks, penalty should work as usual
	penalty, err := adaptor.EngineV2.HookPenalty(blockchain, header6285.Number, header6285.ParentHash, masternodes)
	assert.Nil(t, err)
	assert.Equal(t, 4, len(penalty))
}
