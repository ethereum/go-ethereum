package engine_v2_tests

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/XinFinOrg/XDPoSChain/accounts"
	"github.com/XinFinOrg/XDPoSChain/accounts/abi/bind/backends"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/stretchr/testify/assert"
)

// VoteHandler
func TestVoteMessageHandlerSuccessfullyGeneratedAndProcessQCForFistV2Round(t *testing.T) {
	blockchain, _, currentBlock, signer, signFn, _ := PrepareXDCTestBlockChainForV2Engine(t, 901, params.TestXDPoSMockChainConfig, nil)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	blockInfo := &types.BlockInfo{
		Hash:   currentBlock.Hash(),
		Round:  types.Round(1),
		Number: big.NewInt(901),
	}
	voteForSign := &types.VoteForSign{
		ProposedBlockInfo: blockInfo,
		GapNumber:         450,
	}
	voteSigningHash := types.VoteSigHash(voteForSign)

	// Set round to 1
	engineV2.SetNewRoundFaker(blockchain, types.Round(1), false)
	// Create two vote messages which will not reach vote pool threshold
	signedHash, err := signFn(accounts.Account{Address: signer}, voteSigningHash.Bytes())
	assert.Nil(t, err)
	voteMsg := &types.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         signedHash,
		GapNumber:         450,
	}

	err = engineV2.VoteHandler(blockchain, voteMsg)
	assert.Nil(t, err)
	currentRound, lockQuorumCert, highestQuorumCert, _, _, _ := engineV2.GetPropertiesFaker()
	// initialised with nil and 0 round
	assert.Nil(t, lockQuorumCert)
	assert.Equal(t, types.Round(0), highestQuorumCert.ProposedBlockInfo.Round)
	assert.Equal(t, types.Round(1), currentRound)

	signedHash = SignHashByPK(acc2Key, voteSigningHash.Bytes())

	voteMsg = &types.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         signedHash,
		GapNumber:         450,
	}
	err = engineV2.VoteHandler(blockchain, voteMsg)
	assert.Nil(t, err)
	currentRound, lockQuorumCert, highestQuorumCert, _, _, _ = engineV2.GetPropertiesFaker()
	// Still using the initlised value because we did not yet go to the next round
	assert.Nil(t, lockQuorumCert)
	assert.Equal(t, types.Round(0), highestQuorumCert.ProposedBlockInfo.Round)

	assert.Equal(t, types.Round(1), currentRound)

	// Create a vote message that should trigger vote pool hook and increment the round to 6
	signedHash = SignHashByPK(acc3Key, voteSigningHash.Bytes())
	voteMsg = &types.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         signedHash,
		GapNumber:         450,
	}

	err = engineV2.VoteHandler(blockchain, voteMsg)
	assert.Nil(t, err)
	currentRound, lockQuorumCert, highestQuorumCert, _, _, _ = engineV2.GetPropertiesFaker()
	// The lockQC shall be the parent's QC round number
	assert.Equal(t, types.Round(0), lockQuorumCert.ProposedBlockInfo.Round)
	// The highestQC proposedBlockInfo shall be the same as the one from its votes
	assert.Equal(t, highestQuorumCert.ProposedBlockInfo, voteMsg.ProposedBlockInfo)
	// Check round has now changed from 1 to 2
	assert.Equal(t, types.Round(2), currentRound)
}

