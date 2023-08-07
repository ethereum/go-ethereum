package engine_v2_tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/XinFinOrg/XDPoSChain/accounts/abi/bind/backends"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/stretchr/testify/assert"
)

func TestShouldSendVoteMsgAndCommitGrandGrandParentBlock(t *testing.T) {
	// Block 901 is the first v2 block with round of 1
	blockchain, _, currentBlock, signer, signFn, _ := PrepareXDCTestBlockChainForV2Engine(t, 901, params.TestXDPoSMockChainConfig, nil)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	var extraField types.ExtraFields_v2
	err := utils.DecodeBytesExtraFields(currentBlock.Extra(), &extraField)
	if err != nil {
		t.Fatal("Fail to decode extra data", err)
	}

	err = engineV2.ProposedBlockHandler(blockchain, currentBlock.Header())
	if err != nil {
		t.Fatal("Fail propose proposedBlock handler", err)
	}

	voteMsg := <-engineV2.BroadcastCh
	poolSize := engineV2.GetVotePoolSizeFaker(voteMsg.(*types.Vote))

	assert.Equal(t, poolSize, 1)
	assert.NotNil(t, voteMsg)
	assert.Equal(t, currentBlock.Hash(), voteMsg.(*types.Vote).ProposedBlockInfo.Hash)

	round, _, highestQC, _, _, _ := engineV2.GetPropertiesFaker()
	// Shoud trigger setNewRound
	assert.Equal(t, types.Round(1), round)
	// Should not update the highestQC
	assert.Equal(t, types.Round(0), highestQC.ProposedBlockInfo.Round)

	// Insert another Block, but it won't trigger commit
	blockNum := 902
	blockCoinBase := fmt.Sprintf("0x111000000000000000000000000000000%03d", blockNum)
	block902 := CreateBlock(blockchain, params.TestXDPoSMockChainConfig, currentBlock, blockNum, 2, blockCoinBase, signer, signFn, nil, nil, "")
	err = blockchain.InsertBlock(block902)
	assert.Nil(t, err)
	err = engineV2.ProposedBlockHandler(blockchain, block902.Header())
	if err != nil {
		t.Fatal("Fail propose proposedBlock handler", err)
	}
	// Trigger send vote again but for a new round
	voteMsg = <-engineV2.BroadcastCh
	assert.NotNil(t, voteMsg)
	round, _, highestQC, _, _, _ = engineV2.GetPropertiesFaker()
	// Shoud trigger setNewRound
	assert.Equal(t, types.Round(2), round)
	assert.Equal(t, types.Round(1), highestQC.ProposedBlockInfo.Round)

	// Insert one more Block, but still won't trigger commit
	blockNum = 903
	blockCoinBase = fmt.Sprintf("0x111000000000000000000000000000000%03d", blockNum)
	block903 := CreateBlock(blockchain, params.TestXDPoSMockChainConfig, block902, blockNum, 3, blockCoinBase, signer, signFn, nil, nil, "")
	err = blockchain.InsertBlock(block903)
	assert.Nil(t, err)
	err = engineV2.ProposedBlockHandler(blockchain, block903.Header())
	if err != nil {
		t.Fatal("Fail propose proposedBlock handler", err)
	}
	// Trigger send vote again but for a new round
	voteMsg = <-engineV2.BroadcastCh
	assert.NotNil(t, voteMsg)
	round, _, highestQC, _, _, highestCommitBlock := engineV2.GetPropertiesFaker()
	// Shoud NOT trigger setNewRound as the new block parent QC is round 1 but the currentRound is already 2
	assert.Equal(t, types.Round(3), round)
	assert.Equal(t, types.Round(2), highestQC.ProposedBlockInfo.Round)
	assert.Nil(t, highestCommitBlock)

	// Insert one more Block, this time will trigger commit
	blockNum = 904
	blockCoinBase = fmt.Sprintf("0x111000000000000000000000000000000%03d", blockNum)
	block904 := CreateBlock(blockchain, params.TestXDPoSMockChainConfig, block903, blockNum, 4, blockCoinBase, signer, signFn, nil, nil, "")
	err = blockchain.InsertBlock(block904)
	assert.Nil(t, err)
	err = engineV2.ProposedBlockHandler(blockchain, block904.Header())
	if err != nil {
		t.Fatal("Fail propose proposedBlock handler", err)
	}
	// Trigger send vote again but for a new round
	voteMsg = <-engineV2.BroadcastCh
	assert.NotNil(t, voteMsg)
	round, _, highestQC, _, _, highestCommitBlock = engineV2.GetPropertiesFaker()

	assert.Equal(t, types.Round(4), round)
	assert.Equal(t, types.Round(3), highestQC.ProposedBlockInfo.Round)
	assert.Equal(t, currentBlock.Hash(), highestCommitBlock.Hash)
	assert.Equal(t, currentBlock.Number(), highestCommitBlock.Number)
	assert.Equal(t, types.Round(1), highestCommitBlock.Round)
}

