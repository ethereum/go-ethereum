package pipeline

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"time"
	"unsafe"

	"github.com/scroll-tech/go-ethereum/core"
	"github.com/scroll-tech/go-ethereum/core/state"
	"github.com/scroll-tech/go-ethereum/core/txpool"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/core/vm"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/metrics"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/scroll-tech/go-ethereum/rollup/ccc"
	"github.com/scroll-tech/go-ethereum/rollup/tracing"
)

type ErrorWithTrace struct {
	Trace *types.BlockTrace
	err   error
}

func (e *ErrorWithTrace) Error() string {
	return e.err.Error()
}

func (e *ErrorWithTrace) Unwrap() error {
	return e.err
}

var (
	ErrPipelineDone             = errors.New("pipeline is done")
	ErrUnexpectedL1MessageIndex = errors.New("unexpected L1 message index")

	lifetimeTimer = func() metrics.Timer {
		t := metrics.NewCustomTimer(metrics.NewHistogram(metrics.NewExpDecaySample(128, 0.015)), metrics.NewMeter())
		metrics.DefaultRegistry.Register("pipeline/lifetime", t)
		return t
	}()
	applyTimer       = metrics.NewRegisteredTimer("pipeline/apply", nil)
	applyIdleTimer   = metrics.NewRegisteredTimer("pipeline/apply_idle", nil)
	applyStallTimer  = metrics.NewRegisteredTimer("pipeline/apply_stall", nil)
	encodeTimer      = metrics.NewRegisteredTimer("pipeline/encode", nil)
	encodeIdleTimer  = metrics.NewRegisteredTimer("pipeline/encode_idle", nil)
	encodeStallTimer = metrics.NewRegisteredTimer("pipeline/encode_stall", nil)
	cccTimer         = metrics.NewRegisteredTimer("pipeline/ccc", nil)
	cccIdleTimer     = metrics.NewRegisteredTimer("pipeline/ccc_idle", nil)
)

type Pipeline struct {
	chain     *core.BlockChain
	vmConfig  vm.Config
	parent    *types.Block
	start     time.Time
	wg        sync.WaitGroup
	ctx       context.Context
	cancelCtx context.CancelFunc

	// accumulators
	ccc            *ccc.Checker
	Header         types.Header
	state          *state.StateDB
	nextL1MsgIndex uint64
	blockSize      uint64
	txs            types.Transactions
	coalescedLogs  []*types.Log
	receipts       types.Receipts
	gasPool        *core.GasPool

	// com channels
	txnQueue         chan *txpool.LazyTransaction
	applyStageRespCh <-chan error
	ResultCh         <-chan *Result

	// Test hooks
	beforeTxHook func() // Method to call before processing a transaction.
}

func NewPipeline(
	chain *core.BlockChain,
	vmConfig vm.Config,
	state *state.StateDB,

	header *types.Header,
	nextL1MsgIndex uint64,
	ccc *ccc.Checker,
) *Pipeline {
	// make sure we are not sharing a tracer with the caller and not in debug mode
	vmConfig.Tracer = nil

	ctx, cancel := context.WithCancel(context.Background())
	return &Pipeline{
		chain:          chain,
		vmConfig:       vmConfig,
		parent:         chain.GetBlock(header.ParentHash, header.Number.Uint64()-1),
		nextL1MsgIndex: nextL1MsgIndex,
		Header:         *header,
		ccc:            ccc,
		state:          state,
		gasPool:        new(core.GasPool).AddGas(header.GasLimit),
		ctx:            ctx,
		cancelCtx:      cancel,
	}
}

func (p *Pipeline) WithBeforeTxHook(beforeTxHook func()) *Pipeline {
	p.beforeTxHook = beforeTxHook
	return p
}

func (p *Pipeline) Start(deadline time.Time) error {
	p.start = time.Now()
	p.txnQueue = make(chan *txpool.LazyTransaction)
	applyStageRespCh, applyToEncodeCh, err := p.traceAndApplyStage(p.txnQueue)
	if err != nil {
		log.Error("Failed starting traceAndApplyStage", "err", err)
		return err
	}
	p.applyStageRespCh = applyStageRespCh
	encodeToCccCh := p.encodeStage(applyToEncodeCh)
	p.ResultCh = p.cccStage(encodeToCccCh, deadline)
	return nil
}

// Stop forces pipeline to stop its operation and return whatever progress it has so far
func (p *Pipeline) Stop() {
	if p.txnQueue != nil {
		close(p.txnQueue)
		p.txnQueue = nil
	}
}

