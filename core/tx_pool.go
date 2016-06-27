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

// TxList is a "list" of transactions belonging to an account, sorted by account
// nonce. To allow gaps and avoid constant copying, the list is represented as a
// hash map.
type TxList map[uint64]*types.Transaction

// TxPool contains all currently known transactions. Transactions
// enter the pool when they are received from the network or submitted
// locally. They exit the pool when they are included in the blockchain.
//
// The pool separates processable transactions (which can be applied to the
// current state) and future transactions. Transactions move between those
// two states over time as they are received and processed.
type TxPool struct {
	config       *ChainConfig
	currentState stateFn // The state function which will allow us to do some pre checks
	pendingState *state.ManagedState
	gasLimit     func() *big.Int // The current gas limit function callback
	minGasPrice  *big.Int
	eventMux     *event.TypeMux
	events       event.Subscription
	localTx      *txSet
	mu           sync.RWMutex

	pending map[common.Address]TxList          // All currently processable transactions
	queue   map[common.Address]TxList          // Queued but non-processable transactions
	all     map[common.Hash]*types.Transaction // All transactions to allow lookups

	wg sync.WaitGroup // for shutdown sync

	homestead bool
}

func NewTxPool(config *ChainConfig, eventMux *event.TypeMux, currentStateFn stateFn, gasLimitFn func() *big.Int) *TxPool {
	pool := &TxPool{
		config:       config,
		pending:      make(map[common.Address]TxList),
		queue:        make(map[common.Address]TxList),
		all:          make(map[common.Hash]*types.Transaction),
		eventMux:     eventMux,
		currentState: currentStateFn,
		gasLimit:     gasLimitFn,
		minGasPrice:  new(big.Int),
		pendingState: nil,
		localTx:      newTxSet(),
		events:       eventMux.Subscribe(ChainHeadEvent{}, GasPriceChanged{}, RemovedTransactionEvent{}),
	}

	pool.wg.Add(1)
	go pool.eventLoop()

	return pool
}

