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
	"github.com/scroll-tech/go-ethereum/core/vm"
	"github.com/scroll-tech/go-ethereum/event"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/metrics"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/scroll-tech/go-ethereum/rollup/ccc"
	"github.com/scroll-tech/go-ethereum/rollup/fees"
	"github.com/scroll-tech/go-ethereum/trie"
)

const (
	// txChanSize is the size of channel listening to NewTxsEvent.
	// The number is referenced from the size of tx pool.
	txChanSize = 4096

	// chainHeadChanSize is the size of channel listening to ChainHeadEvent.
	chainHeadChanSize = 10
)

var (
	deadCh = make(chan time.Time)

	ErrUnexpectedL1MessageIndex = errors.New("unexpected L1 message index")

	// Metrics for the skipped txs
	l1SkippedCounter = metrics.NewRegisteredCounter("miner/skipped_txs/l1", nil)
	l2SkippedCounter = metrics.NewRegisteredCounter("miner/skipped_txs/l2", nil)

	collectL1MsgsTimer = metrics.NewRegisteredTimer("miner/collect_l1_msgs", nil)
	prepareTimer       = metrics.NewRegisteredTimer("miner/prepare", nil)
	collectL2Timer     = metrics.NewRegisteredTimer("miner/collect_l2_txns", nil)
	l2CommitTimer      = metrics.NewRegisteredTimer("miner/commit", nil)
	cccStallTimer      = metrics.NewRegisteredTimer("miner/ccc_stall", nil)

	commitReasonCCCCounter      = metrics.NewRegisteredCounter("miner/commit_reason_ccc", nil)
	commitReasonDeadlineCounter = metrics.NewRegisteredCounter("miner/commit_reason_deadline", nil)
	commitGasCounter            = metrics.NewRegisteredCounter("miner/commit_gas", nil)
)

// prioritizedTransaction represents a single transaction that
// should be processed as the first transaction in the next block.
type prioritizedTransaction struct {
	blockNumber uint64
	tx          *types.Transaction
}

// work represents the active block building task
type work struct {
	deadlineTimer   *time.Timer
	deadlineReached bool
	cccLogger       *ccc.Logger
	vmConfig        vm.Config

	reorgReason error

	// accumulated state
	nextL1MsgIndex uint64
	gasPool        *core.GasPool
	blockSize      common.StorageSize

	header        *types.Header
	state         *state.StateDB
	txs           types.Transactions
	receipts      types.Receipts
	coalescedLogs []*types.Log
}

func (w *work) deadlineCh() <-chan time.Time {
	if w == nil {
		return deadCh
	}
	return w.deadlineTimer.C
}

type reorgTrigger struct {
	block  *types.Block
	reason error
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
	startCh chan struct{}
	exitCh  chan struct{}
	reorgCh chan reorgTrigger

	wg      sync.WaitGroup
	current *work

	mu       sync.RWMutex // The lock used to protect the coinbase and extra fields
	coinbase common.Address
	extra    []byte

	snapshotMu       sync.RWMutex // The lock used to protect the snapshots below
	snapshotBlock    *types.Block
	snapshotReceipts types.Receipts
	snapshotState    *state.StateDB

	// atomic status counters
	running int32 // The indicator whether the consensus engine is running or not.
	newTxs  int32 // New arrival transaction count since last sealing work submitting.

	// noempty is the flag used to control whether the feature of pre-seal empty
	// block is enabled. The default value is false(pre-seal is enabled by default).
	// But in some special scenario the consensus engine will seal blocks instantaneously,
	// in this case this feature will add all empty blocks into canonical chain
	// non-stop and no real transaction will be included.
	noempty uint32

	// External functions
	isLocalBlock func(block *types.Block) bool // Function used to determine whether the specified block is mined by local miner.

	prioritizedTx *prioritizedTransaction

	asyncChecker *ccc.AsyncChecker

	// Test hooks
	beforeTxHook func() // Method to call before processing a transaction.

	errCountdown int
	skipTxHash   common.Hash
}

