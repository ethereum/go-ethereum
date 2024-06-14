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
	"math/rand"
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

// txAndReceiptsFields defines Client fields related to transactions and receipts.
type txAndReceiptsFields struct {
	signer types.Signer

	blockReceiptsCache    *lru.Cache[common.Hash, types.Receipts]
	blockReceiptsRequests *requestMap[common.Hash, types.Receipts]
	txPosCache            *lru.Cache[common.Hash, txInBlock]
	txByHashRequests      *requestMap[common.Hash, txResult]
	receiptByHashRequests *requestMap[common.Hash, *types.Receipt]

	trackedTxLock               sync.Mutex
	trackedTxs                  map[common.Address]senderTxs
	headCounter, lastHeadNumber uint64
}

// txResult represents the results of a transaction by hash request.
type txResult struct {
	tx        *types.Transaction
	isPending bool
}

// txInBlock stores the known inclusion positions of a transaction.
// Note that any cached position is only considered valid if the block is canonical.
type txInBlock struct {
	blockNumber uint64
	blockHash   common.Hash
	index       uint
}

// trackedTx stores the information associated with the hashes of tracked transactions.
type trackedTx struct {
	nonce    uint64
	lastSeen uint64 // headCounter value where the tx has been sent or last seen pending
}

// senderTxs stores all tracked transactions originating from a single sender.
type senderTxs map[common.Hash]trackedTx

// initTxAndReceipts initializes the structures related to transactions and receipts.
func (c *Client) initTxAndReceipts() {
	c.blockReceiptsCache = lru.NewCache[common.Hash, types.Receipts](10)
	c.txPosCache = lru.NewCache[common.Hash, txInBlock](10000)
	c.trackedTxs = make(map[common.Address]senderTxs)
	c.signer = types.LatestSigner(c.elConfig)
	c.blockReceiptsRequests = newRequestMap[common.Hash, types.Receipts](c.requestBlockReceipts)
	c.txByHashRequests = newRequestMap[common.Hash, txResult](c.requestTxByHash)
	c.receiptByHashRequests = newRequestMap[common.Hash, *types.Receipt](c.requestReceiptByTxHash)
}

// closeTxAndReceipts shuts down the structures related to transactions and receipts.
func (c *Client) closeTxAndReceipts() {
	c.blockReceiptsRequests.close()
	c.txByHashRequests.close()
	c.receiptByHashRequests.close()
}

// getTxByHash requests and validates the transaction with the specified hash or
// returns it from cache if available as part of the local canonical chain.
// The pending status of the transaction is also returned. Note that the pending
// status cannot be directly validated, unless it is known to be canonical.
// Also note that if a transaction belongs to the server's latest block that is
// not proven by an optimistic update to the beacon light client yet then this
// function will report isPending == true even though the server has reported
// it as non-pending. This behavior ensures the consistency between the local
// canonical chain and the results of getTxByHash and getReceiptByTxHash.
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

