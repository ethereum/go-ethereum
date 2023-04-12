package miner

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"os"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	cmath "github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/common/tracing"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/bor"
	"github.com/ethereum/go-ethereum/consensus/clique"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	// testCode is the testing contract binary code which will initialises some
	// variables in constructor
	testCode = "0x60806040527fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0060005534801561003457600080fd5b5060fc806100436000396000f3fe6080604052348015600f57600080fd5b506004361060325760003560e01c80630c4dae8814603757806398a213cf146053575b600080fd5b603d607e565b6040518082815260200191505060405180910390f35b607c60048036036020811015606757600080fd5b81019080803590602001909291905050506084565b005b60005481565b806000819055507fe9e44f9f7da8c559de847a3232b57364adc0354f15a2cd8dc636d54396f9587a6000546040518082815260200191505060405180910390a15056fea265627a7a723058208ae31d9424f2d0bc2a3da1a5dd659db2d71ec322a17db8f87e19e209e3a1ff4a64736f6c634300050a0032"

	// testGas is the gas required for contract deployment.
	testGas = 144109
)

func init() {
	signer := types.LatestSigner(params.TestChainConfig)

	tx1 := types.MustSignNewTx(testBankKey, signer, &types.AccessListTx{
		ChainID:  params.TestChainConfig.ChainID,
		Nonce:    0,
		To:       &testUserAddress,
		Value:    big.NewInt(1000),
		Gas:      params.TxGas,
		GasPrice: big.NewInt(params.InitialBaseFee),
	})

	pendingTxs = append(pendingTxs, tx1)

	tx2 := types.MustSignNewTx(testBankKey, signer, &types.LegacyTx{
		Nonce:    1,
		To:       &testUserAddress,
		Value:    big.NewInt(1000),
		Gas:      params.TxGas,
		GasPrice: big.NewInt(params.InitialBaseFee),
	})

	newTxs = append(newTxs, tx2)
}

// testWorkerBackend implements worker.Backend interfaces and wraps all information needed during the testing.
type testWorkerBackend struct {
	DB         ethdb.Database
	txPool     *core.TxPool
	chain      *core.BlockChain
	Genesis    *core.Genesis
	uncleBlock *types.Block
}

func newTestWorkerBackend(t TensingObject, chainConfig *params.ChainConfig, engine consensus.Engine, db ethdb.Database, n int) *testWorkerBackend {
	var gspec = core.Genesis{
		Config:   chainConfig,
		Alloc:    core.GenesisAlloc{TestBankAddress: {Balance: testBankFunds}},
		GasLimit: 30_000_000,
	}

	switch e := engine.(type) {
	case *bor.Bor:
		gspec.ExtraData = make([]byte, 32+common.AddressLength+crypto.SignatureLength)
		copy(gspec.ExtraData[32:32+common.AddressLength], TestBankAddress.Bytes())
		e.Authorize(TestBankAddress, func(account accounts.Account, s string, data []byte) ([]byte, error) {
			return crypto.Sign(crypto.Keccak256(data), testBankKey)
		})
	case *clique.Clique:
		gspec.ExtraData = make([]byte, 32+common.AddressLength+crypto.SignatureLength)
		copy(gspec.ExtraData[32:32+common.AddressLength], TestBankAddress.Bytes())
		e.Authorize(TestBankAddress, func(account accounts.Account, s string, data []byte) ([]byte, error) {
			return crypto.Sign(crypto.Keccak256(data), testBankKey)
		})
	case *ethash.Ethash:
	default:
		t.Fatalf("unexpected consensus engine type: %T", engine)
	}

	genesis := gspec.MustCommit(db)

	chain, _ := core.NewBlockChain(db, &core.CacheConfig{TrieDirtyDisabled: true}, gspec.Config, engine, vm.Config{}, nil, nil, nil)
	txpool := core.NewTxPool(testTxPoolConfig, chainConfig, chain)

	// Generate a small n-block chain and an uncle block for it
	if n > 0 {
		blocks, _ := core.GenerateChain(chainConfig, genesis, engine, db, n, func(i int, gen *core.BlockGen) {
			gen.SetCoinbase(TestBankAddress)
		})
		if _, err := chain.InsertChain(blocks); err != nil {
			t.Fatalf("failed to insert origin chain: %v", err)
		}
	}

	parent := genesis
	if n > 0 {
		parent = chain.GetBlockByHash(chain.CurrentBlock().ParentHash())
	}

	blocks, _ := core.GenerateChain(chainConfig, parent, engine, db, 1, func(i int, gen *core.BlockGen) {
		gen.SetCoinbase(testUserAddress)
	})

	return &testWorkerBackend{
		DB:         db,
		chain:      chain,
		txPool:     txpool,
		Genesis:    &gspec,
		uncleBlock: blocks[0],
	}
}

