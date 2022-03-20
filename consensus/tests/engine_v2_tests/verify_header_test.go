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
	// Skip the mining time validation by set mine time to 0
	config.XDPoS.V2.MinePeriod = 0
	// Block 901 is the first v2 block with round of 1
	blockchain, _, _, signer, signFn, _ := PrepareXDCTestBlockChainForV2Engine(t, 910, &config, 0)
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
	assert.Equal(t, consensus.ErrNoValidatorSignature, err)

	blockFromFuture := blockchain.GetBlockByNumber(902).Header()
	blockFromFuture.Time = big.NewInt(time.Now().Unix() + 10000)
	err = adaptor.VerifyHeader(blockchain, blockFromFuture, true)
	assert.Equal(t, consensus.ErrFutureBlock, err)

	invalidQcBlock := blockchain.GetBlockByNumber(902).Header()
	invalidQcBlock.Extra = []byte{}
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

	merkleRoot := "35999dded35e8db12de7e6c1471eb9670c162eec616ecebbaf4fddd4676fb123"
	parentNotExistBlock := blockchain.GetBlockByNumber(901).Header()
	parentNotExistBlock.ParentHash = common.HexToHash(merkleRoot)
	err = adaptor.VerifyHeader(blockchain, parentNotExistBlock, true)
	assert.Equal(t, consensus.ErrUnknownAncestor, err)

	tooFastMinedBlock := blockchain.GetBlockByNumber(902).Header()
	tooFastMinedBlock.Time = big.NewInt(time.Now().Unix() - 2)
	err = adaptor.VerifyHeader(blockchain, tooFastMinedBlock, true)
	assert.Equal(t, utils.ErrInvalidTimestamp, err)

	invalidDifficultyBlock := blockchain.GetBlockByNumber(902).Header()
	invalidDifficultyBlock.Difficulty = big.NewInt(2)
	err = adaptor.VerifyHeader(blockchain, invalidDifficultyBlock, true)
	assert.Equal(t, utils.ErrInvalidDifficulty, err)

	// Creat an invalid QC round
	proposedBlockInfo := &utils.BlockInfo{
		Hash:   blockchain.GetBlockByNumber(902).Hash(),
		Round:  utils.Round(2),
		Number: blockchain.GetBlockByNumber(902).Number(),
	}
	// Genrate QC
	signedHash, err := signFn(accounts.Account{Address: signer}, utils.VoteSigHash(proposedBlockInfo).Bytes())
	if err != nil {
		panic(fmt.Errorf("Error generate QC by creating signedHash: %v", err))
	}
	// Sign from acc 1, 2, 3
	acc1SignedHash := SignHashByPK(acc1Key, utils.VoteSigHash(proposedBlockInfo).Bytes())
	acc2SignedHash := SignHashByPK(acc2Key, utils.VoteSigHash(proposedBlockInfo).Bytes())
	acc3SignedHash := SignHashByPK(acc3Key, utils.VoteSigHash(proposedBlockInfo).Bytes())
	var signatures []utils.Signature
	signatures = append(signatures, signedHash, acc1SignedHash, acc2SignedHash, acc3SignedHash)
	quorumCert := &utils.QuorumCert{
		ProposedBlockInfo: proposedBlockInfo,
		Signatures:        signatures,
	}

	extra := utils.ExtraFields_v2{
		Round:      utils.Round(2),
		QuorumCert: quorumCert,
	}
	extraInBytes, err := extra.EncodeToBytes()
	if err != nil {
		panic(fmt.Errorf("Error encode extra into bytes: %v", err))
	}

	invalidRoundBlock := blockchain.GetBlockByNumber(902).Header()
	invalidRoundBlock.Extra = extraInBytes
	err = adaptor.VerifyHeader(blockchain, invalidRoundBlock, true)
	assert.Equal(t, utils.ErrRoundInvalid, err)

	invalidPenaltiesExistBlock := blockchain.GetBlockByNumber(902).Header()
	invalidPenaltiesExistBlock.Penalties = common.Hex2BytesFixed("123131231", 20)
	err = adaptor.VerifyHeader(blockchain, invalidPenaltiesExistBlock, true)
	assert.Equal(t, utils.ErrPenaltyListDoesNotMatch, err)

	// Not valid validator
	coinbaseValidatorMismatchBlock := blockchain.GetBlockByNumber(902).Header()
	notQualifiedSigner, notQualifiedSignFn, err := getSignerAndSignFn(voterKey)
	assert.Nil(t, err)
	sealHeader(blockchain, coinbaseValidatorMismatchBlock, notQualifiedSigner, notQualifiedSignFn)
	err = adaptor.VerifyHeader(blockchain, coinbaseValidatorMismatchBlock, true)
	assert.Equal(t, utils.ErrCoinbaseAndValidatorMismatch, err)

	// Make the validators not legit by adding something to the penalty
	validatorsNotLegit := blockchain.GetBlockByNumber(901).Header()
	penalties := []common.Address{acc1Addr}
	for _, v := range penalties {
		validatorsNotLegit.Penalties = append(validatorsNotLegit.Penalties, v[:]...)
	}
	err = adaptor.VerifyHeader(blockchain, validatorsNotLegit, true)
	assert.Equal(t, utils.ErrValidatorsNotLegit, err)
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
	// Skip the mining time validation by set mine time to 0
	config.XDPoS.V2.MinePeriod = 0
	// Block 901 is the first v2 block with round of 1
	blockchain, _, currentBlock, signer, signFn, _ := PrepareXDCTestBlockChainForV2Engine(t, 902, &config, 0)
	adaptor := blockchain.Engine().(*XDPoS.XDPoS)

	parentBlock := blockchain.GetBlockByNumber(901)
	proposedBlockInfo := &utils.BlockInfo{
		Hash:   parentBlock.Hash(),
		Round:  utils.Round(1),
		Number: parentBlock.Number(),
	}
	signedHash, err := signFn(accounts.Account{Address: signer}, utils.VoteSigHash(proposedBlockInfo).Bytes())
	assert.Nil(t, err)
	var signatures []utils.Signature
	// Duplicate the signatures
	signatures = append(signatures, signedHash, signedHash, signedHash, signedHash, signedHash, signedHash)
	quorumCert := &utils.QuorumCert{
		ProposedBlockInfo: proposedBlockInfo,
		Signatures:        signatures,
	}

	extra := utils.ExtraFields_v2{
		Round:      utils.Round(2),
		QuorumCert: quorumCert,
	}
	extraInBytes, err := extra.EncodeToBytes()
	if err != nil {
		panic(fmt.Errorf("Error encode extra into bytes: %v", err))
	}
	headerWithDuplicatedSignatures := currentBlock.Header()
	headerWithDuplicatedSignatures.Extra = extraInBytes
	// Happy path
	err = adaptor.VerifyHeader(blockchain, headerWithDuplicatedSignatures, true)
	assert.Equal(t, utils.ErrInvalidQC, err)

}
