package core

import (
	"errors"
	"fmt"
	"math/big"
	"sort"
	"sync"

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
	maxQueued = 200 // max limit of queued txs per address
)

type stateFn func() *state.StateDB

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

	mu      sync.RWMutex
	pending map[common.Hash]*types.Transaction // processable transactions
	queue   map[common.Address]map[common.Hash]*types.Transaction
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
		pendingState: state.ManageState(currentStateFn()),
		events:       eventMux.Subscribe(ChainEvent{}, GasPriceChanged{}),
	}
	go pool.eventLoop()

	return pool
}

func (pool *TxPool) eventLoop() {
	// Track chain events. When a chain events occurs (new chain canon block)
	// we need to know the new state. The new state will help us determine
	// the nonces in the managed state
	for ev := range pool.events.Chan() {
		pool.mu.Lock()

		switch ev := ev.(type) {
		case ChainEvent:
			pool.resetState()
		case GasPriceChanged:
			pool.minGasPrice = ev.Price
		}

		pool.mu.Unlock()
	}
}

func (pool *TxPool) resetState() {
	pool.pendingState = state.ManageState(pool.currentState())

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
			if pool.pendingState.GetNonce(addr) < tx.Nonce() {
				pool.pendingState.SetNonce(addr, tx.Nonce())
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
	glog.V(logger.Info).Infoln("TX Pool stopped")
}

func (pool *TxPool) State() *state.ManagedState {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	return pool.pendingState
}

// validateTx checks whether a transaction is valid according
// to the consensus rules.
func (pool *TxPool) validateTx(tx *types.Transaction) error {
	// Validate sender
	var (
		from common.Address
		err  error
	)

	// Drop transactions under our own minimal accepted gas price
	if pool.minGasPrice.Cmp(tx.GasPrice()) > 0 {
		return ErrCheap
	}

	// Validate the transaction sender and it's sig. Throw
	// if the from fields is invalid.
	if from, err = tx.From(); err != nil {
		return ErrInvalidSender
	}

	// Make sure the account exist. Non existent accounts
	// haven't got funds and well therefor never pass.
	if !pool.currentState().HasAccount(from) {
		return ErrNonExistentAccount
	}

	// Last but not least check for nonce errors
	if pool.currentState().GetNonce(from) > tx.Nonce() {
		return ErrNonce
	}

	// Check the transaction doesn't exceed the current
	// block limit gas.
	if pool.gasLimit().Cmp(tx.GasLimit) < 0 {
		return ErrGasLimit
	}

	// Transactions can't be negative. This may never happen
	// using RLP decoded transactions but may occur if you create
	// a transaction using the RPC for example.
	if tx.Amount.Cmp(common.Big0) < 0 {
		return ErrNegativeValue
	}

	// Transactor should have enough funds to cover the costs
	// cost == V + GP * GL
	total := new(big.Int).Mul(tx.Price, tx.GasLimit)
	total.Add(total, tx.Value())
	if pool.currentState().GetBalance(from).Cmp(total) < 0 {
		return ErrInsufficientFunds
	}

	// Should supply enough intrinsic gas
	if tx.GasLimit.Cmp(IntrinsicGas(tx)) < 0 {
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

	// check and validate the queueue
	self.checkQueue()

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
	if _, ok := pool.pending[hash]; !ok {
		pool.pending[hash] = tx

		// Increment the nonce on the pending state. This can only happen if
		// the nonce is +1 to the previous one.
		pool.pendingState.SetNonce(addr, tx.AccountNonce+1)
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

	return self.add(tx)
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
	sort.Sort(types.TxByNonce{ret})
	return ret
}

// RemoveTransactions removes all given transactions from the pool.
func (self *TxPool) RemoveTransactions(txs types.Transactions) {
	self.mu.Lock()
	defer self.mu.Unlock()
	for _, tx := range txs {
		self.removeTx(tx.Hash())
	}
}

// checkQueue moves transactions that have become processable to main pool.
func (pool *TxPool) checkQueue() {
	state := pool.pendingState

	var addq txQueue
	for address, txs := range pool.queue {
		// guessed nonce is the nonce currently kept by the tx pool (pending state)
		guessedNonce := state.GetNonce(address)
		// true nonce is the nonce known by the last state
		trueNonce := pool.currentState().GetNonce(address)
		addq := addq[:0]
		for hash, tx := range txs {
			if tx.AccountNonce < trueNonce {
				// Drop queued transactions whose nonce is lower than
				// the account nonce because they have been processed.
				delete(txs, hash)
			} else {
				// Collect the remaining transactions for the next pass.
				addq = append(addq, txQueueEntry{hash, address, tx})
			}
		}
		// Find the next consecutive nonce range starting at the
		// current account nonce.
		sort.Sort(addq)
		for i, e := range addq {
			// start deleting the transactions from the queue if they exceed the limit
			if i > maxQueued {
				if glog.V(logger.Debug) {
					glog.Infof("Queued tx limit exceeded for %s. Tx %s removed\n", common.PP(address[:]), common.PP(e.hash[:]))
				}
				delete(pool.queue[address], e.hash)
				continue
			}

			if e.AccountNonce > guessedNonce {
				break
			}
			delete(txs, e.hash)
			pool.addTx(e.hash, address, e.Transaction)
		}
		// Delete the entire queue entry if it became empty.
		if len(txs) == 0 {
			delete(pool.queue, address)
		}
	}
}

func (pool *TxPool) removeTx(hash common.Hash) {
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

// validatePool removes invalid and processed transactions from the main pool.
func (pool *TxPool) validatePool() {
	state := pool.currentState()
	for hash, tx := range pool.pending {
		from, _ := tx.From() // err already checked
		// perform light nonce validation
		if state.GetNonce(from) > tx.Nonce() {
			if glog.V(logger.Core) {
				glog.Infof("removed tx (%x) from pool: low tx nonce\n", hash[:4])
			}
			delete(pool.pending, hash)
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
func (q txQueue) Less(i, j int) bool { return q[i].AccountNonce < q[j].AccountNonce }