func TestShouldNotCommitIfRoundsNotContinousFor3Rounds(t *testing.T) {
	// Block 901 is the first v2 block with round of 1
	blockchain, _, currentBlock, signer, signFn, _ := PrepareXDCTestBlockChainForV2Engine(t, 905, params.TestXDPoSMockChainConfig, nil)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	var extraField types.ExtraFields_v2
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
	assert.Equal(t, currentBlock.Hash(), voteMsg.(*types.Vote).ProposedBlockInfo.Hash)

	round, _, highestQC, _, _, highestCommitBlock := engineV2.GetPropertiesFaker()

	grandGrandParentBlock := blockchain.GetBlockByNumber(902)
	// Shoud trigger setNewRound
	assert.Equal(t, types.Round(5), round)
	assert.Equal(t, types.Round(4), highestQC.ProposedBlockInfo.Round)
	assert.Equal(t, grandGrandParentBlock.Hash(), highestCommitBlock.Hash)
	assert.Equal(t, grandGrandParentBlock.Number(), highestCommitBlock.Number)
	assert.Equal(t, types.Round(2), highestCommitBlock.Round)

	// Injecting new block which have gaps in the round number (Round 7 instead of 6)
	blockNum := 906
	blockCoinBase := fmt.Sprintf("0x111000000000000000000000000000000%03d", blockNum)
	block906 := CreateBlock(blockchain, params.TestXDPoSMockChainConfig, currentBlock, blockNum, 7, blockCoinBase, signer, signFn, nil, nil, "")
	err = blockchain.InsertBlock(block906)
	assert.Nil(t, err)
	err = engineV2.ProposedBlockHandler(blockchain, block906.Header())
	if err != nil {
		t.Fatal("Fail propose proposedBlock handler", err)
	}
	// Trigger send vote again but for a new round
	voteMsg = <-engineV2.BroadcastCh
	assert.NotNil(t, voteMsg)
	round, _, highestQC, _, _, highestCommitBlock = engineV2.GetPropertiesFaker()
	grandGrandParentBlock = blockchain.GetBlockByNumber(903)

	assert.Equal(t, types.Round(6), round)
	assert.Equal(t, types.Round(5), highestQC.ProposedBlockInfo.Round)
	// It commit its grandgrandparent block
	assert.Equal(t, grandGrandParentBlock.Hash(), highestCommitBlock.Hash)
	assert.Equal(t, grandGrandParentBlock.Number(), highestCommitBlock.Number)
	assert.Equal(t, types.Round(3), highestCommitBlock.Round)

	blockNum = 907
	blockCoinBase = fmt.Sprintf("0x111000000000000000000000000000000%03d", blockNum)
	block907 := CreateBlock(blockchain, params.TestXDPoSMockChainConfig, block906, blockNum, 8, blockCoinBase, signer, signFn, nil, nil, "")
	err = blockchain.InsertBlock(block907)
	assert.Nil(t, err)
	err = engineV2.ProposedBlockHandler(blockchain, block907.Header())
	if err != nil {
		t.Fatal("Fail propose proposedBlock handler", err)
	}
	// Trigger send vote again but for a new round
	voteMsg = <-engineV2.BroadcastCh
	assert.NotNil(t, voteMsg)
	round, _, highestQC, _, _, highestCommitBlock = engineV2.GetPropertiesFaker()

	assert.Equal(t, types.Round(8), round)
	assert.Equal(t, types.Round(7), highestQC.ProposedBlockInfo.Round)
	// Should NOT commit, the `grandGrandParentBlock` is still on blockNum 903
	assert.Equal(t, grandGrandParentBlock.Hash(), highestCommitBlock.Hash)
	assert.Equal(t, grandGrandParentBlock.Number(), highestCommitBlock.Number)
	assert.Equal(t, types.Round(3), highestCommitBlock.Round)

}

