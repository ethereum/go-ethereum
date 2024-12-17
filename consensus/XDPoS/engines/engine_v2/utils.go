package engine_v2

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/XinFinOrg/XDPoSChain/accounts"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/consensus"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/crypto"
	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/XinFinOrg/XDPoSChain/rlp"
	"golang.org/x/crypto/sha3"
)

func sigHash(header *types.Header) (hash common.Hash) {
	hasher := sha3.NewLegacyKeccak256()

	enc := []interface{}{
		header.ParentHash,
		header.UncleHash,
		header.Coinbase,
		header.Root,
		header.TxHash,
		header.ReceiptHash,
		header.Bloom,
		header.Difficulty,
		header.Number,
		header.GasLimit,
		header.GasUsed,
		header.Time,
		header.Extra,
		header.MixDigest,
		header.Nonce,
		header.Validators,
		header.Penalties,
	}
	if header.BaseFee != nil {
		enc = append(enc, header.BaseFee)
	}
	rlp.Encode(hasher, enc)
	hasher.Sum(hash[:0])
	return hash
}

func ecrecover(header *types.Header, sigcache *utils.SigLRU) (common.Address, error) {
	// If the signature's already cached, return that
	hash := header.Hash()
	if address, known := sigcache.Get(hash); known {
		return address, nil
	}

	// Recover the public key and the Ethereum address
	pubkey, err := crypto.Ecrecover(sigHash(header).Bytes(), header.Validator)
	if err != nil {
		return common.Address{}, err
	}
	var signer common.Address
	copy(signer[:], crypto.Keccak256(pubkey[1:])[12:])

	sigcache.Add(hash, signer)
	return signer, nil

}

// Get masternodes address from checkpoint Header. Only used for v1 last block
func decodeMasternodesFromHeaderExtra(checkpointHeader *types.Header) []common.Address {
	masternodes := make([]common.Address, (len(checkpointHeader.Extra)-utils.ExtraVanity-utils.ExtraSeal)/common.AddressLength)
	for i := 0; i < len(masternodes); i++ {
		copy(masternodes[i][:], checkpointHeader.Extra[utils.ExtraVanity+i*common.AddressLength:])
	}
	return masternodes
}

func UniqueSignatures(signatureSlice []types.Signature) ([]types.Signature, []types.Signature) {
	keys := make(map[string]bool)
	list := []types.Signature{}
	duplicates := []types.Signature{}
	for _, signature := range signatureSlice {
		hexOfSig := common.Bytes2Hex(signature)
		if _, value := keys[hexOfSig]; !value {
			keys[hexOfSig] = true
			list = append(list, signature)
		} else {
			duplicates = append(duplicates, signature)
		}
	}
	return list, duplicates
}

func (x *XDPoS_v2) signSignature(signingHash common.Hash) (types.Signature, error) {
	// Don't hold the signFn for the whole signing operation
	x.signLock.RLock()
	signer, signFn := x.signer, x.signFn
	x.signLock.RUnlock()

	signedHash, err := signFn(accounts.Account{Address: signer}, signingHash.Bytes())
	if err != nil {
		return nil, fmt.Errorf("error %v while signing hash", err)
	}
	return signedHash, nil
}

func (x *XDPoS_v2) verifyMsgSignature(signedHashToBeVerified common.Hash, signature types.Signature, masternodes []common.Address) (bool, common.Address, error) {
	var signerAddress common.Address
	if len(masternodes) == 0 {
		return false, signerAddress, errors.New("empty masternode list detected when verifying message signatures")
	}
	// Recover the public key and the Ethereum address
	pubkey, err := crypto.Ecrecover(signedHashToBeVerified.Bytes(), signature)
	if err != nil {
		return false, signerAddress, fmt.Errorf("error while verifying message: %v", err)
	}

	copy(signerAddress[:], crypto.Keccak256(pubkey[1:])[12:])
	for _, mn := range masternodes {
		if mn == signerAddress {
			return true, signerAddress, nil
		}
	}

	log.Warn("[verifyMsgSignature] signer is not part of masternode list", "signer", signerAddress, "masternodes", masternodes)
	return false, signerAddress, nil
}

