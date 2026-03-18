// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package eth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/XinFinOrg/XDPoSChain/XDCx"
	"github.com/XinFinOrg/XDPoSChain/XDCx/tradingstate"
	"github.com/XinFinOrg/XDPoSChain/XDCxlending"
	"github.com/XinFinOrg/XDPoSChain/accounts"
	"github.com/XinFinOrg/XDPoSChain/accounts/abi/bind"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/consensus"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"
	"github.com/XinFinOrg/XDPoSChain/contracts"
	"github.com/XinFinOrg/XDPoSChain/core"
	"github.com/XinFinOrg/XDPoSChain/core/bloombits"
	"github.com/XinFinOrg/XDPoSChain/core/rawdb"
	"github.com/XinFinOrg/XDPoSChain/core/state"
	"github.com/XinFinOrg/XDPoSChain/core/txpool"
	"github.com/XinFinOrg/XDPoSChain/core/txpool/locals"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/core/vm"
	"github.com/XinFinOrg/XDPoSChain/eth/downloader"
	"github.com/XinFinOrg/XDPoSChain/eth/gasprice"
	"github.com/XinFinOrg/XDPoSChain/eth/tracers"
	"github.com/XinFinOrg/XDPoSChain/eth/util"
	"github.com/XinFinOrg/XDPoSChain/ethdb"
	"github.com/XinFinOrg/XDPoSChain/event"
	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/XinFinOrg/XDPoSChain/miner"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/XinFinOrg/XDPoSChain/rpc"
)

// EthAPIBackend implements ethapi.Backend and tracers.Backend for full nodes
type EthAPIBackend struct {
	allowUnprotectedTxs bool
	eth                 *Ethereum
	gpo                 *gasprice.Oracle
	XDPoS               *XDPoS.XDPoS
}

func (b *EthAPIBackend) ChainConfig() *params.ChainConfig {
	return b.eth.blockchain.Config()
}

func (b *EthAPIBackend) CurrentBlock() *types.Header {
	return b.eth.blockchain.CurrentBlock()
}

func (b *EthAPIBackend) SetHead(number uint64) {
	b.eth.protocolManager.downloader.Cancel()
	b.eth.blockchain.SetHead(number)
}

func (b *EthAPIBackend) HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Header, error) {
	// Pending block is only known by the miner
	if number == rpc.PendingBlockNumber {
		number = rpc.LatestBlockNumber
	}
	// Otherwise resolve and return the block
	if number == rpc.LatestBlockNumber {
		return b.eth.blockchain.CurrentBlock(), nil
	}
	if number == rpc.FinalizedBlockNumber {
		if b.eth.blockchain.Config().XDPoS == nil {
			return nil, errors.New("PoW does not support confirmed block lookup")
		}
		current := b.eth.blockchain.CurrentBlock()
		if b.eth.blockchain.Config().XDPoS.BlockConsensusVersion(current.Number) == params.ConsensusEngineVersion2 {
			// TO CHECK: why calling config in XDPoS is blocked (not field and method)
			confirmedHash := b.XDPoS.EngineV2.GetLatestCommittedBlockInfo().Hash
			return b.eth.blockchain.GetHeaderByHash(confirmedHash), nil
		} else {
			return nil, errors.New("PoS V1 does not support confirmed block lookup")
		}
	} else if number.Int64() < 0 {
		return nil, fmt.Errorf("invalid block number %d", number.Int64())
	}
	header := b.eth.blockchain.GetHeaderByNumber(uint64(number))
	if header == nil {
		return nil, errors.New("header for number not found")
	}
	return header, nil
}

func (b *EthAPIBackend) HeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Header, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return b.HeaderByNumber(ctx, blockNr)
	}
	if hash, ok := blockNrOrHash.Hash(); ok {
		header := b.eth.blockchain.GetHeaderByHash(hash)
		if header == nil {
			return nil, errors.New("header for hash not found")
		}
		if blockNrOrHash.RequireCanonical && b.eth.blockchain.GetCanonicalHash(header.Number.Uint64()) != hash {
			return nil, errors.New("hash is not currently canonical")
		}
		return header, nil
	}
	return nil, errors.New("invalid arguments; neither block nor hash specified")
}

func (b *EthAPIBackend) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	return b.eth.blockchain.GetHeaderByHash(hash), nil
}

