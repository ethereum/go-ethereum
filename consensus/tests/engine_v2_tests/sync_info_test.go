package engine_v2_tests

import (
	"math/big"
	"testing"

	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/stretchr/testify/assert"
)

func TestSyncInfoShouldSuccessfullyUpdateByQC(t *testing.T) {
	// Block 901 is the first v2 block with starting round of 0
	blockchain, _, currentBlock, _, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 905, params.TestXDPoSMockChainConfig, nil)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	var extraField types.ExtraFields_v2
	err := utils.DecodeBytesExtraFields(currentBlock.Extra(), &extraField)
	if err != nil {
		t.Fatal("Fail to decode extra data", err)
	}

	syncInfoMsg := &types.SyncInfo{
		HighestQuorumCert: extraField.QuorumCert,
		HighestTimeoutCert: &types.TimeoutCert{
			Round:      types.Round(2),
			Signatures: []types.Signature{},
		},
	}

	err = engineV2.SyncInfoHandler(blockchain, syncInfoMsg)
	if err != nil {
		t.Fatal(err)
	}
	round, _, highestQuorumCert, _, _, highestCommitBlock := engineV2.GetPropertiesFaker()
	// QC is parent block's qc, which is pointing at round 4, hence 4 + 1 = 5
	assert.Equal(t, types.Round(5), round)
	assert.Equal(t, extraField.QuorumCert, highestQuorumCert)
	assert.Equal(t, types.Round(2), highestCommitBlock.Round)
	assert.Equal(t, big.NewInt(902), highestCommitBlock.Number)
}

func TestSyncInfoShouldSuccessfullyUpdateByTC(t *testing.T) {
	// Block 901 is the first v2 block with starting round of 0
	blockchain, _, currentBlock, _, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 905, params.TestXDPoSMockChainConfig, nil)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	var extraField types.ExtraFields_v2
	err := utils.DecodeBytesExtraFields(currentBlock.Extra(), &extraField)
	if err != nil {
		t.Fatal("Fail to decode extra data", err)
	}

	highestTC := &types.TimeoutCert{
		Round:      types.Round(6),
		Signatures: []types.Signature{},
	}

	syncInfoMsg := &types.SyncInfo{
		HighestQuorumCert:  extraField.QuorumCert,
		HighestTimeoutCert: highestTC,
	}

	err = engineV2.SyncInfoHandler(blockchain, syncInfoMsg)
	if err != nil {
		t.Fatal(err)
	}
	round, _, highestQuorumCert, _, _, _ := engineV2.GetPropertiesFaker()
	assert.Equal(t, types.Round(7), round)
	assert.Equal(t, extraField.QuorumCert, highestQuorumCert)
}

func TestSkipVerifySyncInfoIfBothQcTcNotQualified(t *testing.T) {
	blockchain, _, _, _, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 905, params.TestXDPoSMockChainConfig, nil)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	// Make the Highest QC in syncInfo point to an old block to simulate it's no longer qualified
	parentBlock := blockchain.GetBlockByNumber(903)
	var extraField types.ExtraFields_v2
	err := utils.DecodeBytesExtraFields(parentBlock.Extra(), &extraField)
	if err != nil {
		t.Fatal("Fail to decode extra data", err)
	}

	highestTC := &types.TimeoutCert{
		Round:      types.Round(5),
		Signatures: []types.Signature{},
	}

	syncInfoMsg := &types.SyncInfo{
		HighestQuorumCert:  extraField.QuorumCert,
		HighestTimeoutCert: highestTC,
	}

	engineV2.SetPropertiesFaker(syncInfoMsg.HighestQuorumCert, syncInfoMsg.HighestTimeoutCert)

	verified, err := engineV2.VerifySyncInfoMessage(blockchain, syncInfoMsg)
	assert.False(t, verified)
	assert.Nil(t, err)
}
