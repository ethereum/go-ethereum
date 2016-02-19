// Copyright 2014 The go-ethereum Authors
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

package core

import (
	"errors"
	"fmt"
	"math/big"
	"sort"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/params"
)

var (
	// Transaction Pool Errors
	ErrInvalidSender      = errors.New("Invalid sender")
	ErrNonce              = errors.New("Nonce too low")
	ErrCheap              = errors.New("Gas price too low for acceptance")
	ErrBalance            = errors.New("Insufficient balance")
	ErrNonExistentAccount = errors.New("Account does not exist or account balance too low")
	ErrInsufficientFunds  = errors.New("Insufficient funds for gas * price + value")
	ErrIntrinsicGas       = errors.New("Intrinsic gas too low")
	ErrGasLimit           = errors.New("Exceeds block gas limit")
	ErrNegativeValue      = errors.New("Negative value")
)

const (
	maxQueued = 64 // max limit of queued txs per address
)

type stateFn func() (*state.StateDB, error)

// TxPool contains all currently known transactions. Transactions
// enter the pool when they are received from the network or submitted
// locally. They exit the pool when they are included in the blockchain.
//
// The pool separates processable transactions (which can be applied to the
// current state) and future transactions. Transactions move between those
// two states over time as they are received and processed.
type TxPool struct {
	quit         chan bool // Quiting channel
	currentState stateFn   // The state function which will allow us to do some pre checkes
	pendingState *state.ManagedState
	gasLimit     func() *big.Int // The current gas limit function callback
	minGasPrice  *big.Int
	eventMux     *event.TypeMux
	events       event.Subscription
	localTx      *txSet
	mu           sync.RWMutex
	pending      map[common.Hash]*types.Transaction // processable transactions
	queue        map[common.Address]map[common.Hash]*types.Transaction

	homestead bool
}

func NewTxPool(eventMux *event.TypeMux, currentStateFn stateFn, gasLimitFn func() *big.Int) *TxPool {
	pool := &TxPool{
		pending:      make(map[common.Hash]*types.Transaction),
		queue:        make(map[common.Address]map[common.Hash]*types.Transaction),
		quit:         make(chan bool),
		eventMux:     eventMux,
		currentState: currentStateFn,
		gasLimit:     gasLimitFn,
		minGasPrice:  new(big.Int),
		pendingState: nil,
		localTx:      newTxSet(),
		events:       eventMux.Subscribe(ChainHeadEvent{}, GasPriceChanged{}, RemovedTransactionEvent{}),
	}

	go pool.eventLoop()

	return pool
}

func (pool *TxPool) eventLoop() {
	// Track chain events. When a chain events occurs (new chain canon block)
	// we need to know the new state. The new state will help us determine
	// the nonces in the managed state
	for ev := range pool.events.Chan() {
		switch ev := ev.Data.(type) {
		case ChainHeadEvent:
			pool.mu.Lock()
			if ev.Block != nil && params.IsHomestead(ev.Block.Number()) {
				pool.homestead = true
			}

			pool.resetState()
			pool.mu.Unlock()
		case GasPriceChanged:
			pool.mu.Lock()
			pool.minGasPrice = ev.Price
			pool.mu.Unlock()
		case RemovedTransactionEvent:
			pool.AddTransactions(ev.Txs)
		}
	}
}

func (pool *TxPool) resetState() {
	currentState, err := pool.currentState()
	if err != nil {
		glog.V(logger.Info).Infoln("failed to get current state: %v", err)
		return
	}
	managedState := state.ManageState(currentState)
	if err != nil {
		glog.V(logger.Info).Infoln("failed to get managed state: %v", err)
		return
	}
	pool.pendingState = managedState

	// validate the pool of pending transactions, this will remove
	// any transactions that have been included in the block or
	// have been invalidated because of another transaction (e.g.
	// higher gas price)
	pool.validatePool()

	// Loop over the pending transactions and base the nonce of the new
	// pending transaction set.
	for _, tx := range pool.pending {
		if addr, err := tx.From(); err == nil {
			// Set the nonce. Transaction nonce can never be lower
			// than the state nonce; validatePool took care of that.
			if pool.pendingState.GetNonce(addr) <= tx.Nonce() {
				pool.pendingState.SetNonce(addr, tx.Nonce()+1)
			}
		}
	}
	// Check the queue and move transactions over to the pending if possible
	// or remove those that have become invalid
	pool.checkQueue()
}

func (pool *TxPool) Stop() {
	close(pool.quit)
	pool.events.Unsubscribe()
	glog.V(logger.Info).Infoln("Transaction pool stopped")
}

func (pool *TxPool) State() *state.ManagedState {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	return pool.pendingState
}