func TestVoteMessageHandlerSuccessfullyGeneratedAndProcessQC(t *testing.T) {
	blockchain, _, currentBlock, signer, signFn, _ := PrepareXDCTestBlockChainForV2Engine(t, 905, params.TestXDPoSMockChainConfig, nil)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	blockInfo := &types.BlockInfo{
		Hash:   currentBlock.Hash(),
		Round:  types.Round(5),
		Number: big.NewInt(905),
	}
	voteForSign := &types.VoteForSign{
		ProposedBlockInfo: blockInfo,
		GapNumber:         450,
	}
	voteSigningHash := types.VoteSigHash(voteForSign)

	// Set round to 5
	engineV2.SetNewRoundFaker(blockchain, types.Round(5), false)
	// Create two vote messages which will not reach vote pool threshold
	signedHash, err := signFn(accounts.Account{Address: signer}, voteSigningHash.Bytes())
	assert.Nil(t, err)
	voteMsg := &types.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         signedHash,
		GapNumber:         450,
	}

	err = engineV2.VoteHandler(blockchain, voteMsg)
	assert.Nil(t, err)
	currentRound, lockQuorumCert, highestQuorumCert, _, _, _ := engineV2.GetPropertiesFaker()
	// initialised with nil and 0 round
	assert.Nil(t, lockQuorumCert)
	assert.Equal(t, types.Round(0), highestQuorumCert.ProposedBlockInfo.Round)
	assert.Equal(t, types.Round(5), currentRound)
	signedHash = SignHashByPK(acc1Key, voteSigningHash.Bytes())
	voteMsg = &types.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         signedHash,
		GapNumber:         450,
	}
	err = engineV2.VoteHandler(blockchain, voteMsg)
	assert.Nil(t, err)
	currentRound, lockQuorumCert, highestQuorumCert, _, _, _ = engineV2.GetPropertiesFaker()
	// Still using the initlised value because we did not yet go to the next round
	assert.Nil(t, lockQuorumCert)
	assert.Equal(t, types.Round(0), highestQuorumCert.ProposedBlockInfo.Round)

	assert.Equal(t, types.Round(5), currentRound)

	// Create another vote which is signed by someone not from the master node list

	randomSigner, randomSignFn, err := backends.SimulateWalletAddressAndSignFn()
	assert.Nil(t, err)
	randomlySignedHash, err := randomSignFn(accounts.Account{Address: randomSigner}, voteSigningHash.Bytes())
	assert.Nil(t, err)
	voteMsg = &types.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         randomlySignedHash,
		GapNumber:         450,
	}
	err = engineV2.VoteHandler(blockchain, voteMsg)
	assert.Nil(t, err)

	currentRound, lockQuorumCert, highestQuorumCert, _, _, _ = engineV2.GetPropertiesFaker()
	// Still using the initlised value because we did not yet go to the next round
	assert.Nil(t, lockQuorumCert)
	assert.Equal(t, types.Round(0), highestQuorumCert.ProposedBlockInfo.Round)
	assert.Equal(t, types.Round(5), currentRound)

	// Create a vote message that should trigger vote pool hook and increment the round to 6
	signedHash = SignHashByPK(acc3Key, voteSigningHash.Bytes())
	voteMsg = &types.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         signedHash,
		GapNumber:         450,
	}

	err = engineV2.VoteHandler(blockchain, voteMsg)
	assert.Nil(t, err)
	currentRound, lockQuorumCert, highestQuorumCert, _, _, highestCommitBlock := engineV2.GetPropertiesFaker()
	// The lockQC shall be the parent's QC round number
	assert.Equal(t, types.Round(4), lockQuorumCert.ProposedBlockInfo.Round)
	// The highestQC proposedBlockInfo shall be the same as the one from its votes
	assert.Equal(t, highestQuorumCert.ProposedBlockInfo, voteMsg.ProposedBlockInfo)
	// Check round has now changed from 5 to 6
	assert.Equal(t, types.Round(6), currentRound)
	// Should trigger ProcessQC and trying to commit from blockNum of 16's grandgrandparent which is blockNum 903 with round 3
	assert.Equal(t, types.Round(3), highestCommitBlock.Round)
	assert.Equal(t, big.NewInt(903), highestCommitBlock.Number)
}

