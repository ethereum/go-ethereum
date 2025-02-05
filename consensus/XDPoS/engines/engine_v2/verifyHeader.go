package engine_v2

import (
	"bytes"
	"math/big"
	"time"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/common/hexutil"
	"github.com/XinFinOrg/XDPoSChain/consensus"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/consensus/misc/eip1559"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/log"
)

// Verify individual header
func (x *XDPoS_v2) verifyHeader(chain consensus.ChainReader, header *types.Header, parents []*types.Header, fullVerify bool) error {
	// If we're running a engine faking, accept any block as valid
	if x.config.V2.SkipV2Validation {
		return nil
	}

	if !x.isInitilised {
		if err := x.initial(chain, header); err != nil {
			return err
		}
	}

	_, check := x.verifiedHeaders.Get(header.Hash())
	if check {
		return nil
	}

	if header.Number == nil {
		return utils.ErrUnknownBlock
	}

	if len(header.Validator) == 0 {
		// This should never happen, if it does, then it means the peer is sending us invalid data.
		return consensus.ErrNoValidatorSignatureV2
	}

	if fullVerify {
		// Don't waste time checking blocks from the future
		if header.Time.Int64() > time.Now().Unix() {
			return consensus.ErrFutureBlock
		}
	}

	// Ensure that the block's timestamp isn't too close to it's parent
	var parent *types.Header
	number := header.Number.Uint64()

	if len(parents) > 0 {
		parent = parents[len(parents)-1]
	} else {
		parent = chain.GetHeader(header.ParentHash, number-1)
	}
	if parent == nil || parent.Number.Uint64() != number-1 || parent.Hash() != header.ParentHash {
		return consensus.ErrUnknownAncestor
	}

	// Verify this is truely a v2 block first
	quorumCert, round, _, err := x.getExtraFields(header)
	if err != nil {
		log.Warn("[verifyHeader] decode extra field error", "err", err)
		return utils.ErrInvalidV2Extra
	}

	minePeriod := uint64(x.config.V2.Config(uint64(round)).MinePeriod)
	if parent.Number.Uint64() > x.config.V2.SwitchBlock.Uint64() && parent.Time.Uint64()+minePeriod > header.Time.Uint64() {
		log.Warn("[verifyHeader] Fail to verify header due to invalid timestamp", "ParentTime", parent.Time.Uint64(), "MinePeriod", minePeriod, "HeaderTime", header.Time.Uint64(), "Hash", header.Hash().Hex())
		return utils.ErrInvalidTimestamp
	}

	if round <= quorumCert.ProposedBlockInfo.Round {
		return utils.ErrRoundInvalid
	}

	err = x.verifyQC(chain, quorumCert, parent)
	if err != nil {
		log.Warn("[verifyHeader] fail to verify QC", "QCNumber", quorumCert.ProposedBlockInfo.Number, "QCsigLength", len(quorumCert.Signatures))
		return err
	}
	// Nonces must be 0x00..0 or 0xff..f, zeroes enforced on checkpoints
	if !bytes.Equal(header.Nonce[:], utils.NonceAuthVote) && !bytes.Equal(header.Nonce[:], utils.NonceDropVote) {
		return utils.ErrInvalidVote
	}
	// Ensure that the mix digest is zero as we don't have fork protection currently
	if header.MixDigest != (common.Hash{}) {
		return utils.ErrInvalidMixDigest
	}
	// Ensure that the block doesn't contain any uncles which are meaningless in XDPoS_v1
	if header.UncleHash != utils.UncleHash {
		return utils.ErrInvalidUncleHash
	}
	// Verify the header's EIP-1559 attributes.
	if err := eip1559.VerifyEip1559Header(chain.Config(), header); err != nil {
		return err
	}
	if header.Difficulty.Cmp(big.NewInt(1)) != 0 {
		return utils.ErrInvalidDifficulty
	}

	var masterNodes []common.Address
	isEpochSwitch, _, err := x.IsEpochSwitch(header) // Verify v2 block that is on the epoch switch
	if err != nil {
		log.Error("[verifyHeader] error when checking if header is epoch switch header", "Hash", header.Hash(), "Number", header.Number, "Error", err)
		return err
	}
	if isEpochSwitch {
		if !bytes.Equal(header.Nonce[:], utils.NonceDropVote) {
			return utils.ErrInvalidCheckpointVote
		}
		if len(header.Validators) == 0 {
			return utils.ErrEmptyEpochSwitchValidators
		}
		if len(header.Validators)%common.AddressLength != 0 {
			return utils.ErrInvalidCheckpointSigners
		}

		localMasterNodes, localPenalties, err := x.calcMasternodes(chain, header.Number, header.ParentHash, round)
		masterNodes = localMasterNodes
		if err != nil {
			log.Error("[verifyHeader] Fail to calculate master nodes list with penalty", "Number", header.Number, "Hash", header.Hash())
			return err
		}

		validatorsAddress := common.ExtractAddressFromBytes(header.Validators)
		if !utils.CompareSignersLists(localMasterNodes, validatorsAddress) {
			for i, addr := range localMasterNodes {
				log.Warn("[verifyHeader] localMasterNodes", "i", i, "addr", addr.Hex())
			}
			for i, addr := range validatorsAddress {
				log.Warn("[verifyHeader] validatorsAddress", "i", i, "addr", addr.Hex())
			}
			return utils.ErrValidatorsNotLegit
		}

		penaltiesAddress := common.ExtractAddressFromBytes(header.Penalties)
		if !utils.CompareSignersLists(localPenalties, penaltiesAddress) {
			for i, addr := range localPenalties {
				log.Warn("[verifyHeader] localPenalties", "i", i, "addr", addr.Hex())
			}
			for i, addr := range penaltiesAddress {
				log.Warn("[verifyHeader] penaltiesAddress", "i", i, "addr", addr.Hex())
			}
			return utils.ErrPenaltiesNotLegit
		}

	} else {
		if len(header.Validators) != 0 {
			log.Warn("[verifyHeader] Validators shall not have values in non-epochSwitch block", "Hash", header.Hash(), "Number", header.Number, "header.Validators", header.Validators)
			return utils.ErrInvalidFieldInNonEpochSwitch
		}
		if len(header.Penalties) != 0 {
			log.Warn("[verifyHeader] Penalties shall not have values in non-epochSwitch block", "Hash", header.Hash(), "Number", header.Number, "header.Penalties", header.Penalties)
			return utils.ErrInvalidFieldInNonEpochSwitch
		}
		masterNodes = x.GetMasternodes(chain, header)
	}

	verified, validatorAddress, err := x.verifyMsgSignature(sigHash(header), header.Validator, masterNodes)
	if err != nil {
		for index, mn := range masterNodes {
			log.Error("[verifyHeader] masternode list during validator verification", "Masternode Address", mn.Hex(), "index", index)
		}
		log.Error("[verifyHeader] Error while verifying header validator signature", "BlockNumber", header.Number, "Hash", header.Hash().Hex(), "validator in hex", hexutil.Encode(header.Validator))
		return err
	}
	if !verified {
		log.Warn("[verifyHeader] Fail to verify the block validator as the validator address not within the masternode list", "BlockNumber", header.Number, "Hash", header.Hash().Hex(), "validatorAddress", validatorAddress.Hex())
		return utils.ErrValidatorNotWithinMasternodes
	}
	if validatorAddress != header.Coinbase {
		log.Warn("[verifyHeader] Header validator and coinbase address not match", "BlockNumber", header.Number, "Hash", header.Hash().Hex(), "validatorAddress", validatorAddress.Hex(), "coinbase", header.Coinbase.Hex())
		return utils.ErrCoinbaseAndValidatorMismatch
	}
	// Check the proposer is the leader
	curIndex := utils.Position(masterNodes, validatorAddress)
	leaderIndex := uint64(round) % x.config.Epoch % uint64(len(masterNodes))
	if masterNodes[leaderIndex] != validatorAddress {
		log.Warn("[verifyHeader] Invalid blocker proposer, not its turn", "curIndex", curIndex, "leaderIndex", leaderIndex, "Hash", header.Hash().Hex(), "masterNodes[leaderIndex]", masterNodes[leaderIndex], "validatorAddress", validatorAddress)
		return utils.ErrNotItsTurn
	}

	x.verifiedHeaders.Add(header.Hash(), struct{}{})
	return nil
}
