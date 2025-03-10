package hooks

import (
	"errors"
	"math/big"
	"time"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/common/sort"
	"github.com/XinFinOrg/XDPoSChain/consensus"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/contracts"
	"github.com/XinFinOrg/XDPoSChain/core"
	"github.com/XinFinOrg/XDPoSChain/core/state"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/eth/util"
	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/XinFinOrg/XDPoSChain/params"
)

// Declaring an enum type Beneficiary of reward
type Beneficiary int

// Enumerating reward beneficiary
const (
	MasterNodeBeneficiary Beneficiary = iota
	ProtectorNodeBeneficiary
	ObserverNodeBeneficiary
)

type RewardLog struct {
	Sign   uint64   `json:"sign"`
	Reward *big.Int `json:"reward"`
}

func AttachConsensusV2Hooks(adaptor *XDPoS.XDPoS, bc *core.BlockChain, chainConfig *params.ChainConfig) {
	// Hook scans for bad masternodes and decide to penalty them
	adaptor.EngineV2.HookPenalty = func(chain consensus.ChainReader, number *big.Int, currentHash common.Hash, candidates []common.Address) ([]common.Address, error) {
		start := time.Now()
		listBlockHash := []common.Hash{}
		// get list block hash & stats total created block
		statMiners := make(map[common.Address]int)
		listBlockHash = append(listBlockHash, currentHash)
		parentNumber := number.Uint64() - 1
		parentHash := currentHash

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
		rewardsMap := make(map[string]interface{})
		// skip hook reward if this is the first v2
		if number == chain.Config().XDPoS.V2.SwitchBlock.Uint64()+1 {
			return rewardsMap, nil
		}
		start := time.Now()

		round, err := adaptor.EngineV2.GetRoundNumber(header)
		if err != nil {
			log.Error("[HookReward] Fail to get round", "error", err)
			return nil, err
		}
		currentConfig := chain.Config().XDPoS.V2.Config(uint64(round))
		// Get signers/signing tx count
		signers, err := GetSigningTxCount(adaptor, chain, header, parentState, currentConfig)

		log.Debug("Time Get Signers", "block", header.Number.Uint64(), "time", common.PrettyDuration(time.Since(start)))
		if err != nil {
			log.Error("[HookReward] Fail to get signers count for reward checkpoint", "error", err)
			return nil, err
		}
		rewardsMap["signers"] = signers[MasterNodeBeneficiary]

		if !chain.Config().IsTIPUpgradeReward(header.Number) {
			// Get reward inflation.
			originalReward := new(big.Int).Mul(new(big.Int).SetUint64(chain.Config().XDPoS.Reward), new(big.Int).SetUint64(params.Ether))
			chainReward := util.RewardInflation(chain, originalReward, number, common.BlocksPerYear)
			rewardSigners, err := CalculateRewardForSigner(chainReward, signers[MasterNodeBeneficiary])
			if err != nil {
				log.Error("[HookReward] Fail to calculate reward for masternode", "error", err)
				return nil, err
			}
			// Add reward for coin holders.
			voterResults := make(map[common.Address]interface{})
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
			rewardsMap["rewards"] = voterResults
		} else {
			rewardsMap["signersProtector"] = signers[ProtectorNodeBeneficiary]
			rewardsMap["signersObserver"] = signers[ObserverNodeBeneficiary]
			epochRewardTotal := new(big.Int).SetUint64(currentConfig.MasternodeReward + currentConfig.ProtectorReward + currentConfig.ObserverReward)
			type rewardWithType struct {
				r   uint64
				t   Beneficiary
				key string
			}
			for _, rwt := range []rewardWithType{
				{currentConfig.MasternodeReward, MasterNodeBeneficiary, "rewards"},
				{currentConfig.ProtectorReward, ProtectorNodeBeneficiary, "rewardsProtector"},
				{currentConfig.ObserverReward, ObserverNodeBeneficiary, "rewardsObserver"},
			} {
				originalReward := new(big.Int).Mul(new(big.Int).SetUint64(rwt.r), new(big.Int).SetUint64(params.Ether))
				chainReward := new(big.Int)
				if !chain.Config().IsTIPEpochHalving(header.Number) {
					chainReward = util.RewardInflation(chain, originalReward, number, common.BlocksPerYear)
				} else {
					halvingSupply := big.NewInt(9000000000) // TODO use config.halvingSupply
					_, epochNum, err := adaptor.EngineV2.IsEpochSwitch(header)
					if err != nil {
						return nil, err
					}
					epochSinceHalving := epochNum // TODO Minus config.epochHalvingOnset
					chainReward = util.RewardHalving(originalReward, epochRewardTotal, halvingSupply, epochSinceHalving)
				}
				rewardSigners, err := CalculateRewardForSigner(chainReward, signers[rwt.t])
				if err != nil {
					log.Error("[HookReward] Fail to calculate reward type 0 for masternode, 1 for protector, 2 for observer", "error", err, "type", rwt.t)
					return nil, err
				}
				// Add reward for coin holders.
				voterResults := make(map[common.Address]interface{})
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
				rewardsMap[rwt.key] = voterResults
			}
		}
		log.Debug("Time Calculated HookReward ", "block", header.Number.Uint64(), "time", common.PrettyDuration(time.Since(start)))
		return rewardsMap, nil
	}
}