// getUncachedTxByHash requests a transaction and its pending status without
// checking in the cache whether it's already canonical. It does check for future
// inclusion and returns isPending == true if it has a receipt in a block over
// headBlockNumber.
// The purpose of this function is sharing code between getTxByHash and
// nonceAndPendingTxs.
func (c *Client) getUncachedTxByHash(ctx context.Context, txHash common.Hash, headBlockNumber uint64) (tx *types.Transaction, isPending bool, err error) {
	req := c.txByHashRequests.request(txHash)
	var txResult txResult
	txResult, err = req.waitForResult(ctx)
	req.release()
	tx, isPending = txResult.tx, txResult.isPending
	if err == nil && !isPending {
		req := c.receiptByHashRequests.request(txHash)
		receipt, err := req.waitForResult(ctx)
		req.release()
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

// getReceiptByTxHash requests and validates the receipt belonging to the transaction
// with the specified hash or returns it from cache if available. Results are only
// returned if they can be validated as part of the current local canonical chain.
// Note that if a receipt belongs to the server's latest block that is not proven
// by an optimistic update to the beacon light client yet then this function cannot
// validate it and will treat it like a pending transaction and return ethereum.NotFound.
func (c *Client) getReceiptByTxHash(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	if pos, ok := c.txPosCache.Get(txHash); ok {
		if hash, ok := c.getCachedHash(pos.blockNumber); ok && hash == pos.blockHash {
			if receipts, ok := c.blockReceiptsCache.Get(pos.blockHash); ok {
				if pos.index >= uint(len(receipts)) {
					return nil, errors.New("transaction index out of range")
				}
				return receipts[pos.index], nil
			}
		}
	}
	req := c.receiptByHashRequests.request(txHash)
	receipt, err := req.waitForResult(ctx)
	req.release()
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

// getBlockReceipts requests and validates a set of block receipts or returns them
// from cache if available.
func (c *Client) getBlockReceipts(ctx context.Context, blockHash common.Hash) (types.Receipts, error) {
	if receipts, ok := c.blockReceiptsCache.Get(blockHash); ok {
		return receipts, nil
	}
	request := c.blockReceiptsRequests.request(blockHash)
	block, err := c.getBlock(ctx, blockHash)
	if err != nil {
		return nil, err
	}
	receipts, err := request.waitForResult(ctx)
	if err == nil {
		receipts, err = c.validateBlockReceipts(block, receipts)
	}
	if err == nil {
		c.blockReceiptsCache.Add(blockHash, receipts)
	}
	request.release()
	return receipts, err
}

// validateBlockReceipts verifies that the given set of receipts belongs to the
// given block. Non-consensus fields of the returned set of receipts are derived
// from the block and the consensus fields of the received receipts which are also
// validated against the receipts root of the block.
// Note that the non-consensus fields of the received receipts could be ignored
// but they are also verified against the derived version in order to detect any
// potential inconsistencies between the server responses and the validation logic.
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
	// Note: this step could be skipped but it is performed as a sanity check.
	jsonReceived, err := json.Marshal(receipts)
	if err != nil {
		return nil, err
	}
	jsonDerived, err := json.Marshal(newReceipts)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(jsonReceived, jsonDerived) {
		return nil, errors.New("received and derived receipts do not match")
	}
	return newReceipts, nil
}

// requestTxByHash requests the transaction belonging to the specified hash from
// the RPC client, along with the current pending status.
// Either a transaction with the specified hash or an error is returned. Note that
// the pending status cannot be directly validated because the light client does
// not track the set of pending transactions.
func (c *Client) requestTxByHash(ctx context.Context, hash common.Hash) (txResult, error) {
	var json *rpcTransaction
	err := c.client.CallContext(ctx, &json, "eth_getTransactionByHash", hash)
	if err != nil {
		return txResult{}, err
	} else if json == nil {
		return txResult{}, ethereum.NotFound
	} else if _, r, _ := json.tx.RawSignatureValues(); r == nil {
		return txResult{}, errors.New("server returned transaction without signature")
	} else if json.tx.Hash() != hash {
		return txResult{}, errors.New("invalid transaction hash")
	}
	if json.From != nil && json.BlockHash != nil {
		setSenderFromServer(json.tx, *json.From, *json.BlockHash)
	}
	return txResult{tx: json.tx, isPending: json.BlockNumber == nil}, nil
}

// requestReceiptByTxHash requests the receipt belonging to the transaction with
// the specified hash from the RPC client.
// Either a receipt or an error is returned. Note that the results are not validated
// at this level as it also requires the block in which the transaction is included.
// Also note that since the receipt depends on the inclusion block, the hash to
// receipt association also depends on the current canonical chain.
func (c *Client) requestReceiptByTxHash(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	var r *types.Receipt
	err := c.client.CallContext(ctx, &r, "eth_getTransactionReceipt", txHash)
	if err == nil && r == nil {
		return nil, ethereum.NotFound
	}
	return r, err
}

// requestBlockReceipts requests the set of receipts belonging to the block with
// the specified hash from the RPC client.
// Either a set of receipts or an error is returned. Note that the results are not
// validated at this level as it also requires the belonging block.
func (c *Client) requestBlockReceipts(ctx context.Context, blockHash common.Hash) (types.Receipts, error) {
	var r []*types.Receipt
	err := c.client.CallContext(ctx, &r, "eth_getBlockReceipts", blockHash)
	if err == nil && r == nil {
		return nil, ethereum.NotFound
	}
	return types.Receipts(r), err
}

// cacheBlockTxPositions iterates through the transactions of a block and caches
// transaction position information. This function should be called for all
// new canonical blocks.
func (c *Client) cacheBlockTxPositions(block *types.Block) {
	blockNumber, blockHash := block.NumberU64(), block.Hash()
	for i, tx := range block.Transactions() {
		c.txPosCache.Add(tx.Hash(), txInBlock{blockNumber: blockNumber, blockHash: blockHash, index: uint(i)})
	}
}

// sendTransaction sends a transaction to the RPC server and adds it to the set of
// tracked transactions.
func (c *Client) sendTransaction(ctx context.Context, tx *types.Transaction) error {
	data, err := tx.MarshalBinary()
	if err != nil {
		return err
	}
	if err := c.client.CallContext(ctx, nil, "eth_sendRawTransaction", hexutil.Encode(data)); err != nil {
		return err
	}
	return c.TrackTransaction(tx)
}

func (c *Client) TrackTransaction(tx *types.Transaction) error {
	sender, err := types.Sender(c.signer, tx)
	if err != nil {
		return err
	}
	c.trackedTxLock.Lock()
	c.markTxAsSeen(sender, tx)
	c.trackedTxLock.Unlock()
	return nil
}

// nonceAndPendingTxs obtains the latest nonce of the given sender account and checks
// the pending status of the tracked transactions originating from that server.
// It returns the current nonce and a nonce-ordered list of pending transactions.
func (c *Client) nonceAndPendingTxs(ctx context.Context, head *btypes.ExecutionHeader, sender common.Address) (uint64, types.Transactions, error) {
	proof, err := c.fetchProof(ctx, proofRequest{blockNumber: head.BlockNumber(), address: sender, storageKeys: ""})
	if err != nil {
		return 0, nil, err
	}
	if err := c.validateProof(proof, head.StateRoot(), sender, nil); err != nil {
		return 0, nil, err
	}

	c.trackedTxLock.Lock()
	senderTxs := c.trackedTxs[sender]
	c.trackedTxLock.Unlock()

	type pendingResult struct {
		pendingTx *types.Transaction
		err       error
	}

	resultCh := make(chan pendingResult, len(senderTxs))
	var reqCount int
	for txHash, trackedTx := range senderTxs {
		if trackedTx.nonce < proof.Nonce {
			continue
		}
		reqCount++
		txHash := txHash
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
	c.trackedTxLock.Lock()
	for _, tx := range pendingList {
		c.markTxAsSeen(sender, tx)
	}
	c.trackedTxLock.Unlock()
	return proof.Nonce, pendingList, nil
}

// allSenders returns the list of all senders that have belonging tracked transactions.
func (c *Client) allSenders() []common.Address {
	c.trackedTxLock.Lock()
	allSenders := make([]common.Address, 0, len(c.trackedTxs))
	for sender := range c.trackedTxs {
		allSenders = append(allSenders, sender)
	}
	c.trackedTxLock.Unlock()
	return allSenders
}

// txAndReceiptsNewHead should be called for every new head. It counts the head
// updates that increase the current block height and discards transactions that
// have not been seen as pending lately based on this head counter.
func (c *Client) txAndReceiptsNewHead(number uint64, hash common.Hash) {
	c.trackedTxLock.Lock()
	if number > c.lastHeadNumber {
		c.lastHeadNumber = number
		c.headCounter++
		c.discardOldTxs()
	}
	c.trackedTxLock.Unlock()
}

// discardOldTxs discards transactions that have not been seen as pending lately.
func (c *Client) discardOldTxs() {
	for sender, senderTxs := range c.trackedTxs {
		for txHash, trackedTx := range senderTxs {
			if trackedTx.lastSeen+maxTxAge < c.headCounter {
				delete(senderTxs, txHash)
			}
		}
		if len(senderTxs) == 0 {
			delete(c.trackedTxs, sender)
		}
	}
}

// markTxAsSeen adds a transaction to the tracked set or updates its last seen
// counter in order to keep it in the set.
func (c *Client) markTxAsSeen(sender common.Address, tx *types.Transaction) {
	senderTxs := c.trackedTxs[sender]
	if senderTxs == nil {
		senderTxs = make(map[common.Hash]trackedTx)
		c.trackedTxs[sender] = senderTxs
	}
	senderTxs[tx.Hash()] = trackedTx{nonce: tx.Nonce(), lastSeen: c.headCounter}
}

func (c *Client) RandomPendingTxs(ctx context.Context, count int) (types.Transactions, error) {
	var txc hexutil.Uint
	err := c.client.CallContext(ctx, &txc, "eth_getBlockTransactionCountByNumber", "pending")
	if err != nil {
		return nil, err
	}
	txCount := int(txc)
	if txCount == 0 {
		return nil, errors.New("no pending transactions")
	}
	txs := make(types.Transactions, count)
	for i := range txs {
		var json *rpcTransaction
		index := rand.Intn(txCount)
		err := c.client.CallContext(ctx, &json, "eth_getTransactionByBlockNumberAndIndex", "pending", hexutil.Uint64(index))
		if err != nil {
			return nil, err
		} else if json == nil {
			return nil, ethereum.NotFound
		} else if _, r, _ := json.tx.RawSignatureValues(); r == nil {
			return nil, errors.New("server returned transaction without signature")
		}
		if json.From != nil && json.BlockHash != nil {
			setSenderFromServer(json.tx, *json.From, *json.BlockHash)
		}
		txs[i] = json.tx
	}
	return txs, nil
}
