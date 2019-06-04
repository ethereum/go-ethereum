// Copyright 2016 The go-ethereum Authors
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

package les

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/bloombits"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/eth/gasprice"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

// ClientBackend is a backend implementation for the light Client only
type ClientBackend struct {
	extRPCEnabled bool
	eth           *LightEthereum
	gpo           *gasprice.Oracle
}

// ChainConfig returns the LightEthereum service config
func (b *ClientBackend) ChainConfig() *params.ChainConfig {
	return b.eth.chainConfig
}

// CurrentBlock returns the LightEthereum service chain current header block
func (b *ClientBackend) CurrentBlock() *types.Block {
	return types.NewBlockWithHeader(b.eth.BlockChain().CurrentHeader())
}

// SetHead cancels the downloader and sets the head to a certain number
func (b *ClientBackend) SetHead(number uint64) {
	b.eth.protocolManager.downloader.Cancel()
	b.eth.blockchain.SetHead(number)
}

// HeaderByNumber returns either the chain's current header or an on-demand-requested header, given a block number
func (b *ClientBackend) HeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Header, error) {
	if blockNr == rpc.LatestBlockNumber || blockNr == rpc.PendingBlockNumber {
		return b.eth.blockchain.CurrentHeader(), nil
	}
	return b.eth.blockchain.GetHeaderByNumberOdr(ctx, uint64(blockNr))
}

// HeaderByHash returns a block header for the given block hash
func (b *ClientBackend) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	return b.eth.blockchain.GetHeaderByHash(hash), nil
}

// BlockByNumber identifies a header by number and then returns the block with the corresponding block hash
func (b *ClientBackend) BlockByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Block, error) {
	header, err := b.HeaderByNumber(ctx, blockNr)
	if header == nil || err != nil {
		return nil, err
	}
	return b.GetBlock(ctx, header.Hash())
}

// StateAndHeaderByNumber identifies a block header by the supplied number, then returns that header's associated state and the header itself
func (b *ClientBackend) StateAndHeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*state.StateDB, *types.Header, error) {
	header, err := b.HeaderByNumber(ctx, blockNr)
	if err != nil {
		return nil, nil, err
	}
	if header == nil {
		return nil, nil, errors.New("header not found")
	}
	return light.NewState(ctx, header, b.eth.odr), header, nil
}

// GetBlock returns a block identified by block hash
func (b *ClientBackend) GetBlock(ctx context.Context, blockHash common.Hash) (*types.Block, error) {
	return b.eth.blockchain.GetBlockByHash(ctx, blockHash)
}

// GetReceipts returns a set of receipts for a block, identified by the supplied hash
func (b *ClientBackend) GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error) {
	if number := rawdb.ReadHeaderNumber(b.eth.chainDb, hash); number != nil {
		return light.GetBlockReceipts(ctx, b.eth.odr, hash, *number)
	}
	return nil, nil
}

// GetLogs identifies the block header number given the supplied hash and then returns the block logs
func (b *ClientBackend) GetLogs(ctx context.Context, hash common.Hash) ([][]*types.Log, error) {
	if number := rawdb.ReadHeaderNumber(b.eth.chainDb, hash); number != nil {
		return light.GetBlockLogs(ctx, b.eth.odr, hash, *number)
	}
	return nil, nil
}

// GetTd retrieves a block's total difficulty in the canonical chain by hash
func (b *ClientBackend) GetTd(hash common.Hash) *big.Int {
	return b.eth.blockchain.GetTdByHash(hash)
}

// GetEVM creates an EVM initialised with an account and balance from the supplied message
func (b *ClientBackend) GetEVM(ctx context.Context, msg core.Message, state *state.StateDB, header *types.Header) (*vm.EVM, func() error, error) {
	state.SetBalance(msg.From(), math.MaxBig256)
	context := core.NewEVMContext(msg, header, b.eth.blockchain, nil)
	return vm.NewEVM(context, state, b.eth.chainConfig, vm.Config{}), state.Error, nil
}

// SendTx add the transaction to the local transaction pool
func (b *ClientBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	return b.eth.txPool.Add(ctx, signedTx)
}

// RemoveTx removes a transaction from the local transaction pool
func (b *ClientBackend) RemoveTx(txHash common.Hash) {
	b.eth.txPool.RemoveTx(txHash)
}

// GetPoolTransactions returns all currently processable transactions.
func (b *ClientBackend) GetPoolTransactions() (types.Transactions, error) {
	return b.eth.txPool.GetTransactions()
}

