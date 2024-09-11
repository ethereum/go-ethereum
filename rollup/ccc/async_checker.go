package ccc

import (
	"context"
	"fmt"
	"time"

	"github.com/sourcegraph/conc/stream"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core"
	"github.com/scroll-tech/go-ethereum/core/rawdb"
	"github.com/scroll-tech/go-ethereum/core/state"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/core/vm"
	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/metrics"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/scroll-tech/go-ethereum/rollup/tracing"
)

var (
	failCounter        = metrics.NewRegisteredCounter("ccc/async/fail", nil)
	checkTimer         = metrics.NewRegisteredTimer("ccc/async/check", nil)
	activeWorkersGauge = metrics.NewRegisteredGauge("ccc/async/active_workers", nil)
)

type Blockchain interface {
	Database() ethdb.Database
	GetBlock(hash common.Hash, number uint64) *types.Block
	StateAt(root common.Hash) (*state.StateDB, error)
	Config() *params.ChainConfig
	GetVMConfig() *vm.Config
	CurrentHeader() *types.Header
	core.ChainContext
}

// AsyncChecker allows a caller to spawn CCC verification tasks
type AsyncChecker struct {
	bc             Blockchain
	onFailingBlock func(*types.Block, error)

	workers      *stream.Stream
	freeCheckers chan *Checker

	// local state to keep track of the chain progressing and terminate tasks early if needed
	currentHead       *types.Header
	forkCtx           context.Context
	forkCtxCancelFunc context.CancelFunc
}

type ErrorWithTxnIdx struct {
	txIdx      uint
	err        error
	shouldSkip bool
}

func (e *ErrorWithTxnIdx) Error() string {
	return fmt.Sprintf("txn at index %d failed with %s", e.txIdx, e.err)
}

func (e *ErrorWithTxnIdx) Unwrap() error {
	return e.err
}

func NewAsyncChecker(bc Blockchain, numWorkers int, lightMode bool) *AsyncChecker {
	forkCtx, forkCtxCancelFunc := context.WithCancel(context.Background())
	return &AsyncChecker{
		bc: bc,
		freeCheckers: func(count int) chan *Checker {
			checkers := make(chan *Checker, count)
			for i := 0; i < count; i++ {
				checkers <- NewChecker(lightMode)
			}
			return checkers
		}(numWorkers),
		workers:           stream.New().WithMaxGoroutines(numWorkers),
		currentHead:       bc.CurrentHeader(),
		forkCtx:           forkCtx,
		forkCtxCancelFunc: forkCtxCancelFunc,
	}
}

func (c *AsyncChecker) WithOnFailingBlock(onFailingBlock func(*types.Block, error)) *AsyncChecker {
	c.onFailingBlock = onFailingBlock
	return c
}

func (c *AsyncChecker) Wait() {
	c.workers.Wait()
}

// Check spawns an async CCC verification task.
func (c *AsyncChecker) Check(block *types.Block) error {
	if block.NumberU64() > c.currentHead.Number.Uint64()+1 {
		log.Error("non continuous chain observed in AsyncChecker", "prev", c.currentHead, "got", block.Header())
	} else if block.ParentHash() != c.currentHead.Hash() {
		// seems like there is a fork happening, a block from the canonical chain must have failed CCC check
		// assume the incoming block is the new tip in the fork
		c.forkCtx, c.forkCtxCancelFunc = context.WithCancel(context.Background())
	}

	c.currentHead = block.Header()
	checker := <-c.freeCheckers
	// all blocks in the same fork share the same context to allow terminating them all at once if needed
	ctx, ctxCancelFunc := c.forkCtx, c.forkCtxCancelFunc
	c.workers.Go(func() stream.Callback {
		return c.checkerTask(block, checker, ctx, ctxCancelFunc)
	})
	return nil
}

func isForkStillActive(forkCtx context.Context) bool {
	select {
	case <-forkCtx.Done():
		// an ancestor block of this block failed CCC check, this fork is not active anymore
		return false
	default:
	}
	return true
}

