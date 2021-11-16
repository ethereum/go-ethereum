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
	"io/ioutil"
	"math/big"
	"path/filepath"

	"github.com/XinFinOrg/XDPoSChain/XDCx/tradingstate"
	"github.com/XinFinOrg/XDPoSChain/XDCxlending"
	"github.com/XinFinOrg/XDPoSChain/accounts/abi/bind"

	"github.com/XinFinOrg/XDPoSChain/XDCx"

	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"

	"github.com/XinFinOrg/XDPoSChain/accounts"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/common/math"
	"github.com/XinFinOrg/XDPoSChain/consensus"
	"github.com/XinFinOrg/XDPoSChain/contracts"
	"github.com/XinFinOrg/XDPoSChain/core"
	"github.com/XinFinOrg/XDPoSChain/core/bloombits"
	"github.com/XinFinOrg/XDPoSChain/core/state"
	stateDatabase "github.com/XinFinOrg/XDPoSChain/core/state"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/core/vm"
	"github.com/XinFinOrg/XDPoSChain/eth/downloader"
	"github.com/XinFinOrg/XDPoSChain/eth/gasprice"
	"github.com/XinFinOrg/XDPoSChain/eth/util"
	"github.com/XinFinOrg/XDPoSChain/ethdb"
	"github.com/XinFinOrg/XDPoSChain/event"
	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/XinFinOrg/XDPoSChain/rpc"
)

// EthApiBackend implements ethapi.Backend for full nodes
type EthApiBackend struct {
	eth *Ethereum
	gpo *gasprice.Oracle
}

func (b *EthApiBackend) ChainConfig() *params.ChainConfig {
	return b.eth.chainConfig
}

func (b *EthApiBackend) CurrentBlock() *types.Block {
	return b.eth.blockchain.CurrentBlock()
}

func (b *EthApiBackend) SetHead(number uint64) {
	b.eth.protocolManager.downloader.Cancel()
	b.eth.blockchain.SetHead(number)
}

func (b *EthApiBackend) HeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Header, error) {
	// Pending block is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		blockNr = rpc.LatestBlockNumber
	}
	// Otherwise resolve and return the block
	if blockNr == rpc.LatestBlockNumber {
		return b.eth.blockchain.CurrentBlock().Header(), nil
	}
	return b.eth.blockchain.GetHeaderByNumber(uint64(blockNr)), nil
}

func (b *EthApiBackend) BlockByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Block, error) {
	// Pending block is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		blockNr = rpc.LatestBlockNumber
	}
	// Otherwise resolve and return the block
	if blockNr == rpc.LatestBlockNumber {
		return b.eth.blockchain.CurrentBlock(), nil
	}
	return b.eth.blockchain.GetBlockByNumber(uint64(blockNr)), nil
}

func (b *EthApiBackend) StateAndHeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*state.StateDB, *types.Header, error) {
	// Pending state is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		blockNr = rpc.LatestBlockNumber
	}
	// Otherwise resolve the block number and return its state
	header, err := b.HeaderByNumber(ctx, blockNr)
	if header == nil || err != nil {
		return nil, nil, err
	}
	stateDb, err := b.eth.BlockChain().StateAt(header.Root)
	return stateDb, header, err
}

func (b *EthApiBackend) GetBlock(ctx context.Context, blockHash common.Hash) (*types.Block, error) {
	return b.eth.blockchain.GetBlockByHash(blockHash), nil
}

func (b *EthApiBackend) GetReceipts(ctx context.Context, blockHash common.Hash) (types.Receipts, error) {
	return core.GetBlockReceipts(b.eth.chainDb, blockHash, core.GetBlockNumber(b.eth.chainDb, blockHash)), nil
}

func (b *EthApiBackend) GetLogs(ctx context.Context, blockHash common.Hash) ([][]*types.Log, error) {
	receipts := core.GetBlockReceipts(b.eth.chainDb, blockHash, core.GetBlockNumber(b.eth.chainDb, blockHash))
	if receipts == nil {
		return nil, nil
	}
	logs := make([][]*types.Log, len(receipts))
	for i, receipt := range receipts {
		logs[i] = receipt.Logs
	}
	return logs, nil
}

func (b *EthApiBackend) GetTd(blockHash common.Hash) *big.Int {
	return b.eth.blockchain.GetTdByHash(blockHash)
}

func (b *EthApiBackend) GetEVM(ctx context.Context, msg core.Message, state *state.StateDB, XDCxState *tradingstate.TradingStateDB, header *types.Header, vmCfg vm.Config) (*vm.EVM, func() error, error) {
	state.SetBalance(msg.From(), math.MaxBig256)
	vmError := func() error { return nil }

	context := core.NewEVMContext(msg, header, b.eth.BlockChain(), nil)
	return vm.NewEVM(context, state, XDCxState, b.eth.chainConfig, vmCfg), vmError, nil
}

func (b *EthApiBackend) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
	return b.eth.BlockChain().SubscribeRemovedLogsEvent(ch)
}