func (b *EthAPIBackend) BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error) {
	// Pending block is only known by the miner
	if number == rpc.PendingBlockNumber {
		number = rpc.LatestBlockNumber
	}
	// Otherwise resolve and return the block
	if number == rpc.LatestBlockNumber {
		header := b.eth.blockchain.CurrentBlock()
		return b.eth.blockchain.GetBlock(header.Hash(), header.Number.Uint64()), nil
	}
	if number == rpc.FinalizedBlockNumber {
		if b.eth.blockchain.Config().XDPoS == nil {
			return nil, errors.New("PoW does not support confirmed block lookup")
		}
		current := b.eth.blockchain.CurrentBlock()
		if current == nil {
			return nil, errors.New("current block is nil")
		}
		if b.eth.blockchain.Config().XDPoS.BlockConsensusVersion(current.Number) == params.ConsensusEngineVersion2 {
			// TO CHECK: why calling config in XDPoS is blocked (not field and method)
			confirmedHash := b.XDPoS.EngineV2.GetLatestCommittedBlockInfo().Hash
			return b.eth.blockchain.GetBlockByHash(confirmedHash), nil
		} else {
			return nil, errors.New("PoS V1 does not support confirmed block lookup")
		}
	} else if number.Int64() < 0 {
		return nil, fmt.Errorf("invalid block number %d", number.Int64())
	}
	return b.eth.blockchain.GetBlockByNumber(uint64(number)), nil
}

func (b *EthAPIBackend) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return b.eth.blockchain.GetBlockByHash(hash), nil
}

// GetBody returns body of a block. It does not resolve special block numbers.
func (b *EthAPIBackend) GetBody(ctx context.Context, hash common.Hash, number rpc.BlockNumber) (*types.Body, error) {
	if number < 0 || hash == (common.Hash{}) {
		return nil, errors.New("invalid arguments; expect hash and no special block numbers")
	}
	if body := b.eth.blockchain.GetBody(hash); body != nil {
		return body, nil
	}
	return nil, errors.New("block body not found")
}

func (b *EthAPIBackend) BlockByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Block, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return b.BlockByNumber(ctx, blockNr)
	}
	if hash, ok := blockNrOrHash.Hash(); ok {
		header := b.eth.blockchain.GetHeaderByHash(hash)
		if header == nil {
			return nil, errors.New("header for hash not found")
		}
		if blockNrOrHash.RequireCanonical && b.eth.blockchain.GetCanonicalHash(header.Number.Uint64()) != hash {
			return nil, errors.New("hash is not currently canonical")
		}
		block := b.eth.blockchain.GetBlock(hash, header.Number.Uint64())
		if block == nil {
			return nil, errors.New("header found, but block body is missing")
		}
		return block, nil
	}
	return nil, errors.New("invalid arguments; neither block nor hash specified")
}

func (b *EthAPIBackend) PendingBlockAndReceipts() (*types.Block, types.Receipts) {
	return b.eth.miner.PendingBlockAndReceipts()
}

func (b *EthAPIBackend) StateAndHeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*state.StateDB, *types.Header, error) {
	// Pending state is only known by the miner
	if number == rpc.PendingBlockNumber {
		number = rpc.LatestBlockNumber
	}
	// Otherwise resolve the block number and return its state
	header, err := b.HeaderByNumber(ctx, number)
	if err != nil {
		return nil, nil, err
	}
	if header == nil {
		return nil, nil, errors.New("header not found")
	}
	stateDb, err := b.eth.BlockChain().StateAt(header.Root)
	if err != nil {
		return nil, nil, err
	}
	return stateDb, header, err
}

func (b *EthAPIBackend) StateAndHeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*state.StateDB, *types.Header, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return b.StateAndHeaderByNumber(ctx, blockNr)
	}
	if hash, ok := blockNrOrHash.Hash(); ok {
		header, err := b.HeaderByHash(ctx, hash)
		if err != nil {
			return nil, nil, err
		}
		if header == nil {
			return nil, nil, errors.New("header for hash not found")
		}
		if blockNrOrHash.RequireCanonical && b.eth.blockchain.GetCanonicalHash(header.Number.Uint64()) != hash {
			return nil, nil, errors.New("hash is not currently canonical")
		}
		stateDb, err := b.eth.BlockChain().StateAt(header.Root)
		if err != nil {
			return nil, nil, err
		}
		return stateDb, header, nil
	}
	return nil, nil, errors.New("invalid arguments; neither block nor hash specified")
}

