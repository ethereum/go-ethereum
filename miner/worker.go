// Copyright 2015 The go-ethereum Authors
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

package miner

import (
	"errors"
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rollup/circuitcapacitychecker"
	"github.com/ethereum/go-ethereum/rollup/tracing"
	"github.com/ethereum/go-ethereum/trie"
)

const (
	// resultQueueSize is the size of channel listening to sealing result.
	resultQueueSize = 10

	// txChanSize is the size of channel listening to NewTxsEvent.
	// The number is referenced from the size of tx pool.
	txChanSize = 4096

	// chainHeadChanSize is the size of channel listening to ChainHeadEvent.
	chainHeadChanSize = 10

	// resubmitAdjustChanSize is the size of resubmitting interval adjustment channel.
	resubmitAdjustChanSize = 10

	// minRecommitInterval is the minimal time interval to recreate the sealing block with
	// any newly arrived transactions.
	minRecommitInterval = 1 * time.Second

	// maxRecommitInterval is the maximum time interval to recreate the sealing block with
	// any newly arrived transactions.
	maxRecommitInterval = 15 * time.Second

	// intervalAdjustRatio is the impact a single interval adjustment has on sealing work
	// resubmitting interval.
	intervalAdjustRatio = 0.1

	// intervalAdjustBias is applied during the new resubmit interval calculation in favor of
	// increasing upper limit or decreasing lower limit so that the limit can be reachable.
	intervalAdjustBias = 200 * 1000.0 * 1000.0

	// staleThreshold is the maximum depth of the acceptable stale block.
	staleThreshold = 7
)

var (
	errBlockInterruptedByNewHead  = errors.New("new head arrived while building block")
	errBlockInterruptedByRecommit = errors.New("recommit interrupt while building block")
	errBlockInterruptedByTimeout  = errors.New("timeout while building block")
)

var (
	// Metrics for the skipped txs
	l1TxGasLimitExceededCounter       = metrics.NewRegisteredCounter("miner/skipped_txs/l1/gas_limit_exceeded", nil)
	l1TxRowConsumptionOverflowCounter = metrics.NewRegisteredCounter("miner/skipped_txs/l1/row_consumption_overflow", nil)
	l2TxRowConsumptionOverflowCounter = metrics.NewRegisteredCounter("miner/skipped_txs/l2/row_consumption_overflow", nil)
	l1TxCccUnknownErrCounter          = metrics.NewRegisteredCounter("miner/skipped_txs/l1/ccc_unknown_err", nil)
	l2TxCccUnknownErrCounter          = metrics.NewRegisteredCounter("miner/skipped_txs/l2/ccc_unknown_err", nil)
	l1TxStrangeErrCounter             = metrics.NewRegisteredCounter("miner/skipped_txs/l1/strange_err", nil)

	l2CommitTxsTimer                = metrics.NewRegisteredTimer("miner/commit/txs_all", nil)
	l2CommitTxTimer                 = metrics.NewRegisteredTimer("miner/commit/tx_all", nil)
	l2CommitTxFailedTimer           = metrics.NewRegisteredTimer("miner/commit/tx_all_failed", nil)
	l2CommitTxTraceTimer            = metrics.NewRegisteredTimer("miner/commit/tx_trace", nil)
	l2CommitTxTraceStateRevertTimer = metrics.NewRegisteredTimer("miner/commit/tx_trace_state_revert", nil)
	l2CommitTxCCCTimer              = metrics.NewRegisteredTimer("miner/commit/tx_ccc", nil)
	l2CommitTxApplyTimer            = metrics.NewRegisteredTimer("miner/commit/tx_apply", nil)

	l2CommitNewWorkTimer                    = metrics.NewRegisteredTimer("miner/commit/new_work_all", nil)
	l2CommitNewWorkL1CollectTimer           = metrics.NewRegisteredTimer("miner/commit/new_work_collect_l1", nil)
	l2CommitNewWorkPrepareTimer             = metrics.NewRegisteredTimer("miner/commit/new_work_prepare", nil)
	l2CommitNewWorkCommitUncleTimer         = metrics.NewRegisteredTimer("miner/commit/new_work_uncle", nil)
	l2CommitNewWorkTidyPendingTxTimer       = metrics.NewRegisteredTimer("miner/commit/new_work_tidy_pending", nil)
	l2CommitNewWorkCommitL1MsgTimer         = metrics.NewRegisteredTimer("miner/commit/new_work_commit_l1_msg", nil)
	l2CommitNewWorkPrioritizedTxCommitTimer = metrics.NewRegisteredTimer("miner/commit/new_work_prioritized", nil)
	l2CommitNewWorkRemoteLocalCommitTimer   = metrics.NewRegisteredTimer("miner/commit/new_work_remote_local", nil)
	l2CommitNewWorkLocalPriceAndNonceTimer  = metrics.NewRegisteredTimer("miner/commit/new_work_local_price_and_nonce", nil)
	l2CommitNewWorkRemotePriceAndNonceTimer = metrics.NewRegisteredTimer("miner/commit/new_work_remote_price_and_nonce", nil)

	l2CommitTimer      = metrics.NewRegisteredTimer("miner/commit/all", nil)
	l2CommitTraceTimer = metrics.NewRegisteredTimer("miner/commit/trace", nil)
	l2CommitCCCTimer   = metrics.NewRegisteredTimer("miner/commit/ccc", nil)
	l2ResultTimer      = metrics.NewRegisteredTimer("miner/result/all", nil)
)

// environment is the worker's current environment and holds all
// information of the sealing block generation.
type environment struct {
	signer    types.Signer
	state     *state.StateDB // apply state changes here
	tcount    int            // tx count in cycle
	blockSize uint64         // approximate size of tx payload in bytes
	gasPool   *core.GasPool  // available gas used to pack transactions
	coinbase  common.Address

	header   *types.Header
	txs      []*types.Transaction
	receipts []*types.Receipt
	sidecars []*types.BlobTxSidecar
	blobs    int

	l1TxCount int // l1 msg count in cycle

	// circuit capacity check related fields
	traceEnv       *tracing.TraceEnv     // env for tracing
	accRows        *types.RowConsumption // accumulated row consumption for a block
	nextL1MsgIndex uint64                // next L1 queue index to be processed
}

// copy creates a deep copy of environment.
func (env *environment) copy() *environment {
	cpy := &environment{
		signer:   env.signer,
		state:    env.state.Copy(),
		tcount:   env.tcount,
		coinbase: env.coinbase,
		header:   types.CopyHeader(env.header),
		receipts: copyReceipts(env.receipts),
	}
	if env.gasPool != nil {
		gasPool := *env.gasPool
		cpy.gasPool = &gasPool
	}
	cpy.txs = make([]*types.Transaction, len(env.txs))
	copy(cpy.txs, env.txs)

	cpy.sidecars = make([]*types.BlobTxSidecar, len(env.sidecars))
	copy(cpy.sidecars, env.sidecars)

	return cpy
}

// discard terminates the background prefetcher go-routine. It should
// always be called for all created environment instances otherwise
// the go-routine leak can happen.
func (env *environment) discard() {
	if env.state == nil {
		return
	}
	env.state.StopPrefetcher()
}

// task contains all information for consensus engine sealing and result submitting.
type task struct {
	receipts  []*types.Receipt
	state     *state.StateDB
	block     *types.Block
	createdAt time.Time

	accRows        *types.RowConsumption // accumulated row consumption in the circuit side
	nextL1MsgIndex uint64                // next L1 queue index to be processed
}

const (
	commitInterruptNone int32 = iota
	commitInterruptNewHead
	commitInterruptResubmit
	commitInterruptTimeout
)

// newWorkReq represents a request for new sealing work submitting with relative interrupt notifier.
type newWorkReq struct {
	interrupt *atomic.Int32
	timestamp int64
}

// newPayloadResult is the result of payload generation.
type newPayloadResult struct {
	err      error
	block    *types.Block
	fees     *big.Int               // total block fees
	sidecars []*types.BlobTxSidecar // collected blobs of blob transactions
}

// getWorkReq represents a request for getting a new sealing work with provided parameters.
type getWorkReq struct {
	params *generateParams
	result chan *newPayloadResult // non-blocking channel
}

// intervalAdjust represents a resubmitting interval adjustment.
type intervalAdjust struct {
	ratio float64
	inc   bool
}

// prioritizedTransaction represents a single transaction that
// should be processed as the first transaction in the next block.
type prioritizedTransaction struct {
	blockNumber uint64
	tx          *types.Transaction
}

