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
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/params"
	"gopkg.in/karalabe/cookiejar.v2/collections/prque"
)

var (
	// Transaction Pool Errors
	ErrInvalidSender     = errors.New("Invalid sender")
	ErrNonce             = errors.New("Nonce too low")
	ErrCheap             = errors.New("Gas price too low for acceptance")
	ErrBalance           = errors.New("Insufficient balance")
	ErrInsufficientFunds = errors.New("Insufficient funds for gas * price + value")
	ErrIntrinsicGas      = errors.New("Intrinsic gas too low")
	ErrGasLimit          = errors.New("Exceeds block gas limit")
	ErrNegativeValue     = errors.New("Negative value")
)

var (
	minPendingPerAccount = uint64(16)    // Min number of guaranteed transaction slots per address
	maxPendingTotal      = uint64(4096)  // Max limit of pending transactions from all accounts (soft)
	maxQueuedPerAccount  = uint64(64)    // Max limit of queued transactions per address
	maxQueuedInTotal     = uint64(1024)  // Max limit of queued transactions from all accounts
	maxQueuedLifetime    = 3 * time.Hour // Max amount of time transactions from idle accounts are queued
	evictionInterval     = time.Minute   // Time interval to check for evictable transactions
)

var (
	// Metrics for the pending pool
	pendingDiscardCounter = metrics.NewCounter("txpool/pending/discard")
	pendingReplaceCounter = metrics.NewCounter("txpool/pending/replace")
	pendingRLCounter      = metrics.NewCounter("txpool/pending/ratelimit") // Dropped due to rate limiting
	pendingNofundsCounter = metrics.NewCounter("txpool/pending/nofunds")   // Dropped due to out-of-funds

	// Metrics for the queued pool
	queuedDiscardCounter = metrics.NewCounter("txpool/queued/discard")
	queuedReplaceCounter = metrics.NewCounter("txpool/queued/replace")
	queuedRLCounter      = metrics.NewCounter("txpool/queued/ratelimit") // Dropped due to rate limiting
	queuedNofundsCounter = metrics.NewCounter("txpool/queued/nofunds")   // Dropped due to out-of-funds

	// General tx metrics
	invalidTxCounter = metrics.NewCounter("txpool/invalid")
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
	config       *params.ChainConfig
	currentState stateFn // The state function which will allow us to do some pre checks
	pendingState *state.ManagedState
	gasLimit     func() *big.Int // The current gas limit function callback
	minGasPrice  *big.Int
	eventMux     *event.TypeMux
	events       event.Subscription
	localTx      *txSet
	signer       types.Signer
	mu           sync.RWMutex

	pending map[common.Address]*txList         // All currently processable transactions
	queue   map[common.Address]*txList         // Queued but non-processable transactions
	all     map[common.Hash]*types.Transaction // All transactions to allow lookups
	beats   map[common.Address]time.Time       // Last heartbeat from each known account

	wg   sync.WaitGroup // for shutdown sync
	quit chan struct{}

	homestead bool
}

func NewTxPool(config *params.ChainConfig, eventMux *event.TypeMux, currentStateFn stateFn, gasLimitFn func() *big.Int) *TxPool {
	pool := &TxPool{
		config:       config,
		signer:       types.NewEIP155Signer(config.ChainId),
		pending:      make(map[common.Address]*txList),
		queue:        make(map[common.Address]*txList),
		all:          make(map[common.Hash]*types.Transaction),
		beats:        make(map[common.Address]time.Time),
		eventMux:     eventMux,
		currentState: currentStateFn,
		gasLimit:     gasLimitFn,
		minGasPrice:  new(big.Int),
		pendingState: nil,
		localTx:      newTxSet(),
		events:       eventMux.Subscribe(ChainHeadEvent{}, GasPriceChanged{}, RemovedTransactionEvent{}),
		quit:         make(chan struct{}),
	}

	pool.resetState()

	pool.wg.Add(2)
	go pool.eventLoop()
	go pool.expirationLoop()

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
			if ev.Block != nil {
				if pool.config.IsHomestead(ev.Block.Number()) {
					pool.homestead = true
				}
			}

			pool.resetState()
			pool.mu.Unlock()
		case GasPriceChanged:
			pool.mu.Lock()
			pool.minGasPrice = ev.Price
			pool.mu.Unlock()
		case RemovedTransactionEvent:
			pool.AddBatch(ev.Txs)
		}
	}
}

