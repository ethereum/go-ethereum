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
	"errors"
	"math"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/consensus"
	"github.com/scroll-tech/go-ethereum/consensus/misc"
	"github.com/scroll-tech/go-ethereum/core"
	"github.com/scroll-tech/go-ethereum/core/rawdb"
	"github.com/scroll-tech/go-ethereum/core/state"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/event"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/metrics"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/scroll-tech/go-ethereum/rollup/circuitcapacitychecker"
	"github.com/scroll-tech/go-ethereum/rollup/fees"
	"github.com/scroll-tech/go-ethereum/rollup/pipeline"
	"github.com/scroll-tech/go-ethereum/trie"
)

const (
	// txChanSize is the size of channel listening to NewTxsEvent.
	// The number is referenced from the size of tx pool.
	txChanSize = 4096

	// chainHeadChanSize is the size of channel listening to ChainHeadEvent.
	chainHeadChanSize = 10

	// chainSideChanSize is the size of channel listening to ChainSideEvent.
	chainSideChanSize = 10

	// minRecommitInterval is the minimal time interval to recreate the mining block with
	// any newly arrived transactions.
	minRecommitInterval = 1 * time.Second
)

var (
	// Metrics for the skipped txs
	l1TxGasLimitExceededCounter       = metrics.NewRegisteredCounter("miner/skipped_txs/l1/gas_limit_exceeded", nil)
	l1TxRowConsumptionOverflowCounter = metrics.NewRegisteredCounter("miner/skipped_txs/l1/row_consumption_overflow", nil)
	l2TxRowConsumptionOverflowCounter = metrics.NewRegisteredCounter("miner/skipped_txs/l2/row_consumption_overflow", nil)
	l1TxCccUnknownErrCounter          = metrics.NewRegisteredCounter("miner/skipped_txs/l1/ccc_unknown_err", nil)
	l2TxCccUnknownErrCounter          = metrics.NewRegisteredCounter("miner/skipped_txs/l2/ccc_unknown_err", nil)
	l1TxStrangeErrCounter             = metrics.NewRegisteredCounter("miner/skipped_txs/l1/strange_err", nil)

	collectL1MsgsTimer = metrics.NewRegisteredTimer("miner/collect_l1_msgs", nil)
	prepareTimer       = metrics.NewRegisteredTimer("miner/prepare", nil)
	collectL2Timer     = metrics.NewRegisteredTimer("miner/collect_l2_txns", nil)
	l2CommitTimer      = metrics.NewRegisteredTimer("miner/commit", nil)

	commitReasonCCCCounter      = metrics.NewRegisteredCounter("miner/commit_reason_ccc", nil)
	commitReasonDeadlineCounter = metrics.NewRegisteredCounter("miner/commit_reason_deadline", nil)
	commitGasCounter            = metrics.NewRegisteredCounter("miner/commit_gas", nil)
)

// newWorkReq represents a request for new sealing work submitting with relative interrupt notifier.
type newWorkReq struct {
	noempty   bool
	timestamp int64
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
	chainSideCh  chan core.ChainSideEvent
	chainSideSub event.Subscription

	// Channels
	newWorkCh chan *newWorkReq
	startCh   chan struct{}
	exitCh    chan struct{}

	wg sync.WaitGroup

	currentPipelineStart time.Time
	currentPipeline      *pipeline.Pipeline

	mu       sync.RWMutex // The lock used to protect the coinbase and extra fields
	coinbase common.Address
	extra    []byte

	snapshotMu       sync.RWMutex // The lock used to protect the snapshots below
	snapshotBlock    *types.Block
	snapshotReceipts types.Receipts
	snapshotState    *state.StateDB

	// atomic status counters
	running   int32 // The indicator whether the consensus engine is running or not.
	newTxs    int32 // New arrival transaction count since last sealing work submitting.
	newL1Msgs int32 // New arrival L1 message count since last sealing work submitting.

	// noempty is the flag used to control whether the feature of pre-seal empty
	// block is enabled. The default value is false(pre-seal is enabled by default).
	// But in some special scenario the consensus engine will seal blocks instantaneously,
	// in this case this feature will add all empty blocks into canonical chain
	// non-stop and no real transaction will be included.
	noempty uint32

	// External functions
	isLocalBlock func(block *types.Block) bool // Function used to determine whether the specified block is mined by local miner.

	circuitCapacityChecker *circuitcapacitychecker.CircuitCapacityChecker
	prioritizedTx          *prioritizedTransaction

	// Test hooks
	beforeTxHook func() // Method to call before processing a transaction.
}

