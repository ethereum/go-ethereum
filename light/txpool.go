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

package light

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/core"
	"github.com/XinFinOrg/XDPoSChain/core/state"
	"github.com/XinFinOrg/XDPoSChain/core/txpool"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/ethdb"
	"github.com/XinFinOrg/XDPoSChain/event"
	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/XinFinOrg/XDPoSChain/params"
)

const (
	// chainHeadChanSize is the size of channel listening to ChainHeadEvent.
	chainHeadChanSize = 10
)

// txPermanent is the number of mined blocks after a mined transaction is
// considered permanent and no rollback is expected
var txPermanent = uint64(500)

// TxPool implements the transaction pool for light clients, which keeps track
// of the status of locally created transactions, detecting if they are included
// in a block (mined) or rolled back. There are no queued transactions since we
// always receive all locally signed transactions in the same order as they are
// created.
type TxPool struct {
	config       *params.ChainConfig
	signer       types.Signer
	quit         chan bool
	txFeed       event.Feed
	scope        event.SubscriptionScope
	chainHeadCh  chan core.ChainHeadEvent
	chainHeadSub event.Subscription
	mu           sync.RWMutex
	chain        *LightChain
	odr          OdrBackend
	chainDb      ethdb.Database
	relay        TxRelayBackend
	head         common.Hash
	nonce        map[common.Address]uint64            // "pending" nonce
	pending      map[common.Hash]*types.Transaction   // pending transactions by tx hash
	mined        map[common.Hash][]*types.Transaction // mined transactions by block hash
	clearIdx     uint64                               // earliest block nr that can contain mined tx info

	homestead bool
	eip2718   bool // Fork indicator whether we are in the eip2718 stage.
	eip1559   bool // Fork indicator whether we are in the eip1559 stage.
}

// TxRelayBackend provides an interface to the mechanism that forwards transacions
// to the ETH network. The implementations of the functions should be non-blocking.
//
// Send instructs backend to forward new transactions
// NewHead notifies backend about a new head after processed by the tx pool,
//
//	including  mined and rolled back transactions since the last event
//
// Discard notifies backend about transactions that should be discarded either
//
//	because they have been replaced by a re-send or because they have been mined
//	long ago and no rollback is expected
type TxRelayBackend interface {
	Send(txs types.Transactions)
	NewHead(head common.Hash, mined []common.Hash, rollback []common.Hash)
	Discard(hashes []common.Hash)
}

// NewTxPool creates a new light transaction pool
func NewTxPool(config *params.ChainConfig, chain *LightChain, relay TxRelayBackend) *TxPool {
	pool := &TxPool{
		config:      config,
		signer:      types.LatestSigner(config),
		nonce:       make(map[common.Address]uint64),
		pending:     make(map[common.Hash]*types.Transaction),
		mined:       make(map[common.Hash][]*types.Transaction),
		quit:        make(chan bool),
		chainHeadCh: make(chan core.ChainHeadEvent, chainHeadChanSize),
		chain:       chain,
		relay:       relay,
		odr:         chain.Odr(),
		chainDb:     chain.Odr().Database(),
		head:        chain.CurrentHeader().Hash(),
		clearIdx:    chain.CurrentHeader().Number.Uint64(),
	}
	// Subscribe events from blockchain
	pool.chainHeadSub = pool.chain.SubscribeChainHeadEvent(pool.chainHeadCh)
	go pool.eventLoop()

	return pool
}

// currentState returns the light state of the current head header
func (p *TxPool) currentState(ctx context.Context) *state.StateDB {
	return NewState(ctx, p.chain.CurrentHeader(), p.odr)
}

// GetNonce returns the "pending" nonce of a given address. It always queries
// the nonce belonging to the latest header too in order to detect if another
// client using the same key sent a transaction.
func (p *TxPool) GetNonce(ctx context.Context, addr common.Address) (uint64, error) {
	state := p.currentState(ctx)
	nonce := state.GetNonce(addr)
	if state.Error() != nil {
		return 0, state.Error()
	}
	sn, ok := p.nonce[addr]
	if ok && sn > nonce {
		nonce = sn
	}
	if !ok || sn < nonce {
		p.nonce[addr] = nonce
	}
	return nonce, nil
}

// txStateChanges stores the recent changes between pending/mined states of
// transactions. True means mined, false means rolled back, no entry means no change
type txStateChanges map[common.Hash]bool

// setState sets the status of a tx to either recently mined or recently rolled back
func (txc txStateChanges) setState(txHash common.Hash, mined bool) {
	val, ent := txc[txHash]
	if ent && (val != mined) {
		delete(txc, txHash)
	} else {
		txc[txHash] = mined
	}
}