// orderedTransactionSet represents a set of transactions and some ordering on top of this set.
type orderedTransactionSet interface {
	// Peek returns the next transaction.
	Peek() *txpool.LazyTransaction

	// Shift removes the next transaction.
	Shift()

	// Pop removes all transactions from the current account.
	Pop()
}

func (p *Pipeline) TryPushTxns(txs orderedTransactionSet, onFailingTxn func(txnIndex int, tx *types.Transaction, err error) bool) *Result {
	for {
		ltx := txs.Peek()
		if ltx == nil {
			break
		}

		result, err := p.TryPushTxn(ltx)
		if result != nil {
			return result
		}

		// TODO: return tx via `TryPushTxn` so that we don't need to resolve it here again
		tx := ltx.Resolve()
		if tx == nil {
			txs.Shift()
			continue
		}

		switch {
		case err == nil, errors.Is(err, core.ErrNonceTooLow):
			txs.Shift()
		default:
			if errors.Is(err, ErrPipelineDone) || onFailingTxn(p.txs.Len(), tx, err) {
				p.Stop()
				return nil
			}

			if tx.IsL1MessageTx() {
				txs.Shift()
			} else {
				txs.Pop()
			}
		}
	}

	return nil
}

func (p *Pipeline) TryPushTxn(tx *txpool.LazyTransaction) (*Result, error) {
	if p.txnQueue == nil {
		return nil, ErrPipelineDone
	}

	select {
	case p.txnQueue <- tx:
	case <-p.ctx.Done():
		return nil, ErrPipelineDone
	case res := <-p.ResultCh:
		return res, nil
	}

	select {
	case err, valid := <-p.applyStageRespCh:
		if !valid {
			return nil, ErrPipelineDone
		}
		return nil, err
	case res := <-p.ResultCh:
		return res, nil
	}
}

// Release releases all resources related to the pipeline
func (p *Pipeline) Release() {
	p.cancelCtx()
	p.wg.Wait()
}

type BlockCandidate struct {
	LastTrace      *types.BlockTrace
	RustTrace      unsafe.Pointer
	NextL1MsgIndex uint64

	// accumulated state
	Header        *types.Header
	State         *state.StateDB
	Txs           types.Transactions
	Receipts      types.Receipts
	CoalescedLogs []*types.Log
}

// sendCancellable tries to send msg to resCh but allows send operation to be cancelled
// by closing cancelCh. Returns true if cancelled.
func sendCancellable[T any, C comparable](resCh chan T, msg T, cancelCh <-chan C) bool {
	var zeroC C

	select {
	case resCh <- msg:
		return false
	case cancelSignal := <-cancelCh:
		if cancelSignal != zeroC {
			panic("shouldn't have happened")
		}
		return true
	}
}