func newWorker(config *Config, chainConfig *params.ChainConfig, engine consensus.Engine, eth Backend, mux *event.TypeMux, isLocalBlock func(*types.Block) bool, init bool) *worker {
	worker := &worker{
		config:                 config,
		chainConfig:            chainConfig,
		engine:                 engine,
		eth:                    eth,
		mux:                    mux,
		chain:                  eth.BlockChain(),
		isLocalBlock:           isLocalBlock,
		txsCh:                  make(chan core.NewTxsEvent, txChanSize),
		chainHeadCh:            make(chan core.ChainHeadEvent, chainHeadChanSize),
		chainSideCh:            make(chan core.ChainSideEvent, chainSideChanSize),
		newWorkCh:              make(chan *newWorkReq),
		exitCh:                 make(chan struct{}),
		startCh:                make(chan struct{}, 1),
		circuitCapacityChecker: circuitcapacitychecker.NewCircuitCapacityChecker(true),
	}
	log.Info("created new worker", "CircuitCapacityChecker ID", worker.circuitCapacityChecker.ID)

	// Subscribe NewTxsEvent for tx pool
	worker.txsSub = eth.TxPool().SubscribeNewTxsEvent(worker.txsCh)

	// Subscribe events for blockchain
	worker.chainHeadSub = eth.BlockChain().SubscribeChainHeadEvent(worker.chainHeadCh)
	worker.chainSideSub = eth.BlockChain().SubscribeChainSideEvent(worker.chainSideCh)

	// Sanitize recommit interval if the user-specified one is too short.
	recommit := worker.config.Recommit
	if recommit < minRecommitInterval {
		log.Warn("Sanitizing miner recommit interval", "provided", recommit, "updated", minRecommitInterval)
		recommit = minRecommitInterval
	}

	// Sanitize account fetch limit.
	if worker.config.MaxAccountsNum == 0 {
		log.Warn("Sanitizing miner account fetch limit", "provided", worker.config.MaxAccountsNum, "updated", math.MaxInt)
		worker.config.MaxAccountsNum = math.MaxInt
	}

	worker.wg.Add(2)
	go worker.mainLoop()
	go worker.newWorkLoop(recommit)

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

// disablePreseal disables pre-sealing mining feature
func (w *worker) disablePreseal() {
	atomic.StoreUint32(&w.noempty, 1)
}

// enablePreseal enables pre-sealing mining feature
func (w *worker) enablePreseal() {
	atomic.StoreUint32(&w.noempty, 0)
}

// pending returns the pending state and corresponding block.
func (w *worker) pending() (*types.Block, *state.StateDB) {
	// return a snapshot to avoid contention on currentMu mutex
	w.snapshotMu.RLock()
	defer w.snapshotMu.RUnlock()
	if w.snapshotState == nil {
		return nil, nil
	}
	return w.snapshotBlock, w.snapshotState.Copy()
}

// pendingBlock returns pending block.
func (w *worker) pendingBlock() *types.Block {
	// return a snapshot to avoid contention on currentMu mutex
	w.snapshotMu.RLock()
	defer w.snapshotMu.RUnlock()
	return w.snapshotBlock
}

// pendingBlockAndReceipts returns pending block and corresponding receipts.
func (w *worker) pendingBlockAndReceipts() (*types.Block, types.Receipts) {
	// return a snapshot to avoid contention on currentMu mutex
	w.snapshotMu.RLock()
	defer w.snapshotMu.RUnlock()
	return w.snapshotBlock, w.snapshotReceipts
}

// start sets the running status as 1 and triggers new work submitting.
func (w *worker) start() {
	atomic.StoreInt32(&w.running, 1)
	w.startCh <- struct{}{}
}

// stop sets the running status as 0.
func (w *worker) stop() {
	atomic.StoreInt32(&w.running, 0)
}

// isRunning returns an indicator whether worker is running or not.
func (w *worker) isRunning() bool {
	return atomic.LoadInt32(&w.running) == 1
}

// close terminates all background threads maintained by the worker.
// Note the worker does not support being closed multiple times.
func (w *worker) close() {
	atomic.StoreInt32(&w.running, 0)
	close(w.exitCh)
	w.wg.Wait()
}

// newWorkLoop is a standalone goroutine to submit new mining work upon received events.
func (w *worker) newWorkLoop(recommit time.Duration) {
	defer w.wg.Done()
	var (
		timestamp int64 // timestamp for each round of mining.
	)

	// commit aborts in-flight transaction execution with given signal and resubmits a new one.
	commit := func(noempty bool) {
		select {
		case w.newWorkCh <- &newWorkReq{noempty: noempty, timestamp: timestamp}:
		case <-w.exitCh:
			return
		}
		atomic.StoreInt32(&w.newTxs, 0)
		atomic.StoreInt32(&w.newL1Msgs, 0)
	}

	for {
		select {
		case <-w.startCh:
			timestamp = time.Now().Unix()
			commit(false)
		case <-w.chainHeadCh:
			timestamp = time.Now().Unix()
			commit(true)
		case <-w.exitCh:
			return
		}
	}
}

// mainLoop is a standalone goroutine to regenerate the sealing task based on the received event.
func (w *worker) mainLoop() {
	defer w.wg.Done()
	defer w.txsSub.Unsubscribe()
	defer w.chainHeadSub.Unsubscribe()
	defer w.chainSideSub.Unsubscribe()

	deadCh := make(chan *pipeline.Result)
	pipelineResultCh := func() <-chan *pipeline.Result {
		if w.currentPipeline == nil {
			return deadCh
		}
		return w.currentPipeline.ResultCh
	}

	for {
		select {
		case req := <-w.newWorkCh:
			w.startNewPipeline(req.timestamp)
		case result := <-pipelineResultCh():
			w.handlePipelineResult(result)
		case ev := <-w.txsCh:
			// Apply transactions to the pending state
			//
			// Note all transactions received may not be continuous with transactions
			// already included in the current mining block. These transactions will
			// be automatically eliminated.
			if w.currentPipeline != nil {
				txs := make(map[common.Address]types.Transactions)
				signer := types.MakeSigner(w.chainConfig, w.currentPipeline.Header.Number)
				for _, tx := range ev.Txs {
					acc, _ := types.Sender(signer, tx)
					txs[acc] = append(txs[acc], tx)
				}
				txset := types.NewTransactionsByPriceAndNonce(signer, txs, w.currentPipeline.Header.BaseFee)
				if result := w.currentPipeline.TryPushTxns(txset, w.onTxFailingInPipeline); result != nil {
					w.handlePipelineResult(result)
				}
			}
			atomic.AddInt32(&w.newTxs, int32(len(ev.Txs)))

		// System stopped
		case <-w.exitCh:
			return
		case <-w.txsSub.Err():
			return
		case <-w.chainHeadSub.Err():
			return
		case <-w.chainSideSub.Err():
			return
		}
	}
}

// updateSnapshot updates pending snapshot block and state.
// Note this function assumes the current variable is thread safe.
func (w *worker) updateSnapshot(current *pipeline.BlockCandidate) {
	w.snapshotMu.Lock()
	defer w.snapshotMu.Unlock()

	w.snapshotBlock = types.NewBlock(
		current.Header,
		current.Txs,
		nil,
		current.Receipts,
		trie.NewStackTrie(nil),
	)
	w.snapshotReceipts = copyReceipts(current.Receipts)
	w.snapshotState = current.State.Copy()
}

func (w *worker) collectPendingL1Messages(startIndex uint64) []types.L1MessageTx {
	maxCount := w.chainConfig.Scroll.L1Config.NumL1MessagesPerBlock
	return rawdb.ReadL1MessagesFrom(w.eth.ChainDb(), startIndex, maxCount)
}

// startNewPipeline generates several new sealing tasks based on the parent block.
func (w *worker) startNewPipeline(timestamp int64) {

	if w.currentPipeline != nil {
		w.currentPipeline.Release()
		w.currentPipeline = nil
	}

	parent := w.chain.CurrentBlock()

	num := parent.Number()
	header := &types.Header{
		ParentHash: parent.Hash(),
		Number:     num.Add(num, common.Big1),
		GasLimit:   core.CalcGasLimit(parent.GasLimit(), w.config.GasCeil),
		Extra:      w.extra,
		Time:       uint64(timestamp),
	}
	// Set baseFee if we are on an EIP-1559 chain
	if w.chainConfig.IsCurie(header.Number) {
		state, err := w.chain.StateAt(parent.Root())
		if err != nil {
			log.Error("Failed to create mining context", "err", err)
			return
		}
		parentL1BaseFee := fees.GetL1BaseFee(state)
		header.BaseFee = misc.CalcBaseFee(w.chainConfig, parent.Header(), parentL1BaseFee)
	}
	// Only set the coinbase if our consensus engine is running (avoid spurious block rewards)
	if w.isRunning() {
		if w.coinbase == (common.Address{}) {
			log.Error("Refusing to mine without etherbase")
			return
		}
		header.Coinbase = w.coinbase
	}

	common.WithTimer(prepareTimer, func() {
		if err := w.engine.Prepare(w.chain, header); err != nil {
			log.Error("Failed to prepare header for mining", "err", err)
			return
		}
	})

	// If we are care about TheDAO hard-fork check whether to override the extra-data or not
	if daoBlock := w.chainConfig.DAOForkBlock; daoBlock != nil {
		// Check whether the block is among the fork extra-override range
		limit := new(big.Int).Add(daoBlock, params.DAOForkExtraRange)
		if header.Number.Cmp(daoBlock) >= 0 && header.Number.Cmp(limit) < 0 {
			// Depending whether we support or oppose the fork, override differently
			if w.chainConfig.DAOForkSupport {
				header.Extra = common.CopyBytes(params.DAOForkBlockExtra)
			} else if bytes.Equal(header.Extra, params.DAOForkBlockExtra) {
				header.Extra = []byte{} // If miner opposes, don't let it use the reserved extra-data
			}
		}
	}

	parentState, err := w.chain.StateAt(parent.Root())
	if err != nil {
		log.Error("failed to fetch parent state", "err", err)
		return
	}

	// Apply special state transition at Curie block
	if w.chainConfig.CurieBlock != nil && w.chainConfig.CurieBlock.Cmp(header.Number) == 0 {
		misc.ApplyCurieHardFork(parentState)

		// zkEVM requirement: Curie transition block contains 0 transactions, bypass pipeline.
		err = w.commit(&pipeline.Result{
			// Note: Signer nodes will not store CCC results for empty blocks in their database.
			// In practice, this is acceptable, since this block will never overflow, and follower
			// nodes will still store CCC results.
			Rows: &types.RowConsumption{},
			FinalBlock: &pipeline.BlockCandidate{
				Header:        header,
				State:         parentState,
				Txs:           types.Transactions{},
				Receipts:      types.Receipts{},
				CoalescedLogs: []*types.Log{},
			},
		})

		if err != nil {
			log.Error("failed to commit Curie fork block", "reason", err)
		}

		return
	}

	// fetch l1Txs
	var l1Messages []types.L1MessageTx
	if w.chainConfig.Scroll.ShouldIncludeL1Messages() {
		common.WithTimer(collectL1MsgsTimer, func() {
			l1Messages = w.collectPendingL1Messages(*rawdb.ReadFirstQueueIndexNotInL2Block(w.eth.ChainDb(), parent.Hash()))
		})
	}

	tidyPendingStart := time.Now()
	// Fill the block with all available pending transactions.
	pending := w.eth.TxPool().PendingWithMax(false, w.config.MaxAccountsNum)
	// Split the pending transactions into locals and remotes
	localTxs, remoteTxs := make(map[common.Address]types.Transactions), pending
	for _, account := range w.eth.TxPool().Locals() {
		if txs := remoteTxs[account]; len(txs) > 0 {
			delete(remoteTxs, account)
			localTxs[account] = txs
		}
	}
	collectL2Timer.UpdateSince(tidyPendingStart)

	var nextL1MsgIndex uint64
	if dbIndex := rawdb.ReadFirstQueueIndexNotInL2Block(w.chain.Database(), parent.Hash()); dbIndex != nil {
		nextL1MsgIndex = *dbIndex
	} else {
		log.Error("failed to read nextL1MsgIndex", "parent", parent.Hash())
		return
	}

	w.currentPipelineStart = time.Now()
	w.currentPipeline = pipeline.NewPipeline(w.chain, w.chain.GetVMConfig(), parentState, header, nextL1MsgIndex, w.getCCC()).WithBeforeTxHook(w.beforeTxHook)

	deadline := time.Unix(int64(header.Time), 0)
	if w.chainConfig.Clique != nil && w.chainConfig.Clique.RelaxedPeriod {
		// clique with relaxed period uses time.Now() as the header.Time, calculate the deadline
		deadline = time.Unix(int64(header.Time+w.chainConfig.Clique.Period), 0)
	}

	if err := w.currentPipeline.Start(deadline); err != nil {
		log.Error("failed to start pipeline", "err", err)
		return
	}

	// Short circuit if there is no available pending transactions.
	// But if we disable empty precommit already, ignore it. Since
	// empty block is necessary to keep the liveness of the network.
	if len(localTxs) == 0 && len(remoteTxs) == 0 && len(l1Messages) == 0 && atomic.LoadUint32(&w.noempty) == 0 {
		return
	}

	if w.chainConfig.Scroll.ShouldIncludeL1Messages() && len(l1Messages) > 0 {
		log.Trace("Processing L1 messages for inclusion", "count", len(l1Messages))
		txs, err := types.NewL1MessagesByQueueIndex(l1Messages)
		if err != nil {
			log.Error("Failed to create L1 message set", "l1Messages", l1Messages, "err", err)
			return
		}

		if result := w.currentPipeline.TryPushTxns(txs, w.onTxFailingInPipeline); result != nil {
			w.handlePipelineResult(result)
			return
		}
	}
	signer := types.MakeSigner(w.chainConfig, header.Number)

	if w.prioritizedTx != nil && w.currentPipeline.Header.Number.Uint64() > w.prioritizedTx.blockNumber {
		w.prioritizedTx = nil
	}
	if w.prioritizedTx != nil {
		from, _ := types.Sender(signer, w.prioritizedTx.tx) // error already checked before
		txList := map[common.Address]types.Transactions{from: []*types.Transaction{w.prioritizedTx.tx}}
		txs := types.NewTransactionsByPriceAndNonce(signer, txList, header.BaseFee)
		if result := w.currentPipeline.TryPushTxns(txs, w.onTxFailingInPipeline); result != nil {
			w.handlePipelineResult(result)
			return
		}
	}

	if len(localTxs) > 0 {
		txs := types.NewTransactionsByPriceAndNonce(signer, localTxs, header.BaseFee)
		if result := w.currentPipeline.TryPushTxns(txs, w.onTxFailingInPipeline); result != nil {
			w.handlePipelineResult(result)
			return
		}
	}
	if len(remoteTxs) > 0 {
		txs := types.NewTransactionsByPriceAndNonce(signer, remoteTxs, header.BaseFee)
		if result := w.currentPipeline.TryPushTxns(txs, w.onTxFailingInPipeline); result != nil {
			w.handlePipelineResult(result)
			return
		}
	}
}

func (w *worker) handlePipelineResult(res *pipeline.Result) error {
	if !w.isRunning() {
		if res != nil && res.FinalBlock != nil {
			w.updateSnapshot(res.FinalBlock)
		}
		w.currentPipeline.Release()
		w.currentPipeline = nil
		return nil
	}

	if res != nil && res.OverflowingTx != nil {
		if res.FinalBlock == nil {
			// first txn overflowed the circuit, skip
			log.Info("Circuit capacity limit reached for a single tx", "tx", res.OverflowingTx.Hash().String(),
				"isL1Message", res.OverflowingTx.IsL1MessageTx(), "reason", res.CCCErr.Error())

			// Store skipped transaction in local db
			overflowingTrace := res.OverflowingTrace
			if !w.config.StoreSkippedTxTraces {
				overflowingTrace = nil
			}
			rawdb.WriteSkippedTransaction(w.eth.ChainDb(), res.OverflowingTx, overflowingTrace, res.CCCErr.Error(),
				w.currentPipeline.Header.Number.Uint64(), nil)

			if overflowingL1MsgTx := res.OverflowingTx.AsL1MessageTx(); overflowingL1MsgTx != nil {
				rawdb.WriteFirstQueueIndexNotInL2Block(w.eth.ChainDb(), w.currentPipeline.Header.ParentHash, overflowingL1MsgTx.QueueIndex+1)
			} else {
				w.prioritizedTx = nil
				w.eth.TxPool().RemoveTx(res.OverflowingTx.Hash(), true)
			}
		} else if !res.OverflowingTx.IsL1MessageTx() {
			// prioritize overflowing L2 message as the first txn next block
			// no need to prioritize L1 messages, they are fetched in order
			// and processed first in every block anyways
			w.prioritizedTx = &prioritizedTransaction{
				blockNumber: w.currentPipeline.Header.Number.Uint64() + 1,
				tx:          res.OverflowingTx,
			}
		}

		switch {
		case res.OverflowingTx.IsL1MessageTx() &&
			errors.Is(res.CCCErr, circuitcapacitychecker.ErrBlockRowConsumptionOverflow):
			l1TxRowConsumptionOverflowCounter.Inc(1)
		case !res.OverflowingTx.IsL1MessageTx() &&
			errors.Is(res.CCCErr, circuitcapacitychecker.ErrBlockRowConsumptionOverflow):
			l2TxRowConsumptionOverflowCounter.Inc(1)
		case res.OverflowingTx.IsL1MessageTx() &&
			errors.Is(res.CCCErr, circuitcapacitychecker.ErrUnknown):
			l1TxCccUnknownErrCounter.Inc(1)
		case !res.OverflowingTx.IsL1MessageTx() &&
			errors.Is(res.CCCErr, circuitcapacitychecker.ErrUnknown):
			l2TxCccUnknownErrCounter.Inc(1)
		}
	}

	var commitError error
	if res != nil && res.FinalBlock != nil {
		if commitError = w.commit(res); commitError == nil {
			return nil
		}
		log.Error("Commit failed", "header", res.FinalBlock.Header, "reason", commitError)
	}
	w.startNewPipeline(time.Now().Unix())
	return commitError
}

// commit runs any post-transaction state modifications, assembles the final block
// and commits new work if consensus engine is running.
func (w *worker) commit(res *pipeline.Result) error {
	defer func(t0 time.Time) {
		l2CommitTimer.Update(time.Since(t0))
	}(time.Now())

	if res.CCCErr != nil {
		commitReasonCCCCounter.Inc(1)
	} else {
		commitReasonDeadlineCounter.Inc(1)
	}
	commitGasCounter.Inc(int64(res.FinalBlock.Header.GasUsed))

	block, err := w.engine.FinalizeAndAssemble(w.chain, res.FinalBlock.Header, res.FinalBlock.State,
		res.FinalBlock.Txs, nil, res.FinalBlock.Receipts)
	if err != nil {
		return err
	}

	sealHash := w.engine.SealHash(block.Header())
	log.Info("Committing new mining work", "number", block.Number(), "sealhash", sealHash,
		"txs", res.FinalBlock.Txs.Len(),
		"gas", block.GasUsed(), "fees", totalFees(block, res.FinalBlock.Receipts),
		"elapsed", common.PrettyDuration(time.Since(w.currentPipelineStart)))

	resultCh, stopCh := make(chan *types.Block), make(chan struct{})
	if err := w.engine.Seal(w.chain, block, resultCh, stopCh); err != nil {
		return err
	}
	// Clique.Seal() will only wait for a second before giving up on us. So make sure there is nothing computational heavy
	// or a call that blocks between the call to Seal and the line below
	block = <-resultCh
	if block == nil {
		return errors.New("missed seal response from consensus engine")
	}

	// verify the generated block with local consensus engine to make sure everything is as expected
	if err = w.engine.VerifyHeader(w.chain, block.Header(), true); err != nil {
		return err
	}

	blockHash := block.Hash()

	for i, receipt := range res.FinalBlock.Receipts {
		// add block location fields
		receipt.BlockHash = blockHash
		receipt.BlockNumber = block.Number()
		receipt.TransactionIndex = uint(i)

		for _, log := range receipt.Logs {
			log.BlockHash = blockHash
		}
	}

	for _, log := range res.FinalBlock.CoalescedLogs {
		log.BlockHash = blockHash
	}

	// It's possible that we've stored L1 queue index for this block previously,
	// in this case do not overwrite it.
	if index := rawdb.ReadFirstQueueIndexNotInL2Block(w.eth.ChainDb(), blockHash); index == nil {
		// Store first L1 queue index not processed by this block.
		// Note: This accounts for both included and skipped messages. This
		// way, if a block only skips messages, we won't reprocess the same
		// messages from the next block.
		log.Trace(
			"Worker WriteFirstQueueIndexNotInL2Block",
			"number", block.Number(),
			"hash", blockHash.String(),
			"nextL1MsgIndex", res.FinalBlock.NextL1MsgIndex,
		)
		rawdb.WriteFirstQueueIndexNotInL2Block(w.eth.ChainDb(), blockHash, res.FinalBlock.NextL1MsgIndex)
	} else {
		log.Trace(
			"Worker WriteFirstQueueIndexNotInL2Block: not overwriting existing index",
			"number", block.Number(),
			"hash", blockHash.String(),
			"index", *index,
			"nextL1MsgIndex", res.FinalBlock.NextL1MsgIndex,
		)
	}
	// Store circuit row consumption.
	log.Trace(
		"Worker write block row consumption",
		"id", w.circuitCapacityChecker.ID,
		"number", block.Number(),
		"hash", blockHash.String(),
		"accRows", res.Rows,
	)

	rawdb.WriteBlockRowConsumption(w.eth.ChainDb(), blockHash, res.Rows)
	// Commit block and state to database.
	_, err = w.chain.WriteBlockWithState(block, res.FinalBlock.Receipts, res.FinalBlock.CoalescedLogs, res.FinalBlock.State, true)
	if err != nil {
		return err
	}

	log.Info("Successfully sealed new block", "number", block.Number(), "sealhash", sealHash, "hash", blockHash)

	// Broadcast the block and announce chain insertion event
	w.mux.Post(core.NewMinedBlockEvent{Block: block})

	w.currentPipeline.Release()
	w.currentPipeline = nil
	return nil
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

// postSideBlock fires a side chain event, only use it for testing.
func (w *worker) postSideBlock(event core.ChainSideEvent) {
	select {
	case w.chainSideCh <- event:
	case <-w.exitCh:
	}
}

func (w *worker) onTxFailingInPipeline(txIndex int, tx *types.Transaction, err error) bool {
	if !w.isRunning() {
		return false
	}

	writeTrace := func() {
		var trace *types.BlockTrace
		var errWithTrace *pipeline.ErrorWithTrace
		if w.config.StoreSkippedTxTraces && errors.As(err, &errWithTrace) {
			trace = errWithTrace.Trace
		}
		rawdb.WriteSkippedTransaction(w.eth.ChainDb(), tx, trace, err.Error(),
			w.currentPipeline.Header.Number.Uint64(), nil)
	}

	switch {
	case errors.Is(err, core.ErrGasLimitReached) && tx.IsL1MessageTx():
		// If this block already contains some L1 messages try again in the next block.
		if txIndex > 0 {
			break
		}
		// A single L1 message leads to out-of-gas. Skip it.
		queueIndex := tx.AsL1MessageTx().QueueIndex
		log.Info("Skipping L1 message", "queueIndex", queueIndex, "tx", tx.Hash().String(), "block",
			w.currentPipeline.Header.Number, "reason", "gas limit exceeded")
		writeTrace()
		l1TxGasLimitExceededCounter.Inc(1)

	case errors.Is(err, core.ErrInsufficientFunds):
		log.Trace("Skipping tx with insufficient funds", "tx", tx.Hash().String())
		w.eth.TxPool().RemoveTx(tx.Hash(), true)

	case errors.Is(err, pipeline.ErrUnexpectedL1MessageIndex):
		log.Warn(
			"Unexpected L1 message queue index in worker",
			"got", tx.AsL1MessageTx().QueueIndex,
		)
	case errors.Is(err, core.ErrGasLimitReached), errors.Is(err, core.ErrNonceTooLow), errors.Is(err, core.ErrNonceTooHigh), errors.Is(err, core.ErrTxTypeNotSupported):
		break
	default:
		// Strange error
		log.Debug("Transaction failed, account skipped", "hash", tx.Hash().String(), "err", err)
		if tx.IsL1MessageTx() {
			queueIndex := tx.AsL1MessageTx().QueueIndex
			log.Info("Skipping L1 message", "queueIndex", queueIndex, "tx", tx.Hash().String(), "block",
				w.currentPipeline.Header.Number, "reason", "strange error", "err", err)
			writeTrace()
			l1TxStrangeErrCounter.Inc(1)
		}
	}
	return false
}

// totalFees computes total consumed miner fees in ETH. Block transactions and receipts have to have the same order.
func totalFees(block *types.Block, receipts []*types.Receipt) *big.Float {
	feesWei := new(big.Int)
	for i, tx := range block.Transactions() {
		minerFee, _ := tx.EffectiveGasTip(block.BaseFee())
		feesWei.Add(feesWei, new(big.Int).Mul(new(big.Int).SetUint64(receipts[i].GasUsed), minerFee))
	}
	return new(big.Float).Quo(new(big.Float).SetInt(feesWei), new(big.Float).SetInt(big.NewInt(params.Ether)))
}