// worker is the main object which takes care of submitting new work to consensus engine
// and gathering the sealing result.
type worker struct {
	config      *Config
	chainConfig *params.ChainConfig
	engine      consensus.Engine
	eth         Backend
	chain       *core.BlockChain

	// Feeds
	pendingLogsFeed event.Feed

	// Subscriptions
	mux          *event.TypeMux
	txsCh        chan core.NewTxsEvent
	txsSub       event.Subscription
	chainHeadCh  chan core.ChainHeadEvent
	chainHeadSub event.Subscription
	l1MsgsCh     chan core.NewL1MsgsEvent
	l1MsgsSub    event.Subscription

	// Channels
	newWorkCh          chan *newWorkReq
	getWorkCh          chan *getWorkReq
	taskCh             chan *task
	resultCh           chan *types.Block
	startCh            chan struct{}
	exitCh             chan struct{}
	resubmitIntervalCh chan time.Duration
	resubmitAdjustCh   chan *intervalAdjust

	wg sync.WaitGroup

	current *environment // An environment for current running cycle.

	mu       sync.RWMutex // The lock used to protect the coinbase and extra fields
	coinbase common.Address
	extra    []byte

	pendingMu    sync.RWMutex
	pendingTasks map[common.Hash]*task

	snapshotMu       sync.RWMutex // The lock used to protect the snapshots below
	snapshotBlock    *types.Block
	snapshotReceipts types.Receipts
	snapshotState    *state.StateDB

	// atomic status counters
	running   atomic.Bool  // The indicator whether the consensus engine is running or not.
	newTxs    atomic.Int32 // New arrival transaction count since last sealing work submitting.
	syncing   atomic.Bool  // The indicator whether the node is still syncing.
	newL1Msgs atomic.Int32 // New arrival L1 message count since last sealing work submitting.

	// newpayloadTimeout is the maximum timeout allowance for creating payload.
	// The default value is 2 seconds but node operator can set it to arbitrary
	// large value. A large timeout allowance may cause Geth to fail creating
	// a non-empty payload within the specified time and eventually miss the slot
	// in case there are some computation expensive transactions in txpool.
	newpayloadTimeout time.Duration

	// recommit is the time interval to re-create sealing work or to re-build
	// payload in proof-of-stake stage.
	recommit time.Duration

	// External functions
	isLocalBlock func(header *types.Header) bool // Function used to determine whether the specified block is mined by local miner.

	circuitCapacityChecker *circuitcapacitychecker.CircuitCapacityChecker
	prioritizedTx          *prioritizedTransaction

	// Test hooks
	newTaskHook  func(*task)                        // Method to call upon receiving a new sealing task.
	skipSealHook func(*task) bool                   // Method to decide whether skipping the sealing.
	fullTaskHook func()                             // Method to call before pushing the full sealing task.
	resubmitHook func(time.Duration, time.Duration) // Method to call upon updating resubmitting interval.
	beforeTxHook func()                             // Method to call before processing a transaction.
}

func newWorker(config *Config, chainConfig *params.ChainConfig, engine consensus.Engine, eth Backend, mux *event.TypeMux, isLocalBlock func(header *types.Header) bool, init bool) *worker {
	worker := &worker{
		config:             config,
		chainConfig:        chainConfig,
		engine:             engine,
		eth:                eth,
		chain:              eth.BlockChain(),
		mux:                mux,
		isLocalBlock:       isLocalBlock,
		coinbase:           config.Etherbase,
		extra:              config.ExtraData,
		pendingTasks:       make(map[common.Hash]*task),
		txsCh:              make(chan core.NewTxsEvent, txChanSize),
		chainHeadCh:        make(chan core.ChainHeadEvent, chainHeadChanSize),
		newWorkCh:          make(chan *newWorkReq),
		getWorkCh:          make(chan *getWorkReq),
		taskCh:             make(chan *task),
		resultCh:           make(chan *types.Block, resultQueueSize),
		startCh:            make(chan struct{}, 1),
		exitCh:             make(chan struct{}),
		resubmitIntervalCh: make(chan time.Duration),
		resubmitAdjustCh:   make(chan *intervalAdjust, resubmitAdjustChanSize),

		l1MsgsCh:               make(chan core.NewL1MsgsEvent, txChanSize),
		circuitCapacityChecker: circuitcapacitychecker.NewCircuitCapacityChecker(true),
	}
	log.Info("created new worker", "CircuitCapacityChecker ID", worker.circuitCapacityChecker.ID)

	// Subscribe for transaction insertion events (whether from network or resurrects)
	worker.txsSub = eth.TxPool().SubscribeTransactions(worker.txsCh, true)
	// Subscribe events for blockchain
	worker.chainHeadSub = eth.BlockChain().SubscribeChainHeadEvent(worker.chainHeadCh)

	// Subscribe NewL1MsgsEvent for sync service
	if s := eth.SyncService(); s != nil {
		worker.l1MsgsSub = s.SubscribeNewL1MsgsEvent(worker.l1MsgsCh)
	} else {
		// create an empty subscription so that the tests won't fail
		worker.l1MsgsSub = event.NewSubscription(func(quit <-chan struct{}) error {
			<-quit
			return nil
		})
	}

	// Sanitize recommit interval if the user-specified one is too short.
	recommit := worker.config.Recommit
	if recommit < minRecommitInterval {
		log.Warn("Sanitizing miner recommit interval", "provided", recommit, "updated", minRecommitInterval)
		recommit = minRecommitInterval
	}
	worker.recommit = recommit

	// Sanitize the timeout config for creating payload.
	newpayloadTimeout := worker.config.NewPayloadTimeout
	if newpayloadTimeout == 0 {
		log.Warn("Sanitizing new payload timeout to default", "provided", newpayloadTimeout, "updated", DefaultConfig.NewPayloadTimeout)
		newpayloadTimeout = DefaultConfig.NewPayloadTimeout
	}
	if newpayloadTimeout < time.Millisecond*100 {
		log.Warn("Low payload timeout may cause high amount of non-full blocks", "provided", newpayloadTimeout, "default", DefaultConfig.NewPayloadTimeout)
	}
	worker.newpayloadTimeout = newpayloadTimeout

	worker.wg.Add(4)
	go worker.mainLoop()
	go worker.newWorkLoop(recommit)
	go worker.resultLoop()
	go worker.taskLoop()

	// Submit first work to initialize pending state.
	if init {
		worker.startCh <- struct{}{}
	}
	return worker
}

// getCCC returns a pointer to this worker's CCC instance.
// Only used in tests.
func (w *worker) getCCC() *circuitcapacitychecker.CircuitCapacityChecker {
	return w.circuitCapacityChecker
}

// setEtherbase sets the etherbase used to initialize the block coinbase field.
func (w *worker) setEtherbase(addr common.Address) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.coinbase = addr
}

// etherbase retrieves the configured etherbase address.
func (w *worker) etherbase() common.Address {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.coinbase
}

func (w *worker) setGasCeil(ceil uint64) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.config.GasCeil = ceil
}

// setExtra sets the content used to initialize the block extra field.
func (w *worker) setExtra(extra []byte) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.extra = extra
}

// setRecommitInterval updates the interval for miner sealing work recommitting.
func (w *worker) setRecommitInterval(interval time.Duration) {
	select {
	case w.resubmitIntervalCh <- interval:
	case <-w.exitCh:
	}
}

// pending returns the pending state and corresponding block. The returned
// values can be nil in case the pending block is not initialized.
func (w *worker) pending() (*types.Block, *state.StateDB) {
	w.snapshotMu.RLock()
	defer w.snapshotMu.RUnlock()
	if w.snapshotState == nil {
		return nil, nil
	}
	return w.snapshotBlock, w.snapshotState.Copy()
}

// pendingBlock returns pending block. The returned block can be nil in case the
// pending block is not initialized.
func (w *worker) pendingBlock() *types.Block {
	w.snapshotMu.RLock()
	defer w.snapshotMu.RUnlock()
	return w.snapshotBlock
}

// pendingBlockAndReceipts returns pending block and corresponding receipts.
// The returned values can be nil in case the pending block is not initialized.
func (w *worker) pendingBlockAndReceipts() (*types.Block, types.Receipts) {
	w.snapshotMu.RLock()
	defer w.snapshotMu.RUnlock()
	return w.snapshotBlock, w.snapshotReceipts
}

// start sets the running status as 1 and triggers new work submitting.
func (w *worker) start() {
	w.running.Store(true)
	w.startCh <- struct{}{}
}

// stop sets the running status as 0.
func (w *worker) stop() {
	w.running.Store(false)
}

// isRunning returns an indicator whether worker is running or not.
func (w *worker) isRunning() bool {
	return w.running.Load()
}

// close terminates all background threads maintained by the worker.
// Note the worker does not support being closed multiple times.
func (w *worker) close() {
	w.running.Store(false)
	close(w.exitCh)
	w.wg.Wait()
}

// recalcRecommit recalculates the resubmitting interval upon feedback.
func recalcRecommit(minRecommit, prev time.Duration, target float64, inc bool) time.Duration {
	var (
		prevF = float64(prev.Nanoseconds())
		next  float64
	)
	if inc {
		next = prevF*(1-intervalAdjustRatio) + intervalAdjustRatio*(target+intervalAdjustBias)
		max := float64(maxRecommitInterval.Nanoseconds())
		if next > max {
			next = max
		}
	} else {
		next = prevF*(1-intervalAdjustRatio) + intervalAdjustRatio*(target-intervalAdjustBias)
		min := float64(minRecommit.Nanoseconds())
		if next < min {
			next = min
		}
	}
	return time.Duration(int64(next))
}

