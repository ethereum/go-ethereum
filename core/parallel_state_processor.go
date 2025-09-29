package core

import (
	"cmp"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/ethereum/go-ethereum/core/vm"
	"golang.org/x/sync/errgroup"
	"slices"
	"time"
)

// ProcessResultWithMetrics wraps ProcessResult with some metrics that are
// emitted when executing blocks containing access lists.
type ProcessResultWithMetrics struct {
	ProcessResult *ProcessResult
	// the time it took to load modified prestate accounts from disk and instantiate statedbs for execution
	PreProcessTime time.Duration
	// the time it took to validate the block post transaction execution and state root calculation
	PostProcessTime time.Duration
	// the time it took to hash the state root, including intermediate node reads
	RootCalcTime time.Duration
	// the time that it took to load the prestate for accounts that were updated as part of
	// the state root update
	PrestateLoadTime time.Duration
	// the time it took to execute all txs in the block
	ExecTime time.Duration
}

// ParallelStateProcessor is used to execute and verify blocks containing
// access lists.
type ParallelStateProcessor struct {
	*StateProcessor
	vmCfg *vm.Config
}

// NewParallelStateProcessor returns a new ParallelStateProcessor instance.
func NewParallelStateProcessor(chain *HeaderChain, vmConfig *vm.Config) ParallelStateProcessor {
	res := NewStateProcessor(chain)
	return ParallelStateProcessor{
		res,
		vmConfig,
	}
}

// called by resultHandler when all transactions have successfully executed.
// performs post-tx state transition (system contracts and withdrawals)
// and calculates the ProcessResult, returning it to be sent on resCh
// by resultHandler
func (p *ParallelStateProcessor) prepareExecResult(block *types.Block, allStateReads *bal.StateAccesses, tExecStart time.Time, postTxState *state.StateDB, receipts types.Receipts) *ProcessResultWithMetrics {
	tExec := time.Since(tExecStart)
	var requests [][]byte
	tPostprocessStart := time.Now()
	header := block.Header()

	balTracer, hooks := NewBlockAccessListTracer(len(block.Transactions()) + 1)
	tracingStateDB := state.NewHookedState(postTxState, hooks)
	context := NewEVMBlockContext(header, p.chain, nil)
	postTxState.SetAccessListIndex(len(block.Transactions()) + 1)

	cfg := vm.Config{
		Tracer:                  hooks,
		NoBaseFee:               p.vmCfg.NoBaseFee,
		EnablePreimageRecording: p.vmCfg.EnablePreimageRecording,
		ExtraEips:               slices.Clone(p.vmCfg.ExtraEips),
		StatelessSelfValidation: p.vmCfg.StatelessSelfValidation,
		EnableWitnessStats:      p.vmCfg.EnableWitnessStats,
	}
	cfg.Tracer = hooks
	evm := vm.NewEVM(context, tracingStateDB, p.chainConfig(), cfg)

	// 1. order the receipts by tx index
	// 2. correctly calculate the cumulative gas used per receipt, returning bad block error if it goes over the allowed
	slices.SortFunc(receipts, func(a, b *types.Receipt) int {
		return cmp.Compare(a.TransactionIndex, b.TransactionIndex)
	})

	var cumulativeGasUsed uint64
	var allLogs []*types.Log
	for _, receipt := range receipts {
		receipt.CumulativeGasUsed = cumulativeGasUsed + receipt.GasUsed
		cumulativeGasUsed += receipt.GasUsed
		if receipt.CumulativeGasUsed > header.GasLimit {
			return &ProcessResultWithMetrics{
				ProcessResult: &ProcessResult{Error: fmt.Errorf("gas limit exceeded")},
			}
		}
		allLogs = append(allLogs, receipt.Logs...)
	}

	// Read requests if Prague is enabled.
	if p.chainConfig().IsPrague(block.Number(), block.Time()) {
		requests = [][]byte{}
		// EIP-6110
		if err := ParseDepositLogs(&requests, allLogs, p.chainConfig()); err != nil {
			return &ProcessResultWithMetrics{
				ProcessResult: &ProcessResult{Error: err},
			}
		}

		// EIP-7002
		err := ProcessWithdrawalQueue(&requests, evm)
		if err != nil {
			return &ProcessResultWithMetrics{
				ProcessResult: &ProcessResult{Error: err},
			}
		}

		// EIP-7251
		err = ProcessConsolidationQueue(&requests, evm)
		if err != nil {
			return &ProcessResultWithMetrics{
				ProcessResult: &ProcessResult{Error: err},
			}
		}
	}

	// Finalize the block, applying any consensus engine specific extras (e.g. block rewards)
	p.chain.Engine().Finalize(p.chain, header, tracingStateDB, block.Body())
	// invoke FinaliseIdxChanges so that withdrawals are accounted for in the state diff
	postTxState.Finalise(true)

	balTracer.OnBlockFinalization()
	diff, stateReads := balTracer.IdxChanges()
	allStateReads.Merge(stateReads)

	balIdx := len(block.Transactions()) + 1
	if err := postTxState.BlockAccessList().ValidateStateDiff(balIdx, diff); err != nil {
		return &ProcessResultWithMetrics{
			ProcessResult: &ProcessResult{Error: err},
		}
	}

	if err := postTxState.BlockAccessList().ValidateStateReads(*allStateReads); err != nil {
		return &ProcessResultWithMetrics{
			ProcessResult: &ProcessResult{Error: err},
		}
	}

	tPostprocess := time.Since(tPostprocessStart)

	return &ProcessResultWithMetrics{
		ProcessResult: &ProcessResult{
			Receipts: receipts,
			Requests: requests,
			Logs:     allLogs,
			GasUsed:  cumulativeGasUsed,
		},
		PostProcessTime: tPostprocess,
		ExecTime:        tExec,
	}
}