// get signing transaction sender count
func GetSigningTxCount(c *XDPoS.XDPoS, chain consensus.ChainReader, header *types.Header, parentState *state.StateDB, currentConfig *params.V2Config) (map[Beneficiary]map[common.Address]*RewardLog, error) {
	// header should be a new epoch switch block
	number := header.Number.Uint64()
	rewardEpochCount := 2
	signEpochCount := 1
	signers := make(map[Beneficiary]map[common.Address]*RewardLog)
	signers[MasterNodeBeneficiary] = make(map[common.Address]*RewardLog)
	signers[ProtectorNodeBeneficiary] = make(map[common.Address]*RewardLog)
	signers[ObserverNodeBeneficiary] = make(map[common.Address]*RewardLog)

	mapBlkHash := map[uint64]common.Hash{}

	// prevent overflow
	if number == 0 {
		return signers, nil
	}

	data := make(map[common.Hash][]common.Address)
	epochCount := 0
	var startBlockNumber, endBlockNumber uint64

	nodesToKeep := make(map[Beneficiary][]common.Address)

	h := header
	for i := number - 1; ; i-- {
		h = chain.GetHeader(h.ParentHash, i)
		isEpochSwitch, _, err := c.IsEpochSwitch(h)
		if err != nil {
			return nil, err
		}
		if isEpochSwitch && i != chain.Config().XDPoS.V2.SwitchBlock.Uint64()+1 {
			epochCount += 1
			if epochCount == signEpochCount {
				endBlockNumber = h.Number.Uint64() - 1
			}
			if epochCount == rewardEpochCount {
				startBlockNumber = h.Number.Uint64() + 1
				nodesToKeep[MasterNodeBeneficiary] = c.GetMasternodesFromCheckpointHeader(h)
				// in reward upgrade, add protector and observer nodes
				if chain.Config().IsTIPUpgradeReward(header.Number) {
					candidates := state.GetCandidates(parentState)
					var ms []utils.Masternode
					for _, candidate := range candidates {
						// ignore "0x0000000000000000000000000000000000000000"
						if !candidate.IsZero() {
							v := state.GetCandidateCap(parentState, candidate)
							ms = append(ms, utils.Masternode{Address: candidate, Stake: v})
						}
					}
					sort.Slice(ms, func(i, j int) bool {
						return ms[i].Stake.Cmp(ms[j].Stake) >= 0
					})
					// find penalty and filter them out
					penalties := common.ExtractAddressFromBytes(h.Penalties)
					filterMap := make(map[common.Address]struct{})
					for _, addr := range penalties {
						filterMap[addr] = struct{}{}
					}
					for _, addr := range nodesToKeep[MasterNodeBeneficiary] {
						filterMap[addr] = struct{}{}
					}
					// find top candidates
					protector := []common.Address{}
					observer := []common.Address{}
					for _, node := range ms {
						if _, ok := filterMap[node.Address]; ok {
							continue
						}
						if len(protector) < currentConfig.MaxProtectorNodes {
							protector = append(protector, node.Address)
						} else {
							observer = append(observer, node.Address)
						}
					}
					nodesToKeep[ProtectorNodeBeneficiary] = protector
					nodesToKeep[ObserverNodeBeneficiary] = observer
				}
				break
			}
		}
		mapBlkHash[i] = h.Hash()
		signingTxs, ok := c.GetCachedSigningTxs(h.Hash())
		if !ok {
			log.Debug("Failed get from cached", "hash", h.Hash().String(), "number", i)
			block := chain.GetBlock(h.Hash(), i)
			txs := block.Transactions()
			signingTxs = c.CacheSigningTxs(h.Hash(), txs)
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
				addrSigners := make(map[Beneficiary]map[common.Address]bool)
				addrSigners[MasterNodeBeneficiary] = make(map[common.Address]bool)
				addrSigners[ProtectorNodeBeneficiary] = make(map[common.Address]bool)
				addrSigners[ObserverNodeBeneficiary] = make(map[common.Address]bool)

				for _, addr := range addrs {
					for _, beneficiary := range []Beneficiary{MasterNodeBeneficiary, ProtectorNodeBeneficiary, ObserverNodeBeneficiary} {
						if _, ok := nodesToKeep[beneficiary]; ok {
							for _, protector := range nodesToKeep[beneficiary] {
								if addr == protector {
									if _, ok := addrSigners[beneficiary][addr]; !ok {
										addrSigners[beneficiary][addr] = true
									}
									break
								}
							}
						}
					}
				}

				for _, beneficiary := range []Beneficiary{MasterNodeBeneficiary, ProtectorNodeBeneficiary, ObserverNodeBeneficiary} {
					for addr := range addrSigners[beneficiary] {
						_, exist := signers[beneficiary][addr]
						if exist {
							signers[beneficiary][addr].Sign++
						} else {
							signers[beneficiary][addr] = &RewardLog{Sign: 1, Reward: new(big.Int)}
						}
					}
				}
			}
		}
	}

	log.Info("Calculate reward at checkpoint", "startBlock", startBlockNumber, "endBlock", endBlockNumber)

	return signers, nil
}

