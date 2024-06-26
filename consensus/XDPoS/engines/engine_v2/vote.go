package engine_v2

import (
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/consensus"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/log"
)

// Once Hot stuff voting rule has verified, this node can then send vote
func (x *XDPoS_v2) sendVote(chainReader consensus.ChainReader, blockInfo *types.BlockInfo) error {
	// First step: Update the highest Voted round
	// Second step: Generate the signature by using node's private key(The signature is the blockInfo signature)
	// Third step: Construct the vote struct with the above signature & blockinfo struct
	// Forth step: Send the vote to broadcast channel

	epochSwitchInfo, err := x.getEpochSwitchInfo(chainReader, nil, blockInfo.Hash)
	if err != nil {
		log.Error("getEpochSwitchInfo when sending out Vote", "BlockInfoHash", blockInfo.Hash, "Error", err)
		return err
	}
	epochSwitchNumber := epochSwitchInfo.EpochSwitchBlockInfo.Number.Uint64()
	gapNumber := epochSwitchNumber - epochSwitchNumber%x.config.Epoch - x.config.Gap
	// prevent overflow
	if epochSwitchNumber-epochSwitchNumber%x.config.Epoch < x.config.Gap {
		gapNumber = 0
	}
	signedHash, err := x.signSignature(types.VoteSigHash(&types.VoteForSign{
		ProposedBlockInfo: blockInfo,
		GapNumber:         gapNumber,
	}))
	if err != nil {
		log.Error("signSignature when sending out Vote", "BlockInfoHash", blockInfo.Hash, "Error", err)
		return err
	}

	x.highestVotedRound = x.currentRound
	voteMsg := &types.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         signedHash,
		GapNumber:         gapNumber,
	}

	err = x.voteHandler(chainReader, voteMsg)
	if err != nil {
		log.Error("sendVote error", "BlockInfoHash", blockInfo.Hash, "Error", err)
		return err
	}
	x.broadcastToBftChannel(voteMsg)
	return nil
}

func (x *XDPoS_v2) voteHandler(chain consensus.ChainReader, voteMsg *types.Vote) error {
	// checkRoundNumber
	if (voteMsg.ProposedBlockInfo.Round != x.currentRound) && (voteMsg.ProposedBlockInfo.Round != x.currentRound+1) {
		return &utils.ErrIncomingMessageRoundTooFarFromCurrentRound{
			Type:          "vote",
			IncomingRound: voteMsg.ProposedBlockInfo.Round,
			CurrentRound:  x.currentRound,
		}
	}

	if x.votePoolCollectionTime.IsZero() {
		log.Info("[voteHandler] set vote pool time", "round", x.currentRound)
		x.votePoolCollectionTime = time.Now()
	}

	// Collect vote
	numberOfVotesInPool, pooledVotes := x.votePool.Add(voteMsg)
	log.Debug("[voteHandler] collect votes", "number", numberOfVotesInPool)
	go x.ForensicsProcessor.DetectEquivocationInVotePool(voteMsg, x.votePool)
	go x.ForensicsProcessor.ProcessVoteEquivocation(chain, x, voteMsg)

	epochInfo, err := x.getEpochSwitchInfo(chain, nil, voteMsg.ProposedBlockInfo.Hash)
	if err != nil {
		return &utils.ErrIncomingMessageBlockNotFound{
			Type:                "vote",
			IncomingBlockHash:   voteMsg.ProposedBlockInfo.Hash,
			IncomingBlockNumber: voteMsg.ProposedBlockInfo.Number,
			Err:                 err,
		}
	}

	certThreshold := x.config.V2.Config(uint64(voteMsg.ProposedBlockInfo.Round)).CertThreshold
	thresholdReached := float64(numberOfVotesInPool) >= float64(epochInfo.MasternodesLen)*certThreshold
	if thresholdReached {
		log.Info(fmt.Sprintf("[voteHandler] Vote pool threashold reached: %v, number of items in the pool: %v", thresholdReached, numberOfVotesInPool))

		// Check if the block already exist, otherwise we try luck with the next vote
		proposedBlockHeader := chain.GetHeaderByHash(voteMsg.ProposedBlockInfo.Hash)
		if proposedBlockHeader == nil {
			log.Info("[voteHandler] The proposed block from vote message does not exist yet, wait for the next vote to try again", "blockNum", voteMsg.ProposedBlockInfo.Number, "Hash", voteMsg.ProposedBlockInfo.Hash, "Round", voteMsg.ProposedBlockInfo.Round)
			return nil
		}

		err := x.VerifyBlockInfo(chain, voteMsg.ProposedBlockInfo, nil)
		if err != nil {
			return err
		}

		x.verifyVotes(chain, pooledVotes, proposedBlockHeader)

		err = x.onVotePoolThresholdReached(chain, pooledVotes, voteMsg, proposedBlockHeader)
		if err != nil {
			return err
		}
		elapsed := time.Since(x.votePoolCollectionTime)
		log.Info("[voteHandler] time cost from receive first vote under QC create", "elapsed", elapsed)
		x.votePoolCollectionTime = time.Time{}
	}

	return nil
}