func TestThrowErrorIfVoteMsgRoundIsMoreThanOneRoundAwayFromCurrentRound(t *testing.T) {
	blockchain, _, _, _, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 905, params.TestXDPoSMockChainConfig, nil)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	blockInfo := &types.BlockInfo{
		Hash:   common.HexToHash("0x1"),
		Round:  types.Round(6),
		Number: big.NewInt(999),
	}

	// Set round to 7
	engineV2.SetNewRoundFaker(blockchain, types.Round(7), false)
	voteMsg := &types.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         []byte{1},
		GapNumber:         450,
	}

	// voteRound > currentRound
	err := engineV2.VoteHandler(blockchain, voteMsg)
	assert.NotNil(t, err)
	assert.Equal(t, "vote message round number: 6 is too far away from currentRound: 7", err.Error())

	// Set round to 5, it's 1 round away, should not trigger failure
	engineV2.SetNewRoundFaker(blockchain, types.Round(5), false)
	err = engineV2.VoteHandler(blockchain, voteMsg)
	assert.Nil(t, err)

	engineV2.SetNewRoundFaker(blockchain, types.Round(4), false)
	err = engineV2.VoteHandler(blockchain, voteMsg)
	assert.NotNil(t, err)
	assert.Equal(t, "vote message round number: 6 is too far away from currentRound: 4", err.Error())

}

func TestProcessVoteMsgThenTimeoutMsg(t *testing.T) {
	blockchain, _, currentBlock, _, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 905, params.TestXDPoSMockChainConfig, nil)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	// Set round to 5
	engineV2.SetNewRoundFaker(blockchain, types.Round(5), false)

	// Start with vote messages
	blockInfo := &types.BlockInfo{
		Hash:   currentBlock.Hash(),
		Round:  types.Round(5),
		Number: big.NewInt(905),
	}
	voteForSign := &types.VoteForSign{
		ProposedBlockInfo: blockInfo,
		GapNumber:         450,
	}
	voteSigningHash := types.VoteSigHash(voteForSign)
	// Create two vote message which will not reach vote pool threshold
	signedHash := SignHashByPK(acc1Key, voteSigningHash.Bytes())
	voteMsg := &types.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         signedHash,
		GapNumber:         450,
	}

	err := engineV2.VoteHandler(blockchain, voteMsg)
	assert.Nil(t, err)
	currentRound, lockQuorumCert, highestQuorumCert, _, _, _ := engineV2.GetPropertiesFaker()
	// initialised with nil and 0 round
	assert.Nil(t, lockQuorumCert)
	assert.Equal(t, types.Round(0), highestQuorumCert.ProposedBlockInfo.Round)

	assert.Equal(t, types.Round(5), currentRound)
	voteMsg = &types.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         SignHashByPK(acc2Key, voteSigningHash.Bytes()),
		GapNumber:         450,
	}
	err = engineV2.VoteHandler(blockchain, voteMsg)
	assert.Nil(t, err)
	currentRound, _, _, _, _, _ = engineV2.GetPropertiesFaker()
	assert.Equal(t, types.Round(5), currentRound)

	// Create a vote message that should trigger vote pool hook
	voteMsg = &types.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         SignHashByPK(acc3Key, voteSigningHash.Bytes()),
		GapNumber:         450,
	}

	err = engineV2.VoteHandler(blockchain, voteMsg)
	assert.Nil(t, err)
	// Check round has now changed from 5 to 6
	currentRound, lockQuorumCert, highestQuorumCert, _, _, _ = engineV2.GetPropertiesFaker()
	// The lockQC shall be the parent's QC round number
	assert.Equal(t, types.Round(4), lockQuorumCert.ProposedBlockInfo.Round)
	// The highestQC proposedBlockInfo shall be the same as the one from its votes
	assert.Equal(t, highestQuorumCert.ProposedBlockInfo, voteMsg.ProposedBlockInfo)

	assert.Equal(t, types.Round(6), currentRound)

	// We shall have highestQuorumCert in engine now, let's do timeout msg to see if we can broadcast SyncInfo which contains both highestQuorumCert and HighestTimeoutCert

	// First, all incoming old timeout msg shall not be processed
	timeoutMsg := &types.Timeout{
		Round:     types.Round(5),
		Signature: []byte{1},
	}

	err = engineV2.TimeoutHandler(blockchain, timeoutMsg)
	assert.NotNil(t, err)
	assert.Equal(t, "timeout message round number: 5 does not match currentRound: 6", err.Error())

	// Ok, let's do the timeout msg which is on the same round as the current round by creating two timeout message which will not reach timeout pool threshold
	timeoutMsg = &types.Timeout{
		Round:     types.Round(6),
		Signature: []byte{1},
	}

	err = engineV2.TimeoutHandler(blockchain, timeoutMsg)
	assert.Nil(t, err)
	currentRound, _, _, _, _, _ = engineV2.GetPropertiesFaker()
	assert.Equal(t, types.Round(6), currentRound)
	timeoutMsg = &types.Timeout{
		Round:     types.Round(6),
		Signature: []byte{2},
	}
	err = engineV2.TimeoutHandler(blockchain, timeoutMsg)
	assert.Nil(t, err)
	currentRound, _, _, _, _, _ = engineV2.GetPropertiesFaker()
	assert.Equal(t, types.Round(6), currentRound)

	// Create a timeout message that should trigger timeout pool hook
	timeoutMsg = &types.Timeout{
		Round:     types.Round(6),
		Signature: []byte{3},
	}

	err = engineV2.TimeoutHandler(blockchain, timeoutMsg)
	assert.Nil(t, err)

	syncInfoMsg := <-engineV2.BroadcastCh
	assert.NotNil(t, syncInfoMsg)

	// Should have HighestQuorumCert from previous round votes
	qc := syncInfoMsg.(*types.SyncInfo).HighestQuorumCert
	assert.NotNil(t, qc)
	assert.Equal(t, types.Round(5), qc.ProposedBlockInfo.Round)

	tc := syncInfoMsg.(*types.SyncInfo).HighestTimeoutCert
	assert.NotNil(t, tc)
	assert.Equal(t, types.Round(6), tc.Round)
	sigatures := []types.Signature{[]byte{1}, []byte{2}, []byte{3}}
	assert.ElementsMatch(t, tc.Signatures, sigatures)
	// Round shall be +1 now
	currentRound, _, _, _, _, _ = engineV2.GetPropertiesFaker()
	assert.Equal(t, types.Round(7), currentRound)
}

