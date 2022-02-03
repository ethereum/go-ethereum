package tests

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/XinFinOrg/XDPoSChain/accounts"
	"github.com/XinFinOrg/XDPoSChain/accounts/abi/bind/backends"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/stretchr/testify/assert"
)

// VoteHandler
func TestVoteMessageHandlerSuccessfullyGeneratedAndProcessQCForFistV2Round(t *testing.T) {
	blockchain, _, currentBlock, signer, signFn, _ := PrepareXDCTestBlockChainForV2Engine(t, 11, params.TestXDPoSMockChainConfigWithV2Engine, 0)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	blockInfo := &utils.BlockInfo{
		Hash:   currentBlock.Hash(),
		Round:  utils.Round(1),
		Number: big.NewInt(11),
	}
	voteSigningHash := utils.VoteSigHash(blockInfo)

	// Set round to 5
	engineV2.SetNewRoundFaker(utils.Round(1), false)
	// Create two vote messages which will not reach vote pool threshold
	signedHash, err := signFn(accounts.Account{Address: signer}, voteSigningHash.Bytes())
	assert.Nil(t, err)
	voteMsg := &utils.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         signedHash,
	}

	err = engineV2.VoteHandler(blockchain, voteMsg)
	assert.Nil(t, err)
	currentRound, lockQuorumCert, highestQuorumCert, _, _ := engineV2.GetProperties()
	// initialised with nil and 0 round
	assert.Nil(t, lockQuorumCert)
	assert.Equal(t, utils.Round(0), highestQuorumCert.ProposedBlockInfo.Round)
	assert.Equal(t, utils.Round(1), currentRound)

	signedHash = SignHashByPK(acc2Key, voteSigningHash.Bytes())

	voteMsg = &utils.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         signedHash,
	}
	err = engineV2.VoteHandler(blockchain, voteMsg)
	assert.Nil(t, err)
	currentRound, lockQuorumCert, highestQuorumCert, _, _ = engineV2.GetProperties()
	// Still using the initlised value because we did not yet go to the next round
	assert.Nil(t, lockQuorumCert)
	assert.Equal(t, utils.Round(0), highestQuorumCert.ProposedBlockInfo.Round)

	assert.Equal(t, utils.Round(1), currentRound)

	// Create a vote message that should trigger vote pool hook and increment the round to 6
	signedHash = SignHashByPK(acc3Key, voteSigningHash.Bytes())
	voteMsg = &utils.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         signedHash,
	}

	err = engineV2.VoteHandler(blockchain, voteMsg)
	assert.Nil(t, err)
	currentRound, lockQuorumCert, highestQuorumCert, _, _ = engineV2.GetProperties()
	// The lockQC shall be the parent's QC round number
	assert.Equal(t, utils.Round(0), lockQuorumCert.ProposedBlockInfo.Round)
	// The highestQC proposedBlockInfo shall be the same as the one from its votes
	assert.Equal(t, highestQuorumCert.ProposedBlockInfo, voteMsg.ProposedBlockInfo)
	// Check round has now changed from 1 to 2
	assert.Equal(t, utils.Round(2), currentRound)
}

