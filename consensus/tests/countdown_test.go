package tests

import (
	"testing"

	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/stretchr/testify/assert"
)

func TestCountdownTimeoutToSendTimeoutMessage(t *testing.T) {
	blockchain, _, _, _, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 11, params.TestXDPoSMockChainConfig, 0)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	engineV2.SetNewRoundFaker(utils.Round(1), true)

	timeoutMsg := <-engineV2.BroadcastCh
	poolSize := engineV2.GetTimeoutPoolSize(timeoutMsg.(*utils.Timeout))
	assert.Equal(t, poolSize, 1)
	assert.NotNil(t, timeoutMsg)

	valid, err := engineV2.VerifyTimeoutMessage(blockchain, timeoutMsg.(*utils.Timeout))
	// We can only test valid = false for now as the implementation for getCurrentRoundMasterNodes is not complete
	assert.False(t, valid)
	// This shows we are able to decode the timeout message, which is what this test is all about
	assert.Regexp(t, "Empty masternode list detected when verifying message signatures", err.Error())
}