func TestVoteMessageShallNotThrowErrorIfBlockNotYetExist(t *testing.T) {
	blockchain, _, currentBlock, signer, signFn, _ := PrepareXDCTestBlockChainForV2Engine(t, 905, params.TestXDPoSMockChainConfig, nil)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	// Create a new block but don't inject it into the chain yet
	blockNum := 906
	blockCoinBase := fmt.Sprintf("0x111000000000000000000000000000000%03d", blockNum)
	block := CreateBlock(blockchain, params.TestXDPoSMockChainConfig, currentBlock, blockNum, 6, blockCoinBase, signer, signFn, nil, nil, "")

	blockInfo := &types.BlockInfo{
		Hash:   block.Header().Hash(),
		Round:  types.Round(6),
		Number: big.NewInt(906),
	}
	voteForSign := &types.VoteForSign{
		ProposedBlockInfo: blockInfo,
		GapNumber:         450,
	}
	voteSigningHash := types.VoteSigHash(voteForSign)

	// Set round to 6
	engineV2.SetNewRoundFaker(blockchain, types.Round(6), false)
	// Create two vote messages which will not reach vote pool threshold
	voteMsg := &types.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         SignHashByPK(acc1Key, voteSigningHash.Bytes()),
		GapNumber:         450,
	}

	err := engineV2.VoteHandler(blockchain, voteMsg)
	assert.Nil(t, err)

	voteMsg = &types.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         SignHashByPK(acc2Key, voteSigningHash.Bytes()),
		GapNumber:         450,
	}
	err = engineV2.VoteHandler(blockchain, voteMsg)
	assert.Nil(t, err)

	// Create a vote message that should trigger vote pool hook, but it shall not produce any QC yet
	voteMsg = &types.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         SignHashByPK(acc3Key, voteSigningHash.Bytes()),
		GapNumber:         450,
	}

	err = engineV2.VoteHandler(blockchain, voteMsg)
	assert.Nil(t, err)
	currentRound, lockQuorumCert, highestQuorumCert, _, _, _ := engineV2.GetPropertiesFaker()
	// Still using the initlised value because we did not yet go to the next round
	assert.Nil(t, lockQuorumCert)
	assert.Equal(t, types.Round(0), highestQuorumCert.ProposedBlockInfo.Round)

	assert.Equal(t, types.Round(6), currentRound)

	// Now, inject the block into the chain
	err = blockchain.InsertBlock(block)
	assert.Nil(t, err)

	voteMsg = &types.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         SignHashByPK(voterKey, voteSigningHash.Bytes()),
		GapNumber:         450,
	}

	err = engineV2.VoteHandler(blockchain, voteMsg)
	assert.Nil(t, err)

	currentRound, lockQuorumCert, highestQuorumCert, _, _, highestCommitBlock := engineV2.GetPropertiesFaker()
	// The lockQC shall be the parent's QC round number
	assert.Equal(t, types.Round(5), lockQuorumCert.ProposedBlockInfo.Round)
	// The highestQC proposedBlockInfo shall be the same as the one from its votes
	assert.Equal(t, highestQuorumCert.ProposedBlockInfo, voteMsg.ProposedBlockInfo)
	assert.Equal(t, types.Round(7), currentRound)
	// Should trigger ProcessQC and trying to commit from blockNum of 16's grandgrandparent which is blockNum 904 with round 4
	assert.Equal(t, types.Round(4), highestCommitBlock.Round)
	assert.Equal(t, big.NewInt(904), highestCommitBlock.Number)
}

