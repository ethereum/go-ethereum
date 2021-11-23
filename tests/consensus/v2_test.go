package consensus

import (
	"math/big"
	"testing"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/stretchr/testify/assert"
)

func TestCountdownTimeoutToSendTimeoutMessage(t *testing.T) {
	blockchain, _, _, _ := PrepareXDCTestBlockChain(t, 11, params.TestXDPoSMockChainConfigWithV2Engine)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	engineV2.SetNewRoundFaker(utils.Round(1), true)

	timeoutMsg := <-engineV2.BroadcastCh
	assert.NotNil(t, timeoutMsg)

	valid, err := engineV2.VerifyTimeoutMessage(timeoutMsg.(*utils.Timeout))
	// We can only test valid = false for now as the implementation for getCurrentRoundMasterNodes is not complete
	assert.False(t, valid)
	// This shows we are able to decode the timeout message, which is what this test is all about
	assert.Regexp(t, "^Masternodes does not contain signer addres.*", err.Error())
}

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
	assert.Equal(t, utils.Round(1), engineV2.GetCurrentRound())
	timeoutMsg = &utils.Timeout{
		Round:     utils.Round(1),
		Signature: []byte{2},
	}
	err = engineV2.TimeoutHandler(timeoutMsg)
	assert.Nil(t, err)
	assert.Equal(t, utils.Round(1), engineV2.GetCurrentRound())
	// Create a timeout message that should trigger timeout pool hook
	timeoutMsg = &utils.Timeout{
		Round:     utils.Round(1),
		Signature: []byte{3},
	}

	err = engineV2.TimeoutHandler(timeoutMsg)
	assert.Nil(t, err)

	syncInfoMsg := <-engineV2.BroadcastCh
	assert.NotNil(t, syncInfoMsg)

	// Should have QC, however, we did not inilise it, hence will show default empty value
	qc := syncInfoMsg.(utils.SyncInfo).HighestQuorumCert
	assert.NotNil(t, qc)

	tc := syncInfoMsg.(utils.SyncInfo).HighestTimeoutCert
	assert.NotNil(t, tc)
	assert.Equal(t, tc.Round, utils.Round(1))
	sigatures := []utils.Signature{[]byte{1}, []byte{2}, []byte{3}}
	assert.ElementsMatch(t, tc.Signatures, sigatures)
	assert.Equal(t, utils.Round(2), engineV2.GetCurrentRound())
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

// VoteHandler
func TestVoteMessageHandlerSuccessfullyGeneratedQC(t *testing.T) {
	blockchain, _, _, _ := PrepareXDCTestBlockChain(t, 11, params.TestXDPoSMockChainConfigWithV2Engine)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	blockInfo := &utils.BlockInfo{
		Hash:   common.HexToHash("0x1"),
		Round:  utils.Round(1),
		Number: big.NewInt(999),
	}

	// Set round to 1
	engineV2.SetNewRoundFaker(utils.Round(1), false)
	// Create two timeout message which will not reach vote pool threshold
	voteMsg := &utils.Vote{
		ProposedBlockInfo: *blockInfo,
		Signature:         []byte{1},
	}

	err := engineV2.VoteHandler(voteMsg)
	assert.Nil(t, err)
	assert.Equal(t, utils.Round(1), engineV2.GetCurrentRound())
	voteMsg = &utils.Vote{
		ProposedBlockInfo: *blockInfo,
		Signature:         []byte{2},
	}
	err = engineV2.VoteHandler(voteMsg)
	assert.Nil(t, err)
	assert.Equal(t, utils.Round(1), engineV2.GetCurrentRound())

	// Create a vote message that should trigger vote pool hook
	voteMsg = &utils.Vote{
		ProposedBlockInfo: *blockInfo,
		Signature:         []byte{3},
	}

	err = engineV2.VoteHandler(voteMsg)
	assert.Nil(t, err)
	// Check round has now changed from 1 to 2
	assert.Equal(t, utils.Round(2), engineV2.GetCurrentRound())
}