func newWorker(config *Config, chainConfig *params.ChainConfig, engine consensus.Engine, eth Backend, mux *event.TypeMux, isLocalBlock func(*types.Block) bool, init bool) *worker {
	worker := &worker{
		config:       config,
		chainConfig:  chainConfig,
		engine:       engine,
		eth:          eth,
		mux:          mux,
		chain:        eth.BlockChain(),
		isLocalBlock: isLocalBlock,
		txsCh:        make(chan core.NewTxsEvent, txChanSize),
		chainHeadCh:  make(chan core.ChainHeadEvent, chainHeadChanSize),
		exitCh:       make(chan struct{}),
		startCh:      make(chan struct{}, 1),
		reorgCh:      make(chan reorgTrigger, 1),
	}
	worker.asyncChecker = ccc.NewAsyncChecker(worker.chain, config.CCCMaxWorkers, false).WithOnFailingBlock(worker.onBlockFailingCCC)

	// Subscribe NewTxsEvent for tx pool
	worker.txsSub = eth.TxPool().SubscribeNewTxsEvent(worker.txsCh)

	// Subscribe events for blockchain
	worker.chainHeadSub = eth.BlockChain().SubscribeChainHeadEvent(worker.chainHeadCh)

	// Sanitize account fetch limit.
	if worker.config.MaxAccountsNum == 0 {
		log.Warn("Sanitizing miner account fetch limit", "provided", worker.config.MaxAccountsNum, "updated", math.MaxInt)
		worker.config.MaxAccountsNum = math.MaxInt
	}

	worker.wg.Add(1)
	go worker.mainLoop()

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

// checkHeadRowConsumption will start some initial workers to CCC check block close to the HEAD
func (w *worker) checkHeadRowConsumption() error {
	checkStart := uint64(1)
	numOfBlocksToCheck := uint64(w.config.CCCMaxWorkers + 1)
	currentHeight := w.chain.CurrentHeader().Number.Uint64()
	if currentHeight > numOfBlocksToCheck {
		checkStart = currentHeight - numOfBlocksToCheck
	}

	for curBlockNum := checkStart; curBlockNum <= currentHeight; curBlockNum++ {
		block := w.chain.GetBlockByNumber(curBlockNum)
		// only spawn CCC checkers for blocks with no row consumption data stored in DB
		if rawdb.ReadBlockRowConsumption(w.chain.Database(), block.Hash()) == nil {
			if err := w.asyncChecker.Check(block); err != nil {
				return err
			}
		}
	}

	return nil
}

// mainLoop is a standalone goroutine to regenerate the sealing task based on the received event.
func (w *worker) mainLoop() {
	defer w.wg.Done()
	defer w.asyncChecker.Wait()
	defer w.txsSub.Unsubscribe()
	defer w.chainHeadSub.Unsubscribe()
	defer func() {
		// training wheels on
		// lets not crash the node and allow us some time to inspect
		p := recover()
		if p != nil {
			log.Error("worker mainLoop panic", "panic", p)
		}
	}()

	var err error
	for {
		if _, isRetryable := err.(retryableCommitError); isRetryable {
			if _, err = w.tryCommitNewWork(time.Now(), w.current.header.ParentHash, w.current.reorgReason); err != nil {
				continue
			}
		} else if err != nil {
			log.Error("failed to mine block", "err", err)
			w.current = nil
		}

		// check for reorgs first to lower the chances of trying to handle another
		// event eventhough a reorg is pending (due to Go `select` pseudo-randomly picking a case
		// to execute if multiple of them are ready)
		select {
		case trigger := <-w.reorgCh:
			err = w.handleReorg(&trigger)
			continue
		default:
		}

		select {
		case <-w.startCh:
			if err := w.checkHeadRowConsumption(); err != nil {
				log.Error("failed to start head checkers", "err", err)
				return
			}

			_, err = w.tryCommitNewWork(time.Now(), w.chain.CurrentHeader().Hash(), nil)
		case trigger := <-w.reorgCh:
			err = w.handleReorg(&trigger)
		case chainHead := <-w.chainHeadCh:
			if w.isCanonical(chainHead.Block.Header()) {
				_, err = w.tryCommitNewWork(time.Now(), chainHead.Block.Hash(), nil)
			}
		case <-w.current.deadlineCh():
			w.current.deadlineReached = true
			if len(w.current.txs) > 0 {
				_, err = w.commit(false)
			}
		case ev := <-w.txsCh:
			// Apply transactions to the pending state
			//
			// Note all transactions received may not be continuous with transactions
			// already included in the current mining block. These transactions will
			// be automatically eliminated.
			if w.current != nil {
				shouldCommit, _ := w.processTxnSlice(ev.Txs)
				if shouldCommit || w.current.deadlineReached {
					_, err = w.commit(false)
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
		}
	}
}

// updateSnapshot updates pending snapshot block and state.
// Note this function assumes the current variable is thread safe.
func (w *worker) updateSnapshot() {
	w.snapshotMu.Lock()
	defer w.snapshotMu.Unlock()

	w.snapshotBlock = types.NewBlock(
		w.current.header,
		w.current.txs,
		nil,
		w.current.receipts,
		trie.NewStackTrie(nil),
	)
	w.snapshotReceipts = copyReceipts(w.current.receipts)
	w.snapshotState = w.current.state.Copy()
}

func (w *worker) collectPendingL1Messages(startIndex uint64) []types.L1MessageTx {
	maxCount := w.chainConfig.Scroll.L1Config.NumL1MessagesPerBlock
	return rawdb.ReadL1MessagesFrom(w.eth.ChainDb(), startIndex, maxCount)
}

// newWork
func (w *worker) newWork(now time.Time, parentHash common.Hash, reorgReason error) error {
	parent := w.chain.GetBlockByHash(parentHash)
	header := &types.Header{
		ParentHash: parent.Hash(),
		Number:     new(big.Int).Add(parent.Number(), common.Big1),
		GasLimit:   core.CalcGasLimit(parent.GasLimit(), w.config.GasCeil),
		Extra:      w.extra,
		Time:       uint64(now.Unix()),
	}

	parentState, err := w.chain.StateAt(parent.Root())
	if err != nil {
		return fmt.Errorf("failed to fetch parent state: %w", err)
	}

	// Set baseFee if we are on an EIP-1559 chain
	if w.chainConfig.IsCurie(header.Number) {
		parentL1BaseFee := fees.GetL1BaseFee(parentState)
		header.BaseFee = misc.CalcBaseFee(w.chainConfig, parent.Header(), parentL1BaseFee)
	}
	// Only set the coinbase if our consensus engine is running (avoid spurious block rewards)
	if w.isRunning() {
		if w.coinbase == (common.Address{}) {
			return errors.New("refusing to mine without etherbase")
		}
		header.Coinbase = w.coinbase
	}

	prepareStart := time.Now()
	if err := w.engine.Prepare(w.chain, header); err != nil {
		return fmt.Errorf("failed to prepare header for mining: %w", err)
	}
	prepareTimer.UpdateSince(prepareStart)

	var nextL1MsgIndex uint64
	if dbVal := rawdb.ReadFirstQueueIndexNotInL2Block(w.eth.ChainDb(), header.ParentHash); dbVal != nil {
		nextL1MsgIndex = *dbVal
	}

	vmConfig := *w.chain.GetVMConfig()
	cccLogger := ccc.NewLogger()
	vmConfig.Debug = true
	vmConfig.Tracer = cccLogger

	deadline := time.Unix(int64(header.Time), 0)
	if w.chainConfig.Clique != nil && w.chainConfig.Clique.RelaxedPeriod {
		// clique with relaxed period uses time.Now() as the header.Time, calculate the deadline
		deadline = time.Unix(int64(header.Time+w.chainConfig.Clique.Period), 0)
	}

	w.current = &work{
		deadlineTimer:  time.NewTimer(time.Until(deadline)),
		cccLogger:      cccLogger,
		vmConfig:       vmConfig,
		header:         header,
		state:          parentState,
		txs:            types.Transactions{},
		receipts:       types.Receipts{},
		coalescedLogs:  []*types.Log{},
		gasPool:        new(core.GasPool).AddGas(header.GasLimit),
		nextL1MsgIndex: nextL1MsgIndex,
		reorgReason:    reorgReason,
	}
	return nil
}

// tryCommitNewWork
func (w *worker) tryCommitNewWork(now time.Time, parent common.Hash, reorgReason error) (common.Hash, error) {
	err := w.newWork(now, parent, reorgReason)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed creating new work: %w", err)
	}

	shouldCommit, err := w.handleForks()
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed handling forks: %w", err)
	}

	// check if we are reorging
	reorging := w.chain.GetBlockByNumber(w.current.header.Number.Uint64()) != nil
	if !shouldCommit && reorging {
		shouldCommit, err = w.processReorgedTxns(w.current.reorgReason)
	}
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed handling reorged txns: %w", err)
	}

	if !shouldCommit {
		shouldCommit, err = w.processTxPool()
	}
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed processing tx pool: %w", err)
	}

	if shouldCommit {
		// if reorging, force committing even if we are not "running"
		// this can happen when sequencer is instructed to shutdown while handling a reorg
		// we should make sure reorg is not interrupted
		if blockHash, err := w.commit(reorging); err != nil {
			return common.Hash{}, fmt.Errorf("failed committing new work: %w", err)
		} else {
			return blockHash, nil
		}
	}
	return common.Hash{}, nil
}