func (b *EthApiBackend) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	return b.eth.BlockChain().SubscribeChainEvent(ch)
}

func (b *EthApiBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return b.eth.BlockChain().SubscribeChainHeadEvent(ch)
}

func (b *EthApiBackend) SubscribeChainSideEvent(ch chan<- core.ChainSideEvent) event.Subscription {
	return b.eth.BlockChain().SubscribeChainSideEvent(ch)
}

func (b *EthApiBackend) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.eth.BlockChain().SubscribeLogsEvent(ch)
}

func (b *EthApiBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	return b.eth.txPool.AddLocal(signedTx)
}

// SendOrderTx send order via backend
func (b *EthApiBackend) SendOrderTx(ctx context.Context, signedTx *types.OrderTransaction) error {
	return b.eth.orderPool.AddLocal(signedTx)
}

// SendLendingTx send order via backend
func (b *EthApiBackend) SendLendingTx(ctx context.Context, signedTx *types.LendingTransaction) error {
	return b.eth.lendingPool.AddLocal(signedTx)
}

func (b *EthApiBackend) GetPoolTransactions() (types.Transactions, error) {
	pending, err := b.eth.txPool.Pending()
	if err != nil {
		return nil, err
	}
	var txs types.Transactions
	for _, batch := range pending {
		txs = append(txs, batch...)
	}
	return txs, nil
}

func (b *EthApiBackend) GetPoolTransaction(hash common.Hash) *types.Transaction {
	return b.eth.txPool.Get(hash)
}

func (b *EthApiBackend) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	return b.eth.txPool.State().GetNonce(addr), nil
}

func (b *EthApiBackend) Stats() (pending int, queued int) {
	return b.eth.txPool.Stats()
}

func (b *EthApiBackend) TxPoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	return b.eth.TxPool().Content()
}

func (b *EthApiBackend) OrderTxPoolContent() (map[common.Address]types.OrderTransactions, map[common.Address]types.OrderTransactions) {
	return b.eth.OrderPool().Content()
}
func (b *EthApiBackend) OrderStats() (pending int, queued int) {
	return b.eth.txPool.Stats()
}

func (b *EthApiBackend) SubscribeTxPreEvent(ch chan<- core.TxPreEvent) event.Subscription {
	return b.eth.TxPool().SubscribeTxPreEvent(ch)
}

func (b *EthApiBackend) Downloader() *downloader.Downloader {
	return b.eth.Downloader()
}

func (b *EthApiBackend) ProtocolVersion() int {
	return b.eth.EthVersion()
}

func (b *EthApiBackend) SuggestPrice(ctx context.Context) (*big.Int, error) {
	return b.gpo.SuggestPrice(ctx)
}

func (b *EthApiBackend) ChainDb() ethdb.Database {
	return b.eth.ChainDb()
}

func (b *EthApiBackend) EventMux() *event.TypeMux {
	return b.eth.EventMux()
}

func (b *EthApiBackend) AccountManager() *accounts.Manager {
	return b.eth.AccountManager()
}

func (b *EthApiBackend) BloomStatus() (uint64, uint64) {
	sections, _, _ := b.eth.bloomIndexer.Sections()
	return params.BloomBitsBlocks, sections
}

func (b *EthApiBackend) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {
	for i := 0; i < bloomFilterThreads; i++ {
		go session.Multiplex(bloomRetrievalBatch, bloomRetrievalWait, b.eth.bloomRequests)
	}
}

