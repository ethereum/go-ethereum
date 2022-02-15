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

// Copyright 2021-2022 The go-xpayments Authors
// This file is part of go-xpayments.

package xps

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/xpaymentsorg/go-xpayments"
	"github.com/xpaymentsorg/go-xpayments/accounts"
	"github.com/xpaymentsorg/go-xpayments/common"
	"github.com/xpaymentsorg/go-xpayments/consensus"
	"github.com/xpaymentsorg/go-xpayments/core"
	"github.com/xpaymentsorg/go-xpayments/core/bloombits"
	"github.com/xpaymentsorg/go-xpayments/core/rawdb"
	"github.com/xpaymentsorg/go-xpayments/core/state"
	"github.com/xpaymentsorg/go-xpayments/core/types"
	"github.com/xpaymentsorg/go-xpayments/core/vm"
	"github.com/xpaymentsorg/go-xpayments/event"
	"github.com/xpaymentsorg/go-xpayments/miner"
	"github.com/xpaymentsorg/go-xpayments/params"
	"github.com/xpaymentsorg/go-xpayments/rpc"
	"github.com/xpaymentsorg/go-xpayments/xps/gasprice"
	"github.com/xpaymentsorg/go-xpayments/xpsdb"
	// "github.com/ethereum/go-ethereum"
	// "github.com/ethereum/go-ethereum/accounts"
	// "github.com/ethereum/go-ethereum/common"
	// "github.com/ethereum/go-ethereum/consensus"
	// "github.com/ethereum/go-ethereum/core"
	// "github.com/ethereum/go-ethereum/core/bloombits"
	// "github.com/ethereum/go-ethereum/core/rawdb"
	// "github.com/ethereum/go-ethereum/core/state"
	// "github.com/ethereum/go-ethereum/core/types"
	// "github.com/ethereum/go-ethereum/core/vm"
	// "github.com/ethereum/go-ethereum/eth/gasprice"
	// "github.com/ethereum/go-ethereum/ethdb"
	// "github.com/ethereum/go-ethereum/event"
	// "github.com/ethereum/go-ethereum/miner"
	// "github.com/ethereum/go-ethereum/params"
	// "github.com/ethereum/go-ethereum/rpc"
)

// XpsAPIBackend implements xpsapi.Backend for full nodes
type XpsAPIBackend struct {
	extRPCEnabled       bool
	allowUnprotectedTxs bool
	xps                 *xPayments
	gpo                 *gasprice.Oracle
}

// ChainConfig returns the active chain configuration.
func (b *XpsAPIBackend) ChainConfig() *params.ChainConfig {
	return b.xps.blockchain.Config()
}

func (b *XpsAPIBackend) CurrentBlock() *types.Block {
	return b.xps.blockchain.CurrentBlock()
}

func (b *XpsAPIBackend) SetHead(number uint64) {
	b.xps.handler.downloader.Cancel()
	b.xps.blockchain.SetHead(number)
}

func (b *XpsAPIBackend) HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Header, error) {
	// Pending block is only known by the miner
	if number == rpc.PendingBlockNumber {
		block := b.xps.miner.PendingBlock()
		return block.Header(), nil
	}
	// Otherwise resolve and return the block
	if number == rpc.LatestBlockNumber {
		return b.xps.blockchain.CurrentBlock().Header(), nil
	}
	return b.xps.blockchain.GetHeaderByNumber(uint64(number)), nil
}

func (b *XpsAPIBackend) HeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Header, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return b.HeaderByNumber(ctx, blockNr)
	}
	if hash, ok := blockNrOrHash.Hash(); ok {
		header := b.xps.blockchain.GetHeaderByHash(hash)
		if header == nil {
			return nil, errors.New("header for hash not found")
		}
		if blockNrOrHash.RequireCanonical && b.xps.blockchain.GetCanonicalHash(header.Number.Uint64()) != hash {
			return nil, errors.New("hash is not currently canonical")
		}
		return header, nil
	}
	return nil, errors.New("invalid arguments; neither block nor hash specified")
}

func (b *XpsAPIBackend) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	return b.xps.blockchain.GetHeaderByHash(hash), nil
}

func (b *XpsAPIBackend) BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error) {
	// Pending block is only known by the miner
	if number == rpc.PendingBlockNumber {
		block := b.xps.miner.PendingBlock()
		return block, nil
	}
	// Otherwise resolve and return the block
	if number == rpc.LatestBlockNumber {
		return b.xps.blockchain.CurrentBlock(), nil
	}
	return b.xps.blockchain.GetBlockByNumber(uint64(number)), nil
}