// newWorkLoop is a standalone goroutine to submit new sealing work upon received events.
func (w *worker) newWorkLoop(recommit time.Duration) {
	defer w.wg.Done()
	var (
		interrupt   *atomic.Int32
		minRecommit = recommit // minimal resubmit interval specified by user.
		timestamp   int64      // timestamp for each round of sealing.
	)

	timer := time.NewTimer(0)
	defer timer.Stop()
	<-timer.C // discard the initial tick

	// commit aborts in-flight transaction execution with given signal and resubmits a new one.
	commit := func(s int32) {
		if interrupt != nil {
			interrupt.Store(s)
		}
		interrupt = new(atomic.Int32)
		select {
		case w.newWorkCh <- &newWorkReq{interrupt: interrupt, timestamp: timestamp}:
		case <-w.exitCh:
			return
		}
		timer.Reset(recommit)
		w.newTxs.Store(0)
		w.newL1Msgs.Store(0)
	}
	// clearPending cleans the stale pending tasks.
	clearPending := func(number uint64) {
		w.pendingMu.Lock()
		for h, t := range w.pendingTasks {
			if t.block.NumberU64()+staleThreshold <= number {
				delete(w.pendingTasks, h)
			}
		}
		w.pendingMu.Unlock()
	}

	for {
		select {
		case <-w.startCh:
			clearPending(w.chain.CurrentBlock().Number.Uint64())
			timestamp = time.Now().Unix()
			commit(commitInterruptNewHead)

		case head := <-w.chainHeadCh:
			clearPending(head.Block.NumberU64())
			timestamp = time.Now().Unix()
			commit(commitInterruptNewHead)

		case <-timer.C:
			// If sealing is running resubmit a new work cycle periodically to pull in
			// higher priced transactions. Disable this overhead for pending blocks.
			if w.isRunning() && (w.chainConfig.Clique == nil || w.chainConfig.Clique.Period > 0) {
				// Short circuit if no new transaction arrives.
				if w.newTxs.Load() == 0 && w.newL1Msgs.Load() == 0 {
					timer.Reset(recommit)
					continue
				}
				commit(commitInterruptResubmit)
			}

		case interval := <-w.resubmitIntervalCh:
			// Adjust resubmit interval explicitly by user.
			if interval < minRecommitInterval {
				log.Warn("Sanitizing miner recommit interval", "provided", interval, "updated", minRecommitInterval)
				interval = minRecommitInterval
			}
			log.Info("Miner recommit interval update", "from", minRecommit, "to", interval)
			minRecommit, recommit = interval, interval

			if w.resubmitHook != nil {
				w.resubmitHook(minRecommit, recommit)
			}

		case adjust := <-w.resubmitAdjustCh:
			// Adjust resubmit interval by feedback.
			if adjust.inc {
				before := recommit
				target := float64(recommit.Nanoseconds()) / adjust.ratio
				recommit = recalcRecommit(minRecommit, recommit, target, true)
				log.Trace("Increase miner recommit interval", "from", before, "to", recommit)
			} else {
				before := recommit
				recommit = recalcRecommit(minRecommit, recommit, float64(minRecommit.Nanoseconds()), false)
				log.Trace("Decrease miner recommit interval", "from", before, "to", recommit)
			}

			if w.resubmitHook != nil {
				w.resubmitHook(minRecommit, recommit)
			}

		case <-w.exitCh:
			return
		}
	}
}

// mainLoop is responsible for generating and submitting sealing work based on
// the received event. It can support two modes: automatically generate task and
// submit it or return task according to given parameters for various proposes.
func (w *worker) mainLoop() {
	defer w.wg.Done()
	defer w.txsSub.Unsubscribe()
	defer w.l1MsgsSub.Unsubscribe()
	defer w.chainHeadSub.Unsubscribe()
	defer func() {
		if w.current != nil {
			w.current.discard()
		}
	}()

	for {
		select {
		case req := <-w.newWorkCh:
			w.commitWork(req.interrupt, req.timestamp)
			// new block created.

		case req := <-w.getWorkCh:
			req.result <- w.generateWork(req.params)

		case ev := <-w.txsCh:
			// Apply transactions to the pending state if we're not sealing
			//
			// Note all transactions received may not be continuous with transactions
			// already included in the current sealing block. These transactions will
			// be automatically eliminated.
			if !w.isRunning() && w.current != nil {
				// If block is already full, abort
				if gp := w.current.gasPool; gp != nil && gp.Gas() < params.TxGas {
					continue
				}
				txs := make(map[common.Address][]*txpool.LazyTransaction, len(ev.Txs))
				for _, tx := range ev.Txs {
					acc, _ := types.Sender(w.current.signer, tx)
					txs[acc] = append(txs[acc], &txpool.LazyTransaction{
						Pool:      w.eth.TxPool(), // We don't know where this came from, yolo resolve from everywhere
						Hash:      tx.Hash(),
						Tx:        nil, // Do *not* set this! We need to resolve it later to pull blobs in
						Time:      tx.Time(),
						GasFeeCap: tx.GasFeeCap(),
						GasTipCap: tx.GasTipCap(),
						Gas:       tx.Gas(),
						BlobGas:   tx.BlobGas(),
					})
				}
				txset := newTransactionsByPriceAndNonce(w.current.signer, txs, w.current.header.BaseFee)
				tcount := w.current.tcount
				w.commitTransactions(w.current, txset, nil)

				// Only update the snapshot if any new transactions were added
				// to the pending block
				if tcount != w.current.tcount {
					w.updateSnapshot(w.current)
				}
			} else {
				// Special case, if the consensus engine is 0 period clique(dev mode),
				// submit sealing work here since all empty submission will be rejected
				// by clique. Of course the advance sealing(empty submission) is disabled.
				if w.chainConfig.Clique != nil && w.chainConfig.Clique.Period == 0 {
					w.commitWork(nil, time.Now().Unix())
				}
			}
			w.newTxs.Add(int32(len(ev.Txs)))

		case ev := <-w.l1MsgsCh:
			w.newL1Msgs.Add(int32(ev.Count))

		// System stopped
		case <-w.exitCh:
			return
		case <-w.txsSub.Err():
			return
		case <-w.l1MsgsSub.Err():
			return
		case <-w.chainHeadSub.Err():
			return
		}
	}
}

// taskLoop is a standalone goroutine to fetch sealing task from the generator and
// push them to consensus engine.
func (w *worker) taskLoop() {
	defer w.wg.Done()
	var (
		stopCh chan struct{}
		prev   common.Hash
	)

	// interrupt aborts the in-flight sealing task.
	interrupt := func() {
		if stopCh != nil {
			close(stopCh)
			stopCh = nil
		}
	}
	for {
		select {
		case task := <-w.taskCh:
			if w.newTaskHook != nil {
				w.newTaskHook(task)
			}
			// Reject duplicate sealing work due to resubmitting.
			sealHash := w.engine.SealHash(task.block.Header())
			if sealHash == prev {
				continue
			}
			// Interrupt previous sealing operation
			interrupt()
			stopCh, prev = make(chan struct{}), sealHash

			if w.skipSealHook != nil && w.skipSealHook(task) {
				continue
			}
			w.pendingMu.Lock()
			w.pendingTasks[sealHash] = task
			w.pendingMu.Unlock()

			if err := w.engine.Seal(w.chain, task.block, w.resultCh, stopCh); err != nil {
				log.Warn("Block sealing failed", "err", err)
				w.pendingMu.Lock()
				delete(w.pendingTasks, sealHash)
				w.pendingMu.Unlock()
			}
		case <-w.exitCh:
			interrupt()
			return
		}
	}
}