func (b *EthAPIBackend) GetHeader(ctx context.Context, hash common.Hash) *types.Header {
	return b.eth.blockchain.GetHeaderByHash(hash)
}

func (b *EthAPIBackend) GetBlock(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return b.eth.blockchain.GetBlockByHash(hash), nil
}

func (b *EthAPIBackend) GetReceipts(ctx context.Context, blockHash common.Hash) (types.Receipts, error) {
	return b.eth.blockchain.GetReceiptsByHash(blockHash), nil
}

func (b *EthAPIBackend) GetLogs(ctx context.Context, hash common.Hash, number uint64) ([][]*types.Log, error) {
	return rawdb.ReadLogs(b.eth.chainDb, hash, number), nil
}

func (b *EthAPIBackend) GetTd(ctx context.Context, hash common.Hash) *big.Int {
	return b.eth.blockchain.GetTdByHash(hash)
}

func (b *EthAPIBackend) GetEVM(ctx context.Context, state *state.StateDB, XDCxState *tradingstate.TradingStateDB, header *types.Header, vmConfig *vm.Config, blockCtx *vm.BlockContext) (*vm.EVM, func() error, error) {
	vmError := func() error { return nil }
	if vmConfig == nil {
		vmConfig = b.eth.blockchain.GetVMConfig()
	}
	var context vm.BlockContext
	if blockCtx != nil {
		context = *blockCtx
	} else {
		context = core.NewEVMBlockContext(header, b.eth.BlockChain(), nil)
	}
	return vm.NewEVM(context, state, XDCxState, b.eth.blockchain.Config(), *vmConfig), vmError, nil
}

func (b *EthAPIBackend) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
	return b.eth.BlockChain().SubscribeRemovedLogsEvent(ch)
}

func (b *EthAPIBackend) SubscribePendingLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.eth.miner.SubscribePendingLogs(ch)
}

func (b *EthAPIBackend) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	return b.eth.BlockChain().SubscribeChainEvent(ch)
}

func (b *EthAPIBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return b.eth.BlockChain().SubscribeChainHeadEvent(ch)
}

func (b *EthAPIBackend) SubscribeChainSideEvent(ch chan<- core.ChainSideEvent) event.Subscription {
	return b.eth.BlockChain().SubscribeChainSideEvent(ch)
}

func (b *EthAPIBackend) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.eth.BlockChain().SubscribeLogsEvent(ch)
}

func (b *EthAPIBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	err := b.eth.txPool.Add([]*types.Transaction{signedTx}, false)[0]

	// If the local transaction tracker is not configured, returns whatever
	// returned from the txpool.
	if b.eth.localTxTracker == nil {
		return err
	}
	// If the transaction fails with an error indicating it is invalid, or if there is
	// very little chance it will be accepted later (e.g., the gas price is below the
	// configured minimum, or the sender has insufficient funds to cover the cost),
	// propagate the error to the user.
	if err != nil && !locals.IsTemporaryReject(err) {
		return err
	}
	// No error will be returned to user if the transaction fails with a temporary
	// error and might be accepted later (e.g., the transaction pool is full).
	// Locally submitted transactions will be resubmitted later via the local tracker.
	b.eth.localTxTracker.Track(signedTx)
	return nil
}

func (b *EthAPIBackend) GetPoolTransactions() (types.Transactions, error) {
	pending := b.eth.txPool.Pending(txpool.PendingFilter{})
	var txs types.Transactions
	for _, batch := range pending {
		for _, lazy := range batch {
			if tx := lazy.Resolve(); tx != nil {
				txs = append(txs, tx)
			}
		}
	}
	return txs, nil
}

func (b *EthAPIBackend) GetPoolTransaction(hash common.Hash) *types.Transaction {
	if tx := b.eth.txPool.Get(hash); tx != nil {
		return tx
	}
	return nil
}

func (b *EthAPIBackend) GetTransaction(ctx context.Context, txHash common.Hash) (*types.Transaction, common.Hash, uint64, uint64, error) {
	tx, blockHash, blockNumber, index := rawdb.ReadTransaction(b.eth.ChainDb(), txHash)
	return tx, blockHash, blockNumber, index, nil
}

