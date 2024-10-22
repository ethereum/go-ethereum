package engine_v2_tests

import (
	"encoding/json"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/XinFinOrg/XDPoSChain/accounts"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/consensus"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/stretchr/testify/assert"
)

func TestShouldVerifyBlock(t *testing.T) {
	b, err := json.Marshal(params.TestXDPoSMockChainConfig)
	assert.Nil(t, err)
	configString := string(b)

	var config params.ChainConfig
	err = json.Unmarshal([]byte(configString), &config)
	assert.Nil(t, err)
	// Enable verify
	config.XDPoS.V2.SkipV2Validation = false
	// Block 901 is the first v2 block with round of 1
	blockchain, _, _, signer, signFn, _ := PrepareXDCTestBlockChainForV2Engine(t, 910, &config, nil)
	adaptor := blockchain.Engine().(*XDPoS.XDPoS)

	// Happy path
	happyPathHeader := blockchain.GetBlockByNumber(901).Header()
	err = adaptor.VerifyHeader(blockchain, happyPathHeader, true)
	assert.Nil(t, err)

	// Unhappy path

	// Verify non-epoch switch block
	err = adaptor.VerifyHeader(blockchain, blockchain.GetBlockByNumber(902).Header(), true)
	assert.Nil(t, err)

	nonEpochSwitchWithValidators := blockchain.GetBlockByNumber(902).Header()
	nonEpochSwitchWithValidators.Validators = acc1Addr.Bytes()
	err = adaptor.VerifyHeader(blockchain, nonEpochSwitchWithValidators, true)
	assert.Equal(t, utils.ErrInvalidFieldInNonEpochSwitch, err)

	noValidatorBlock := blockchain.GetBlockByNumber(902).Header()
	noValidatorBlock.Validator = []byte{}
	err = adaptor.VerifyHeader(blockchain, noValidatorBlock, true)
	assert.Equal(t, consensus.ErrNoValidatorSignatureV2, err)

	blockFromFuture := blockchain.GetBlockByNumber(902).Header()
	blockFromFuture.Time = big.NewInt(time.Now().Unix() + 10000)
	err = adaptor.VerifyHeader(blockchain, blockFromFuture, true)
	assert.Equal(t, consensus.ErrFutureBlock, err)

	invalidQcBlock := blockchain.GetBlockByNumber(902).Header()
	invalidQcBlock.Extra = []byte{2}
	err = adaptor.VerifyHeader(blockchain, invalidQcBlock, true)
	assert.Equal(t, utils.ErrInvalidV2Extra, err)

	// Epoch switch
	invalidAuthNonceBlock := blockchain.GetBlockByNumber(901).Header()
	invalidAuthNonceBlock.Nonce = types.BlockNonce{123}
	err = adaptor.VerifyHeader(blockchain, invalidAuthNonceBlock, true)
	assert.Equal(t, utils.ErrInvalidVote, err)

	emptyValidatorsBlock := blockchain.GetBlockByNumber(901).Header()
	emptyValidatorsBlock.Validators = []byte{}
	err = adaptor.VerifyHeader(blockchain, emptyValidatorsBlock, true)
	assert.Equal(t, utils.ErrEmptyEpochSwitchValidators, err)

	invalidValidatorsSignerBlock := blockchain.GetBlockByNumber(901).Header()
	invalidValidatorsSignerBlock.Validators = []byte{123}
	err = adaptor.VerifyHeader(blockchain, invalidValidatorsSignerBlock, true)
	assert.Equal(t, utils.ErrInvalidCheckpointSigners, err)

	// non-epoch switch
	invalidValidatorsExistBlock := blockchain.GetBlockByNumber(902).Header()
	invalidValidatorsExistBlock.Validators = []byte{123}
	err = adaptor.VerifyHeader(blockchain, invalidValidatorsExistBlock, true)
	assert.Equal(t, utils.ErrInvalidFieldInNonEpochSwitch, err)

	invalidPenaltiesExistBlock := blockchain.GetBlockByNumber(902).Header()
	invalidPenaltiesExistBlock.Penalties = common.Hex2BytesFixed("123131231", 20)
	err = adaptor.VerifyHeader(blockchain, invalidPenaltiesExistBlock, true)
	assert.Equal(t, utils.ErrInvalidFieldInNonEpochSwitch, err)

	merkleRoot := "35999dded35e8db12de7e6c1471eb9670c162eec616ecebbaf4fddd4676fb123"
	parentNotExistBlock := blockchain.GetBlockByNumber(901).Header()
	parentNotExistBlock.ParentHash = common.HexToHash(merkleRoot)
	err = adaptor.VerifyHeader(blockchain, parentNotExistBlock, true)
	assert.Equal(t, consensus.ErrUnknownAncestor, err)

	block901 := blockchain.GetBlockByNumber(901).Header()
	tooFastMinedBlock := blockchain.GetBlockByNumber(902).Header()
	tooFastMinedBlock.Time = big.NewInt(block901.Time.Int64() - 10)
	err = adaptor.VerifyHeader(blockchain, tooFastMinedBlock, true)
	assert.Equal(t, utils.ErrInvalidTimestamp, err)

	invalidDifficultyBlock := blockchain.GetBlockByNumber(902).Header()
	invalidDifficultyBlock.Difficulty = big.NewInt(2)
	err = adaptor.VerifyHeader(blockchain, invalidDifficultyBlock, true)
	assert.Equal(t, utils.ErrInvalidDifficulty, err)

	// Create an invalid QC round
	proposedBlockInfo := &types.BlockInfo{
		Hash:   blockchain.GetBlockByNumber(902).Hash(),
		Round:  types.Round(2),
		Number: blockchain.GetBlockByNumber(902).Number(),
	}
	voteForSign := &types.VoteForSign{
		ProposedBlockInfo: proposedBlockInfo,
		GapNumber:         450,
	}
	// Genrate QC
	signedHash, err := signFn(accounts.Account{Address: signer}, types.VoteSigHash(voteForSign).Bytes())
	if err != nil {
		panic(fmt.Errorf("error generate QC by creating signedHash: %v", err))
	}
	// Sign from acc 1, 2, 3
	acc1SignedHash := SignHashByPK(acc1Key, types.VoteSigHash(voteForSign).Bytes())
	acc2SignedHash := SignHashByPK(acc2Key, types.VoteSigHash(voteForSign).Bytes())
	acc3SignedHash := SignHashByPK(acc3Key, types.VoteSigHash(voteForSign).Bytes())
	var signatures []types.Signature
	signatures = append(signatures, signedHash, acc1SignedHash, acc2SignedHash, acc3SignedHash)
	quorumCert := &types.QuorumCert{
		ProposedBlockInfo: proposedBlockInfo,
		Signatures:        signatures,
		GapNumber:         450,
	}

	extra := types.ExtraFields_v2{
		Round:      types.Round(2),
		QuorumCert: quorumCert,
	}
	extraInBytes, err := extra.EncodeToBytes()
	if err != nil {
		panic(fmt.Errorf("error encode extra into bytes: %v", err))
	}

	invalidRoundBlock := blockchain.GetBlockByNumber(902).Header()
	invalidRoundBlock.Extra = extraInBytes
	err = adaptor.VerifyHeader(blockchain, invalidRoundBlock, true)
	assert.Equal(t, utils.ErrRoundInvalid, err)

	// Not valid validator
	coinbaseValidatorMismatchBlock := blockchain.GetBlockByNumber(902).Header()
	notQualifiedSigner, notQualifiedSignFn, err := getSignerAndSignFn(voterKey)
	assert.Nil(t, err)
	sealHeader(blockchain, coinbaseValidatorMismatchBlock, notQualifiedSigner, notQualifiedSignFn)
	err = adaptor.VerifyHeader(blockchain, coinbaseValidatorMismatchBlock, true)
	assert.Equal(t, utils.ErrCoinbaseAndValidatorMismatch, err)

	// Make the validators not legit by adding something to the validator
	validatorsNotLegit := blockchain.GetBlockByNumber(901).Header()
	validatorsNotLegit.Validators = append(validatorsNotLegit.Validators, acc1Addr[:]...)
	err = adaptor.VerifyHeader(blockchain, validatorsNotLegit, true)
	assert.Equal(t, utils.ErrValidatorsNotLegit, err)

	// Make the penalties not legit by adding something to the penalty
	penaltiesNotLegit := blockchain.GetBlockByNumber(901).Header()
	penaltiesNotLegit.Penalties = append(penaltiesNotLegit.Penalties, acc1Addr[:]...)
	err = adaptor.VerifyHeader(blockchain, penaltiesNotLegit, true)
	assert.Equal(t, utils.ErrPenaltiesNotLegit, err)
}