// resultLoop is a standalone goroutine to handle sealing result submitting
// and flush relative data to the database.
func (w *worker) resultLoop() {
	defer w.wg.Done()
	for {
		select {
		case block := <-w.resultCh:
			// Short circuit when receiving empty result.
			if block == nil {
				continue
			}
			// Short circuit when receiving duplicate result caused by resubmitting.
			if w.chain.HasBlock(block.Hash(), block.NumberU64()) {
				continue
			}
			var (
				sealhash = w.engine.SealHash(block.Header())
				hash     = block.Hash()
			)
			w.pendingMu.RLock()
			task, exist := w.pendingTasks[sealhash]
			w.pendingMu.RUnlock()
			if !exist {
				log.Error("Block found but no relative pending task", "number", block.Number(), "sealhash", sealhash, "hash", hash)
				continue
			}
			startTime := time.Now()
			// Different block could share same sealhash, deep copy here to prevent write-write conflict.
			var (
				receipts = make([]*types.Receipt, len(task.receipts))
				logs     []*types.Log
			)
			for i, taskReceipt := range task.receipts {
				receipt := new(types.Receipt)
				receipts[i] = receipt
				*receipt = *taskReceipt

				// add block location fields
				receipt.BlockHash = hash
				receipt.BlockNumber = block.Number()
				receipt.TransactionIndex = uint(i)

				// Update the block hash in all logs since it is now available and not when the
				// receipt/log of individual transactions were created.
				receipt.Logs = make([]*types.Log, len(taskReceipt.Logs))
				for i, taskLog := range taskReceipt.Logs {
					log := new(types.Log)
					receipt.Logs[i] = log
					*log = *taskLog
					log.BlockHash = hash
				}
				logs = append(logs, receipt.Logs...)
			}
			// It's possible that we've stored L1 queue index for this block previously,
			// in this case do not overwrite it.
			if index := rawdb.ReadFirstQueueIndexNotInL2Block(w.eth.ChainDb(), hash); index == nil {
				// Store first L1 queue index not processed by this block.
				// Note: This accounts for both included and skipped messages. This
				// way, if a block only skips messages, we won't reprocess the same
				// messages from the next block.
				log.Trace(
					"Worker WriteFirstQueueIndexNotInL2Block",
					"number", block.Number(),
					"hash", hash.String(),
					"task.nextL1MsgIndex", task.nextL1MsgIndex,
				)
				rawdb.WriteFirstQueueIndexNotInL2Block(w.eth.ChainDb(), hash, task.nextL1MsgIndex)
			} else {
				log.Trace(
					"Worker WriteFirstQueueIndexNotInL2Block: not overwriting existing index",
					"number", block.Number(),
					"hash", hash.String(),
					"index", *index,
					"task.nextL1MsgIndex", task.nextL1MsgIndex,
				)
			}
			// Store circuit row consumption.
			log.Trace(
				"Worker write block row consumption",
				"id", w.circuitCapacityChecker.ID,
				"number", block.Number(),
				"hash", hash.String(),
				"accRows", task.accRows,
			)
			rawdb.WriteBlockRowConsumption(w.eth.ChainDb(), hash, task.accRows)
			// Commit block and state to database.
			_, err := w.chain.WriteBlockAndSetHead(block, receipts, logs, task.state, true)
			if err != nil {
				l2ResultTimer.Update(time.Since(startTime))
				log.Error("Failed writing block to chain", "err", err)
				continue
			}
			log.Info("Successfully sealed new block", "number", block.Number(), "sealhash", sealhash, "hash", hash,
				"elapsed", common.PrettyDuration(time.Since(task.createdAt)))

			// Broadcast the block and announce chain insertion event
			w.mux.Post(core.NewMinedBlockEvent{Block: block})

			l2ResultTimer.Update(time.Since(startTime))

		case <-w.exitCh:
			return
		}
	}
}

// makeEnv creates a new environment for the sealing block.
func (w *worker) makeEnv(parent *types.Header, header *types.Header, coinbase common.Address) (*environment, error) {
	// Retrieve the parent state to execute on top and start a prefetcher for
	// the miner to speed block sealing up a bit.
	state, err := w.chain.StateAt(parent.Root)
	if err != nil {
		return nil, err
	}

	// don't finalize the state during tracing for circuit capacity checker, otherwise we cannot revert.
	// and even if we don't finalize the state, the `refund` value will still be correct, as explained in `CommitTransaction`
	finaliseStateAfterApply := false
	traceEnv, err := tracing.CreateTraceEnv(w.chainConfig, w.chain, w.engine, w.eth.ChainDb(), state, parent,
		// new block with a placeholder tx, for traceEnv's ExecutionResults length & TxStorageTraces length
		types.NewBlockWithHeader(header).WithBody([]*types.Transaction{types.NewTx(&types.LegacyTx{})}, nil),
		finaliseStateAfterApply)
	if err != nil {
		return nil, err
	}

	state.StartPrefetcher("miner")

	// Note the passed coinbase may be different with header.Coinbase.
	env := &environment{
		signer:   types.MakeSigner(w.chainConfig, header.Number, header.Time),
		state:    state,
		coinbase: coinbase,
		header:   header,
		traceEnv: traceEnv,
		accRows:  nil,
	}
	// Keep track of transactions which return errors so they can be removed
	env.tcount = 0
	env.blockSize = 0
	env.blockSize = 0
	env.l1TxCount = 0
	env.nextL1MsgIndex = traceEnv.StartL1QueueIndex
	return env, nil
}

// updateSnapshot updates pending snapshot block, receipts and state.
func (w *worker) updateSnapshot(env *environment) {
	w.snapshotMu.Lock()
	defer w.snapshotMu.Unlock()

	w.snapshotBlock = types.NewBlock(
		env.header,
		env.txs,
		nil,
		env.receipts,
		trie.NewStackTrie(nil),
	)
	w.snapshotReceipts = copyReceipts(env.receipts)
	w.snapshotState = env.state.Copy()
}

func (w *worker) commitTransaction(env *environment, tx *types.Transaction) ([]*types.Log, *types.BlockTrace, error) {
	if tx.Type() == types.BlobTxType {
		return w.commitBlobTransaction(env, tx)
	}
	receipt, traces, accRows, err := w.applyTransaction(env, tx)
	if err != nil {
		return nil, nil, err
	}
	env.txs = append(env.txs, tx)
	env.receipts = append(env.receipts, receipt)
	env.accRows = accRows
	return receipt.Logs, traces, nil
}

func (w *worker) commitBlobTransaction(env *environment, tx *types.Transaction) ([]*types.Log, *types.BlockTrace, error) {
	sc := tx.BlobTxSidecar()
	if sc == nil {
		panic("blob transaction without blobs in miner")
	}
	// Checking against blob gas limit: It's kind of ugly to perform this check here, but there
	// isn't really a better place right now. The blob gas limit is checked at block validation time
	// and not during execution. This means core.ApplyTransaction will not return an error if the
	// tx has too many blobs. So we have to explicitly check it here.
	if (env.blobs+len(sc.Blobs))*params.BlobTxBlobGasPerBlob > params.MaxBlobGasPerBlock {
		return nil, nil, errors.New("max data blobs reached")
	}
	receipt, traces, accRows, err := w.applyTransaction(env, tx)
	if err != nil {
		return nil, nil, err
	}
	env.txs = append(env.txs, tx.WithoutBlobTxSidecar())
	env.receipts = append(env.receipts, receipt)
	env.accRows = accRows
	env.sidecars = append(env.sidecars, sc)
	env.blobs += len(sc.Blobs)
	*env.header.BlobGasUsed += receipt.BlobGasUsed
	return receipt.Logs, traces, nil
}

// applyTransaction runs the transaction. If execution fails, state and gas pool are reverted.
func (w *worker) applyTransaction(env *environment, tx *types.Transaction) (*types.Receipt, *types.BlockTrace, *types.RowConsumption, error) {
	var (
		traces  *types.BlockTrace
		accRows *types.RowConsumption
		receipt *types.Receipt
		err     error
	)

	// do not do CCC checks on follower nodes
	if w.isRunning() {
		defer func(t0 time.Time) {
			l2CommitTxTimer.Update(time.Since(t0))
			if err != nil {
				l2CommitTxFailedTimer.Update(time.Since(t0))
			}
		}(time.Now())

		// do gas limit check up-front and do not run CCC if it fails
		if env.gasPool.Gas() < tx.Gas() {
			return nil, nil, nil, core.ErrGasLimitReached
		}

		snap := env.state.Snapshot()

		log.Trace(
			"Worker apply ccc for tx",
			"id", w.circuitCapacityChecker.ID,
			"txHash", tx.Hash().Hex(),
		)

		// 1. we have to check circuit capacity before `core.ApplyTransaction`,
		// because if the tx can be successfully executed but circuit capacity overflows, it will be inconvenient to revert.
		// 2. even if we don't commit to the state during the tracing (which means `clearJournalAndRefund` is not called during the tracing),
		// the `refund` value will still be correct, because:
		// 2.1 when starting handling the first tx, `state.refund` is 0 by default,
		// 2.2 after tracing, the state is either committed in `core.ApplyTransaction`, or reverted, so the `state.refund` can be cleared,
		// 2.3 when starting handling the following txs, `state.refund` comes as 0
		withTimer(l2CommitTxTraceTimer, func() {
			traces, err = env.traceEnv.GetBlockTrace(
				types.NewBlockWithHeader(env.header).WithBody([]*types.Transaction{tx}, nil),
			)
		})
		withTimer(l2CommitTxTraceStateRevertTimer, func() {
			// `env.traceEnv.State` & `env.state` share a same pointer to the state, so only need to revert `env.state`
			// revert to snapshot for calling `core.ApplyMessage` again, (both `traceEnv.GetBlockTrace` & `core.ApplyTransaction` will call `core.ApplyMessage`)
			env.state.RevertToSnapshot(snap)
		})
		if err != nil {
			return nil, nil, nil, err
		}
		withTimer(l2CommitTxCCCTimer, func() {
			accRows, err = w.circuitCapacityChecker.ApplyTransaction(traces)
		})
		if err != nil {
			return nil, traces, accRows, err
		}
		log.Trace(
			"Worker apply ccc for tx result",
			"id", w.circuitCapacityChecker.ID,
			"txHash", tx.Hash().Hex(),
			"accRows", accRows,
		)
	}

	var (
		snap = env.state.Snapshot() // create new snapshot for `core.ApplyTransaction`
		gp   = env.gasPool.Gas()
	)
	withTimer(l2CommitTxApplyTimer, func() {
		receipt, err = core.ApplyTransaction(w.chainConfig, w.chain, &env.coinbase, env.gasPool, env.state, env.header, tx, &env.header.GasUsed, *w.chain.GetVMConfig())
	})
	if err != nil {
		env.state.RevertToSnapshot(snap)
		env.gasPool.SetGas(gp)
		if accRows != nil {
			// At this point, we have called CCC but the transaction failed in `ApplyTransaction`.
			// If we skip this tx and continue to pack more, the next tx will likely fail with
			// `circuitcapacitychecker.ErrUnknown`. However, at this point we cannot decide whether
			// we should seal the block or skip the tx and continue, so we simply return the error.
			log.Error(
				"GetBlockTrace passed but ApplyTransaction failed, ccc is left in inconsistent state",
				"blockNumber", env.header.Number,
				"txHash", tx.Hash().Hex(),
				"err", err,
			)
		}
	}
	return receipt, traces, accRows, err
}

