package pipeline

import (
	"errors"
	"time"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core"
	"github.com/scroll-tech/go-ethereum/core/state"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/core/vm"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/metrics"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/scroll-tech/go-ethereum/rollup/circuitcapacitychecker"
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
	ErrApplyStageDone           = errors.New("apply stage is done")
	ErrUnexpectedL1MessageIndex = errors.New("unexpected L1 message index")

	lifetimeTimer   = metrics.NewRegisteredTimer("pipeline/lifetime", nil)
	applyTimer      = metrics.NewRegisteredTimer("pipeline/apply", nil)
	applyIdleTimer  = metrics.NewRegisteredTimer("pipeline/apply_idle", nil)
	applyStallTimer = metrics.NewRegisteredTimer("pipeline/apply_stall", nil)
	cccTimer        = metrics.NewRegisteredTimer("pipeline/ccc", nil)
	cccIdleTimer    = metrics.NewRegisteredTimer("pipeline/ccc_idle", nil)
)

type Pipeline struct {
	chain    *core.BlockChain
	vmConfig *vm.Config
	parent   *types.Block
	start    time.Time

	// accumalators
	ccc            *circuitcapacitychecker.CircuitCapacityChecker
	Header         types.Header
	state          *state.StateDB
	nextL1MsgIndex uint64
	blockSize      common.StorageSize
	txs            types.Transactions
	coalescedLogs  []*types.Log
	receipts       types.Receipts
	gasPool        *core.GasPool

	// com channels
	txnQueue         chan *types.Transaction
	applyStageRespCh <-chan error
	ResultCh         <-chan *Result

	// Test hooks
	beforeTxHook func() // Method to call before processing a transaction.
}

func NewPipeline(
	chain *core.BlockChain,
	vmConfig *vm.Config,
	state *state.StateDB,

	header *types.Header,
	nextL1MsgIndex uint64,
	ccc *circuitcapacitychecker.CircuitCapacityChecker,
) *Pipeline {
	return &Pipeline{
		chain:          chain,
		vmConfig:       vmConfig,
		parent:         chain.GetBlock(header.ParentHash, header.Number.Uint64()-1),
		nextL1MsgIndex: nextL1MsgIndex,
		Header:         *header,
		ccc:            ccc,
		state:          state,
		gasPool:        new(core.GasPool).AddGas(header.GasLimit),
	}
}

func (p *Pipeline) WithBeforeTxHook(beforeTxHook func()) *Pipeline {
	p.beforeTxHook = beforeTxHook
	return p
}

func (p *Pipeline) Start(deadline time.Time) error {
	p.start = time.Now()
	p.txnQueue = make(chan *types.Transaction)
	applyStageRespCh, candidateCh, err := p.traceAndApplyStage(p.txnQueue)
	if err != nil {
		log.Error("Failed starting traceAndApplyStage", "err", err)
		return err
	}
	p.applyStageRespCh = applyStageRespCh
	p.ResultCh = p.cccStage(candidateCh, deadline)
	return nil
}