func TestConfigSwitchOnDifferentCertThreshold(t *testing.T) {
	b, err := json.Marshal(params.TestXDPoSMockChainConfig)
	assert.Nil(t, err)
	configString := string(b)

	var config params.ChainConfig
	err = json.Unmarshal([]byte(configString), &config)
	assert.Nil(t, err)
	// Enable verify
	config.XDPoS.V2.SkipV2Validation = false
	// Block 901 is the first v2 block with round of 1
	blockchain, _, _, _, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 915, &config, nil)

	adaptor := blockchain.Engine().(*XDPoS.XDPoS)

	// Genrate 911 QC
	proposedBlockInfo := &types.BlockInfo{
		Hash:   blockchain.GetBlockByNumber(911).Hash(),
		Round:  types.Round(11),
		Number: blockchain.GetBlockByNumber(911).Number(),
	}
	voteForSign := &types.VoteForSign{
		ProposedBlockInfo: proposedBlockInfo,
		GapNumber:         450,
	}

	// Sign from acc 1, 2, 3
	acc1SignedHash := SignHashByPK(acc1Key, types.VoteSigHash(voteForSign).Bytes())
	acc2SignedHash := SignHashByPK(acc2Key, types.VoteSigHash(voteForSign).Bytes())
	acc3SignedHash := SignHashByPK(acc3Key, types.VoteSigHash(voteForSign).Bytes())
	var signaturesFirst []types.Signature
	signaturesFirst = append(signaturesFirst, acc1SignedHash, acc2SignedHash, acc3SignedHash)
	quorumCert := &types.QuorumCert{
		ProposedBlockInfo: proposedBlockInfo,
		Signatures:        signaturesFirst,
		GapNumber:         450,
	}

	extra := types.ExtraFields_v2{
		Round:      types.Round(12),
		QuorumCert: quorumCert,
	}
	extraInBytes, _ := extra.EncodeToBytes()

	// after 910 require 4 signs, but we only give 3 signs
	block912 := blockchain.GetBlockByNumber(912).Header()
	block912.Extra = extraInBytes
	err = adaptor.VerifyHeader(blockchain, block912, true)

	assert.Equal(t, utils.ErrInvalidQCSignatures, err)

	// Make we verification process use the corresponding config
	// Genrate 910 QC
	proposedBlockInfo = &types.BlockInfo{
		Hash:   blockchain.GetBlockByNumber(910).Hash(),
		Round:  types.Round(10),
		Number: blockchain.GetBlockByNumber(910).Number(),
	}
	voteForSign = &types.VoteForSign{
		ProposedBlockInfo: proposedBlockInfo,
		GapNumber:         450,
	}

	// Sign from acc 1, 2, 3
	acc1SignedHash = SignHashByPK(acc1Key, types.VoteSigHash(voteForSign).Bytes())
	acc2SignedHash = SignHashByPK(acc2Key, types.VoteSigHash(voteForSign).Bytes())
	acc3SignedHash = SignHashByPK(acc3Key, types.VoteSigHash(voteForSign).Bytes())
	voteSignedHash := SignHashByPK(voterKey, types.VoteSigHash(voteForSign).Bytes())

	var signaturesThr []types.Signature
	signaturesThr = append(signaturesThr, acc1SignedHash, acc2SignedHash, acc3SignedHash, voteSignedHash)
	quorumCert = &types.QuorumCert{
		ProposedBlockInfo: proposedBlockInfo,
		Signatures:        signaturesThr,
		GapNumber:         450,
	}

	extra = types.ExtraFields_v2{
		Round:      types.Round(11),
		QuorumCert: quorumCert,
	}
	extraInBytes, _ = extra.EncodeToBytes()

	// QC contains 910, so it requires 3 signatures, not use block number to determine which config to use
	block911 := blockchain.GetBlockByNumber(911).Header()
	block911.Extra = extraInBytes
	err = adaptor.VerifyHeader(blockchain, block911, true)

	// error ErrValidatorNotWithinMasternodes means verifyQC is passed and move to next verification process
	assert.Equal(t, utils.ErrValidatorNotWithinMasternodes, err)
}