func TestProposedBlockMessageHandlerSuccessfullyGenerateVote(t *testing.T) {
	blockchain, _, currentBlock, _, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 906, params.TestXDPoSMockChainConfig, nil)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	// Set current round to 5
	engineV2.SetNewRoundFaker(blockchain, types.Round(5), false)

	var extraField types.ExtraFields_v2
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
	assert.Equal(t, currentBlock.Hash(), voteMsg.(*types.Vote).ProposedBlockInfo.Hash)

	round, _, highestQC, _, _, _ := engineV2.GetPropertiesFaker()
	// Shoud trigger setNewRound
	assert.Equal(t, types.Round(6), round)
	assert.Equal(t, extraField.QuorumCert.Signatures, highestQC.Signatures)
}

// Should not set new round if proposedBlockInfo round is less than currentRound.
// NOTE: This shall not even happen because we have `verifyQC` before being passed into ProposedBlockHandler
func TestShouldNotSetNewRound(t *testing.T) {
	blockchain, _, currentBlock, _, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 906, params.TestXDPoSMockChainConfig, nil)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	// Set current round to 6
	engineV2.SetNewRoundFaker(blockchain, types.Round(6), false)

	var extraField types.ExtraFields_v2
	err := utils.DecodeBytesExtraFields(currentBlock.Extra(), &extraField)
	if err != nil {
		t.Fatal("Fail to decode extra data", err)
	}

	err = engineV2.ProposedBlockHandler(blockchain, currentBlock.Header())
	if err != nil {
		t.Fatal("Fail propose proposedBlock handler", err)
	}

	round, _, highestQC, _, _, _ := engineV2.GetPropertiesFaker()
	// Shoud not trigger setNewRound
	assert.Equal(t, types.Round(6), round)
	assert.Equal(t, extraField.QuorumCert.Signatures, highestQC.Signatures)
}

func TestShouldNotSendVoteMessageIfAlreadyVoteForThisRound(t *testing.T) {
	blockchain, _, currentBlock, _, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 906, params.TestXDPoSMockChainConfig, nil)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	// Set current round to 5
	engineV2.SetNewRoundFaker(blockchain, types.Round(5), false)

	err := engineV2.ProposedBlockHandler(blockchain, currentBlock.Header())
	if err != nil {
		t.Fatal("Fail propose proposedBlock handler", err)
	}

	voteMsg := <-engineV2.BroadcastCh
	assert.NotNil(t, voteMsg)
	assert.Equal(t, currentBlock.Hash(), voteMsg.(*types.Vote).ProposedBlockInfo.Hash)

	round, _, _, _, highestVotedRound, _ := engineV2.GetPropertiesFaker()
	// Shoud trigger setNewRound
	assert.Equal(t, types.Round(6), round)
	assert.Equal(t, types.Round(6), highestVotedRound)

	// Let's send again, this time, it shall not broadcast any vote message, because HigestVoteRound is same as currentRound
	err = engineV2.ProposedBlockHandler(blockchain, currentBlock.Header())
	if err != nil {
		t.Fatal("Fail propose proposedBlock handler again", err)
	}
	// Should not receive anything from the channel
	select {
	case <-engineV2.BroadcastCh:
		t.Fatal("Should not trigger vote")
	case <-time.After(3 * time.Second):
		// Shoud not trigger setNewRound
		round, _, _, _, highestVotedRound, _ = engineV2.GetPropertiesFaker()
		assert.Equal(t, types.Round(6), round)
		assert.Equal(t, types.Round(6), highestVotedRound)
	}
}

