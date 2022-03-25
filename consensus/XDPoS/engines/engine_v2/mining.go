package engine_v2

import (
	"errors"
	"math/big"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/consensus"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/log"
)

// Using parent and current round to find the finalised master node list(with penalties applied from last epoch)
func (x *XDPoS_v2) checkYourturnWithinFinalisedMasternodes(chain consensus.ChainReader, round utils.Round, parent *types.Header, signer common.Address) (bool, error) {
	isEpochSwitch, _, err := x.isEpochSwitchAtRound(round, parent)
	if err != nil {
		log.Error("[checkYourturnWithinFinalisedMasternodes] check epoch switch at round failed", "Error", err)
		return false, err
	}
	var masterNodes []common.Address
	if isEpochSwitch {
		if x.config.V2.SwitchBlock.Cmp(parent.Number) == 0 {
			// the initial master nodes of v1->v2 switch contains penalties node
			_, _, masterNodes, err = x.getExtraFields(parent)
			if err != nil {
				log.Error("[checkYourturnWithinFinalisedMasternodes] Cannot find snapshot at gap num of last V1", "err", err, "number", x.config.V2.SwitchBlock.Uint64())
				return false, err
			}
		} else {
			masterNodes, _, err = x.calcMasternodes(chain, big.NewInt(0).Add(parent.Number, big.NewInt(1)), parent.Hash())
			if err != nil {
				log.Error("[checkYourturnWithinFinalisedMasternodes] Cannot calcMasternodes at gap num ", "err", err, "parent number", parent.Number)
				return false, err
			}
		}
	} else {
		// this block and parent belong to the same epoch
		masterNodes = x.GetMasternodes(chain, parent)
	}

	if len(masterNodes) == 0 {
		log.Error("[checkYourturnWithinFinalisedMasternodes] Fail to find any master nodes from current block round epoch", "Hash", parent.Hash(), "CurrentRound", round, "Number", parent.Number)
		return false, errors.New("masternodes not found")
	}

	curIndex := utils.Position(masterNodes, signer)
	if curIndex == -1 {
		log.Debug("[checkYourturnWithinFinalisedMasternodes] Not authorised signer", "MN", masterNodes, "Hash", parent.Hash(), "signer", signer)
		return false, nil
	}

	for i, s := range masterNodes {
		log.Debug("[checkYourturnWithinFinalisedMasternodes] Masternode:", "index", i, "address", s.String(), "parentBlockNum", parent.Number)
	}

	leaderIndex := uint64(round) % x.config.Epoch % uint64(len(masterNodes))
	if masterNodes[leaderIndex] != signer {
		log.Debug("[checkYourturnWithinFinalisedMasternodes] Not my turn", "curIndex", curIndex, "leaderIndex", leaderIndex, "Hash", parent.Hash().Hex(), "masterNodes[leaderIndex]", masterNodes[leaderIndex], "signer", signer)
		return false, nil
	}

	log.Debug("[checkYourturnWithinFinalisedMasternodes] Yes, it's my turn based on parent block", "ParentHash", parent.Hash().Hex(), "ParentBlockNumber", parent.Number.Uint64())
	return true, nil
}
