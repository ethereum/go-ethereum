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
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/rpc"
)

// ScrollAPI provides private RPC methods to query the L1 message database.
type ScrollAPI struct {
	eth *Ethereum
}

// l1MessageTxRPC is the RPC-layer representation of an L1 message.
type l1MessageTxRPC struct {
	QueueIndex uint64          `json:"queueIndex"`
	Gas        uint64          `json:"gas"`
	To         *common.Address `json:"to"`
	Value      *hexutil.Big    `json:"value"`
	Data       hexutil.Bytes   `json:"data"`
	Sender     common.Address  `json:"sender"`
	Hash       common.Hash     `json:"hash"`
}

// NewScrollAPI creates a new RPC service to query the L1 message database.
func NewScrollAPI(eth *Ethereum) *ScrollAPI {
	return &ScrollAPI{eth: eth}
}

// GetL1SyncHeight returns the latest synced L1 block height from the local database.
func (api *ScrollAPI) GetL1SyncHeight(ctx context.Context) (height *uint64, err error) {
	return rawdb.ReadSyncedL1BlockNumber(api.eth.ChainDb()), nil
}

// GetL1MessageByIndex queries an L1 message by its index in the local database.
func (api *ScrollAPI) GetL1MessageByIndex(ctx context.Context, queueIndex uint64) (height *l1MessageTxRPC, err error) {
	msg := rawdb.ReadL1Message(api.eth.ChainDb(), queueIndex)
	if msg == nil {
		return nil, nil
	}
	rpcMsg := l1MessageTxRPC{
		QueueIndex: msg.QueueIndex,
		Gas:        msg.Gas,
		To:         msg.To,
		Value:      (*hexutil.Big)(msg.Value),
		Data:       msg.Data,
		Sender:     msg.Sender,
		Hash:       types.NewTx(msg).Hash(),
	}
	return &rpcMsg, nil
}

// GetFirstQueueIndexNotInL2Block returns the first L1 message queue index that is
// not included in the chain up to and including the provided block.
func (api *ScrollAPI) GetFirstQueueIndexNotInL2Block(ctx context.Context, hash common.Hash) (queueIndex *uint64, err error) {
	return rawdb.ReadFirstQueueIndexNotInL2Block(api.eth.ChainDb(), hash), nil
}

// GetLatestRelayedQueueIndex returns the highest L1 message queue index included in the canonical chain.
func (api *ScrollAPI) GetLatestRelayedQueueIndex(ctx context.Context) (queueIndex *uint64, err error) {
	block := api.eth.blockchain.CurrentBlock()
	queueIndex, err = api.GetFirstQueueIndexNotInL2Block(ctx, block.Hash())
	if queueIndex == nil || err != nil {
		return queueIndex, err
	}
	if *queueIndex == 0 {
		return nil, nil
	}
	lastIncluded := *queueIndex - 1
	return &lastIncluded, nil
}

// rpcMarshalBlock uses the generalized output filler, then adds the total difficulty field, which requires
// a `ScrollAPI`.
func (api *ScrollAPI) rpcMarshalBlock(ctx context.Context, b *types.Block, fullTx bool) (map[string]interface{}, error) {
	fields := ethapi.RPCMarshalBlock(b, true, fullTx, api.eth.APIBackend.ChainConfig())
	fields["totalDifficulty"] = (*hexutil.Big)(api.eth.APIBackend.GetTd(ctx, b.Hash()))
	rc := rawdb.ReadBlockRowConsumption(api.eth.ChainDb(), b.Hash())
	if rc != nil {
		fields["rowConsumption"] = rc
	} else {
		fields["rowConsumption"] = nil
	}
	return fields, nil
}

// GetBlockByHash returns the requested block. When fullTx is true all transactions in the block are returned in full
// detail, otherwise only the transaction hash is returned.
func (api *ScrollAPI) GetBlockByHash(ctx context.Context, hash common.Hash, fullTx bool) (map[string]interface{}, error) {
	block, err := api.eth.APIBackend.BlockByHash(ctx, hash)
	if block != nil {
		return api.rpcMarshalBlock(ctx, block, fullTx)
	}
	return nil, err
}

// GetBlockByNumber returns the requested block. When fullTx is true all transactions in the block are returned in full
// detail, otherwise only the transaction hash is returned.
func (api *ScrollAPI) GetBlockByNumber(ctx context.Context, number rpc.BlockNumber, fullTx bool) (map[string]interface{}, error) {
	block, err := api.eth.APIBackend.BlockByNumber(ctx, number)
	if block != nil {
		return api.rpcMarshalBlock(ctx, block, fullTx)
	}
	return nil, err
}

// GetNumSkippedTransactions returns the number of skipped transactions.
func (api *ScrollAPI) GetNumSkippedTransactions(ctx context.Context) (uint64, error) {
	return rawdb.ReadNumSkippedTransactions(api.eth.ChainDb()), nil
}

