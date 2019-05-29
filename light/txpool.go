// Copyright 2016 The go-ethereum Authors
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

package light

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

const (
	chainHeadChanSize        = 10 // Size of channel listening to ChainHeadEvent.
	maxRelayTransactionCount = 4  // Amount of transactions to be relayed per request.
	maxStatusRequestCount    = 16 // Amount of hashes to be sent per status request.
	maxActiveTasks           = 16 // Maximum number of active running tasks.
	maxFetchFailureAllowance = 5  // Maximum number of failure allowance of data retrieval.
	maxQueryUnknownAllowance = 8  // Maximum number of failure allowance of transaction status query.

	reActiveTaskInterval = 5 * time.Second // The time interval for re-active task schedule.
)

var (
	// txPermanent is the number of mined blocks after a mined transaction is
	// considered finalized and no rollback is expected.
	txPermanent = uint64(64)

	// statusQueryResendDelay is the time duration for status query delay.
	statusQueryResendDelay = 5 * time.Second

	errPoolClosed            = errors.New("txpool is closed")
	errInvalidOldChain       = errors.New("invalid old chain")
	errInvalidNewChain       = errors.New("invalid new chain")
	errNilTransaction        = errors.New("the transaction is nil")
	errDuplicatedTransaction = errors.New("duplicated transaction")
	errInvalidNonce          = errors.New("transaction nonce is invalid")
)

// task wraps all necessary methods for txpool task.
type task interface {
	taskId() uint64
	error() error
	do(*TxPool)
}

// baseTask defines basic fields shared in different pool tasks.
type baseTask struct {
	id  uint64
	err error
}

func (t *baseTask) taskId() uint64 { return t.id }
func (t *baseTask) error() error   { return t.err }

// relayTask is generated to relay a local transaction to network.
type relayTask struct {
	baseTask
	txs types.Transactions
}

// do relays the given transactions to network and returns the unsend transactions.
func (t *relayTask) do(pool *TxPool) {
	t.err = pool.relay.Send(t.txs)
}

// queryTask is generated to query the transaction status.
type queryTask struct {
	baseTask
	hashes   []common.Hash
	response []TxStatus
}

// do sends a status query request and waits for the response.
func (t *queryTask) do(pool *TxPool) {
	ctx, cancelFn := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFn()

	t.response, t.err = GetTransactionStatus(ctx, pool.odr, t.hashes)
}

// queryTask is generated to fetch the missing block body and corresponding receipts.
type fetchTask struct {
	baseTask
	number    uint64
	blockHash common.Hash
}

// do retrieves the specified block body and receipts.
func (t *fetchTask) do(pool *TxPool) {
	ctx, cancelFn := context.WithTimeout(context.Background(), time.Second*5)
	defer cancelFn()

	if rawdb.HasBody(pool.chainDb, t.blockHash, t.number) && rawdb.HasReceipts(pool.chainDb, t.blockHash, t.number) {
		return
	}
	block, err := GetBlock(ctx, pool.odr, t.blockHash, t.number)
	if err != nil {
		t.err = err
		return
	}
	_, err = GetBlockReceipts(ctx, pool.odr, t.blockHash, t.number)
	if err != nil {
		t.err = err
		return
	}
	rawdb.WriteTxLookupEntries(pool.chainDb, block)
}

// newTxRequest represents a request to add a batch of new transactions.
type newTxRequest struct {
	txs   []*types.Transaction
	errCh chan error
}

// nonceRequest represents a request to fetch the nonce of specific address.
type nonceRequest struct {
	addr  common.Address
	resCh chan *uint64
}

// getTxRequest represents a request to fetch all pending transactions or
// a single specified pending transaction.
type getTxRequest struct {
	all   bool
	hash  common.Hash
	resCh chan []*types.Transaction
}