/*
 1. Insert 20 masternode before gap block
 2. Prepare 20 masternode block header with round 9000
 3. verify this header while node is on round 899,
    This is to simulate node is syncing from remote during config switch
*/
func TestConfigSwitchOnDifferentMasternodeCount(t *testing.T) {
	b, err := json.Marshal(params.TestXDPoSMockChainConfig)
	assert.Nil(t, err)
	configString := string(b)

	var config params.ChainConfig
	err = json.Unmarshal([]byte(configString), &config)
	assert.Nil(t, err)
	// Enable verify
	config.XDPoS.V2.SkipV2Validation = false
	// Block 901 is the first v2 block with round of 1
	blockchain, _, currentBlock, _, _, _ := PrepareXDCTestBlockChainForV2Engine(t, int(config.XDPoS.Epoch)*2, &config, nil)
	adaptor := blockchain.Engine().(*XDPoS.XDPoS)
	x := adaptor.EngineV2

	// Generate round 900 header, num 1800
	header1800 := blockchain.GetBlockByNumber(1800).Header()

	snap, err := x.GetSnapshot(blockchain, currentBlock.Header())
	assert.Nil(t, err)
	assert.Equal(t, len(snap.NextEpochCandidates), 20)
	header1800.Validators = []byte{}
	for i := 0; i < 20; i++ {
		header1800.Validators = append(header1800.Validators, snap.NextEpochCandidates[i].Bytes()...)
	}

	round, err := x.GetRoundNumber(header1800)
	assert.Nil(t, err)
	assert.Equal(t, round, types.Round(900))

	adaptor.EngineV2.SetNewRoundFaker(blockchain, 899, false)

	err = adaptor.VerifyHeader(blockchain, header1800, true)

	// error ErrValidatorNotWithinMasternodes means verifyQC is passed and move to next verification process
	assert.Equal(t, utils.ErrValidatorNotWithinMasternodes, err)
}

