package engine_v2

import (
	"fmt"
	"math/big"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/consensus"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/log"
)

// get epoch switch of the previous `limit` epoch
func (x *XDPoS_v2) getPreviousEpochSwitchInfoByHash(chain consensus.ChainReader, hash common.Hash, limit int) (*types.EpochSwitchInfo, error) {
	epochSwitchInfo, err := x.getEpochSwitchInfo(chain, nil, hash)
	if err != nil {
		log.Error("[getPreviousEpochSwitchInfoByHash] Adaptor v2 getEpochSwitchInfo has error, potentially bug", "err", err)
		return nil, err
	}
	for i := 0; i < limit; i++ {
		epochSwitchInfo, err = x.getEpochSwitchInfo(chain, nil, epochSwitchInfo.EpochSwitchParentBlockInfo.Hash)
		if err != nil {
			log.Error("[getPreviousEpochSwitchInfoByHash] Adaptor v2 getEpochSwitchInfo has error, potentially bug", "err", err)
			return nil, err
		}
	}
	return epochSwitchInfo, nil
}

// Given header and its hash, get epoch switch info from the epoch switch block of that epoch,
// header is allow to be nil.
func (x *XDPoS_v2) getEpochSwitchInfo(chain consensus.ChainReader, header *types.Header, hash common.Hash) (*types.EpochSwitchInfo, error) {
	e, ok := x.epochSwitches.Get(hash)
	if ok {
		epochSwitchInfo := e.(*types.EpochSwitchInfo)
		log.Debug("[getEpochSwitchInfo] cache hit", "number", epochSwitchInfo.EpochSwitchBlockInfo.Number, "hash", hash.Hex())
		return epochSwitchInfo, nil
	}
	h := header
	if h == nil {
		log.Debug("[getEpochSwitchInfo] header doesn't provide, get header by hash", "hash", hash.Hex())
		h = chain.GetHeaderByHash(hash)
		if h == nil {
			log.Warn("[getEpochSwitchInfo] can not find header from db", "hash", hash.Hex())
			return nil, fmt.Errorf("[getEpochSwitchInfo] can not find header from db hash %v", hash.Hex())
		}
	}
	isEpochSwitch, _, err := x.IsEpochSwitch(h)
	if err != nil {
		return nil, err
	}
	if isEpochSwitch {
		log.Debug("[getEpochSwitchInfo] header is epoch switch", "hash", hash.Hex(), "number", h.Number.Uint64())
		quorumCert, round, masternodes, err := x.getExtraFields(h)
		if err != nil {
			return nil, err
		}
		epochSwitchInfo := &types.EpochSwitchInfo{
			Masternodes: masternodes,
			EpochSwitchBlockInfo: &types.BlockInfo{
				Hash:   hash,
				Number: h.Number,
				Round:  round,
			},
		}
		if quorumCert != nil {
			epochSwitchInfo.EpochSwitchParentBlockInfo = quorumCert.ProposedBlockInfo
		}

		x.epochSwitches.Add(hash, epochSwitchInfo)
		return epochSwitchInfo, nil
	}
	epochSwitchInfo, err := x.getEpochSwitchInfo(chain, nil, h.ParentHash)
	if err != nil {
		log.Error("[getEpochSwitchInfo] recursive error", "err", err, "hash", hash.Hex(), "number", h.Number.Uint64())
		return nil, err
	}
	log.Debug("[getEpochSwitchInfo] get epoch switch info recursively", "hash", hash.Hex(), "number", h.Number.Uint64())
	x.epochSwitches.Add(hash, epochSwitchInfo)
	return epochSwitchInfo, nil
}

// IsEpochSwitchAtRound() is used by miner to check whether it mines a block in the same epoch with parent
func (x *XDPoS_v2) isEpochSwitchAtRound(round types.Round, parentHeader *types.Header) (bool, uint64, error) {
	epochNum := x.config.V2.SwitchBlock.Uint64()/x.config.Epoch + uint64(round)/x.config.Epoch
	// if parent is last v1 block and this is first v2 block, this is treated as epoch switch
	if parentHeader.Number.Cmp(x.config.V2.SwitchBlock) == 0 {
		return true, epochNum, nil
	}

	_, parentRound, _, err := x.getExtraFields(parentHeader)
	if err != nil {
		log.Error("[IsEpochSwitch] decode header error", "err", err, "header", parentHeader, "extra", common.Bytes2Hex(parentHeader.Extra))
		return false, 0, err
	}
	if round <= parentRound {
		// this round is no larger than parentRound, should return false
		return false, epochNum, nil
	}

	epochStartRound := round - round%types.Round(x.config.Epoch)
	return parentRound < epochStartRound, epochNum, nil
}

func (x *XDPoS_v2) GetCurrentEpochSwitchBlock(chain consensus.ChainReader, blockNum *big.Int) (uint64, uint64, error) {
	header := chain.GetHeaderByNumber(blockNum.Uint64())
	epochSwitchInfo, err := x.getEpochSwitchInfo(chain, header, header.Hash())
	if err != nil {
		log.Error("[GetCurrentEpochSwitchBlock] Fail to get epoch switch info", "Num", header.Number, "Hash", header.Hash())
		return 0, 0, err
	}

	currentCheckpointNumber := epochSwitchInfo.EpochSwitchBlockInfo.Number.Uint64()
	epochNum := x.config.V2.SwitchBlock.Uint64()/x.config.Epoch + uint64(epochSwitchInfo.EpochSwitchBlockInfo.Round)/x.config.Epoch
	return currentCheckpointNumber, epochNum, nil
}

func (x *XDPoS_v2) IsEpochSwitch(header *types.Header) (bool, uint64, error) {
	// Return true directly if we are examing the last v1 block. This could happen if the calling function is examing parent block
	if header.Number.Cmp(x.config.V2.SwitchBlock) == 0 {
		log.Info("[IsEpochSwitch] examing last v1 block")
		return true, header.Number.Uint64() / x.config.Epoch, nil
	}

	quorumCert, round, _, err := x.getExtraFields(header)
	if err != nil {
		log.Error("[IsEpochSwitch] decode header error", "err", err, "header", header, "extra", common.Bytes2Hex(header.Extra))
		return false, 0, err
	}
	parentRound := quorumCert.ProposedBlockInfo.Round
	epochStartRound := round - round%types.Round(x.config.Epoch)
	epochNum := x.config.V2.SwitchBlock.Uint64()/x.config.Epoch + uint64(round)/x.config.Epoch
	// if parent is last v1 block and this is first v2 block, this is treated as epoch switch
	if quorumCert.ProposedBlockInfo.Number.Cmp(x.config.V2.SwitchBlock) == 0 {
		log.Info("[IsEpochSwitch] true, parent equals V2.SwitchBlock", "round", round, "number", header.Number.Uint64(), "hash", header.Hash())
		return true, epochNum, nil
	}
	log.Debug("[IsEpochSwitch]", "is", parentRound < epochStartRound, "parentRound", parentRound, "round", round, "number", header.Number.Uint64(), "epochNum", epochNum, "hash", header.Hash())
	return parentRound < epochStartRound, epochNum, nil
}
