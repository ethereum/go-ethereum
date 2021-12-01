package consensus

import (
	"math/big"
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

	proposedBlockInfo := &utils.BlockInfo{
		Hash:   currentBlock.Hash(),
		Round:  utils.Round(12),
		Number: big.NewInt(12),
	}

	err = engineV2.ProposedBlockHandler(blockchain, proposedBlockInfo, &extraField.QuorumCert)
	if err != nil {
		t.Fatal("Fail propose proposedBlock handler", err)
	}

	voteMsg := <-engineV2.BroadcastCh
	assert.NotNil(t, voteMsg)
	assert.Equal(t, proposedBlockInfo.Hash, voteMsg.(*utils.Vote).ProposedBlockInfo.Hash)

	round, _, highestQC := engineV2.GetProperties()
	assert.Equal(t, utils.Round(12), round)
	assert.Equal(t, extraField.QuorumCert.Signatures, highestQC.Signatures)
}