func (w *worker) commitTransactions(env *environment, txs orderedTransactionSet, interrupt *atomic.Int32) (bool, error) {
	defer func(t0 time.Time) {
		l2CommitTxsTimer.Update(time.Since(t0))
	}(time.Now())

	var circuitCapacityReached bool

	gasLimit := env.header.GasLimit
	if env.gasPool == nil {
		env.gasPool = new(core.GasPool).AddGas(gasLimit)
	}
	var coalescedLogs []*types.Log

	var loops int64
loop:
	for {
		if w.beforeTxHook != nil {
			w.beforeTxHook()
		}

		loops++

		// Check interruption signal and abort building if it's fired.
		if interrupt != nil {
			if signal := interrupt.Load(); signal != commitInterruptNone {
				return circuitCapacityReached, signalToErr(signal)
			}
		}
		// seal block early if we're over time
		// note: current.header.Time = max(parent.Time + cliquePeriod, now())
		if env.tcount > 0 && w.chainConfig.Clique != nil && uint64(time.Now().Unix()) > env.header.Time {
			circuitCapacityReached = true // skip subsequent invocations of commitTransactions
			break
		}
		// If we don't have enough gas for any further transactions then we're done.
		if env.gasPool.Gas() < params.TxGas {
			log.Trace("Not enough gas for further transactions", "have", env.gasPool, "want", params.TxGas)
			break
		}
		// Retrieve the next transaction and abort if all done.
		ltx := txs.Peek()
		if ltx == nil {
			break
		}
		// If we have collected enough transactions then we're done
		// Originally we only limit l2txs count, but now strictly limit total txs number.
		if !w.chainConfig.Scroll.IsValidTxCount(env.tcount + 1) {
			log.Trace("Transaction count limit reached", "have", env.tcount, "want", w.chainConfig.Scroll.MaxTxPerBlock)
			break
		}
		// If we don't have enough space for the next transaction, skip the account.
		if env.gasPool.Gas() < ltx.Gas {
			log.Trace("Not enough gas left for transaction", "hash", ltx.Hash, "left", env.gasPool.Gas(), "needed", ltx.Gas)
			txs.Pop()
			continue
		}
		if left := uint64(params.MaxBlobGasPerBlock - env.blobs*params.BlobTxBlobGasPerBlob); left < ltx.BlobGas {
			log.Trace("Not enough blob gas left for transaction", "hash", ltx.Hash, "left", left, "needed", ltx.BlobGas)
			txs.Pop()
			continue
		}
		// Transaction seems to fit, pull it up from the pool
		tx := ltx.Resolve()
		if tx == nil {
			log.Trace("Ignoring evicted transaction", "hash", ltx.Hash)
			txs.Pop()
			continue
		}
		if tx.IsL1MessageTx() && tx.AsL1MessageTx().QueueIndex != env.nextL1MsgIndex {
			log.Error(
				"Unexpected L1 message queue index in worker",
				"expected", env.nextL1MsgIndex,
				"got", tx.AsL1MessageTx().QueueIndex,
			)
			break
		}
		if !tx.IsL1MessageTx() && !w.chainConfig.Scroll.IsValidBlockSize(env.blockSize+tx.Size()) {
			log.Trace("Block size limit reached", "have", env.blockSize, "want", w.chainConfig.Scroll.MaxTxPayloadBytesPerBlock, "tx", tx.Size())
			txs.Pop() // skip transactions from this account
			continue
		}
		// Error may be ignored here. The error has already been checked
		// during transaction acceptance is the transaction pool.
		from, _ := types.Sender(env.signer, tx)

		// Check whether the tx is replay protected. If we're not in the EIP155 hf
		// phase, start ignoring the sender until we do.
		if tx.Protected() && !w.chainConfig.IsEIP155(env.header.Number) {
			log.Trace("Ignoring replay protected transaction", "hash", ltx.Hash, "eip155", w.chainConfig.EIP155Block)
			txs.Pop()
			continue
		}
		// Start executing the transaction
		env.state.SetTxContext(tx.Hash(), env.tcount)

		logs, traces, err := w.commitTransaction(env, tx)
		switch {
		case errors.Is(err, core.ErrNonceTooLow):
			// New head notification data race between the transaction pool and miner, shift
			log.Trace("Skipping transaction with low nonce", "hash", ltx.Hash, "sender", from, "nonce", tx.Nonce())
			txs.Shift()

		case errors.Is(err, nil):
			// Everything ok, collect the logs and shift in the next transaction from the same account
			coalescedLogs = append(coalescedLogs, logs...)
			env.tcount++
			txs.Shift()

			if tx.IsL1MessageTx() {
				queueIndex := tx.AsL1MessageTx().QueueIndex
				log.Debug("Including L1 message", "queueIndex", queueIndex, "tx", tx.Hash().String())
				env.l1TxCount++
				env.nextL1MsgIndex = queueIndex + 1
			} else {
				// only consider block size limit for L2 transactions
				env.blockSize += tx.Size()
			}

		case errors.Is(err, core.ErrGasLimitReached) && tx.IsL1MessageTx():
			// If this block already contains some L1 messages,
			// terminate here and try again in the next block.
			if env.l1TxCount > 0 {
				break loop
			}
			// A single L1 message leads to out-of-gas. Skip it.
			queueIndex := tx.AsL1MessageTx().QueueIndex
			log.Info("Skipping L1 message", "queueIndex", queueIndex, "tx", tx.Hash().String(), "block", env.header.Number, "reason", "gas limit exceeded")
			env.nextL1MsgIndex = queueIndex + 1
			txs.Shift()
			if w.config.StoreSkippedTxTraces {
				rawdb.WriteSkippedTransaction(w.eth.ChainDb(), tx, traces, "gas limit exceeded", env.header.Number.Uint64(), nil)
			} else {
				rawdb.WriteSkippedTransaction(w.eth.ChainDb(), tx, nil, "gas limit exceeded", env.header.Number.Uint64(), nil)
			}
			l1TxGasLimitExceededCounter.Inc(1)

		// Circuit capacity check
		case errors.Is(err, circuitcapacitychecker.ErrBlockRowConsumptionOverflow):
			if env.tcount >= 1 {
				// 1. Circuit capacity limit reached in a block, and it's not the first tx:
				// don't pop or shift, just quit the loop immediately;
				// though it might still be possible to add some "smaller" txs,
				// but it's a trade-off between tracing overhead & block usage rate
				log.Trace("Circuit capacity limit reached in a block", "acc_rows", env.accRows, "tx", tx.Hash().String())
				log.Info("Skipping message", "tx", tx.Hash().String(), "block", env.header.Number, "reason", "accumulated row consumption overflow")

				// Prioritize transaction for the next block.
				// If there are no new L1 messages, this transaction will be the 1st transaction in the next block,
				// at which point we can definitively decide if we should skip it or not.
				log.Debug("Prioritizing transaction for next block", "blockNumber", env.header.Number.Uint64()+1, "tx", tx.Hash().String())
				w.prioritizedTx = &prioritizedTransaction{
					blockNumber: env.header.Number.Uint64() + 1,
					tx:          tx,
				}
				w.newTxs.Add(int32(1))

				circuitCapacityReached = true
				break loop
			} else {
				// 2. Circuit capacity limit reached in a block, and it's the first tx: skip the tx
				log.Trace("Circuit capacity limit reached for a single tx", "tx", tx.Hash().String())

				if tx.IsL1MessageTx() {
					// Skip L1 message transaction,
					// shift to the next from the account because we shouldn't skip the entire txs from the same account
					txs.Shift()

					queueIndex := tx.AsL1MessageTx().QueueIndex
					log.Info("Skipping L1 message", "queueIndex", queueIndex, "tx", tx.Hash().String(), "block", env.header.Number, "reason", "first tx row consumption overflow")
					env.nextL1MsgIndex = queueIndex + 1
					l1TxRowConsumptionOverflowCounter.Inc(1)
				} else {
					// Skip L2 transaction and all other transactions from the same sender account
					log.Info("Skipping L2 message", "tx", tx.Hash().String(), "block", env.header.Number, "reason", "first tx row consumption overflow")
					txs.Pop()
					w.eth.TxPool().RemoveTx(tx.Hash(), true, true)
					l2TxRowConsumptionOverflowCounter.Inc(1)
				}

				// Reset ccc so that we can process other transactions for this block
				w.circuitCapacityChecker.Reset()
				log.Trace("Worker reset ccc", "id", w.circuitCapacityChecker.ID)
				circuitCapacityReached = false

				// Store skipped transaction in local db
				if w.config.StoreSkippedTxTraces {
					rawdb.WriteSkippedTransaction(w.eth.ChainDb(), tx, traces, "row consumption overflow", env.header.Number.Uint64(), nil)
				} else {
					rawdb.WriteSkippedTransaction(w.eth.ChainDb(), tx, nil, "row consumption overflow", env.header.Number.Uint64(), nil)
				}
			}

		case (errors.Is(err, circuitcapacitychecker.ErrUnknown) && tx.IsL1MessageTx()):
			// Circuit capacity check: unknown circuit capacity checker error for L1MessageTx,
			// shift to the next from the account because we shouldn't skip the entire txs from the same account
			queueIndex := tx.AsL1MessageTx().QueueIndex
			log.Trace("Unknown circuit capacity checker error for L1MessageTx", "tx", tx.Hash().String(), "queueIndex", queueIndex)
			log.Info("Skipping L1 message", "queueIndex", queueIndex, "tx", tx.Hash().String(), "block", env.header.Number, "reason", "unknown row consumption error")
			env.nextL1MsgIndex = queueIndex + 1
			// TODO: propagate more info about the error from CCC
			if w.config.StoreSkippedTxTraces {
				rawdb.WriteSkippedTransaction(w.eth.ChainDb(), tx, traces, "unknown circuit capacity checker error", env.header.Number.Uint64(), nil)
			} else {
				rawdb.WriteSkippedTransaction(w.eth.ChainDb(), tx, nil, "unknown circuit capacity checker error", env.header.Number.Uint64(), nil)
			}
			l1TxCccUnknownErrCounter.Inc(1)

			// Normally we would do `txs.Shift()` here.
			// However, after `ErrUnknown`, ccc might remain in an
			// inconsistent state, so we cannot pack more transactions.
			circuitCapacityReached = true
			w.checkCurrentTxNumWithCCC(env.tcount)
			break loop

		case (errors.Is(err, circuitcapacitychecker.ErrUnknown) && !tx.IsL1MessageTx()):
			// Circuit capacity check: unknown circuit capacity checker error for L2MessageTx, skip the account
			log.Trace("Unknown circuit capacity checker error for L2MessageTx", "tx", tx.Hash().String())
			log.Info("Skipping L2 message", "tx", tx.Hash().String(), "block", env.header.Number, "reason", "unknown row consumption error")
			// TODO: propagate more info about the error from CCC
			if w.config.StoreSkippedTxTraces {
				rawdb.WriteSkippedTransaction(w.eth.ChainDb(), tx, traces, "unknown circuit capacity checker error", env.header.Number.Uint64(), nil)
			} else {
				rawdb.WriteSkippedTransaction(w.eth.ChainDb(), tx, nil, "unknown circuit capacity checker error", env.header.Number.Uint64(), nil)
			}
			l2TxCccUnknownErrCounter.Inc(1)

			// Normally we would do `txs.Pop()` here.
			// However, after `ErrUnknown`, ccc might remain in an
			// inconsistent state, so we cannot pack more transactions.
			w.eth.TxPool().RemoveTx(tx.Hash(), true, true)
			circuitCapacityReached = true
			w.checkCurrentTxNumWithCCC(env.tcount)
			break loop

		case (errors.Is(err, core.ErrInsufficientFunds) || errors.Is(errors.Unwrap(err), core.ErrInsufficientFunds)):
			log.Trace("Skipping tx with insufficient funds", "sender", from, "tx", tx.Hash().String())
			txs.Pop()
			w.eth.TxPool().RemoveTx(tx.Hash(), true, true)

		default:
			// Transaction is regarded as invalid, drop all consecutive transactions from
			// the same sender because of `nonce-too-high` clause.
			log.Debug("Transaction failed, account skipped", "hash", ltx.Hash.String(), "err", err)
			if tx.IsL1MessageTx() {
				queueIndex := tx.AsL1MessageTx().QueueIndex
				log.Info("Skipping L1 message", "queueIndex", queueIndex, "tx", tx.Hash().String(), "block", env.header.Number, "reason", "strange error", "err", err)
				env.nextL1MsgIndex = queueIndex + 1
				if w.config.StoreSkippedTxTraces {
					rawdb.WriteSkippedTransaction(w.eth.ChainDb(), tx, traces, fmt.Sprintf("strange error: %v", err), env.header.Number.Uint64(), nil)
				} else {
					rawdb.WriteSkippedTransaction(w.eth.ChainDb(), tx, nil, fmt.Sprintf("strange error: %v", err), env.header.Number.Uint64(), nil)
				}
				l1TxStrangeErrCounter.Inc(1)
			}
			txs.Pop()
		}
	}
	if !w.isRunning() && len(coalescedLogs) > 0 {
		// We don't push the pendingLogsEvent while we are sealing. The reason is that
		// when we are sealing, the worker will regenerate a sealing block every 3 seconds.
		// In order to avoid pushing the repeated pendingLog, we disable the pending log pushing.

		// make a copy, the state caches the logs and these logs get "upgraded" from pending to mined
		// logs by filling in the block hash when the block was mined by the local miner. This can
		// cause a race condition if a log was "upgraded" before the PendingLogsEvent is processed.
		cpy := make([]*types.Log, len(coalescedLogs))
		for i, l := range coalescedLogs {
			cpy[i] = new(types.Log)
			*cpy[i] = *l
		}
		w.pendingLogsFeed.Send(cpy)
	}
	return circuitCapacityReached, nil
}

