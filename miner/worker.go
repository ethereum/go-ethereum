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
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"os"
	"runtime"
	"runtime/pprof"
	ptrace "runtime/trace"
	"sync"
	"sync/atomic"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/holiman/uint256"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/ethereum/go-ethereum/common"
	cmath "github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/common/tracing"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/bor"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/blockstm"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
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

	// metrics gauge to track total and empty blocks sealed by a miner
	sealedBlocksCounter      = metrics.NewRegisteredCounter("worker/sealedBlocks", nil)
	sealedEmptyBlocksCounter = metrics.NewRegisteredCounter("worker/sealedEmptyBlocks", nil)
	txCommitInterruptCounter = metrics.NewRegisteredCounter("worker/txCommitInterrupt", nil)
)

// environment is the worker's current environment and holds all
// information of the sealing block generation.
type environment struct {
	signer   types.Signer
	state    *state.StateDB // apply state changes here
	tcount   int            // tx count in cycle
	gasPool  *core.GasPool  // available gas used to pack transactions
	coinbase common.Address

	header   *types.Header
	txs      []*types.Transaction
	receipts []*types.Receipt
	sidecars []*types.BlobTxSidecar
	blobs    int

	depsMVFullWriteList [][]blockstm.WriteDescriptor
	mvReadMapList       []map[blockstm.Key]blockstm.ReadDescriptor
}

