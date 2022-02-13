package hooks

import (
	"math/big"
	"time"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/consensus"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"
	"github.com/XinFinOrg/XDPoSChain/core"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/XinFinOrg/XDPoSChain/params"
)

func AttachConsensusV2Hooks(adaptor *XDPoS.XDPoS, bc *core.BlockChain, chainConfig *params.ChainConfig) {
	// Hook scans for bad masternodes and decide to penalty them
	adaptor.EngineV2.HookPenalty = func(chain consensus.ChainReader, number *big.Int, parentHash common.Hash, candidates []common.Address) ([]common.Address, error) {
		start := time.Now()
		listBlockHash := make([]common.Hash, chain.Config().XDPoS.Epoch)

		// get list block hash & stats total created block
		statMiners := make(map[common.Address]int)
		listBlockHash[0] = parentHash
		parentNumber := number.Uint64() - 1
		pHash := parentHash
		for i := uint64(1); ; i++ {
			parentHeader := chain.GetHeader(pHash, parentNumber)
			b, _, err := adaptor.EngineV2.IsEpochSwitch(parentHeader)
			if err != nil {
				log.Error("[HookPenalty]", "err", err)
				return []common.Address{}, err
			}
			if b {
				break
			}
			miner := parentHeader.Coinbase // we can directly use coinbase, since it's verified (Verification is a TODO)
			value, exist := statMiners[miner]
			if exist {
				value = value + 1
			} else {
				value = 1
			}
			statMiners[miner] = value
			pHash = parentHeader.ParentHash
			parentNumber--
			listBlockHash[i] = pHash
		}

		// add list not miner to penalties
		preMasternodes := adaptor.EngineV2.GetMasternodesByHash(chain, parentHash)
		penalties := []common.Address{}
		for miner, total := range statMiners {
			if total < common.MinimunMinerBlockPerEpoch {
				log.Debug("Find a node not enough requirement create block", "addr", miner.Hex(), "total", total)
				penalties = append(penalties, miner)
			}
		}
		for _, addr := range preMasternodes {
			if _, exist := statMiners[addr]; !exist {
				log.Debug("Find a node don't create block", "addr", addr.Hex())
				penalties = append(penalties, addr)
			}
		}

		// get list check penalties signing block & list master nodes wil comeback
		// start to calc comeback at v2 block + limitPenaltyEpoch to avoid reading v1 blocks
		comebackHeight := (common.LimitPenaltyEpoch+1)*chain.Config().XDPoS.Epoch + chain.Config().XDPoS.V2.SwitchBlock.Uint64()
		penComebacks := []common.Address{}
		if number.Uint64() > comebackHeight {
			pens := adaptor.EngineV2.GetPreviousPenaltyByHash(chain, parentHash, common.LimitPenaltyEpoch)
			for _, p := range pens {
				for _, addr := range candidates {
					if p == addr {
						penComebacks = append(penComebacks, p)
						break
					}
				}
			}
		}

		// Loop for each block to check missing sign. with comeback nodes
		mapBlockHash := map[common.Hash]bool{}
		startRange := common.RangeReturnSigner - 1
		// to prevent visiting outside index of listBlockHash
		if startRange >= len(listBlockHash) {
			startRange = len(listBlockHash) - 1
		}
		for i := startRange; i >= 0; i-- {
			if len(penComebacks) > 0 {
				blockNumber := number.Uint64() - uint64(i) - 1
				bhash := listBlockHash[i]
				if blockNumber%common.MergeSignRange == 0 {
					mapBlockHash[bhash] = true
				}
				signData, ok := adaptor.GetCachedSigningTxs(bhash)
				if !ok {
					block := chain.GetBlock(bhash, blockNumber)
					txs := block.Transactions()
					signData = adaptor.CacheSigningTxs(bhash, txs)
				}
				txs := signData.([]*types.Transaction)
				// Check signer signed?
				for _, tx := range txs {
					blkHash := common.BytesToHash(tx.Data()[len(tx.Data())-32:])
					from := *tx.From()
					if mapBlockHash[blkHash] {
						for j, addr := range penComebacks {
							if from == addr {
								// Remove it from dupSigners.
								penComebacks = append(penComebacks[:j], penComebacks[j+1:]...)
								break
							}
						}
					}
				}
			} else {
				break
			}
		}

		log.Debug("Time Calculated HookPenaltyV2 ", "block", number, "pen comeback nodes", len(penComebacks), "not enough miner", len(penalties), "time", common.PrettyDuration(time.Since(start)))
		for _, comeback := range penComebacks {
			ok := true
			for _, p := range penalties {
				if p == comeback {
					ok = false
					break
				}
			}
			if ok {
				penalties = append(penalties, comeback)
			}
		}
		return penalties, nil
	}

}