func (c *AsyncChecker) checkerTask(block *types.Block, ccc *Checker, forkCtx context.Context, forkCtxCancelFunc context.CancelFunc) stream.Callback {
	activeWorkersGauge.Inc(1)
	checkStart := time.Now()
	defer func() {
		checkTimer.UpdateSince(checkStart)
		c.freeCheckers <- ccc
		activeWorkersGauge.Dec(1)
	}()

	noopCb := func() {}
	parent := c.bc.GetBlock(block.ParentHash(), block.NumberU64()-1)
	if parent == nil {
		return noopCb // not part of a chain
	}

	var err error
	failingCallback := func() {
		failCounter.Inc(1)
		if isForkStillActive(forkCtx) {
			// we failed the CCC check, cancel the context to signal all tasks preceding this one to terminate early
			forkCtxCancelFunc()
			if c.onFailingBlock != nil {
				c.onFailingBlock(block, err)
			}
		}
	}

	statedb, err := c.bc.StateAt(parent.Root())
	if err != nil {
		return failingCallback
	}

	header := block.Header()
	header.GasUsed = 0
	gasPool := new(core.GasPool).AddGas(header.GasLimit)
	ccc.Reset()

	var accRc *types.RowConsumption
	for txIdx, tx := range block.Transactions() {
		if !isForkStillActive(forkCtx) {
			return noopCb
		}

		var curRc *types.RowConsumption
		curRc, err = c.checkTxAndApply(parent, header, statedb, gasPool, tx, ccc)
		if err != nil {
			err = &ErrorWithTxnIdx{
				txIdx: uint(txIdx),
				err:   err,
				// if the txn is the first in block or the additional resource utilization caused
				// by this txn alone is enough to overflow the circuit, skip
				shouldSkip: txIdx == 0 || curRc.Difference(*accRc).IsOverflown(),
			}
			return failingCallback
		}
		accRc = curRc
	}

	return func() {
		if isForkStillActive(forkCtx) {
			// all good, write the row consumption
			log.Debug("CCC passed", "blockhash", block.Hash(), "height", block.NumberU64())
			rawdb.WriteBlockRowConsumption(c.bc.Database(), block.Hash(), accRc)
		}
	}
}

func (c *AsyncChecker) checkTxAndApply(parent *types.Block, header *types.Header, state *state.StateDB, gasPool *core.GasPool, tx *types.Transaction, ccc *Checker) (*types.RowConsumption, error) {
	// don't commit the state during tracing for circuit capacity checker, otherwise we cannot revert.
	// and even if we don't commit the state, the `refund` value will still be correct, as explained in `CommitTransaction`
	commitStateAfterApply := false
	snap := state.Snapshot()

	// 1. we have to check circuit capacity before `core.ApplyTransaction`,
	// because if the tx can be successfully executed but circuit capacity overflows, it will be inconvenient to revert.
	// 2. even if we don't commit to the state during the tracing (which means `clearJournalAndRefund` is not called during the tracing),
	// the `refund` value will still be correct, because:
	// 2.1 when starting handling the first tx, `state.refund` is 0 by default,
	// 2.2 after tracing, the state is either committed in `core.ApplyTransaction`, or reverted, so the `state.refund` can be cleared,
	// 2.3 when starting handling the following txs, `state.refund` comes as 0
	trace, err := tracing.NewTracerWrapper().CreateTraceEnvAndGetBlockTrace(c.bc.Config(), c.bc, c.bc.Engine(), c.bc.Database(),
		state, parent.Header(), types.NewBlockWithHeader(header).WithBody([]*types.Transaction{tx}, nil), commitStateAfterApply)
	// `w.current.traceEnv.State` & `w.current.state` share a same pointer to the state, so only need to revert `w.current.state`
	// revert to snapshot for calling `core.ApplyMessage` again, (both `traceEnv.GetBlockTrace` & `core.ApplyTransaction` will call `core.ApplyMessage`)
	state.RevertToSnapshot(snap)
	if err != nil {
		return nil, err
	}

	rc, err := ccc.ApplyTransaction(trace)
	if err != nil {
		return rc, err
	}

	_, err = core.ApplyTransaction(c.bc.Config(), c.bc, nil /* coinbase will default to chainConfig.Scroll.FeeVaultAddress */, gasPool,
		state, header, tx, &header.GasUsed, *c.bc.GetVMConfig())
	if err != nil {
		return nil, err
	}
	return rc, nil
}
