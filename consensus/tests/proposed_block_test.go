package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/stretchr/testify/assert"
)

func TestShouldSendVoteMsgAndCommitGrandGrandParentBlock(t *testing.T) {
	// Block 11 is the first v2 block with round of 1
	blockchain, _, currentBlock, signer, signFn, _ := PrepareXDCTestBlockChainForV2Engine(t, 11, params.TestXDPoSMockChainConfigWithV2Engine, 0)
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

	round, _, highestQC, _, _ := engineV2.GetProperties()
	// Shoud trigger setNewRound
	assert.Equal(t, utils.Round(1), round)
	// Should not update the highestQC
	assert.Equal(t, utils.Round(0), highestQC.ProposedBlockInfo.Round)

	// Insert another Block, but it won't trigger commit
	blockNum := 12
	blockCoinBase := fmt.Sprintf("0x111000000000000000000000000000000%03d", blockNum)
	blockHeader := createBlock(params.TestXDPoSMockChainConfigWithV2Engine, currentBlock, blockNum, 2, blockCoinBase, signer, signFn)
	block12, err := insertBlock(blockchain, blockHeader)
	if err != nil {
		t.Fatal(err)
	}
	err = engineV2.ProposedBlockHandler(blockchain, block12.Header())
	if err != nil {
		t.Fatal("Fail propose proposedBlock handler", err)
	}
	// Trigger send vote again but for a new round
	voteMsg = <-engineV2.BroadcastCh
	assert.NotNil(t, voteMsg)
	round, _, highestQC, _, _ = engineV2.GetProperties()
	// Shoud trigger setNewRound
	assert.Equal(t, utils.Round(2), round)
	assert.Equal(t, utils.Round(1), highestQC.ProposedBlockInfo.Round)

	// Insert one more Block, but still won't trigger commit
	blockNum = 13
	blockCoinBase = fmt.Sprintf("0x111000000000000000000000000000000%03d", blockNum)
	blockHeader = createBlock(params.TestXDPoSMockChainConfigWithV2Engine, block12, blockNum, 3, blockCoinBase, signer, signFn)
	block13, err := insertBlock(blockchain, blockHeader)
	if err != nil {
		t.Fatal(err)
	}
	err = engineV2.ProposedBlockHandler(blockchain, block13.Header())
	if err != nil {
		t.Fatal("Fail propose proposedBlock handler", err)
	}
	// Trigger send vote again but for a new round
	voteMsg = <-engineV2.BroadcastCh
	assert.NotNil(t, voteMsg)
	round, _, highestQC, _, highestCommitBlock := engineV2.GetProperties()
	// Shoud NOT trigger setNewRound as the new block parent QC is round 1 but the currentRound is already 2
	assert.Equal(t, utils.Round(3), round)
	assert.Equal(t, utils.Round(2), highestQC.ProposedBlockInfo.Round)
	assert.Nil(t, highestCommitBlock)

	// Insert one more Block, this time will trigger commit
	blockNum = 14
	blockCoinBase = fmt.Sprintf("0x111000000000000000000000000000000%03d", blockNum)
	blockHeader = createBlock(params.TestXDPoSMockChainConfigWithV2Engine, block13, blockNum, 4, blockCoinBase, signer, signFn)
	block14, err := insertBlock(blockchain, blockHeader)
	if err != nil {
		t.Fatal(err)
	}
	err = engineV2.ProposedBlockHandler(blockchain, block14.Header())
	if err != nil {
		t.Fatal("Fail propose proposedBlock handler", err)
	}
	// Trigger send vote again but for a new round
	voteMsg = <-engineV2.BroadcastCh
	assert.NotNil(t, voteMsg)
	round, _, highestQC, _, highestCommitBlock = engineV2.GetProperties()

	assert.Equal(t, utils.Round(4), round)
	assert.Equal(t, utils.Round(3), highestQC.ProposedBlockInfo.Round)
	assert.Equal(t, currentBlock.Hash(), highestCommitBlock.Hash)
	assert.Equal(t, currentBlock.Number(), highestCommitBlock.Number)
	assert.Equal(t, utils.Round(1), highestCommitBlock.Round)
}