// generateParams wraps various of settings for generating sealing task.
type generateParams struct {
	timestamp   uint64            // The timstamp for sealing task
	forceTime   bool              // Flag whether the given timestamp is immutable or not
	parentHash  common.Hash       // Parent block hash, empty means the latest chain head
	coinbase    common.Address    // The fee recipient address for including transaction
	random      common.Hash       // The randomness generated by beacon chain, empty before the merge
	withdrawals types.Withdrawals // List of withdrawals to include in block.
	beaconRoot  *common.Hash      // The beacon root (cancun field).
	noTxs       bool              // Flag whether an empty block without any transaction is expected
}

// prepareWork constructs the sealing task according to the given parameters,
// either based on the last chain head or specified parent. In this function
// the pending transactions are not filled yet, only the empty task returned.
func (w *worker) prepareWork(genParams *generateParams) (*environment, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	// Find the parent block for sealing task
	parent := w.chain.CurrentBlock()
	if genParams.parentHash != (common.Hash{}) {
		block := w.chain.GetBlockByHash(genParams.parentHash)
		if block == nil {
			return nil, fmt.Errorf("missing parent")
		}
		parent = block.Header()
	}
	// Sanity check the timestamp correctness, recap the timestamp
	// to parent+1 if the mutation is allowed.
	timestamp := genParams.timestamp
	if parent.Time >= timestamp {
		if genParams.forceTime {
			return nil, fmt.Errorf("invalid timestamp, parent %d given %d", parent.Time, timestamp)
		}
		timestamp = parent.Time + 1
	}
	// Construct the sealing block header.
	header := &types.Header{
		ParentHash: parent.Hash(),
		Number:     new(big.Int).Add(parent.Number, common.Big1),
		GasLimit:   core.CalcGasLimit(parent.GasLimit, w.config.GasCeil),
		Time:       timestamp,
		Coinbase:   genParams.coinbase,
	}
	// Set the extra field.
	if len(w.extra) != 0 {
		header.Extra = w.extra
	}
	// Set the randomness field from the beacon chain if it's available.
	if genParams.random != (common.Hash{}) {
		header.MixDigest = genParams.random
	}
	// Set baseFee and GasLimit if we are on an EIP-1559 chain
	if w.chainConfig.IsLondon(header.Number) {
		header.BaseFee = eip1559.CalcBaseFee(w.chainConfig, parent)
		if !w.chainConfig.IsLondon(parent.Number) {
			parentGasLimit := parent.GasLimit * w.chainConfig.ElasticityMultiplier()
			header.GasLimit = core.CalcGasLimit(parentGasLimit, w.config.GasCeil)
		}
	}
	// Apply EIP-4844, EIP-4788.
	if w.chainConfig.IsCancun(header.Number, header.Time) {
		var excessBlobGas uint64
		if w.chainConfig.IsCancun(parent.Number, parent.Time) {
			excessBlobGas = eip4844.CalcExcessBlobGas(*parent.ExcessBlobGas, *parent.BlobGasUsed)
		} else {
			// For the first post-fork block, both parent.data_gas_used and parent.excess_data_gas are evaluated as 0
			excessBlobGas = eip4844.CalcExcessBlobGas(0, 0)
		}
		header.BlobGasUsed = new(uint64)
		header.ExcessBlobGas = &excessBlobGas
		header.ParentBeaconRoot = genParams.beaconRoot
	}
	// Run the consensus preparation with the default or customized consensus engine.
	if err := w.engine.Prepare(w.chain, header); err != nil {
		log.Error("Failed to prepare header for sealing", "err", err)
		return nil, err
	}
	// Could potentially happen if starting to mine in an odd state.
	// Note genParams.coinbase can be different with header.Coinbase
	// since clique algorithm can modify the coinbase field in header.
	env, err := w.makeEnv(parent, header, genParams.coinbase)
	if err != nil {
		log.Error("Failed to create sealing context", "err", err)
		return nil, err
	}
	if header.ParentBeaconRoot != nil {
		context := core.NewEVMBlockContext(header, w.chain, w.chainConfig, nil)
		vmenv := vm.NewEVM(context, vm.TxContext{}, env.state, w.chainConfig, vm.Config{})
		core.ProcessBeaconBlockRoot(*header.ParentBeaconRoot, vmenv, env.state)
	}
	return env, nil
}

