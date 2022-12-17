package engine_v2_tests

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"
	"github.com/XinFinOrg/XDPoSChain/core/state"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/eth/hooks"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/stretchr/testify/assert"
)

func TestHookRewardV2(t *testing.T) {
	b, err := json.Marshal(params.TestXDPoSMockChainConfig)
	assert.Nil(t, err)
	configString := string(b)

	var config params.ChainConfig
	err = json.Unmarshal([]byte(configString), &config)
	assert.Nil(t, err)
	// set switch to 1800, so that it covers 901-1799, 1800-2700 two epochs
	config.XDPoS.V2.SwitchBlock.SetUint64(1800)

	blockchain, _, _, signer, signFn, _ := PrepareXDCTestBlockChainForV2Engine(t, int(config.XDPoS.Epoch)*5, &config, nil)

	adaptor := blockchain.Engine().(*XDPoS.XDPoS)
	hooks.AttachConsensusV2Hooks(adaptor, blockchain, &config)
	assert.NotNil(t, adaptor.EngineV2.HookReward)
	// forcely insert signing tx into cache, to give rewards.
	header915 := blockchain.GetHeaderByNumber(config.XDPoS.Epoch + 15)
	header916 := blockchain.GetHeaderByNumber(config.XDPoS.Epoch + 16)
	header1799 := blockchain.GetHeaderByNumber(config.XDPoS.Epoch*2 - 1)
	header1801 := blockchain.GetHeaderByNumber(config.XDPoS.Epoch*2 + 1)
	tx, err := signingTxWithSignerFn(header915, 0, signer, signFn)
	assert.Nil(t, err)
	adaptor.CacheSigningTxs(header916.Hash(), []*types.Transaction{tx})
	statedb, err := blockchain.StateAt(header1799.Root)
	assert.Nil(t, err)
	parentState := statedb.Copy()
	reward, err := adaptor.EngineV2.HookReward(blockchain, statedb, parentState, header1801)
	assert.Nil(t, err)
	assert.Zero(t, len(reward))
	header2699 := blockchain.GetHeaderByNumber(config.XDPoS.Epoch*3 - 1)
	header2700 := blockchain.GetHeaderByNumber(config.XDPoS.Epoch * 3)
	statedb, err = blockchain.StateAt(header2699.Root)
	assert.Nil(t, err)
	parentState = statedb.Copy()
	reward, err = adaptor.EngineV2.HookReward(blockchain, statedb, parentState, header2700)
	assert.Nil(t, err)
	owner := state.GetCandidateOwner(parentState, signer)
	result := reward["rewards"].(map[common.Address]interface{})
	assert.Equal(t, 1, len(result))
	for _, x := range result {
		r := x.(map[common.Address]*big.Int)
		a, _ := big.NewInt(0).SetString("225000000000000000000", 10)
		assert.Zero(t, a.Cmp(r[owner]))
		b, _ := big.NewInt(0).SetString("25000000000000000000", 10)
		assert.Zero(t, b.Cmp(r[common.HexToAddress("0x0000000000000000000000000000000000000068")]))
	}
	header2685 := blockchain.GetHeaderByNumber(config.XDPoS.Epoch*2 + 885)
	header2716 := blockchain.GetHeaderByNumber(config.XDPoS.Epoch*3 + 16)
	header3599 := blockchain.GetHeaderByNumber(config.XDPoS.Epoch*4 - 1)
	header3600 := blockchain.GetHeaderByNumber(config.XDPoS.Epoch * 4)
	tx, err = signingTxWithSignerFn(header2685, 0, signer, signFn)
	assert.Nil(t, err)
	// signed block hash and block contains tx are in different epoch, we should get same rewards
	adaptor.CacheSigningTxs(header2716.Hash(), []*types.Transaction{tx})
	statedb, err = blockchain.StateAt(header3599.Root)
	assert.Nil(t, err)
	parentState = statedb.Copy()
	reward, err = adaptor.EngineV2.HookReward(blockchain, statedb, parentState, header3600)
	assert.Nil(t, err)
	result = reward["rewards"].(map[common.Address]interface{})
	assert.Equal(t, 1, len(result))
	for _, x := range result {
		r := x.(map[common.Address]*big.Int)
		a, _ := big.NewInt(0).SetString("225000000000000000000", 10)
		assert.Zero(t, a.Cmp(r[owner]))
		b, _ := big.NewInt(0).SetString("25000000000000000000", 10)
		assert.Zero(t, b.Cmp(r[config.XDPoS.FoudationWalletAddr]))
	}
	// if no signing tx, then reward will be 0
	header4499 := blockchain.GetHeaderByNumber(config.XDPoS.Epoch*5 - 1)
	header4500 := blockchain.GetHeaderByNumber(config.XDPoS.Epoch * 5)
	statedb, err = blockchain.StateAt(header4499.Root)
	assert.Nil(t, err)
	parentState = statedb.Copy()
	reward, err = adaptor.EngineV2.HookReward(blockchain, statedb, parentState, header4500)
	assert.Nil(t, err)
	result = reward["rewards"].(map[common.Address]interface{})
	assert.Equal(t, 0, len(result))
}

