package engine_v2

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/consensus"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/log"
)

func (x *XDPoS_v2) VerifyTimeoutMessage(chain consensus.ChainReader, timeoutMsg *types.Timeout) (bool, error) {
	if timeoutMsg.Round < x.currentRound {
		log.Debug("[VerifyTimeoutMessage] Disqualified timeout message as the proposed round does not match currentRound", "timeoutHash", timeoutMsg.Hash(), "timeoutRound", timeoutMsg.Round, "currentRound", x.currentRound)
		return false, nil
	}

	epochInfo, err := x.getTCEpochInfo(chain, timeoutMsg.Round)
	if err != nil {
		log.Error("[VerifyTimeoutMessage] Fail to get epochInfo for timeout message", "tcGapNumber", timeoutMsg.GapNumber, "tcRound", timeoutMsg.Round, "error", err)
		return false, err
	}

	verified, signer, err := x.verifyMsgSignature(types.TimeoutSigHash(&types.TimeoutForSign{
		Round:     timeoutMsg.Round,
		GapNumber: timeoutMsg.GapNumber,
	}), timeoutMsg.Signature, epochInfo.Masternodes)

	if err != nil {
		log.Warn("[VerifyTimeoutMessage] cannot verify timeout signature", "err", err)
		return false, err
	}

	timeoutMsg.SetSigner(signer)
	return verified, nil
}

/*
Entry point for handling timeout message to process below:
*/
func (x *XDPoS_v2) TimeoutHandler(blockChainReader consensus.ChainReader, timeout *types.Timeout) error {
	x.lock.Lock()
	defer x.lock.Unlock()
	return x.timeoutHandler(blockChainReader, timeout)
}

func (x *XDPoS_v2) timeoutHandler(blockChainReader consensus.ChainReader, timeout *types.Timeout) error {
	// checkRoundNumber
	if timeout.Round != x.currentRound {
		return &utils.ErrIncomingMessageRoundNotEqualCurrentRound{
			Type:          "timeout",
			IncomingRound: timeout.Round,
			CurrentRound:  x.currentRound,
		}
	}
	// Collect timeout, generate TC
	numberOfTimeoutsInPool, pooledTimeouts := x.timeoutPool.Add(timeout)
	log.Debug("[timeoutHandler] collect timeout", "number", numberOfTimeoutsInPool)

	epochInfo, err := x.getEpochSwitchInfo(blockChainReader, blockChainReader.CurrentHeader(), blockChainReader.CurrentHeader().Hash())
	if err != nil {
		log.Error("[timeoutHandler] Error when getting epoch switch Info", "error", err)
		return fmt.Errorf("fail on timeoutHandler due to failure in getting epoch switch info, %s", err)
	}

	// Threshold reached
	certThreshold := x.config.V2.Config(uint64(timeout.Round)).CertThreshold
	isThresholdReached := float64(numberOfTimeoutsInPool) >= float64(epochInfo.MasternodesLen)*certThreshold
	if isThresholdReached {
		log.Info(fmt.Sprintf("Timeout pool threashold reached: %v, number of items in the pool: %v", isThresholdReached, numberOfTimeoutsInPool))
		err := x.onTimeoutPoolThresholdReached(blockChainReader, pooledTimeouts, timeout, timeout.GapNumber)
		if err != nil {
			return err
		}
	}
	return nil
}

/*
Function that will be called by timeoutPool when it reached threshold.
In the engine v2, we will need to:
 1. Genrate TC
 2. processTC()
 3. generateSyncInfo()
*/
func (x *XDPoS_v2) onTimeoutPoolThresholdReached(blockChainReader consensus.ChainReader, pooledTimeouts map[common.Hash]utils.PoolObj, currentTimeoutMsg utils.PoolObj, gapNumber uint64) error {
	signatures := []types.Signature{}
	for _, v := range pooledTimeouts {
		signatures = append(signatures, v.(*types.Timeout).Signature)
	}
	// Genrate TC
	timeoutCert := &types.TimeoutCert{
		Round:      currentTimeoutMsg.(*types.Timeout).Round,
		Signatures: signatures,
		GapNumber:  gapNumber,
	}
	// Process TC
	err := x.processTC(blockChainReader, timeoutCert)
	if err != nil {
		log.Error("[onTimeoutPoolThresholdReached] Fail to process TC", "TcRound", timeoutCert.Round, "NumberOfTcSig", len(timeoutCert.Signatures), "GapNumber", gapNumber, "Error", err)
		return err
	}

	log.Info("[onTimeoutPoolThresholdReached] process TC successfully", "TcRound", timeoutCert.Round, "NumberOfTcSig", len(timeoutCert.Signatures))
	return nil
}