func (x *XDPoS_v2) verifyVotes(chain consensus.ChainReader, votes map[common.Hash]utils.PoolObj, header *types.Header) {
	masternodes := x.GetMasternodes(chain, header)
	start := time.Now()
	emptySigner := common.Address{}
	// Filter out non-Master nodes signatures
	var wg sync.WaitGroup
	wg.Add(len(votes))
	for h, vote := range votes {
		go func(hash common.Hash, v *types.Vote) {
			defer wg.Done()
			signerAddress := v.GetSigner()
			if signerAddress != emptySigner {
				// verify that signer belongs to the final masternodes, we have not do so in previous steps
				if len(masternodes) == 0 {
					log.Error("[verifyVotes] empty masternode list detected when verifying message signatures")
				}
				for _, mn := range masternodes {
					if mn == signerAddress {
						return
					}
				}
				// if signer does not belong to final masternodes, we remove the signer
				v.SetSigner(emptySigner)
				log.Debug("[verifyVotes] find a vote does not belong to final masternodes", "signer", signerAddress)
				return
			}
			signedVote := types.VoteSigHash(&types.VoteForSign{
				ProposedBlockInfo: v.ProposedBlockInfo,
				GapNumber:         v.GapNumber,
			})
			verified, masterNode, err := x.verifyMsgSignature(signedVote, v.Signature, masternodes)
			if err != nil {
				log.Warn("[verifyVotes] error while verifying vote signature", "error", err.Error())
				return
			}

			if !verified {
				log.Warn("[verifyVotes] non-verified vote signature", "verified", verified)
				return
			}
			v.SetSigner(masterNode)
		}(h, vote.(*types.Vote))
	}
	wg.Wait()
	elapsed := time.Since(start)
	log.Debug("[verifyVotes] verify message signatures of vote pool took", "elapsed", elapsed)
}

/*
Function that will be called by votePool when it reached threshold.
In the engine v2, we will need to generate and process QC
*/
func (x *XDPoS_v2) onVotePoolThresholdReached(chain consensus.ChainReader, pooledVotes map[common.Hash]utils.PoolObj, currentVoteMsg utils.PoolObj, proposedBlockHeader *types.Header) error {
	// The signature list may contain empty entey. we only care the ones with values
	var validSignatures []types.Signature
	emptySigner := common.Address{}
	for _, vote := range pooledVotes {
		if vote.GetSigner() != emptySigner {
			validSignatures = append(validSignatures, vote.(*types.Vote).Signature)
		}
	}

	epochInfo, err := x.getEpochSwitchInfo(chain, nil, currentVoteMsg.(*types.Vote).ProposedBlockInfo.Hash)
	if err != nil {
		log.Error("[voteHandler] Error when getting epoch switch Info", "error", err)
		return errors.New("fail on voteHandler due to failure in getting epoch switch info")
	}

	// Skip and wait for the next vote to process again if valid votes is less than what we required
	certThreshold := x.config.V2.Config(uint64(currentVoteMsg.(*types.Vote).ProposedBlockInfo.Round)).CertThreshold
	if float64(len(validSignatures)) < float64(epochInfo.MasternodesLen)*certThreshold {
		log.Warn("[onVotePoolThresholdReached] Not enough valid signatures to generate QC", "VotesSignaturesAfterFilter", validSignatures, "NumberOfValidVotes", len(validSignatures), "NumberOfVotes", len(pooledVotes))
		return nil
	}
	// Genrate QC
	quorumCert := &types.QuorumCert{
		ProposedBlockInfo: currentVoteMsg.(*types.Vote).ProposedBlockInfo,
		Signatures:        validSignatures,
		GapNumber:         currentVoteMsg.(*types.Vote).GapNumber,
	}
	err = x.processQC(chain, quorumCert)
	if err != nil {
		log.Error("Error while processing QC in the Vote handler after reaching pool threshold, ", err)
		return err
	}
	log.Info("Successfully processed the vote and produced QC!", "QcRound", quorumCert.ProposedBlockInfo.Round, "QcNumOfSig", len(quorumCert.Signatures), "QcHash", quorumCert.ProposedBlockInfo.Hash, "QcNumber", quorumCert.ProposedBlockInfo.Number.Uint64())
	return nil
}