// TxRelayBackend provides an interface to the mechanism that forwards transactions
// to the ETH network. The implementations of the functions should be non-blocking.
//
// Send:
// 		instructs backend to forward new transactions
// Discard:
// 		notifies backend about transactions that should be discarded either
//  	because they have been replaced by a re-send or because they have been mined
//  	long ago and no rollback is expected
type TxRelayBackend interface {
	Send(txs types.Transactions) error
	Discard(hashes []common.Hash)
}

// TxPool implements the transaction pool for light clients, which keeps track
// of the status of locally created transactions, detecting if they are included
// in a block (mined) or rolled back. There are no queued transactions since we
// always receive all locally signed transactions in the same order as they are
// created.
type TxPool struct {
	config      core.TxPoolConfig
	chainConfig *params.ChainConfig
	signer      types.Signer
	txFeed      event.Feed
	scope       event.SubscriptionScope
	chain       *LightChain
	odr         OdrBackend
	chainDb     ethdb.Database
	relay       TxRelayBackend
	journal     *core.TxJournal // Journal of local transaction to back up to disk

	exitCh    chan struct{}
	nonceCh   chan *nonceRequest
	newTxCh   chan *newTxRequest
	getTxCh   chan *getTxRequest
	pendingCh chan chan int
	wg        sync.WaitGroup
}

// NewTxPool creates a new light transaction pool
func NewTxPool(config core.TxPoolConfig, chainConfig *params.ChainConfig, chain *LightChain, relay TxRelayBackend) *TxPool {
	// Sanitize transaction pool configuration
	if config.Rejournal < time.Second {
		log.Warn("Sanitizing invalid txpool journal time", "provided", config.Rejournal, "updated", time.Second)
		config.Rejournal = time.Second
	}
	pool := &TxPool{
		config:      config,
		chainConfig: chainConfig,
		signer:      types.NewEIP155Signer(chainConfig.ChainID),
		chain:       chain,
		relay:       relay,
		odr:         chain.Odr(),
		chainDb:     chain.Odr().Database(),

		exitCh:    make(chan struct{}),
		nonceCh:   make(chan *nonceRequest),
		newTxCh:   make(chan *newTxRequest),
		getTxCh:   make(chan *getTxRequest),
		pendingCh: make(chan chan int),
	}
	pool.wg.Add(1)
	go pool.mainLoop()

	// If local transactions and journaling is enabled, load from disk
	if config.Journal != "" {
		pool.journal = core.NewTxJournal(config.Journal)

		if err := pool.journal.Load(func(txs []*types.Transaction) []error {
			var (
				errs = make([]error, len(txs))
				ctx  = context.Background()
			)
			for _, tx := range txs {
				errs = append(errs, pool.Add(ctx, tx))
			}
			return errs
		}); err != nil {
			log.Warn("Failed to load transaction journal", "err", err)
		}
		txs, _ := pool.Content()
		if err := pool.journal.Rotate(txs); err != nil {
			log.Warn("Failed to rotate transaction journal", "err", err)
		}
	}

	return pool
}

// currentState returns the light state of the current head header
func (pool *TxPool) currentState(ctx context.Context) *state.StateDB {
	return NewState(ctx, pool.chain.CurrentHeader(), pool.odr)
}

// validateTx checks whether a transaction is valid according to the consensus rules.
func (pool *TxPool) validateTx(tx *types.Transaction, homestead bool) (from common.Address, err error) {
	ctx, cancelFn := context.WithTimeout(context.Background(), time.Second*5)
	defer cancelFn()

	// Validate the transaction sender and it's sig. Throw
	// if the from fields is invalid.
	if from, err = types.Sender(pool.signer, tx); err != nil {
		return from, core.ErrInvalidSender
	}
	// Last but not least check for nonce errors
	latestState := pool.currentState(ctx)
	if n := latestState.GetNonce(from); n > tx.Nonce() {
		return from, core.ErrNonceTooLow
	}
	// Check the transaction doesn't exceed the current
	// block limit gas.
	header := pool.chain.CurrentHeader()
	if header.GasLimit < tx.Gas() {
		return from, core.ErrGasLimit
	}
	// Transactions can't be negative. This may never happen
	// using RLP decoded transactions but may occur if you create
	// a transaction using the RPC for example.
	if tx.Value().Sign() < 0 {
		return from, core.ErrNegativeValue
	}
	// Transactor should have enough funds to cover the costs
	// cost == V + GP * GL
	if b := latestState.GetBalance(from); b.Cmp(tx.Cost()) < 0 {
		return from, core.ErrInsufficientFunds
	}
	// Should supply enough intrinsic gas
	gas, err := core.IntrinsicGas(tx.Data(), tx.To() == nil, homestead)
	if err != nil {
		return from, err
	}
	if tx.Gas() < gas {
		return from, core.ErrIntrinsicGas
	}
	return from, latestState.Error()
}