func (b *XpsAPIBackend) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return b.xps.blockchain.GetBlockByHash(hash), nil
}

func (b *XpsAPIBackend) BlockByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Block, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return b.BlockByNumber(ctx, blockNr)
	}
	if hash, ok := blockNrOrHash.Hash(); ok {
		header := b.xps.blockchain.GetHeaderByHash(hash)
		if header == nil {
			return nil, errors.New("header for hash not found")
		}
		if blockNrOrHash.RequireCanonical && b.xps.blockchain.GetCanonicalHash(header.Number.Uint64()) != hash {
			return nil, errors.New("hash is not currently canonical")
		}
		block := b.xps.blockchain.GetBlock(hash, header.Number.Uint64())
		if block == nil {
			return nil, errors.New("header found, but block body is missing")
		}
		return block, nil
	}
	return nil, errors.New("invalid arguments; neither block nor hash specified")
}

func (b *XpsAPIBackend) PendingBlockAndReceipts() (*types.Block, types.Receipts) {
	return b.xps.miner.PendingBlockAndReceipts()
}

func (b *XpsAPIBackend) StateAndHeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*state.StateDB, *types.Header, error) {
	// Pending state is only known by the miner
	if number == rpc.PendingBlockNumber {
		block, state := b.xps.miner.Pending()
		return state, block.Header(), nil
	}
	// Otherwise resolve the block number and return its state
	header, err := b.HeaderByNumber(ctx, number)
	if err != nil {
		return nil, nil, err
	}
	if header == nil {
		return nil, nil, errors.New("header not found")
	}
	stateDb, err := b.xps.BlockChain().StateAt(header.Root)
	return stateDb, header, err
}

func (b *XpsAPIBackend) StateAndHeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*state.StateDB, *types.Header, error) {
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
		if blockNrOrHash.RequireCanonical && b.xps.blockchain.GetCanonicalHash(header.Number.Uint64()) != hash {
			return nil, nil, errors.New("hash is not currently canonical")
		}
		stateDb, err := b.xps.BlockChain().StateAt(header.Root)
		return stateDb, header, err
	}
	return nil, nil, errors.New("invalid arguments; neither block nor hash specified")
}

func (b *XpsAPIBackend) GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error) {
	return b.xps.blockchain.GetReceiptsByHash(hash), nil
}

func (b *XpsAPIBackend) GetLogs(ctx context.Context, hash common.Hash) ([][]*types.Log, error) {
	db := b.xps.ChainDb()
	number := rawdb.ReadHeaderNumber(db, hash)
	if number == nil {
		return nil, errors.New("failed to get block number from hash")
	}
	logs := rawdb.ReadLogs(db, hash, *number, b.xps.blockchain.Config())
	if logs == nil {
		return nil, errors.New("failed to get logs for block")
	}
	return logs, nil
}

func (b *XpsAPIBackend) GetTd(ctx context.Context, hash common.Hash) *big.Int {
	if header := b.xps.blockchain.GetHeaderByHash(hash); header != nil {
		return b.xps.blockchain.GetTd(hash, header.Number.Uint64())
	}
	return nil
}

func (b *XpsAPIBackend) GetEVM(ctx context.Context, msg core.Message, state *state.StateDB, header *types.Header, vmConfig *vm.Config) (*vm.EVM, func() error, error) {
	vmError := func() error { return nil }
	if vmConfig == nil {
		vmConfig = b.xps.blockchain.GetVMConfig()
	}
	txContext := core.NewEVMTxContext(msg)
	context := core.NewEVMBlockContext(header, b.xps.BlockChain(), nil)
	return vm.NewEVM(context, txContext, state, b.xps.blockchain.Config(), *vmConfig), vmError, nil
}

func (b *XpsAPIBackend) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
	return b.xps.BlockChain().SubscribeRemovedLogsEvent(ch)
}

func (b *XpsAPIBackend) SubscribePendingLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.xps.miner.SubscribePendingLogs(ch)
}

func (b *XpsAPIBackend) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	return b.xps.BlockChain().SubscribeChainEvent(ch)
}

func (b *XpsAPIBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return b.xps.BlockChain().SubscribeChainHeadEvent(ch)
}

func (b *XpsAPIBackend) SubscribeChainSideEvent(ch chan<- core.ChainSideEvent) event.Subscription {
	return b.xps.BlockChain().SubscribeChainSideEvent(ch)
}

func (b *XpsAPIBackend) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.xps.BlockChain().SubscribeLogsEvent(ch)
}

func (b *XpsAPIBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	return b.xps.txPool.AddLocal(signedTx)
}