func (x *XDPoS_v2) getTCEpochInfo(chain consensus.ChainReader, timeoutRound types.Round) (*types.EpochSwitchInfo, error) {
	epochSwitchInfo, err := x.getEpochSwitchInfo(chain, (chain.CurrentHeader()), (chain.CurrentHeader()).Hash())
	if err != nil {
		log.Error("[getTCEpochInfo] Error when getting epoch switch info", "error", err)
		return nil, fmt.Errorf("fail on getTCEpochInfo due to failure in getting epoch switch info, %s", err)
	}

	epochRound := epochSwitchInfo.EpochSwitchBlockInfo.Round
	tempTCEpoch := x.config.V2.SwitchEpoch + uint64(epochRound)/x.config.Epoch

	epochBlockInfo := &types.BlockInfo{
		Hash:   epochSwitchInfo.EpochSwitchBlockInfo.Hash,
		Round:  epochRound,
		Number: epochSwitchInfo.EpochSwitchBlockInfo.Number,
	}
	log.Info("[getTCEpochInfo] Init epochInfo", "number", epochBlockInfo.Number, "round", epochRound, "tcRound", timeoutRound, "tcEpoch", tempTCEpoch)
	for epochBlockInfo.Round > timeoutRound && tempTCEpoch > 0 {
		tempTCEpoch--
		epochBlockInfo, err = x.GetBlockByEpochNumber(chain, tempTCEpoch)
		if err != nil {
			log.Error("[getTCEpochInfo] Error when getting epoch block info by tc round", "error", err)
			return nil, fmt.Errorf("fail on getTCEpochInfo due to failure in getting epoch block info tc round, %s", err)
		}
		log.Debug("[getTCEpochInfo] Loop to get right epochInfo", "number", epochBlockInfo.Number, "round", epochBlockInfo.Round, "tcRound", timeoutRound, "tcEpoch", tempTCEpoch)
	}
	tcEpoch := tempTCEpoch
	log.Info("[getTCEpochInfo] Final TC epochInfo", "number", epochBlockInfo.Number, "round", epochBlockInfo.Round, "tcRound", timeoutRound, "tcEpoch", tcEpoch)

	epochInfo, err := x.getEpochSwitchInfo(chain, nil, epochBlockInfo.Hash)
	if err != nil {
		log.Error("[getTCEpochInfo] Error when getting epoch switch info", "error", err)
		return nil, fmt.Errorf("fail on getTCEpochInfo due to failure in getting epoch switch info, %s", err)
	}
	return epochInfo, nil
}

func (x *XDPoS_v2) verifyTC(chain consensus.ChainReader, timeoutCert *types.TimeoutCert) error {
	if timeoutCert == nil || timeoutCert.Signatures == nil {
		log.Warn("[verifyTC] TC or TC signatures is Nil")
		return utils.ErrInvalidTC
	}

	epochInfo, err := x.getTCEpochInfo(chain, timeoutCert.Round)
	if err != nil {
		return err
	}

	signedTimeoutObj := types.TimeoutSigHash(&types.TimeoutForSign{
		Round:     timeoutCert.Round,
		GapNumber: timeoutCert.GapNumber,
	})

	numValidSignatures, err := x.countValidSignatures(signedTimeoutObj, timeoutCert.Signatures, epochInfo.Masternodes)
	if err != nil {
		log.Error("[verifyTC] Error while verifying TC message signatures", "tcRound", timeoutCert.Round, "tcGapNumber", timeoutCert.GapNumber, "tcSignLen", len(timeoutCert.Signatures), "Error", err)
		return err
	}

	certThreshold := x.config.V2.Config(uint64(timeoutCert.Round)).CertThreshold
	if float64(numValidSignatures) < float64(epochInfo.MasternodesLen)*certThreshold {
		log.Warn("[verifyTC] Invalid TC Signature is less or empty", "tcRound", timeoutCert.Round, "tcGapNumber", timeoutCert.GapNumber, "tcSignLen", len(timeoutCert.Signatures), "certThreshold", float64(epochInfo.MasternodesLen)*certThreshold)
		return utils.ErrInvalidTCSignatures
	}

	return nil
}

/*
1. Update highestTC
2. Check TC round >= node's currentRound. If yes, call setNewRound
*/
func (x *XDPoS_v2) processTC(blockChainReader consensus.ChainReader, timeoutCert *types.TimeoutCert) error {
	if x.highestTimeoutCert.Round < timeoutCert.Round {
		x.highestTimeoutCert = timeoutCert
	}
	if timeoutCert.Round >= x.currentRound {
		x.setNewRound(blockChainReader, timeoutCert.Round+1)
	}
	return nil
}