func TestVoteMessageHandlerSuccessfullyGeneratedAndProcessQC(t *testing.T) {
	blockchain, _, currentBlock, signer, signFn, _ := PrepareXDCTestBlockChainForV2Engine(t, 15, params.TestXDPoSMockChainConfigWithV2Engine, 0)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	blockInfo := &utils.BlockInfo{
		Hash:   currentBlock.Hash(),
		Round:  utils.Round(5),
		Number: big.NewInt(15),
	}
	voteSigningHash := utils.VoteSigHash(blockInfo)

	// Set round to 5
	engineV2.SetNewRoundFaker(utils.Round(5), false)
	// Create two vote messages which will not reach vote pool threshold
	signedHash, err := signFn(accounts.Account{Address: signer}, voteSigningHash.Bytes())
	assert.Nil(t, err)
	voteMsg := &utils.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         signedHash,
	}

	err = engineV2.VoteHandler(blockchain, voteMsg)
	assert.Nil(t, err)
	currentRound, lockQuorumCert, highestQuorumCert, _, _ := engineV2.GetProperties()
	// initialised with nil and 0 round
	assert.Nil(t, lockQuorumCert)
	assert.Equal(t, utils.Round(0), highestQuorumCert.ProposedBlockInfo.Round)
	assert.Equal(t, utils.Round(5), currentRound)
	signedHash = SignHashByPK(acc1Key, voteSigningHash.Bytes())
	voteMsg = &utils.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         signedHash,
	}
	err = engineV2.VoteHandler(blockchain, voteMsg)
	assert.Nil(t, err)
	currentRound, lockQuorumCert, highestQuorumCert, _, _ = engineV2.GetProperties()
	// Still using the initlised value because we did not yet go to the next round
	assert.Nil(t, lockQuorumCert)
	assert.Equal(t, utils.Round(0), highestQuorumCert.ProposedBlockInfo.Round)

	assert.Equal(t, utils.Round(5), currentRound)

	// Create another vote which is signed by someone not from the master node list
	randomSigner, randomSignFn, err := backends.SimulateWalletAddressAndSignFn()
	assert.Nil(t, err)
	randomlySignedHash, err := randomSignFn(accounts.Account{Address: randomSigner}, voteSigningHash.Bytes())
	assert.Nil(t, err)
	voteMsg = &utils.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         randomlySignedHash,
	}
	err = engineV2.VoteHandler(blockchain, voteMsg)
	assert.Nil(t, err)
	currentRound, lockQuorumCert, highestQuorumCert, _, _ = engineV2.GetProperties()
	// Still using the initlised value because we did not yet go to the next round
	assert.Nil(t, lockQuorumCert)
	assert.Equal(t, utils.Round(0), highestQuorumCert.ProposedBlockInfo.Round)
	assert.Equal(t, utils.Round(5), currentRound)

	// Create a vote message that should trigger vote pool hook and increment the round to 6
	signedHash = SignHashByPK(acc3Key, voteSigningHash.Bytes())
	voteMsg = &utils.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         signedHash,
	}

	err = engineV2.VoteHandler(blockchain, voteMsg)
	assert.Nil(t, err)
	currentRound, lockQuorumCert, highestQuorumCert, _, highestCommitBlock := engineV2.GetProperties()
	// The lockQC shall be the parent's QC round number
	assert.Equal(t, utils.Round(4), lockQuorumCert.ProposedBlockInfo.Round)
	// The highestQC proposedBlockInfo shall be the same as the one from its votes
	assert.Equal(t, highestQuorumCert.ProposedBlockInfo, voteMsg.ProposedBlockInfo)
	// Check round has now changed from 5 to 6
	assert.Equal(t, utils.Round(6), currentRound)
	// Should trigger ProcessQC and trying to commit from blockNum of 16's grandgrandparent which is blockNum 13 with round 3
	assert.Equal(t, utils.Round(3), highestCommitBlock.Round)
	assert.Equal(t, big.NewInt(13), highestCommitBlock.Number)
}

func TestThrowErrorIfVoteMsgRoundIsMoreThanOneRoundAwayFromCurrentRound(t *testing.T) {
	blockchain, _, _, _, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 15, params.TestXDPoSMockChainConfigWithV2Engine, 0)
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
	assert.Equal(t, "vote message round number: 6 is too far away from currentRound: 7", err.Error())

	// Set round to 5, it's 1 round away, should not trigger failure
	engineV2.SetNewRoundFaker(utils.Round(5), false)
	err = engineV2.VoteHandler(blockchain, voteMsg)
	assert.Nil(t, err)

	engineV2.SetNewRoundFaker(utils.Round(4), false)
	err = engineV2.VoteHandler(blockchain, voteMsg)
	assert.NotNil(t, err)
	assert.Equal(t, "vote message round number: 6 is too far away from currentRound: 4", err.Error())

}

