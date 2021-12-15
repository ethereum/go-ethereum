package tests

import (
	"testing"
	"time"

	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/stretchr/testify/assert"
)

func TestProcessFirstV2BlockAndSendVoteMsg(t *testing.T) {
	// Block 11 is the first v2 block with round of 1
	blockchain, _, currentBlock, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 11, params.TestXDPoSMockChainConfigWithV2Engine, 0)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	var extraField utils.ExtraFields_v2
	err := utils.DecodeBytesExtraFields(currentBlock.Extra(), &extraField)
	if err != nil {
		t.Fatal("Fail to decode extra data", err)
	}

	err = engineV2.ProposedBlockHandler(blockchain, currentBlock.Header())
	if err != nil {
		t.Fatal("Fail propose proposedBlock handler", err)
	}

	voteMsg := <-engineV2.BroadcastCh
	poolSize := engineV2.GetVotePoolSize(voteMsg.(*utils.Vote))

	assert.Equal(t, poolSize, 1)
	assert.NotNil(t, voteMsg)
	assert.Equal(t, currentBlock.Hash(), voteMsg.(*utils.Vote).ProposedBlockInfo.Hash)

	round, _, highestQC, _ := engineV2.GetProperties()
	// Shoud trigger setNewRound
	assert.Equal(t, utils.Round(1), round)
	// Should not update the highestQC
	assert.Equal(t, utils.Round(0), highestQC.ProposedBlockInfo.Round)

}

func TestProposedBlockMessageHandlerSuccessfullyGenerateVote(t *testing.T) {
	blockchain, _, currentBlock, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 16, params.TestXDPoSMockChainConfigWithV2Engine, 0)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	// Set current round to 5
	engineV2.SetNewRoundFaker(utils.Round(5), false)

	var extraField utils.ExtraFields_v2
	err := utils.DecodeBytesExtraFields(currentBlock.Extra(), &extraField)
	if err != nil {
		t.Fatal("Fail to decode extra data", err)
	}

	err = engineV2.ProposedBlockHandler(blockchain, currentBlock.Header())
	if err != nil {
		t.Fatal("Fail propose proposedBlock handler", err)
	}

	voteMsg := <-engineV2.BroadcastCh
	assert.NotNil(t, voteMsg)
	assert.Equal(t, currentBlock.Hash(), voteMsg.(*utils.Vote).ProposedBlockInfo.Hash)

	round, _, highestQC, _ := engineV2.GetProperties()
	// Shoud trigger setNewRound
	assert.Equal(t, utils.Round(6), round)
	assert.Equal(t, extraField.QuorumCert.Signatures, highestQC.Signatures)
}

// Should not set new round if proposedBlockInfo round is less than currentRound.
// NOTE: This shall not even happen because we have `verifyQC` before being passed into ProposedBlockHandler
func TestShouldNotSetNewRound(t *testing.T) {
	blockchain, _, currentBlock, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 16, params.TestXDPoSMockChainConfigWithV2Engine, 0)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	// Set current round to 6
	engineV2.SetNewRoundFaker(utils.Round(6), false)

	var extraField utils.ExtraFields_v2
	err := utils.DecodeBytesExtraFields(currentBlock.Extra(), &extraField)
	if err != nil {
		t.Fatal("Fail to decode extra data", err)
	}

	err = engineV2.ProposedBlockHandler(blockchain, currentBlock.Header())
	if err != nil {
		t.Fatal("Fail propose proposedBlock handler", err)
	}

	round, _, highestQC, _ := engineV2.GetProperties()
	// Shoud not trigger setNewRound
	assert.Equal(t, utils.Round(6), round)
	assert.Equal(t, extraField.QuorumCert.Signatures, highestQC.Signatures)
}

func TestShouldNotSendVoteMessageIfAlreadyVoteForThisRound(t *testing.T) {
	blockchain, _, currentBlock, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 16, params.TestXDPoSMockChainConfigWithV2Engine, 0)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	// Set current round to 5
	engineV2.SetNewRoundFaker(utils.Round(5), false)

	err := engineV2.ProposedBlockHandler(blockchain, currentBlock.Header())
	if err != nil {
		t.Fatal("Fail propose proposedBlock handler", err)
	}

	voteMsg := <-engineV2.BroadcastCh
	assert.NotNil(t, voteMsg)
	assert.Equal(t, currentBlock.Hash(), voteMsg.(*utils.Vote).ProposedBlockInfo.Hash)

	round, _, _, highestVotedRound := engineV2.GetProperties()
	// Shoud trigger setNewRound
	assert.Equal(t, utils.Round(6), round)
	assert.Equal(t, utils.Round(6), highestVotedRound)

	// Let's send again, this time, it shall not broadcast any vote message, because HigestVoteRound is same as currentRound
	err = engineV2.ProposedBlockHandler(blockchain, currentBlock.Header())
	if err != nil {
		t.Fatal("Fail propose proposedBlock handler again", err)
	}
	// Should not receive anything from the channel
	select {
	case <-engineV2.BroadcastCh:
		t.Fatal("Should not trigger vote")
	case <-time.After(5 * time.Second):
		// Shoud not trigger setNewRound
		round, _, _, highestVotedRound = engineV2.GetProperties()
		assert.Equal(t, utils.Round(6), round)
		assert.Equal(t, utils.Round(6), highestVotedRound)
	}
}

