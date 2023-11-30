package miner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	// nolint:typecheck

	"github.com/ethereum/go-ethereum/common"
	cmath "github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/common/tracing"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/blockstm"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
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
		chain:               eth.BlockChain(),
		mux:                 mux,
		isLocalBlock:        isLocalBlock,
		coinbase:            config.Etherbase,
		extra:               config.ExtraData,
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
	// Subscribe NewTxsEvent for tx pool
	worker.txsSub = eth.TxPool().SubscribeNewTxsEvent(worker.txsCh)
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
	defer func() {
		if w.current != nil {
			w.current.discard()
		}
	}()

	for {
		select {
		case req := <-w.newWorkCh:
			if w.chainConfig.ChainID.Cmp(params.BorMainnetChainConfig.ChainID) == 0 || w.chainConfig.ChainID.Cmp(params.MumbaiChainConfig.ChainID) == 0 {
				if w.eth.PeerCount() > 0 {
					//nolint:contextcheck
					w.commitWorkWithDelay(req.ctx, req.interrupt, req.noempty, req.timestamp, delay, opcodeDelay)
				}
			} else {
				//nolint:contextcheck
				w.commitWorkWithDelay(req.ctx, req.interrupt, req.noempty, req.timestamp, delay, opcodeDelay)
			}

		case req := <-w.getWorkCh:
			block, fees, err := w.generateWork(req.ctx, req.params)
			req.result <- &newPayloadResult{
				err:   err,
				block: block,
				fees:  fees,
			}

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
		}
	}
}

// commitWorkWithDelay is commitWork() with extra params to induce artficial delays for tests such as commit-interrupt.
func (w *worker) commitWorkWithDelay(ctx context.Context, interrupt *atomic.Int32, noempty bool, timestamp int64, delay uint, opcodeDelay uint) {
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
		_ = w.commit(ctx, work.copy(), nil, false, start)
	}
	// Fill pending transactions from the txpool into the block.
	err = w.fillTransactionsWithDelay(ctx, interrupt, work, interruptCtx)

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
	_ = w.commit(ctx, work.copy(), w.fullTaskHook, true, start)

	// Swap out the old work with the new one, terminating any leftover
	// prefetcher processes in the mean time and starting a new one.
	if w.current != nil {
		w.current.discard()
	}

	w.current = work
}

// fillTransactionsWithDelay is fillTransactions() with extra params to induce artficial delays for tests such as commit-interrupt.
// nolint:gocognit
func (w *worker) fillTransactionsWithDelay(ctx context.Context, interrupt *atomic.Int32, env *environment, interruptCtx context.Context) error {
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
			err = w.commitTransactionsWithDelay(env, txs, interrupt, interruptCtx)
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
			err = w.commitTransactionsWithDelay(env, txs, interrupt, interruptCtx)
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

// commitTransactionsWithDelay is commitTransactions() with extra params to induce artficial delays for tests such as commit-interrupt.
// nolint:gocognit, unparam
func (w *worker) commitTransactionsWithDelay(env *environment, txs *transactionsByPriceAndNonce, interrupt *atomic.Int32, interruptCtx context.Context) error {
	gasLimit := env.header.GasLimit
	if env.gasPool == nil {
		env.gasPool = new(core.GasPool).AddGas(gasLimit)
	}

	var coalescedLogs []*types.Log

	var depsMVReadList [][]blockstm.ReadDescriptor

	var depsMVFullWriteList [][]blockstm.WriteDescriptor

	var mvReadMapList []map[blockstm.Key]blockstm.ReadDescriptor

	var deps map[int]map[int]bool

	chDeps := make(chan blockstm.TxDep)

	var count int

	var depsWg sync.WaitGroup

	EnableMVHashMap := false

	// create and add empty mvHashMap in statedb
	if EnableMVHashMap {
		depsMVReadList = [][]blockstm.ReadDescriptor{}

		depsMVFullWriteList = [][]blockstm.WriteDescriptor{}

		mvReadMapList = []map[blockstm.Key]blockstm.ReadDescriptor{}

		deps = map[int]map[int]bool{}

		chDeps = make(chan blockstm.TxDep)

		count = 0

		depsWg.Add(1)

		go func(chDeps chan blockstm.TxDep) {
			for t := range chDeps {
				deps = blockstm.UpdateDeps(deps, t)
			}

			depsWg.Done()
		}(chDeps)
	}

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
			if EnableMVHashMap {
				env.state.AddEmptyMVHashMap()
			}

			// case of interrupting by timeout
			select {
			case <-interruptCtx.Done():
				txCommitInterruptCounter.Inc(1)
				log.Warn("Tx Level Interrupt")
				break mainloop
			default:
			}
		}

		// Check interruption signal and abort building if it's fired.
		if interrupt != nil {
			if signal := interrupt.Load(); signal != commitInterruptNone {
				breakCause = "interrupt"
				return signalToErr(signal)
			}
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

		// not prioritising conditional transaction, yet.
		//nolint:nestif
		if options := tx.Tx.GetOptions(); options != nil {
			if err := env.header.ValidateBlockNumberOptions4337(options.BlockNumberMin, options.BlockNumberMax); err != nil {
				log.Trace("Dropping conditional transaction", "from", from, "hash", tx.Tx.Hash(), "reason", err)
				txs.Pop()

				continue
			}

			if err := env.header.ValidateTimestampOptions4337(options.TimestampMin, options.TimestampMax); err != nil {
				log.Trace("Dropping conditional transaction", "from", from, "hash", tx.Tx.Hash(), "reason", err)
				txs.Pop()

				continue
			}

			if err := env.state.ValidateKnownAccounts(options.KnownAccounts); err != nil {
				log.Trace("Dropping conditional transaction", "from", from, "hash", tx.Tx.Hash(), "reason", err)
				txs.Pop()

				continue
			}
		}

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

			if EnableMVHashMap {
				depsMVReadList = append(depsMVReadList, env.state.MVReadList())
				depsMVFullWriteList = append(depsMVFullWriteList, env.state.MVFullWriteList())
				mvReadMapList = append(mvReadMapList, env.state.MVReadMap())

				temp := blockstm.TxDep{
					Index:         env.tcount - 1,
					ReadList:      depsMVReadList[count],
					FullWriteList: depsMVFullWriteList,
				}

				chDeps <- temp
				count++
			}

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

		if EnableMVHashMap {
			env.state.ClearReadMap()
			env.state.ClearWriteMap()
		}
	}

	// nolint:nestif
	if EnableMVHashMap && w.IsRunning() {
		close(chDeps)
		depsWg.Wait()

		var blockExtraData types.BlockExtraData

		tempVanity := env.header.Extra[:types.ExtraVanityLength]
		tempSeal := env.header.Extra[len(env.header.Extra)-types.ExtraSealLength:]

		if len(mvReadMapList) > 0 {
			tempDeps := make([][]uint64, len(mvReadMapList))

			for j := range deps[0] {
				tempDeps[0] = append(tempDeps[0], uint64(j))
			}

			delayFlag := true

			for i := 1; i <= len(mvReadMapList)-1; i++ {
				reads := mvReadMapList[i-1]

				_, ok1 := reads[blockstm.NewSubpathKey(env.coinbase, state.BalancePath)]
				_, ok2 := reads[blockstm.NewSubpathKey(common.HexToAddress(w.chainConfig.Bor.CalculateBurntContract(env.header.Number.Uint64())), state.BalancePath)]

				if ok1 || ok2 {
					delayFlag = false
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
