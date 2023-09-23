package miner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	// nolint:typecheck

	"github.com/ethereum/go-ethereum/common"
	cmath "github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/common/tracing"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	lru "github.com/hashicorp/golang-lru"
)

// newWorkerWithDelay is newWorker() with extra params to induce artficial delays for tests such as commit-interrupt.
// nolint:staticcheck
func newWorkerWithDelay(config *Config, chainConfig *params.ChainConfig, engine consensus.Engine, eth Backend, mux *event.TypeMux, isLocalBlock func(header *types.Header) bool, init bool, delay uint, opcodeDelay uint) *worker {
	worker := &worker{
		config:              config,
		chainConfig:         chainConfig,
		engine:              engine,
		eth:                 eth,
		mux:                 mux,
		chain:               eth.BlockChain(),
		isLocalBlock:        isLocalBlock,
		pendingTasks:        make(map[common.Hash]*task),
		txsCh:               make(chan core.NewTxsEvent, txChanSize),
		chainHeadCh:         make(chan core.ChainHeadEvent, chainHeadChanSize),
		chainSideCh:         make(chan core.ChainSideEvent, chainSideChanSize),
		newWorkCh:           make(chan *newWorkReq),
		getWorkCh:           make(chan *getWorkReq),
		taskCh:              make(chan *task),
		resultCh:            make(chan *types.Block, resultQueueSize),
		exitCh:              make(chan struct{}),
		startCh:             make(chan struct{}, 1),
		resubmitIntervalCh:  make(chan time.Duration),
		resubmitAdjustCh:    make(chan *intervalAdjust, resubmitAdjustChanSize),
		interruptCommitFlag: config.CommitInterruptFlag,
	}
	worker.noempty.Store(true)
	worker.profileCount = new(int32)
	// Subscribe NewTxsEvent for tx pool
	worker.txsSub = eth.TxPool().SubscribeNewTxsEvent(worker.txsCh)
	// Subscribe events for blockchain
	worker.chainHeadSub = eth.BlockChain().SubscribeChainHeadEvent(worker.chainHeadCh)
	worker.chainSideSub = eth.BlockChain().SubscribeChainSideEvent(worker.chainSideCh)

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

	ctx := tracing.WithTracer(context.Background(), otel.GetTracerProvider().Tracer("MinerWorker"))

	worker.wg.Add(4)

	go worker.mainLoopWithDelay(ctx, delay, opcodeDelay)
	go worker.newWorkLoop(ctx, recommit)
	go worker.resultLoop()
	go worker.taskLoop()

	// Submit first work to initialize pending state.
	if init {
		worker.startCh <- struct{}{}
	}

	return worker
}

// mainLoopWithDelay is mainLoop() with extra params to induce artficial delays for tests such as commit-interrupt.
// nolint:gocognit
func (w *worker) mainLoopWithDelay(ctx context.Context, delay uint, opcodeDelay uint) {
	defer w.wg.Done()
	defer w.txsSub.Unsubscribe()
	defer w.chainHeadSub.Unsubscribe()
	defer w.chainSideSub.Unsubscribe()
	defer func() {
		if w.current != nil {
			w.current.discard()
		}
	}()

	cleanTicker := time.NewTicker(time.Second * 10)
	defer cleanTicker.Stop()

	for {
		select {
		case req := <-w.newWorkCh:
			i := req.interrupt.Load()
			//nolint:contextcheck
			w.commitWorkWithDelay(req.ctx, &i, req.noempty, req.timestamp, delay, opcodeDelay)

		case req := <-w.getWorkCh:
			//nolint:contextcheck
			block, _, err := w.generateWork(req.ctx, req.params)
			if err != nil {
				req.result <- nil
			} else {
				payload := newPayloadResult{
					err:   nil,
					block: block,
					fees:  block.BaseFee(),
				}
				req.result <- &payload
			}

		case ev := <-w.txsCh:
			// Apply transactions to the pending state if we're not sealing
			//
			// Note all transactions received may not be continuous with transactions
			// already included in the current sealing block. These transactions will
			// be automatically eliminated.
			if !w.IsRunning() && w.current != nil {
				// If block is already full, abort
				if gp := w.current.gasPool; gp != nil && gp.Gas() < params.TxGas {
					continue
				}
				txs := make(map[common.Address][]*txpool.LazyTransaction, len(ev.Txs))
				for _, tx := range ev.Txs {
					acc, _ := types.Sender(w.current.signer, tx)
					txs[acc] = append(txs[acc], &txpool.LazyTransaction{
						Hash:      tx.Hash(),
						Tx:        &txpool.Transaction{Tx: tx},
						Time:      tx.Time(),
						GasFeeCap: tx.GasFeeCap(),
						GasTipCap: tx.GasTipCap(),
					})
				}
				txset := newTransactionsByPriceAndNonce(w.current.signer, txs, w.current.header.BaseFee)
				tcount := w.current.tcount
				w.commitTransactions(w.current, txset, nil, context.Background())

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
		case <-w.chainSideSub.Err():
			return
		}
	}
}

// commitWorkWithDelay is commitWork() with extra params to induce artficial delays for tests such as commit-interrupt.
func (w *worker) commitWorkWithDelay(ctx context.Context, interrupt *int32, noempty bool, timestamp int64, delay uint, opcodeDelay uint) {
	start := time.Now()

	var (
		work *environment
		err  error
	)

	tracing.Exec(ctx, "", "worker.prepareWork", func(ctx context.Context, span trace.Span) {
		// Set the coinbase if the worker is running or it's required
		var coinbase common.Address
		if w.IsRunning() {
			if w.coinbase == (common.Address{}) {
				log.Error("Refusing to mine without etherbase")
				return
			}

			coinbase = w.coinbase // Use the preset address as the fee recipient
		}

		work, err = w.prepareWork(&generateParams{
			timestamp: uint64(timestamp),
			coinbase:  coinbase,
		})
	})

	if err != nil {
		return
	}

	//nolint:contextcheck
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
		// nolint : staticcheck
		interruptCtx = context.WithValue(interruptCtx, vm.InterruptCtxDelayKey, delay)
		// nolint : staticcheck
		interruptCtx = context.WithValue(interruptCtx, vm.InterruptCtxOpcodeDelayKey, opcodeDelay)
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
		err = w.commit(ctx, work.copy(), nil, false, start)
		if err != nil {
			return
		}
	}

	// Fill pending transactions from the txpool
	w.fillTransactionsWithDelay(ctx, interrupt, work, interruptCtx)

	err = w.commit(ctx, work.copy(), w.fullTaskHook, true, start)
	if err != nil {
		return
	}

	// Swap out the old work with the new one, terminating any leftover
	// prefetcher processes in the mean time and starting a new one.
	if w.current != nil {
		w.current.discard()
	}

	w.current = work
}