// reorg takes two blocks, an old chain and a new chain and returns all reverted block hashes.
func (pool *TxPool) reorg(newHeader *types.Header, oldHeader *types.Header) ([]common.Hash, error) {
	var oldHashes []common.Hash

	// first reduce whoever is higher bound
	if oldHeader.Number.Uint64() > newHeader.Number.Uint64() {
		// reduce old chain
		for ; oldHeader != nil && oldHeader.Number.Uint64() != newHeader.Number.Uint64(); oldHeader = pool.chain.GetHeader(oldHeader.ParentHash, oldHeader.Number.Uint64()-1) {
			oldHashes = append(oldHashes, oldHeader.Hash())
		}
	} else {
		// reduce new chain and append new chain blocks for inserting later on
		for ; newHeader != nil && newHeader.Number.Uint64() != oldHeader.Number.Uint64(); newHeader = pool.chain.GetHeader(newHeader.ParentHash, newHeader.Number.Uint64()-1) {
		}
	}
	if oldHeader == nil {
		return nil, errInvalidOldChain
	}
	if newHeader == nil {
		return nil, errInvalidNewChain
	}

	for {
		if oldHeader.Hash() == newHeader.Hash() {
			break
		}
		oldHashes = append(oldHashes, oldHeader.Hash())

		oldHeader, newHeader = pool.chain.GetHeader(oldHeader.ParentHash, oldHeader.Number.Uint64()-1), pool.chain.GetHeader(newHeader.ParentHash, newHeader.Number.Uint64()-1)
		if oldHeader == nil {
			return nil, errInvalidOldChain
		}
		if newHeader == nil {
			return nil, errInvalidNewChain
		}
	}
	return oldHashes, nil
}

