package tests

import (
	"testing"

	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/stretchr/testify/assert"
)

// ProposeBlock handler
func TestProposedBlockMessageHandlerSuccessfullyGenerateVote(t *testing.T) {
	blockchain, _, currentBlock, _ := PrepareXDCTestBlockChainForV2Engine(t, 11, params.TestXDPoSMockChainConfigWithV2Engine)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	// Set round to 11
	engineV2.SetNewRoundFaker(utils.Round(11), false)

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

	round, _, highestQC := engineV2.GetProperties()
	// Shoud not trigger setNewRound
	assert.Equal(t, utils.Round(11), round)
	assert.Equal(t, extraField.QuorumCert.Signatures, highestQC.Signatures)
}