func (b *testWorkerBackend) BlockChain() *core.BlockChain { return b.chain }
func (b *testWorkerBackend) TxPool() *core.TxPool         { return b.txPool }
func (b *testWorkerBackend) StateAtBlock(block *types.Block, reexec uint64, base *state.StateDB, checkLive bool, preferDisk bool) (statedb *state.StateDB, err error) {
	return nil, errors.New("not supported")
}

func (b *testWorkerBackend) newRandomUncle() (*types.Block, error) {
	var parent *types.Block

	cur := b.chain.CurrentBlock()

	if cur.NumberU64() == 0 {
		parent = b.chain.Genesis()
	} else {
		parent = b.chain.GetBlockByHash(b.chain.CurrentBlock().ParentHash())
	}

	var err error

	blocks, _ := core.GenerateChain(b.chain.Config(), parent, b.chain.Engine(), b.DB, 1, func(i int, gen *core.BlockGen) {
		var addr = make([]byte, common.AddressLength)

		_, err = rand.Read(addr)
		if err != nil {
			return
		}

		gen.SetCoinbase(common.BytesToAddress(addr))
	})

	return blocks[0], err
}

func (b *testWorkerBackend) newRandomTx(creation bool) *types.Transaction {
	var tx *types.Transaction

	gasPrice := big.NewInt(10 * params.InitialBaseFee)

	if creation {
		tx, _ = types.SignTx(types.NewContractCreation(b.txPool.Nonce(TestBankAddress), big.NewInt(0), testGas, gasPrice, common.FromHex(testCode)), types.HomesteadSigner{}, testBankKey)
	} else {
		tx, _ = types.SignTx(types.NewTransaction(b.txPool.Nonce(TestBankAddress), testUserAddress, big.NewInt(1000), params.TxGas, gasPrice, nil), types.HomesteadSigner{}, testBankKey)
	}

	return tx
}

func NewTestWorker(t TensingObject, chainConfig *params.ChainConfig, engine consensus.Engine, db ethdb.Database, blocks int, noempty uint32, delay uint) (*worker, *testWorkerBackend, func()) {
	backend := newTestWorkerBackend(t, chainConfig, engine, db, blocks)
	backend.txPool.AddLocals(pendingTxs)

	var w *worker

	if delay != 0 {
		//nolint:staticcheck
		w = newWorkerWithDelay(testConfig, chainConfig, engine, backend, new(event.TypeMux), nil, false, delay)
	} else {
		//nolint:staticcheck
		w = newWorker(testConfig, chainConfig, engine, backend, new(event.TypeMux), nil, false)
	}

	w.setEtherbase(TestBankAddress)

	// enable empty blocks
	w.noempty = noempty

	return w, backend, w.close
}