func TestShouldNotSendVoteMsgIfBlockInfoRoundNotEqualCurrentRound(t *testing.T) {
	blockchain, _, currentBlock, _, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 906, params.TestXDPoSMockChainConfig, nil)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	// Set current round to 8
	engineV2.SetNewRoundFaker(blockchain, types.Round(8), false)

	var extraField types.ExtraFields_v2
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
	case <-time.After(3 * time.Second):
		// Shoud not trigger setNewRound
		round, _, _, _, _, _ := engineV2.GetPropertiesFaker()
		assert.Equal(t, types.Round(8), round)
	}
}

/*
	Block and round relationship diagram for this test
	... - 13(3) - 14(4) - 15(5) - 16(6)
            \ 14'(7)
*/
func TestShouldNotSendVoteMsgIfBlockNotExtendedFromAncestor(t *testing.T) {
	// Block number 905, 906 have forks and forkedBlock is the 906th
	var numOfForks = new(int)
	*numOfForks = 3
	blockchain, _, currentBlock, _, _, forkedBlock := PrepareXDCTestBlockChainForV2Engine(t, 906, params.TestXDPoSMockChainConfig, &ForkedBlockOptions{numOfForkedBlocks: numOfForks})
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	var extraField types.ExtraFields_v2
	err := utils.DecodeBytesExtraFields(forkedBlock.Extra(), &extraField)
	if err != nil {
		t.Fatal("Fail to decode extra data", err)
	}
	assert.Equal(t, types.Round(9), extraField.Round)
	// Set the lockQC and other pre-requist properties by block 906
	err = engineV2.ProposedBlockHandler(blockchain, currentBlock.Header())
	if err != nil {
		t.Fatal("Error while handling block 16", err)
	}
	vote := <-engineV2.BroadcastCh
	assert.Equal(t, types.Round(6), vote.(*types.Vote).ProposedBlockInfo.Round)

	// Find the first forked block at block 14th
	firstForkedBlock := blockchain.GetBlockByHash(blockchain.GetBlockByHash(forkedBlock.ParentHash()).ParentHash())
	engineV2.SetNewRoundFaker(blockchain, types.Round(7), false)
	err = engineV2.ProposedBlockHandler(blockchain, firstForkedBlock.Header())
	if err != nil {
		t.Fatal("Fail propose proposedBlock handler", err)
	}
	// Should not receive anything from the channel
	select {
	case <-engineV2.BroadcastCh:
		t.Fatal("Should not trigger vote")
	case <-time.After(3 * time.Second):
		// Shoud not trigger setNewRound
		round, _, _, _, _, _ := engineV2.GetPropertiesFaker()
		assert.Equal(t, types.Round(7), round)
	}
}

func TestShouldSendVoteMsg(t *testing.T) {
	// Block number 15, 16 have forks and forkedBlock is the 16th
	blockchain, _, _, _, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 903, params.TestXDPoSMockChainConfig, nil)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	// Block 901 is first v2 block
	for i := 901; i < 904; i++ {
		blockHeader := blockchain.GetBlockByNumber(uint64(i)).Header()
		err := engineV2.ProposedBlockHandler(blockchain, blockHeader)
		if err != nil {
			t.Fatal(err)
		}
		round, _, _, _, _, _ := engineV2.GetPropertiesFaker()
		assert.Equal(t, types.Round(i-900), round)
		vote := <-engineV2.BroadcastCh
		assert.Equal(t, round, vote.(*types.Vote).ProposedBlockInfo.Round)
	}
}

func TestProposedBlockMessageHandlerNotGenerateVoteIfSignerNotInMNlist(t *testing.T) {
	blockchain, _, currentBlock, _, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 906, params.TestXDPoSMockChainConfig, nil)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2
	differentSigner, differentSignFn, err := backends.SimulateWalletAddressAndSignFn()
	assert.Nil(t, err)
	// Let's change the address
	engineV2.Authorize(differentSigner, differentSignFn)

	// Set current round to 5
	engineV2.SetNewRoundFaker(blockchain, types.Round(5), false)

	var extraField types.ExtraFields_v2
	err = utils.DecodeBytesExtraFields(currentBlock.Extra(), &extraField)
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
	case <-time.After(2 * time.Second):
		// Shoud not trigger setNewRound
		round, _, _, _, _, _ := engineV2.GetPropertiesFaker()
		assert.Equal(t, types.Round(6), round)
	}
}