// SyncStatus includes L2 block sync height, L1 rollup sync height,
// L1 message sync height, and L2 finalized block height.
type SyncStatus struct {
	L2BlockSyncHeight      uint64 `json:"l2BlockSyncHeight,omitempty"`
	L1RollupSyncHeight     uint64 `json:"l1RollupSyncHeight,omitempty"`
	L1MessageSyncHeight    uint64 `json:"l1MessageSyncHeight,omitempty"`
	L2FinalizedBlockHeight uint64 `json:"l2FinalizedBlockHeight,omitempty"`
}

// SyncStatus returns the overall rollup status including L2 block sync height, L1 rollup sync height,
// L1 message sync height, and L2 finalized block height.
func (api *ScrollAPI) SyncStatus(_ context.Context) *SyncStatus {
	status := &SyncStatus{}

	l2BlockHeader := api.eth.blockchain.CurrentHeader()
	if l2BlockHeader != nil {
		status.L2BlockSyncHeight = l2BlockHeader.Number.Uint64()
	}

	l1RollupSyncHeightPtr := rawdb.ReadRollupEventSyncedL1BlockNumber(api.eth.ChainDb())
	if l1RollupSyncHeightPtr != nil {
		status.L1RollupSyncHeight = *l1RollupSyncHeightPtr
	}

	l1MessageSyncHeightPtr := rawdb.ReadSyncedL1BlockNumber(api.eth.ChainDb())
	if l1MessageSyncHeightPtr != nil {
		status.L1MessageSyncHeight = *l1MessageSyncHeightPtr
	}

	l2FinalizedBlockHeightPtr := rawdb.ReadFinalizedL2BlockNumber(api.eth.ChainDb())
	if l2FinalizedBlockHeightPtr != nil {
		status.L2FinalizedBlockHeight = *l2FinalizedBlockHeightPtr
	}

	return status
}

// EstimateL1DataFee returns an estimate of the L1 data fee required to
// process the given transaction against the current pending block.
func (api *ScrollAPI) EstimateL1DataFee(ctx context.Context, args ethapi.TransactionArgs, blockNrOrHash *rpc.BlockNumberOrHash) (*hexutil.Uint64, error) {
	bNrOrHash := rpc.BlockNumberOrHashWithNumber(rpc.PendingBlockNumber)
	if blockNrOrHash != nil {
		bNrOrHash = *blockNrOrHash
	}

	l1DataFee, err := ethapi.EstimateL1MsgFee(ctx, api.eth.APIBackend, args, bNrOrHash, nil, 0, api.eth.APIBackend.RPCGasCap(), api.eth.APIBackend.ChainConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to estimate L1 data fee: %w", err)
	}

	result := hexutil.Uint64(l1DataFee.Uint64())
	return &result, nil
}

// RPCTransaction is the standard RPC transaction return type with some additional skip-related fields.
type RPCTransaction struct {
	ethapi.RPCTransaction
	SkipReason      string       `json:"skipReason"`
	SkipBlockNumber *hexutil.Big `json:"skipBlockNumber"`
	SkipBlockHash   *common.Hash `json:"skipBlockHash,omitempty"`

	// wrapped traces, currently only available for `scroll_getSkippedTransaction` API, when `MinerStoreSkippedTxTracesFlag` is set
	Traces *types.BlockTrace `json:"traces,omitempty"`
}

// GetSkippedTransaction returns a skipped transaction by its hash.
func (api *ScrollAPI) GetSkippedTransaction(ctx context.Context, hash common.Hash) (*RPCTransaction, error) {
	stx := rawdb.ReadSkippedTransaction(api.eth.ChainDb(), hash)
	if stx == nil {
		return nil, nil
	}
	var rpcTx RPCTransaction
	rpcTx.RPCTransaction = *ethapi.NewRPCTransaction(stx.Tx, common.Hash{}, 0, 0, 0, nil, api.eth.blockchain.Config())
	rpcTx.SkipReason = stx.Reason
	rpcTx.SkipBlockNumber = (*hexutil.Big)(new(big.Int).SetUint64(stx.BlockNumber))
	rpcTx.SkipBlockHash = stx.BlockHash
	if len(stx.TracesBytes) != 0 {
		traces := &types.BlockTrace{}
		if err := json.Unmarshal(stx.TracesBytes, traces); err != nil {
			return nil, fmt.Errorf("fail to Unmarshal traces for skipped tx, hash: %s, err: %w", hash.String(), err)
		}
		rpcTx.Traces = traces
	}
	return &rpcTx, nil
}

// GetSkippedTransactionHashes returns a list of skipped transaction hashes between the two indices provided (inclusive).
func (api *ScrollAPI) GetSkippedTransactionHashes(ctx context.Context, from uint64, to uint64) ([]common.Hash, error) {
	it := rawdb.IterateSkippedTransactionsFrom(api.eth.ChainDb(), from)
	defer it.Release()

	var hashes []common.Hash

	for it.Next() {
		if it.Index() > to {
			break
		}
		hashes = append(hashes, it.TransactionHash())
	}

	return hashes, nil
}
