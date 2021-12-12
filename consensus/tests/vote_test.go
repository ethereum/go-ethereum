package tests

import (
	"math/big"
	"testing"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/stretchr/testify/assert"
)

// VoteHandler
func TestVoteMessageHandlerSuccessfullyGeneratedAndProcessQCForFistV2Round(t *testing.T) {
	blockchain, _, currentBlock, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 11, params.TestXDPoSMockChainConfigWithV2Engine, 0)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	blockInfo := &utils.BlockInfo{
		Hash:   currentBlock.Hash(),
		Round:  utils.Round(1),
		Number: big.NewInt(11),
	}

	// Set round to 5
	engineV2.SetNewRoundFaker(utils.Round(1), false)
	// Create two timeout message which will not reach vote pool threshold
	voteMsg := &utils.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         []byte{1},
	}

	err := engineV2.VoteHandler(blockchain, voteMsg)
	assert.Nil(t, err)
	currentRound, lockQuorumCert, highestQuorumCert, _ := engineV2.GetProperties()
	// initialised with nil and 0 round
	assert.Nil(t, lockQuorumCert)
	assert.Nil(t, highestQuorumCert)
	assert.Equal(t, utils.Round(1), currentRound)
	voteMsg = &utils.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         []byte{2},
	}
	err = engineV2.VoteHandler(blockchain, voteMsg)
	assert.Nil(t, err)
	currentRound, lockQuorumCert, highestQuorumCert, _ = engineV2.GetProperties()
	// Still using the initlised value because we did not yet go to the next round
	assert.Nil(t, lockQuorumCert)
	assert.Nil(t, highestQuorumCert)

	assert.Equal(t, utils.Round(1), currentRound)

	// Create a vote message that should trigger vote pool hook and increment the round to 6
	voteMsg = &utils.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         []byte{3},
	}

	err = engineV2.VoteHandler(blockchain, voteMsg)
	assert.Nil(t, err)
	currentRound, lockQuorumCert, highestQuorumCert, _ = engineV2.GetProperties()
	// The lockQC shall be the parent's QC round number
	assert.Equal(t, utils.Round(0), lockQuorumCert.ProposedBlockInfo.Round)
	// The highestQC proposedBlockInfo shall be the same as the one from its votes
	assert.Equal(t, highestQuorumCert.ProposedBlockInfo, voteMsg.ProposedBlockInfo)
	// Check round has now changed from 5 to 6
	assert.Equal(t, utils.Round(2), currentRound)
}

func TestVoteMessageHandlerSuccessfullyGeneratedAndProcessQC(t *testing.T) {
	blockchain, _, currentBlock, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 15, params.TestXDPoSMockChainConfigWithV2Engine, 0)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	blockInfo := &utils.BlockInfo{
		Hash:   currentBlock.Hash(),
		Round:  utils.Round(5),
		Number: big.NewInt(15),
	}

	// Set round to 5
	engineV2.SetNewRoundFaker(utils.Round(5), false)
	// Create two timeout message which will not reach vote pool threshold
	voteMsg := &utils.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         []byte{1},
	}

	err := engineV2.VoteHandler(blockchain, voteMsg)
	assert.Nil(t, err)
	currentRound, lockQuorumCert, highestQuorumCert, _ := engineV2.GetProperties()
	// initialised with nil and 0 round
	assert.Nil(t, lockQuorumCert)
	assert.Nil(t, highestQuorumCert)
	assert.Equal(t, utils.Round(5), currentRound)
	voteMsg = &utils.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         []byte{2},
	}
	err = engineV2.VoteHandler(blockchain, voteMsg)
	assert.Nil(t, err)
	currentRound, lockQuorumCert, highestQuorumCert, _ = engineV2.GetProperties()
	// Still using the initlised value because we did not yet go to the next round
	assert.Nil(t, lockQuorumCert)
	assert.Nil(t, highestQuorumCert)

	assert.Equal(t, utils.Round(5), currentRound)

	// Create a vote message that should trigger vote pool hook and increment the round to 6
	voteMsg = &utils.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         []byte{3},
	}

	err = engineV2.VoteHandler(blockchain, voteMsg)
	assert.Nil(t, err)
	currentRound, lockQuorumCert, highestQuorumCert, _ = engineV2.GetProperties()
	// The lockQC shall be the parent's QC round number
	assert.Equal(t, utils.Round(4), lockQuorumCert.ProposedBlockInfo.Round)
	// The highestQC proposedBlockInfo shall be the same as the one from its votes
	assert.Equal(t, highestQuorumCert.ProposedBlockInfo, voteMsg.ProposedBlockInfo)
	// Check round has now changed from 5 to 6
	assert.Equal(t, utils.Round(6), currentRound)
}

