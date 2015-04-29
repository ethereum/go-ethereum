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
	"gopkg.in/fatih/set.v0"
)

var (
	ErrInvalidSender      = errors.New("Invalid sender")
	ErrNonce              = errors.New("Nonce too low")
	ErrBalance            = errors.New("Insufficient balance")
	ErrNonExistentAccount = errors.New("Account does not exist")
	ErrInsufficientFunds  = errors.New("Insufficient funds for gas * price + value")
	ErrIntrinsicGas       = errors.New("Intrinsic gas too low")
	ErrGasLimit           = errors.New("Exceeds block gas limit")
)

const txPoolQueueSize = 50

type TxPoolHook chan *types.Transaction
type TxMsg struct{ Tx *types.Transaction }

type stateFn func() *state.StateDB

const (
	minGasPrice = 1000000
)

type TxProcessor interface {
	ProcessTransaction(tx *types.Transaction)
}

// The tx pool a thread safe transaction pool handler. In order to
// guarantee a non blocking pool we use a queue channel which can be
// independently read without needing access to the actual pool.
type TxPool struct {
	mu sync.RWMutex
	// Queueing channel for reading and writing incoming
	// transactions to
	queueChan chan *types.Transaction
	// Quiting channel
	quit chan bool
	// The state function which will allow us to do some pre checkes
	currentState stateFn
	// The current gas limit function callback
	gasLimit func() *big.Int
	// The actual pool
	txs           map[common.Hash]*types.Transaction
	invalidHashes *set.Set

	queue map[common.Address]types.Transactions

	subscribers []chan TxMsg

	eventMux *event.TypeMux
}

func NewTxPool(eventMux *event.TypeMux, currentStateFn stateFn, gasLimitFn func() *big.Int) *TxPool {
	txPool := &TxPool{
		txs:           make(map[common.Hash]*types.Transaction),
		queue:         make(map[common.Address]types.Transactions),
		queueChan:     make(chan *types.Transaction, txPoolQueueSize),
		quit:          make(chan bool),
		eventMux:      eventMux,
		invalidHashes: set.New(),
		currentState:  currentStateFn,
		gasLimit:      gasLimitFn,
	}
	return txPool
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

	// Validate curve param
	v, _, _ := tx.Curve()
	if v > 28 || v < 27 {
		return fmt.Errorf("tx.v != (28 || 27) => %v", v)
	}

	if !pool.currentState().HasAccount(from) {
		return ErrNonExistentAccount
	}

	if pool.gasLimit().Cmp(tx.GasLimit) < 0 {
		return ErrGasLimit
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

	self.queueTx(tx)

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

	if glog.V(logger.Debug) {
		glog.Infof("(t) %x => %s (%v) %x\n", from, toname, tx.Value, tx.Hash())
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

func (self *TxPool) GetTransactions() (txs types.Transactions) {
	self.mu.RLock()
	defer self.mu.RUnlock()

	txs = make(types.Transactions, self.Size())
	i := 0
	for _, tx := range self.txs {
		txs[i] = tx
		i++
	}

	return
}

func (self *TxPool) GetQueuedTransactions() types.Transactions {
	self.mu.RLock()
	defer self.mu.RUnlock()

	var txs types.Transactions
	for _, ts := range self.queue {
		txs = append(txs, ts...)
	}

	return txs
}

func (self *TxPool) RemoveTransactions(txs types.Transactions) {
	self.mu.Lock()
	defer self.mu.Unlock()

	for _, tx := range txs {
		delete(self.txs, tx.Hash())
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

func (self *TxPool) queueTx(tx *types.Transaction) {
	from, _ := tx.From()
	self.queue[from] = append(self.queue[from], tx)
}

func (pool *TxPool) addTx(tx *types.Transaction) {
	if _, ok := pool.txs[tx.Hash()]; !ok {
		pool.txs[tx.Hash()] = tx
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
	for address, txs := range pool.queue {
		sort.Sort(types.TxByNonce{txs})

		var (
			nonce = statedb.GetNonce(address)
			start int
		)
		// Clean up the transactions first and determine the start of the nonces
		for _, tx := range txs {
			if tx.Nonce() >= nonce {
				break
			}
			start++
		}
		pool.queue[address] = txs[start:]

		// expected nonce
		enonce := nonce
		for _, tx := range pool.queue[address] {
			// If the expected nonce does not match up with the next one
			// (i.e. a nonce gap), we stop the loop
			if enonce != tx.Nonce() {
				break
			}
			enonce++

			pool.addTx(tx)
		}
		// delete the entire queue entry if it's empty. There's no need to keep it
		if len(pool.queue[address]) == 0 {
			delete(pool.queue, address)
		}
	}
}

func (pool *TxPool) removeTx(hash common.Hash) {
	// delete from pending pool
	delete(pool.txs, hash)

	// delete from queue
out:
	for address, txs := range pool.queue {
		for i, tx := range txs {
			if tx.Hash() == hash {
				if len(txs) == 1 {
					// if only one tx, remove entire address entry
					delete(pool.queue, address)
				} else {
					pool.queue[address][len(txs)-1], pool.queue[address] = nil, append(txs[:i], txs[i+1:]...)
				}
				break out
			}
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

			pool.removeTx(hash)
		}
	}
}