func (x *XDPoS_v2) getExtraFields(header *types.Header) (*types.QuorumCert, types.Round, []common.Address, error) {

	var masternodes []common.Address

	// last v1 block
	if header.Number.Cmp(x.config.V2.SwitchBlock) == 0 {
		masternodes = decodeMasternodesFromHeaderExtra(header)
		return nil, types.Round(0), masternodes, nil
	}

	// v2 block
	masternodes = x.GetMasternodesFromEpochSwitchHeader(header)
	var decodedExtraField types.ExtraFields_v2
	err := utils.DecodeBytesExtraFields(header.Extra, &decodedExtraField)
	if err != nil {
		log.Error("[getExtraFields] error on decode extra fields", "err", err, "extra", header.Extra)
		return nil, types.Round(0), masternodes, err
	}
	return decodedExtraField.QuorumCert, decodedExtraField.Round, masternodes, nil
}

func (x *XDPoS_v2) GetRoundNumber(header *types.Header) (types.Round, error) {
	// If not v2 yet, return 0
	if header.Number.Cmp(x.config.V2.SwitchBlock) <= 0 {
		return types.Round(0), nil
	} else {
		var decodedExtraField types.ExtraFields_v2
		err := utils.DecodeBytesExtraFields(header.Extra, &decodedExtraField)
		if err != nil {
			return types.Round(0), err
		}
		return decodedExtraField.Round, nil
	}
}

func (x *XDPoS_v2) GetSignersFromSnapshot(chain consensus.ChainReader, header *types.Header) ([]common.Address, error) {
	snap, err := x.getSnapshot(chain, header.Number.Uint64(), false)
	if err != nil {
		return nil, err
	}
	return snap.NextEpochCandidates, err
}

func (x *XDPoS_v2) CalculateMissingRounds(chain consensus.ChainReader, header *types.Header) (*utils.PublicApiMissedRoundsMetadata, error) {
	var missedRounds []utils.MissedRoundInfo
	switchInfo, err := x.getEpochSwitchInfo(chain, header, header.Hash())
	if err != nil {
		return nil, err
	}
	masternodes := switchInfo.Masternodes

	// Loop through from the epoch switch block to the current "header" block
	nextHeader := header
	for nextHeader.Number.Cmp(switchInfo.EpochSwitchBlockInfo.Number) > 0 {
		parentHeader := chain.GetHeaderByHash(nextHeader.ParentHash)
		parentRound, err := x.GetRoundNumber(parentHeader)
		if err != nil {
			return nil, err
		}
		currRound, err := x.GetRoundNumber(nextHeader)
		if err != nil {
			return nil, err
		}
		// This indicates that an increment in the round number is missing during the block production process.
		if parentRound+1 != currRound {
			// We need to iterate from the parentRound to the currRound to determine which miner did not perform mining.
			for i := parentRound + 1; i < currRound; i++ {
				leaderIndex := uint64(i) % x.config.Epoch % uint64(len(masternodes))
				whosTurn := masternodes[leaderIndex]
				missedRounds = append(
					missedRounds,
					utils.MissedRoundInfo{
						Round:            i,
						Miner:            whosTurn,
						CurrentBlockHash: nextHeader.Hash(),
						CurrentBlockNum:  nextHeader.Number,
						ParentBlockHash:  parentHeader.Hash(),
						ParentBlockNum:   parentHeader.Number,
					},
				)
			}
		}
		// Assign the pointer to the next one
		nextHeader = parentHeader
	}
	missedRoundsMetadata := &utils.PublicApiMissedRoundsMetadata{
		EpochRound:       switchInfo.EpochSwitchBlockInfo.Round,
		EpochBlockNumber: switchInfo.EpochSwitchBlockInfo.Number,
		MissedRounds:     missedRounds,
	}

	return missedRoundsMetadata, nil
}

func (x *XDPoS_v2) getBlockByEpochNumberInCache(chain consensus.ChainReader, estRound types.Round) *types.BlockInfo {
	epochSwitchInCache := make([]*types.BlockInfo, 0)
	for r := estRound; r < estRound+types.Round(x.config.Epoch); r++ {
		blockInfo, ok := x.round2epochBlockInfo.Get(r)
		if ok && blockInfo != nil {
			epochSwitchInCache = append(epochSwitchInCache, blockInfo)
		}
	}
	if len(epochSwitchInCache) == 1 {
		return epochSwitchInCache[0]
	} else if len(epochSwitchInCache) == 0 {
		return nil
	}
	// when multiple cache hits, need to find the one in main chain
	for _, blockInfo := range epochSwitchInCache {
		header := chain.GetHeaderByNumber(blockInfo.Number.Uint64())
		if header == nil {
			continue
		}
		if header.Hash() == blockInfo.Hash {
			return blockInfo
		}
	}
	return nil
}