func (pool *TxPool) Stats() (pending int, queued int) {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	pending = len(pool.pending)
	for _, txs := range pool.queue {
		queued += len(txs)
	}
	return
}

// Content retrieves the data content of the transaction pool, returning all the
// pending as well as queued transactions, grouped by account and nonce.
func (pool *TxPool) Content() (map[common.Address]map[uint64][]*types.Transaction, map[common.Address]map[uint64][]*types.Transaction) {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	// Retrieve all the pending transactions and sort by account and by nonce
	pending := make(map[common.Address]map[uint64][]*types.Transaction)
	for _, tx := range pool.pending {
		account, _ := tx.From()

		owned, ok := pending[account]
		if !ok {
			owned = make(map[uint64][]*types.Transaction)
			pending[account] = owned
		}
		owned[tx.Nonce()] = append(owned[tx.Nonce()], tx)
	}
	// Retrieve all the queued transactions and sort by account and by nonce
	queued := make(map[common.Address]map[uint64][]*types.Transaction)
	for account, txs := range pool.queue {
		owned := make(map[uint64][]*types.Transaction)
		for _, tx := range txs {
			owned[tx.Nonce()] = append(owned[tx.Nonce()], tx)
		}
		queued[account] = owned
	}
	return pending, queued
}

// SetLocal marks a transaction as local, skipping gas price
//  check against local miner minimum in the future
func (pool *TxPool) SetLocal(tx *types.Transaction) {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	pool.localTx.add(tx.Hash())
}

// validateTx checks whether a transaction is valid according
// to the consensus rules.
func (pool *TxPool) validateTx(tx *types.Transaction) error {
	local := pool.localTx.contains(tx.Hash())
	// Drop transactions under our own minimal accepted gas price
	if !local && pool.minGasPrice.Cmp(tx.GasPrice()) > 0 {
		return ErrCheap
	}

	currentState, err := pool.currentState()
	if err != nil {
		return err
	}

	from, err := tx.From()
	if err != nil {
		return ErrInvalidSender
	}

	// Make sure the account exist. Non existent accounts
	// haven't got funds and well therefor never pass.
	if !currentState.HasAccount(from) {
		return ErrNonExistentAccount
	}

	// Last but not least check for nonce errors
	if currentState.GetNonce(from) > tx.Nonce() {
		return ErrNonce
	}

	// Check the transaction doesn't exceed the current
	// block limit gas.
	if pool.gasLimit().Cmp(tx.Gas()) < 0 {
		return ErrGasLimit
	}

	// Transactions can't be negative. This may never happen
	// using RLP decoded transactions but may occur if you create
	// a transaction using the RPC for example.
	if tx.Value().Cmp(common.Big0) < 0 {
		return ErrNegativeValue
	}

	// Transactor should have enough funds to cover the costs
	// cost == V + GP * GL
	if currentState.GetBalance(from).Cmp(tx.Cost()) < 0 {
		return ErrInsufficientFunds
	}

	intrGas := IntrinsicGas(tx.Data(), MessageCreatesContract(tx), pool.homestead)
	if tx.Gas().Cmp(intrGas) < 0 {
		return ErrIntrinsicGas
	}

	return nil
}

// validate and queue transactions.
func (self *TxPool) add(tx *types.Transaction) error {
	hash := tx.Hash()

	if self.pending[hash] != nil {
		return fmt.Errorf("Known transaction (%x)", hash[:4])
	}
	err := self.validateTx(tx)
	if err != nil {
		return err
	}
	self.queueTx(hash, tx)

	if glog.V(logger.Debug) {
		var toname string
		if to := tx.To(); to != nil {
			toname = common.Bytes2Hex(to[:4])
		} else {
			toname = "[NEW_CONTRACT]"
		}
		// we can ignore the error here because From is
		// verified in ValidateTransaction.
		f, _ := tx.From()
		from := common.Bytes2Hex(f[:4])
		glog.Infof("(t) %x => %s (%v) %x\n", from, toname, tx.Value, hash)
	}

	return nil
}

// queueTx will queue an unknown transaction
func (self *TxPool) queueTx(hash common.Hash, tx *types.Transaction) {
	from, _ := tx.From() // already validated
	if self.queue[from] == nil {
		self.queue[from] = make(map[common.Hash]*types.Transaction)
	}
	self.queue[from][hash] = tx
}