// handleForks
func (w *worker) handleForks() (bool, error) {
	if w.chainConfig.CurieBlock != nil && w.chainConfig.CurieBlock.Cmp(w.current.header.Number) == 0 {
		misc.ApplyCurieHardFork(w.current.state)
		return true, nil
	}
	return false, nil
}

// processTxPool
func (w *worker) processTxPool() (bool, error) {
	tidyPendingStart := time.Now()
	// Fill the block with all available pending transactions.
	pending := w.eth.TxPool().PendingWithMax(false, w.config.MaxAccountsNum)

	// Allow txpool to be reorged as we build current block
	w.eth.TxPool().ResumeReorgs()

	// Split the pending transactions into locals and remotes
	localTxs, remoteTxs := make(map[common.Address]types.Transactions), pending
	for _, account := range w.eth.TxPool().Locals() {
		if txs := remoteTxs[account]; len(txs) > 0 {
			delete(remoteTxs, account)
			localTxs[account] = txs
		}
	}
	collectL2Timer.UpdateSince(tidyPendingStart)

	// fetch l1Txs
	var l1Messages []types.L1MessageTx
	if w.chainConfig.Scroll.ShouldIncludeL1Messages() {
		common.WithTimer(collectL1MsgsTimer, func() {
			l1Messages = w.collectPendingL1Messages(w.current.nextL1MsgIndex)
		})
	}

	// Short circuit if there is no available pending transactions.
	// But if we disable empty precommit already, ignore it. Since
	// empty block is necessary to keep the liveness of the network.
	if len(localTxs) == 0 && len(remoteTxs) == 0 && len(l1Messages) == 0 && atomic.LoadUint32(&w.noempty) == 0 {
		return false, nil
	}

	if w.chainConfig.Scroll.ShouldIncludeL1Messages() && len(l1Messages) > 0 {
		log.Trace("Processing L1 messages for inclusion", "count", len(l1Messages))
		txs, err := types.NewL1MessagesByQueueIndex(l1Messages)
		if err != nil {
			return false, fmt.Errorf("failed to create L1 message set: %w", err)
		}

		if shouldCommit, err := w.processTxns(txs); err != nil {
			return false, fmt.Errorf("failed to include l1 msgs: %w", err)
		} else if shouldCommit {
			return true, nil
		}
	}

	signer := types.MakeSigner(w.chainConfig, w.current.header.Number)
	if w.prioritizedTx != nil && w.current.header.Number.Uint64() > w.prioritizedTx.blockNumber {
		w.prioritizedTx = nil
	}
	if w.prioritizedTx != nil {
		from, _ := types.Sender(signer, w.prioritizedTx.tx) // error already checked before
		txList := map[common.Address]types.Transactions{from: []*types.Transaction{w.prioritizedTx.tx}}
		txs := types.NewTransactionsByPriceAndNonce(signer, txList, w.current.header.BaseFee)

		if shouldCommit, err := w.processTxns(txs); err != nil {
			return false, fmt.Errorf("failed to include prioritized tx: %w", err)
		} else if shouldCommit {
			return true, nil
		}
	}

	if len(localTxs) > 0 {
		txs := types.NewTransactionsByPriceAndNonce(signer, localTxs, w.current.header.BaseFee)
		if shouldCommit, err := w.processTxns(txs); err != nil {
			return false, fmt.Errorf("failed to include locals: %w", err)
		} else if shouldCommit {
			return true, nil
		}
	}

	if len(remoteTxs) > 0 {
		txs := types.NewTransactionsByPriceAndNonce(signer, remoteTxs, w.current.header.BaseFee)
		if shouldCommit, err := w.processTxns(txs); err != nil {
			return false, fmt.Errorf("failed to include remotes: %w", err)
		} else if shouldCommit {
			return true, nil
		}
	}

	return false, nil
}

