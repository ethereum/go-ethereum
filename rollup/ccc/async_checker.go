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

	// tests
	blockNumberToFail uint64
	txnIdxToFail      uint64
}

type ErrorWithTxnIdx struct {
	TxIdx      uint
	err        error
	ShouldSkip bool
	AccRc      *types.RowConsumption
}

func (e *ErrorWithTxnIdx) Error() string {
	return fmt.Sprintf("txn at index %d failed with %s (rc = %s)", e.TxIdx, e.err, fmt.Sprint(e.AccRc))
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
	if c.bc.Config().IsEuclid(block.Time()) {
		// Euclid blocks use MPT and CCC doesn't support them
		return nil
	}

	if block.NumberU64() > c.currentHead.Number.Uint64()+1 {
		log.Warn("non continuous chain observed in AsyncChecker", "prev", c.currentHead, "got", block.Header())
	}

	if block.ParentHash() != c.currentHead.Hash() {
		// seems like there is a fork happening, a block from the canonical chain must have failed CCC check
		// assume the incoming block is the new tip in the fork
		c.forkCtx, c.forkCtxCancelFunc = context.WithCancel(context.Background())
	}

	c.currentHead = block.Header()
	checker := <-c.freeCheckers
	// all blocks in the same fork share the same context to allow terminating them all at once if needed
	ctx, ctxCancelFunc := c.forkCtx, c.forkCtxCancelFunc
	c.workers.Go(func() stream.Callback {
		taskCb := c.checkerTask(block, checker, ctx, ctxCancelFunc)
		return func() {
			taskCb()
			c.freeCheckers <- checker
		}
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

	if c.blockNumberToFail == block.NumberU64() {
		err = &ErrorWithTxnIdx{
			TxIdx: uint(c.txnIdxToFail),
			err:   err,
		}
		c.blockNumberToFail = 0
		return failingCallback
	}

	statedb, err := c.bc.StateAt(parent.Root())
	if err != nil {
		return failingCallback
	}

	header := block.Header()
	ccc.Reset()

	accRc := new(types.RowConsumption)
	for txIdx, tx := range block.Transactions() {
		if !isForkStillActive(forkCtx) {
			return noopCb
		}

		var curRc *types.RowConsumption
		curRc, err = c.checkTx(parent, header, statedb, tx, ccc)
		if err != nil {
			err = &ErrorWithTxnIdx{
				TxIdx: uint(txIdx),
				err:   err,
				// if the txn is the first in block or the additional resource utilization caused
				// by this txn alone is enough to overflow the circuit, skip
				ShouldSkip: txIdx == 0 || curRc == nil || curRc.Difference(*accRc).IsOverflown(),
				AccRc:      curRc,
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

func (c *AsyncChecker) checkTx(parent *types.Block, header *types.Header, state *state.StateDB, tx *types.Transaction, ccc *Checker) (*types.RowConsumption, error) {
	trace, err := tracing.NewTracerWrapper().CreateTraceEnvAndGetBlockTrace(c.bc.Config(), c.bc, c.bc.Engine(), c.bc.Database(),
		state, parent, types.NewBlockWithHeader(header).WithBody([]*types.Transaction{tx}, nil), true)
	if err != nil {
		return nil, err
	}

	return ccc.ApplyTransaction(trace)
}

// ScheduleError forces a block to error on a given transaction index
func (c *AsyncChecker) ScheduleError(blockNumber uint64, txnIndx uint64) {
	c.blockNumberToFail = blockNumber
	c.txnIdxToFail = txnIndx
}
