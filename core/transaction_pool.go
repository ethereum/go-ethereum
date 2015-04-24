package core

import (
	"errors"
	"fmt"
	"math/big"
	"sync"

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
	ErrImpossibleNonce    = errors.New("Impossible nonce")
	ErrNonExistentAccount = errors.New("Account does not exist")
	ErrInsufficientFunds  = errors.New("Insufficient funds")
	ErrIntrinsicGas       = errors.New("Intrinsic gas too low")
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
	// The actual pool
	txs           map[common.Hash]*types.Transaction
	invalidHashes *set.Set

	subscribers []chan TxMsg

	eventMux *event.TypeMux
}

func NewTxPool(eventMux *event.TypeMux, currentStateFn stateFn) *TxPool {
	return &TxPool{
		txs:           make(map[common.Hash]*types.Transaction),
		queueChan:     make(chan *types.Transaction, txPoolQueueSize),
		quit:          make(chan bool),
		eventMux:      eventMux,
		invalidHashes: set.New(),
		currentState:  currentStateFn,
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

	if pool.currentState().GetBalance(from).Cmp(new(big.Int).Mul(tx.Price, tx.GasLimit)) < 0 {
		return ErrInsufficientFunds
	}

	if tx.GasLimit.Cmp(IntrinsicGas(tx)) < 0 {
		return ErrIntrinsicGas
	}

	if pool.currentState().GetNonce(from) > tx.Nonce() {
		return ErrImpossibleNonce
	}

	return nil
}

func (self *TxPool) addTx(tx *types.Transaction) {
	self.txs[tx.Hash()] = tx
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

	self.addTx(tx)

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

	// Notify the subscribers
	go self.eventMux.Post(TxPreEvent{tx})

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
			glog.V(logger.Debug).Infoln(err)
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

func (self *TxPool) RemoveSet(txs types.Transactions) {
	self.mu.Lock()
	defer self.mu.Unlock()
	for _, tx := range txs {
		delete(self.txs, tx.Hash())
	}
}

func (self *TxPool) InvalidateSet(hashes *set.Set) {
	self.mu.Lock()
	defer self.mu.Unlock()

	hashes.Each(func(v interface{}) bool {
		delete(self.txs, v.(common.Hash))
		return true
	})
	self.invalidHashes.Merge(hashes)
}

func (pool *TxPool) Flush() {
	pool.txs = make(map[common.Hash]*types.Transaction)
}

func (pool *TxPool) Start() {
}

func (pool *TxPool) Stop() {
	pool.Flush()

	glog.V(logger.Info).Infoln("TX Pool stopped")
}