func (pool *TxPool) resetState() {
	currentState, err := pool.currentState()
	if err != nil {
		glog.V(logger.Error).Infof("Failed to get current state: %v", err)
		return
	}
	managedState := state.ManageState(currentState)
	if err != nil {
		glog.V(logger.Error).Infof("Failed to get managed state: %v", err)
		return
	}
	pool.pendingState = managedState

	// validate the pool of pending transactions, this will remove
	// any transactions that have been included in the block or
	// have been invalidated because of another transaction (e.g.
	// higher gas price)
	pool.demoteUnexecutables(currentState)

	// Update all accounts to the latest known pending nonce
	for addr, list := range pool.pending {
		txs := list.Flatten() // Heavy but will be cached and is needed by the miner anyway
		pool.pendingState.SetNonce(addr, txs[len(txs)-1].Nonce()+1)
	}
	// Check the queue and move transactions over to the pending if possible
	// or remove those that have become invalid
	pool.promoteExecutables(currentState)
}

func (pool *TxPool) Stop() {
	pool.events.Unsubscribe()
	close(pool.quit)
	pool.wg.Wait()
	glog.V(logger.Info).Infoln("Transaction pool stopped")
}

func (pool *TxPool) State() *state.ManagedState {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	return pool.pendingState
}

// Stats retrieves the current pool stats, namely the number of pending and the
// number of queued (non-executable) transactions.
func (pool *TxPool) Stats() (pending int, queued int) {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	for _, list := range pool.pending {
		pending += list.Len()
	}
	for _, list := range pool.queue {
		queued += list.Len()
	}
	return
}

// Content retrieves the data content of the transaction pool, returning all the
// pending as well as queued transactions, grouped by account and sorted by nonce.
func (pool *TxPool) Content() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	pending := make(map[common.Address]types.Transactions)
	for addr, list := range pool.pending {
		pending[addr] = list.Flatten()
	}
	queued := make(map[common.Address]types.Transactions)
	for addr, list := range pool.queue {
		queued[addr] = list.Flatten()
	}
	return pending, queued
}