// addTx will add a transaction to the pending (processable queue) list of transactions
func (pool *TxPool) addTx(hash common.Hash, addr common.Address, tx *types.Transaction) {
	// init delayed since tx pool could have been started before any state sync
	if pool.pendingState == nil {
		pool.resetState()
	}

	if _, ok := pool.pending[hash]; !ok {
		pool.pending[hash] = tx

		// Increment the nonce on the pending state. This can only happen if
		// the nonce is +1 to the previous one.
		pool.pendingState.SetNonce(addr, tx.Nonce()+1)
		// Notify the subscribers. This event is posted in a goroutine
		// because it's possible that somewhere during the post "Remove transaction"
		// gets called which will then wait for the global tx pool lock and deadlock.
		go pool.eventMux.Post(TxPreEvent{tx})
	}
}

// Add queues a single transaction in the pool if it is valid.
func (self *TxPool) Add(tx *types.Transaction) error {
	self.mu.Lock()
	defer self.mu.Unlock()

	if err := self.add(tx); err != nil {
		return err
	}
	self.checkQueue()
	return nil
}

// AddTransactions attempts to queue all valid transactions in txs.
func (self *TxPool) AddTransactions(txs []*types.Transaction) {
	self.mu.Lock()
	defer self.mu.Unlock()

	for _, tx := range txs {
		if err := self.add(tx); err != nil {
			glog.V(logger.Debug).Infoln("tx error:", err)
		} else {
			h := tx.Hash()
			glog.V(logger.Debug).Infof("tx %x\n", h[:4])
		}
	}

	// check and validate the queueue
	self.checkQueue()
}

// GetTransaction returns a transaction if it is contained in the pool
// and nil otherwise.
func (tp *TxPool) GetTransaction(hash common.Hash) *types.Transaction {
	// check the txs first
	if tx, ok := tp.pending[hash]; ok {
		return tx
	}
	// check queue
	for _, txs := range tp.queue {
		if tx, ok := txs[hash]; ok {
			return tx
		}
	}
	return nil
}

// GetTransactions returns all currently processable transactions.
// The returned slice may be modified by the caller.
func (self *TxPool) GetTransactions() (txs types.Transactions) {
	self.mu.Lock()
	defer self.mu.Unlock()

	// check queue first
	self.checkQueue()
	// invalidate any txs
	self.validatePool()

	txs = make(types.Transactions, len(self.pending))
	i := 0
	for _, tx := range self.pending {
		txs[i] = tx
		i++
	}
	return txs
}

// GetQueuedTransactions returns all non-processable transactions.
func (self *TxPool) GetQueuedTransactions() types.Transactions {
	self.mu.RLock()
	defer self.mu.RUnlock()

	var ret types.Transactions
	for _, txs := range self.queue {
		for _, tx := range txs {
			ret = append(ret, tx)
		}
	}
	sort.Sort(types.TxByNonce(ret))
	return ret
}

// RemoveTransactions removes all given transactions from the pool.
func (self *TxPool) RemoveTransactions(txs types.Transactions) {
	self.mu.Lock()
	defer self.mu.Unlock()
	for _, tx := range txs {
		self.RemoveTx(tx.Hash())
	}
}

// RemoveTx removes the transaction with the given hash from the pool.
func (pool *TxPool) RemoveTx(hash common.Hash) {
	// delete from pending pool
	delete(pool.pending, hash)
	// delete from queue
	for address, txs := range pool.queue {
		if _, ok := txs[hash]; ok {
			if len(txs) == 1 {
				// if only one tx, remove entire address entry.
				delete(pool.queue, address)
			} else {
				delete(txs, hash)
			}
			break
		}
	}
}

// checkQueue moves transactions that have become processable to main pool.
func (pool *TxPool) checkQueue() {
	// init delayed since tx pool could have been started before any state sync
	if pool.pendingState == nil {
		pool.resetState()
	}

	var promote txQueue
	for address, txs := range pool.queue {
		currentState, err := pool.currentState()
		if err != nil {
			glog.Errorf("could not get current state: %v", err)
			return
		}
		balance := currentState.GetBalance(address)

		var (
			guessedNonce = pool.pendingState.GetNonce(address) // nonce currently kept by the tx pool (pending state)
			trueNonce    = currentState.GetNonce(address)      // nonce known by the last state
		)
		promote = promote[:0]
		for hash, tx := range txs {
			// Drop processed or out of fund transactions
			if tx.Nonce() < trueNonce || balance.Cmp(tx.Cost()) < 0 {
				if glog.V(logger.Core) {
					glog.Infof("removed tx (%v) from pool queue: low tx nonce or out of funds\n", tx)
				}
				delete(txs, hash)
				continue
			}
			// Collect the remaining transactions for the next pass.
			promote = append(promote, txQueueEntry{hash, address, tx})
		}
		// Find the next consecutive nonce range starting at the current account nonce,
		// pushing the guessed nonce forward if we add consecutive transactions.
		sort.Sort(promote)
		for i, entry := range promote {
			// If we reached a gap in the nonces, enforce transaction limit and stop
			if entry.Nonce() > guessedNonce {
				if len(promote)-i > maxQueued {
					if glog.V(logger.Debug) {
						glog.Infof("Queued tx limit exceeded for %s. Tx %s removed\n", common.PP(address[:]), common.PP(entry.hash[:]))
					}
					for _, drop := range promote[i+maxQueued:] {
						delete(txs, drop.hash)
					}
				}
				break
			}
			// Otherwise promote the transaction and move the guess nonce if needed
			pool.addTx(entry.hash, address, entry.Transaction)
			delete(txs, entry.hash)

			if entry.Nonce() == guessedNonce {
				guessedNonce++
			}
		}
		// Delete the entire queue entry if it became empty.
		if len(txs) == 0 {
			delete(pool.queue, address)
		}
	}
}