func TestProcessVoteMsgFailIfVerifyBlockInfoFail(t *testing.T) {
	blockchain, _, currentBlock, _, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 905, params.TestXDPoSMockChainConfig, nil)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	// Set round to 5
	engineV2.SetNewRoundFaker(blockchain, types.Round(5), false)

	// Start with vote messages
	blockInfo := &types.BlockInfo{
		Hash:   currentBlock.ParentHash(),
		Round:  types.Round(5),
		Number: big.NewInt(905),
	}
	voteForSign := &types.VoteForSign{
		ProposedBlockInfo: blockInfo,
		GapNumber:         450,
	}
	voteSigningHash := types.VoteSigHash(voteForSign)
	// Create two vote message which will not reach vote pool threshold
	signedHash := SignHashByPK(acc1Key, voteSigningHash.Bytes())
	voteMsg := &types.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         signedHash,
		GapNumber:         450,
	}

	err := engineV2.VoteHandler(blockchain, voteMsg)
	assert.Nil(t, err)
	currentRound, lockQuorumCert, highestQuorumCert, _, _, _ := engineV2.GetPropertiesFaker()
	// initialised with nil and 0 round
	assert.Nil(t, lockQuorumCert)
	assert.Equal(t, types.Round(0), highestQuorumCert.ProposedBlockInfo.Round)

	assert.Equal(t, types.Round(5), currentRound)
	voteMsg = &types.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         SignHashByPK(acc2Key, voteSigningHash.Bytes()),
		GapNumber:         450,
	}
	err = engineV2.VoteHandler(blockchain, voteMsg)
	assert.Nil(t, err)
	currentRound, _, _, _, _, _ = engineV2.GetPropertiesFaker()
	assert.Equal(t, types.Round(5), currentRound)

	// Create a vote message that should trigger vote pool hook
	voteMsg = &types.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         SignHashByPK(acc3Key, voteSigningHash.Bytes()),
		GapNumber:         450,
	}

	err = engineV2.VoteHandler(blockchain, voteMsg)
	expectedError := fmt.Errorf("[VerifyBlockInfo] chain header number does not match for the received blockInfo at hash: %v", blockInfo.Hash.Hex())
	assert.Equal(t, expectedError, err)
}