func txToLazyTx(txPool *txpool.TxPool, tx *types.Transaction) *txpool.LazyTransaction {
	if tx.IsL1MessageTx() {
		return &txpool.LazyTransaction{
			Pool:      nil, // we should never resolve a L1MessageTx from the txpool and we never need to
			Hash:      tx.Hash(),
			Tx:        tx, // set the tx directly, we don't need to resolve it
			Time:      tx.Time(),
			GasFeeCap: tx.GasFeeCap(),
			GasTipCap: tx.GasTipCap(),
			Gas:       tx.Gas(),
			BlobGas:   tx.BlobGas(),
		}
	}

	return &txpool.LazyTransaction{
		Pool:      txPool,
		Hash:      tx.Hash(),
		Tx:        nil, // Do *not* set this! We need to resolve it later to pull blobs in
		Time:      tx.Time(),
		GasFeeCap: tx.GasFeeCap(),
		GasTipCap: tx.GasTipCap(),
		Gas:       tx.Gas(),
		BlobGas:   tx.BlobGas(),
	}
}

// fillTransactions retrieves the pending transactions from the txpool and fills them
// into the given sealing block. The transaction selection and ordering strategy can
// be customized with the plugin in the future.
func (w *worker) fillTransactions(interrupt *atomic.Int32, env *environment) error {
	// fetch l1Txs
	var l1Messages []types.L1MessageTx
	if w.chainConfig.Scroll.ShouldIncludeL1Messages() {
		withTimer(l2CommitNewWorkL1CollectTimer, func() {
			l1Messages = w.collectPendingL1Messages(env.nextL1MsgIndex)
		})
	}

	tidyPendingStart := time.Now()
	pending := w.eth.TxPool().Pending(true)

	// Split the pending transactions into locals and remotes.
	localTxs, remoteTxs := make(map[common.Address][]*txpool.LazyTransaction), pending
	for _, account := range w.eth.TxPool().Locals() {
		if txs := remoteTxs[account]; len(txs) > 0 {
			delete(remoteTxs, account)
			localTxs[account] = txs
		}
	}
	l2CommitNewWorkTidyPendingTxTimer.UpdateSince(tidyPendingStart)

	// Fill the block with all available pending transactions.
	var circuitCapacityReached bool
	var err error
	commitL1MsgStart := time.Now()
	if w.chainConfig.Scroll.ShouldIncludeL1Messages() && len(l1Messages) > 0 {
		log.Trace("Processing L1 messages for inclusion", "count", len(l1Messages))
		txs, err := newL1MessagesByQueueIndex(w.eth.TxPool(), l1Messages)
		if err != nil {
			log.Error("Failed to create L1 message set", "l1Messages", l1Messages, "err", err)
			return err
		}
		circuitCapacityReached, err = w.commitTransactions(env, txs, interrupt)
		if err != nil {
			l2CommitNewWorkCommitL1MsgTimer.UpdateSince(commitL1MsgStart)
			return err
		}
	}
	l2CommitNewWorkCommitL1MsgTimer.UpdateSince(commitL1MsgStart)
	prioritizedTxStart := time.Now()
	if w.prioritizedTx != nil && w.current.header.Number.Uint64() > w.prioritizedTx.blockNumber {
		w.prioritizedTx = nil
	}
	if !circuitCapacityReached && w.prioritizedTx != nil && w.current.header.Number.Uint64() == w.prioritizedTx.blockNumber {
		tx := w.prioritizedTx.tx
		from, _ := types.Sender(w.current.signer, tx) // error already checked before
		// we don't know where this came from, yolo resolve from everywhere (w.eth.TxPool())
		txList := map[common.Address][]*txpool.LazyTransaction{from: {txToLazyTx(w.eth.TxPool(), tx)}}
		// usually we should distinguish l1txs and l2txs:
		// use `newL1MessagesByQueueIndex` for l1txs and `newTransactionsByPriceAndNonce` is for l2txs;
		// but here there's only 1 tx, and hence no need for sorting, we could just simply use `newTransactionsByPriceAndNonce`
		// (but we fill the LazyTransaction's tx first, in case it's a l1tx and cannot be resolved from the mempool).
		txs := newTransactionsByPriceAndNonce(w.current.signer, txList, env.header.BaseFee)
		circuitCapacityReached, err = w.commitTransactions(env, txs, interrupt)
		if err != nil {
			l2CommitNewWorkPrioritizedTxCommitTimer.UpdateSince(prioritizedTxStart)
			return err
		}
	}
	l2CommitNewWorkPrioritizedTxCommitTimer.UpdateSince(prioritizedTxStart)
	remoteLocalStart := time.Now()
	if !circuitCapacityReached && len(localTxs) > 0 {
		localTxPriceAndNonceStart := time.Now()
		txs := newTransactionsByPriceAndNonce(env.signer, localTxs, env.header.BaseFee)
		l2CommitNewWorkLocalPriceAndNonceTimer.UpdateSince(localTxPriceAndNonceStart)
		if circuitCapacityReached, err = w.commitTransactions(env, txs, interrupt); err != nil {
			l2CommitNewWorkRemoteLocalCommitTimer.UpdateSince(remoteLocalStart)
			return err
		}
	}
	if !circuitCapacityReached && len(remoteTxs) > 0 {
		remoteTxPriceAndNonceStart := time.Now()
		txs := newTransactionsByPriceAndNonce(env.signer, remoteTxs, env.header.BaseFee)
		l2CommitNewWorkRemotePriceAndNonceTimer.UpdateSince(remoteTxPriceAndNonceStart)
		if _, err = w.commitTransactions(env, txs, interrupt); err != nil {
			l2CommitNewWorkRemoteLocalCommitTimer.UpdateSince(remoteLocalStart)
			return err
		}
	}
	l2CommitNewWorkRemoteLocalCommitTimer.UpdateSince(remoteLocalStart)
	return nil
}

// generateWork generates a sealing block based on the given parameters.
func (w *worker) generateWork(params *generateParams) *newPayloadResult {
	work, err := w.prepareWork(params)
	if err != nil {
		return &newPayloadResult{err: err}
	}
	defer work.discard()

	if !params.noTxs {
		interrupt := new(atomic.Int32)
		timer := time.AfterFunc(w.newpayloadTimeout, func() {
			interrupt.Store(commitInterruptTimeout)
		})
		defer timer.Stop()

		err := w.fillTransactions(interrupt, work)
		if errors.Is(err, errBlockInterruptedByTimeout) {
			log.Warn("Block building is interrupted", "allowance", common.PrettyDuration(w.newpayloadTimeout))
		}
	}
	block, err := w.engine.FinalizeAndAssemble(w.chain, work.header, work.state, work.txs, nil, work.receipts, params.withdrawals)
	if err != nil {
		return &newPayloadResult{err: err}
	}
	return &newPayloadResult{
		block:    block,
		fees:     totalFees(block, work.receipts),
		sidecars: work.sidecars,
	}
}