func TestProcessVoteMsgThenTimeoutMsg(t *testing.T) {
	blockchain, _, currentBlock, _, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 15, params.TestXDPoSMockChainConfigWithV2Engine, 0)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	// Set round to 5
	engineV2.SetNewRoundFaker(utils.Round(5), false)

	// Start with vote messages
	blockInfo := &utils.BlockInfo{
		Hash:   currentBlock.Hash(),
		Round:  utils.Round(5),
		Number: big.NewInt(11),
	}
	voteSigningHash := utils.VoteSigHash(blockInfo)
	// Create two vote message which will not reach vote pool threshold
	signedHash := SignHashByPK(acc1Key, voteSigningHash.Bytes())
	voteMsg := &utils.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         signedHash,
	}

	err := engineV2.VoteHandler(blockchain, voteMsg)
	assert.Nil(t, err)
	currentRound, lockQuorumCert, highestQuorumCert, _, _ := engineV2.GetProperties()
	// initialised with nil and 0 round
	assert.Nil(t, lockQuorumCert)
	assert.Equal(t, utils.Round(0), highestQuorumCert.ProposedBlockInfo.Round)

	assert.Equal(t, utils.Round(5), currentRound)
	voteMsg = &utils.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         SignHashByPK(acc2Key, voteSigningHash.Bytes()),
	}
	err = engineV2.VoteHandler(blockchain, voteMsg)
	assert.Nil(t, err)
	currentRound, _, _, _, _ = engineV2.GetProperties()
	assert.Equal(t, utils.Round(5), currentRound)

	// Create a vote message that should trigger vote pool hook
	voteMsg = &utils.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         SignHashByPK(acc3Key, voteSigningHash.Bytes()),
	}

	err = engineV2.VoteHandler(blockchain, voteMsg)
	assert.Nil(t, err)
	// Check round has now changed from 5 to 6
	currentRound, lockQuorumCert, highestQuorumCert, _, _ = engineV2.GetProperties()
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
	assert.Equal(t, "timeout message round number: 5 does not match currentRound: 6", err.Error())

	// Ok, let's do the timeout msg which is on the same round as the current round by creating two timeout message which will not reach timeout pool threshold
	timeoutMsg = &utils.Timeout{
		Round:     utils.Round(6),
		Signature: []byte{1},
	}

	err = engineV2.TimeoutHandler(timeoutMsg)
	assert.Nil(t, err)
	currentRound, _, _, _, _ = engineV2.GetProperties()
	assert.Equal(t, utils.Round(6), currentRound)
	timeoutMsg = &utils.Timeout{
		Round:     utils.Round(6),
		Signature: []byte{2},
	}
	err = engineV2.TimeoutHandler(timeoutMsg)
	assert.Nil(t, err)
	currentRound, _, _, _, _ = engineV2.GetProperties()
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
	currentRound, _, _, _, _ = engineV2.GetProperties()
	assert.Equal(t, utils.Round(7), currentRound)
}

func TestVoteMessageShallNotThrowErrorIfBlockNotYetExist(t *testing.T) {
	blockchain, _, currentBlock, signer, signFn, _ := PrepareXDCTestBlockChainForV2Engine(t, 15, params.TestXDPoSMockChainConfigWithV2Engine, 0)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	// Create a new block but don't inject it into the chain yet
	blockNum := 16
	blockCoinBase := fmt.Sprintf("0x111000000000000000000000000000000%03d", blockNum)
	block := CreateBlock(blockchain, params.TestXDPoSMockChainConfigWithV2Engine, currentBlock, blockNum, 6, blockCoinBase, signer, signFn)

	blockInfo := &utils.BlockInfo{
		Hash:   block.Header().Hash(),
		Round:  utils.Round(6),
		Number: big.NewInt(16),
	}
	voteSigningHash := utils.VoteSigHash(blockInfo)

	// Set round to 6
	engineV2.SetNewRoundFaker(utils.Round(6), false)
	// Create two vote messages which will not reach vote pool threshold
	voteMsg := &utils.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         SignHashByPK(acc1Key, voteSigningHash.Bytes()),
	}

	err := engineV2.VoteHandler(blockchain, voteMsg)
	assert.Nil(t, err)

	voteMsg = &utils.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         SignHashByPK(acc2Key, voteSigningHash.Bytes()),
	}
	err = engineV2.VoteHandler(blockchain, voteMsg)
	assert.Nil(t, err)

	// Create a vote message that should trigger vote pool hook, but it shall not produce any QC yet
	voteMsg = &utils.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         SignHashByPK(acc3Key, voteSigningHash.Bytes()),
	}

	err = engineV2.VoteHandler(blockchain, voteMsg)
	assert.Nil(t, err)
	currentRound, lockQuorumCert, highestQuorumCert, _, _ := engineV2.GetProperties()
	// Still using the initlised value because we did not yet go to the next round
	assert.Nil(t, lockQuorumCert)
	assert.Equal(t, utils.Round(0), highestQuorumCert.ProposedBlockInfo.Round)

	assert.Equal(t, utils.Round(6), currentRound)

	// Now, inject the block into the chain
	blockchain.InsertBlock(block)

	voteMsg = &utils.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         SignHashByPK(voterKey, voteSigningHash.Bytes()),
	}

	err = engineV2.VoteHandler(blockchain, voteMsg)
	assert.Nil(t, err)

	currentRound, lockQuorumCert, highestQuorumCert, _, highestCommitBlock := engineV2.GetProperties()
	// The lockQC shall be the parent's QC round number
	assert.Equal(t, utils.Round(5), lockQuorumCert.ProposedBlockInfo.Round)
	// The highestQC proposedBlockInfo shall be the same as the one from its votes
	assert.Equal(t, highestQuorumCert.ProposedBlockInfo, voteMsg.ProposedBlockInfo)
	assert.Equal(t, utils.Round(7), currentRound)
	// Should trigger ProcessQC and trying to commit from blockNum of 16's grandgrandparent which is blockNum 14 with round 4
	assert.Equal(t, utils.Round(4), highestCommitBlock.Round)
	assert.Equal(t, big.NewInt(14), highestCommitBlock.Number)
}