// mainLoop is responsible for transaction relaying, status querying,
// missing block fetching and reorg logic handling.
func (pool *TxPool) mainLoop() {
	defer pool.wg.Done()

	// missingBlock represents a missing block with a batch of
	// locally created transaction hashes it contains.
	type missingBlock struct {
		number   uint64
		failure  int
		txHashes []common.Hash
	}
	// includedBlock represents a mined block with a batch of
	// locally created transactions it contains.
	type includedBlock struct {
		number uint64
		txs    []*types.Transaction
	}
	// sentTransaction represents a transaction which has been sent
	// via relay backend and some query statistic for better request
	// scheduling.
	type sentTransaction struct {
		tx         *types.Transaction
		unknown    int
		queryUntil time.Time
	}
	var (
		homestead    bool
		taskId       uint64
		runningTasks []task

		chainHeadSub event.Subscription
		chainHeadCh  = make(chan core.ChainHeadEvent, chainHeadChanSize)

		headHeader    = pool.chain.CurrentHeader()
		taskdone      = make(chan task, maxActiveTasks)
		nonces        = make(map[common.Address]uint64)          // The pending account nonce.
		pending       = make(map[common.Hash]*types.Transaction) // The transactions hasn't been sent.
		sent          = make(map[common.Hash]*sentTransaction)   // The transactions has been sent but not included.
		fetching      = make(map[common.Hash]*types.Transaction) // The transactions has been included but the block is missing.
		missingBlocks = make(map[common.Hash]*missingBlock)      // The missing blocks which contains some locally created transactions.
		included      = make(map[common.Hash]*includedBlock)     // The transactions has been included and retrieved but not finalized.
	)

	chainHeadSub = pool.chain.SubscribeChainHeadEvent(chainHeadCh)
	defer chainHeadSub.Unsubscribe()

	// startTasks spawns the given tasks and marks them
	// as the running tasks.
	startTasks := func(tasks []task) {
		for _, t := range tasks {
			go func(t task) { t.do(pool); taskdone <- t }(t)
		}
		runningTasks = append(runningTasks, tasks...)
	}
	// delTask removes t from runningTasks
	delTask := func(t task) {
		for i := range runningTasks {
			if runningTasks[i].taskId() == t.taskId() {
				runningTasks = append(runningTasks[:i], runningTasks[i+1:]...)
				break
			}
		}
	}

	timer := time.NewTimer(0)
	<-timer.C // discard the initial tick

	// scheduleTasks gathers sendable tasks as many as possible.
	// Transaction relay tasks always have the highest priority,
	// block retrieving tasks have the second highest priority.
	scheduleTasks := func(nofetch bool, noquery bool) {
		var tasks []task
		// Schedule transaction relay tasks first.
		if len(pending) > 0 && len(runningTasks) < maxActiveTasks {
			var txs types.Transactions
			for _, tx := range pending {
				txs = append(txs, tx)
			}
			for i := 0; i < len(txs) && len(tasks) < maxActiveTasks-len(runningTasks); i += maxRelayTransactionCount {
				if i+maxRelayTransactionCount <= len(txs) {
					tasks = append(tasks, &relayTask{baseTask: baseTask{id: taskId}, txs: txs[i : i+maxRelayTransactionCount]})
				} else {
					tasks = append(tasks, &relayTask{baseTask: baseTask{id: taskId}, txs: txs[i:]})
				}
				taskId += 1
			}
		}
		// Schedule fetching tasks then.
		if len(missingBlocks) > 0 && len(tasks) < maxActiveTasks-len(runningTasks) && !nofetch {
			for hash, block := range missingBlocks {
				if len(tasks) >= maxActiveTasks-len(runningTasks) {
					break
				}
				if block.number > headHeader.Number.Uint64() {
					continue
				}
				tasks = append(tasks, &fetchTask{baseTask: baseTask{id: taskId}, blockHash: hash, number: block.number})
				taskId += 1
			}
		}
		// Schedule status query tasks last.
		if len(sent) > 0 && len(tasks) < maxActiveTasks-len(runningTasks) && !noquery {
			var hashes []common.Hash
			for hash, info := range sent {
				// Ignore delayed status query tasks. Don't send this kind of
				// request too frequently to act as a DoS attacker.
				if time.Now().Before(info.queryUntil) {
					continue
				}
				hashes = append(hashes, hash)
			}
			for i := 0; i < len(hashes) && len(tasks) < maxActiveTasks-len(runningTasks); i += maxStatusRequestCount {
				if i+maxStatusRequestCount <= len(hashes) {
					tasks = append(tasks, &queryTask{baseTask: baseTask{id: taskId}, hashes: hashes[i : i+maxStatusRequestCount]})
				} else {
					tasks = append(tasks, &queryTask{baseTask: baseTask{id: taskId}, hashes: hashes[i:]})
				}
				taskId += 1
			}
		}
		startTasks(tasks)
		timer.Reset(reActiveTaskInterval)
	}
	// delBlock removes the specific block body, receipt and txlookup data from
	// the disk due to chain reorg.
	delBlock := func(hash common.Hash, number uint64) {
		batch := pool.chainDb.NewBatch()
		defer batch.Write()

		rawdb.DeleteTxLookupEntry(batch, hash)
		rawdb.DeleteBody(batch, hash, number)
		rawdb.DeleteReceipts(batch, hash, number)
	}
	// gatherAll retrieves all transactions which are not included in the chain.
	gatherAll := func() []*types.Transaction {
		var txs []*types.Transaction
		for _, tx := range pending {
			txs = append(txs, tx)
		}
		for _, tx := range sent {
			txs = append(txs, tx.tx)
		}
		for _, tx := range fetching {
			txs = append(txs, tx)
		}
		return txs
	}
	// replace replaces the original transaction with a new
	// one which has the same nonce but different hash.
	// Note, we don't have ANY guarantee that the resend can
	// be successful.
	replace := func(tx *types.Transaction) {
		var replaced bool
		for h, p := range pending {
			if p.Nonce() == tx.Nonce() {
				delete(pending, h)
				pending[tx.Hash()] = tx
				replaced = true
				break
			}
		}
		if !replaced {
			for h, s := range sent {
				if s.tx.Nonce() == tx.Nonce() {
					delete(sent, h)
					pending[tx.Hash()] = tx
					break
				}
			}
		}
	}

	journal := time.NewTicker(pool.config.Rejournal)
	defer journal.Stop()

running:
	for {
		select {
		case req := <-pool.newTxCh:
			// This channel is used to receive new locally
			// created transactions.

			// Short circuit if there is no new transaction.
			if len(req.txs) == 0 {
				continue
			}
			for _, tx := range req.txs {
				var injected bool
				if tx == nil {
					req.errCh <- errNilTransaction
				}
				// Ensure the transaction is not duplicated.
				hash := tx.Hash()
				if _, ok := pending[hash]; ok {
					req.errCh <- errDuplicatedTransaction
				}
				if _, ok := sent[hash]; ok {
					req.errCh <- errDuplicatedTransaction
				}
				if _, ok := fetching[hash]; ok {
					req.errCh <- errDuplicatedTransaction
				}
				// Check the correctness of the transaction.
				from, err := pool.validateTx(tx, homestead)
				if err != nil {
					req.errCh <- err
				}
				if tx.Nonce() > nonces[from] {
					// We don't accept transaction which has nonce gap.
					req.errCh <- errInvalidNonce
				} else if tx.Nonce() < nonces[from] {
					replace(tx)
					req.errCh <- nil
					injected = true
				} else {
					pending[hash], nonces[from] = tx, tx.Nonce()+1
					req.errCh <- nil
					injected = true
				}
				if pool.journal != nil && injected {
					if err := pool.journal.Insert(tx); err != nil {
						log.Warn("Failed to journal local transaction", "err", err)
					}
				}
			}
			scheduleTasks(true, true) // Only schedule transaction relay tasks.

		case t := <-taskdone:
			// This channel is used to trace all running tasks
			// and process them when tasks are finished.
			var (
				failed bool
				noPeer bool
			)
			if err := t.error(); err != nil {
				failed, noPeer = true, err == ErrNoPeers
			}
			switch tt := t.(type) {
			case *relayTask:
				if !failed {
					for _, tx := range tt.txs {
						hash := tx.Hash()
						// The scheduler may initiate multiple relay tasks for the
						// same pending transaction. Ignore the non-existed pending
						// transaction here to avoid the effects of duplicate tasks.
						if _, exist := pending[hash]; !exist {
							continue
						}
						delete(pending, hash)
						sent[hash] = &sentTransaction{tx: tx, queryUntil: time.Now()}
					}
				} else {
					// The relay task is failed mostly because of there is no available
					// peer, just ignore the error here and wait next round wave scheduling.
				}
			case *queryTask:
				if !failed {
					for i, hash := range tt.hashes {
						// The scheduler may initiate multiple query tasks for the
						// same transaction. Ignore the non-existed sent transaction
						// here to avoid the effects of duplicate tasks.
						if _, exist := sent[hash]; !exist {
							continue
						}
						stat, info := tt.response[i], sent[hash]
						if stat.Status == core.TxStatusIncluded {
							block := missingBlocks[stat.Lookup.BlockHash]
							if block == nil {
								block = &missingBlock{number: stat.Lookup.BlockIndex}
								missingBlocks[stat.Lookup.BlockHash] = block
							}
							block.txHashes = append(block.txHashes, hash)
							fetching[hash] = info.tx
							delete(sent, hash)
						} else {
							// If the status query task fails a sufficient number of times,
							// then discard the task and re-send the transaction.
							// The following scenarios may cause these situations.
							// * client relays the transaction to some servers while
							//   the transaction fee is too low so that the transaction
							//   is discarded later.
							// * client relays the transaction to some malicious servers.
							info.unknown += 1
							if info.unknown > maxQueryUnknownAllowance {
								pending[hash] = info.tx
								delete(sent, hash)
								log.Debug("Discard query task", "txhash", info.tx.Hash())
							} else {
								// Linearly increase the request delay based on the number
								// of failed requests sent.
								// It can take several seconds to a few minutes for a transaction
								// to be mined from initiation to packaging.
								info.queryUntil = time.Now().Add(time.Duration(sent[hash].unknown) * statusQueryResendDelay)
								log.Debug("Delay query task", "txhash", info.tx.Hash(), "delay", common.PrettyDuration(time.Duration(sent[hash].unknown)*statusQueryResendDelay))
							}
						}
					}
				} else {
					// Do nothing, wait scheduler to schedule another wave of tasks.
				}
			case *fetchTask:
				// The scheduler may initiate multiple fetch tasks for the
				// same missing block. Ignore the non-existed block here
				// to avoid the effects of duplicate tasks.
				missBlock := missingBlocks[tt.blockHash]
				if missBlock == nil {
					continue
				}
				if failed {
					// If the fetching task fails a sufficient number of times,
					// then discard the task and re-query the status of transaction.
					// Otherwise there is an attack vector here, malicious nodes
					// can feed fake status message and light client will fall into
					// an infinite loop to retrieve some non-existent data.
					missBlock.failure += 1
					if missBlock.failure >= maxFetchFailureAllowance {
						for _, hash := range missBlock.txHashes {
							sent[hash] = &sentTransaction{tx: fetching[hash], queryUntil: time.Now()}
							delete(fetching, hash)
						}
						delete(missingBlocks, tt.blockHash)
						log.Debug("Discard fetching task", "hash", tt.blockHash, "number", tt.number)
					}
				} else {
					block := included[tt.blockHash]
					if block == nil {
						block = &includedBlock{number: tt.number}
						included[tt.blockHash] = block
					}
					for _, hash := range missBlock.txHashes {
						block.txs = append(block.txs, fetching[hash])
						delete(fetching, hash)
					}
					delete(missingBlocks, tt.blockHash)
					log.Debug("Fetch block successfully", "hash", tt.blockHash, "number", block.number)
				}
			}
			delTask(t)
			if !noPeer {
				scheduleTasks(false, false)
			}

		case ev := <-chainHeadCh:
			// This channel is used to receive new chain head for
			// reorg logic handling.
			oldHashes, err := pool.reorg(ev.Block.Header(), headHeader)
			if err != nil {
				continue
			}
			headHeader, homestead = ev.Block.Header(), pool.chainConfig.IsHomestead(ev.Block.Number())

			var reschedule bool
			for _, hash := range oldHashes {
				// Demote all included transactions(although the corresponding
				// blocks are still missing) to pending status.
				// Note, since we don't have block bodies for new canonical
				// chain, so that we can't determine whether the reorged txs
				// have been included in the canonical blocks.
				if block, ok := missingBlocks[hash]; ok {
					for _, txHash := range block.txHashes {
						pending[txHash] = fetching[txHash]
						delete(fetching, txHash)
					}
					delete(missingBlocks, hash)
					reschedule = true
					log.Debug("Demote transactions due to chain reorg", "blockhash", hash, "number", block.number)
				}
				// Demote all included transactions to pending status.
				if block, ok := included[hash]; ok {
					for _, tx := range block.txs {
						pending[tx.Hash()] = tx
					}
					delBlock(hash, block.number)
					delete(included, hash)
					reschedule = true
					log.Debug("Demote transactions due to chain reorg", "blockhash", hash, "number", block.number)
				}
			}
			// If the included transactions are stable enough, finalize them.
			var hashes []common.Hash
			for hash, block := range included {
				if block.number+txPermanent <= headHeader.Number.Uint64() {
					for _, tx := range block.txs {
						hashes = append(hashes, tx.Hash())
					}
					delete(included, hash)
					log.Debug("Finalize block", "hash", hash, "number", block.number)
				}
			}
			if len(hashes) > 0 {
				pool.relay.Discard(hashes)
			}
			if reschedule {
				scheduleTasks(true, true)
			}
		case <-timer.C:
			// This channel is used to trigger next wave of
			// task scheduling to ensure the liveness of the
			// scheduler.
			scheduleTasks(false, false)
		case req := <-pool.nonceCh:
			// This channel is used to request account pending
			// nonce of specified address.
			if nonce, exist := nonces[req.addr]; exist {
				req.resCh <- &nonce
			} else {
				req.resCh <- nil
			}
		case req := <-pool.getTxCh:
			// This channel is used to request pending transactions
			// with specified hash or all pending transactions.
			var txs []*types.Transaction
			if req.all {
				txs = gatherAll()
			} else {
				txs = append(txs, pending[req.hash])
			}
			req.resCh <- txs
		case <-journal.C:
			// This channel is used to dump all local pending transactions
			// into tx journal.
			if pool.journal != nil {
				// Retrieve all the pending transactions and sort by account and by nonce
				locals := make(map[common.Address]types.Transactions)
				for _, tx := range gatherAll() {
					account, _ := types.Sender(pool.signer, tx)
					locals[account] = append(locals[account], tx)
				}
				for _, txs := range locals {
					sort.Sort(types.TxByNonce(txs))
				}
				if err := pool.journal.Rotate(locals); err != nil {
					log.Warn("Failed to rotate local tx journal", "err", err)
				}
			}
		case <-pool.exitCh:
			break running
		}
	}
	log.Debug("Txpool is spinning down")
}

