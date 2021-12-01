package consensus

import (
	"testing"

	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/stretchr/testify/assert"
)

// Timeout handler
func TestTimeoutMessageHandlerSuccessfullyGenerateTCandSyncInfo(t *testing.T) {
	blockchain, _, _, _ := PrepareXDCTestBlockChain(t, 11, params.TestXDPoSMockChainConfigWithV2Engine)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	// Set round to 1
	engineV2.SetNewRoundFaker(utils.Round(1), false)
	// Create two timeout message which will not reach timeout pool threshold
	timeoutMsg := &utils.Timeout{
		Round:     utils.Round(1),
		Signature: []byte{1},
	}

	err := engineV2.TimeoutHandler(timeoutMsg)
	assert.Nil(t, err)
	currentRound, _, _ := engineV2.GetProperties()
	assert.Equal(t, utils.Round(1), currentRound)
	timeoutMsg = &utils.Timeout{
		Round:     utils.Round(1),
		Signature: []byte{2},
	}
	err = engineV2.TimeoutHandler(timeoutMsg)
	assert.Nil(t, err)
	currentRound, _, _ = engineV2.GetProperties()
	assert.Equal(t, utils.Round(1), currentRound)
	// Create a timeout message that should trigger timeout pool hook
	timeoutMsg = &utils.Timeout{
		Round:     utils.Round(1),
		Signature: []byte{3},
	}

	err = engineV2.TimeoutHandler(timeoutMsg)
	assert.Nil(t, err)

	syncInfoMsg := <-engineV2.BroadcastCh

	currentRound, _, _ = engineV2.GetProperties()

	assert.NotNil(t, syncInfoMsg)

	// Should have QC, however, we did not inilise it, hence will show default empty value
	qc := syncInfoMsg.(utils.SyncInfo).HighestQuorumCert
	assert.NotNil(t, qc)

	tc := syncInfoMsg.(utils.SyncInfo).HighestTimeoutCert
	assert.NotNil(t, tc)
	assert.Equal(t, tc.Round, utils.Round(1))
	sigatures := []utils.Signature{[]byte{1}, []byte{2}, []byte{3}}
	assert.ElementsMatch(t, tc.Signatures, sigatures)
	assert.Equal(t, utils.Round(2), currentRound)
}

func TestThrowErrorIfTimeoutMsgRoundNotEqualToCurrentRound(t *testing.T) {
	blockchain, _, _, _ := PrepareXDCTestBlockChain(t, 11, params.TestXDPoSMockChainConfigWithV2Engine)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	// Set round to 3
	engineV2.SetNewRoundFaker(utils.Round(3), false)
	timeoutMsg := &utils.Timeout{
		Round:     utils.Round(2),
		Signature: []byte{1},
	}

	err := engineV2.TimeoutHandler(timeoutMsg)
	assert.NotNil(t, err)
	// Timeout msg round > currentRound
	assert.Equal(t, "Timeout message round number: 2 does not match currentRound: 3", err.Error())

	// Set round to 1
	engineV2.SetNewRoundFaker(utils.Round(1), false)
	err = engineV2.TimeoutHandler(timeoutMsg)
	assert.NotNil(t, err)
	// Timeout msg round < currentRound
	assert.Equal(t, "Timeout message round number: 2 does not match currentRound: 1", err.Error())
}