// validatePool removes invalid and processed transactions from the main pool.
// If a transaction is removed for being invalid (e.g. out of funds), all sub-
// sequent (Still valid) transactions are moved back into the future queue. This
// is important to prevent a drained account from DOSing the network with non
// executable transactions.
func (pool *TxPool) validatePool() {
	state, err := pool.currentState()
	if err != nil {
		glog.V(logger.Info).Infoln("failed to get current state: %v", err)
		return
	}
	balanceCache := make(map[common.Address]*big.Int)

	// Clean up the pending pool, accumulating invalid nonces
	gaps := make(map[common.Address]uint64)

	for hash, tx := range pool.pending {
		sender, _ := tx.From() // err already checked

		// Perform light nonce and balance validation
		balance := balanceCache[sender]
		if balance == nil {
			balance = state.GetBalance(sender)
			balanceCache[sender] = balance
		}
		if past := state.GetNonce(sender) > tx.Nonce(); past || balance.Cmp(tx.Cost()) < 0 {
			// Remove an already past it invalidated transaction
			if glog.V(logger.Core) {
				glog.Infof("removed tx (%v) from pool: low tx nonce or out of funds\n", tx)
			}
			delete(pool.pending, hash)

			// Track the smallest invalid nonce to postpone subsequent transactions
			if !past {
				if prev, ok := gaps[sender]; !ok || tx.Nonce() < prev {
					gaps[sender] = tx.Nonce()
				}
			}
		}
	}
	// Move all transactions after a gap back to the future queue
	if len(gaps) > 0 {
		for hash, tx := range pool.pending {
			sender, _ := tx.From()
			if gap, ok := gaps[sender]; ok && tx.Nonce() >= gap {
				if glog.V(logger.Core) {
					glog.Infof("postponed tx (%v) due to introduced gap\n", tx)
				}
				pool.queueTx(hash, tx)
				delete(pool.pending, hash)
			}
		}
	}
}

type txQueue []txQueueEntry

type txQueueEntry struct {
	hash common.Hash
	addr common.Address
	*types.Transaction
}

func (q txQueue) Len() int           { return len(q) }
func (q txQueue) Swap(i, j int)      { q[i], q[j] = q[j], q[i] }
func (q txQueue) Less(i, j int) bool { return q[i].Nonce() < q[j].Nonce() }

// txSet represents a set of transaction hashes in which entries
//  are automatically dropped after txSetDuration time
type txSet struct {
	txMap          map[common.Hash]struct{}
	txOrd          map[uint64]txOrdType
	addPtr, delPtr uint64
}

const txSetDuration = time.Hour * 2

// txOrdType represents an entry in the time-ordered list of transaction hashes
type txOrdType struct {
	hash common.Hash
	time time.Time
}

// newTxSet creates a new transaction set
func newTxSet() *txSet {
	return &txSet{
		txMap: make(map[common.Hash]struct{}),
		txOrd: make(map[uint64]txOrdType),
	}
}

// contains returns true if the set contains the given transaction hash
// (not thread safe, should be called from a locked environment)
func (self *txSet) contains(hash common.Hash) bool {
	_, ok := self.txMap[hash]
	return ok
}

// add adds a transaction hash to the set, then removes entries older than txSetDuration
// (not thread safe, should be called from a locked environment)
func (self *txSet) add(hash common.Hash) {
	self.txMap[hash] = struct{}{}
	now := time.Now()
	self.txOrd[self.addPtr] = txOrdType{hash: hash, time: now}
	self.addPtr++
	delBefore := now.Add(-txSetDuration)
	for self.delPtr < self.addPtr && self.txOrd[self.delPtr].time.Before(delBefore) {
		delete(self.txMap, self.txOrd[self.delPtr].hash)
		delete(self.txOrd, self.delPtr)
		self.delPtr++
	}
}