// Add validates a new transaction and sets its state pending if processable.
// It also updates the locally stored nonce if necessary.
func (pool *TxPool) Add(ctx context.Context, tx *types.Transaction) error {
	errCh := make(chan error, 1)

	select {
	case pool.newTxCh <- &newTxRequest{txs: []*types.Transaction{tx}, errCh: errCh}:
	case <-pool.exitCh:
		return errPoolClosed
	}
	if err := <-errCh; err != nil {
		return err
	}
	// Notify the subscribers. This event is posted in a goroutine
	// because it's possible that somewhere during the post "Remove transaction"
	// gets called which will then wait for the global tx pool lock and deadlock.
	go pool.txFeed.Send(core.NewTxsEvent{Txs: types.Transactions{tx}})

	log.Debug("Pooled new transaction", "hash", tx.Hash(), "from", log.Lazy{Fn: func() common.Address { from, _ := types.Sender(pool.signer, tx); return from }}, "to", tx.To())
	return nil
}

// AddTransactions adds all valid transactions to the pool and passes them to
// the tx relay backend
func (pool *TxPool) AddBatch(ctx context.Context, txs []*types.Transaction) error {
	errCh := make(chan error, len(txs))

	select {
	case pool.newTxCh <- &newTxRequest{txs: txs, errCh: errCh}:
	case <-pool.exitCh:
		return errPoolClosed
	}
	// Wait for the transaction injection result.
	var (
		errs  []error
		added []*types.Transaction
	)
	for i := 0; i < len(txs); i++ {
		if err := <-errCh; err != nil {
			log.Warn("Failed to pool new transaction", "hash", txs[i].Hash(), "from", log.Lazy{Fn: func() common.Address { from, _ := types.Sender(pool.signer, txs[i]); return from }}, "to", txs[i].To(), "error", err)
			errs = append(errs, err)
			continue
		}
		added = append(added, txs[i])
		log.Debug("Pooled new transaction", "hash", txs[i].Hash(), "from", log.Lazy{Fn: func() common.Address { from, _ := types.Sender(pool.signer, txs[i]); return from }}, "to", txs[i].To())
	}
	// Notify the subscribers. This event is posted in a goroutine
	// because it's possible that somewhere during the post "Remove transaction"
	// gets called which will then wait for the global tx pool lock and deadlock.
	go pool.txFeed.Send(core.NewTxsEvent{Txs: added})

	if len(errs) != 0 {
		return fmt.Errorf("%v", errs)
	}
	return nil
}