// processTxnSlice
func (w *worker) processTxnSlice(txns types.Transactions) (bool, error) {
	txsMap := make(map[common.Address]types.Transactions)
	signer := types.MakeSigner(w.chainConfig, w.current.header.Number)
	for _, tx := range txns {
		acc, _ := types.Sender(signer, tx)
		txsMap[acc] = append(txsMap[acc], tx)
	}
	txset := types.NewTransactionsByPriceAndNonce(signer, txsMap, w.current.header.BaseFee)
	return w.processTxns(txset)
}

// processReorgedTxns
func (w *worker) processReorgedTxns(reason error) (bool, error) {
	reorgedBlock := w.chain.GetBlockByNumber(w.current.header.Number.Uint64())
	commitGasCounter.Dec(int64(reorgedBlock.GasUsed()))
	reorgedTxns := reorgedBlock.Transactions()
	var errorWithTxnIdx *ccc.ErrorWithTxnIdx
	if len(reorgedTxns) > 0 && errors.As(reason, &errorWithTxnIdx) {
		if errorWithTxnIdx.ShouldSkip {
			w.skipTransaction(reorgedTxns[errorWithTxnIdx.TxIdx], reason)
		}

		// if errorWithTxnIdx.TxIdx is 0, we will end up creating an empty block.
		// This is necessary to make sure that same height can not fail CCC check multiple times.
		// Each reorg forces a block to be appended to the chain. If we let the same block to trigger
		// multiple reorgs, we can't guarantee an upper bound on reorg depth anymore. We can revisit this
		// when we can handle reorgs on sidechains that we are building to replace the canonical chain.
		reorgedTxns = reorgedTxns[:errorWithTxnIdx.TxIdx]
	}

	w.processTxnSlice(reorgedTxns)
	return true, nil
}