func (x *XDPoS_v2) binarySearchBlockByEpochNumber(chain consensus.ChainReader, targetEpochNum uint64, start, end uint64) (*types.BlockInfo, error) {
	// `end` must be larger than the target and `start` could be the target
	for start < end {
		header := chain.GetHeaderByNumber((start + end) / 2)
		if header == nil {
			return nil, errors.New("header nil in binary search")
		}
		isEpochSwitch, epochNum, err := x.IsEpochSwitch(header)
		if err != nil {
			return nil, err
		}
		if epochNum == targetEpochNum {
			_, round, _, err := x.getExtraFields(header)
			if err != nil {
				return nil, err
			}
			if isEpochSwitch {
				return &types.BlockInfo{
					Hash:   header.Hash(),
					Round:  round,
					Number: header.Number,
				}, nil
			} else {
				end = header.Number.Uint64()
				// trick to shorten the search
				estStart := end - uint64(round)%x.config.Epoch
				if start < estStart {
					start = estStart
				}
			}
		} else if epochNum > targetEpochNum {
			end = header.Number.Uint64()
		} else if epochNum < targetEpochNum {
			// if start keeps the same, means no result and the search is over
			nextStart := header.Number.Uint64()
			if nextStart == start {
				break
			}
			start = nextStart
		}
	}
	return nil, errors.New("no epoch switch header in binary search (all rounds in this epoch are missed, which is very rare)")
}

func (x *XDPoS_v2) GetBlockByEpochNumber(chain consensus.ChainReader, targetEpochNum uint64) (*types.BlockInfo, error) {
	currentHeader := chain.CurrentHeader()
	epochSwitchInfo, err := x.getEpochSwitchInfo(chain, currentHeader, currentHeader.Hash())
	if err != nil {
		return nil, err
	}
	epochNum := x.config.V2.SwitchBlock.Uint64()/x.config.Epoch + uint64(epochSwitchInfo.EpochSwitchBlockInfo.Round)/x.config.Epoch
	// if current epoch is this epoch, we early return the result
	if targetEpochNum == epochNum {
		return epochSwitchInfo.EpochSwitchBlockInfo, nil
	}
	if targetEpochNum > epochNum {
		return nil, errors.New("input epoch number > current epoch number")
	}
	if targetEpochNum < x.config.V2.SwitchBlock.Uint64()/x.config.Epoch {
		return nil, errors.New("input epoch number < v2 begin epoch number")
	}
	// the block's round should be in [estRound,estRound+Epoch-1]
	estRound := types.Round((targetEpochNum - x.config.V2.SwitchBlock.Uint64()/x.config.Epoch) * x.config.Epoch)
	// check the round2epochBlockInfo cache
	blockInfo := x.getBlockByEpochNumberInCache(chain, estRound)
	if blockInfo != nil {
		return blockInfo, nil
	}
	// if cache miss, we do search
	epoch := big.NewInt(int64(x.config.Epoch))
	estblockNumDiff := new(big.Int).Mul(epoch, big.NewInt(int64(epochNum-targetEpochNum)))
	estBlockNum := new(big.Int).Sub(epochSwitchInfo.EpochSwitchBlockInfo.Number, estblockNumDiff)
	if estBlockNum.Cmp(x.config.V2.SwitchBlock) == -1 {
		estBlockNum.Set(x.config.V2.SwitchBlock)
	}
	// if the targrt is close, we search brute-forcily
	closeEpochNum := uint64(2)
	if closeEpochNum >= epochNum-targetEpochNum {
		estBlockHeader := chain.GetHeaderByNumber(estBlockNum.Uint64())
		epochSwitchInfos, err := x.GetEpochSwitchInfoBetween(chain, estBlockHeader, currentHeader)
		if err != nil {
			return nil, err
		}
		for _, info := range epochSwitchInfos {
			epochNum := x.config.V2.SwitchBlock.Uint64()/x.config.Epoch + uint64(info.EpochSwitchBlockInfo.Round)/x.config.Epoch
			if epochNum == targetEpochNum {
				return info.EpochSwitchBlockInfo, nil
			}
		}
	}
	// else, we use binary search
	return x.binarySearchBlockByEpochNumber(chain, targetEpochNum, estBlockNum.Uint64(), epochSwitchInfo.EpochSwitchBlockInfo.Number.Uint64())
}