func (p *Pipeline) TryPushTxns(txs types.OrderedTransactionSet, onFailingTxn func(txnIndex int, tx *types.Transaction, err error) bool) *Result {
	for {
		tx := txs.Peek()
		if tx == nil {
			break
		}

		result, err := p.TryPushTxn(tx)
		if result != nil {
			return result
		}

		switch {
		case err == nil, errors.Is(err, core.ErrNonceTooLow):
			txs.Shift()
		default:
			if errors.Is(err, ErrApplyStageDone) || onFailingTxn(p.txs.Len(), tx, err) {
				close(p.txnQueue)
				p.txnQueue = nil
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

func (p *Pipeline) TryPushTxn(tx *types.Transaction) (*Result, error) {
	if p.txnQueue == nil {
		return nil, ErrApplyStageDone
	}

	select {
	case p.txnQueue <- tx:
	case res := <-p.ResultCh:
		return res, nil
	}

	select {
	case err, valid := <-p.applyStageRespCh:
		if !valid {
			return nil, ErrApplyStageDone
		}
		return nil, err
	case res := <-p.ResultCh:
		return res, nil
	}
}

func (p *Pipeline) Kill() {
	if p.txnQueue != nil {
		close(p.txnQueue)
	}

	select {
	case <-p.applyStageRespCh:
		<-p.ResultCh
	case <-p.ResultCh:
		<-p.applyStageRespCh
	}
}

type BlockCandidate struct {
	LastTrace      *types.BlockTrace
	NextL1MsgIndex uint64

	// accumulated state
	Header        *types.Header
	State         *state.StateDB
	Txs           types.Transactions
	Receipts      types.Receipts
	CoalescedLogs []*types.Log
}

func (p *Pipeline) traceAndApplyStage(txsIn <-chan *types.Transaction) (<-chan error, <-chan *BlockCandidate, error) {
	p.state.StartPrefetcher("miner")
	newCandidateCh := make(chan *BlockCandidate)
	resCh := make(chan error)
	go func() {
		defer func() {
			close(newCandidateCh)
			close(resCh)
			p.state.StopPrefetcher()
		}()

		var tx *types.Transaction
		for {
			applyIdleTimer.Time(func() {
				tx = <-txsIn
			})
			if tx == nil {
				return
			}

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

			if tx.IsL1MessageTx() && tx.AsL1MessageTx().QueueIndex != p.nextL1MsgIndex {
				// Continue, we might still be able to include some L2 messages
				resCh <- ErrUnexpectedL1MessageIndex
				continue
			}

			if !tx.IsL1MessageTx() && !p.chain.Config().Scroll.IsValidBlockSize(p.blockSize+tx.Size()) {
				// can't fit this txn in this block, silently ignore and continue looking for more txns
				resCh <- nil
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
				select {
				case newCandidateCh <- &BlockCandidate{
					LastTrace:      trace,
					NextL1MsgIndex: p.nextL1MsgIndex,

					Header:        types.CopyHeader(&p.Header),
					State:         p.state.Copy(),
					Txs:           p.txs,
					Receipts:      p.receipts,
					CoalescedLogs: p.coalescedLogs,
				}:
				case tx = <-txsIn:
					if tx != nil {
						panic("shouldn't have happened")
					}
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
			resCh <- err
		}
	}()
	return resCh, newCandidateCh, nil
}

type Result struct {
	OverflowingTx    *types.Transaction
	OverflowingTrace *types.BlockTrace
	CCCErr           error

	Rows       *types.RowConsumption
	FinalBlock *BlockCandidate
}

func (p *Pipeline) cccStage(candidates <-chan *BlockCandidate, deadline time.Time) <-chan *Result {
	p.ccc.Reset()
	resultCh := make(chan *Result)
	var lastCandidate *BlockCandidate
	var lastAccRows *types.RowConsumption
	var deadlineReached bool

	go func() {
		defer func() {
			close(resultCh)
			lifetimeTimer.UpdateSince(p.start)
		}()
		for {
			idleStart := time.Now()
			select {
			case <-time.After(time.Until(deadline)):
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
				// avoid deadline case being triggered again and again
				deadline = time.Now().Add(time.Hour)
			case candidate := <-candidates:
				cccIdleTimer.UpdateSince(idleStart)
				cccStart := time.Now()
				var accRows *types.RowConsumption
				var err error
				if candidate != nil {
					accRows, err = p.ccc.ApplyTransaction(candidate.LastTrace)
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

	// don't commit the state during tracing for circuit capacity checker, otherwise we cannot revert.
	// and even if we don't commit the state, the `refund` value will still be correct, as explained in `CommitTransaction`
	commitStateAfterApply := false
	snap := p.state.Snapshot()

	// 1. we have to check circuit capacity before `core.ApplyTransaction`,
	// because if the tx can be successfully executed but circuit capacity overflows, it will be inconvenient to revert.
	// 2. even if we don't commit to the state during the tracing (which means `clearJournalAndRefund` is not called during the tracing),
	// the `refund` value will still be correct, because:
	// 2.1 when starting handling the first tx, `state.refund` is 0 by default,
	// 2.2 after tracing, the state is either committed in `core.ApplyTransaction`, or reverted, so the `state.refund` can be cleared,
	// 2.3 when starting handling the following txs, `state.refund` comes as 0
	trace, err = tracing.NewTracerWrapper().CreateTraceEnvAndGetBlockTrace(p.chain.Config(), p.chain, p.chain.Engine(), p.chain.Database(),
		p.state, p.parent, types.NewBlockWithHeader(&p.Header).WithBody([]*types.Transaction{tx}, nil), commitStateAfterApply)
	// `w.current.traceEnv.State` & `w.current.state` share a same pointer to the state, so only need to revert `w.current.state`
	// revert to snapshot for calling `core.ApplyMessage` again, (both `traceEnv.GetBlockTrace` & `core.ApplyTransaction` will call `core.ApplyMessage`)
	p.state.RevertToSnapshot(snap)
	if err != nil {
		return nil, nil, err
	}

	// create new snapshot for `core.ApplyTransaction`
	snap = p.state.Snapshot()

	var receipt *types.Receipt
	receipt, err = core.ApplyTransaction(p.chain.Config(), p.chain, nil /* coinbase will default to chainConfig.Scroll.FeeVaultAddress */, p.gasPool,
		p.state, &p.Header, tx, &p.Header.GasUsed, *p.vmConfig)
	if err != nil {
		p.state.RevertToSnapshot(snap)
		return nil, trace, err
	}
	return receipt, trace, nil
}
