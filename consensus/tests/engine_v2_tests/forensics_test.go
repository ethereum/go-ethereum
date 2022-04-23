package engine_v2_tests

import (
	"math/big"
	"testing"
	"time"

	"github.com/XinFinOrg/XDPoSChain/accounts"
	"github.com/XinFinOrg/XDPoSChain/accounts/abi/bind/backends"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/stretchr/testify/assert"
)

func TestProcessQcShallSetForensicsCommittedQc(t *testing.T) {
	blockchain, _, currentBlock, signer, signFn, _ := PrepareXDCTestBlockChainForV2Engine(t, 905, params.TestXDPoSMockChainConfig, 0)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	// Assuming we are getting block 906 which have QC pointing at block 905
	blockInfo := &utils.BlockInfo{
		Hash:   currentBlock.Hash(),
		Round:  utils.Round(5),
		Number: big.NewInt(905),
	}
	voteForSign := &utils.VoteForSign{
		ProposedBlockInfo: blockInfo,
		GapNumber:         450,
	}
	voteSigningHash := utils.VoteSigHash(voteForSign)

	// Set round to 5
	engineV2.SetNewRoundFaker(blockchain, utils.Round(5), false)
	// Create two vote messages which will not reach vote pool threshold
	signedHash, err := signFn(accounts.Account{Address: signer}, voteSigningHash.Bytes())
	assert.Nil(t, err)
	voteMsg := &utils.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         signedHash,
		GapNumber:         450,
	}

	err = engineV2.VoteHandler(blockchain, voteMsg)
	assert.Nil(t, err)
	signedHash = SignHashByPK(acc1Key, voteSigningHash.Bytes())
	voteMsg = &utils.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         signedHash,
		GapNumber:         450,
	}
	err = engineV2.VoteHandler(blockchain, voteMsg)
	assert.Nil(t, err)

	// Create another vote which is signed by someone not from the master node list
	randomSigner, randomSignFn, err := backends.SimulateWalletAddressAndSignFn()
	assert.Nil(t, err)
	randomlySignedHash, err := randomSignFn(accounts.Account{Address: randomSigner}, voteSigningHash.Bytes())
	assert.Nil(t, err)
	voteMsg = &utils.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         randomlySignedHash,
		GapNumber:         450,
	}
	err = engineV2.VoteHandler(blockchain, voteMsg)
	assert.Nil(t, err)

	// Create a vote message that should trigger vote pool hook and increment the round to 6
	signedHash = SignHashByPK(acc3Key, voteSigningHash.Bytes())
	voteMsg = &utils.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         signedHash,
		GapNumber:         450,
	}

	err = engineV2.VoteHandler(blockchain, voteMsg)
	assert.Nil(t, err)

	time.Sleep(5000 * time.Millisecond)
	assert.Equal(t, 3, len(engineV2.GetForensicsFaker().HighestCommittedQCs))
}

func TestSetCommittedQCsInOrder(t *testing.T) {
	blockchain, _, currentBlock, _, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 905, params.TestXDPoSMockChainConfig, 0)
	forensics := blockchain.Engine().(*XDPoS.XDPoS).EngineV2.GetForensicsFaker()

	var headers []types.Header
	var decodedExtraField utils.ExtraFields_v2
	// Decode the qc1 and qc2
	err := utils.DecodeBytesExtraFields(currentBlock.Header().Extra, &decodedExtraField)
	assert.Nil(t, err)
	err = forensics.SetCommittedQCs(append(headers, *blockchain.GetHeaderByNumber(903), *blockchain.GetHeaderByNumber(902)), *decodedExtraField.QuorumCert)
	assert.NotNil(t, err)
	assert.Equal(t, "Headers shall be on the same chain and in the right order", err.Error())

	err = forensics.SetCommittedQCs(append(headers, *blockchain.GetHeaderByNumber(903), *blockchain.GetHeaderByNumber(904)), *decodedExtraField.QuorumCert)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(forensics.HighestCommittedQCs))
}