func TestShouldNotCommitIfRoundsNotContinousFor3Rounds(t *testing.T) {
	// Block 11 is the first v2 block with round of 1
	blockchain, _, currentBlock, signer, signFn, _ := PrepareXDCTestBlockChainForV2Engine(t, 15, params.TestXDPoSMockChainConfigWithV2Engine, 0)
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
	assert.NotNil(t, voteMsg)
	assert.Equal(t, currentBlock.Hash(), voteMsg.(*utils.Vote).ProposedBlockInfo.Hash)

	round, _, highestQC, _, highestCommitBlock := engineV2.GetProperties()

	grandGrandParentBlock := blockchain.GetBlockByNumber(12)
	// Shoud trigger setNewRound
	assert.Equal(t, utils.Round(5), round)
	assert.Equal(t, utils.Round(4), highestQC.ProposedBlockInfo.Round)
	assert.Equal(t, grandGrandParentBlock.Hash(), highestCommitBlock.Hash)
	assert.Equal(t, grandGrandParentBlock.Number(), highestCommitBlock.Number)
	assert.Equal(t, utils.Round(2), highestCommitBlock.Round)

	// Injecting new block which have gaps in the round number (Round 7 instead of 6)
	blockNum := 16
	blockCoinBase := fmt.Sprintf("0x111000000000000000000000000000000%03d", blockNum)
	blockHeader := createBlock(params.TestXDPoSMockChainConfigWithV2Engine, currentBlock, blockNum, 7, blockCoinBase, signer, signFn)
	block16, err := insertBlock(blockchain, blockHeader)
	if err != nil {
		t.Fatal(err)
	}
	err = engineV2.ProposedBlockHandler(blockchain, block16.Header())
	if err != nil {
		t.Fatal("Fail propose proposedBlock handler", err)
	}
	// Trigger send vote again but for a new round
	voteMsg = <-engineV2.BroadcastCh
	assert.NotNil(t, voteMsg)
	round, _, highestQC, _, highestCommitBlock = engineV2.GetProperties()
	grandGrandParentBlock = blockchain.GetBlockByNumber(13)

	assert.Equal(t, utils.Round(6), round)
	assert.Equal(t, utils.Round(5), highestQC.ProposedBlockInfo.Round)
	// It commit its grandgrandparent block
	assert.Equal(t, grandGrandParentBlock.Hash(), highestCommitBlock.Hash)
	assert.Equal(t, grandGrandParentBlock.Number(), highestCommitBlock.Number)
	assert.Equal(t, utils.Round(3), highestCommitBlock.Round)

	blockNum = 17
	blockCoinBase = fmt.Sprintf("0x111000000000000000000000000000000%03d", blockNum)
	blockHeader = createBlock(params.TestXDPoSMockChainConfigWithV2Engine, block16, blockNum, 8, blockCoinBase, signer, signFn)
	block17, err := insertBlock(blockchain, blockHeader)
	if err != nil {
		t.Fatal(err)
	}
	err = engineV2.ProposedBlockHandler(blockchain, block17.Header())
	if err != nil {
		t.Fatal("Fail propose proposedBlock handler", err)
	}
	// Trigger send vote again but for a new round
	voteMsg = <-engineV2.BroadcastCh
	assert.NotNil(t, voteMsg)
	round, _, highestQC, _, highestCommitBlock = engineV2.GetProperties()

	assert.Equal(t, utils.Round(8), round)
	assert.Equal(t, utils.Round(7), highestQC.ProposedBlockInfo.Round)
	// Should NOT commit, the `grandGrandParentBlock` is still on blockNum 13
	assert.Equal(t, grandGrandParentBlock.Hash(), highestCommitBlock.Hash)
	assert.Equal(t, grandGrandParentBlock.Number(), highestCommitBlock.Number)
	assert.Equal(t, utils.Round(3), highestCommitBlock.Round)

}

func TestProposedBlockMessageHandlerSuccessfullyGenerateVote(t *testing.T) {
	blockchain, _, currentBlock, _, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 16, params.TestXDPoSMockChainConfigWithV2Engine, 0)
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

	round, _, highestQC, _, _ := engineV2.GetProperties()
	// Shoud trigger setNewRound
	assert.Equal(t, utils.Round(6), round)
	assert.Equal(t, extraField.QuorumCert.Signatures, highestQC.Signatures)
}

// Should not set new round if proposedBlockInfo round is less than currentRound.
// NOTE: This shall not even happen because we have `verifyQC` before being passed into ProposedBlockHandler
func TestShouldNotSetNewRound(t *testing.T) {
	blockchain, _, currentBlock, _, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 16, params.TestXDPoSMockChainConfigWithV2Engine, 0)
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

	round, _, highestQC, _, _ := engineV2.GetProperties()
	// Shoud not trigger setNewRound
	assert.Equal(t, utils.Round(6), round)
	assert.Equal(t, extraField.QuorumCert.Signatures, highestQC.Signatures)
}

func TestShouldNotSendVoteMessageIfAlreadyVoteForThisRound(t *testing.T) {
	blockchain, _, currentBlock, _, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 16, params.TestXDPoSMockChainConfigWithV2Engine, 0)
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

	round, _, _, highestVotedRound, _ := engineV2.GetProperties()
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
		round, _, _, highestVotedRound, _ = engineV2.GetProperties()
		assert.Equal(t, utils.Round(6), round)
		assert.Equal(t, utils.Round(6), highestVotedRound)
	}
}

func TestShouldNotSendVoteMsgIfBlockInfoRoundNotEqualCurrentRound(t *testing.T) {
	blockchain, _, currentBlock, _, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 16, params.TestXDPoSMockChainConfigWithV2Engine, 0)
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
		round, _, _, _, _ := engineV2.GetProperties()
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
	blockchain, _, currentBlock, _, _, forkedBlock := PrepareXDCTestBlockChainForV2Engine(t, 16, params.TestXDPoSMockChainConfigWithV2Engine, 3)
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
		round, _, _, _, _ := engineV2.GetProperties()
		assert.Equal(t, utils.Round(7), round)
	}
}

func TestShouldSendVoteMsg(t *testing.T) {
	// Block number 15, 16 have forks and forkedBlock is the 16th
	blockchain, _, _, _, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 13, params.TestXDPoSMockChainConfigWithV2Engine, 0)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	// Block 11 is first v2 block
	for i := 11; i < 14; i++ {
		blockHeader := blockchain.GetBlockByNumber(uint64(i)).Header()
		err := engineV2.ProposedBlockHandler(blockchain, blockHeader)
		if err != nil {
			t.Fatal(err)
		}
		round, _, _, _, _ := engineV2.GetProperties()
		assert.Equal(t, utils.Round(i-10), round)
		vote := <-engineV2.BroadcastCh
		assert.Equal(t, round, vote.(*utils.Vote).ProposedBlockInfo.Round)
	}
}
