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
	ErrInvalidSender      = errors.New("Invalid sender")
	ErrNonce              = errors.New("Nonce too low")
	ErrBalance            = errors.New("Insufficient balance")
	ErrNonExistentAccount = errors.New("Account does not exist or account balance too low")
	ErrInsufficientFunds  = errors.New("Insufficient funds for gas * price + value")
	ErrIntrinsicGas       = errors.New("Intrinsic gas too low")
	ErrGasLimit           = errors.New("Exceeds block gas limit")
	ErrNegativeValue      = errors.New("Negative value")
)

type stateFn func() *state.StateDB

// The tx pool a thread safe transaction pool handler. In order to
// guarantee a non blocking pool we use a queue channel which can be
// independently read without needing access to the actual pool.
type TxPool struct {
	quit         chan bool       // Quiting channel
	currentState stateFn         // The state function which will allow us to do some pre checkes
	gasLimit     func() *big.Int // The current gas limit function callback
	eventMux     *event.TypeMux

	mu    sync.RWMutex
	txs   map[common.Hash]*types.Transaction // The actual pool
	queue map[common.Address]map[common.Hash]*types.Transaction
}

func NewTxPool(eventMux *event.TypeMux, currentStateFn stateFn, gasLimitFn func() *big.Int) *TxPool {
	return &TxPool{
		txs:          make(map[common.Hash]*types.Transaction),
		queue:        make(map[common.Address]map[common.Hash]*types.Transaction),
		quit:         make(chan bool),
		eventMux:     eventMux,
		currentState: currentStateFn,
		gasLimit:     gasLimitFn,
	}
}

func (pool *TxPool) Start() {
	// Queue timer will tick so we can attempt to move items from the queue to the
	// main transaction pool.
	queueTimer := time.NewTicker(300 * time.Millisecond)
	// Removal timer will tick and attempt to remove bad transactions (account.nonce>tx.nonce)
	removalTimer := time.NewTicker(1 * time.Second)
done:
	for {
		select {
		case <-queueTimer.C:
			pool.checkQueue()
		case <-removalTimer.C:
			pool.validatePool()
		case <-pool.quit:
			break done
		}
	}
}

func (pool *TxPool) ValidateTransaction(tx *types.Transaction) error {
	// Validate sender
	var (
		from common.Address
		err  error
	)

	if from, err = tx.From(); err != nil {
		return ErrInvalidSender
	}

	if !pool.currentState().HasAccount(from) {
		return ErrNonExistentAccount
	}

	if pool.gasLimit().Cmp(tx.GasLimit) < 0 {
		return ErrGasLimit
	}

	if tx.Amount.Cmp(common.Big0) < 0 {
		return ErrNegativeValue
	}

	total := new(big.Int).Mul(tx.Price, tx.GasLimit)
	total.Add(total, tx.Value())
	if pool.currentState().GetBalance(from).Cmp(total) < 0 {
		return ErrInsufficientFunds
	}

	if tx.GasLimit.Cmp(IntrinsicGas(tx)) < 0 {
		return ErrIntrinsicGas
	}

	if pool.currentState().GetNonce(from) > tx.Nonce() {
		return ErrNonce
	}

	return nil
}

func (self *TxPool) add(tx *types.Transaction) error {
	hash := tx.Hash()

	/* XXX I'm unsure about this. This is extremely dangerous and may result
	 in total black listing of certain transactions
	if self.invalidHashes.Has(hash) {
		return fmt.Errorf("Invalid transaction (%x)", hash[:4])
	}
	*/
	if self.txs[hash] != nil {
		return fmt.Errorf("Known transaction (%x)", hash[:4])
	}
	err := self.ValidateTransaction(tx)
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

func (self *TxPool) Size() int {
	return len(self.txs)
}

func (self *TxPool) Add(tx *types.Transaction) error {
	self.mu.Lock()
	defer self.mu.Unlock()

	return self.add(tx)
}

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

// GetTransaction allows you to check the pending and queued transaction in the
// transaction pool.
// It has two stategies, first check the pool (map) then check the queue
func (tp *TxPool) GetTransaction(hash common.Hash) *types.Transaction {
	// check the txs first
	if tx, ok := tp.txs[hash]; ok {
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

func (self *TxPool) GetTransactions() (txs types.Transactions) {
	self.mu.RLock()
	defer self.mu.RUnlock()

	txs = make(types.Transactions, self.Size())
	i := 0
	for _, tx := range self.txs {
		txs[i] = tx
		i++
	}
	return txs
}

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

func (self *TxPool) RemoveTransactions(txs types.Transactions) {
	self.mu.Lock()
	defer self.mu.Unlock()
	for _, tx := range txs {
		self.removeTx(tx.Hash())
	}
}

func (pool *TxPool) Flush() {
	pool.txs = make(map[common.Hash]*types.Transaction)
}

func (pool *TxPool) Stop() {
	pool.Flush()
	close(pool.quit)

	glog.V(logger.Info).Infoln("TX Pool stopped")
}

func (self *TxPool) queueTx(hash common.Hash, tx *types.Transaction) {
	from, _ := tx.From() // already validated
	if self.queue[from] == nil {
		self.queue[from] = make(map[common.Hash]*types.Transaction)
	}
	self.queue[from][hash] = tx
}

func (pool *TxPool) addTx(hash common.Hash, tx *types.Transaction) {
	if _, ok := pool.txs[hash]; !ok {
		pool.txs[hash] = tx
		// Notify the subscribers. This event is posted in a goroutine
		// because it's possible that somewhere during the post "Remove transaction"
		// gets called which will then wait for the global tx pool lock and deadlock.
		go pool.eventMux.Post(TxPreEvent{tx})
	}
}

// check queue will attempt to insert
func (pool *TxPool) checkQueue() {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	statedb := pool.currentState()
	var addq txQueue
	for address, txs := range pool.queue {
		curnonce := statedb.GetNonce(address)
		addq := addq[:0]
		for hash, tx := range txs {
			if tx.AccountNonce < curnonce {
				// Drop queued transactions whose nonce is lower than
				// the account nonce because they have been processed.
				delete(txs, hash)
			} else {
				// Collect the remaining transactions for the next pass.
				addq = append(addq, txQueueEntry{hash, tx})
			}
		}
		// Find the next consecutive nonce range starting at the
		// current account nonce.
		sort.Sort(addq)
		for _, e := range addq {
			if e.AccountNonce != curnonce {
				break
			}
			curnonce++
			delete(txs, e.hash)
			pool.addTx(e.hash, e.Transaction)
		}
		// Delete the entire queue entry if it became empty.
		if len(txs) == 0 {
			delete(pool.queue, address)
		}
	}
}

func (pool *TxPool) removeTx(hash common.Hash) {
	// delete from pending pool
	delete(pool.txs, hash)
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

func (pool *TxPool) validatePool() {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	for hash, tx := range pool.txs {
		if err := pool.ValidateTransaction(tx); err != nil {
			if glog.V(logger.Info) {
				glog.Infof("removed tx (%x) from pool: %v\n", hash[:4], err)
			}
			delete(pool.txs, hash)
		}
	}
}

type txQueue []txQueueEntry

type txQueueEntry struct {
	hash common.Hash
	*types.Transaction
}

func (q txQueue) Len() int           { return len(q) }
func (q txQueue) Swap(i, j int)      { q[i], q[j] = q[j], q[i] }
func (q txQueue) Less(i, j int) bool { return q[i].AccountNonce < q[j].AccountNonce }