// fillTransactionsWithDelay is fillTransactions() with extra params to induce artficial delays for tests such as commit-interrupt.
// nolint:gocognit
func (w *worker) fillTransactionsWithDelay(ctx context.Context, interrupt *int32, env *environment, interruptCtx context.Context) {
	ctx, span := tracing.StartSpan(ctx, "fillTransactions")
	defer tracing.EndSpan(span)

	// Split the pending transactions into locals and remotes
	// Fill the block with all available pending transactions.

	var (
		localTxsCount  int
		remoteTxsCount int
	)

	pending := w.eth.TxPool().Pending(true)
	localTxs, remoteTxs := make(map[common.Address][]*txpool.LazyTransaction), pending

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
		committed       bool
	)

	if localTxsCount > 0 {
		var txs *transactionsByPriceAndNonce

		tracing.Exec(ctx, "", "worker.LocalTransactionsByPriceAndNonce", func(ctx context.Context, span trace.Span) {
			var baseFee *uint256.Int
			if env.header.BaseFee != nil {
				baseFee = cmath.FromBig(env.header.BaseFee)
			}

			txs := newTransactionsByPriceAndNonce(env.signer, localTxs, baseFee.ToBig())

			tracing.SetAttributes(
				span,
				attribute.Int("len of tx local Heads", txs.GetTxs()),
			)
		})

		tracing.Exec(ctx, "", "worker.LocalCommitTransactions", func(ctx context.Context, span trace.Span) {
			committed = w.commitTransactionsWithDelay(env, txs, interrupt, interruptCtx)
		})

		if committed {
			return
		}

		localEnvTCount = env.tcount
	}

	if remoteTxsCount > 0 {
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
			committed = w.commitTransactionsWithDelay(env, txs, interrupt, interruptCtx)
		})

		if committed {
			return
		}

		remoteEnvTCount = env.tcount
	}

	tracing.SetAttributes(
		span,
		attribute.Int("len of final local txs ", localEnvTCount),
		attribute.Int("len of final remote txs", remoteEnvTCount),
	)
}