// GetPending returns the number of currently pending (locally created) transactions
func (pool *TxPool) GetPending() (int, error) {
	resCh := make(chan int, 1)
	select {
	case pool.pendingCh <- resCh:
		return <-resCh, nil
	case <-pool.exitCh:
		return 0, errPoolClosed
	}
}

// GetNonce returns the "pending" nonce of a given address. It always queries
// the nonce belonging to the latest header too in order to detect if another
// client using the same key sent a transaction.
func (pool *TxPool) GetNonce(ctx context.Context, addr common.Address) (uint64, error) {
	resCh := make(chan *uint64, 1)

	select {
	case pool.nonceCh <- &nonceRequest{addr: addr, resCh: resCh}:
	case <-pool.exitCh:
		return 0, errPoolClosed
	}
	if nonce := <-resCh; nonce != nil {
		return *nonce, nil
	}
	state := pool.currentState(ctx)
	nonce := state.GetNonce(addr)
	if state.Error() != nil {
		return 0, state.Error()
	}
	return nonce, nil
}

// GetPendingTransaction returns a transaction if it is contained in the pool
// and nil otherwise.
func (pool *TxPool) GetPendingTransaction(hash common.Hash) (*types.Transaction, error) {
	resCh := make(chan []*types.Transaction, 1)

	select {
	case pool.getTxCh <- &getTxRequest{hash: hash, resCh: resCh}:
		txs := <-resCh
		if len(txs) == 1 {
			return txs[0], nil
		}
		return nil, nil
	case <-pool.exitCh:
		return nil, errPoolClosed
	}
}