// getLists creates lists of mined and rolled back tx hashes
func (txc txStateChanges) getLists() (mined []common.Hash, rollback []common.Hash) {
	for hash, val := range txc {
		if val {
			mined = append(mined, hash)
		} else {
			rollback = append(rollback, hash)
		}
	}
	return
}

// checkMinedTxs checks newly added blocks for the currently pending transactions
// and marks them as mined if necessary. It also stores block position in the db
// and adds them to the received txStateChanges map.
func (p *TxPool) checkMinedTxs(ctx context.Context, hash common.Hash, number uint64, txc txStateChanges) error {
	// If no transactions are pending, we don't care about anything
	if len(p.pending) == 0 {
		return nil
	}
	block, err := GetBlock(ctx, p.odr, hash, number)
	if err != nil {
		return err
	}
	// Gather all the local transaction mined in this block
	list := p.mined[hash]
	for _, tx := range block.Transactions() {
		if _, ok := p.pending[tx.Hash()]; ok {
			list = append(list, tx)
		}
	}
	// If some transactions have been mined, write the needed data to disk and update
	if list != nil {
		// Retrieve all the receipts belonging to this block and write the lookup table
		if _, err := GetBlockReceipts(ctx, p.odr, hash, number); err != nil { // ODR caches, ignore results
			return err
		}
		if err := core.WriteTxLookupEntries(p.chainDb, block); err != nil {
			return err
		}
		// Update the transaction pool's state
		for _, tx := range list {
			delete(p.pending, tx.Hash())
			txc.setState(tx.Hash(), true)
		}
		p.mined[hash] = list
	}
	return nil
}

// rollbackTxs marks the transactions contained in recently rolled back blocks
// as rolled back. It also removes any positional lookup entries.
func (p *TxPool) rollbackTxs(hash common.Hash, txc txStateChanges) {
	if list, ok := p.mined[hash]; ok {
		for _, tx := range list {
			txHash := tx.Hash()
			core.DeleteTxLookupEntry(p.chainDb, txHash)
			p.pending[txHash] = tx
			txc.setState(txHash, false)
		}
		delete(p.mined, hash)
	}
}

// reorgOnNewHead sets a new head header, processing (and rolling back if necessary)
// the blocks since the last known head and returns a txStateChanges map containing
// the recently mined and rolled back transaction hashes. If an error (context
// timeout) occurs during checking new blocks, it leaves the locally known head
// at the latest checked block and still returns a valid txStateChanges, making it
// possible to continue checking the missing blocks at the next chain head event
func (p *TxPool) reorgOnNewHead(ctx context.Context, newHeader *types.Header) (txStateChanges, error) {
	txc := make(txStateChanges)
	oldh := p.chain.GetHeaderByHash(p.head)
	newh := newHeader
	// find common ancestor, create list of rolled back and new block hashes
	var oldHashes, newHashes []common.Hash
	for oldh.Hash() != newh.Hash() {
		if oldh.Number.Uint64() >= newh.Number.Uint64() {
			oldHashes = append(oldHashes, oldh.Hash())
			oldh = p.chain.GetHeader(oldh.ParentHash, oldh.Number.Uint64()-1)
		}
		if oldh.Number.Uint64() < newh.Number.Uint64() {
			newHashes = append(newHashes, newh.Hash())
			newh = p.chain.GetHeader(newh.ParentHash, newh.Number.Uint64()-1)
			if newh == nil {
				// happens when CHT syncing, nothing to do
				newh = oldh
			}
		}
	}
	if oldh.Number.Uint64() < p.clearIdx {
		p.clearIdx = oldh.Number.Uint64()
	}
	// roll back old blocks
	for _, hash := range oldHashes {
		p.rollbackTxs(hash, txc)
	}
	p.head = oldh.Hash()
	// check mined txs of new blocks (array is in reversed order)
	for i := len(newHashes) - 1; i >= 0; i-- {
		hash := newHashes[i]
		if err := p.checkMinedTxs(ctx, hash, newHeader.Number.Uint64()-uint64(i), txc); err != nil {
			return txc, err
		}
		p.head = hash
	}

	// clear old mined tx entries of old blocks
	if idx := newHeader.Number.Uint64(); idx > p.clearIdx+txPermanent {
		idx2 := idx - txPermanent
		if len(p.mined) > 0 {
			for i := p.clearIdx; i < idx2; i++ {
				hash := core.GetCanonicalHash(p.chainDb, i)
				if list, ok := p.mined[hash]; ok {
					hashes := make([]common.Hash, len(list))
					for i, tx := range list {
						hashes[i] = tx.Hash()
					}
					p.relay.Discard(hashes)
					delete(p.mined, hash)
				}
			}
		}
		p.clearIdx = idx2
	}

	return txc, nil
}

