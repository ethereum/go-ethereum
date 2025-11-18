package hooks

import (
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/consensus"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"
	"github.com/XinFinOrg/XDPoSChain/contracts"
	"github.com/XinFinOrg/XDPoSChain/core"
	"github.com/XinFinOrg/XDPoSChain/core/state"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/eth/util"
	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/XinFinOrg/XDPoSChain/params"
)

func AttachConsensusV2Hooks(adaptor *XDPoS.XDPoS, bc *core.BlockChain, chainConfig *params.ChainConfig) {
	// Hook scans for bad masternodes and decide to penalty them
	adaptor.EngineV2.HookPenalty = func(chain consensus.ChainReader, number *big.Int, parentHash common.Hash, candidates []common.Address) ([]common.Address, error) {
		start := time.Now()
		listBlockHash := []common.Hash{}
		// get list block hash & stats total created block
		statMiners := make(map[common.Address]int)
		listBlockHash = append(listBlockHash, parentHash)
		parentNumber := number.Uint64() - 1
		currentHash := parentHash

		// check and wait the latest block is already in the disk
		// sometimes blocks are yet inserted into block
		for timeout := 0; ; timeout++ {
			parentHeader := chain.GetHeader(parentHash, parentNumber)
			if parentHeader != nil { // found the latest block in the disk
				break
			}
			log.Info("[V2 Hook Penalty] parentHeader is nil, wait block to be writen in disk", "parentNumber", parentNumber)
			time.Sleep(time.Second) // 1s

			if timeout > 30 { // wait over 30s
				log.Error("[V2 Hook Penalty] parentHeader is nil, wait too long not writen in to disk", "parentNumber", parentNumber)
				return []common.Address{}, errors.New("parentHeader is nil")
			}
		}

		for i := uint64(1); ; i++ {
			parentHeader := chain.GetHeader(parentHash, parentNumber)
			if parentHeader == nil {
				log.Error("[HookPenalty] fail to get parent header")
				return []common.Address{}, fmt.Errorf("hook penalty fail to get parent header at number: %v, hash: %v", parentNumber, parentHash)
			}
			isEpochSwitch, _, err := adaptor.EngineV2.IsEpochSwitch(parentHeader)
			if err != nil {
				log.Error("[HookPenalty] isEpochSwitch", "err", err)
				return []common.Address{}, err
			}
			if isEpochSwitch {
				break
			}
			miner := parentHeader.Coinbase // we can directly use coinbase, since it's verified
			_, exist := statMiners[miner]
			if exist {
				statMiners[miner]++
			} else {
				statMiners[miner] = 1
			}
			parentNumber--
			parentHash = parentHeader.ParentHash
			listBlockHash = append(listBlockHash, parentHash)
		}

		// add list not miner to penalties
		preMasternodes := adaptor.EngineV2.GetMasternodesByHash(chain, currentHash)
		penalties := []common.Address{}
		for miner, total := range statMiners {
			if total < common.MinimunMinerBlockPerEpoch {
				log.Info("[HookPenalty] Find a node does not create enough block", "addr", miner.Hex(), "total", total, "require", common.MinimunMinerBlockPerEpoch)
				penalties = append(penalties, miner)
			}
		}
		for _, addr := range preMasternodes {
			if _, exist := statMiners[addr]; !exist {
				log.Info("[HookPenalty] Find a node do not create any block", "addr", addr.Hex())
				penalties = append(penalties, addr)
			}
		}

		// get list check penalties signing block & list master nodes wil comeback
		// start to calc comeback at v2 block + limitPenaltyEpochV2 to avoid reading v1 blocks
		comebackHeight := (common.LimitPenaltyEpochV2+1)*chain.Config().XDPoS.Epoch + chain.Config().XDPoS.V2.SwitchBlock.Uint64()
		penComebacks := []common.Address{}
		if number.Uint64() > comebackHeight {
			pens := adaptor.EngineV2.GetPreviousPenaltyByHash(chain, currentHash, common.LimitPenaltyEpochV2)
			for _, p := range pens {
				for _, addr := range candidates {
					if p == addr {
						log.Info("[HookPenalty] get previous penalty node and add into comeback list", "addr", addr)
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
			if len(penComebacks) == 0 {
				break
			}
			blockNumber := number.Uint64() - uint64(i) - 1
			bhash := listBlockHash[i]
			if blockNumber%common.MergeSignRange == 0 {
				mapBlockHash[bhash] = true
			}
			signingTxs, ok := adaptor.GetCachedSigningTxs(bhash)
			if !ok {
				block := chain.GetBlock(bhash, blockNumber)
				txs := block.Transactions()
				signingTxs = adaptor.CacheSigningTxs(bhash, txs)
			}
			// Check signer signed?
			for _, tx := range signingTxs {
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
		}

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

		for i, p := range penalties {
			log.Info("[HookPenalty] Final penalty list", "index", i, "addr", p)
		}
		log.Info("[HookPenalty] Time Calculated HookPenaltyV2 ", "block", number, "time", common.PrettyDuration(time.Since(start)))
		return penalties, nil
	}

	// Hook calculates reward for masternodes
	adaptor.EngineV2.HookReward = func(chain consensus.ChainReader, stateBlock *state.StateDB, parentState *state.StateDB, header *types.Header) (map[string]interface{}, error) {
		number := header.Number.Uint64()
		foundationWalletAddr := chain.Config().XDPoS.FoudationWalletAddr
		if foundationWalletAddr == (common.Address{}) {
			log.Error("Foundation Wallet Address is empty", "error", foundationWalletAddr)
			return nil, errors.New("foundation wallet address is empty")
		}
		rewards := make(map[string]interface{})
		// skip hook reward if this is the first v2
		if number == chain.Config().XDPoS.V2.SwitchBlock.Uint64()+1 {
			return rewards, nil
		}
		start := time.Now()
		// Get reward inflation.
		chainReward := new(big.Int).Mul(new(big.Int).SetUint64(chain.Config().XDPoS.Reward), new(big.Int).SetUint64(params.Ether))
		chainReward = util.RewardInflation(chain, chainReward, number, common.BlocksPerYear)

		// Get signers/signing tx count
		totalSigner := new(uint64)
		signers, err := GetSigningTxCount(adaptor, chain, header, totalSigner)

		log.Debug("Time Get Signers", "block", header.Number.Uint64(), "time", common.PrettyDuration(time.Since(start)))
		if err != nil {
			log.Error("[HookReward] Fail to get signers count for reward checkpoint", "error", err)
			return nil, err
		}
		rewards["signers"] = signers
		rewardSigners, err := contracts.CalculateRewardForSigner(chainReward, signers, *totalSigner)
		if err != nil {
			log.Error("[HookReward] Fail to calculate reward for signers", "error", err)
			return nil, err
		}
		// Add reward for coin holders.
		voterResults := make(map[common.Address]interface{})
		if len(signers) > 0 {
			for signer, calcReward := range rewardSigners {
				rewards, err := contracts.CalculateRewardForHolders(foundationWalletAddr, parentState, signer, calcReward, number)
				if err != nil {
					log.Error("[HookReward] Fail to calculate reward for holders.", "error", err)
					return nil, err
				}
				if len(rewards) > 0 {
					for holder, reward := range rewards {
						stateBlock.AddBalance(holder, reward)
					}
				}
				voterResults[signer] = rewards
			}
		}
		rewards["rewards"] = voterResults
		log.Debug("Time Calculated HookReward ", "block", header.Number.Uint64(), "time", common.PrettyDuration(time.Since(start)))
		return rewards, nil
	}
}

// get signing transaction sender count
func GetSigningTxCount(c *XDPoS.XDPoS, chain consensus.ChainReader, header *types.Header, totalSigner *uint64) (map[common.Address]*contracts.RewardLog, error) {
	// header should be a new epoch switch block
	number := header.Number.Uint64()
	rewardEpochCount := 2
	signEpochCount := 1
	signers := make(map[common.Address]*contracts.RewardLog)
	mapBlkHash := map[uint64]common.Hash{}

	// prevent overflow
	if number == 0 {
		return signers, nil
	}

	data := make(map[common.Hash][]common.Address)
	epochCount := 0
	var masternodes []common.Address
	var startBlockNumber, endBlockNumber uint64
	for i := number - 1; ; i-- {
		header = chain.GetHeader(header.ParentHash, i)
		isEpochSwitch, _, err := c.IsEpochSwitch(header)
		if err != nil {
			return nil, err
		}
		if isEpochSwitch && i != chain.Config().XDPoS.V2.SwitchBlock.Uint64()+1 {
			epochCount += 1
			if epochCount == signEpochCount {
				endBlockNumber = header.Number.Uint64() - 1
			}
			if epochCount == rewardEpochCount {
				startBlockNumber = header.Number.Uint64() + 1
				masternodes = c.GetMasternodesFromCheckpointHeader(header)
				break
			}
		}
		mapBlkHash[i] = header.Hash()
		signingTxs, ok := c.GetCachedSigningTxs(header.Hash())
		if !ok {
			log.Debug("Failed get from cached", "hash", header.Hash().String(), "number", i)
			block := chain.GetBlock(header.Hash(), i)
			if block != nil {
				txs := block.Transactions()
				signingTxs = c.CacheSigningTxs(header.Hash(), txs)
			}
		}
		for _, tx := range signingTxs {
			blkHash := common.BytesToHash(tx.Data()[len(tx.Data())-32:])
			from := *tx.From()
			data[blkHash] = append(data[blkHash], from)
		}
		// prevent overflow
		if i == 0 {
			return signers, nil
		}
	}

	for i := startBlockNumber; i <= endBlockNumber; i++ {
		if i%common.MergeSignRange == 0 {
			addrs := data[mapBlkHash[i]]
			// Filter duplicate address.
			if len(addrs) > 0 {
				addrSigners := make(map[common.Address]bool)
				for _, masternode := range masternodes {
					for _, addr := range addrs {
						if addr == masternode {
							if _, ok := addrSigners[addr]; !ok {
								addrSigners[addr] = true
							}
							break
						}
					}
				}

				for addr := range addrSigners {
					_, exist := signers[addr]
					if exist {
						signers[addr].Sign++
					} else {
						signers[addr] = &contracts.RewardLog{Sign: 1, Reward: new(big.Int)}
					}
					*totalSigner++
				}
			}
		}
	}

	log.Info("Calculate reward at checkpoint", "startBlock", startBlockNumber, "endBlock", endBlockNumber)

	return signers, nil
}