// processTxns
func (w *worker) processTxns(txs types.OrderedTransactionSet) (bool, error) {
	for {
		tx := txs.Peek()
		if tx == nil {
			break
		}

		shouldCommit, err := w.processTxn(tx)
		if shouldCommit {
			return true, nil
		}

		switch {
		case err == nil, errors.Is(err, core.ErrNonceTooLow):
			txs.Shift()
		default:
			w.onTxFailing(w.current.txs.Len(), tx, err)
			if errors.Is(err, ccc.ErrBlockRowConsumptionOverflow) && w.current.txs.Len() > 0 {
				return true, nil
			}

			if tx.IsL1MessageTx() {
				txs.Shift()
			} else {
				txs.Pop()
			}
		}
	}

	return false, nil
}

// processTxn
func (w *worker) processTxn(tx *types.Transaction) (bool, error) {
	if w.beforeTxHook != nil {
		w.beforeTxHook()
	}

	// If we don't have enough gas for any further transactions then we're done
	if w.current.gasPool.Gas() < params.TxGas {
		return true, nil
	}

	// If we have collected enough transactions then we're done
	// Originally we only limit l2txs count, but now strictly limit total txs number.
	if !w.chain.Config().Scroll.IsValidTxCount(w.current.txs.Len() + 1) {
		return true, nil
	}

	if tx.IsL1MessageTx() && tx.AsL1MessageTx().QueueIndex != w.current.nextL1MsgIndex {
		// Continue, we might still be able to include some L2 messages
		return false, ErrUnexpectedL1MessageIndex
	}

	if !tx.IsL1MessageTx() && !w.chain.Config().Scroll.IsValidBlockSize(w.current.blockSize+tx.Size()) {
		// can't fit this txn in this block, silently ignore and continue looking for more txns
		return false, errors.New("tx too big")
	}

	// Start executing the transaction
	w.current.state.SetTxContext(tx.Hash(), w.current.txs.Len())

	// create new snapshot for `core.ApplyTransaction`
	snapState := w.current.state.Snapshot()
	snapGasPool := *w.current.gasPool
	snapGasUsed := w.current.header.GasUsed
	snapCccLogger := w.current.cccLogger.Snapshot()

	w.forceTestErr(tx)
	receipt, err := core.ApplyTransaction(w.chain.Config(), w.chain, nil /* coinbase will default to chainConfig.Scroll.FeeVaultAddress */, w.current.gasPool,
		w.current.state, w.current.header, tx, &w.current.header.GasUsed, w.current.vmConfig)
	if err != nil {
		w.current.state.RevertToSnapshot(snapState)
		*w.current.gasPool = snapGasPool
		w.current.header.GasUsed = snapGasUsed
		*w.current.cccLogger = *snapCccLogger
		return false, err
	}

	// Everything ok, collect the logs and shift in the next transaction from the same account
	w.current.coalescedLogs = append(w.current.coalescedLogs, receipt.Logs...)
	w.current.txs = append(w.current.txs, tx)
	w.current.receipts = append(w.current.receipts, receipt)

	if !tx.IsL1MessageTx() {
		// only consider block size limit for L2 transactions
		w.current.blockSize += tx.Size()
	} else {
		w.current.nextL1MsgIndex = tx.AsL1MessageTx().QueueIndex + 1
	}
	return false, nil
}

