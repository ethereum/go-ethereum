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
	"sort"
	"sync"

	"github.com/ethereum/go-ethereum"
	btypes "github.com/ethereum/go-ethereum/beacon/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"
)

const maxTxAge = 300 // sent transactions are typically remembered for an hour after last seen pending

type txAndReceiptsFields struct {
	signer types.Signer

	receiptsCache    *lru.Cache[common.Hash, types.Receipts]
	receiptsRequests *requestMap[common.Hash, types.Receipts]
	txPosCache       *lru.Cache[common.Hash, txInBlock]

	sentTxLock                  sync.Mutex
	sentTxs                     map[common.Address]senderTxs
	headCounter, lastHeadNumber uint64
}

type txInBlock struct {
	blockNumber uint64
	blockHash   common.Hash // only considered valid if the block is canonical
	index       uint
}

type sentTx struct {
	nonce    uint64
	lastSeen uint64 // headCounter value where the tx has been sent or last seen pending
}

type senderTxs map[common.Hash]sentTx

func (c *Client) initTxAndReceipts() {
	c.receiptsCache = lru.NewCache[common.Hash, types.Receipts](10)
	c.txPosCache = lru.NewCache[common.Hash, txInBlock](10000)
	c.sentTxs = make(map[common.Address]senderTxs)
	c.signer = types.LatestSigner(c.elConfig)
	c.receiptsRequests = newRequestMap[common.Hash, types.Receipts](c.requestBlockReceipts)
}

func (c *Client) closeTxAndReceipts() {
	c.receiptsRequests.close()
}

func (c *Client) getTxByHash(ctx context.Context, txHash common.Hash) (tx *types.Transaction, isPending bool, err error) {
	if pos, ok := c.txPosCache.Get(txHash); ok {
		if hash, ok := c.getCachedHash(pos.blockNumber); ok && hash == pos.blockHash {
			if block, ok := c.blockCache.Get(pos.blockHash); ok {
				transactions := block.Transactions()
				if pos.index >= uint(len(transactions)) {
					return nil, false, errors.New("transaction index out of range")
				}
				return transactions[pos.index], false, nil
			}
		}
	}
	var headBlockNumber uint64
	if head := c.getHead(); head != nil {
		headBlockNumber = head.BlockNumber()
	}
	return c.getUncachedTxByHash(ctx, txHash, headBlockNumber)
}

func (c *Client) getUncachedTxByHash(ctx context.Context, txHash common.Hash, headBlockNumber uint64) (tx *types.Transaction, isPending bool, err error) {
	tx, isPending, err = c.requestTxByHash(ctx, txHash)
	if err == nil && !isPending {
		receipt, err := c.requestReceiptByTxHash(ctx, txHash)
		if err == ethereum.NotFound {
			return tx, false, nil
		}
		if err != nil {
			return nil, false, err
		}
		if !receipt.BlockNumber.IsUint64() || headBlockNumber < receipt.BlockNumber.Uint64() {
			// consider it pending if it's reported to be included higher than the light chain head
			isPending = true
		}
	}
	return
}

func (c *Client) getReceiptByTxHash(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	if pos, ok := c.txPosCache.Get(txHash); ok {
		if hash, ok := c.getCachedHash(pos.blockNumber); ok && hash == pos.blockHash {
			if receipts, ok := c.receiptsCache.Get(pos.blockHash); ok {
				if pos.index >= uint(len(receipts)) {
					return nil, errors.New("transaction index out of range")
				}
				return receipts[pos.index], nil
			}
		}
	}
	receipt, err := c.requestReceiptByTxHash(ctx, txHash)
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
	if head := c.getHead(); head == nil || head.BlockNumber() < blockNumber {
		// consider it pending if it's reported to be included higher than the light chain head
		return nil, ethereum.NotFound
	}
	canonicalHash, err := c.getHash(ctx, blockNumber)
	if err != nil {
		return nil, err
	}
	if receipt.BlockHash != canonicalHash {
		return nil, errors.New("receipt references non-canonical block")
	}
	// check if it is the actual canonical receipt at the given position
	receipts, err := c.getBlockReceipts(ctx, receipt.BlockHash)
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
	c.txPosCache.Add(receipt.TxHash, txInBlock{blockNumber: blockNumber, blockHash: receipt.BlockHash, index: receipt.TransactionIndex})
	return receipt, err
}

func (c *Client) getBlockReceipts(ctx context.Context, blockHash common.Hash) (types.Receipts, error) {
	if receipts, ok := c.receiptsCache.Get(blockHash); ok {
		return receipts, nil
	}
	request := c.receiptsRequests.request(blockHash)
	block, err := c.getBlock(ctx, blockHash)
	if err != nil {
		return nil, err
	}
	receipts, err := request.getResult(ctx)
	if err == nil {
		receipts, err = c.validateBlockReceipts(block, receipts)
	}
	if err == nil {
		c.receiptsCache.Add(blockHash, receipts)
	}
	request.release()
	return receipts, err
}