func TestHookRewardV2SplitReward(t *testing.T) {
	b, err := json.Marshal(params.TestXDPoSMockChainConfig)
	assert.Nil(t, err)
	configString := string(b)

	var config params.ChainConfig
	err = json.Unmarshal([]byte(configString), &config)
	assert.Nil(t, err)
	// set switch to 1800, so that it covers 901-1799, 1800-2700 two epochs
	config.XDPoS.V2.SwitchBlock.SetUint64(1800)

	blockchain, _, _, signer, signFn, _ := PrepareXDCTestBlockChainForV2Engine(t, int(config.XDPoS.Epoch)*3, &config, nil)

	adaptor := blockchain.Engine().(*XDPoS.XDPoS)
	hooks.AttachConsensusV2Hooks(adaptor, blockchain, &config)
	assert.NotNil(t, adaptor.EngineV2.HookReward)
	// forcely insert signing tx into cache, to give rewards.
	header915 := blockchain.GetHeaderByNumber(config.XDPoS.Epoch + 15)
	header916 := blockchain.GetHeaderByNumber(config.XDPoS.Epoch + 16)
	// header917 := blockchain.GetHeaderByNumber(config.XDPoS.Epoch + 17)
	header1785 := blockchain.GetHeaderByNumber(config.XDPoS.Epoch*2 - 15)
	header1799 := blockchain.GetHeaderByNumber(config.XDPoS.Epoch*2 - 1)
	header1801 := blockchain.GetHeaderByNumber(config.XDPoS.Epoch*2 + 1)
	tx, err := signingTxWithSignerFn(header915, 0, signer, signFn)
	assert.Nil(t, err)
	adaptor.CacheSigningTxs(header916.Hash(), []*types.Transaction{tx})
	tx2, err := signingTxWithKey(header915, 0, acc1Key)
	assert.Nil(t, err)
	tx3, err := signingTxWithKey(header1785, 0, acc1Key)
	assert.Nil(t, err)
	adaptor.CacheSigningTxs(header1799.Hash(), []*types.Transaction{tx2, tx3})

	statedb, err := blockchain.StateAt(header1799.Root)
	assert.Nil(t, err)
	parentState := statedb.Copy()
	reward, err := adaptor.EngineV2.HookReward(blockchain, statedb, parentState, header1801)
	assert.Nil(t, err)
	assert.Zero(t, len(reward))
	header2699 := blockchain.GetHeaderByNumber(config.XDPoS.Epoch*3 - 1)
	header2700 := blockchain.GetHeaderByNumber(config.XDPoS.Epoch * 3)
	statedb, err = blockchain.StateAt(header2699.Root)
	assert.Nil(t, err)
	parentState = statedb.Copy()
	reward, err = adaptor.EngineV2.HookReward(blockchain, statedb, parentState, header2700)
	assert.Nil(t, err)
	result := reward["rewards"].(map[common.Address]interface{})
	assert.Equal(t, 2, len(result))
	// two signing account, 3 txs, reward is split by 1:2 (total reward is 250...000)
	for addr, x := range result {
		if addr == acc1Addr {
			r := x.(map[common.Address]*big.Int)
			owner := state.GetCandidateOwner(parentState, acc1Addr)
			a, _ := big.NewInt(0).SetString("149999999999999999999", 10)
			assert.Zero(t, a.Cmp(r[owner]))
			b, _ := big.NewInt(0).SetString("16666666666666666666", 10)
			assert.Zero(t, b.Cmp(r[common.HexToAddress("0x0000000000000000000000000000000000000068")]))
		} else if addr == signer {
			r := x.(map[common.Address]*big.Int)
			owner := state.GetCandidateOwner(parentState, signer)
			a, _ := big.NewInt(0).SetString("74999999999999999999", 10)
			assert.Zero(t, a.Cmp(r[owner]))
			b, _ := big.NewInt(0).SetString("8333333333333333333", 10)
			assert.Zero(t, b.Cmp(r[common.HexToAddress("0x0000000000000000000000000000000000000068")]))
		}
	}
}