// GetPoolTransaction returns a transaction if it is contained in the pool and nil otherwise.
func (b *ClientBackend) GetPoolTransaction(txHash common.Hash) *types.Transaction {
	return b.eth.txPool.GetTransaction(txHash)
}

// GetTransaction retrieves a canonical transaction by hash and also returns its position in the chain
func (b *ClientBackend) GetTransaction(ctx context.Context, txHash common.Hash) (*types.Transaction, common.Hash, uint64, uint64, error) {
	return light.GetTransaction(ctx, b.eth.odr, txHash)
}

// GetPoolNonce returns the "pending" nonce of a given address. It always queries
// the nonce belonging to the latest header too in order to detect if another
// client using the same key sent a transaction.
func (b *ClientBackend) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	return b.eth.txPool.GetNonce(ctx, addr)
}

// Stats returns the number of currently pending (locally created) transactions
func (b *ClientBackend) Stats() (pending int, queued int) {
	return b.eth.txPool.Stats(), 0
}

// TxPoolContent retrieves the data content of the transaction pool, returning all the
// pending as well as queued transactions, grouped by account and nonce.
func (b *ClientBackend) TxPoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	return b.eth.txPool.Content()
}

// SubscribeNewTxsEvent registers a subscription of core.NewTxsEvent and
// starts sending event to the given channel.
func (b *ClientBackend) SubscribeNewTxsEvent(ch chan<- core.NewTxsEvent) event.Subscription {
	return b.eth.txPool.SubscribeNewTxsEvent(ch)
}

// SubscribeChainEvent registers a subscription of ChainEvent.
func (b *ClientBackend) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	return b.eth.blockchain.SubscribeChainEvent(ch)
}

// SubscribeChainHeadEvent registers a subscription of ChainHeadEvent.
func (b *ClientBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return b.eth.blockchain.SubscribeChainHeadEvent(ch)
}

// SubscribeChainSideEvent registers a subscription of ChainSideEvent.
func (b *ClientBackend) SubscribeChainSideEvent(ch chan<- core.ChainSideEvent) event.Subscription {
	return b.eth.blockchain.SubscribeChainSideEvent(ch)
}

// SubscribeLogsEvent implements the interface of filters.Backend
// LightChain does not send logs events, so return an empty subscription.
func (b *ClientBackend) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.eth.blockchain.SubscribeLogsEvent(ch)
}

// SubscribeRemovedLogsEvent implements the interface of filters.Backend
// LightChain does not send core.RemovedLogsEvent, so return an empty subscription.
func (b *ClientBackend) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
	return b.eth.blockchain.SubscribeRemovedLogsEvent(ch)
}

// Downloader returns the LightEthereum downloader
func (b *ClientBackend) Downloader() *downloader.Downloader {
	return b.eth.Downloader()
}

// ProtocolVersion return the Les version number offset by 10000
func (b *ClientBackend) ProtocolVersion() int {
	return b.eth.LesVersion() + 10000
}

// SuggestPrice returns the gas price oracle price suggestion
func (b *ClientBackend) SuggestPrice(ctx context.Context) (*big.Int, error) {
	return b.gpo.SuggestPrice(ctx)
}

// ChainDb returns the ClientBackend db representation
func (b *ClientBackend) ChainDb() ethdb.Database {
	return b.eth.chainDb
}

func (b *ClientBackend) EventMux() *event.TypeMux {
	return b.eth.eventMux
}

// AccountManager returns an overarching account manager needed for signing transactions
func (b *ClientBackend) AccountManager() *accounts.Manager {
	return b.eth.accountManager
}

// ExtRPCEnabled indicates if the external RPC API is enabled
func (b *ClientBackend) ExtRPCEnabled() bool {
	return b.extRPCEnabled
}

// RPCGasCap return the  global gas cap for eth_call over rpc (DoS protection)
func (b *ClientBackend) RPCGasCap() *big.Int {
	return b.eth.config.RPCGasCap
}

// BloomStatus returns the number of BloomBitsBlocksClient (the number of blocks a single bloom bit section vector
// contains on the light client side) and sections, (which is the number of processed sections maintained by the indexer
// and also the information about the last header indexed for potential canonical verifications).
func (b *ClientBackend) BloomStatus() (uint64, uint64) {
	if b.eth.bloomIndexer == nil {
		return 0, 0
	}
	sections, _, _ := b.eth.bloomIndexer.Sections()
	return params.BloomBitsBlocksClient, sections
}

func (b *ClientBackend) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {
	for i := 0; i < bloomFilterThreads; i++ {
		go session.Multiplex(bloomRetrievalBatch, bloomRetrievalWait, b.eth.bloomRequests)
	}
}
