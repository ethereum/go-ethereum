package chain

import (
	"bytes"
	"container/list"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/chain/types"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/state"
	"github.com/ethereum/go-ethereum/wire"
)

var txplogger = logger.NewLogger("TXP")

const txPoolQueueSize = 50

type TxPoolHook chan *types.Transaction
type TxMsgTy byte

const (
	minGasPrice = 1000000
)

var MinGasPrice = big.NewInt(10000000000000)

type TxMsg struct {
	Tx   *types.Transaction
	Type TxMsgTy
}

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
	Ethereum EthManager
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
}

func NewTxPool(ethereum EthManager) *TxPool {
	return &TxPool{
		pool:      list.New(),
		queueChan: make(chan *types.Transaction, txPoolQueueSize),
		quit:      make(chan bool),
		Ethereum:  ethereum,
	}
}

// Blocking function. Don't use directly. Use QueueTransaction instead
func (pool *TxPool) addTransaction(tx *types.Transaction) {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()

	pool.pool.PushBack(tx)

	// Broadcast the transaction to the rest of the peers
	pool.Ethereum.Broadcast(wire.MsgTxTy, []interface{}{tx.RlpData()})
}

func (pool *TxPool) ValidateTransaction(tx *types.Transaction) error {
	// Get the last block so we can retrieve the sender and receiver from
	// the merkle trie
	block := pool.Ethereum.ChainManager().CurrentBlock
	// Something has gone horribly wrong if this happens
	if block == nil {
		return fmt.Errorf("No last block on the block chain")
	}

	if len(tx.Recipient) != 0 && len(tx.Recipient) != 20 {
		return fmt.Errorf("Invalid recipient. len = %d", len(tx.Recipient))
	}

	if tx.v > 28 || tx.v < 27 {
		return fmt.Errorf("tx.v != (28 || 27)")
	}

	if tx.GasPrice.Cmp(MinGasPrice) < 0 {
		return fmt.Errorf("Gas price to low. Require %v > Got %v", MinGasPrice, tx.GasPrice)
	}

	// Get the sender
	sender := pool.Ethereum.BlockManager().CurrentState().GetAccount(tx.Sender())

	totAmount := new(big.Int).Set(tx.Value)
	// Make sure there's enough in the sender's account. Having insufficient
	// funds won't invalidate this transaction but simple ignores it.
	if sender.Balance().Cmp(totAmount) < 0 {
		return fmt.Errorf("Insufficient amount in sender's (%x) account", tx.Sender())
	}

	if tx.IsContract() {
		if tx.GasPrice.Cmp(big.NewInt(minGasPrice)) < 0 {
			return fmt.Errorf("Gasprice too low, %s given should be at least %d.", tx.GasPrice, minGasPrice)
		}
	}

	// Increment the nonce making each tx valid only once to prevent replay
	// attacks

	return nil
}

func (self *TxPool) Add(tx *types.Transaction) error {
	hash := tx.Hash()
	foundTx := FindTx(self.pool, func(tx *Transaction, e *list.Element) bool {
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

	tmp := make([]byte, 4)
	copy(tmp, tx.Recipient)

	txplogger.Debugf("(t) %x => %x (%v) %x\n", tx.Sender()[:4], tmp, tx.Value, tx.Hash())

	// Notify the subscribers
	self.Ethereum.EventMux().Post(TxPreEvent{tx})

	return nil
}

func (pool *TxPool) queueHandler() {
out:
	for {
		select {
		case tx := <-pool.queueChan:
			hash := tx.Hash()
			foundTx := FindTx(pool.pool, func(tx *types.Transaction, e *list.Element) bool {
				return bytes.Compare(tx.Hash(), hash) == 0
			})

			if foundTx != nil {
				break
			}

			// Validate the transaction
			err := pool.ValidateTransaction(tx)
			if err != nil {
				txplogger.Debugln("Validating Tx failed", err)
			} else {
				// Call blocking version.
				pool.addTransaction(tx)

				tmp := make([]byte, 4)
				copy(tmp, tx.Recipient)

				txplogger.Debugf("(t) %x => %x (%v) %x\n", tx.Sender()[:4], tmp, tx.Value, tx.Hash())

				// Notify the subscribers
				pool.Ethereum.EventMux().Post(TxPreEvent{tx})
			}
		case <-pool.quit:
			break out
		}
	}
}

func (pool *TxPool) CurrentTransactions() []*types.Transaction {
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

func (pool *TxPool) RemoveInvalid(state *state.State) {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()

	for e := pool.pool.Front(); e != nil; e = e.Next() {
		tx := e.Value.(*types.Transaction)
		sender := state.GetAccount(tx.Sender())
		err := pool.ValidateTransaction(tx)
		if err != nil || sender.Nonce >= tx.Nonce {
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
	txList := pool.CurrentTransactions()

	// Recreate a new list all together
	// XXX Is this the fastest way?
	pool.pool = list.New()

	return txList
}

func (pool *TxPool) Start() {
	go pool.queueHandler()
}

func (pool *TxPool) Stop() {
	close(pool.quit)

	pool.Flush()

	txplogger.Infoln("Stopped")
}