//nolint:staticcheck
func newWorkerWithDelay(config *Config, chainConfig *params.ChainConfig, engine consensus.Engine, eth Backend, mux *event.TypeMux, isLocalBlock func(header *types.Header) bool, init bool, delay uint) *worker {
	worker := &worker{
		config:             config,
		chainConfig:        chainConfig,
		engine:             engine,
		eth:                eth,
		mux:                mux,
		chain:              eth.BlockChain(),
		isLocalBlock:       isLocalBlock,
		localUncles:        make(map[common.Hash]*types.Block),
		remoteUncles:       make(map[common.Hash]*types.Block),
		unconfirmed:        newUnconfirmedBlocks(eth.BlockChain(), sealingLogAtDepth),
		pendingTasks:       make(map[common.Hash]*task),
		txsCh:              make(chan core.NewTxsEvent, txChanSize),
		chainHeadCh:        make(chan core.ChainHeadEvent, chainHeadChanSize),
		chainSideCh:        make(chan core.ChainSideEvent, chainSideChanSize),
		newWorkCh:          make(chan *newWorkReq),
		getWorkCh:          make(chan *getWorkReq),
		taskCh:             make(chan *task),
		resultCh:           make(chan *types.Block, resultQueueSize),
		exitCh:             make(chan struct{}),
		startCh:            make(chan struct{}, 1),
		resubmitIntervalCh: make(chan time.Duration),
		resubmitAdjustCh:   make(chan *intervalAdjust, resubmitAdjustChanSize),
		noempty:            1,
	}
	worker.profileCount = new(int32)
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

	ctx := tracing.WithTracer(context.Background(), otel.GetTracerProvider().Tracer("MinerWorker"))

	worker.wg.Add(4)

	go worker.mainLoopWithDelay(ctx, delay)
	go worker.newWorkLoop(ctx, recommit)
	go worker.resultLoop()
	go worker.taskLoop()

	// Submit first work to initialize pending state.
	if init {
		worker.startCh <- struct{}{}
	}

	return worker
}