func (b *EthApiBackend) GetIPCClient() (bind.ContractBackend, error) {
	// func (b *EthApiBackend) GetIPCClient() (*ethclient.Client, error) {
	client, err := b.eth.blockchain.GetClient()
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (b *EthApiBackend) GetEngine() consensus.Engine {
	return b.eth.engine
}

func (s *EthApiBackend) GetRewardByHash(hash common.Hash) map[string]map[string]map[string]*big.Int {
	header := s.eth.blockchain.GetHeaderByHash(hash)
	if header != nil {
		data, err := ioutil.ReadFile(filepath.Join(common.StoreRewardFolder, header.Number.String()+"."+header.Hash().Hex()))
		if err == nil {
			rewards := make(map[string]map[string]map[string]*big.Int)
			err = json.Unmarshal(data, &rewards)
			if err == nil {
				return rewards
			}
		} else {
			data, err = ioutil.ReadFile(filepath.Join(common.StoreRewardFolder, header.Number.String()+"."+header.HashNoValidator().Hex()))
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
func (b *EthApiBackend) GetVotersRewards(masternodeAddr common.Address) map[common.Address]*big.Int {
	chain := b.eth.blockchain
	block := chain.CurrentBlock()
	number := block.Number().Uint64()
	engine := b.GetEngine().(*XDPoS.XDPoS)
	foundationWalletAddr := chain.Config().XDPoS.FoudationWalletAddr
	lastCheckpointNumber := number - (number % b.ChainConfig().XDPoS.Epoch) - b.ChainConfig().XDPoS.Epoch // calculate for 2 epochs ago
	lastCheckpointBlock := chain.GetBlockByNumber(lastCheckpointNumber)
	rCheckpoint := chain.Config().XDPoS.RewardCheckpoint

	state, err := chain.StateAt(lastCheckpointBlock.Root())
	if err != nil {
		fmt.Println("ERROR Trying to getting state at", lastCheckpointNumber, " Error ", err)
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
		log.Crit("Fail to get signers for reward checkpoint", "error", err)
		return nil
	}

	rewardSigners, err := contracts.CalculateRewardForSigner(chainReward, signers, *totalSigner)
	if err != nil {
		log.Crit("Fail to calculate reward for signers", "error", err)
		return nil
	}

	if len(signers) <= 0 {
		return nil
	}

	// Add reward for coin voters of input masternode.
	var voterResults map[common.Address]*big.Int
	for signer, calcReward := range rewardSigners {
		if signer == masternodeAddr {
			err, rewards := contracts.CalculateRewardForHolders(foundationWalletAddr, state, masternodeAddr, calcReward, number)
			if err != nil {
				log.Crit("Fail to calculate reward for holders.", "error", err)
				return nil
			}
			voterResults = rewards
			break
		}
	}

	return voterResults

}

// GetVotersCap return all voters's capability at a checkpoint
func (b *EthApiBackend) GetVotersCap(checkpoint *big.Int, masterAddr common.Address, voters []common.Address) map[common.Address]*big.Int {
	chain := b.eth.blockchain
	checkpointBlock := chain.GetBlockByNumber(checkpoint.Uint64())
	state, err := chain.StateAt(checkpointBlock.Root())

	if err != nil {
		fmt.Println("ERROR Trying to getting state at", checkpoint, " Error ", err)
		return nil
	}

	voterCaps := make(map[common.Address]*big.Int)
	for _, voteAddr := range voters {
		voterCap := stateDatabase.GetVoterCap(state, masterAddr, voteAddr)
		voterCaps[voteAddr] = voterCap
	}
	return voterCaps
}

// GetEpochDuration return latest generating velocity epoch by minute
// ie 30min for each epoch
func (b *EthApiBackend) GetEpochDuration() *big.Int {
	chain := b.eth.blockchain
	block := chain.CurrentBlock()
	number := block.Number().Uint64()
	lastCheckpointNumber := number - (number % b.ChainConfig().XDPoS.Epoch)
	lastCheckpointBlockTime := chain.GetBlockByNumber(lastCheckpointNumber).Time()
	secondToLastCheckpointNumber := lastCheckpointNumber - b.ChainConfig().XDPoS.Epoch
	secondToLastCheckpointBlockTime := chain.GetBlockByNumber(secondToLastCheckpointNumber).Time()

	return secondToLastCheckpointBlockTime.Add(secondToLastCheckpointBlockTime, lastCheckpointBlockTime.Mul(lastCheckpointBlockTime, new(big.Int).SetInt64(-1)))
}

// GetMasternodesCap return a cap of all masternode at a checkpoint
func (b *EthApiBackend) GetMasternodesCap(checkpoint uint64) map[common.Address]*big.Int {
	checkpointBlock := b.eth.blockchain.GetBlockByNumber(checkpoint)
	state, err := b.eth.blockchain.StateAt(checkpointBlock.Root())

	if err != nil {
		fmt.Println("ERROR Trying to getting state at", checkpoint, " Error ", err)
		return nil
	}

	candicates := stateDatabase.GetCandidates(state)

	masternodesCap := map[common.Address]*big.Int{}
	for _, candicate := range candicates {
		masternodesCap[candicate] = stateDatabase.GetCandidateCap(state, candicate)
	}

	return masternodesCap
}

func (b *EthApiBackend) GetBlocksHashCache(blockNr uint64) []common.Hash {
	return b.eth.blockchain.GetBlocksHashCache(blockNr)
}

func (b *EthApiBackend) AreTwoBlockSamePath(bh1 common.Hash, bh2 common.Hash) bool {
	return b.eth.blockchain.AreTwoBlockSamePath(bh1, bh2)
}

// GetOrderNonce get order nonce
func (b *EthApiBackend) GetOrderNonce(address common.Hash) (uint64, error) {
	XDCxService := b.eth.GetXDCX()
	if XDCxService != nil {
		author, err := b.GetEngine().Author(b.CurrentBlock().Header())
		if err != nil {
			return 0, err
		}
		XDCxState, err := XDCxService.GetTradingState(b.CurrentBlock(), author)
		if err != nil {
			return 0, err
		}
		return XDCxState.GetNonce(address), nil
	}
	return 0, errors.New("cannot find XDCx service")
}

func (b *EthApiBackend) XDCxService() *XDCx.XDCX {
	return b.eth.XDCX
}

func (b *EthApiBackend) LendingService() *XDCxlending.Lending {
	return b.eth.Lending
}