func TestConfigSwitchOnDifferentMindPeriod(t *testing.T) {
	b, err := json.Marshal(params.TestXDPoSMockChainConfig)
	assert.Nil(t, err)
	configString := string(b)

	var config params.ChainConfig
	err = json.Unmarshal([]byte(configString), &config)
	assert.Nil(t, err)
	// Enable verify
	config.XDPoS.V2.SkipV2Validation = false
	// Block 901 is the first v2 block with round of 1
	blockchain, _, _, _, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 915, &config, nil)

	adaptor := blockchain.Engine().(*XDPoS.XDPoS)

	// Genrate 911 QC
	proposedBlockInfo := &types.BlockInfo{
		Hash:   blockchain.GetBlockByNumber(911).Hash(),
		Round:  types.Round(11),
		Number: blockchain.GetBlockByNumber(911).Number(),
	}
	voteForSign := &types.VoteForSign{
		ProposedBlockInfo: proposedBlockInfo,
		GapNumber:         450,
	}

	// Sign from acc 1, 2, 3
	acc1SignedHash := SignHashByPK(acc1Key, types.VoteSigHash(voteForSign).Bytes())
	acc2SignedHash := SignHashByPK(acc2Key, types.VoteSigHash(voteForSign).Bytes())
	acc3SignedHash := SignHashByPK(acc3Key, types.VoteSigHash(voteForSign).Bytes())
	var signaturesFirst []types.Signature
	signaturesFirst = append(signaturesFirst, acc1SignedHash, acc2SignedHash, acc3SignedHash)
	quorumCert := &types.QuorumCert{
		ProposedBlockInfo: proposedBlockInfo,
		Signatures:        signaturesFirst,
		GapNumber:         450,
	}

	extra := types.ExtraFields_v2{
		Round:      types.Round(12),
		QuorumCert: quorumCert,
	}
	extraInBytes, _ := extra.EncodeToBytes()

	// after 910 require 5 signs, but we only give 3 signs
	block911 := blockchain.GetBlockByNumber(911).Header()
	block911.Extra = extraInBytes
	block911.Time = big.NewInt(blockchain.GetBlockByNumber(910).Time().Int64() + 2) //2 is previous config, should get the right config from round
	err = adaptor.VerifyHeader(blockchain, block911, true)

	assert.Equal(t, utils.ErrInvalidTimestamp, err)
}

func TestShouldFailIfNotEnoughQCSignatures(t *testing.T) {
	b, err := json.Marshal(params.TestXDPoSMockChainConfig)
	assert.Nil(t, err)
	configString := string(b)

	var config params.ChainConfig
	err = json.Unmarshal([]byte(configString), &config)
	assert.Nil(t, err)
	// Enable verify
	config.XDPoS.V2.SkipV2Validation = false
	// Block 901 is the first v2 block with round of 1
	blockchain, _, currentBlock, signer, signFn, _ := PrepareXDCTestBlockChainForV2Engine(t, 902, &config, nil)
	adaptor := blockchain.Engine().(*XDPoS.XDPoS)

	parentBlock := blockchain.GetBlockByNumber(901)
	proposedBlockInfo := &types.BlockInfo{
		Hash:   parentBlock.Hash(),
		Round:  types.Round(1),
		Number: parentBlock.Number(),
	}
	voteForSign := &types.VoteForSign{
		ProposedBlockInfo: proposedBlockInfo,
		GapNumber:         450,
	}
	signedHash, err := signFn(accounts.Account{Address: signer}, types.VoteSigHash(voteForSign).Bytes())
	assert.Nil(t, err)
	var signatures []types.Signature
	// Duplicate the signatures
	signatures = append(signatures, signedHash, signedHash, signedHash, signedHash, signedHash, signedHash)
	quorumCert := &types.QuorumCert{
		ProposedBlockInfo: proposedBlockInfo,
		Signatures:        signatures,
		GapNumber:         450,
	}

	extra := types.ExtraFields_v2{
		Round:      types.Round(2),
		QuorumCert: quorumCert,
	}
	extraInBytes, err := extra.EncodeToBytes()
	if err != nil {
		panic(fmt.Errorf("error encode extra into bytes: %v", err))
	}
	headerWithDuplicatedSignatures := currentBlock.Header()
	headerWithDuplicatedSignatures.Extra = extraInBytes
	// Happy path
	err = adaptor.VerifyHeader(blockchain, headerWithDuplicatedSignatures, true)
	assert.Equal(t, utils.ErrInvalidQCSignatures, err)

}