// nolint:gocognit
func (w *worker) mainLoopWithDelay(ctx context.Context, delay uint) {
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
			//nolint:contextcheck
			w.commitWorkWithDelay(req.ctx, req.interrupt, req.noempty, req.timestamp, delay)

		case req := <-w.getWorkCh:
			//nolint:contextcheck
			block, err := w.generateWork(req.ctx, req.params)
			if err != nil {
				req.err = err
				req.result <- nil
			} else {
				req.result <- block
			}

		case ev := <-w.chainSideCh:
			// Short circuit for duplicate side blocks
			if _, exist := w.localUncles[ev.Block.Hash()]; exist {
				continue
			}

			if _, exist := w.remoteUncles[ev.Block.Hash()]; exist {
				continue
			}

			// Add side block to possible uncle block set depending on the author.
			if w.isLocalBlock != nil && w.isLocalBlock(ev.Block.Header()) {
				w.localUncles[ev.Block.Hash()] = ev.Block
			} else {
				w.remoteUncles[ev.Block.Hash()] = ev.Block
			}

			// If our sealing block contains less than 2 uncle blocks,
			// add the new uncle block if valid and regenerate a new
			// sealing block for higher profit.
			if w.isRunning() && w.current != nil && len(w.current.uncles) < 2 {
				start := time.Now()
				if err := w.commitUncle(w.current, ev.Block.Header()); err == nil {
					commitErr := w.commit(ctx, w.current.copy(), nil, true, start)
					if commitErr != nil {
						log.Error("error while committing work for mining", "err", commitErr)
					}
				}
			}

		case <-cleanTicker.C:
			chainHead := w.chain.CurrentBlock()
			for hash, uncle := range w.localUncles {
				if uncle.NumberU64()+staleThreshold <= chainHead.NumberU64() {
					delete(w.localUncles, hash)
				}
			}

			for hash, uncle := range w.remoteUncles {
				if uncle.NumberU64()+staleThreshold <= chainHead.NumberU64() {
					delete(w.remoteUncles, hash)
				}
			}

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

				txs := make(map[common.Address]types.Transactions)

				for _, tx := range ev.Txs {
					acc, _ := types.Sender(w.current.signer, tx)
					txs[acc] = append(txs[acc], tx)
				}

				txset := types.NewTransactionsByPriceAndNonce(w.current.signer, txs, cmath.FromBig(w.current.header.BaseFee))
				tcount := w.current.tcount

				interruptCh, stopFn := getInterruptTimer(ctx, w.current, w.chain.CurrentBlock())
				w.commitTransactionsWithDelay(w.current, txset, nil, interruptCh, delay)

				// Only update the snapshot if any new transactions were added
				// to the pending block
				if tcount != w.current.tcount {
					w.updateSnapshot(w.current)
				}

				stopFn()
			} else {
				// Special case, if the consensus engine is 0 period clique(dev mode),
				// submit sealing work here since all empty submission will be rejected
				// by clique. Of course the advance sealing(empty submission) is disabled.
				if w.chainConfig.Clique != nil && w.chainConfig.Clique.Period == 0 {
					w.commitWork(ctx, nil, true, time.Now().Unix())
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

// nolint:gocognit
func (w *worker) commitTransactionsWithDelay(env *environment, txs *types.TransactionsByPriceAndNonce, interrupt *int32, interruptCh chan struct{}, delay uint) bool {
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
		// case of interrupting by timeout
		select {
		case <-interruptCh:
			commitInterruptCounter.Inc(1)
			break mainloop
		default:
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
		// If we don't have enough gas for any further transactions then we're done
		if env.gasPool.Gas() < params.TxGas {
			log.Trace("Not enough gas for further transactions", "have", env.gasPool, "want", params.TxGas)
			// nolint:goconst
			breakCause = "Not enough gas for further transactions"
			break
		}
		// Retrieve the next transaction and abort if all done
		tx := txs.Peek()
		if tx == nil {
			// nolint:goconst
			breakCause = "all transactions has been included"
			break
		}
		// Error may be ignored here. The error has already been checked
		// during transaction acceptance is the transaction pool.
		//
		// We use the eip155 signer regardless of the current hf.
		from, _ := types.Sender(env.signer, tx)
		// Check whether the tx is replay protected. If we're not in the EIP155 hf
		// phase, start ignoring the sender until we do.
		if tx.Protected() && !w.chainConfig.IsEIP155(env.header.Number) {
			log.Trace("Ignoring reply protected transaction", "hash", tx.Hash(), "eip155", w.chainConfig.EIP155Block)

			txs.Pop()
			continue
		}
		// Start executing the transaction
		env.state.Prepare(tx.Hash(), env.tcount)

		var start time.Time

		log.OnDebug(func(log.Logging) {
			start = time.Now()
		})

		logs, err := w.commitTransaction(env, tx)
		time.Sleep(time.Duration(delay) * time.Millisecond)

		switch {
		case errors.Is(err, core.ErrGasLimitReached):
			// Pop the current out-of-gas transaction without shifting in the next from the account
			log.Trace("Gas limit exceeded for current block", "sender", from)
			txs.Pop()

		case errors.Is(err, core.ErrNonceTooLow):
			// New head notification data race between the transaction pool and miner, shift
			log.Trace("Skipping transaction with low nonce", "sender", from, "nonce", tx.Nonce())
			txs.Shift()

		case errors.Is(err, core.ErrNonceTooHigh):
			// Reorg notification data race between the transaction pool and miner, skip account =
			log.Trace("Skipping account with hight nonce", "sender", from, "nonce", tx.Nonce())
			txs.Pop()

		case errors.Is(err, nil):
			// Everything ok, collect the logs and shift in the next transaction from the same account
			coalescedLogs = append(coalescedLogs, logs...)
			env.tcount++
			txs.Shift()

			log.OnDebug(func(lg log.Logging) {
				lg("Committed new tx", "tx hash", tx.Hash(), "from", from, "to", tx.To(), "nonce", tx.Nonce(), "gas", tx.Gas(), "gasPrice", tx.GasPrice(), "value", tx.Value(), "time spent", time.Since(start))
			})

		case errors.Is(err, core.ErrTxTypeNotSupported):
			// Pop the unsupported transaction without shifting in the next from the account
			log.Trace("Skipping unsupported transaction type", "sender", from, "type", tx.Type())
			txs.Pop()

		default:
			// Strange error, discard the transaction and get the next in line (note, the
			// nonce-too-high clause will prevent us from executing in vain).
			log.Debug("Transaction failed, account skipped", "hash", tx.Hash(), "err", err)
			txs.Shift()
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
	// Notify resubmit loop to decrease resubmitting interval if current interval is larger
	// than the user-specified one.
	if interrupt != nil {
		w.resubmitAdjustCh <- &intervalAdjust{inc: false}
	}

	return false
}

func (w *worker) commitWorkWithDelay(ctx context.Context, interrupt *int32, noempty bool, timestamp int64, delay uint) {
	start := time.Now()

	var (
		work *environment
		err  error
	)

	tracing.Exec(ctx, "", "worker.prepareWork", func(ctx context.Context, span trace.Span) {
		// Set the coinbase if the worker is running or it's required
		var coinbase common.Address
		if w.isRunning() {
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

	var interruptCh chan struct{}

	stopFn := func() {}
	defer func() {
		stopFn()
	}()

	if !noempty {
		interruptCh, stopFn = getInterruptTimer(ctx, work, w.chain.CurrentBlock())
	}

	ctx, span := tracing.StartSpan(ctx, "commitWork")
	defer tracing.EndSpan(span)

	tracing.SetAttributes(
		span,
		attribute.Int("number", int(work.header.Number.Uint64())),
	)

	// Create an empty block based on temporary copied state for
	// sealing in advance without waiting block execution finished.
	if !noempty && atomic.LoadUint32(&w.noempty) == 0 {
		err = w.commit(ctx, work.copy(), nil, false, start)
		if err != nil {
			return
		}
	}

	// Fill pending transactions from the txpool
	w.fillTransactionsWithDelay(ctx, interrupt, work, interruptCh, delay)

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

// nolint:gocognit
func (w *worker) fillTransactionsWithDelay(ctx context.Context, interrupt *int32, env *environment, interruptCh chan struct{}, delay uint) {
	ctx, span := tracing.StartSpan(ctx, "fillTransactions")
	defer tracing.EndSpan(span)

	// Split the pending transactions into locals and remotes
	// Fill the block with all available pending transactions.

	var (
		localTxsCount  int
		remoteTxsCount int
		localTxs       = make(map[common.Address]types.Transactions)
		remoteTxs      map[common.Address]types.Transactions
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

		pending := w.eth.TxPool().Pending(ctx, true)
		remoteTxs = pending

		postPendingTime := time.Now()

		for _, account := range w.eth.TxPool().Locals() {
			if txs := remoteTxs[account]; len(txs) > 0 {
				delete(remoteTxs, account)
				localTxs[account] = txs
			}
		}

		postLocalsTime := time.Now()

		localTxsCount = len(localTxs)
		remoteTxsCount = len(remoteTxs)

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
		var txs *types.TransactionsByPriceAndNonce

		tracing.Exec(ctx, "", "worker.LocalTransactionsByPriceAndNonce", func(ctx context.Context, span trace.Span) {
			txs = types.NewTransactionsByPriceAndNonce(env.signer, localTxs, cmath.FromBig(env.header.BaseFee))

			tracing.SetAttributes(
				span,
				attribute.Int("len of tx local Heads", txs.GetTxs()),
			)
		})

		tracing.Exec(ctx, "", "worker.LocalCommitTransactions", func(ctx context.Context, span trace.Span) {
			committed = w.commitTransactionsWithDelay(env, txs, interrupt, interruptCh, delay)
		})

		if committed {
			return
		}

		localEnvTCount = env.tcount
	}

	if remoteTxsCount > 0 {
		var txs *types.TransactionsByPriceAndNonce

		tracing.Exec(ctx, "", "worker.RemoteTransactionsByPriceAndNonce", func(ctx context.Context, span trace.Span) {
			txs = types.NewTransactionsByPriceAndNonce(env.signer, remoteTxs, cmath.FromBig(env.header.BaseFee))

			tracing.SetAttributes(
				span,
				attribute.Int("len of tx remote Heads", txs.GetTxs()),
			)
		})

		tracing.Exec(ctx, "", "worker.RemoteCommitTransactions", func(ctx context.Context, span trace.Span) {
			committed = w.commitTransactionsWithDelay(env, txs, interrupt, interruptCh, delay)
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
