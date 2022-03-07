package tests

import (
	"encoding/json"
	"testing"

	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/stretchr/testify/assert"
)

func TestShouldVerifyBlock(t *testing.T) {
	b, err := json.Marshal(params.TestXDPoSMockChainConfig)
	assert.Nil(t, err)
	configString := string(b)

	var config params.ChainConfig
	err = json.Unmarshal([]byte(configString), &config)
	assert.Nil(t, err)
	// Enable verify
	config.XDPoS.V2.SkipV2Validation = false
	// Skip the mining time validation by set mine time to 0
	config.XDPoS.V2.MinePeriod = 0
	// Block 901 is the first v2 block with round of 1
	blockchain, _, _, _, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 910, &config, 0)
	adaptor := blockchain.Engine().(*XDPoS.XDPoS)

	// Happy path
	err = adaptor.VerifyHeader(blockchain, blockchain.GetBlockByNumber(901).Header(), true)
	assert.Nil(t, err)

	// Verify non-epoch switch block
	err = adaptor.VerifyHeader(blockchain, blockchain.GetBlockByNumber(902).Header(), true)
	assert.Nil(t, err)

	nonEpochSwitchWithValidators := blockchain.GetBlockByNumber(902).Header()
	nonEpochSwitchWithValidators.Validators = acc1Addr.Bytes()
	err = adaptor.VerifyHeader(blockchain, nonEpochSwitchWithValidators, true)
	assert.Equal(t, utils.ErrInvalidFieldInNonEpochSwitch, err)
}