func (b *EthAPIBackend) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	return b.eth.txPool.PoolNonce(addr), nil
}

func (b *EthAPIBackend) Stats() (runnable int, blocked int) {
	return b.eth.txPool.Stats()
}

func (b *EthAPIBackend) TxPoolContent() (map[common.Address][]*types.Transaction, map[common.Address][]*types.Transaction) {
	return b.eth.txPool.Content()
}

func (b *EthAPIBackend) TxPoolContentFrom(addr common.Address) ([]*types.Transaction, []*types.Transaction) {
	return b.eth.txPool.ContentFrom(addr)
}

func (b *EthAPIBackend) TxPool() *txpool.TxPool {
	return b.eth.txPool
}

func (b *EthAPIBackend) SubscribeNewTxsEvent(ch chan<- core.NewTxsEvent) event.Subscription {
	return b.eth.txPool.SubscribeTransactions(ch, true)
}

func (b *EthAPIBackend) Downloader() *downloader.Downloader {
	return b.eth.Downloader()
}

func (b *EthAPIBackend) ProtocolVersion() int {
	return b.eth.EthVersion()
}

func (b *EthAPIBackend) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	return b.gpo.SuggestTipCap(ctx)
}

func (b *EthAPIBackend) FeeHistory(ctx context.Context, blockCount uint64, lastBlock rpc.BlockNumber, rewardPercentiles []float64) (firstBlock *big.Int, reward [][]*big.Int, baseFee []*big.Int, gasUsedRatio []float64, err error) {
	return b.gpo.FeeHistory(ctx, blockCount, lastBlock, rewardPercentiles)
}

func (b *EthAPIBackend) BlobBaseFee(ctx context.Context) *big.Int {
	return new(big.Int)
}

func (b *EthAPIBackend) ChainDb() ethdb.Database {
	return b.eth.ChainDb()
}

func (b *EthAPIBackend) EventMux() *event.TypeMux {
	return b.eth.EventMux()
}

func (b *EthAPIBackend) UnprotectedAllowed() bool {
	return b.allowUnprotectedTxs
}

func (b *EthAPIBackend) RPCGasCap() uint64 {
	return b.eth.config.RPCGasCap
}

func (b *EthAPIBackend) RPCEVMTimeout() time.Duration {
	return b.eth.config.RPCEVMTimeout
}

func (b *EthAPIBackend) AccountManager() *accounts.Manager {
	return b.eth.AccountManager()
}

func (b *EthAPIBackend) RPCTxFeeCap() float64 {
	return b.eth.config.RPCTxFeeCap
}

func (b *EthAPIBackend) BloomStatus() (uint64, uint64) {
	sections, _, _ := b.eth.bloomIndexer.Sections()
	return params.BloomBitsBlocks, sections
}

func (b *EthAPIBackend) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {
	for i := 0; i < bloomFilterThreads; i++ {
		go session.Multiplex(bloomRetrievalBatch, bloomRetrievalWait, b.eth.bloomRequests)
	}
}

func (b *EthAPIBackend) Engine() consensus.Engine {
	return b.eth.engine
}

func (b *EthAPIBackend) Miner() *miner.Miner {
	return b.eth.Miner()
}

func (b *EthAPIBackend) StateAtBlock(ctx context.Context, block *types.Block, reexec uint64, base *state.StateDB, readOnly bool, preferDisk bool) (*state.StateDB, tracers.StateReleaseFunc, error) {
	return b.eth.StateAtBlock(ctx, block, reexec, base, readOnly, preferDisk)
}

func (b *EthAPIBackend) StateAtTransaction(ctx context.Context, block *types.Block, txIndex int, reexec uint64) (*types.Transaction, vm.BlockContext, *state.StateDB, tracers.StateReleaseFunc, error) {
	return b.eth.stateAtTransaction(ctx, block, txIndex, reexec)
}