func TestThrowErrorIfVoteMsgRoundNotEqualToCurrentRound(t *testing.T) {
	blockchain, _, _, _ := PrepareXDCTestBlockChain(t, 11, params.TestXDPoSMockChainConfigWithV2Engine)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	blockInfo := &utils.BlockInfo{
		Hash:   common.HexToHash("0x1"),
		Round:  utils.Round(2),
		Number: big.NewInt(999),
	}

	// Set round to 3
	engineV2.SetNewRoundFaker(utils.Round(3), false)
	voteMsg := &utils.Vote{
		ProposedBlockInfo: *blockInfo,
		Signature:         []byte{1},
	}

	// voteRound > currentRound
	err := engineV2.VoteHandler(voteMsg)
	assert.NotNil(t, err)
	assert.Equal(t, "Vote message round number: 2 does not match currentRound: 3", err.Error())

	// Set round to 1
	engineV2.SetNewRoundFaker(utils.Round(1), false)
	err = engineV2.VoteHandler(voteMsg)
	assert.NotNil(t, err)
	// voteRound < currentRound
	assert.Equal(t, "Vote message round number: 2 does not match currentRound: 1", err.Error())
}

func TestProcessVoteMsgThenTimeoutMsg(t *testing.T) {
	blockchain, _, _, _ := PrepareXDCTestBlockChain(t, 11, params.TestXDPoSMockChainConfigWithV2Engine)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	// Set round to 1
	engineV2.SetNewRoundFaker(utils.Round(1), false)

	// Start with vote messages
	blockInfo := &utils.BlockInfo{
		Hash:   common.HexToHash("0x1"),
		Round:  utils.Round(1),
		Number: big.NewInt(999),
	}
	// Create two timeout message which will not reach vote pool threshold
	voteMsg := &utils.Vote{
		ProposedBlockInfo: *blockInfo,
		Signature:         []byte{1},
	}

	err := engineV2.VoteHandler(voteMsg)
	assert.Nil(t, err)
	assert.Equal(t, utils.Round(1), engineV2.GetCurrentRound())
	voteMsg = &utils.Vote{
		ProposedBlockInfo: *blockInfo,
		Signature:         []byte{2},
	}
	err = engineV2.VoteHandler(voteMsg)
	assert.Nil(t, err)
	assert.Equal(t, utils.Round(1), engineV2.GetCurrentRound())

	// Create a vote message that should trigger vote pool hook
	voteMsg = &utils.Vote{
		ProposedBlockInfo: *blockInfo,
		Signature:         []byte{3},
	}

	err = engineV2.VoteHandler(voteMsg)
	assert.Nil(t, err)
	// Check round has now changed from 1 to 2
	assert.Equal(t, utils.Round(2), engineV2.GetCurrentRound())

	// We shall have highestQuorumCert in engine now, let's do timeout msg to see if we can broadcast SyncInfo which contains both highestQuorumCert and HighestTimeoutCert

	// First, all incoming old timeout msg shall not be processed
	timeoutMsg := &utils.Timeout{
		Round:     utils.Round(1),
		Signature: []byte{1},
	}

	err = engineV2.TimeoutHandler(timeoutMsg)
	assert.NotNil(t, err)
	assert.Equal(t, "Timeout message round number: 1 does not match currentRound: 2", err.Error())

	// Ok, let's do the timeout msg which is on the same round as the current round by creating two timeout message which will not reach timeout pool threshold
	timeoutMsg = &utils.Timeout{
		Round:     utils.Round(2),
		Signature: []byte{1},
	}

	err = engineV2.TimeoutHandler(timeoutMsg)
	assert.Nil(t, err)
	assert.Equal(t, utils.Round(2), engineV2.GetCurrentRound())
	timeoutMsg = &utils.Timeout{
		Round:     utils.Round(2),
		Signature: []byte{2},
	}
	err = engineV2.TimeoutHandler(timeoutMsg)
	assert.Nil(t, err)
	assert.Equal(t, utils.Round(2), engineV2.GetCurrentRound())
	// Create a timeout message that should trigger timeout pool hook
	timeoutMsg = &utils.Timeout{
		Round:     utils.Round(2),
		Signature: []byte{3},
	}

	err = engineV2.TimeoutHandler(timeoutMsg)
	assert.Nil(t, err)

	syncInfoMsg := <-engineV2.BroadcastCh
	assert.NotNil(t, syncInfoMsg)

	// Should have HighestQuorumCert from previous round votes
	qc := syncInfoMsg.(utils.SyncInfo).HighestQuorumCert
	assert.NotNil(t, qc)
	assert.Equal(t, utils.Round(1), qc.ProposedBlockInfo.Round)

	tc := syncInfoMsg.(utils.SyncInfo).HighestTimeoutCert
	assert.NotNil(t, tc)
	assert.Equal(t, tc.Round, utils.Round(2))
	sigatures := []utils.Signature{[]byte{1}, []byte{2}, []byte{3}}
	assert.ElementsMatch(t, tc.Signatures, sigatures)
	// Round shall be +1 now
	assert.Equal(t, utils.Round(3), engineV2.GetCurrentRound())
}
