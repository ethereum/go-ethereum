// Copyright 2024 The go-ethereum Authors
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

package lightclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"
)

type txAndReceipts struct {
	client           *rpc.Client
	canonicalChain   *canonicalChain
	blocksAndHeaders *blocksAndHeaders
	elConfig         *params.ChainConfig
	receiptsCache    *lru.Cache[common.Hash, types.Receipts]
	receiptsRequests *requestMap[common.Hash, types.Receipts]
	txPosCache       *lru.Cache[common.Hash, txInBlock]
}

type txInBlock struct {
	blockNumber uint64
	blockHash   common.Hash // only considered valid if the block is canonical
	index       uint
}

func newTxAndReceipts(client *rpc.Client, canonicalChain *canonicalChain, blocksAndHeaders *blocksAndHeaders, elConfig *params.ChainConfig) *txAndReceipts {
	t := &txAndReceipts{
		client:           client,
		canonicalChain:   canonicalChain,
		blocksAndHeaders: blocksAndHeaders,
		elConfig:         elConfig,
		receiptsCache:    lru.NewCache[common.Hash, types.Receipts](10),
		txPosCache:       lru.NewCache[common.Hash, txInBlock](10000),
	}
	t.receiptsRequests = newRequestMap[common.Hash, types.Receipts](t.requestBlockReceipts)
	return t
}

func (t *txAndReceipts) getTxByHash(ctx context.Context, txHash common.Hash) (tx *types.Transaction, isPending bool, err error) {
	if pos, ok := t.txPosCache.Get(txHash); ok {
		if hash, ok := t.canonicalChain.getCachedHash(pos.blockNumber); ok && hash == pos.blockHash {
			if block, ok := t.blocksAndHeaders.blockCache.Get(pos.blockHash); ok {
				return block.Transactions()[pos.index], false, nil //TODO index range check
			}
		}
	}
	tx, isPending, err = t.requestTxByHash(ctx, txHash)
	if err == nil && !isPending {
		receipt, err := t.requestReceiptByTxHash(ctx, txHash)
		if err == ethereum.NotFound {
			return tx, false, nil
		}
		if err != nil {
			return nil, false, err
		}
		if head := t.canonicalChain.getHead(); head == nil ||
			!receipt.BlockNumber.IsUint64() || head.BlockNumber() < receipt.BlockNumber.Uint64() {
			// consider it pending if it's reported to be included higher than the light chain head
			isPending = true
		}
	}
	return
}

func (t *txAndReceipts) getReceiptByTxHash(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	if pos, ok := t.txPosCache.Get(txHash); ok {
		if hash, ok := t.canonicalChain.getCachedHash(pos.blockNumber); ok && hash == pos.blockHash {
			if receipts, ok := t.receiptsCache.Get(pos.blockHash); ok {
				return receipts[pos.index], nil //TODO index range check
			}
		}
	}
	receipt, err := t.requestReceiptByTxHash(ctx, txHash)
	if err != nil {
		return nil, err
	}
	// check if it indeed belongs to the requested transaction
	if receipt.TxHash != txHash {
		return nil, errors.New("receipt references another transaction")
	}
	// check if its inclusion position is canonical
	if !receipt.BlockNumber.IsUint64() {
		return nil, errors.New("receipt references non-canonical block")
	}
	blockNumber := receipt.BlockNumber.Uint64()
	if head := t.canonicalChain.getHead(); head == nil || head.BlockNumber() < blockNumber {
		// consider it pending if it's reported to be included higher than the light chain head
		return nil, ethereum.NotFound
	}
	canonicalHash, err := t.canonicalChain.getHash(ctx, blockNumber)
	if err != nil {
		return nil, err
	}
	if receipt.BlockHash != canonicalHash {
		return nil, errors.New("receipt references non-canonical block")
	}
	// check if it is the actual canonical receipt at the given position
	receipts, err := t.getBlockReceipts(ctx, receipt.BlockHash)
	if err != nil {
		return nil, err
	}
	if receipt.TransactionIndex >= uint(len(receipts)) {
		return nil, errors.New("receipt references out-of-range transaction index")
	}
	// compare the JSON encoding of received and canonical versions
	jsonReceived, err := json.Marshal(receipt)
	if err != nil {
		return nil, err
	}
	jsonCanonical, err := json.Marshal(receipts[receipt.TransactionIndex])
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(jsonReceived, jsonCanonical) {
		return nil, errors.New("received and derived receipts do not match")
	}
	t.txPosCache.Add(receipt.TxHash, txInBlock{blockNumber: blockNumber, blockHash: receipt.BlockHash, index: receipt.TransactionIndex})
	return receipt, err
}

