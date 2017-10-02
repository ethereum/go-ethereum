// Copyright 2015 The go-burnout Authors
// This file is part of the go-burnout library.
//
// The go-burnout library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-burnout library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-burnout library. If not, see <http://www.gnu.org/licenses/>.

package brn

import (
	"context"
	"math/big"

	"github.com/burnoutcoin/go-burnout/accounts"
	"github.com/burnoutcoin/go-burnout/common"
	"github.com/burnoutcoin/go-burnout/common/math"
	"github.com/burnoutcoin/go-burnout/core"
	"github.com/burnoutcoin/go-burnout/core/bloombits"
	"github.com/burnoutcoin/go-burnout/core/state"
	"github.com/burnoutcoin/go-burnout/core/types"
	"github.com/burnoutcoin/go-burnout/core/vm"
	"github.com/burnoutcoin/go-burnout/brn/downloader"
	"github.com/burnoutcoin/go-burnout/brn/gasprice"
	"github.com/burnoutcoin/go-burnout/brndb"
	"github.com/burnoutcoin/go-burnout/event"
	"github.com/burnoutcoin/go-burnout/params"
	"github.com/burnoutcoin/go-burnout/rpc"
)

// EthApiBackend implements ethapi.Backend for full nodes
type EthApiBackend struct {
	brn *Burnout
	gpo *gasprice.Oracle
}

func (b *EthApiBackend) ChainConfig() *params.ChainConfig {
	return b.brn.chainConfig
}

func (b *EthApiBackend) CurrentBlock() *types.Block {
	return b.brn.blockchain.CurrentBlock()
}

func (b *EthApiBackend) SetHead(number uint64) {
	b.brn.protocolManager.downloader.Cancel()
	b.brn.blockchain.SetHead(number)
}

func (b *EthApiBackend) HeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Header, error) {
	// Pending block is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block := b.brn.miner.PendingBlock()
		return block.Header(), nil
	}
	// Otherwise resolve and return the block
	if blockNr == rpc.LatestBlockNumber {
		return b.brn.blockchain.CurrentBlock().Header(), nil
	}
	return b.brn.blockchain.GetHeaderByNumber(uint64(blockNr)), nil
}

func (b *EthApiBackend) BlockByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Block, error) {
	// Pending block is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block := b.brn.miner.PendingBlock()
		return block, nil
	}
	// Otherwise resolve and return the block
	if blockNr == rpc.LatestBlockNumber {
		return b.brn.blockchain.CurrentBlock(), nil
	}
	return b.brn.blockchain.GetBlockByNumber(uint64(blockNr)), nil
}

func (b *EthApiBackend) StateAndHeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*state.StateDB, *types.Header, error) {
	// Pending state is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block, state := b.brn.miner.Pending()
		return state, block.Header(), nil
	}
	// Otherwise resolve the block number and return its state
	header, err := b.HeaderByNumber(ctx, blockNr)
	if header == nil || err != nil {
		return nil, nil, err
	}
	stateDb, err := b.brn.BlockChain().StateAt(header.Root)
	return stateDb, header, err
}

func (b *EthApiBackend) GetBlock(ctx context.Context, blockHash common.Hash) (*types.Block, error) {
	return b.brn.blockchain.GetBlockByHash(blockHash), nil
}

func (b *EthApiBackend) GetReceipts(ctx context.Context, blockHash common.Hash) (types.Receipts, error) {
	return core.GetBlockReceipts(b.brn.chainDb, blockHash, core.GetBlockNumber(b.brn.chainDb, blockHash)), nil
}

func (b *EthApiBackend) GetTd(blockHash common.Hash) *big.Int {
	return b.brn.blockchain.GetTdByHash(blockHash)
}

func (b *EthApiBackend) GetEVM(ctx context.Context, msg core.Message, state *state.StateDB, header *types.Header, vmCfg vm.Config) (*vm.EVM, func() error, error) {
	state.SetBalance(msg.From(), math.MaxBig256)
	vmError := func() error { return nil }

	context := core.NewEVMContext(msg, header, b.brn.BlockChain(), nil)
	return vm.NewEVM(context, state, b.brn.chainConfig, vmCfg), vmError, nil
}

func (b *EthApiBackend) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
	return b.brn.BlockChain().SubscribeRemovedLogsEvent(ch)
}

func (b *EthApiBackend) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	return b.brn.BlockChain().SubscribeChainEvent(ch)
}

func (b *EthApiBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return b.brn.BlockChain().SubscribeChainHeadEvent(ch)
}

func (b *EthApiBackend) SubscribeChainSideEvent(ch chan<- core.ChainSideEvent) event.Subscription {
	return b.brn.BlockChain().SubscribeChainSideEvent(ch)
}

func (b *EthApiBackend) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.brn.BlockChain().SubscribeLogsEvent(ch)
}

func (b *EthApiBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	return b.brn.txPool.AddLocal(signedTx)
}

func (b *EthApiBackend) GetPoolTransactions() (types.Transactions, error) {
	pending, err := b.brn.txPool.Pending()
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
	return b.brn.txPool.Get(hash)
}

func (b *EthApiBackend) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	return b.brn.txPool.State().GetNonce(addr), nil
}

func (b *EthApiBackend) Stats() (pending int, queued int) {
	return b.brn.txPool.Stats()
}

func (b *EthApiBackend) TxPoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	return b.brn.TxPool().Content()
}

func (b *EthApiBackend) SubscribeTxPreEvent(ch chan<- core.TxPreEvent) event.Subscription {
	return b.brn.TxPool().SubscribeTxPreEvent(ch)
}

func (b *EthApiBackend) Downloader() *downloader.Downloader {
	return b.brn.Downloader()
}

func (b *EthApiBackend) ProtocolVersion() int {
	return b.brn.EthVersion()
}

func (b *EthApiBackend) SuggestPrice(ctx context.Context) (*big.Int, error) {
	return b.gpo.SuggestPrice(ctx)
}

func (b *EthApiBackend) ChainDb() brndb.Database {
	return b.brn.ChainDb()
}

func (b *EthApiBackend) EventMux() *event.TypeMux {
	return b.brn.EventMux()
}

func (b *EthApiBackend) AccountManager() *accounts.Manager {
	return b.brn.AccountManager()
}

func (b *EthApiBackend) BloomStatus() (uint64, uint64) {
	sections, _, _ := b.brn.bloomIndexer.Sections()
	return params.BloomBitsBlocks, sections
}

func (b *EthApiBackend) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {
	for i := 0; i < bloomFilterThreads; i++ {
		go session.Multiplex(bloomRetrievalBatch, bloomRetrievalWait, b.brn.bloomRequests)
	}
}