// retryableCommitError wraps an error that happened during commit phase and indicates that worker can retry to build a new block
type retryableCommitError struct {
	inner error
}

func (e retryableCommitError) Error() string {
	return e.inner.Error()
}

func (e retryableCommitError) Unwrap() error {
	return e.inner
}

// commit runs any post-transaction state modifications, assembles the final block
// and commits new work if consensus engine is running.
func (w *worker) commit(force bool) (common.Hash, error) {
	sealDelay := time.Duration(0)
	defer func(t0 time.Time) {
		l2CommitTimer.Update(time.Since(t0) - sealDelay)
	}(time.Now())

	w.updateSnapshot()
	if !w.isRunning() && !force {
		return common.Hash{}, nil
	}

	block, err := w.engine.FinalizeAndAssemble(w.chain, w.current.header, w.current.state,
		w.current.txs, nil, w.current.receipts)
	if err != nil {
		return common.Hash{}, err
	}

	sealHash := w.engine.SealHash(block.Header())
	log.Info("Committing new mining work", "number", block.Number(), "sealhash", sealHash,
		"txs", w.current.txs.Len(),
		"gas", block.GasUsed(), "fees", totalFees(block, w.current.receipts))

	resultCh, stopCh := make(chan *types.Block), make(chan struct{})
	if err := w.engine.Seal(w.chain, block, resultCh, stopCh); err != nil {
		return common.Hash{}, err
	}
	// Clique.Seal() will only wait for a second before giving up on us. So make sure there is nothing computational heavy
	// or a call that blocks between the call to Seal and the line below. Seal might introduce some delay, so we keep track of
	// that artificially added delay and subtract it from overall runtime of commit().
	sealStart := time.Now()
	block = <-resultCh
	sealDelay = time.Since(sealStart)
	if block == nil {
		return common.Hash{}, errors.New("missed seal response from consensus engine")
	}

	// verify the generated block with local consensus engine to make sure everything is as expected
	if err = w.engine.VerifyHeader(w.chain, block.Header(), true); err != nil {
		return common.Hash{}, retryableCommitError{inner: err}
	}

	blockHash := block.Hash()

	for i, receipt := range w.current.receipts {
		// add block location fields
		receipt.BlockHash = blockHash
		receipt.BlockNumber = block.Number()
		receipt.TransactionIndex = uint(i)

		for _, log := range receipt.Logs {
			log.BlockHash = blockHash
		}
	}

	for _, log := range w.current.coalescedLogs {
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
			"nextL1MsgIndex", w.current.nextL1MsgIndex,
		)
		rawdb.WriteFirstQueueIndexNotInL2Block(w.eth.ChainDb(), blockHash, w.current.nextL1MsgIndex)
	} else {
		log.Trace(
			"Worker WriteFirstQueueIndexNotInL2Block: not overwriting existing index",
			"number", block.Number(),
			"hash", blockHash.String(),
			"index", *index,
			"nextL1MsgIndex", w.current.nextL1MsgIndex,
		)
	}

	// A new block event will trigger a reorg in the txpool, pause reorgs to defer this until we fetch txns for next block.
	// We may end up trying to process txns that we already included in the previous block, but they will all fail the nonce check
	w.eth.TxPool().PauseReorgs()

	// Commit block and state to database.
	_, err = w.chain.WriteBlockWithState(block, w.current.receipts, w.current.coalescedLogs, w.current.state, true)
	if err != nil {
		return common.Hash{}, err
	}

	log.Info("Successfully sealed new block", "number", block.Number(), "sealhash", sealHash, "hash", blockHash)

	// Broadcast the block and announce chain insertion event
	w.mux.Post(core.NewMinedBlockEvent{Block: block})

	checkStart := time.Now()
	if err = w.asyncChecker.Check(block); err != nil {
		log.Error("failed to launch CCC background task", "err", err)
	}
	cccStallTimer.UpdateSince(checkStart)

	commitGasCounter.Inc(int64(block.GasUsed()))
	if w.current.deadlineReached {
		commitReasonDeadlineCounter.Inc(1)
	} else {
		commitReasonCCCCounter.Inc(1)
	}
	w.current = nil
	return block.Hash(), nil
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