// Hot stuff rule to decide whether this node is eligible to vote for the received block
func (x *XDPoS_v2) verifyVotingRule(blockChainReader consensus.ChainReader, blockInfo *types.BlockInfo, quorumCert *types.QuorumCert) (bool, error) {
	// Make sure this node has not voted for this round.
	if x.currentRound <= x.highestVotedRound {
		log.Info("Failed to pass the voting rule verification, currentRound is not large then highestVoteRound", "x.currentRound", x.currentRound, "x.highestVotedRound", x.highestVotedRound)
		return false, nil
	}
	/*
		HotStuff Voting rule:
		header's round == local current round, AND (one of the following two:)
		header's block extends lockQuorumCert's ProposedBlockInfo (we need a isExtending(block_a, block_b) function), OR
		header's QC's ProposedBlockInfo.Round > lockQuorumCert's ProposedBlockInfo.Round
	*/
	if blockInfo.Round != x.currentRound {
		log.Info("Failed to pass the voting rule verification, blockRound is not equal currentRound", "x.currentRound", x.currentRound, "blockInfo.Round", blockInfo.Round)
		return false, nil
	}
	// XDPoS v1.0 switch to v2.0, the proposed block can always pass voting rule
	if x.lockQuorumCert == nil {
		return true, nil
	}

	if quorumCert.ProposedBlockInfo.Round > x.lockQuorumCert.ProposedBlockInfo.Round {
		return true, nil
	}

	isExtended, err := x.isExtendingFromAncestor(blockChainReader, blockInfo, x.lockQuorumCert.ProposedBlockInfo)
	if err != nil {
		log.Error("Failed to pass the voting rule verification, error on isExtendingFromAncestor", "err", err, "blockInfo", blockInfo, "x.lockQuorumCert.ProposedBlockInfo", x.lockQuorumCert.ProposedBlockInfo)
		return false, err
	}

	if !isExtended {
		log.Warn("Failed to pass the voting rule verification, block is not extended from ancestor", "blockInfo", blockInfo, "x.lockQuorumCert.ProposedBlockInfo", x.lockQuorumCert.ProposedBlockInfo)
		return false, nil
	}

	return true, nil
}

func (x *XDPoS_v2) isExtendingFromAncestor(blockChainReader consensus.ChainReader, currentBlock *types.BlockInfo, ancestorBlock *types.BlockInfo) (bool, error) {
	blockNumDiff := int(big.NewInt(0).Sub(currentBlock.Number, ancestorBlock.Number).Int64())

	nextBlockHash := currentBlock.Hash
	for i := 0; i < blockNumDiff; i++ {
		parentBlock := blockChainReader.GetHeaderByHash(nextBlockHash)
		if parentBlock == nil {
			return false, fmt.Errorf("could not find its parent block when checking whether currentBlock %v with hash %v is extending from the ancestorBlock %v", currentBlock.Number, currentBlock.Hash, ancestorBlock.Number)
		} else {
			nextBlockHash = parentBlock.ParentHash
		}
		log.Debug("[isExtendingFromAncestor] Found parent block", "CurrentBlockHash", currentBlock.Hash, "ParentHash", nextBlockHash)
	}

	if nextBlockHash == ancestorBlock.Hash {
		return true, nil
	}
	return false, nil
}

func (x *XDPoS_v2) hygieneVotePool() {
	x.lock.RLock()
	round := x.currentRound
	x.lock.RUnlock()
	votePoolKeys := x.votePool.PoolObjKeysList()

	// Extract round number
	for _, k := range votePoolKeys {
		keyedRound, err := strconv.ParseInt(strings.Split(k, ":")[0], 10, 64)
		if err != nil {
			log.Error("[hygieneVotePool] Error while trying to get keyedRound inside pool", "Error", err)
			continue
		}
		// Clean up any votes round that is 10 rounds older
		if keyedRound < int64(round)-utils.PoolHygieneRound {
			log.Debug("[hygieneVotePool] Cleaned vote poll at round", "Round", keyedRound, "currentRound", round, "Key", k)
			x.votePool.ClearByPoolKey(k)
		}
	}
}

func (x *XDPoS_v2) ReceivedVotes() map[string]map[common.Hash]utils.PoolObj {
	return x.votePool.Get()
}
