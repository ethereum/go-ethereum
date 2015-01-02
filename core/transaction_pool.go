package core

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/logger"
	"gopkg.in/fatih/set.v0"
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

type TxProcessor interface {
	ProcessTransaction(tx *types.Transaction)
}

// The tx pool a thread safe transaction pool handler. In order to
// guarantee a non blocking pool we use a queue channel which can be
// independently read without needing access to the actual pool.
type TxPool struct {
	// Queueing channel for reading and writing incoming
	// transactions to
	queueChan chan *types.Transaction
	// Quiting channel
	quit chan bool
	// The actual pool
	//pool *list.List
	pool *set.Set

	SecondaryProcessor TxProcessor

	subscribers []chan TxMsg

	stateQuery StateQuery
	eventMux   *event.TypeMux
}

func NewTxPool(stateQuery StateQuery, eventMux *event.TypeMux) *TxPool {
	return &TxPool{
		pool:       set.New(),
		queueChan:  make(chan *types.Transaction, txPoolQueueSize),
		quit:       make(chan bool),
		stateQuery: stateQuery,
		eventMux:   eventMux,
	}
}

func (pool *TxPool) addTransaction(tx *types.Transaction) {

	pool.pool.Add(tx)

	// Broadcast the transaction to the rest of the peers
	pool.eventMux.Post(TxPreEvent{tx})
}

func (pool *TxPool) ValidateTransaction(tx *types.Transaction) error {
	if len(tx.To()) != 0 && len(tx.To()) != 20 {
		return fmt.Errorf("Invalid recipient. len = %d", len(tx.To()))
	}

	v, _, _ := tx.Curve()
	if v > 28 || v < 27 {
		return fmt.Errorf("tx.v != (28 || 27)")
	}

	// Get the sender
	senderAddr := tx.From()
	if senderAddr == nil {
		return fmt.Errorf("invalid sender")
	}
	sender := pool.stateQuery.GetAccount(senderAddr)

	totAmount := new(big.Int).Set(tx.Value())
	// Make sure there's enough in the sender's account. Having insufficient
	// funds won't invalidate this transaction but simple ignores it.
	if sender.Balance().Cmp(totAmount) < 0 {
		return fmt.Errorf("Insufficient amount in sender's (%x) account", tx.From())
	}

	return nil
}

func (self *TxPool) Add(tx *types.Transaction) error {
	hash := tx.Hash()
	if self.pool.Has(tx) {
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
	return self.pool.Size()
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
	txList := make([]*types.Transaction, pool.Size())
	i := 0
	pool.pool.Each(func(v interface{}) bool {
		txList[i] = v.(*types.Transaction)
		i++

		return true
	})

	return txList
}

func (pool *TxPool) RemoveInvalid(query StateQuery) {
	var removedTxs types.Transactions
	pool.pool.Each(func(v interface{}) bool {
		tx := v.(*types.Transaction)
		sender := query.GetAccount(tx.From())
		err := pool.ValidateTransaction(tx)
		if err != nil || sender.Nonce >= tx.Nonce() {
			removedTxs = append(removedTxs, tx)
		}

		return true
	})
	pool.RemoveSet(removedTxs)
}

func (self *TxPool) RemoveSet(txs types.Transactions) {
	for _, tx := range txs {
		self.pool.Remove(tx)
	}
}

func (pool *TxPool) Flush() []*types.Transaction {
	txList := pool.GetTransactions()
	pool.pool.Clear()

	return txList
}

func (pool *TxPool) Start() {
}

func (pool *TxPool) Stop() {
	pool.Flush()

	txplogger.Infoln("Stopped")
}