func (w *worker) onTxFailing(txIndex int, tx *types.Transaction, err error) {
	if !w.isRunning() {
		return
	}

	if errors.Is(err, ccc.ErrBlockRowConsumptionOverflow) {
		if txIndex > 0 {
			if !tx.IsL1MessageTx() {
				// prioritize overflowing L2 message as the first txn next block
				// no need to prioritize L1 messages, they are fetched in order
				// and processed first in every block anyways
				w.prioritizedTx = &prioritizedTransaction{
					blockNumber: w.current.header.Number.Uint64() + 1,
					tx:          tx,
				}
			}
			return
		}

		// first txn overflowed the circuit, skip
		w.skipTransaction(tx, err)
	} else if tx.IsL1MessageTx() {
		if errors.Is(err, ErrUnexpectedL1MessageIndex) {
			log.Warn(
				"Unexpected L1 message queue index in worker", "got", tx.AsL1MessageTx().QueueIndex,
			)
			return
		} else if txIndex > 0 {
			// If this block already contains some L1 messages try again in the next block.
			return
		}

		queueIndex := tx.AsL1MessageTx().QueueIndex
		log.Warn("Skipping L1 message", "queueIndex", queueIndex, "tx", tx.Hash().String(), "block",
			w.current.header.Number, "reason", err)
		rawdb.WriteSkippedTransaction(w.eth.ChainDb(), tx, nil, err.Error(),
			w.current.header.Number.Uint64(), nil)
		w.current.nextL1MsgIndex = queueIndex + 1
		l1SkippedCounter.Inc(1)
	} else if errors.Is(err, core.ErrInsufficientFunds) {
		log.Trace("Skipping tx with insufficient funds", "tx", tx.Hash().String())
		w.eth.TxPool().RemoveTx(tx.Hash(), true)
	}
}