func (p *Pipeline) traceAndApplyStage(txsIn <-chan *txpool.LazyTransaction) (<-chan error, <-chan *BlockCandidate, error) {
	p.state.StartPrefetcher("miner")
	downstreamCh := make(chan *BlockCandidate, p.downstreamChCapacity())
	resCh := make(chan error)
	p.wg.Add(1)
	go func() {
		defer func() {
			close(downstreamCh)
			close(resCh)
			p.state.StopPrefetcher()
			p.wg.Done()
		}()

		var ltx *txpool.LazyTransaction
		for {
			idleStart := time.Now()
			select {
			case ltx = <-txsIn:
				if ltx == nil {
					return
				}
			case <-p.ctx.Done():
				return
			}
			applyIdleTimer.UpdateSince(idleStart)

			applyStart := time.Now()

			// If we don't have enough gas for any further transactions then we're done
			if p.gasPool.Gas() < params.TxGas {
				return
			}

			// If we have collected enough transactions then we're done
			// Originally we only limit l2txs count, but now strictly limit total txs number.
			if !p.chain.Config().Scroll.IsValidTxCount(p.txs.Len() + 1) {
				return
			}

			if p.gasPool.Gas() < ltx.Gas {
				// we don't have enough space for the next transaction, skip the account and continue looking for more txns
				sendCancellable(resCh, core.ErrGasLimitReached, p.ctx.Done())
				continue
			}

			// TODO: blob gas check

			tx := ltx.Resolve()
			if tx == nil {
				log.Trace("Ignoring evicted transaction", "hash", ltx.Hash)
				// can't resolve the tx, silently ignore and continue looking for more txns
				sendCancellable(resCh, errors.New("cannot resolve evicted tx"), p.ctx.Done())
				continue
			}

			if tx.IsL1MessageTx() && tx.AsL1MessageTx().QueueIndex != p.nextL1MsgIndex {
				// Continue, we might still be able to include some L2 messages
				sendCancellable(resCh, ErrUnexpectedL1MessageIndex, p.ctx.Done())
				continue
			}

			if !tx.IsL1MessageTx() && !p.chain.Config().Scroll.IsValidBlockSize(p.blockSize+tx.Size()) {
				// can't fit this txn in this block, silently ignore and continue looking for more txns
				sendCancellable(resCh, nil, p.ctx.Done())
				continue
			}

			// Start executing the transaction
			p.state.SetTxContext(tx.Hash(), p.txs.Len())
			receipt, trace, err := p.traceAndApply(tx)

			if p.txs.Len() == 0 && tx.IsL1MessageTx() && err != nil {
				// L1 message errored as the first txn, skip
				p.nextL1MsgIndex = tx.AsL1MessageTx().QueueIndex + 1
			}

			if err == nil {
				// Everything ok, collect the logs and shift in the next transaction from the same account
				p.coalescedLogs = append(p.coalescedLogs, receipt.Logs...)
				p.txs = append(p.txs, tx)
				p.receipts = append(p.receipts, receipt)

				if !tx.IsL1MessageTx() {
					// only consider block size limit for L2 transactions
					p.blockSize += tx.Size()
				} else {
					p.nextL1MsgIndex = tx.AsL1MessageTx().QueueIndex + 1
				}

				stallStart := time.Now()
				if sendCancellable(downstreamCh, &BlockCandidate{
					NextL1MsgIndex: p.nextL1MsgIndex,
					LastTrace:      trace,

					Header:        types.CopyHeader(&p.Header),
					State:         p.state.Copy(),
					Txs:           p.txs,
					Receipts:      p.receipts,
					CoalescedLogs: p.coalescedLogs,
				}, p.ctx.Done()) {
					// next stage terminated and caller terminated us as well
					return
				}
				applyStallTimer.UpdateSince(stallStart)
			}
			if err != nil && trace != nil {
				err = &ErrorWithTrace{
					Trace: trace,
					err:   err,
				}
			}
			applyTimer.UpdateSince(applyStart)
			sendCancellable(resCh, err, p.ctx.Done())
		}
	}()
	return resCh, downstreamCh, nil
}

type Result struct {
	OverflowingTx    *types.Transaction
	OverflowingTrace *types.BlockTrace
	CCCErr           error

	Rows       *types.RowConsumption
	FinalBlock *BlockCandidate
}

func (p *Pipeline) encodeStage(traces <-chan *BlockCandidate) <-chan *BlockCandidate {
	downstreamCh := make(chan *BlockCandidate, p.downstreamChCapacity())
	p.wg.Add(1)

	go func() {
		defer func() {
			close(downstreamCh)
			p.wg.Done()
		}()
		buffer := new(bytes.Buffer)
		for {
			idleStart := time.Now()
			select {
			case trace := <-traces:
				if trace == nil {
					return
				}
				encodeIdleTimer.UpdateSince(idleStart)

				encodeStart := time.Now()
				if p.ccc != nil {
					trace.RustTrace = ccc.MakeRustTrace(trace.LastTrace, buffer)
					if trace.RustTrace == nil {
						log.Error("making rust trace", "txHash", trace.LastTrace.Transactions[0].TxHash)
						// ignore the error here, CCC stage will catch it and treat it as a CCC error
					}
				}
				encodeTimer.UpdateSince(encodeStart)

				stallStart := time.Now()
				if sendCancellable(downstreamCh, trace, p.ctx.Done()) && trace.RustTrace != nil {
					// failed to send the trace downstream, free it here.
					ccc.FreeRustTrace(trace.RustTrace)
				}
				encodeStallTimer.UpdateSince(stallStart)
			case <-p.ctx.Done():
				return
			}

		}
	}()
	return downstreamCh
}