func TestShouldNotSendVoteMsgIfBlockInfoRoundNotEqualCurrentRound(t *testing.T) {
	blockchain, _, currentBlock, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 16, params.TestXDPoSMockChainConfigWithV2Engine, 0)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	// Set current round to 8
	engineV2.SetNewRoundFaker(utils.Round(8), false)

	var extraField utils.ExtraFields_v2
	err := utils.DecodeBytesExtraFields(currentBlock.Extra(), &extraField)
	if err != nil {
		t.Fatal("Fail to decode extra data", err)
	}

	err = engineV2.ProposedBlockHandler(blockchain, currentBlock.Header())
	if err != nil {
		t.Fatal("Fail propose proposedBlock handler", err)
	}
	// Should not receive anything from the channel
	select {
	case <-engineV2.BroadcastCh:
		t.Fatal("Should not trigger vote")
	case <-time.After(5 * time.Second):
		// Shoud not trigger setNewRound
		round, _, _, _ := engineV2.GetProperties()
		assert.Equal(t, utils.Round(8), round)
	}
}

/*
	Block and round relationship diagram for this test
	... - 13(3) - 14(4) - 15(5) - 16(6)
            \ 14'(7)
*/
func TestShouldNotSendVoteMsgIfBlockNotExtendedFromAncestor(t *testing.T) {
	// Block number 15, 16 have forks and forkedBlock is the 16th
	blockchain, _, currentBlock, _, forkedBlock := PrepareXDCTestBlockChainForV2Engine(t, 16, params.TestXDPoSMockChainConfigWithV2Engine, 3)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	var extraField utils.ExtraFields_v2
	err := utils.DecodeBytesExtraFields(forkedBlock.Extra(), &extraField)
	if err != nil {
		t.Fatal("Fail to decode extra data", err)
	}
	assert.Equal(t, utils.Round(9), extraField.Round)
	// Set the lockQC and other pre-requist properties by block 16
	err = engineV2.ProposedBlockHandler(blockchain, currentBlock.Header())
	if err != nil {
		t.Fatal("Error while handling block 16", err)
	}
	vote := <-engineV2.BroadcastCh
	assert.Equal(t, utils.Round(6), vote.(*utils.Vote).ProposedBlockInfo.Round)

	// Find the first forked block at block 14th
	firstForkedBlock := blockchain.GetBlockByHash(blockchain.GetBlockByHash(forkedBlock.ParentHash()).ParentHash())
	engineV2.SetNewRoundFaker(utils.Round(7), false)
	err = engineV2.ProposedBlockHandler(blockchain, firstForkedBlock.Header())
	if err != nil {
		t.Fatal("Fail propose proposedBlock handler", err)
	}
	// Should not receive anything from the channel
	select {
	case <-engineV2.BroadcastCh:
		t.Fatal("Should not trigger vote")
	case <-time.After(5 * time.Second):
		// Shoud not trigger setNewRound
		round, _, _, _ := engineV2.GetProperties()
		assert.Equal(t, utils.Round(7), round)
	}
}

func TestShouldSendVoteMsg(t *testing.T) {
	// Block number 15, 16 have forks and forkedBlock is the 16th
	blockchain, _, _, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 13, params.TestXDPoSMockChainConfigWithV2Engine, 0)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	// Block 11 is first v2 block
	for i := 11; i < 14; i++ {
		blockHeader := blockchain.GetBlockByNumber(uint64(i)).Header()
		err := engineV2.ProposedBlockHandler(blockchain, blockHeader)
		if err != nil {
			t.Fatal(err)
		}
		round, _, _, _ := engineV2.GetProperties()
		assert.Equal(t, utils.Round(i-10), round)
		vote := <-engineV2.BroadcastCh
		assert.Equal(t, round, vote.(*utils.Vote).ProposedBlockInfo.Round)
	}
}