type txExecResult struct {
	idx     int // transaction index
	receipt *types.Receipt
	err     error // non-EVM error which would render the block invalid

	stateReads bal.StateAccesses
}

// resultHandler polls until all transactions have finished executing and the
// state root calculation is complete. The result is emitted on resCh.
func (p *ParallelStateProcessor) resultHandler(block *types.Block, preTxStateReads bal.StateAccesses, postTxState *state.StateDB, tExecStart time.Time, txResCh <-chan txExecResult, stateRootCalcResCh <-chan stateRootCalculationResult, resCh chan *ProcessResultWithMetrics) {
	// 1. if the block has transactions, receive the execution results from all of them and return an error on resCh if any txs err'd
	// 2. once all txs are executed, compute the post-tx state transition and produce the ProcessResult sending it on resCh (or an error if the post-tx state didn't match what is reported in the BAL)
	var receipts []*types.Receipt
	gp := new(GasPool)
	gp.SetGas(block.GasLimit())
	var execErr error
	var numTxComplete int

	allReads := make(bal.StateAccesses)
	allReads.Merge(preTxStateReads)
	if len(block.Transactions()) > 0 {
	loop:
		for {
			select {
			case res := <-txResCh:
				if execErr == nil {
					if res.err != nil {
						execErr = res.err
					} else {
						if err := gp.SubGas(res.receipt.GasUsed); err != nil {
							execErr = err
						} else {
							receipts = append(receipts, res.receipt)
							allReads.Merge(res.stateReads)
						}
					}
				}
				numTxComplete++
				if numTxComplete == len(block.Transactions()) {
					break loop
				}
			}
		}

		if execErr != nil {
			resCh <- &ProcessResultWithMetrics{ProcessResult: &ProcessResult{Error: execErr}}
			return
		}
	}

	execResults := p.prepareExecResult(block, &allReads, tExecStart, postTxState, receipts)
	rootCalcRes := <-stateRootCalcResCh

	if execResults.ProcessResult.Error != nil {
		resCh <- execResults
	} else if rootCalcRes.err != nil {
		resCh <- &ProcessResultWithMetrics{ProcessResult: &ProcessResult{Error: rootCalcRes.err}}
	} else {
		execResults.RootCalcTime = rootCalcRes.rootCalcTime
		execResults.PrestateLoadTime = rootCalcRes.prestateLoadTime
		resCh <- execResults
	}
}

type stateRootCalculationResult struct {
	err              error
	prestateLoadTime time.Duration
	rootCalcTime     time.Duration
	root             common.Hash
}

// calcAndVerifyRoot performs the post-state root hash calculation, verifying
// it against what is reported by the block and returning a result on resCh.
func (p *ParallelStateProcessor) calcAndVerifyRoot(preState *state.StateDB, block *types.Block, resCh chan stateRootCalculationResult) {
	// calculate and apply the block state modifications
	root, prestateLoadTime, rootCalcTime := preState.BlockAccessList().StateRoot(preState)

	res := stateRootCalculationResult{
		root:             root,
		prestateLoadTime: prestateLoadTime,
		rootCalcTime:     rootCalcTime,
	}

	if root != block.Root() {
		res.err = fmt.Errorf("state root mismatch. local: %x. remote: %x", root, block.Root())
	}
	resCh <- res
}