// Generate and send timeout into BFT channel.
/*
	1. timeout.round = currentRound
	2. Sign the signature
	3. send to broadcast channel
*/
func (x *XDPoS_v2) sendTimeout(chain consensus.ChainReader) error {
	// Construct the gapNumber
	var gapNumber uint64
	currentBlockHeader := chain.CurrentHeader()
	isEpochSwitch, epochNum, err := x.isEpochSwitchAtRound(x.currentRound, currentBlockHeader)
	if err != nil {
		log.Error("[sendTimeout] Error while checking if the currentBlock is epoch switch", "currentRound", x.currentRound, "currentBlockNum", currentBlockHeader.Number, "currentBlockHash", currentBlockHeader.Hash(), "epochNum", epochNum)
		return err
	}

	if isEpochSwitch {
		// Notice this +1 is because we expect a block whos is the child of currentHeader
		currentNumber := currentBlockHeader.Number.Uint64() + 1
		gapNumber = currentNumber - currentNumber%x.config.Epoch
		if gapNumber > x.config.Gap {
			gapNumber -= x.config.Gap
		} else {
			gapNumber = 0
		}
		log.Debug("[sendTimeout] is epoch switch when sending out timeout message", "currentNumber", currentNumber, "gapNumber", gapNumber)
	} else {
		epochSwitchInfo, err := x.getEpochSwitchInfo(chain, currentBlockHeader, currentBlockHeader.Hash())
		if err != nil {
			log.Error("[sendTimeout] Error when trying to get current epoch switch info for a non-epoch block", "currentRound", x.currentRound, "currentBlockNum", currentBlockHeader.Number, "currentBlockHash", currentBlockHeader.Hash(), "epochNum", epochNum)
			return err
		}
		gapNumber = epochSwitchInfo.EpochSwitchBlockInfo.Number.Uint64() - epochSwitchInfo.EpochSwitchBlockInfo.Number.Uint64()%x.config.Epoch
		if gapNumber > x.config.Gap {
			gapNumber -= x.config.Gap
		} else {
			gapNumber = 0
		}
		log.Debug("[sendTimeout] non-epoch-switch block found its epoch block and calculated the gapNumber", "epochSwitchInfo.EpochSwitchBlockInfo.Number", epochSwitchInfo.EpochSwitchBlockInfo.Number.Uint64(), "gapNumber", gapNumber)
	}

	signedHash, err := x.signSignature(types.TimeoutSigHash(&types.TimeoutForSign{
		Round:     x.currentRound,
		GapNumber: gapNumber,
	}))
	if err != nil {
		log.Error("[sendTimeout] signSignature when sending out TC", "Error", err, "round", x.currentRound, "gap", gapNumber)
		return err
	}
	timeoutMsg := &types.Timeout{
		Round:     x.currentRound,
		Signature: signedHash,
		GapNumber: gapNumber,
	}

	timeoutMsg.SetSigner(x.signer)
	log.Warn("[sendTimeout] Timeout message generated, ready to send!", "timeoutMsgRound", timeoutMsg.Round, "timeoutMsgGapNumber", timeoutMsg.GapNumber, "whosTurn", x.whosTurn)
	err = x.timeoutHandler(chain, timeoutMsg)
	if err != nil {
		log.Error("TimeoutHandler error", "TimeoutRound", timeoutMsg.Round, "Error", err)
		return err
	}
	x.broadcastToBftChannel(timeoutMsg)
	return nil
}

/*
Function that will be called by timer when countdown reaches its threshold.
In the engine v2, we would need to broadcast timeout messages to other peers
*/
func (x *XDPoS_v2) OnCountdownTimeout(time time.Time, chain interface{}) error {
	x.lock.Lock()
	defer x.lock.Unlock()

	// Check if we are within the master node list
	allow := x.allowedToSend(chain.(consensus.ChainReader), chain.(consensus.ChainReader).CurrentHeader(), "timeout")
	if !allow {
		return nil
	}
	x.processSyncInfoPool(chain.(consensus.ChainReader))

	err := x.sendTimeout(chain.(consensus.ChainReader))
	if err != nil {
		log.Error("Error while sending out timeout message at time: ", "time", time, "err", err)
		return err
	}

	x.timeoutCount++
	if x.timeoutCount%x.config.V2.GetCurrentConfig().TimeoutSyncThreshold == 0 {
		syncInfo := x.getSyncInfo()
		log.Info("[OnCountdownTimeout] Timeout sync threshold reached, send syncInfo message", "QC round", syncInfo.HighestQuorumCert.ProposedBlockInfo.Round, "QC num", syncInfo.HighestQuorumCert.ProposedBlockInfo.Number, "QC sigs", len(syncInfo.HighestQuorumCert.Signatures), "TC round", syncInfo.HighestTimeoutCert.Round, "TC sigs", len(syncInfo.HighestTimeoutCert.Signatures))
		x.broadcastToBftChannel(syncInfo)
	}

	return nil
}

func (x *XDPoS_v2) hygieneTimeoutPool() {
	x.lock.RLock()
	currentRound := x.currentRound
	x.lock.RUnlock()
	timeoutPoolKeys := x.timeoutPool.PoolObjKeysList()

	// Extract round number
	for _, k := range timeoutPoolKeys {
		keyedRound, err := strconv.ParseInt(strings.Split(k, ":")[0], 10, 64)
		if err != nil {
			log.Error("[hygieneTimeoutPool] Error while trying to get keyedRound inside pool", "Error", err)
			continue
		}
		// Clean up any timeouts round that is 10 rounds older
		if keyedRound < int64(currentRound)-utils.PoolHygieneRound {
			log.Debug("[hygieneTimeoutPool] Cleaned timeout pool at round", "Round", keyedRound, "CurrentRound", currentRound, "Key", k)
			x.timeoutPool.ClearByPoolKey(k)
		}
	}
}

func (x *XDPoS_v2) ReceivedTimeouts() map[string]map[common.Hash]utils.PoolObj {
	return x.timeoutPool.Get()
}
