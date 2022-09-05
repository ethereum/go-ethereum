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
func (x *XDPoS_v2) yourturn(chain consensus.ChainReader, round types.Round, parent *types.Header, signer common.Address) (bool, error) {
	if round <= x.highestSelfMinedRound {
		log.Warn("[yourturn] Already mined on this round", "Round", round, "highestSelfMinedRound", x.highestSelfMinedRound, "ParentHash", parent.Hash().Hex(), "ParentNumber", parent.Number)
		return false, utils.ErrAlreadyMined
	}

	isEpochSwitch, _, err := x.isEpochSwitchAtRound(round, parent)
	if err != nil {
		log.Error("[yourturn] check epoch switch at round failed", "Error", err)
		return false, err
	}
	var masterNodes []common.Address
	if isEpochSwitch {
		masterNodes, _, err = x.calcMasternodes(chain, big.NewInt(0).Add(parent.Number, big.NewInt(1)), parent.Hash())
		if err != nil {
			log.Error("[yourturn] Cannot calcMasternodes at gap num ", "err", err, "parent number", parent.Number)
			return false, err
		}
	} else {
		// this block and parent belong to the same epoch
		masterNodes = x.GetMasternodes(chain, parent)
	}

	if len(masterNodes) == 0 {
		log.Error("[yourturn] Fail to find any master nodes from current block round epoch", "Hash", parent.Hash(), "CurrentRound", round, "Number", parent.Number)
		return false, errors.New("masternodes not found")
	}

	curIndex := utils.Position(masterNodes, signer)
	if curIndex == -1 {
		log.Warn("[yourturn] I am not in masternodes list", "Hash", parent.Hash(), "signer", signer)
		return false, nil
	}

	for i, s := range masterNodes {
		log.Debug("[yourturn] Masternode:", "index", i, "address", s.String(), "parentBlockNum", parent.Number)
	}

	leaderIndex := uint64(round) % x.config.Epoch % uint64(len(masterNodes))
	x.whosTurn = masterNodes[leaderIndex]
	if x.whosTurn != signer {
		log.Info("[yourturn] Not my turn", "curIndex", curIndex, "leaderIndex", leaderIndex, "Hash", parent.Hash().Hex(), "whosTurn", x.whosTurn, "myaddr", signer)
		return false, nil
	}

	log.Info("[yourturn] Yes, it's my turn based on parent block", "ParentHash", parent.Hash().Hex(), "ParentBlockNumber", parent.Number.Uint64())
	return true, nil
}