// Calculate reward for signers.
func CalculateRewardForSigner(chainReward *big.Int, signers map[common.Address]*RewardLog) (map[common.Address]*big.Int, error) {
	totalSignerCount := uint64(0)
	for _, rLog := range signers {
		totalSignerCount += rLog.Sign
	}
	resultSigners := make(map[common.Address]*big.Int)
	// Add reward for signers.
	if totalSignerCount > 0 {
		for signer, rLog := range signers {
			// Add reward for signer.
			calcReward := new(big.Int)
			calcReward.Div(chainReward, new(big.Int).SetUint64(totalSignerCount))
			calcReward.Mul(calcReward, new(big.Int).SetUint64(rLog.Sign))
			rLog.Reward = calcReward

			resultSigners[signer] = calcReward
		}
	}

	log.Info("Signers data", "totalSigner", totalSignerCount, "totalReward", chainReward)
	for addr, signer := range signers {
		log.Debug("Signer reward", "signer", addr, "sign", signer.Sign, "reward", signer.Reward)
	}

	return resultSigners, nil
}

// func TestRewardBeZero(t *testing.T) {
// 	billion := big.NewInt(1000000000)
// 	epochRewardTotal := big.NewInt(16000)
// 	epochRewardTotal.Mul(epochRewardTotal, billion)
// 	epochReward1 := big.NewInt(10000)
// 	epochReward1.Mul(epochReward1, billion)
// 	epochReward2 := big.NewInt(4000)
// 	epochReward2.Mul(epochReward2, billion)
// 	epochReward3 := big.NewInt(2000)
// 	epochReward3.Mul(epochReward3, billion)
// 	// 45 Billion - 39 Billion XDC (1 XDC = 10^9 wei)
// 	halvingSupply := big.NewInt(6000000000)
// 	halvingSupply.Mul(halvingSupply, billion)
// 	sum := big.NewInt(0)
// 	for i := uint64(0); i < 30000000; i++ {
// 		r := new(big.Int).Add(RewardHalving(epochReward1, epochRewardTotal, halvingSupply, i), RewardHalving(epochReward2, epochRewardTotal, halvingSupply, i))
// 		r.Add(r, RewardHalving(epochReward3, epochRewardTotal, halvingSupply, i))
// 		if r.BitLen() == 0 {
// 			t.Log("reward be 0 at i=", i) // reward be 0 at i= 11225088, wich is more than 200 years in the future
// 			break
// 		}
// 		sum.Add(sum, r)
// 	}
// 	t.Log("sum", sum) // sum 5999999999982635022, which is less than total, and never reach totoal
// 	assert.True(t, sum.Cmp(halvingSupply) < 0)
// }