// commitTransactionsWithDelay is commitTransactions() with extra params to induce artficial delays for tests such as commit-interrupt.
// nolint:gocognit, unparam
func (w *worker) commitTransactionsWithDelay(env *environment, txs *transactionsByPriceAndNonce, interrupt *int32, interruptCtx context.Context) bool {
	gasLimit := env.header.GasLimit
	if env.gasPool == nil {
		env.gasPool = new(core.GasPool).AddGas(gasLimit)
	}

	var coalescedLogs []*types.Log

	initialGasLimit := env.gasPool.Gas()
	initialTxs := txs.GetTxs()

	var breakCause string

	defer func() {
		log.OnDebug(func(lg log.Logging) {
			lg("commitTransactions-stats",
				"initialTxsCount", initialTxs,
				"initialGasLimit", initialGasLimit,
				"resultTxsCount", txs.GetTxs(),
				"resultGapPool", env.gasPool.Gas(),
				"exitCause", breakCause)
		})
	}()

mainloop:
	for {
		if interruptCtx != nil {
			// case of interrupting by timeout
			select {
			case <-interruptCtx.Done():
				txCommitInterruptCounter.Inc(1)
				log.Warn("Tx Level Interrupt")
				break mainloop
			default:
			}
		}

		// In the following three cases, we will interrupt the execution of the transaction.
		// (1) new head block event arrival, the interrupt signal is 1
		// (2) worker start or restart, the interrupt signal is 1
		// (3) worker recreate the sealing block with any newly arrived transactions, the interrupt signal is 2.
		// For the first two cases, the semi-finished work will be discarded.
		// For the third case, the semi-finished work will be submitted to the consensus engine.
		if interrupt != nil && atomic.LoadInt32(interrupt) != commitInterruptNone {
			// Notify resubmit loop to increase resubmitting interval due to too frequent commits.
			if atomic.LoadInt32(interrupt) == commitInterruptResubmit {
				ratio := float64(gasLimit-env.gasPool.Gas()) / float64(gasLimit)
				if ratio < 0.1 {
					// nolint:goconst
					ratio = 0.1
				}
				w.resubmitAdjustCh <- &intervalAdjust{
					ratio: ratio,
					inc:   true,
				}
			}
			// nolint:goconst
			breakCause = "interrupt"

			return atomic.LoadInt32(interrupt) == commitInterruptNewHead
		}
		// If we don't have enough gas for any further transactions then we're done.
		if env.gasPool.Gas() < params.TxGas {
			breakCause = "Not enough gas for further transactions"
			log.Trace("Not enough gas for further transactions", "have", env.gasPool, "want", params.TxGas)
			break
		}
		// Retrieve the next transaction and abort if all done.
		ltx := txs.Peek()
		if ltx == nil {
			breakCause = "all transactions has been included"
			break
		}
		tx := ltx.Resolve()
		if tx == nil {
			log.Warn("Ignoring evicted transaction")

			txs.Pop()
			continue
		}
		// Error may be ignored here. The error has already been checked
		// during transaction acceptance is the transaction pool.
		from, _ := types.Sender(env.signer, tx.Tx)

		// Check whether the tx is replay protected. If we're not in the EIP155 hf
		// phase, start ignoring the sender until we do.
		if tx.Tx.Protected() && !w.chainConfig.IsEIP155(env.header.Number) {
			log.Trace("Ignoring reply protected transaction", "hash", tx.Tx.Hash(), "eip155", w.chainConfig.EIP155Block)

			txs.Pop()
			continue
		}
		// Start executing the transaction
		env.state.SetTxContext(tx.Tx.Hash(), env.tcount)

		var start time.Time

		log.OnDebug(func(log.Logging) {
			start = time.Now()
		})

		logs, err := w.commitTransaction(env, tx.Tx, interruptCtx)

		if interruptCtx != nil {
			if delay := interruptCtx.Value(vm.InterruptCtxDelayKey); delay != nil {
				// nolint : durationcheck
				time.Sleep(time.Duration(delay.(uint)) * time.Millisecond)
			}
		}

		switch {
		case errors.Is(err, core.ErrNonceTooLow):
			// New head notification data race between the transaction pool and miner, shift
			log.Trace("Skipping transaction with low nonce", "sender", from, "nonce", tx.Tx.Nonce())
			txs.Shift()

		case errors.Is(err, nil):
			// Everything ok, collect the logs and shift in the next transaction from the same account
			coalescedLogs = append(coalescedLogs, logs...)
			env.tcount++

			txs.Shift()

			log.OnDebug(func(lg log.Logging) {
				lg("Committed new tx", "tx hash", tx.Tx.Hash(), "from", from, "to", tx.Tx.To(), "nonce", tx.Tx.Nonce(), "gas", tx.Tx.Gas(), "gasPrice", tx.Tx.GasPrice(), "value", tx.Tx.Value(), "time spent", time.Since(start))
			})

		default:
			// Transaction is regarded as invalid, drop all consecutive transactions from
			// the same sender because of `nonce-too-high` clause.
			log.Debug("Transaction failed, account skipped", "hash", tx.Tx.Hash(), "err", err)
			txs.Pop()
		}
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
	// Notify resubmit loop to decrease resubmitting interval if current interval is larger
	// than the user-specified one.
	if interrupt != nil {
		w.resubmitAdjustCh <- &intervalAdjust{inc: false}
	}

	return false
}