// Pending retrieves all currently processable transactions, groupped by origin
// account and sorted by nonce. The returned transaction set is a copy and can be
// freely modified by calling code.
func (pool *TxPool) Pending() (map[common.Address]types.Transactions, error) {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	state, err := pool.currentState()
	if err != nil {
		return nil, err
	}

	// check queue first
	pool.promoteExecutables(state)

	// invalidate any txs
	pool.demoteUnexecutables(state)

	pending := make(map[common.Address]types.Transactions)
	for addr, list := range pool.pending {
		pending[addr] = list.Flatten()
	}
	return pending, nil
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

	from, err := types.Sender(pool.signer, tx)
	if err != nil {
		return ErrInvalidSender
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

	intrGas := IntrinsicGas(tx.Data(), tx.To() == nil, pool.homestead)
	if tx.Gas().Cmp(intrGas) < 0 {
		return ErrIntrinsicGas
	}

	return nil
}

// add validates a transaction and inserts it into the non-executable queue for
// later pending promotion and execution.
func (pool *TxPool) add(tx *types.Transaction) error {
	// If the transaction is alreayd known, discard it
	hash := tx.Hash()
	if pool.all[hash] != nil {
		return fmt.Errorf("Known transaction: %x", hash[:4])
	}
	// Otherwise ensure basic validation passes and queue it up
	if err := pool.validateTx(tx); err != nil {
		invalidTxCounter.Inc(1)
		return err
	}
	pool.enqueueTx(hash, tx)

	// Print a log message if low enough level is set
	if glog.V(logger.Debug) {
		rcpt := "[NEW_CONTRACT]"
		if to := tx.To(); to != nil {
			rcpt = common.Bytes2Hex(to[:4])
		}
		from, _ := types.Sender(pool.signer, tx) // from already verified during tx validation
		glog.Infof("(t) 0x%x => %s (%v) %x\n", from[:4], rcpt, tx.Value, hash)
	}
	return nil
}

// enqueueTx inserts a new transaction into the non-executable transaction queue.
//
// Note, this method assumes the pool lock is held!
func (pool *TxPool) enqueueTx(hash common.Hash, tx *types.Transaction) {
	// Try to insert the transaction into the future queue
	from, _ := types.Sender(pool.signer, tx) // already validated
	if pool.queue[from] == nil {
		pool.queue[from] = newTxList(false)
	}
	inserted, old := pool.queue[from].Add(tx)
	if !inserted {
		queuedDiscardCounter.Inc(1)
		return // An older transaction was better, discard this
	}
	// Discard any previous transaction and mark this
	if old != nil {
		delete(pool.all, old.Hash())
		queuedReplaceCounter.Inc(1)
	}
	pool.all[hash] = tx
}

// promoteTx adds a transaction to the pending (processable) list of transactions.
//
// Note, this method assumes the pool lock is held!
func (pool *TxPool) promoteTx(addr common.Address, hash common.Hash, tx *types.Transaction) {
	// Try to insert the transaction into the pending queue
	if pool.pending[addr] == nil {
		pool.pending[addr] = newTxList(true)
	}
	list := pool.pending[addr]

	inserted, old := list.Add(tx)
	if !inserted {
		// An older transaction was better, discard this
		delete(pool.all, hash)
		pendingDiscardCounter.Inc(1)
		return
	}
	// Otherwise discard any previous transaction and mark this
	if old != nil {
		delete(pool.all, old.Hash())
		pendingReplaceCounter.Inc(1)
	}
	pool.all[hash] = tx // Failsafe to work around direct pending inserts (tests)

	// Set the potentially new pending nonce and notify any subsystems of the new tx
	pool.beats[addr] = time.Now()
	pool.pendingState.SetNonce(addr, tx.Nonce()+1)
	go pool.eventMux.Post(TxPreEvent{tx})
}

// Add queues a single transaction in the pool if it is valid.
func (pool *TxPool) Add(tx *types.Transaction) error {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	if err := pool.add(tx); err != nil {
		return err
	}

	state, err := pool.currentState()
	if err != nil {
		return err
	}

	pool.promoteExecutables(state)

	return nil
}

// AddBatch attempts to queue a batch of transactions.
func (pool *TxPool) AddBatch(txs []*types.Transaction) error {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	for _, tx := range txs {
		if err := pool.add(tx); err != nil {
			glog.V(logger.Debug).Infoln("tx error:", err)
		}
	}

	state, err := pool.currentState()
	if err != nil {
		return err
	}

	pool.promoteExecutables(state)

	return nil
}

// Get returns a transaction if it is contained in the pool
// and nil otherwise.
func (pool *TxPool) Get(hash common.Hash) *types.Transaction {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	return pool.all[hash]
}

// Remove removes the transaction with the given hash from the pool.
func (pool *TxPool) Remove(hash common.Hash) {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	pool.removeTx(hash)
}

// RemoveBatch removes all given transactions from the pool.
func (pool *TxPool) RemoveBatch(txs types.Transactions) {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	for _, tx := range txs {
		pool.removeTx(tx.Hash())
	}
}

// removeTx removes a single transaction from the queue, moving all subsequent
// transactions back to the future queue.
func (pool *TxPool) removeTx(hash common.Hash) {
	// Fetch the transaction we wish to delete
	tx, ok := pool.all[hash]
	if !ok {
		return
	}
	addr, _ := types.Sender(pool.signer, tx) // already validated during insertion

	// Remove it from the list of known transactions
	delete(pool.all, hash)

	// Remove the transaction from the pending lists and reset the account nonce
	if pending := pool.pending[addr]; pending != nil {
		if removed, invalids := pending.Remove(tx); removed {
			// If no more transactions are left, remove the list
			if pending.Empty() {
				delete(pool.pending, addr)
				delete(pool.beats, addr)
			} else {
				// Otherwise postpone any invalidated transactions
				for _, tx := range invalids {
					pool.enqueueTx(tx.Hash(), tx)
				}
			}
			// Update the account nonce if needed
			if nonce := tx.Nonce(); pool.pendingState.GetNonce(addr) > nonce {
				pool.pendingState.SetNonce(addr, tx.Nonce())
			}
		}
	}
	// Transaction is in the future queue
	if future := pool.queue[addr]; future != nil {
		future.Remove(tx)
		if future.Empty() {
			delete(pool.queue, addr)
		}
	}
}

// promoteExecutables moves transactions that have become processable from the
// future queue to the set of pending transactions. During this process, all
// invalidated transactions (low nonce, low balance) are deleted.
func (pool *TxPool) promoteExecutables(state *state.StateDB) {
	// Iterate over all accounts and promote any executable transactions
	queued := uint64(0)
	for addr, list := range pool.queue {
		// Drop all transactions that are deemed too old (low nonce)
		for _, tx := range list.Forward(state.GetNonce(addr)) {
			if glog.V(logger.Core) {
				glog.Infof("Removed old queued transaction: %v", tx)
			}
			delete(pool.all, tx.Hash())
		}
		// Drop all transactions that are too costly (low balance)
		drops, _ := list.Filter(state.GetBalance(addr))
		for _, tx := range drops {
			if glog.V(logger.Core) {
				glog.Infof("Removed unpayable queued transaction: %v", tx)
			}
			delete(pool.all, tx.Hash())
			queuedNofundsCounter.Inc(1)
		}
		// Gather all executable transactions and promote them
		for _, tx := range list.Ready(pool.pendingState.GetNonce(addr)) {
			if glog.V(logger.Core) {
				glog.Infof("Promoting queued transaction: %v", tx)
			}
			pool.promoteTx(addr, tx.Hash(), tx)
		}
		// Drop all transactions over the allowed limit
		for _, tx := range list.Cap(int(maxQueuedPerAccount)) {
			if glog.V(logger.Core) {
				glog.Infof("Removed cap-exceeding queued transaction: %v", tx)
			}
			delete(pool.all, tx.Hash())
			queuedRLCounter.Inc(1)
		}
		queued += uint64(list.Len())

		// Delete the entire queue entry if it became empty.
		if list.Empty() {
			delete(pool.queue, addr)
		}
	}
	// If the pending limit is overflown, start equalizing allowances
	pending := uint64(0)
	for _, list := range pool.pending {
		pending += uint64(list.Len())
	}
	if pending > maxPendingTotal {
		pendingBeforeCap := pending
		// Assemble a spam order to penalize large transactors first
		spammers := prque.New()
		for addr, list := range pool.pending {
			// Only evict transactions from high rollers
			if uint64(list.Len()) > minPendingPerAccount {
				// Skip local accounts as pools should maintain backlogs for themselves
				for _, tx := range list.txs.items {
					if !pool.localTx.contains(tx.Hash()) {
						spammers.Push(addr, float32(list.Len()))
					}
					break // Checking on transaction for locality is enough
				}
			}
		}
		// Gradually drop transactions from offenders
		offenders := []common.Address{}
		for pending > maxPendingTotal && !spammers.Empty() {
			// Retrieve the next offender if not local address
			offender, _ := spammers.Pop()
			offenders = append(offenders, offender.(common.Address))

			// Equalize balances until all the same or below threshold
			if len(offenders) > 1 {
				// Calculate the equalization threshold for all current offenders
				threshold := pool.pending[offender.(common.Address)].Len()

				// Iteratively reduce all offenders until below limit or threshold reached
				for pending > maxPendingTotal && pool.pending[offenders[len(offenders)-2]].Len() > threshold {
					for i := 0; i < len(offenders)-1; i++ {
						list := pool.pending[offenders[i]]
						list.Cap(list.Len() - 1)
						pending--
					}
				}
			}
		}
		// If still above threshold, reduce to limit or min allowance
		if pending > maxPendingTotal && len(offenders) > 0 {
			for pending > maxPendingTotal && uint64(pool.pending[offenders[len(offenders)-1]].Len()) > minPendingPerAccount {
				for _, addr := range offenders {
					list := pool.pending[addr]
					list.Cap(list.Len() - 1)
					pending--
				}
			}
		}
		pendingRLCounter.Inc(int64(pendingBeforeCap - pending))
	}
	// If we've queued more transactions than the hard limit, drop oldest ones
	if queued > maxQueuedInTotal {
		// Sort all accounts with queued transactions by heartbeat
		addresses := make(addresssByHeartbeat, 0, len(pool.queue))
		for addr, _ := range pool.queue {
			addresses = append(addresses, addressByHeartbeat{addr, pool.beats[addr]})
		}
		sort.Sort(addresses)

		// Drop transactions until the total is below the limit
		for drop := queued - maxQueuedInTotal; drop > 0; {
			addr := addresses[len(addresses)-1]
			list := pool.queue[addr.address]

			addresses = addresses[:len(addresses)-1]

			// Drop all transactions if they are less than the overflow
			if size := uint64(list.Len()); size <= drop {
				for _, tx := range list.Flatten() {
					pool.removeTx(tx.Hash())
				}
				drop -= size
				queuedRLCounter.Inc(int64(size))
				continue
			}
			// Otherwise drop only last few transactions
			txs := list.Flatten()
			for i := len(txs) - 1; i >= 0 && drop > 0; i-- {
				pool.removeTx(txs[i].Hash())
				drop--
				queuedRLCounter.Inc(1)
			}
		}
	}
}

// demoteUnexecutables removes invalid and processed transactions from the pools
// executable/pending queue and any subsequent transactions that become unexecutable
// are moved back into the future queue.
func (pool *TxPool) demoteUnexecutables(state *state.StateDB) {
	// Iterate over all accounts and demote any non-executable transactions
	for addr, list := range pool.pending {
		nonce := state.GetNonce(addr)

		// Drop all transactions that are deemed too old (low nonce)
		for _, tx := range list.Forward(nonce) {
			if glog.V(logger.Core) {
				glog.Infof("Removed old pending transaction: %v", tx)
			}
			delete(pool.all, tx.Hash())
		}
		// Drop all transactions that are too costly (low balance), and queue any invalids back for later
		drops, invalids := list.Filter(state.GetBalance(addr))
		for _, tx := range drops {
			if glog.V(logger.Core) {
				glog.Infof("Removed unpayable pending transaction: %v", tx)
			}
			delete(pool.all, tx.Hash())
			pendingNofundsCounter.Inc(1)
		}
		for _, tx := range invalids {
			if glog.V(logger.Core) {
				glog.Infof("Demoting pending transaction: %v", tx)
			}
			pool.enqueueTx(tx.Hash(), tx)
		}
		// Delete the entire queue entry if it became empty.
		if list.Empty() {
			delete(pool.pending, addr)
			delete(pool.beats, addr)
		}
	}
}

// expirationLoop is a loop that periodically iterates over all accounts with
// queued transactions and drop all that have been inactive for a prolonged amount
// of time.
func (pool *TxPool) expirationLoop() {
	defer pool.wg.Done()

	evict := time.NewTicker(evictionInterval)
	defer evict.Stop()

	for {
		select {
		case <-evict.C:
			pool.mu.Lock()
			for addr := range pool.queue {
				if time.Since(pool.beats[addr]) > maxQueuedLifetime {
					for _, tx := range pool.queue[addr].Flatten() {
						pool.removeTx(tx.Hash())
					}
				}
			}
			pool.mu.Unlock()

		case <-pool.quit:
			return
		}
	}
}

// addressByHeartbeat is an account address tagged with its last activity timestamp.
type addressByHeartbeat struct {
	address   common.Address
	heartbeat time.Time
}

type addresssByHeartbeat []addressByHeartbeat

func (a addresssByHeartbeat) Len() int           { return len(a) }
func (a addresssByHeartbeat) Less(i, j int) bool { return a[i].heartbeat.Before(a[j].heartbeat) }
func (a addresssByHeartbeat) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

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