func (p *Pipeline) cccStage(candidates <-chan *BlockCandidate, deadline time.Time) <-chan *Result {
	if p.ccc != nil {
		p.ccc.Reset()
	}
	resultCh := make(chan *Result, 1)
	var lastCandidate *BlockCandidate
	var lastAccRows *types.RowConsumption
	var deadlineReached bool

	p.wg.Add(1)
	go func() {
		deadlineTimer := time.NewTimer(time.Until(deadline))
		defer func() {
			close(resultCh)
			deadlineTimer.Stop()
			lifetimeTimer.UpdateSince(p.start)
			// consume candidates and free all rust traces
			for candidate := range candidates {
				if candidate == nil {
					break
				}
				if candidate.RustTrace != nil {
					ccc.FreeRustTrace(candidate.RustTrace)
				}
			}
			p.wg.Done()
		}()
		for {
			idleStart := time.Now()
			select {
			case <-p.ctx.Done():
				return
			case <-deadlineTimer.C:
				cccIdleTimer.UpdateSince(idleStart)
				// note: currently we don't allow empty blocks, but if we ever do; make sure to CCC check it first
				if lastCandidate != nil {
					resultCh <- &Result{
						Rows:       lastAccRows,
						FinalBlock: lastCandidate,
					}
					return
				}
				deadlineReached = true
			case candidate := <-candidates:
				cccIdleTimer.UpdateSince(idleStart)
				cccStart := time.Now()
				var accRows *types.RowConsumption
				var err error
				if candidate != nil && p.ccc != nil {
					if candidate.RustTrace != nil {
						accRows, err = p.ccc.ApplyTransactionRustTrace(candidate.RustTrace)
					} else {
						err = errors.New("no rust trace")
					}
					lastTxn := candidate.Txs[candidate.Txs.Len()-1]
					cccTimer.UpdateSince(cccStart)
					if err != nil {
						resultCh <- &Result{
							OverflowingTx:    lastTxn,
							OverflowingTrace: candidate.LastTrace,
							CCCErr:           err,
							Rows:             lastAccRows,
							FinalBlock:       lastCandidate,
						}
						return
					}

					lastCandidate = candidate
					lastAccRows = accRows
				} else if candidate != nil && p.ccc == nil {
					lastCandidate = candidate
				}

				// immediately close the block if deadline reached or apply stage is done
				if candidate == nil || deadlineReached {
					resultCh <- &Result{
						Rows:       lastAccRows,
						FinalBlock: lastCandidate,
					}
					return
				}
			}
		}
	}()
	return resultCh
}

func (p *Pipeline) traceAndApply(tx *types.Transaction) (*types.Receipt, *types.BlockTrace, error) {
	var trace *types.BlockTrace
	var err error

	if p.beforeTxHook != nil {
		p.beforeTxHook()
	}

	// do gas limit check up-front and do not run CCC if it fails
	if p.gasPool.Gas() < tx.Gas() {
		return nil, nil, core.ErrGasLimitReached
	}

	if p.ccc != nil {
		// don't commit the state during tracing for circuit capacity checker, otherwise we cannot revert.
		// and even if we don't commit the state, the `refund` value will still be correct, as explained in `CommitTransaction`
		finaliseStateAfterApply := false
		snap := p.state.Snapshot()

		// 1. we have to check circuit capacity before `core.ApplyTransaction`,
		// because if the tx can be successfully executed but circuit capacity overflows, it will be inconvenient to revert.
		// 2. even if we don't commit to the state during the tracing (which means `clearJournalAndRefund` is not called during the tracing),
		// the `refund` value will still be correct, because:
		// 2.1 when starting handling the first tx, `state.refund` is 0 by default,
		// 2.2 after tracing, the state is either committed in `core.ApplyTransaction`, or reverted, so the `state.refund` can be cleared,
		// 2.3 when starting handling the following txs, `state.refund` comes as 0
		trace, err = tracing.NewTracerWrapper().CreateTraceEnvAndGetBlockTrace(p.chain.Config(), p.chain, p.chain.Engine(), p.chain.Database(),
			p.state, p.parent.Header(), types.NewBlockWithHeader(&p.Header).WithBody([]*types.Transaction{tx}, nil), finaliseStateAfterApply)
		// `w.current.traceEnv.State` & `w.current.state` share a same pointer to the state, so only need to revert `w.current.state`
		// revert to snapshot for calling `core.ApplyMessage` again, (both `traceEnv.GetBlockTrace` & `core.ApplyTransaction` will call `core.ApplyMessage`)
		p.state.RevertToSnapshot(snap)
		if err != nil {
			return nil, nil, err
		}
	}

	// create new snapshot for `core.ApplyTransaction`
	snap := p.state.Snapshot()

	var receipt *types.Receipt
	receipt, err = core.ApplyTransaction(p.chain.Config(), p.chain, nil /* coinbase will default to chainConfig.Scroll.FeeVaultAddress */, p.gasPool,
		p.state, &p.Header, tx, &p.Header.GasUsed, p.vmConfig)
	if err != nil {
		p.state.RevertToSnapshot(snap)
		return nil, trace, err
	}
	return receipt, trace, nil
}

// downstreamChCapacity returns the channel capacity that should be used for downstream channels.
// It aims to minimize stalls caused by different computational costs of different transactions
func (p *Pipeline) downstreamChCapacity() int {
	cap := 1
	if p.chain.Config().Scroll.MaxTxPerBlock != nil {
		cap = *p.chain.Config().Scroll.MaxTxPerBlock
	}
	return cap
}