// blockCheckTimeout is the time limit for checking new blocks for mined
// transactions. Checking resumes at the next chain head event if timed out.
const blockCheckTimeout = time.Second * 3

// eventLoop processes chain head events and also notifies the tx relay backend
// about the new head hash and tx state changes
func (p *TxPool) eventLoop() {
	for {
		select {
		case ev := <-p.chainHeadCh:
			p.setNewHead(ev.Block.Header())
			// hack in order to avoid hogging the lock; this part will
			// be replaced by a subsequent PR.
			time.Sleep(time.Millisecond)

		// System stopped
		case <-p.chainHeadSub.Err():
			return
		}
	}
}

func (p *TxPool) setNewHead(head *types.Header) {
	p.mu.Lock()
	defer p.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), blockCheckTimeout)
	defer cancel()

	txc, _ := p.reorgOnNewHead(ctx, head)
	m, r := txc.getLists()
	p.relay.NewHead(p.head, m, r)

	// Update fork indicator by next pending block number
	next := new(big.Int).Add(head.Number, big.NewInt(1))
	p.homestead = p.config.IsHomestead(head.Number)
	p.eip2718 = p.config.IsEIP1559(next)
	p.eip1559 = p.config.IsEIP1559(next)
}

// Stop stops the light transaction pool
func (p *TxPool) Stop() {
	// Unsubscribe all subscriptions registered from txpool
	p.scope.Close()
	// Unsubscribe subscriptions registered from blockchain
	p.chainHeadSub.Unsubscribe()
	close(p.quit)
	log.Info("Transaction pool stopped")
}

// SubscribeNewTxsEvent registers a subscription of core.NewTxsEvent and
// starts sending event to the given channel.
func (p *TxPool) SubscribeNewTxsEvent(ch chan<- core.NewTxsEvent) event.Subscription {
	return p.scope.Track(p.txFeed.Subscribe(ch))
}

// Stats returns the number of currently pending (locally created) transactions
func (p *TxPool) Stats() (pending int) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	pending = len(p.pending)
	return
}

// validateTx checks whether a transaction is valid according to the consensus rules.
func (p *TxPool) validateTx(ctx context.Context, tx *types.Transaction) error {
	// Validate sender
	var (
		from common.Address
		err  error
	)

	// check if sender is in black list
	if tx.From() != nil && common.Blacklist[*tx.From()] {
		return fmt.Errorf("reject transaction with sender in black-list: %v", tx.From().Hex())
	}
	// check if receiver is in black list
	if tx.To() != nil && common.Blacklist[*tx.To()] {
		return fmt.Errorf("reject transaction with receiver in black-list: %v", tx.To().Hex())
	}

	// validate minFee slot for XDCZ
	if tx.IsXDCZApplyTransaction() {
		copyState := p.currentState(ctx).Copy()
		if err := core.ValidateXDCZApplyTransaction(p.chain, nil, copyState, common.BytesToAddress(tx.Data()[4:])); err != nil {
			return err
		}
	}
	// validate balance slot, token decimal for XDCX
	if tx.IsXDCXApplyTransaction() {
		copyState := p.currentState(ctx).Copy()
		if err := core.ValidateXDCXApplyTransaction(p.chain, nil, copyState, common.BytesToAddress(tx.Data()[4:])); err != nil {
			return err
		}
	}

	// Validate the transaction sender and it's sig. Throw
	// if the from fields is invalid.
	if from, err = types.Sender(p.signer, tx); err != nil {
		return txpool.ErrInvalidSender
	}
	// Last but not least check for nonce errors
	currentState := p.currentState(ctx)
	if n := currentState.GetNonce(from); n > tx.Nonce() {
		return core.ErrNonceTooLow
	}

	// Check the transaction doesn't exceed the current
	// block limit gas.
	header := p.chain.GetHeaderByHash(p.head)
	if header.GasLimit < tx.Gas() {
		return txpool.ErrGasLimit
	}

	// Transactions can't be negative. This may never happen
	// using RLP decoded transactions but may occur if you create
	// a transaction using the RPC for example.
	if tx.Value().Sign() < 0 {
		return txpool.ErrNegativeValue
	}

	// Transactor should have enough funds to cover the costs
	// cost == V + GP * GL
	if b := currentState.GetBalance(from); b.Cmp(tx.Cost()) < 0 {
		return core.ErrInsufficientFunds
	}

	// Should supply enough intrinsic gas
	gas, err := core.IntrinsicGas(tx.Data(), tx.AccessList(), tx.To() == nil, p.homestead, p.eip1559)
	if err != nil {
		return err
	}
	if tx.Gas() < gas {
		return core.ErrIntrinsicGas
	}
	return currentState.Error()
}