func TestShouldVerifyHeaders(t *testing.T) {
	b, err := json.Marshal(params.TestXDPoSMockChainConfig)
	assert.Nil(t, err)
	configString := string(b)

	var config params.ChainConfig
	err = json.Unmarshal([]byte(configString), &config)
	assert.Nil(t, err)
	// Enable verify
	config.XDPoS.V2.SkipV2Validation = false
	// Block 901 is the first v2 block with round of 1
	blockchain, _, _, _, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 910, &config, nil)
	adaptor := blockchain.Engine().(*XDPoS.XDPoS)

	// Happy path
	var happyPathHeaders []*types.Header
	happyPathHeaders = append(happyPathHeaders, blockchain.GetBlockByNumber(899).Header(), blockchain.GetBlockByNumber(900).Header(), blockchain.GetBlockByNumber(901).Header(), blockchain.GetBlockByNumber(902).Header())
	// Randomly set full verify
	var fullVerifies []bool
	fullVerifies = append(fullVerifies, false, true, true, false)
	_, results := adaptor.VerifyHeaders(blockchain, happyPathHeaders, fullVerifies)
	var verified []bool
	for {
		select {
		case result := <-results:
			if result != nil {
				panic("Error received while verifying headers")
			}
			verified = append(verified, true)
		case <-time.After(time.Duration(5) * time.Second): // It should be very fast to verify headers
			if len(verified) == len(happyPathHeaders) {
				return
			} else {
				panic("Suppose to have verified 3 block headers")
			}
		}
	}
}

func TestShouldVerifyHeadersEvenIfParentsNotYetWrittenIntoDB(t *testing.T) {
	b, err := json.Marshal(params.TestXDPoSMockChainConfig)
	assert.Nil(t, err)
	configString := string(b)

	var config params.ChainConfig
	err = json.Unmarshal([]byte(configString), &config)
	assert.Nil(t, err)
	// Enable verify
	config.XDPoS.V2.SkipV2Validation = false
	// Block 901 is the first v2 block with round of 1
	blockchain, _, block910, signer, signFn, _ := PrepareXDCTestBlockChainForV2Engine(t, 910, &config, nil)
	adaptor := blockchain.Engine().(*XDPoS.XDPoS)

	var headersTobeVerified []*types.Header

	// Create block 911 but don't write into DB
	blockNumber := 911
	roundNumber := int64(blockNumber) - config.XDPoS.V2.SwitchBlock.Int64()
	block911 := CreateBlock(blockchain, &config, block910, blockNumber, roundNumber, signer.Hex(), signer, signFn, nil, nil, "")

	// Create block 912 and not write into DB as well
	blockNumber = 912
	roundNumber = int64(blockNumber) - config.XDPoS.V2.SwitchBlock.Int64()
	block912 := CreateBlock(blockchain, &config, block911, blockNumber, roundNumber, signer.Hex(), signer, signFn, nil, nil, "")

	headersTobeVerified = append(headersTobeVerified, block910.Header(), block911.Header(), block912.Header())
	// Randomly set full verify
	var fullVerifies []bool
	fullVerifies = append(fullVerifies, true, true, true)
	_, results := adaptor.VerifyHeaders(blockchain, headersTobeVerified, fullVerifies)

	var verified []bool
	for {
		select {
		case result := <-results:
			if result != nil {
				panic("Error received while verifying headers")
			}
			verified = append(verified, true)
		case <-time.After(time.Duration(5) * time.Second): // It should be very fast to verify headers
			if len(verified) == len(headersTobeVerified) {
				return
			} else {
				panic("Suppose to have verified 3 block headers")
			}
		}
	}
}