// copy creates a deep copy of environment.
func (env *environment) copy() *environment {
	cpy := &environment{
		signer:              env.signer,
		state:               env.state.Copy(),
		tcount:              env.tcount,
		coinbase:            env.coinbase,
		header:              types.CopyHeader(env.header),
		receipts:            copyReceipts(env.receipts),
		depsMVFullWriteList: env.depsMVFullWriteList,
		mvReadMapList:       env.mvReadMapList,
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
	//nolint:containedctx
	ctx       context.Context
	receipts  []*types.Receipt
	state     *state.StateDB
	block     *types.Block
	createdAt time.Time
}

const (
	commitInterruptNone int32 = iota
	commitInterruptNewHead
	commitInterruptResubmit
	commitInterruptTimeout
)

// newWorkReq represents a request for new sealing work submitting with relative interrupt notifier.
type newWorkReq struct {
	//nolint:containedctx
	ctx       context.Context
	interrupt *atomic.Int32
	noempty   bool
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
	//nolint:containedctx
	ctx    context.Context
	params *generateParams
	result chan *newPayloadResult // non-blocking channel
}

// intervalAdjust represents a resubmitting interval adjustment.
type intervalAdjust struct {
	ratio float64
	inc   bool
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
	tip      *big.Int // Minimum tip needed for non-local transaction to include them

	pendingMu    sync.RWMutex
	pendingTasks map[common.Hash]*task

	snapshotMu       sync.RWMutex // The lock used to protect the snapshots below
	snapshotBlock    *types.Block
	snapshotReceipts types.Receipts
	snapshotState    *state.StateDB

	// atomic status counters
	running atomic.Bool  // The indicator whether the consensus engine is running or not.
	newTxs  atomic.Int32 // New arrival transaction count since last sealing work submitting.
	syncing atomic.Bool  // The indicator whether the node is still syncing.

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

	// Test hooks
	newTaskHook  func(*task)                        // Method to call upon receiving a new sealing task.
	skipSealHook func(*task) bool                   // Method to decide whether skipping the sealing.
	fullTaskHook func()                             // Method to call before pushing the full sealing task.
	resubmitHook func(time.Duration, time.Duration) // Method to call upon updating resubmitting interval.

	profileCount        *int32 // Global count for profiling
	interruptCommitFlag bool   // Interrupt commit ( Default true )
	interruptedTxCache  *vm.TxCache

	// noempty is the flag used to control whether the feature of pre-seal empty
	// block is enabled. The default value is false(pre-seal is enabled by default).
	// But in some special scenario the consensus engine will seal blocks instantaneously,
	// in this case this feature will add all empty blocks into canonical chain
	// non-stop and no real transaction will be included.
	noempty atomic.Bool
}

//nolint:staticcheck
func newWorker(config *Config, chainConfig *params.ChainConfig, engine consensus.Engine, eth Backend, mux *event.TypeMux, isLocalBlock func(header *types.Header) bool, init bool) *worker {
	worker := &worker{
		config:              config,
		chainConfig:         chainConfig,
		engine:              engine,
		eth:                 eth,
		chain:               eth.BlockChain(),
		mux:                 mux,
		isLocalBlock:        isLocalBlock,
		coinbase:            config.Etherbase,
		extra:               config.ExtraData,
		tip:                 config.GasPrice,
		pendingTasks:        make(map[common.Hash]*task),
		txsCh:               make(chan core.NewTxsEvent, txChanSize),
		chainHeadCh:         make(chan core.ChainHeadEvent, chainHeadChanSize),
		newWorkCh:           make(chan *newWorkReq),
		getWorkCh:           make(chan *getWorkReq),
		taskCh:              make(chan *task),
		resultCh:            make(chan *types.Block, resultQueueSize),
		startCh:             make(chan struct{}, 1),
		exitCh:              make(chan struct{}),
		resubmitIntervalCh:  make(chan time.Duration),
		resubmitAdjustCh:    make(chan *intervalAdjust, resubmitAdjustChanSize),
		interruptCommitFlag: config.CommitInterruptFlag,
	}
	worker.noempty.Store(true)
	worker.profileCount = new(int32)
	// Subscribe for transaction insertion events (whether from network or resurrects)
	worker.txsSub = eth.TxPool().SubscribeTransactions(worker.txsCh, true)
	// Subscribe events for blockchain
	worker.chainHeadSub = eth.BlockChain().SubscribeChainHeadEvent(worker.chainHeadCh)

	interruptedTxCache, err := lru.New(vm.InterruptedTxCacheSize)
	if err != nil {
		log.Warn("Failed to create interrupted tx cache", "err", err)
	}

	worker.interruptedTxCache = &vm.TxCache{
		Cache: interruptedTxCache,
	}

	if !worker.interruptCommitFlag {
		worker.noempty.Store(false)
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

	ctx := tracing.WithTracer(context.Background(), otel.GetTracerProvider().Tracer("MinerWorker"))

	worker.wg.Add(4)

	go worker.mainLoop(ctx)
	go worker.newWorkLoop(ctx, recommit)
	go worker.resultLoop()
	go worker.taskLoop()

	// Submit first work to initialize pending state.
	if init {
		worker.startCh <- struct{}{}
	}

	return worker
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

// setGasTip sets the minimum miner tip needed to include a non-local transaction.
func (w *worker) setGasTip(tip *big.Int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.tip = tip
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
func (w *worker) IsRunning() bool {
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
	//var (
	//	prevF = float64(prev.Nanoseconds())
	//	next  float64
	//)
	//if inc {
	//	next = prevF*(1-intervalAdjustRatio) + intervalAdjustRatio*(target+intervalAdjustBias)
	//	max := float64(maxRecommitInterval.Nanoseconds())
	//	if next > max {
	//		next = max
	//	}
	//} else {
	//	next = prevF*(1-intervalAdjustRatio) + intervalAdjustRatio*(target-intervalAdjustBias)
	//	min := float64(minRecommit.Nanoseconds())
	//	if next < min {
	//		next = min
	//	}
	//}
	return prev
}

// newWorkLoop is a standalone goroutine to submit new sealing work upon received events.
//
//nolint:gocognit
func (w *worker) newWorkLoop(ctx context.Context, recommit time.Duration) {
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
	commit := func(noempty bool, s int32) {
		ctx, span := tracing.Trace(ctx, "worker.newWorkLoop.commit")
		tracing.EndSpan(span)
		if interrupt != nil {
			interrupt.Store(s)
		}

		interrupt = new(atomic.Int32)
		select {
		case w.newWorkCh <- &newWorkReq{interrupt: interrupt, timestamp: timestamp, ctx: ctx, noempty: noempty}:
		case <-w.exitCh:
			return
		}
		timer.Reset(recommit)
		w.newTxs.Store(0)
	}
	// clearPending cleans the stale pending tasks.
	clearPending := func(number uint64) {
		_, span := tracing.Trace(ctx, "worker.newWorkLoop.clearPending")
		tracing.EndSpan(span)

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
			commit(false, commitInterruptNewHead)

		case head := <-w.chainHeadCh:
			clearPending(head.Block.NumberU64())

			timestamp = time.Now().Unix()
			commit(false, commitInterruptNewHead)

		case <-timer.C:
			// If sealing is running resubmit a new work cycle periodically to pull in
			// higher priced transactions. Disable this overhead for pending blocks.
			if w.IsRunning() && (w.chainConfig.Clique == nil || w.chainConfig.Clique.Period > 0) {
				// Short circuit if no new transaction arrives.
				if w.newTxs.Load() == 0 {
					timer.Reset(recommit)
					continue
				}
				commit(true, commitInterruptResubmit)
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
// nolint: gocognit, contextcheck
func (w *worker) mainLoop(ctx context.Context) {
	defer w.wg.Done()
	defer w.txsSub.Unsubscribe()
	defer w.chainHeadSub.Unsubscribe()
	defer func() {
		if w.current != nil {
			w.current.discard()
		}
	}()

	for {
		select {
		case req := <-w.newWorkCh:
			if w.chainConfig.ChainID.Cmp(params.BorMainnetChainConfig.ChainID) == 0 || w.chainConfig.ChainID.Cmp(params.MumbaiChainConfig.ChainID) == 0 || w.chainConfig.ChainID.Cmp(params.AmoyChainConfig.ChainID) == 0 {
				if w.eth.PeerCount() > 0 {
					//nolint:contextcheck
					w.commitWork(req.ctx, req.interrupt, req.noempty, req.timestamp)
				}
			} else {
				//nolint:contextcheck
				w.commitWork(req.ctx, req.interrupt, req.noempty, req.timestamp)
			}

		case req := <-w.getWorkCh:
			req.result <- w.generateWork(ctx, req.params)

		case ev := <-w.txsCh:
			// Apply transactions to the pending state if we're not sealing
			//
			// Note all transactions received may not be continuous with transactions
			// already included in the current sealing block. These transactions will
			// be automatically eliminated.
			// nolint : nestif
			if !w.IsRunning() && w.current != nil {
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
				w.commitTransactions(w.current, txset, nil, new(big.Int), context.Background())

				// Only update the snapshot if any new transactons were added
				// to the pending block
				if tcount != w.current.tcount {
					w.updateSnapshot(w.current)
				}
			} else {
				// Special case, if the consensus engine is 0 period clique(dev mode),
				// submit sealing work here since all empty submission will be rejected
				// by clique. Of course the advance sealing(empty submission) is disabled.
				if w.chainConfig.Clique != nil && w.chainConfig.Clique.Period == 0 {
					w.commitWork(ctx, nil, true, time.Now().Unix())
				}
			}

			w.newTxs.Add(int32(len(ev.Txs)))

		// System stopped
		case <-w.exitCh:
			return
		case <-w.txsSub.Err():
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

			if err := w.engine.Seal(task.ctx, w.chain, task.block, w.resultCh, stopCh); err != nil {
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

			oldBlock := w.chain.GetBlockByNumber(block.NumberU64())
			if oldBlock != nil {
				oldBlockAuthor, _ := w.chain.Engine().Author(oldBlock.Header())
				newBlockAuthor, _ := w.chain.Engine().Author(block.Header())

				if oldBlockAuthor == newBlockAuthor {
					log.Info("same block ", "height", block.NumberU64())
					continue
				}
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
			// Different block could share same sealhash, deep copy here to prevent write-write conflict.
			var (
				receipts = make([]*types.Receipt, len(task.receipts))
				logs     []*types.Log
				err      error
			)

			tracing.Exec(task.ctx, "", "resultLoop", func(ctx context.Context, span trace.Span) {
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
				// Commit block and state to database.
				tracing.Exec(ctx, "", "resultLoop.WriteBlockAndSetHead", func(ctx context.Context, span trace.Span) {
					_, err = w.chain.WriteBlockAndSetHead(ctx, block, receipts, logs, task.state, true)
				})

				tracing.SetAttributes(
					span,
					attribute.String("hash", hash.String()),
					attribute.Int("number", int(block.Number().Uint64())),
					attribute.Int("txns", block.Transactions().Len()),
					attribute.Int("gas used", int(block.GasUsed())),
					attribute.Int("elapsed", int(time.Since(task.createdAt).Milliseconds())),
					attribute.Bool("error", err != nil),
				)
			})

			if err != nil {
				log.Error("Failed writing block to chain", "err", err)
				continue
			}

			log.Info("Successfully sealed new block", "number", block.Number(), "sealhash", sealhash, "hash", hash,
				"elapsed", common.PrettyDuration(time.Since(task.createdAt)))

			// Broadcast the block and announce chain insertion event
			w.mux.Post(core.NewMinedBlockEvent{Block: block})

			sealedBlocksCounter.Inc(1)

			if block.Transactions().Len() == 0 {
				sealedEmptyBlocksCounter.Inc(1)
			}

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

	state.StartPrefetcher("miner")

	// Note the passed coinbase may be different with header.Coinbase.
	env := &environment{
		signer:   types.MakeSigner(w.chainConfig, header.Number, header.Time),
		state:    state,
		coinbase: coinbase,
		header:   header,
	}
	// Keep track of transactions which return errors so they can be removed
	env.tcount = 0

	env.depsMVFullWriteList = [][]blockstm.WriteDescriptor{}
	env.mvReadMapList = []map[blockstm.Key]blockstm.ReadDescriptor{}

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

func (w *worker) commitTransaction(env *environment, tx *types.Transaction, interruptCtx context.Context) ([]*types.Log, error) {
	var (
		snap = env.state.Snapshot()
		gp   = env.gasPool.Gas()
	)

	// nolint : staticcheck
	interruptCtx = vm.SetCurrentTxOnContext(interruptCtx, tx.Hash())

	receipt, err := core.ApplyTransaction(w.chainConfig, w.chain, &env.coinbase, env.gasPool, env.state, env.header, tx, &env.header.GasUsed, *w.chain.GetVMConfig(), interruptCtx)
	if err != nil {
		env.state.RevertToSnapshot(snap)
		env.gasPool.SetGas(gp)

		return nil, err
	}
	env.txs = append(env.txs, tx)
	env.receipts = append(env.receipts, receipt)

	return receipt.Logs, nil
}

func (w *worker) commitTransactions(env *environment, txs *transactionsByPriceAndNonce, interrupt *atomic.Int32, minTip *big.Int, interruptCtx context.Context) error {
	gasLimit := env.header.GasLimit
	if env.gasPool == nil {
		env.gasPool = new(core.GasPool).AddGas(gasLimit)
	}

	var coalescedLogs []*types.Log

	var deps map[int]map[int]bool

	chDeps := make(chan blockstm.TxDep)

	var depsWg sync.WaitGroup
	var once sync.Once

	EnableMVHashMap := w.chainConfig.IsCancun(env.header.Number)

	// create and add empty mvHashMap in statedb
	if EnableMVHashMap && w.IsRunning() {
		deps = map[int]map[int]bool{}

		chDeps = make(chan blockstm.TxDep)

		// Make sure we safely close the channel in case of interrupt
		defer once.Do(func() {
			close(chDeps)
		})

		depsWg.Add(1)

		go func(chDeps chan blockstm.TxDep) {
			for t := range chDeps {
				deps = blockstm.UpdateDeps(deps, t)
			}

			depsWg.Done()
		}(chDeps)
	}

	var lastTxHash common.Hash

mainloop:
	for {
		// Check interruption signal and abort building if it's fired.
		if interrupt != nil {
			if signal := interrupt.Load(); signal != commitInterruptNone {
				return signalToErr(signal)
			}
		}

		if interruptCtx != nil {
			if EnableMVHashMap && w.IsRunning() {
				env.state.AddEmptyMVHashMap()
			}

			// case of interrupting by timeout
			select {
			case <-interruptCtx.Done():
				txCommitInterruptCounter.Inc(1)
				log.Warn("Tx Level Interrupt", "hash", lastTxHash)
				break mainloop
			default:
			}
		}

		// If we don't have enough gas for any further transactions then we're done.
		if env.gasPool.Gas() < params.TxGas {
			log.Trace("Not enough gas for further transactions", "have", env.gasPool, "want", params.TxGas)
			break
		}
		// Retrieve the next transaction and abort if all done.
		ltx, tip := txs.Peek()
		if ltx == nil {
			break
		}
		lastTxHash = ltx.Hash
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
		// If we don't receive enough tip for the next transaction, skip the account
		if tip.Cmp(minTip) < 0 {
			log.Trace("Not enough tip for transaction", "hash", ltx.Hash, "tip", tip, "needed", minTip)
			break // If the next-best is too low, surely no better will be available
		}
		// Transaction seems to fit, pull it up from the pool
		tx := ltx.Resolve()
		if tx == nil {
			log.Trace("Ignoring evicted transaction", "hash", ltx.Hash)
			txs.Pop()
			continue
		}
		// Error may be ignored here. The error has already been checked
		// during transaction acceptance is the transaction pool.
		from, _ := types.Sender(env.signer, tx)

		// not prioritising conditional transaction, yet.
		//nolint:nestif
		if options := tx.GetOptions(); options != nil {
			if err := env.header.ValidateBlockNumberOptionsPIP15(options.BlockNumberMin, options.BlockNumberMax); err != nil {
				log.Trace("Dropping conditional transaction", "from", from, "hash", tx.Hash(), "reason", err)
				txs.Pop()

				continue
			}

			if err := env.header.ValidateTimestampOptionsPIP15(options.TimestampMin, options.TimestampMax); err != nil {
				log.Trace("Dropping conditional transaction", "from", from, "hash", tx.Hash(), "reason", err)
				txs.Pop()

				continue
			}

			if err := env.state.ValidateKnownAccounts(options.KnownAccounts); err != nil {
				log.Trace("Dropping conditional transaction", "from", from, "hash", tx.Hash(), "reason", err)
				txs.Pop()

				continue
			}
		}

		// Check whether the tx is replay protected. If we're not in the EIP155 hf
		// phase, start ignoring the sender until we do.
		if tx.Protected() && !w.chainConfig.IsEIP155(env.header.Number) {
			log.Trace("Ignoring replay protected transaction", "hash", ltx.Hash, "eip155", w.chainConfig.EIP155Block)
			txs.Pop()
			continue
		}
		// Start executing the transaction
		env.state.SetTxContext(tx.Hash(), env.tcount)

		logs, err := w.commitTransaction(env, tx, interruptCtx)

		switch {
		case errors.Is(err, core.ErrNonceTooLow):
			// New head notification data race between the transaction pool and miner, shift
			log.Trace("Skipping transaction with low nonce", "hash", ltx.Hash, "sender", from, "nonce", tx.Nonce())
			txs.Shift()

		case errors.Is(err, nil):
			// Everything ok, collect the logs and shift in the next transaction from the same account
			coalescedLogs = append(coalescedLogs, logs...)
			env.tcount++

			if EnableMVHashMap && w.IsRunning() {
				env.depsMVFullWriteList = append(env.depsMVFullWriteList, env.state.MVFullWriteList())
				env.mvReadMapList = append(env.mvReadMapList, env.state.MVReadMap())

				if env.tcount > len(env.depsMVFullWriteList) {
					log.Warn("blockstm - env.tcount > len(env.depsMVFullWriteList)", "env.tcount", env.tcount, "len(depsMVFullWriteList)", len(env.depsMVFullWriteList))
				}

				temp := blockstm.TxDep{
					Index:         env.tcount - 1,
					ReadList:      env.state.MVReadList(),
					FullWriteList: env.depsMVFullWriteList,
				}

				chDeps <- temp
			}

			txs.Shift()

		default:
			// Transaction is regarded as invalid, drop all consecutive transactions from
			// the same sender because of `nonce-too-high` clause.
			log.Debug("Transaction failed, account skipped", "hash", ltx.Hash, "err", err)
			txs.Pop()
		}

		if EnableMVHashMap && w.IsRunning() {
			env.state.ClearReadMap()
			env.state.ClearWriteMap()
		}
	}

	// nolint:nestif
	if EnableMVHashMap && w.IsRunning() {
		once.Do(func() {
			close(chDeps)
		})
		depsWg.Wait()

		var blockExtraData types.BlockExtraData

		tempVanity := env.header.Extra[:types.ExtraVanityLength]
		tempSeal := env.header.Extra[len(env.header.Extra)-types.ExtraSealLength:]

		if len(env.mvReadMapList) > 0 {
			tempDeps := make([][]uint64, len(env.mvReadMapList))

			for j := range deps[0] {
				tempDeps[0] = append(tempDeps[0], uint64(j))
			}

			delayFlag := true

			for i := 1; i <= len(env.mvReadMapList)-1; i++ {
				reads := env.mvReadMapList[i-1]

				_, ok1 := reads[blockstm.NewSubpathKey(env.coinbase, state.BalancePath)]
				_, ok2 := reads[blockstm.NewSubpathKey(common.HexToAddress(w.chainConfig.Bor.CalculateBurntContract(env.header.Number.Uint64())), state.BalancePath)]

				if ok1 || ok2 {
					delayFlag = false
					break
				}

				for j := range deps[i] {
					tempDeps[i] = append(tempDeps[i], uint64(j))
				}
			}

			if err := rlp.DecodeBytes(env.header.Extra[types.ExtraVanityLength:len(env.header.Extra)-types.ExtraSealLength], &blockExtraData); err != nil {
				log.Error("error while decoding block extra data", "err", err)
				return err
			}

			if delayFlag {
				blockExtraData.TxDependency = tempDeps
			} else {
				blockExtraData.TxDependency = nil
			}
		} else {
			blockExtraData.TxDependency = nil
		}

		blockExtraDataBytes, err := rlp.EncodeToBytes(blockExtraData)
		if err != nil {
			log.Error("error while encoding block extra data: %v", err)
			return err
		}

		env.header.Extra = []byte{}

		env.header.Extra = append(tempVanity, blockExtraDataBytes...)

		env.header.Extra = append(env.header.Extra, tempSeal...)
	}

	if !w.IsRunning() && len(coalescedLogs) > 0 {
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

	return nil
}

// generateParams wraps various of settings for generating sealing task.
type generateParams struct {
	timestamp   uint64            // The timestamp for sealing task
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

	header.BlobGasUsed = nil
	header.ExcessBlobGas = nil
	header.ParentBeaconRoot = nil

	// Run the consensus preparation with the default or customized consensus engine.
	if err := w.engine.Prepare(w.chain, header); err != nil {
		switch err.(type) {
		case *bor.UnauthorizedSignerError:
			log.Debug("Failed to prepare header for sealing", "err", err)
		default:
			log.Error("Failed to prepare header for sealing", "err", err)
		}

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
		context := core.NewEVMBlockContext(header, w.chain, nil)
		vmenv := vm.NewEVM(context, vm.TxContext{}, env.state, w.chainConfig, vm.Config{})
		core.ProcessBeaconBlockRoot(*header.ParentBeaconRoot, vmenv, env.state)
	}
	return env, nil
}

func startProfiler(profile string, filepath string, number uint64) (func() error, error) {
	var (
		buf bytes.Buffer
		err error
	)

	closeFn := func() {}

	switch profile {
	case "cpu":
		err = pprof.StartCPUProfile(&buf)

		if err == nil {
			closeFn = func() {
				pprof.StopCPUProfile()
			}
		}
	case "trace":
		err = ptrace.Start(&buf)

		if err == nil {
			closeFn = func() {
				ptrace.Stop()
			}
		}
	case "heap":
		runtime.GC()

		err = pprof.WriteHeapProfile(&buf)
	default:
		log.Info("Incorrect profile name")
	}

	if err != nil {
		return func() error {
			closeFn()
			return nil
		}, err
	}

	closeFnNew := func() error {
		var err error

		closeFn()

		if buf.Len() == 0 {
			return nil
		}

		f, err := os.Create(filepath + "/" + profile + "-" + fmt.Sprint(number) + ".prof")
		if err != nil {
			return err
		}

		defer f.Close()

		_, err = f.Write(buf.Bytes())

		return err
	}

	return closeFnNew, nil
}

// fillTransactions retrieves the pending transactions from the txpool and fills them
// into the given sealing block. The transaction selection and ordering strategy can
// be customized with the plugin in the future.
//
//nolint:gocognit
func (w *worker) fillTransactions(ctx context.Context, interrupt *atomic.Int32, env *environment, interruptCtx context.Context) error {
	ctx, span := tracing.StartSpan(ctx, "fillTransactions")
	defer tracing.EndSpan(span)

	// Split the pending transactions into locals and remotes
	// Fill the block with all available pending transactions.
	pending := w.eth.TxPool().Pending(true)
	localTxs, remoteTxs := make(map[common.Address][]*txpool.LazyTransaction), pending

	var (
		localTxsCount  int
		remoteTxsCount int
	)

	// TODO: move to config or RPC
	const profiling = false

	if profiling {
		doneCh := make(chan struct{})

		defer func() {
			close(doneCh)
		}()

		go func(number uint64) {
			closeFn := func() error {
				return nil
			}

			for {
				select {
				case <-time.After(150 * time.Millisecond):
					// Check if we've not crossed limit
					if attempt := atomic.AddInt32(w.profileCount, 1); attempt >= 10 {
						log.Info("Completed profiling", "attempt", attempt)

						return
					}

					log.Info("Starting profiling in fill transactions", "number", number)

					dir, err := os.MkdirTemp("", fmt.Sprintf("bor-traces-%s-", time.Now().UTC().Format("2006-01-02-150405Z")))
					if err != nil {
						log.Error("Error in profiling", "path", dir, "number", number, "err", err)
						return
					}

					// grab the cpu profile
					closeFnInternal, err := startProfiler("cpu", dir, number)
					if err != nil {
						log.Error("Error in profiling", "path", dir, "number", number, "err", err)
						return
					}

					closeFn = func() error {
						err := closeFnInternal()

						log.Info("Completed profiling", "path", dir, "number", number, "error", err)

						return nil
					}

				case <-doneCh:
					err := closeFn()

					if err != nil {
						log.Info("closing fillTransactions", "number", number, "error", err)
					}

					return
				}
			}
		}(env.header.Number.Uint64())
	}

	tracing.Exec(ctx, "", "worker.SplittingTransactions", func(ctx context.Context, span trace.Span) {
		prePendingTime := time.Now()

		pending := w.eth.TxPool().Pending(true)
		remoteTxs = pending

		postPendingTime := time.Now()

		for _, account := range w.eth.TxPool().Locals() {
			if txs := remoteTxs[account]; len(txs) > 0 {
				delete(remoteTxs, account)

				localTxs[account] = txs
			}
		}

		postLocalsTime := time.Now()

		tracing.SetAttributes(
			span,
			attribute.Int("len of local txs", localTxsCount),
			attribute.Int("len of remote txs", remoteTxsCount),
			attribute.String("time taken by Pending()", fmt.Sprintf("%v", postPendingTime.Sub(prePendingTime))),
			attribute.String("time taken by Locals()", fmt.Sprintf("%v", postLocalsTime.Sub(postPendingTime))),
		)
	})

	var (
		localEnvTCount  int
		remoteEnvTCount int
		err             error
	)

	w.mu.RLock()
	tip := w.tip
	w.mu.RUnlock()

	if len(localTxs) > 0 {
		var txs *transactionsByPriceAndNonce

		tracing.Exec(ctx, "", "worker.LocalTransactionsByPriceAndNonce", func(ctx context.Context, span trace.Span) {
			var baseFee *uint256.Int
			if env.header.BaseFee != nil {
				baseFee = cmath.FromBig(env.header.BaseFee)
			}

			txs = newTransactionsByPriceAndNonce(env.signer, localTxs, baseFee.ToBig())

			tracing.SetAttributes(
				span,
				attribute.Int("len of tx local Heads", txs.GetTxs()),
			)
		})

		tracing.Exec(ctx, "", "worker.LocalCommitTransactions", func(ctx context.Context, span trace.Span) {
			err = w.commitTransactions(env, txs, interrupt, new(big.Int), interruptCtx)
		})

		if err != nil {
			return err
		}

		localEnvTCount = env.tcount
	}

	if len(remoteTxs) > 0 {
		var txs *transactionsByPriceAndNonce

		tracing.Exec(ctx, "", "worker.RemoteTransactionsByPriceAndNonce", func(ctx context.Context, span trace.Span) {
			var baseFee *uint256.Int
			if env.header.BaseFee != nil {
				baseFee = cmath.FromBig(env.header.BaseFee)
			}

			txs = newTransactionsByPriceAndNonce(env.signer, remoteTxs, baseFee.ToBig())

			tracing.SetAttributes(
				span,
				attribute.Int("len of tx remote Heads", txs.GetTxs()),
			)
		})

		tracing.Exec(ctx, "", "worker.RemoteCommitTransactions", func(ctx context.Context, span trace.Span) {
			err = w.commitTransactions(env, txs, interrupt, tip, interruptCtx)
		})

		if err != nil {
			return err
		}

		remoteEnvTCount = env.tcount
	}

	tracing.SetAttributes(
		span,
		attribute.Int("len of final local txs ", localEnvTCount),
		attribute.Int("len of final remote txs", remoteEnvTCount),
	)

	return nil
}

// generateWork generates a sealing block based on the given parameters.
func (w *worker) generateWork(ctx context.Context, params *generateParams) *newPayloadResult {
	work, err := w.prepareWork(params)
	if err != nil {
		return &newPayloadResult{err: err}
	}
	defer work.discard()

	// nolint : contextcheck
	var interruptCtx = context.Background()

	if !params.noTxs {
		interrupt := new(atomic.Int32)

		timer := time.AfterFunc(w.newpayloadTimeout, func() {
			interrupt.Store(commitInterruptTimeout)
		})
		defer timer.Stop()

		err := w.fillTransactions(ctx, interrupt, work, interruptCtx)
		if errors.Is(err, errBlockInterruptedByTimeout) {
			log.Warn("Block building is interrupted", "allowance", common.PrettyDuration(w.newpayloadTimeout))
		}
	}
	block, err := w.engine.FinalizeAndAssemble(ctx, w.chain, work.header, work.state, work.txs, nil, work.receipts, params.withdrawals)
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
func (w *worker) commitWork(ctx context.Context, interrupt *atomic.Int32, noempty bool, timestamp int64) {
	// Abort committing if node is still syncing
	if w.syncing.Load() {
		return
	}
	start := time.Now()

	var (
		work *environment
		err  error
	)

	tracing.Exec(ctx, "", "worker.prepareWork", func(ctx context.Context, span trace.Span) {
		// Set the coinbase if the worker is running or it's required
		var coinbase common.Address
		if w.IsRunning() {
			coinbase = w.etherbase()
			if coinbase == (common.Address{}) {
				log.Error("Refusing to mine without etherbase")
				return
			}
		}

		work, err = w.prepareWork(&generateParams{
			timestamp: uint64(timestamp),
			coinbase:  coinbase,
		})
	})

	if err != nil {
		return
	}

	// nolint:contextcheck
	var interruptCtx = context.Background()

	stopFn := func() {}
	defer func() {
		stopFn()
	}()

	if !noempty && w.interruptCommitFlag {
		block := w.chain.GetBlockByHash(w.chain.CurrentBlock().Hash())
		interruptCtx, stopFn = getInterruptTimer(ctx, work, block)
		// nolint : staticcheck
		interruptCtx = vm.PutCache(interruptCtx, w.interruptedTxCache)
	}

	ctx, span := tracing.StartSpan(ctx, "commitWork")
	defer tracing.EndSpan(span)

	tracing.SetAttributes(
		span,
		attribute.Int("number", int(work.header.Number.Uint64())),
	)

	// Create an empty block based on temporary copied state for
	// sealing in advance without waiting block execution finished.
	if !noempty && !w.noempty.Load() {
		_ = w.commit(ctx, work.copy(), nil, false, start)
	}
	// Fill pending transactions from the txpool into the block.
	err = w.fillTransactions(ctx, interrupt, work, interruptCtx)

	switch {
	case err == nil:
		// The entire block is filled, decrease resubmit interval in case
		// of current interval is larger than the user-specified one.
		w.adjustResubmitInterval(&intervalAdjust{inc: false})

	case errors.Is(err, errBlockInterruptedByRecommit):
		// Notify resubmit loop to increase resubmitting interval if the
		// interruption is due to frequent commits.
		gaslimit := work.header.GasLimit

		ratio := float64(gaslimit-work.gasPool.Gas()) / float64(gaslimit)
		if ratio < 0.1 {
			ratio = 0.1
		}
		w.adjustResubmitInterval(&intervalAdjust{
			ratio: ratio,
			inc:   true,
		})

	case errors.Is(err, errBlockInterruptedByNewHead):
		// If the block building is interrupted by newhead event, discard it
		// totally. Committing the interrupted block introduces unnecessary
		// delay, and possibly causes miner to mine on the previous head,
		// which could result in higher uncle rate.
		work.discard()
		return
	}
	// Submit the generated block for consensus sealing.
	_ = w.commit(ctx, work.copy(), w.fullTaskHook, true, start)

	// Swap out the old work with the new one, terminating any leftover
	// prefetcher processes in the mean time and starting a new one.
	if w.current != nil {
		w.current.discard()
	}

	w.current = work
}

func getInterruptTimer(ctx context.Context, work *environment, current *types.Block) (context.Context, func()) {
	delay := time.Until(time.Unix(int64(work.header.Time), 0))

	interruptCtx, cancel := context.WithTimeout(context.Background(), delay)

	blockNumber := current.NumberU64() + 1

	go func() {
		select {
		case <-interruptCtx.Done():
			if interruptCtx.Err() != context.Canceled {
				log.Info("Commit Interrupt. Pre-committing the current block", "block", blockNumber)
				cancel()
			}
		case <-ctx.Done(): // nothing to do
		}
	}()

	return interruptCtx, cancel
}

// commit runs any post-transaction state modifications, assembles the final block
// and commits new work if consensus engine is running.
// Note the assumption is held that the mutation is allowed to the passed env, do
// the deep copy first.
func (w *worker) commit(ctx context.Context, env *environment, interval func(), update bool, start time.Time) error {
	if w.IsRunning() {
		ctx, span := tracing.StartSpan(ctx, "commit")
		defer tracing.EndSpan(span)

		if interval != nil {
			interval()
		}
		// Create a local environment copy, avoid the data race with snapshot state.
		// https://github.com/ethereum/go-ethereum/issues/24299
		env := env.copy()
		// Withdrawals are set to nil here, because this is only called in PoW.
		block, err := w.engine.FinalizeAndAssemble(ctx, w.chain, env.header, env.state, env.txs, nil, env.receipts, nil)
		tracing.SetAttributes(
			span,
			attribute.Int("number", int(env.header.Number.Uint64())),
			attribute.String("hash", env.header.Hash().String()),
			attribute.String("sealhash", w.engine.SealHash(env.header).String()),
			attribute.Int("len of env.txs", len(env.txs)),
			attribute.Bool("error", err != nil),
		)

		if err != nil {
			return err
		}

		// If we're post merge, just ignore
		if !w.isTTDReached(block.Header()) {
			select {
			case w.taskCh <- &task{ctx: ctx, receipts: env.receipts, state: env.state, block: block, createdAt: time.Now()}:
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
	ctx := tracing.WithTracer(context.Background(), otel.GetTracerProvider().Tracer("getSealingBlock"))

	req := &getWorkReq{
		params: params,
		result: make(chan *newPayloadResult, 1),
		ctx:    ctx,
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

// adjustResubmitInterval adjusts the resubmit interval.
func (w *worker) adjustResubmitInterval(message *intervalAdjust) {
	select {
	case w.resubmitAdjustCh <- message:
	default:
		log.Warn("the resubmitAdjustCh is full, discard the message")
	}
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
