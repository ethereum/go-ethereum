package core

import (
	"bytes"
	"container/list"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/state"
)

var txplogger = logger.NewLogger("TXP")

const txPoolQueueSize = 50

type TxPoolHook chan *types.Transaction
type TxMsg struct {
	Tx *types.Transaction
}

const (
	minGasPrice = 1000000
)

var MinGasPrice = big.NewInt(10000000000000)

func EachTx(pool *list.List, it func(*types.Transaction, *list.Element) bool) {
	for e := pool.Front(); e != nil; e = e.Next() {
		if it(e.Value.(*types.Transaction), e) {
			break
		}
	}
}

func FindTx(pool *list.List, finder func(*types.Transaction, *list.Element) bool) *types.Transaction {
	for e := pool.Front(); e != nil; e = e.Next() {
		if tx, ok := e.Value.(*types.Transaction); ok {
			if finder(tx, e) {
				return tx
			}
		}
	}

	return nil
}

type TxProcessor interface {
	ProcessTransaction(tx *types.Transaction)
}

// The tx pool a thread safe transaction pool handler. In order to
// guarantee a non blocking pool we use a queue channel which can be
// independently read without needing access to the actual pool. If the
// pool is being drained or synced for whatever reason the transactions
// will simple queue up and handled when the mutex is freed.
type TxPool struct {
	// The mutex for accessing the Tx pool.
	mutex sync.Mutex
	// Queueing channel for reading and writing incoming
	// transactions to
	queueChan chan *types.Transaction
	// Quiting channel
	quit chan bool
	// The actual pool
	pool *list.List

	SecondaryProcessor TxProcessor

	subscribers []chan TxMsg

	chainManager *ChainManager
	eventMux     *event.TypeMux
}

func NewTxPool(chainManager *ChainManager, eventMux *event.TypeMux) *TxPool {
	return &TxPool{
		pool:         list.New(),
		queueChan:    make(chan *types.Transaction, txPoolQueueSize),
		quit:         make(chan bool),
		chainManager: chainManager,
		eventMux:     eventMux,
	}
}

// Blocking function. Don't use directly. Use QueueTransaction instead
func (pool *TxPool) addTransaction(tx *types.Transaction) {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()

	pool.pool.PushBack(tx)

	// Broadcast the transaction to the rest of the peers
	pool.eventMux.Post(TxPreEvent{tx})
}

func (pool *TxPool) ValidateTransaction(tx *types.Transaction) error {
	// Get the last block so we can retrieve the sender and receiver from
	// the merkle trie
	block := pool.chainManager.CurrentBlock
	// Something has gone horribly wrong if this happens
	if block == nil {
		return fmt.Errorf("No last block on the block chain")
	}

	if len(tx.To()) != 0 && len(tx.To()) != 20 {
		return fmt.Errorf("Invalid recipient. len = %d", len(tx.To()))
	}

	v, _, _ := tx.Curve()
	if v > 28 || v < 27 {
		return fmt.Errorf("tx.v != (28 || 27)")
	}

	// Get the sender
	sender := pool.chainManager.State().GetAccount(tx.Sender())

	totAmount := new(big.Int).Set(tx.Value())
	// Make sure there's enough in the sender's account. Having insufficient
	// funds won't invalidate this transaction but simple ignores it.
	if sender.Balance().Cmp(totAmount) < 0 {
		return fmt.Errorf("Insufficient amount in sender's (%x) account", tx.From())
	}

	// Increment the nonce making each tx valid only once to prevent replay
	// attacks

	return nil
}

func (self *TxPool) Add(tx *types.Transaction) error {
	hash := tx.Hash()
	foundTx := FindTx(self.pool, func(tx *types.Transaction, e *list.Element) bool {
		return bytes.Compare(tx.Hash(), hash) == 0
	})

	if foundTx != nil {
		return fmt.Errorf("Known transaction (%x)", hash[0:4])
	}

	err := self.ValidateTransaction(tx)
	if err != nil {
		return err
	}

	self.addTransaction(tx)

	txplogger.Debugf("(t) %x => %x (%v) %x\n", tx.From()[:4], tx.To()[:4], tx.Value, tx.Hash())

	// Notify the subscribers
	go self.eventMux.Post(TxPreEvent{tx})

	return nil
}

func (self *TxPool) Size() int {
	return self.pool.Len()
}

func (self *TxPool) AddTransactions(txs []*types.Transaction) {
	for _, tx := range txs {
		if err := self.Add(tx); err != nil {
			txplogger.Infoln(err)
		} else {
			txplogger.Infof("tx %x\n", tx.Hash()[0:4])
		}
	}
}

func (pool *TxPool) GetTransactions() []*types.Transaction {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()

	txList := make([]*types.Transaction, pool.pool.Len())
	i := 0
	for e := pool.pool.Front(); e != nil; e = e.Next() {
		tx := e.Value.(*types.Transaction)

		txList[i] = tx

		i++
	}

	return txList
}

func (pool *TxPool) RemoveInvalid(state *state.StateDB) {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()

	for e := pool.pool.Front(); e != nil; e = e.Next() {
		tx := e.Value.(*types.Transaction)
		sender := state.GetAccount(tx.Sender())
		err := pool.ValidateTransaction(tx)
		if err != nil || sender.Nonce >= tx.Nonce() {
			pool.pool.Remove(e)
		}
	}
}

func (self *TxPool) RemoveSet(txs types.Transactions) {
	self.mutex.Lock()
	defer self.mutex.Unlock()

	for _, tx := range txs {
		EachTx(self.pool, func(t *types.Transaction, element *list.Element) bool {
			if t == tx {
				self.pool.Remove(element)
				return true // To stop the loop
			}
			return false
		})
	}
}

func (pool *TxPool) Flush() []*types.Transaction {
	txList := pool.GetTransactions()

	// Recreate a new list all together
	// XXX Is this the fastest way?
	pool.pool = list.New()

	return txList
}

func (pool *TxPool) Start() {
	//go pool.queueHandler()
}

func (pool *TxPool) Stop() {
	pool.Flush()

	txplogger.Infoln("Stopped")
}