func TestThrowErrorIfVoteMsgRoundNotEqualToCurrentRound(t *testing.T) {
	blockchain, _, _, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 15, params.TestXDPoSMockChainConfigWithV2Engine, 0)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	blockInfo := &utils.BlockInfo{
		Hash:   common.HexToHash("0x1"),
		Round:  utils.Round(6),
		Number: big.NewInt(999),
	}

	// Set round to 7
	engineV2.SetNewRoundFaker(utils.Round(7), false)
	voteMsg := &utils.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         []byte{1},
	}

	// voteRound > currentRound
	err := engineV2.VoteHandler(blockchain, voteMsg)
	assert.NotNil(t, err)
	assert.Equal(t, "Vote message round number: 6 does not match currentRound: 7", err.Error())

	// Set round to 5
	engineV2.SetNewRoundFaker(utils.Round(5), false)
	err = engineV2.VoteHandler(blockchain, voteMsg)
	assert.NotNil(t, err)
	// voteRound < currentRound
	assert.Equal(t, "Vote message round number: 6 does not match currentRound: 5", err.Error())
}

func TestProcessVoteMsgThenTimeoutMsg(t *testing.T) {
	blockchain, _, currentBlock, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 15, params.TestXDPoSMockChainConfigWithV2Engine, 0)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	// Set round to 5
	engineV2.SetNewRoundFaker(utils.Round(5), false)

	// Start with vote messages
	blockInfo := &utils.BlockInfo{
		Hash:   currentBlock.Hash(),
		Round:  utils.Round(5),
		Number: big.NewInt(11),
	}
	// Create two vote message which will not reach vote pool threshold
	voteMsg := &utils.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         []byte{1},
	}

	err := engineV2.VoteHandler(blockchain, voteMsg)
	assert.Nil(t, err)
	currentRound, lockQuorumCert, highestQuorumCert, _ := engineV2.GetProperties()
	// initialised with nil and 0 round
	assert.Nil(t, lockQuorumCert)
	assert.Nil(t, highestQuorumCert)

	assert.Equal(t, utils.Round(5), currentRound)
	voteMsg = &utils.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         []byte{2},
	}
	err = engineV2.VoteHandler(blockchain, voteMsg)
	assert.Nil(t, err)
	currentRound, _, _, _ = engineV2.GetProperties()
	assert.Equal(t, utils.Round(5), currentRound)

	// Create a vote message that should trigger vote pool hook
	voteMsg = &utils.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         []byte{3},
	}

	err = engineV2.VoteHandler(blockchain, voteMsg)
	assert.Nil(t, err)
	// Check round has now changed from 5 to 6
	currentRound, lockQuorumCert, highestQuorumCert, _ = engineV2.GetProperties()
	// The lockQC shall be the parent's QC round number
	assert.Equal(t, utils.Round(4), lockQuorumCert.ProposedBlockInfo.Round)
	// The highestQC proposedBlockInfo shall be the same as the one from its votes
	assert.Equal(t, highestQuorumCert.ProposedBlockInfo, voteMsg.ProposedBlockInfo)

	assert.Equal(t, utils.Round(6), currentRound)

	// We shall have highestQuorumCert in engine now, let's do timeout msg to see if we can broadcast SyncInfo which contains both highestQuorumCert and HighestTimeoutCert

	// First, all incoming old timeout msg shall not be processed
	timeoutMsg := &utils.Timeout{
		Round:     utils.Round(5),
		Signature: []byte{1},
	}

	err = engineV2.TimeoutHandler(timeoutMsg)
	assert.NotNil(t, err)
	assert.Equal(t, "Timeout message round number: 5 does not match currentRound: 6", err.Error())

	// Ok, let's do the timeout msg which is on the same round as the current round by creating two timeout message which will not reach timeout pool threshold
	timeoutMsg = &utils.Timeout{
		Round:     utils.Round(6),
		Signature: []byte{1},
	}

	err = engineV2.TimeoutHandler(timeoutMsg)
	assert.Nil(t, err)
	currentRound, _, _, _ = engineV2.GetProperties()
	assert.Equal(t, utils.Round(6), currentRound)
	timeoutMsg = &utils.Timeout{
		Round:     utils.Round(6),
		Signature: []byte{2},
	}
	err = engineV2.TimeoutHandler(timeoutMsg)
	assert.Nil(t, err)
	currentRound, _, _, _ = engineV2.GetProperties()
	assert.Equal(t, utils.Round(6), currentRound)

	// Create a timeout message that should trigger timeout pool hook
	timeoutMsg = &utils.Timeout{
		Round:     utils.Round(6),
		Signature: []byte{3},
	}

	err = engineV2.TimeoutHandler(timeoutMsg)
	assert.Nil(t, err)

	syncInfoMsg := <-engineV2.BroadcastCh
	assert.NotNil(t, syncInfoMsg)

	// Should have HighestQuorumCert from previous round votes
	qc := syncInfoMsg.(*utils.SyncInfo).HighestQuorumCert
	assert.NotNil(t, qc)
	assert.Equal(t, utils.Round(5), qc.ProposedBlockInfo.Round)

	tc := syncInfoMsg.(*utils.SyncInfo).HighestTimeoutCert
	assert.NotNil(t, tc)
	assert.Equal(t, utils.Round(6), tc.Round)
	sigatures := []utils.Signature{[]byte{1}, []byte{2}, []byte{3}}
	assert.ElementsMatch(t, tc.Signatures, sigatures)
	// Round shall be +1 now
	currentRound, _, _, _ = engineV2.GetProperties()
	assert.Equal(t, utils.Round(7), currentRound)
}
