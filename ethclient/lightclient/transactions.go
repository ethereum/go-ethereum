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
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"
)

const maxTxAge = 300 // sent transactions are typically remembered for an hour after last seen pending

type txAndReceipts struct {
	client           *rpc.Client
	canonicalChain   *canonicalChain
	blocksAndHeaders *blocksAndHeaders
	lightState       *lightState
	elConfig         *params.ChainConfig
	signer           types.Signer

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

func newTxAndReceipts(client *rpc.Client, canonicalChain *canonicalChain, blocksAndHeaders *blocksAndHeaders, elConfig *params.ChainConfig) *txAndReceipts {
	t := &txAndReceipts{
		client:           client,
		canonicalChain:   canonicalChain,
		blocksAndHeaders: blocksAndHeaders,
		elConfig:         elConfig,
		receiptsCache:    lru.NewCache[common.Hash, types.Receipts](10),
		txPosCache:       lru.NewCache[common.Hash, txInBlock](10000),
		sentTxs:          make(map[common.Address]senderTxs),
		signer:           types.LatestSigner(elConfig),
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
	var headBlockNumber uint64
	if head := t.canonicalChain.getHead(); head != nil {
		headBlockNumber = head.BlockNumber()
	}
	return t.getUncachedTxByHash(ctx, txHash, headBlockNumber)
}

func (t *txAndReceipts) getUncachedTxByHash(ctx context.Context, txHash common.Hash, headBlockNumber uint64) (tx *types.Transaction, isPending bool, err error) {
	tx, isPending, err = t.requestTxByHash(ctx, txHash)
	if err == nil && !isPending {
		receipt, err := t.requestReceiptByTxHash(ctx, txHash)
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
	} else if json.tx.Hash() != hash {
		return nil, false, errors.New("invalid transaction hash")
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

func (t *txAndReceipts) sendTransaction(ctx context.Context, tx *types.Transaction) error {
	sender, err := types.Sender(t.signer, tx)
	if err != nil {
		return nil
	}
	data, err := tx.MarshalBinary()
	if err != nil {
		return err
	}
	if err := t.client.CallContext(ctx, nil, "eth_sendRawTransaction", hexutil.Encode(data)); err != nil {
		return err
	}
	t.sentTxLock.Lock()
	t.markTxAsSeen(sender, tx)
	t.sentTxLock.Unlock()
	return nil
}

func (t *txAndReceipts) markTxAsSeen(sender common.Address, tx *types.Transaction) {
	senderTxs := t.sentTxs[sender]
	if senderTxs == nil {
		senderTxs = make(map[common.Hash]sentTx)
		t.sentTxs[sender] = senderTxs
	}
	senderTxs[tx.Hash()] = sentTx{nonce: tx.Nonce(), lastSeen: t.headCounter}
}

func (t *txAndReceipts) newHead(number uint64, hash common.Hash) {
	t.sentTxLock.Lock()
	if number > t.lastHeadNumber {
		t.lastHeadNumber = number
		t.headCounter++
		t.discardOldTxs()
	}
	t.sentTxLock.Unlock()
}

func (t *txAndReceipts) discardOldTxs() {
	for sender, senderTxs := range t.sentTxs {
		for txHash, sentTx := range senderTxs {
			if sentTx.lastSeen+maxTxAge < t.headCounter {
				delete(senderTxs, txHash)
			}
		}
		if len(senderTxs) == 0 {
			delete(t.sentTxs, sender)
		}
	}
}

func (t *txAndReceipts) allSenders() []common.Address {
	t.sentTxLock.Lock()
	allSenders := make([]common.Address, 0, len(t.sentTxs))
	for sender := range t.sentTxs {
		allSenders = append(allSenders, sender)
	}
	t.sentTxLock.Unlock()
	return allSenders
}

func (t *txAndReceipts) nonceAndPendingTxs(ctx context.Context, head *btypes.ExecutionHeader, sender common.Address) (uint64, types.Transactions, error) {
	proof, err := t.lightState.fetchProof(ctx, proofRequest{blockNumber: head.BlockNumber(), address: sender, storageKeys: ""})
	if err != nil {
		return 0, nil, err
	}
	if err := t.lightState.validateProof(proof, head.StateRoot(), sender, nil); err != nil {
		return 0, nil, err
	}

	t.sentTxLock.Lock()
	senderTxs := t.sentTxs[sender]
	t.sentTxLock.Unlock()

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
			tx, isPending, err := t.getUncachedTxByHash(ctx, txHash, head.BlockNumber())
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
	t.sentTxLock.Lock()
	for _, tx := range pendingList {
		t.markTxAsSeen(sender, tx)
	}
	t.sentTxLock.Unlock()
	return proof.Nonce, pendingList, nil
}