func (t *txAndReceipts) getBlockReceipts(ctx context.Context, blockHash common.Hash) (types.Receipts, error) {
	if receipts, ok := t.receiptsCache.Get(blockHash); ok {
		return receipts, nil
	}
	request := t.receiptsRequests.request(blockHash)
	block, err := t.blocksAndHeaders.getBlock(ctx, blockHash)
	if err != nil {
		return nil, err
	}
	receipts, err := request.getResult(ctx)
	if err == nil {
		receipts, err = t.validateBlockReceipts(block, receipts)
	}
	if err == nil {
		t.receiptsCache.Add(blockHash, receipts)
	}
	request.release()
	return receipts, err
}

func (t *txAndReceipts) validateBlockReceipts(block *types.Block, receipts types.Receipts) (types.Receipts, error) {
	// verify consensus fields agains receipts hash in block
	var hash common.Hash
	if len(receipts) == 0 {
		hash = types.EmptyReceiptsHash
	} else {
		hash = types.DeriveSha(receipts, trie.NewStackTrie(nil))
	}
	if hash != block.ReceiptHash() {
		return nil, errors.New("invalid receipts hash")
	}
	// copy verified consensus fields
	newReceipts := make(types.Receipts, len(receipts))
	for i, receipt := range receipts {
		enc, err := receipt.MarshalBinary()
		if err != nil {
			return nil, err
		}
		newReceipt := &types.Receipt{}
		if err := newReceipt.UnmarshalBinary(enc); err != nil {
			return nil, err
		}
		newReceipts[i] = newReceipt
	}
	// derive non-consensus fields again
	var blobGasPrice *big.Int
	if block.ExcessBlobGas() != nil {
		blobGasPrice = eip4844.CalcBlobFee(*block.ExcessBlobGas())
	}
	if err := newReceipts.DeriveFields(t.elConfig, block.Hash(), block.NumberU64(), block.Time(), block.BaseFee(), blobGasPrice, block.Transactions()); err != nil {
		return nil, err
	}
	// compare the JSON encoding of received and derived versions
	jsonReceived, err := json.Marshal(receipts)
	if err != nil {
		return nil, err
	}
	jsonDerived, err := json.Marshal(receipts)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(jsonReceived, jsonDerived) {
		return nil, errors.New("received and derived receipts do not match")
	}
	return newReceipts, nil
}

func (t *txAndReceipts) requestTxByHash(ctx context.Context, hash common.Hash) (tx *types.Transaction, isPending bool, err error) {
	var json *rpcTransaction
	err = t.client.CallContext(ctx, &json, "eth_getTransactionByHash", hash)
	if err != nil {
		return nil, false, err
	} else if json == nil {
		return nil, false, ethereum.NotFound
	} else if _, r, _ := json.tx.RawSignatureValues(); r == nil {
		return nil, false, errors.New("server returned transaction without signature")
	}
	if json.From != nil && json.BlockHash != nil {
		setSenderFromServer(json.tx, *json.From, *json.BlockHash)
	}
	return json.tx, json.BlockNumber == nil, nil
}

func (t *txAndReceipts) requestReceiptByTxHash(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	var r *types.Receipt
	err := t.client.CallContext(ctx, &r, "eth_getTransactionReceipt", txHash)
	if err == nil && r == nil {
		return nil, ethereum.NotFound
	}
	return r, err
}

func (t *txAndReceipts) requestBlockReceipts(ctx context.Context, blockHash common.Hash) (types.Receipts, error) {
	var r []*types.Receipt
	err := t.client.CallContext(ctx, &r, "eth_getBlockReceipts", blockHash)
	if err == nil && r == nil {
		return nil, ethereum.NotFound
	}
	return types.Receipts(r), err
}

func (t *txAndReceipts) cacheBlockTxPositions(block *types.Block) {
	blockNumber, blockHash := block.NumberU64(), block.Hash()
	for i, tx := range block.Transactions() {
		t.txPosCache.Add(tx.Hash(), txInBlock{blockNumber: blockNumber, blockHash: blockHash, index: uint(i)})
	}
}