// add validates a new transaction and sets its state pending if processable.
// It also updates the locally stored nonce if necessary.
func (p *TxPool) add(ctx context.Context, tx *types.Transaction) error {
	hash := tx.Hash()

	if p.pending[hash] != nil {
		return fmt.Errorf("known transaction (%x)", hash[:4])
	}
	err := p.validateTx(ctx, tx)
	if err != nil {
		return err
	}

	if _, ok := p.pending[hash]; !ok {
		p.pending[hash] = tx

		nonce := tx.Nonce() + 1

		addr, _ := types.Sender(p.signer, tx)
		if nonce > p.nonce[addr] {
			p.nonce[addr] = nonce
		}

		// Notify the subscribers. This event is posted in a goroutine
		// because it's possible that somewhere during the post "Remove transaction"
		// gets called which will then wait for the global tx pool lock and deadlock.
		go p.txFeed.Send(core.NewTxsEvent{Txs: types.Transactions{tx}})
	}

	// Print a log message if low enough level is set
	if log.Enabled(log.LevelDebug) {
		from, _ := types.Sender(p.signer, tx)
		log.Debug("Pooled new transaction", "hash", hash, "from", from, "to", tx.To())
	}
	return nil
}

// Add adds a transaction to the pool if valid and passes it to the tx relay
// backend
func (p *TxPool) Add(ctx context.Context, tx *types.Transaction) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	data, err := tx.MarshalBinary()
	if err != nil {
		return err
	}

	if err := p.add(ctx, tx); err != nil {
		return err
	}
	//fmt.Println("Send", tx.Hash())
	p.relay.Send(types.Transactions{tx})

	p.chainDb.Put(tx.Hash().Bytes(), data)
	return nil
}

// AddTransactions adds all valid transactions to the pool and passes them to
// the tx relay backend
func (p *TxPool) AddBatch(ctx context.Context, txs []*types.Transaction) {
	p.mu.Lock()
	defer p.mu.Unlock()
	var sendTx types.Transactions

	for _, tx := range txs {
		if err := p.add(ctx, tx); err == nil {
			sendTx = append(sendTx, tx)
		}
	}
	if len(sendTx) > 0 {
		p.relay.Send(sendTx)
	}
}

// GetTransaction returns a transaction if it is contained in the pool
// and nil otherwise.
func (p *TxPool) GetTransaction(hash common.Hash) *types.Transaction {
	// check the txs first
	if tx, ok := p.pending[hash]; ok {
		return tx
	}
	return nil
}

// GetTransactions returns all currently processable transactions.
// The returned slice may be modified by the caller.
func (p *TxPool) GetTransactions() (txs types.Transactions, err error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	txs = make(types.Transactions, len(p.pending))
	i := 0
	for _, tx := range p.pending {
		txs[i] = tx
		i++
	}
	return txs, nil
}

// Content retrieves the data content of the transaction pool, returning all the
// pending as well as queued transactions, grouped by account and nonce.
func (p *TxPool) Content() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Retrieve all the pending transactions and sort by account and by nonce
	pending := make(map[common.Address]types.Transactions)
	for _, tx := range p.pending {
		account, _ := types.Sender(p.signer, tx)
		pending[account] = append(pending[account], tx)
	}
	// There are no queued transactions in a light pool, just return an empty map
	queued := make(map[common.Address]types.Transactions)
	return pending, queued
}

// ContentFrom retrieves the data content of the transaction pool, returning the
// pending as well as queued transactions of this address, grouped by nonce.
func (pool *TxPool) ContentFrom(addr common.Address) (types.Transactions, types.Transactions) {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	// Retrieve the pending transactions and sort by nonce
	var pending types.Transactions
	for _, tx := range pool.pending {
		account, _ := types.Sender(pool.signer, tx)
		if account != addr {
			continue
		}
		pending = append(pending, tx)
	}
	// There are no queued transactions in a light pool, just return an empty map
	return pending, types.Transactions{}
}

// RemoveTransactions removes all given transactions from the pool.
func (p *TxPool) RemoveTransactions(txs types.Transactions) {
	p.mu.Lock()
	defer p.mu.Unlock()
	var hashes []common.Hash
	for _, tx := range txs {
		//self.RemoveTx(tx.Hash())
		hash := tx.Hash()
		delete(p.pending, hash)
		p.chainDb.Delete(hash[:])
		hashes = append(hashes, hash)
	}
	p.relay.Discard(hashes)
}

// RemoveTx removes the transaction with the given hash from the pool.
func (p *TxPool) RemoveTx(hash common.Hash) {
	p.mu.Lock()
	defer p.mu.Unlock()
	// delete from pending pool
	delete(p.pending, hash)
	p.chainDb.Delete(hash[:])
	p.relay.Discard([]common.Hash{hash})
}