func (c *Client) validateBlockReceipts(block *types.Block, receipts types.Receipts) (types.Receipts, error) {
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
	if err := newReceipts.DeriveFields(c.elConfig, block.Hash(), block.NumberU64(), block.Time(), block.BaseFee(), blobGasPrice, block.Transactions()); err != nil {
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

func (c *Client) requestTxByHash(ctx context.Context, hash common.Hash) (tx *types.Transaction, isPending bool, err error) {
	var json *rpcTransaction
	err = c.client.CallContext(ctx, &json, "eth_getTransactionByHash", hash)
	if err != nil {
		return nil, false, err
	} else if json == nil {
		return nil, false, ethereum.NotFound
	} else if _, r, _ := json.tx.RawSignatureValues(); r == nil {
		return nil, false, errors.New("server returned transaction without signature")
	} else if json.tx.Hash() != hash {
		return nil, false, errors.New("invalid transaction hash")
	}
	if json.From != nil && json.BlockHash != nil {
		setSenderFromServer(json.tx, *json.From, *json.BlockHash)
	}
	return json.tx, json.BlockNumber == nil, nil
}

func (c *Client) requestReceiptByTxHash(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	var r *types.Receipt
	err := c.client.CallContext(ctx, &r, "eth_getTransactionReceipt", txHash)
	if err == nil && r == nil {
		return nil, ethereum.NotFound
	}
	return r, err
}

func (c *Client) requestBlockReceipts(ctx context.Context, blockHash common.Hash) (types.Receipts, error) {
	var r []*types.Receipt
	err := c.client.CallContext(ctx, &r, "eth_getBlockReceipts", blockHash)
	if err == nil && r == nil {
		return nil, ethereum.NotFound
	}
	return types.Receipts(r), err
}

func (c *Client) cacheBlockTxPositions(block *types.Block) {
	blockNumber, blockHash := block.NumberU64(), block.Hash()
	for i, tx := range block.Transactions() {
		c.txPosCache.Add(tx.Hash(), txInBlock{blockNumber: blockNumber, blockHash: blockHash, index: uint(i)})
	}
}

func (c *Client) sendTransaction(ctx context.Context, tx *types.Transaction) error {
	sender, err := types.Sender(c.signer, tx)
	if err != nil {
		return nil
	}
	data, err := tx.MarshalBinary()
	if err != nil {
		return err
	}
	if err := c.client.CallContext(ctx, nil, "eth_sendRawTransaction", hexutil.Encode(data)); err != nil {
		return err
	}
	c.sentTxLock.Lock()
	c.markTxAsSeen(sender, tx)
	c.sentTxLock.Unlock()
	return nil
}

func (c *Client) markTxAsSeen(sender common.Address, tx *types.Transaction) {
	senderTxs := c.sentTxs[sender]
	if senderTxs == nil {
		senderTxs = make(map[common.Hash]sentTx)
		c.sentTxs[sender] = senderTxs
	}
	senderTxs[tx.Hash()] = sentTx{nonce: tx.Nonce(), lastSeen: c.headCounter}
}

func (c *Client) txAndReceiptsNewHead(number uint64, hash common.Hash) {
	c.sentTxLock.Lock()
	if number > c.lastHeadNumber {
		c.lastHeadNumber = number
		c.headCounter++
		c.discardOldTxs()
	}
	c.sentTxLock.Unlock()
}

func (c *Client) discardOldTxs() {
	for sender, senderTxs := range c.sentTxs {
		for txHash, sentTx := range senderTxs {
			if sentTx.lastSeen+maxTxAge < c.headCounter {
				delete(senderTxs, txHash)
			}
		}
		if len(senderTxs) == 0 {
			delete(c.sentTxs, sender)
		}
	}
}

func (c *Client) allSenders() []common.Address {
	c.sentTxLock.Lock()
	allSenders := make([]common.Address, 0, len(c.sentTxs))
	for sender := range c.sentTxs {
		allSenders = append(allSenders, sender)
	}
	c.sentTxLock.Unlock()
	return allSenders
}

func (c *Client) nonceAndPendingTxs(ctx context.Context, head *btypes.ExecutionHeader, sender common.Address) (uint64, types.Transactions, error) {
	proof, err := c.fetchProof(ctx, proofRequest{blockNumber: head.BlockNumber(), address: sender, storageKeys: ""})
	if err != nil {
		return 0, nil, err
	}
	if err := c.validateProof(proof, head.StateRoot(), sender, nil); err != nil {
		return 0, nil, err
	}

	c.sentTxLock.Lock()
	senderTxs := c.sentTxs[sender]
	c.sentTxLock.Unlock()

	type pendingResult struct {
		pendingTx *types.Transaction
		err       error
	}

	resultCh := make(chan pendingResult, len(senderTxs))
	var reqCount int
	for txHash, sentTx := range senderTxs {
		if sentTx.nonce <= proof.Nonce {
			continue
		}
		reqCount++
		go func() {
			tx, isPending, err := c.getUncachedTxByHash(ctx, txHash, head.BlockNumber())
			if err == nil && isPending {
				resultCh <- pendingResult{pendingTx: tx}
			} else {
				resultCh <- pendingResult{err: err}
			}
		}()
	}
	pendingList := make(types.Transactions, 0, len(senderTxs))
	for ; reqCount > 0; reqCount-- {
		res := <-resultCh
		if res.err != nil {
			return 0, nil, err
		}
		if res.pendingTx != nil {
			pendingList = append(pendingList, res.pendingTx)
		}
	}
	sort.Sort(types.TxByNonce(pendingList))
	c.sentTxLock.Lock()
	for _, tx := range pendingList {
		c.markTxAsSeen(sender, tx)
	}
	c.sentTxLock.Unlock()
	return proof.Nonce, pendingList, nil
}