func TestVerifyVoteMsg(t *testing.T) {
	blockchain, _, currentBlock, signer, signFn, _ := PrepareXDCTestBlockChainForV2Engine(t, 915, params.TestXDPoSMockChainConfig, nil)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	blockInfo := &types.BlockInfo{
		Hash:   currentBlock.Hash(),
		Round:  types.Round(14),
		Number: big.NewInt(915),
	}
	voteForSign := &types.VoteForSign{
		ProposedBlockInfo: blockInfo,
		GapNumber:         450,
	}

	// Valid message but disqualified as the round does not match
	voteMsg := &types.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         []byte{1},
		GapNumber:         450,
	}
	engineV2.SetNewRoundFaker(blockchain, types.Round(15), false)
	verified, err := engineV2.VerifyVoteMessage(blockchain, voteMsg)
	assert.False(t, verified)
	assert.Nil(t, err)

	// Invalid vote message with wrong signature
	engineV2.SetNewRoundFaker(blockchain, types.Round(14), false)
	verified, err = engineV2.VerifyVoteMessage(blockchain, voteMsg)
	assert.False(t, verified)
	assert.Equal(t, "Error while verifying message: invalid signature length", err.Error())

	// Valid vote message from a master node
	signHash, _ := signFn(accounts.Account{Address: signer}, types.VoteSigHash(voteForSign).Bytes())
	voteMsg = &types.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         signHash,
		GapNumber:         450,
	}

	verified, err = engineV2.VerifyVoteMessage(blockchain, voteMsg)
	assert.Equal(t, voteMsg.GetSigner(), signer)
	assert.True(t, verified)
	assert.Nil(t, err)
}

func TestVoteMsgMissingSnapshot(t *testing.T) {
	blockchain, _, currentBlock, signer, signFn, _ := PrepareXDCTestBlockChainForV2Engine(t, 915, params.TestXDPoSMockChainConfig, nil)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	blockInfo := &types.BlockInfo{
		Hash:   currentBlock.Hash(),
		Round:  types.Round(14),
		Number: big.NewInt(915),
	}
	voteForSign := &types.VoteForSign{
		ProposedBlockInfo: blockInfo,
		GapNumber:         450,
	}

	signHash, _ := signFn(accounts.Account{Address: signer}, types.VoteSigHash(voteForSign).Bytes())
	voteMsg := &types.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         signHash,
		GapNumber:         1350, // missing 1350 snapshot
	}
	engineV2.SetNewRoundFaker(blockchain, types.Round(14), false)
	verified, err := engineV2.VerifyVoteMessage(blockchain, voteMsg)
	assert.False(t, verified)
	assert.NotNil(t, err)
}

func TestVoteMessageHandlerWrongGapNumber(t *testing.T) {
	blockchain, _, currentBlock, signer, signFn, _ := PrepareXDCTestBlockChainForV2Engine(t, 905, params.TestXDPoSMockChainConfig, nil)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	blockInfo := &types.BlockInfo{
		Hash:   currentBlock.Hash(),
		Round:  types.Round(5),
		Number: big.NewInt(905),
	}
	voteForSign := &types.VoteForSign{
		ProposedBlockInfo: blockInfo,
		GapNumber:         450,
	}
	voteSigningHash := types.VoteSigHash(voteForSign)

	// Set round to 5
	engineV2.SetNewRoundFaker(blockchain, types.Round(5), false)
	// Create two vote messages which will not reach vote pool threshold
	signedHash, _ := signFn(accounts.Account{Address: signer}, voteSigningHash.Bytes())
	voteMsg := &types.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         signedHash,
		GapNumber:         450,
	}
	engineV2.VoteHandler(blockchain, voteMsg)
	signedHash = SignHashByPK(acc1Key, voteSigningHash.Bytes())
	voteMsg = &types.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         signedHash,
		GapNumber:         450,
	}
	engineV2.VoteHandler(blockchain, voteMsg)

	// Create a vote message that has wrong gap number
	voteForSign = &types.VoteForSign{
		ProposedBlockInfo: blockInfo,
		GapNumber:         451,
	}
	voteSigningHash = types.VoteSigHash(voteForSign)
	signedHash = SignHashByPK(acc3Key, voteSigningHash.Bytes())
	voteMsg = &types.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         signedHash,
		GapNumber:         451,
	}

	err := engineV2.VoteHandler(blockchain, voteMsg)
	// Shall not even trigger the vote threashold as vote pool key also contains the gapNumber
	assert.Nil(t, err)
}