func (pool *TxPool) eventLoop() {
	defer pool.wg.Done()

	// Track chain events. When a chain events occurs (new chain canon block)
	// we need to know the new state. The new state will help us determine
	// the nonces in the managed state
	for ev := range pool.events.Chan() {
		switch ev := ev.Data.(type) {
		case ChainHeadEvent:
			pool.mu.Lock()
			if ev.Block != nil && pool.config.IsHomestead(ev.Block.Number()) {
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
	for addr, txs := range pool.pending {
		// Set the nonce. Transaction nonce can never be lower
		// than the state nonce; validatePool took care of that.
		for nonce, _ := range txs {
			if pool.pendingState.GetNonce(addr) <= nonce {
				pool.pendingState.SetNonce(addr, nonce+1)
			}
		}
	}
	// Check the queue and move transactions over to the pending if possible
	// or remove those that have become invalid
	pool.checkQueue()
}

func (pool *TxPool) Stop() {
	pool.events.Unsubscribe()
	pool.wg.Wait()
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

	for _, txs := range pool.pending {
		pending += len(txs)
	}
	for _, txs := range pool.queue {
		queued += len(txs)
	}
	return
}

// Content retrieves the data content of the transaction pool, returning all the
// pending as well as queued transactions, grouped by account and nonce.
func (pool *TxPool) Content() (map[common.Address]TxList, map[common.Address]TxList) {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	// Retrieve all the pending transactions and sort by account and by nonce
	pending := make(map[common.Address]TxList)
	for addr, txs := range pool.pending {
		copy := make(TxList)
		for nonce, tx := range txs {
			copy[nonce] = tx
		}
		pending[addr] = copy
	}
	// Retrieve all the queued transactions and sort by account and by nonce
	queued := make(map[common.Address]TxList)
	for addr, txs := range pool.queue {
		copy := make(TxList)
		for nonce, tx := range txs {
			copy[nonce] = tx
		}
		queued[addr] = copy
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

	if self.all[hash] != nil {
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

// queueTx will queue an unknown transaction.
func (self *TxPool) queueTx(hash common.Hash, tx *types.Transaction) {
	addr, _ := tx.From() // already validated
	if self.queue[addr] == nil {
		self.queue[addr] = make(TxList)
	}
	// If the nonce is already used, discard the lower priced transaction
	nonce := tx.Nonce()

	if old, ok := self.queue[addr][nonce]; ok {
		if old.GasPrice().Cmp(tx.GasPrice()) >= 0 {
			return // Old was better, discard this
		}
		delete(self.all, old.Hash()) // New is better, drop and overwrite old one
	}
	self.queue[addr][nonce] = tx
	self.all[hash] = tx
}

// addTx will moves a transaction from the non-executable queue to the pending
// (processable) list of transactions.
func (pool *TxPool) addTx(addr common.Address, tx *types.Transaction) {
	// Init delayed since tx pool could have been started before any state sync
	if pool.pendingState == nil {
		pool.resetState()
	}
	// If the nonce is already used, discard the lower priced transaction
	hash, nonce := tx.Hash(), tx.Nonce()

	if old, ok := pool.pending[addr][nonce]; ok {
		oldHash := old.Hash()

		switch {
		case oldHash == hash: // Nothing changed, noop
			return
		case old.GasPrice().Cmp(tx.GasPrice()) >= 0: // Old was better, discard this
			delete(pool.all, hash)
			return
		default: // New is better, discard old
			delete(pool.all, oldHash)
		}
	}
	// The transaction is being kept, insert it into the tx pool
	if _, ok := pool.pending[addr]; !ok {
		pool.pending[addr] = make(TxList)
	}
	pool.pending[addr][nonce] = tx
	pool.all[hash] = tx

	// Increment the nonce on the pending state. This can only happen if
	// the nonce is +1 to the previous one.
	pool.pendingState.SetNonce(addr, nonce+1)

	// Notify the subscribers. This event is posted in a goroutine
	// because it's possible that somewhere during the post "Remove transaction"
	// gets called which will then wait for the global tx pool lock and deadlock.
	go pool.eventMux.Post(TxPreEvent{tx})
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

	// check and validate the queue
	self.checkQueue()
}

// GetTransaction returns a transaction if it is contained in the pool
// and nil otherwise.
func (tp *TxPool) GetTransaction(hash common.Hash) *types.Transaction {
	tp.mu.RLock()
	defer tp.mu.RUnlock()

	return tp.all[hash]
}

// GetTransactions returns all currently processable transactions.
// The returned slice may be modified by the caller.
func (self *TxPool) GetTransactions() types.Transactions {
	self.mu.Lock()
	defer self.mu.Unlock()

	// check queue first
	self.checkQueue()

	// invalidate any txs
	self.validatePool()

	count := 0
	for _, txs := range self.pending {
		count += len(txs)
	}
	pending := make(types.Transactions, 0, count)
	for _, txs := range self.pending {
		for _, tx := range txs {
			pending = append(pending, tx)
		}
	}
	return pending
}

// RemoveTransactions removes all given transactions from the pool.
func (self *TxPool) RemoveTransactions(txs types.Transactions) {
	self.mu.Lock()
	defer self.mu.Unlock()

	for _, tx := range txs {
		self.removeTx(tx.Hash())
	}
}

// RemoveTx removes the transaction with the given hash from the pool.
func (pool *TxPool) RemoveTx(hash common.Hash) {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	pool.removeTx(hash)
}

func (pool *TxPool) removeTx(hash common.Hash) {
	// Fetch the transaction we wish to delete
	tx, ok := pool.all[hash]
	if !ok {
		return
	}
	addr, _ := tx.From()

	// Remove it from all internal lists
	delete(pool.all, hash)

	delete(pool.pending[addr], tx.Nonce())
	if len(pool.pending[addr]) == 0 {
		delete(pool.pending, addr)
	}
	delete(pool.queue[addr], tx.Nonce())
	if len(pool.queue[addr]) == 0 {
		delete(pool.queue, addr)
	}
}

// checkQueue moves transactions that have become processable from the pool's
// queue to the set of pending transactions.
func (pool *TxPool) checkQueue() {
	// Init delayed since tx pool could have been started before any state sync
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
		for nonce, tx := range txs {
			// Drop processed or out of fund transactions
			if nonce < trueNonce || balance.Cmp(tx.Cost()) < 0 {
				if glog.V(logger.Core) {
					glog.Infof("removed tx (%v) from pool queue: low tx nonce or out of funds\n", tx)
				}
				delete(txs, nonce)
				delete(pool.all, tx.Hash())

				continue
			}
			// Collect the remaining transactions for the next pass.
			promote = append(promote, txQueueEntry{address, tx})
		}
		// Find the next consecutive nonce range starting at the current account nonce,
		// pushing the guessed nonce forward if we add consecutive transactions.
		sort.Sort(promote)
		for i, entry := range promote {
			// If we reached a gap in the nonces, enforce transaction limit and stop
			if entry.Nonce() > guessedNonce {
				if len(promote)-i > maxQueued {
					if glog.V(logger.Debug) {
						glog.Infof("Queued tx limit exceeded for %s. Tx %s removed\n", common.PP(address[:]), common.PP(entry.Hash().Bytes()))
					}
					for _, drop := range promote[i+maxQueued:] {
						delete(txs, drop.Nonce())
						delete(pool.all, drop.Hash())
					}
				}
				break
			}
			// Otherwise promote the transaction and move the guess nonce if needed
			pool.addTx(address, entry.Transaction)
			delete(txs, entry.Nonce())

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

	for addr, txs := range pool.pending {
		for nonce, tx := range txs {
			// Perform light nonce and balance validation
			balance := balanceCache[addr]
			if balance == nil {
				balance = state.GetBalance(addr)
				balanceCache[addr] = balance
			}
			if past := state.GetNonce(addr) > nonce; past || balance.Cmp(tx.Cost()) < 0 {
				// Remove an already past it invalidated transaction
				if glog.V(logger.Core) {
					glog.Infof("removed tx (%v) from pool: low tx nonce or out of funds\n", tx)
				}
				delete(pool.pending[addr], nonce)
				if len(pool.pending[addr]) == 0 {
					delete(pool.pending, addr)
				}
				delete(pool.all, tx.Hash())

				// Track the smallest invalid nonce to postpone subsequent transactions
				if !past {
					if prev, ok := gaps[addr]; !ok || nonce < prev {
						gaps[addr] = nonce
					}
				}
			}
		}
	}
	// Move all transactions after a gap back to the future queue
	if len(gaps) > 0 {
		for addr, txs := range pool.pending {
			for nonce, tx := range txs {
				if gap, ok := gaps[addr]; ok && nonce >= gap {
					if glog.V(logger.Core) {
						glog.Infof("postponed tx (%v) due to introduced gap\n", tx)
					}
					delete(pool.pending[addr], nonce)
					if len(pool.pending[addr]) == 0 {
						delete(pool.pending, addr)
					}
					pool.queueTx(tx.Hash(), tx)
				}
			}
		}
	}
}

type txQueue []txQueueEntry

type txQueueEntry struct {
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