// skipTransaction
func (w *worker) skipTransaction(tx *types.Transaction, err error) {
	log.Info("Circuit capacity limit reached for a single tx", "isL1Message", tx.IsL1MessageTx(), "tx", tx.Hash().String())
	rawdb.WriteSkippedTransaction(w.eth.ChainDb(), tx, nil, err.Error(),
		w.current.header.Number.Uint64(), nil)
	if tx.IsL1MessageTx() {
		w.current.nextL1MsgIndex = tx.AsL1MessageTx().QueueIndex + 1
		l1SkippedCounter.Inc(1)
	} else {
		if w.prioritizedTx != nil && w.prioritizedTx.tx.Hash() == tx.Hash() {
			w.prioritizedTx = nil
		}

		w.eth.TxPool().RemoveTx(tx.Hash(), true)
		l2SkippedCounter.Inc(1)
	}
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

func (w *worker) forceTestErr(tx *types.Transaction) {
	if w.skipTxHash == tx.Hash() {
		w.current.cccLogger.ForceError()
	}

	w.errCountdown--
	if w.errCountdown == 0 {
		w.current.cccLogger.ForceError()
	}
}

// scheduleCCCError schedules an CCC error with a countdown, only used in tests.
func (w *worker) scheduleCCCError(countdown int) {
	w.errCountdown = countdown
}

// skip forces a txn to be skipped by worker
func (w *worker) skip(txHash common.Hash) {
	w.skipTxHash = txHash
}

// onBlockFailingCCC is called when block produced by worker fails CCC
func (w *worker) onBlockFailingCCC(failingBlock *types.Block, err error) {
	log.Warn("block failed CCC", "hash", failingBlock.Hash().Hex(), "number", failingBlock.NumberU64(), "err", err)
	w.reorgCh <- reorgTrigger{
		block:  failingBlock,
		reason: err,
	}
}

// handleReorg reorgs all blocks following the trigger block
func (w *worker) handleReorg(trigger *reorgTrigger) error {
	parentHash := trigger.block.ParentHash()
	reorgReason := trigger.reason

	for {
		if !w.isCanonical(trigger.block.Header()) {
			// trigger block is no longer part of the canonical chain, we are done
			return nil
		}

		newBlockHash, err := w.tryCommitNewWork(time.Now(), parentHash, reorgReason)
		if err != nil {
			return err
		}

		// we created replacement blocks for all existing blocks in canonical chain, but not quite ready to commit the new HEAD
		if newBlockHash == (common.Hash{}) {
			// force committing the new canonical head to trigger a reorg in blockchain
			// otherwise we might ignore CCC errors from the new side chain since it is not canonical yet
			newBlockHash, err = w.commit(true)
			if err != nil {
				return err
			}
		}

		parentHash = newBlockHash
		reorgReason = nil // clear reorg reason after trigger block gets reorged
	}
}

func (w *worker) isCanonical(header *types.Header) bool {
	return w.chain.GetBlockByNumber(header.Number.Uint64()).Hash() == header.Hash()
}
