package tests

import (
	"encoding/json"
	"testing"

	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"
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
	blockchain, _, currentBlock, _, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 901, &config, 0)
	adaptor := blockchain.Engine().(*XDPoS.XDPoS)

	// Happy path
	err = adaptor.VerifyHeader(blockchain, currentBlock.Header(), true)
	assert.Nil(t, err)

	// TODO: unhappy path XIN-135: https://hashlabs.atlassian.net/wiki/spaces/HASHLABS/pages/95944705/Verify+header
}