func TestVotePoolKeepGoodHygiene(t *testing.T) {
	blockchain, _, currentBlock, signer, signFn, _ := PrepareXDCTestBlockChainForV2Engine(t, 905, params.TestXDPoSMockChainConfig, nil)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	blockInfo := &types.BlockInfo{
		Hash:   currentBlock.Hash(),
		Round:  types.Round(5),
		Number: big.NewInt(905),
	}
	voteForSign := &types.VoteForSign{
		ProposedBlockInfo: blockInfo,
		GapNumber:         450,
	}
	voteSigningHash := types.VoteSigHash(voteForSign)

	// Set round to 5
	engineV2.SetNewRoundFaker(blockchain, types.Round(5), false)
	// Create two vote messages which will not reach vote pool threshold
	signedHash, _ := signFn(accounts.Account{Address: signer}, voteSigningHash.Bytes())
	voteMsg := &types.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         signedHash,
		GapNumber:         450,
	}
	engineV2.VoteHandler(blockchain, voteMsg)

	// Inject a second vote with round 16
	blockInfo = &types.BlockInfo{
		Hash:   currentBlock.Hash(),
		Round:  types.Round(16),
		Number: big.NewInt(906),
	}
	voteForSign = &types.VoteForSign{
		ProposedBlockInfo: blockInfo,
		GapNumber:         450,
	}
	voteSigningHash = types.VoteSigHash(voteForSign)

	// Set round to 16
	engineV2.SetNewRoundFaker(blockchain, types.Round(16), false)
	// Create two vote messages which will not reach vote pool threshold
	signedHash, _ = signFn(accounts.Account{Address: signer}, voteSigningHash.Bytes())
	voteMsg = &types.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         signedHash,
		GapNumber:         450,
	}
	engineV2.VoteHandler(blockchain, voteMsg)

	// Inject a second vote with round 25, which is less than 10 rounds difference to the last vote round
	blockInfo = &types.BlockInfo{
		Hash:   currentBlock.Hash(),
		Round:  types.Round(25),
		Number: big.NewInt(907),
	}
	voteForSign = &types.VoteForSign{
		ProposedBlockInfo: blockInfo,
		GapNumber:         450,
	}
	voteSigningHash = types.VoteSigHash(voteForSign)

	// Set round to 25
	engineV2.SetNewRoundFaker(blockchain, types.Round(25), false)
	// Create two vote messages which will not reach vote pool threshold
	signedHash, _ = signFn(accounts.Account{Address: signer}, voteSigningHash.Bytes())
	voteMsg = &types.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         signedHash,
		GapNumber:         450,
	}
	engineV2.VoteHandler(blockchain, voteMsg)

	// Let's keep good Hygiene
	engineV2.HygieneVotePoolFaker()
	// Let's wait for 5 second for the goroutine
	<-time.After(5 * time.Second)
	keyList := engineV2.GetVotePoolKeyListFaker()

	assert.Equal(t, 2, len(keyList))
	for _, k := range keyList {
		keyedRound, err := strconv.ParseInt(strings.Split(k, ":")[0], 10, 64)
		assert.Nil(t, err)
		if keyedRound < 25-10 {
			assert.Fail(t, "Did not clean up the vote pool")
		}
	}
}