func (b *EthAPIBackend) GetIPCClient() (bind.ContractBackend, error) {
	client, err := b.eth.blockchain.GetClient()
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (b *EthAPIBackend) CurrentHeader() *types.Header {
	return b.eth.blockchain.CurrentHeader()
}

func (b *EthAPIBackend) GetRewardByHash(hash common.Hash) map[string]map[string]map[string]*big.Int {
	header := b.eth.blockchain.GetHeaderByHash(hash)
	if header != nil {
		data, err := os.ReadFile(filepath.Join(common.StoreRewardFolder, header.Number.String()+"."+header.Hash().Hex()))
		if err == nil {
			rewards := make(map[string]map[string]map[string]*big.Int)
			err = json.Unmarshal(data, &rewards)
			if err == nil {
				return rewards
			}
		} else {
			data, err = os.ReadFile(filepath.Join(common.StoreRewardFolder, header.Number.String()+"."+header.HashNoValidator().Hex()))
			if err == nil {
				rewards := make(map[string]map[string]map[string]*big.Int)
				err = json.Unmarshal(data, &rewards)
				if err == nil {
					return rewards
				}
			}
		}
	}
	return make(map[string]map[string]map[string]*big.Int)
}

// GetVotersRewards return a map of voters of snapshot at given block hash
// there is a function engine.HookReward nearly does the same thing but
// it does change the stateDB too - so can't use it here
// Steps:
// 1. Checking back to state of last checkpoint
// 2. Get list signers + reward at that checkpoint
// 3. Find out the list signers_reward for input masternode's reward
// 4. Calculate voters's rewards for input masternode
func (b *EthAPIBackend) GetVotersRewards(masternodeAddr common.Address) map[common.Address]*big.Int {
	chain := b.eth.blockchain
	header := chain.CurrentBlock()
	number := header.Number.Uint64()
	engine := b.Engine().(*XDPoS.XDPoS)
	foundationWalletAddr := chain.Config().XDPoS.FoundationWalletAddr

	// calculate for 2 epochs ago
	currentCheckpointNumber, _, err := engine.GetCurrentEpochSwitchBlock(chain, header.Number)
	if err != nil {
		log.Error("[GetVotersRewards] Fail to get GetCurrentEpochSwitchBlock for current checkpoint block", "block", header.Number, "err", err)
	}
	lastCheckpointNumber, _, err := engine.GetCurrentEpochSwitchBlock(chain, big.NewInt(int64(currentCheckpointNumber-1)))
	if err != nil {
		log.Error("[GetVotersRewards] Fail to get GetCurrentEpochSwitchBlock for last checkpoint block", "block", header.Number, "err", err)
	}

	lastCheckpointBlock := chain.GetBlockByNumber(lastCheckpointNumber)
	rCheckpoint := chain.Config().XDPoS.RewardCheckpoint

	state, err := chain.StateAt(lastCheckpointBlock.Root())
	if err != nil {
		log.Error("fail to get state in GetVotersRewards", "lastCheckpointNumber", lastCheckpointNumber, "err", err)
		return nil
	}
	if state == nil {
		log.Error("fail to get state in GetVotersRewards", "lastCheckpointNumber", lastCheckpointNumber)
		return nil
	}

	if foundationWalletAddr == (common.Address{}) {
		log.Error("Foundation Wallet Address is empty", "error", foundationWalletAddr)
		return nil
	}

	if lastCheckpointNumber <= 0 || lastCheckpointNumber-rCheckpoint <= 0 || foundationWalletAddr == (common.Address{}) {
		return nil
	}

	// Get signers in blockSigner smartcontract.
	// Get reward inflation.
	chainReward := new(big.Int).Mul(new(big.Int).SetUint64(chain.Config().XDPoS.Reward), new(big.Int).SetUint64(params.Ether))
	chainReward = util.RewardInflation(chain, chainReward, number, common.BlocksPerYear)
	totalSigner := new(uint64)
	signers, err := contracts.GetRewardForCheckpoint(engine, chain, lastCheckpointBlock.Header(), rCheckpoint, totalSigner)

	if err != nil {
		log.Error("Fail to get signers for reward checkpoint", "error", err)
		return nil
	}

	rewardSigners, err := contracts.CalculateRewardForSigner(chainReward, signers, *totalSigner)
	if err != nil {
		log.Error("Fail to calculate reward for signers", "error", err)
		return nil
	}

	if len(signers) <= 0 {
		return nil
	}

	// Add reward for coin voters of input masternode.
	var voterResults map[common.Address]*big.Int
	for signer, calcReward := range rewardSigners {
		if signer == masternodeAddr {
			rewards, err := contracts.CalculateRewardForHolders(foundationWalletAddr, state, masternodeAddr, calcReward, number)
			if err != nil {
				log.Error("Fail to calculate reward for holders.", "error", err)
				return nil
			}
			voterResults = rewards
			break
		}
	}
	return voterResults
}

// GetVotersCap return all voters's capability at a checkpoint
func (b *EthAPIBackend) GetVotersCap(checkpoint *big.Int, masterAddr common.Address, voters []common.Address) map[common.Address]*big.Int {
	chain := b.eth.blockchain
	checkpointBlock := chain.GetBlockByNumber(checkpoint.Uint64())
	statedb, err := chain.StateAt(checkpointBlock.Root())

	if err != nil {
		log.Error("fail to get state in GetVotersCap", "checkpoint", checkpoint, "err", err)
		return nil
	}
	if statedb == nil {
		log.Error("fail to get state in GetVotersCap", "checkpoint", checkpoint)
		return nil
	}

	voterCaps := make(map[common.Address]*big.Int)
	for _, voteAddr := range voters {
		voterCap := statedb.GetVoterCap(masterAddr, voteAddr)
		voterCaps[voteAddr] = voterCap
	}
	return voterCaps
}

// GetEpochDuration return latest generating velocity epoch by minute
// ie 30min for each epoch
func (b *EthAPIBackend) GetEpochDuration() *big.Int {
	chain := b.eth.blockchain
	header := chain.CurrentBlock()
	number := header.Number.Uint64()
	lastCheckpointNumber := number - (number % b.ChainConfig().XDPoS.Epoch)
	lastCheckpointBlockTime := chain.GetBlockByNumber(lastCheckpointNumber).Time()
	secondToLastCheckpointNumber := lastCheckpointNumber - b.ChainConfig().XDPoS.Epoch
	secondToLastCheckpointBlockTime := chain.GetBlockByNumber(secondToLastCheckpointNumber).Time()

	return new(big.Int).SetInt64(int64(secondToLastCheckpointBlockTime) - int64(lastCheckpointBlockTime))
}

// GetMasternodesCap return a cap of all masternode at a checkpoint
func (b *EthAPIBackend) GetMasternodesCap(checkpoint uint64) map[common.Address]*big.Int {
	checkpointBlock := b.eth.blockchain.GetBlockByNumber(checkpoint)
	statedb, err := b.eth.blockchain.StateAt(checkpointBlock.Root())
	if err != nil {
		log.Error("fail to get state in GetMasternodesCap", "checkpoint", checkpoint, "err", err)
		return nil
	}
	if statedb == nil {
		log.Error("fail to get state in GetMasternodesCap", "checkpoint", checkpoint)
		return nil
	}

	candicates := statedb.GetCandidates()

	masternodesCap := map[common.Address]*big.Int{}
	for _, candicate := range candicates {
		masternodesCap[candicate] = statedb.GetCandidateCap(candicate)
	}

	return masternodesCap
}

func (b *EthAPIBackend) GetBlocksHashCache(blockNr uint64) []common.Hash {
	return b.eth.blockchain.GetBlocksHashCache(blockNr)
}

func (b *EthAPIBackend) AreTwoBlockSamePath(bh1 common.Hash, bh2 common.Hash) bool {
	return b.eth.blockchain.AreTwoBlockSamePath(bh1, bh2)
}

func (b *EthAPIBackend) XDCxService() *XDCx.XDCX {
	return b.eth.XDCX
}

func (b *EthAPIBackend) LendingService() *XDCxlending.Lending {
	return b.eth.Lending
}

func (b *EthAPIBackend) GetPeer() int {
	return b.eth.protocolManager.peers.Len()
}

// ValidateMasternode checks if node's address is in set of masternodes
func (b *EthAPIBackend) ValidateMasternode() (bool, error) {
	return b.eth.ValidateMasternode()
}

func (b *EthAPIBackend) StartStaking(local bool) error {
	return b.eth.StartStaking(local)
}

func (b *EthAPIBackend) StopStaking() {
	b.eth.StopStaking()
}

func (b *EthAPIBackend) IsStaking() bool {
	return b.eth.IsStaking()
}

func (b *EthAPIBackend) BlockChain() *core.BlockChain {
	return b.eth.blockchain
}