// execTx executes single transaction returning a result which includes state accessed/modified
func (p *ParallelStateProcessor) execTx(block *types.Block, tx *types.Transaction, txIdx int, db *state.StateDB, signer types.Signer) *txExecResult {
	header := block.Header()
	balTracer, hooks := NewBlockAccessListTracer(txIdx + 1)
	tracingStateDB := state.NewHookedState(db, hooks)
	context := NewEVMBlockContext(header, p.chain, nil)

	cfg := vm.Config{
		Tracer:                  hooks,
		NoBaseFee:               p.vmCfg.NoBaseFee,
		EnablePreimageRecording: p.vmCfg.EnablePreimageRecording,
		ExtraEips:               slices.Clone(p.vmCfg.ExtraEips),
		StatelessSelfValidation: p.vmCfg.StatelessSelfValidation,
		EnableWitnessStats:      p.vmCfg.EnableWitnessStats,
	}
	cfg.Tracer = hooks
	evm := vm.NewEVM(context, tracingStateDB, p.chainConfig(), cfg)

	msg, err := TransactionToMessage(tx, signer, header.BaseFee)
	if err != nil {
		err = fmt.Errorf("could not apply tx %d [%v]: %w", txIdx, tx.Hash().Hex(), err)
		return &txExecResult{err: err}
	}
	gp := new(GasPool)
	gp.SetGas(block.GasLimit())
	db.SetTxContext(tx.Hash(), txIdx)
	var gasUsed uint64
	receipt, err := ApplyTransactionWithEVM(msg, gp, db, block.Number(), block.Hash(), context.Time, tx, &gasUsed, evm)
	if err != nil {
		err := fmt.Errorf("could not apply tx %d [%v]: %w", txIdx, tx.Hash().Hex(), err)
		return &txExecResult{err: err}
	}

	diff, accesses := balTracer.IdxChanges()
	if err := db.BlockAccessList().ValidateStateDiff(txIdx+1, diff); err != nil {
		return &txExecResult{err: err}
	}

	return &txExecResult{
		idx:        txIdx,
		receipt:    receipt,
		stateReads: accesses,
	}
}

// Process performs EVM execution and state root computation for a block which is known
// to contain an access list.
func (p *ParallelStateProcessor) Process(block *types.Block, statedb *state.StateDB, cfg vm.Config) (*ProcessResultWithMetrics, error) {
	var (
		header = block.Header()
		resCh  = make(chan *ProcessResultWithMetrics)
		signer = types.MakeSigner(p.chainConfig(), header.Number, header.Time)
	)

	txResCh := make(chan txExecResult)
	pStart := time.Now()
	var (
		tPreprocess      time.Duration // time to create a set of prestates for parallel transaction execution
		tExecStart       time.Time
		rootCalcResultCh = make(chan stateRootCalculationResult)
	)

	// Mutate the block and state according to any hard-fork specs
	if p.chainConfig().DAOForkSupport && p.chainConfig().DAOForkBlock != nil && p.chainConfig().DAOForkBlock.Cmp(block.Number()) == 0 {
		misc.ApplyDAOHardFork(statedb)
	}
	var (
		context vm.BlockContext
	)
	alReader := state.NewBALReader(block, statedb)
	statedb.SetBlockAccessList(alReader)

	balTracer, hooks := NewBlockAccessListTracer(0)
	tracingStateDB := state.NewHookedState(statedb, hooks)
	// TODO: figure out exactly why we need to set the hooks on the TracingStateDB and the vm.Config
	cfg.Tracer = hooks

	context = NewEVMBlockContext(header, p.chain, nil)
	evm := vm.NewEVM(context, tracingStateDB, p.chainConfig(), cfg)

	if beaconRoot := block.BeaconRoot(); beaconRoot != nil {
		ProcessBeaconBlockRoot(*beaconRoot, evm)
	}
	if p.chainConfig().IsPrague(block.Number(), block.Time()) || p.chainConfig().IsVerkle(block.Number(), block.Time()) {
		ProcessParentBlockHash(block.ParentHash(), evm)
	}

	// TODO: weird that I have to manually call finalize here
	balTracer.OnPreTxExecutionDone()

	diff, stateReads := balTracer.IdxChanges()
	if err := statedb.BlockAccessList().ValidateStateDiff(0, diff); err != nil {
		return nil, err
	}

	// compute the post-tx state prestate (before applying final block system calls and eip-4895 withdrawals)
	// the post-tx state transition is verified by resultHandler
	postTxState := statedb.Copy()

	tPreprocess = time.Since(pStart)

	// execute transactions and state root calculation in parallel

	// TODO: figure out how to funnel the state reads from the bal tracer through to the post-block-exec state/slot read
	// validation
	tExecStart = time.Now()
	go p.resultHandler(block, stateReads, postTxState, tExecStart, txResCh, rootCalcResultCh, resCh)
	var workers errgroup.Group
	startingState := statedb.Copy()
	for i, tx := range block.Transactions() {
		tx := tx
		i := i
		workers.Go(func() error {
			res := p.execTx(block, tx, i, startingState.Copy(), signer)
			txResCh <- *res
			return nil
		})
	}

	go p.calcAndVerifyRoot(statedb, block, rootCalcResultCh)

	res := <-resCh
	if res.ProcessResult.Error != nil {
		return nil, res.ProcessResult.Error
	}
	res.PreProcessTime = tPreprocess
	//	res.PreProcessLoadTime = tPreprocessLoad
	return res, nil
}