func (b *XpsAPIBackend) GetPoolTransactions() (types.Transactions, error) {
	pending := b.xps.txPool.Pending(false)
	var txs types.Transactions
	for _, batch := range pending {
		txs = append(txs, batch...)
	}
	return txs, nil
}

func (b *XpsAPIBackend) GetPoolTransaction(hash common.Hash) *types.Transaction {
	return b.xps.txPool.Get(hash)
}

func (b *XpsAPIBackend) GetTransaction(ctx context.Context, txHash common.Hash) (*types.Transaction, common.Hash, uint64, uint64, error) {
	tx, blockHash, blockNumber, index := rawdb.ReadTransaction(b.xps.ChainDb(), txHash)
	return tx, blockHash, blockNumber, index, nil
}

func (b *XpsAPIBackend) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	return b.xps.txPool.Nonce(addr), nil
}

func (b *XpsAPIBackend) Stats() (pending int, queued int) {
	return b.xps.txPool.Stats()
}

func (b *XpsAPIBackend) TxPoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	return b.xps.TxPool().Content()
}

func (b *XpsAPIBackend) TxPoolContentFrom(addr common.Address) (types.Transactions, types.Transactions) {
	return b.xps.TxPool().ContentFrom(addr)
}

func (b *XpsAPIBackend) TxPool() *core.TxPool {
	return b.xps.TxPool()
}

func (b *XpsAPIBackend) SubscribeNewTxsEvent(ch chan<- core.NewTxsEvent) event.Subscription {
	return b.xps.TxPool().SubscribeNewTxsEvent(ch)
}

func (b *XpsAPIBackend) SyncProgress() xpayments.SyncProgress {
	return b.xps.Downloader().Progress()
}

func (b *XpsAPIBackend) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	return b.gpo.SuggestTipCap(ctx)
}

func (b *XpsAPIBackend) FeeHistory(ctx context.Context, blockCount int, lastBlock rpc.BlockNumber, rewardPercentiles []float64) (firstBlock *big.Int, reward [][]*big.Int, baseFee []*big.Int, gasUsedRatio []float64, err error) {
	return b.gpo.FeeHistory(ctx, blockCount, lastBlock, rewardPercentiles)
}

func (b *XpsAPIBackend) ChainDb() xpsdb.Database {
	return b.xps.ChainDb()
}

func (b *XpsAPIBackend) EventMux() *event.TypeMux {
	return b.xps.EventMux()
}

func (b *XpsAPIBackend) AccountManager() *accounts.Manager {
	return b.xps.AccountManager()
}

func (b *XpsAPIBackend) ExtRPCEnabled() bool {
	return b.extRPCEnabled
}

func (b *XpsAPIBackend) UnprotectedAllowed() bool {
	return b.allowUnprotectedTxs
}

func (b *XpsAPIBackend) RPCGasCap() uint64 {
	return b.xps.config.RPCGasCap
}

func (b *XpsAPIBackend) RPCEVMTimeout() time.Duration {
	return b.xps.config.RPCEVMTimeout
}

func (b *XpsAPIBackend) RPCTxFeeCap() float64 {
	return b.xps.config.RPCTxFeeCap
}

func (b *XpsAPIBackend) BloomStatus() (uint64, uint64) {
	sections, _, _ := b.xps.bloomIndexer.Sections()
	return params.BloomBitsBlocks, sections
}

func (b *XpsAPIBackend) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {
	for i := 0; i < bloomFilterThreads; i++ {
		go session.Multiplex(bloomRetrievalBatch, bloomRetrievalWait, b.xps.bloomRequests)
	}
}

func (b *XpsAPIBackend) Engine() consensus.Engine {
	return b.xps.engine
}

func (b *XpsAPIBackend) CurrentHeader() *types.Header {
	return b.xps.blockchain.CurrentHeader()
}

func (b *XpsAPIBackend) Miner() *miner.Miner {
	return b.xps.Miner()
}

func (b *XpsAPIBackend) StartMining(threads int) error {
	return b.xps.StartMining(threads)
}

func (b *XpsAPIBackend) StateAtBlock(ctx context.Context, block *types.Block, reexec uint64, base *state.StateDB, checkLive, preferDisk bool) (*state.StateDB, error) {
	return b.xps.StateAtBlock(block, reexec, base, checkLive, preferDisk)
}

func (b *XpsAPIBackend) StateAtTransaction(ctx context.Context, block *types.Block, txIndex int, reexec uint64) (core.Message, vm.BlockContext, *state.StateDB, error) {
	return b.xps.stateAtTransaction(block, txIndex, reexec)
}