// GetAllPendingTransactions returns all currently processable transactions.
// The returned slice may be modified by the caller.
func (pool *TxPool) GetAllPendingTransactions() (txs types.Transactions, err error) {
	resCh := make(chan []*types.Transaction, 1)

	select {
	case pool.getTxCh <- &getTxRequest{all: true, resCh: resCh}:
		return <-resCh, nil
	case <-pool.exitCh:
		return nil, errPoolClosed
	}
}

// Content retrieves the data content of the transaction pool, returning all the
// pending as well as queued transactions, grouped by account and nonce.
func (pool *TxPool) Content() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	txs, err := pool.GetAllPendingTransactions()
	if err != nil {
		return nil, nil
	}
	// Retrieve all the pending transactions and sort by account and by nonce
	pending := make(map[common.Address]types.Transactions)
	for _, tx := range txs {
		account, _ := types.Sender(pool.signer, tx)
		pending[account] = append(pending[account], tx)
	}
	for _, txs := range pending {
		sort.Sort(types.TxByNonce(txs))
	}
	// There are no queued transactions in a light pool, just return an empty map
	queued := make(map[common.Address]types.Transactions)
	return pending, queued
}

// SubscribeNewTxsEvent registers a subscription of core.NewTxsEvent and
// starts sending event to the given channel.
func (pool *TxPool) SubscribeNewTxsEvent(ch chan<- core.NewTxsEvent) event.Subscription {
	return pool.scope.Track(pool.txFeed.Subscribe(ch))
}

// Stop stops the light transaction pool
func (pool *TxPool) Stop() {
	pool.scope.Close() // Unsubscribe all subscriptions registered from txpool
	close(pool.exitCh)
	pool.wg.Wait()

	if pool.journal != nil {
		pool.journal.Close()
	}
	log.Info("Transaction pool stopped")
}