// commitWork generates several new sealing tasks based on the parent block
// and submit them to the sealer.
func (w *worker) commitWork(interrupt *atomic.Int32, timestamp int64) {
	// Abort committing if node is still syncing
	if w.syncing.Load() {
		return
	}

	defer func(t0 time.Time) {
		l2CommitNewWorkTimer.Update(time.Since(t0))
	}(time.Now())

	start := time.Now()
	w.circuitCapacityChecker.Reset()
	log.Trace("Worker reset ccc", "id", w.circuitCapacityChecker.ID)

	// Set the coinbase if the worker is running or it's required
	var coinbase common.Address
	if w.isRunning() {
		coinbase = w.etherbase()
		if coinbase == (common.Address{}) {
			log.Error("Refusing to mine without etherbase")
			return
		}
	}
	// TODO:
	// 1. l2CommitNewWorkPrepareTimer
	// 2. no need for l2CommitNewWorkCommitUncleTimer any more?
	work, err := w.prepareWork(&generateParams{
		timestamp: uint64(timestamp),
		coinbase:  coinbase,
	})
	if err != nil {
		return
	}
	// Fill pending transactions from the txpool into the block.
	err = w.fillTransactions(interrupt, work)
	switch {
	case err == nil:
		// The entire block is filled, decrease resubmit interval in case
		// of current interval is larger than the user-specified one.
		w.resubmitAdjustCh <- &intervalAdjust{inc: false}

	case errors.Is(err, errBlockInterruptedByRecommit):
		// Notify resubmit loop to increase resubmitting interval if the
		// interruption is due to frequent commits.
		gaslimit := work.header.GasLimit
		ratio := float64(gaslimit-work.gasPool.Gas()) / float64(gaslimit)
		if ratio < 0.1 {
			ratio = 0.1
		}
		w.resubmitAdjustCh <- &intervalAdjust{
			ratio: ratio,
			inc:   true,
		}

	case errors.Is(err, errBlockInterruptedByNewHead):
		// If the block building is interrupted by newhead event, discard it
		// totally. Committing the interrupted block introduces unnecessary
		// delay, and possibly causes miner to mine on the previous head,
		// which could result in higher uncle rate.
		work.discard()
		return
	}
	// Submit the generated block for consensus sealing.
	w.commit(work.copy(), w.fullTaskHook, true, start)

	// Swap out the old work with the new one, terminating any leftover
	// prefetcher processes in the mean time and starting a new one.
	if w.current != nil {
		w.current.discard()
	}
	w.current = work
}

func (w *worker) calcAndSetAccRowsForEnv(env *environment) error {
	log.Trace(
		"Worker apply ccc for empty block",
		"id", w.circuitCapacityChecker.ID,
		"number", env.header.Number,
		"hash", env.header.Hash().String(),
	)
	var traces *types.BlockTrace
	var err error
	withTimer(l2CommitTraceTimer, func() {
		traces, err = env.traceEnv.GetBlockTrace(types.NewBlockWithHeader(env.header))
	})
	if err != nil {
		return err
	}
	if traces == nil {
		log.Warn("running in light mode and traces is nil, don't update `env.accRows`")
		return nil
	}

	// truncate ExecutionResults&TxStorageTraces, because we declare their lengths with a dummy tx before;
	// however, we need to clean it up for an empty block
	traces.ExecutionResults = traces.ExecutionResults[:0]
	traces.TxStorageTraces = traces.TxStorageTraces[:0]
	var accRows *types.RowConsumption
	withTimer(l2CommitCCCTimer, func() {
		accRows, err = w.circuitCapacityChecker.ApplyBlock(traces)
	})
	if err != nil {
		return err
	}
	log.Trace(
		"Worker apply ccc for empty block result",
		"id", w.circuitCapacityChecker.ID,
		"number", env.header.Number,
		"hash", env.header.Hash().String(),
		"accRows", accRows,
	)
	env.accRows = accRows
	return nil
}

// commit runs any post-transaction state modifications, assembles the final block
// and commits new work if consensus engine is running.
// Note the assumption is held that the mutation is allowed to the passed env, do
// the deep copy first.
func (w *worker) commit(env *environment, interval func(), update bool, start time.Time) error {
	defer func(t0 time.Time) {
		l2CommitTimer.Update(time.Since(t0))
	}(time.Now())

	if w.isRunning() {
		if interval != nil {
			interval()
		}
		// Create a local environment copy, avoid the data race with snapshot state.
		// https://github.com/ethereum/go-ethereum/issues/24299
		env := env.copy()
		// set env.accRows for empty-but-not-genesis block
		if (env.header.Number.Uint64() != 0) && (env.accRows == nil || len(*env.accRows) == 0) {
			if err := w.calcAndSetAccRowsForEnv(env); err != nil {
				return err
			}
		}
		// Withdrawals are set to nil here, because this is only called in PoW.
		block, err := w.engine.FinalizeAndAssemble(w.chain, env.header, env.state, env.txs, nil, env.receipts, nil)
		if err != nil {
			return err
		}
		// If we're post merge, just ignore
		if !w.isTTDReached(block.Header()) {
			select {
			case w.taskCh <- &task{receipts: env.receipts, state: env.state, block: block, createdAt: time.Now(), accRows: env.accRows, nextL1MsgIndex: env.nextL1MsgIndex}:
				fees := totalFees(block, env.receipts)
				feesInEther := new(big.Float).Quo(new(big.Float).SetInt(fees), big.NewFloat(params.Ether))
				log.Info("Commit new sealing work", "number", block.Number(), "sealhash", w.engine.SealHash(block.Header()),
					"txs", env.tcount, "gas", block.GasUsed(), "fees", feesInEther,
					"elapsed", common.PrettyDuration(time.Since(start)))

			case <-w.exitCh:
				log.Info("Worker has exited")
			}
		}
	}
	if update {
		w.updateSnapshot(env)
	}
	return nil
}

// getSealingBlock generates the sealing block based on the given parameters.
// The generation result will be passed back via the given channel no matter
// the generation itself succeeds or not.
func (w *worker) getSealingBlock(params *generateParams) *newPayloadResult {
	req := &getWorkReq{
		params: params,
		result: make(chan *newPayloadResult, 1),
	}
	select {
	case w.getWorkCh <- req:
		return <-req.result
	case <-w.exitCh:
		return &newPayloadResult{err: errors.New("miner closed")}
	}
}

// isTTDReached returns the indicator if the given block has reached the total
// terminal difficulty for The Merge transition.
func (w *worker) isTTDReached(header *types.Header) bool {
	td, ttd := w.chain.GetTd(header.ParentHash, header.Number.Uint64()-1), w.chain.Config().TerminalTotalDifficulty
	return td != nil && ttd != nil && td.Cmp(ttd) >= 0
}

func (w *worker) checkCurrentTxNumWithCCC(expected int) {
	match, got, err := w.circuitCapacityChecker.CheckTxNum(expected)
	if err != nil {
		log.Error("failed to CheckTxNum in ccc", "err", err)
		return
	}
	if !match {
		log.Error("tx count in miner is different with CCC", "w.current.tcount", w.current.tcount, "got", got)
	}
}

func (w *worker) collectPendingL1Messages(startIndex uint64) []types.L1MessageTx {
	maxCount := w.chainConfig.Scroll.L1Config.NumL1MessagesPerBlock
	return rawdb.ReadL1MessagesFrom(w.eth.ChainDb(), startIndex, maxCount)
}

// copyReceipts makes a deep copy of the given receipts.
func copyReceipts(receipts []*types.Receipt) []*types.Receipt {
	result := make([]*types.Receipt, len(receipts))
	for i, l := range receipts {
		cpy := *l
		result[i] = &cpy
	}
	return result
}

// totalFees computes total consumed miner fees in Wei. Block transactions and receipts have to have the same order.
func totalFees(block *types.Block, receipts []*types.Receipt) *big.Int {
	feesWei := new(big.Int)
	for i, tx := range block.Transactions() {
		minerFee, _ := tx.EffectiveGasTip(block.BaseFee())
		feesWei.Add(feesWei, new(big.Int).Mul(new(big.Int).SetUint64(receipts[i].GasUsed), minerFee))
	}
	return feesWei
}

// signalToErr converts the interruption signal to a concrete error type for return.
// The given signal must be a valid interruption signal.
func signalToErr(signal int32) error {
	switch signal {
	case commitInterruptNewHead:
		return errBlockInterruptedByNewHead
	case commitInterruptResubmit:
		return errBlockInterruptedByRecommit
	case commitInterruptTimeout:
		return errBlockInterruptedByTimeout
	default:
		panic(fmt.Errorf("undefined signal %d", signal))
	}
}

func withTimer(timer metrics.Timer, f func()) {
	if metrics.Enabled {
		timer.Time(f)
	} else {
		f()
	}
}
