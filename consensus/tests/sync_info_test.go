package tests

import (
	"testing"

	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/stretchr/testify/assert"
)

func TestSyncInfoShouldSuccessfullyUpdateByQC(t *testing.T) {
	// Block 11 is the first v2 block with starting round of 0
	blockchain, _, currentBlock, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 15, params.TestXDPoSMockChainConfigWithV2Engine, 0)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	var extraField utils.ExtraFields_v2
	err := utils.DecodeBytesExtraFields(currentBlock.Extra(), &extraField)
	if err != nil {
		t.Fatal("Fail to decode extra data", err)
	}

	syncInfoMsg := &utils.SyncInfo{
		HighestQuorumCert: extraField.QuorumCert,
		HighestTimeoutCert: &utils.TimeoutCert{
			Round:      utils.Round(2),
			Signatures: []utils.Signature{},
		},
	}

	err = engineV2.SyncInfoHandler(blockchain, syncInfoMsg)
	if err != nil {
		t.Fatal(err)
	}
	round, _, highestQuorumCert, _ := engineV2.GetProperties()
	// QC is parent block's qc, which is pointing at round 4, hence 4 + 1 = 5
	assert.Equal(t, utils.Round(5), round)
	assert.Equal(t, extraField.QuorumCert, highestQuorumCert)
}

func TestSyncInfoShouldSuccessfullyUpdateByTC(t *testing.T) {
	// Block 11 is the first v2 block with starting round of 0
	blockchain, _, currentBlock, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 15, params.TestXDPoSMockChainConfigWithV2Engine, 0)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	var extraField utils.ExtraFields_v2
	err := utils.DecodeBytesExtraFields(currentBlock.Extra(), &extraField)
	if err != nil {
		t.Fatal("Fail to decode extra data", err)
	}

	highestTC := &utils.TimeoutCert{
		Round:      utils.Round(6),
		Signatures: []utils.Signature{},
	}

	syncInfoMsg := &utils.SyncInfo{
		HighestQuorumCert:  extraField.QuorumCert,
		HighestTimeoutCert: highestTC,
	}

	err = engineV2.SyncInfoHandler(blockchain, syncInfoMsg)
	if err != nil {
		t.Fatal(err)
	}
	round, _, highestQuorumCert, _ := engineV2.GetProperties()
	assert.Equal(t, utils.Round(7), round)
	assert.Equal(t, extraField.QuorumCert, highestQuorumCert)
}
