package engine_v2_tests

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/XinFinOrg/XDPoSChain/common"
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

func TestVerifySyncInfoIfTCRoundIsAtNextEpoch(t *testing.T) {
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
		Round:      types.Round(899),
		Signatures: []types.Signature{},
	}

	timeoutForSign := &types.TimeoutForSign{
		Round:     types.Round(900),
		GapNumber: 450,
	}

	// Sign from acc 1, 2, 3 and voter
	acc1SignedHash := SignHashByPK(acc1Key, types.TimeoutSigHash(timeoutForSign).Bytes())
	acc2SignedHash := SignHashByPK(acc2Key, types.TimeoutSigHash(timeoutForSign).Bytes())
	acc3SignedHash := SignHashByPK(acc3Key, types.TimeoutSigHash(timeoutForSign).Bytes())
	voterSignedHash := SignHashByPK(voterKey, types.TimeoutSigHash(timeoutForSign).Bytes())

	var signatures []types.Signature
	signatures = append(signatures, acc1SignedHash, acc2SignedHash, acc3SignedHash, voterSignedHash)

	syncInfoTC := &types.TimeoutCert{
		Round:      timeoutForSign.Round,
		Signatures: signatures,
		GapNumber:  timeoutForSign.GapNumber,
	}

	syncInfoMsg := &types.SyncInfo{
		HighestQuorumCert:  extraField.QuorumCert,
		HighestTimeoutCert: syncInfoTC,
	}

	engineV2.SetPropertiesFaker(syncInfoMsg.HighestQuorumCert, highestTC)

	verified, err := engineV2.VerifySyncInfoMessage(blockchain, syncInfoMsg)
	assert.True(t, verified)
	assert.Nil(t, err)
}

func TestVerifySyncInfoIfTcUseDifferentEpoch(t *testing.T) {
	config := params.TestXDPoSMockChainConfig
	blockchain, _, currentBlock, signer, signFn, _ := PrepareXDCTestBlockChainForV2Engine(t, 1349, config, nil)
	adaptor := blockchain.Engine().(*XDPoS.XDPoS)
	x := adaptor.EngineV2

	// Insert block 1350
	t.Logf("Inserting block with propose at 1350...")
	blockCoinbaseA := "0xaaa0000000000000000000000000000000001350"
	// NOTE: voterAddr never exist in the Masternode list, but all acc1,2,3 already does
	tx, err := voteTX(37117, 0, signer.String())
	if err != nil {
		t.Fatal(err)
	}
	//Get from block validator error message
	merkleRoot := "8a355a8636d1aae24d5a63df0318534e09110891d6ab7bf20587da64725083be"
	header := &types.Header{
		Root:       common.HexToHash(merkleRoot),
		Number:     big.NewInt(int64(1350)),
		ParentHash: currentBlock.Hash(),
		Coinbase:   common.HexToAddress(blockCoinbaseA),
	}

	header.Extra = generateV2Extra(450, currentBlock, signer, signFn, nil)

	parentBlock, err := createBlockFromHeader(blockchain, header, []*types.Transaction{tx}, signer, signFn, config)
	assert.Nil(t, err)
	err = blockchain.InsertBlock(parentBlock)
	assert.Nil(t, err)
	// 1350 is a gap block, need to update the snapshot
	err = blockchain.UpdateM1()
	assert.Nil(t, err)
	t.Logf("Inserting block from 1351 to 1799...")
	for i := 1351; i <= 1799; i++ {
		blockCoinbase := fmt.Sprintf("0xaaa000000000000000000000000000000000%4d", i)
		//Get from block validator error message
		header = &types.Header{
			Root:       common.HexToHash(merkleRoot),
			Number:     big.NewInt(int64(i)),
			ParentHash: parentBlock.Hash(),
			Coinbase:   common.HexToAddress(blockCoinbase),
		}

		header.Extra = generateV2Extra(int64(i)-900, parentBlock, signer, signFn, nil)

		block, err := createBlockFromHeader(blockchain, header, nil, signer, signFn, config)
		if err != nil {
			t.Fatal(err)
		}
		err = blockchain.InsertBlock(block)
		assert.Nil(t, err)
		parentBlock = block
	}
	t.Logf("build epoch block with new set of masternodes")
	blockCoinbase := fmt.Sprintf("0xaaa0000000000000000000000000000000001800")
	//Get from block validator error message
	header = &types.Header{
		Root:       common.HexToHash(merkleRoot),
		Number:     big.NewInt(int64(1800)),
		ParentHash: parentBlock.Hash(),
		Coinbase:   common.HexToAddress(blockCoinbase),
	}

	header.Extra = generateV2Extra(900, parentBlock, signer, signFn, nil)
	validators := []byte{}

	snap, err := x.GetSnapshot(blockchain, parentBlock.Header())
	assert.Nil(t, err)

	for _, v := range snap.NextEpochCandidates {
		validators = append(validators, v[:]...)
	}
	// set up 1 more masternode to make it difference
	validators = append(validators, voterAddr[:]...)
	header.Validators = validators
	block, err := createBlockFromHeader(blockchain, header, nil, signer, signFn, config)
	if err != nil {
		t.Fatal(err)
	}
	err = blockchain.InsertBlock(block)
	assert.Nil(t, err)
	parentBlock = block

	var extraField types.ExtraFields_v2
	err = utils.DecodeBytesExtraFields(parentBlock.Extra(), &extraField)
	if err != nil {
		t.Fatal("Fail to decode extra data", err)
	}

	timeoutForSign := &types.TimeoutForSign{
		Round:     types.Round(899),
		GapNumber: 450,
	}

	// Sign from acc 1, 2, 3 and voter
	acc1SignedHash := SignHashByPK(acc1Key, types.TimeoutSigHash(timeoutForSign).Bytes())
	acc2SignedHash := SignHashByPK(acc2Key, types.TimeoutSigHash(timeoutForSign).Bytes())
	acc3SignedHash := SignHashByPK(acc3Key, types.TimeoutSigHash(timeoutForSign).Bytes())
	voterSignedHash := SignHashByPK(voterKey, types.TimeoutSigHash(timeoutForSign).Bytes())

	var signatures []types.Signature
	signatures = append(signatures, acc1SignedHash, acc2SignedHash, acc3SignedHash, voterSignedHash)

	newTC := &types.TimeoutCert{
		Round:      timeoutForSign.Round,
		Signatures: signatures,
		GapNumber:  timeoutForSign.GapNumber,
	}

	syncInfoMsg := &types.SyncInfo{
		HighestQuorumCert:  extraField.QuorumCert,
		HighestTimeoutCert: newTC,
	}

	x.SetPropertiesFaker(syncInfoMsg.HighestQuorumCert, &types.TimeoutCert{
		Round:      types.Round(898),
		Signatures: []types.Signature{},
	})

	verified, err := x.VerifySyncInfoMessage(blockchain, syncInfoMsg)
	assert.True(t, verified)
	assert.Nil(t, err)
}
