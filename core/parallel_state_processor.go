package core

import (
	"cmp"
	"context"
	"fmt"
	"runtime"
	"slices"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/ethereum/go-ethereum/core/vm"
	"golang.org/x/sync/errgroup"
)

// ProcessResultWithMetrics wraps ProcessResult with timing breakdown for BAL block processing.
type ProcessResultWithMetrics struct {
	ProcessResult          *ProcessResult
	PreProcessTime         time.Duration
	StateTransitionMetrics *state.BALStateTransitionMetrics
	ExecTime               time.Duration
	PostProcessTime        time.Duration

	// Preimages holds the SHA3 preimages recorded during the execution of
	// the block.
	Preimages map[common.Hash][]byte
}

// errResult wraps an error into a new ProcessResultWithMetrics instance
func errResult(err error) *ProcessResultWithMetrics {
	return &ProcessResultWithMetrics{ProcessResult: &ProcessResult{Error: err}}
}

// ParallelStateProcessor is used to execute and verify blocks containing
// access lists.
type ParallelStateProcessor struct {
	*StateProcessor
	vmCfg *vm.Config
}

// NewParallelStateProcessor returns a new ParallelStateProcessor instance.
func NewParallelStateProcessor(chain *HeaderChain, vmConfig *vm.Config) *ParallelStateProcessor {
	return &ParallelStateProcessor{
		StateProcessor: NewStateProcessor(chain),
		vmCfg:          vmConfig,
	}
}

// execVMConfig returns the subset of the configured VM options that is safe to
// reuse across the parallel per-transaction and post-transaction executions.
// Only the fields explicitly copied here are propagated (mirroring the original
// per-tx behaviour); notably the full caller-supplied config is used only for
// pre-execution in processBlockPreTx.
func (p *ParallelStateProcessor) execVMConfig() vm.Config {
	return vm.Config{
		NoBaseFee:               p.vmCfg.NoBaseFee,
		EnablePreimageRecording: p.vmCfg.EnablePreimageRecording,
		ExtraEips:               slices.Clone(p.vmCfg.ExtraEips),
	}
}

// called by resultHandler when all transactions have successfully executed.
// performs post-tx state transition (system contracts and withdrawals)
// and calculates the ProcessResult, returning it to be sent on resCh
// by resultHandler
func (p *ParallelStateProcessor) prepareExecResult(block *types.Block, tExecStart time.Time, preTxBAL *bal.ConstructionBlockAccessList, accessList *bal.AccessListReader, statedb *state.StateDB, results []txExecResult) *ProcessResultWithMetrics {
	tExec := time.Since(tExecStart)
	tPostprocessStart := time.Now()
	header := block.Header()

	// The post-execution changes are recorded at the BAL index immediately
	// following the last transaction.
	lastBALIdx := len(block.Transactions()) + 1
	postTxState := statedb.WithReader(state.NewReaderWithAccessList(statedb.Reader(), accessList, lastBALIdx))

	evm := vm.NewEVM(NewEVMBlockContext(header, p.chain, nil), postTxState, p.chainConfig(), p.execVMConfig())

	// 1. order the receipts by tx index
	// 2. correctly calculate the cumulative gas used per receipt, returning bad block error if it goes over the allowed
	slices.SortFunc(results, func(a, b txExecResult) int {
		return cmp.Compare(a.receipt.TransactionIndex, b.receipt.TransactionIndex)
	})

	var (
		// Per-dimension cumulative sums for 2D block gas (EIP-8037).
		sumRegular        uint64
		sumState          uint64
		cumulativeReceipt uint64 // cumulative receipt gas (what users pay)

		allLogs     []*types.Log
		allReceipts []*types.Receipt
	)
	for _, result := range results {
		sumRegular += result.txRegular
		sumState += result.txState

		cumulativeReceipt += result.execGas
		result.receipt.CumulativeGasUsed = cumulativeReceipt
		allLogs = append(allLogs, result.receipt.Logs...)
		allReceipts = append(allReceipts, result.receipt)
	}
	// Block gas = max(sum_regular, sum_state) per EIP-8037.
	blockGasUsed := max(sumRegular, sumState)
	if blockGasUsed > header.GasLimit {
		return errResult(fmt.Errorf("gas limit exceeded"))
	}

	requests, postBAL, err := PostExecution(context.Background(), p.chainConfig(), block.Number(), block.Time(), allLogs, evm, uint32(lastBALIdx))
	if err != nil {
		return errResult(err)
	}

	p.chain.Engine().Finalize(p.chain, block.Header(), evm.StateDB, block.Body(), uint32(lastBALIdx), postBAL)

	// Gather preimages from block execution.  postTxState contains
	// preimages from pre/post tx system contract execution.
	preimages := make(map[common.Hash][]byte)
	for hash, preimage := range postTxState.Preimages() {
		preimages[hash] = preimage
	}
	for i := range results {
		for hash, preimage := range results[i].preimages {
			if _, ok := preimages[hash]; !ok {
				preimages[hash] = preimage
			}
		}
	}

	blockAccessList := bal.NewConstructionBlockAccessList()
	blockAccessList.Merge(preTxBAL)
	blockAccessList.Merge(postBAL)
	for _, res := range results {
		blockAccessList.Merge(res.blockAccessList)
	}

	// TODO: do we move validation to ValidateState?
	if block.AccessList().Hash() != blockAccessList.ToEncodingObj().Hash() {
		// TODO: expose json string method on encoding block access list and log it here
		return errResult(fmt.Errorf("invalid block access list: mismatch between local and remote block access list"))
	}

	tPostprocess := time.Since(tPostprocessStart)

	return &ProcessResultWithMetrics{
		ProcessResult: &ProcessResult{
			Receipts: allReceipts,
			Requests: requests,
			Logs:     allLogs,
			GasUsed:  blockGasUsed,
			Bal:      blockAccessList,
		},
		PostProcessTime: tPostprocess,
		ExecTime:        tExec,
		Preimages:       preimages,
	}
}

type txExecResult struct {
	receipt *types.Receipt
	err     error  // non-EVM error which would render the block invalid
	execGas uint64 // gas reported on the receipt (what the user pays)

	// Per-tx dimensional gas for Amsterdam 2D gas accounting (EIP-8037).
	txRegular uint64
	txState   uint64

	blockAccessList *bal.ConstructionBlockAccessList

	// holds preimages recorded during execution of the tx
	preimages map[common.Hash][]byte
}

// resultHandler polls until all transactions have finished executing and the
// state root calculation is complete. The result is emitted on resCh.
func (p *ParallelStateProcessor) resultHandler(block *types.Block, preTxBAL *bal.ConstructionBlockAccessList, prepared *bal.AccessListReader, statedb *state.StateDB, tExecStart time.Time, txResCh <-chan txExecResult, stateRootCalcResCh <-chan stateRootCalculationResult, resCh chan *ProcessResultWithMetrics) {
	// 1. if the block has transactions, receive the execution results from all of them and return an error on resCh if any txs err'd
	// 2. once all txs are executed, compute the post-tx state transition and produce the ProcessResult sending it on resCh (or an error if the post-tx state didn't match what is reported in the BAL)
	var (
		results              []txExecResult
		cumulativeStateGas   uint64
		cumulativeRegularGas uint64
		execErr              error
	)

	if numTx := len(block.Transactions()); numTx > 0 {
		for completed := 0; completed < numTx; completed++ {
			res := <-txResCh
			if execErr != nil {
				// A block-invalidating result was already seen; keep draining so
				// the worker goroutines don't block on their sends.
				continue
			}
			switch {
			case res.err != nil:
				execErr = res.err
			default:
				bottleneck := max(cumulativeRegularGas+res.txRegular, cumulativeStateGas+res.txState)
				if bottleneck > block.GasLimit() {
					execErr = fmt.Errorf("block used too much gas in bottleneck dimension: %d. block gas limit is %d", bottleneck, block.GasLimit())
					continue
				}
				cumulativeRegularGas += res.txRegular
				cumulativeStateGas += res.txState
				results = append(results, res)
			}
		}

		if execErr != nil {
			// Drain stateRootCalcResCh so the calcAndVerifyRoot goroutine can exit.
			<-stateRootCalcResCh
			resCh <- errResult(execErr)
			return
		}
	}

	execResults := p.prepareExecResult(block, tExecStart, preTxBAL, prepared, statedb, results)
	rootCalcRes := <-stateRootCalcResCh

	switch {
	case execResults.ProcessResult.Error != nil:
		resCh <- execResults
	case rootCalcRes.err != nil:
		resCh <- errResult(rootCalcRes.err)
	default:
		execResults.StateTransitionMetrics = rootCalcRes.metrics
		resCh <- execResults
	}
}

type stateRootCalculationResult struct {
	err     error
	metrics *state.BALStateTransitionMetrics
}

// calcAndVerifyRoot performs the post-state root hash calculation, verifying
// it against what is reported by the block and returning a result on resCh.
func (p *ParallelStateProcessor) calcAndVerifyRoot(block *types.Block, stateTransition *state.BALStateTransition, resCh chan stateRootCalculationResult) {
	root := stateTransition.IntermediateRoot(false)

	res := stateRootCalculationResult{
		metrics: stateTransition.Metrics(),
	}
	if root != block.Root() {
		res.err = fmt.Errorf("state root mismatch. local: %x. remote: %x", root, block.Root())
	}
	resCh <- res
}

// execTx executes a single transaction returning a result which includes state accessed/modified.
func (p *ParallelStateProcessor) execTx(block *types.Block, tx *types.Transaction, balIdx int, db *state.StateDB, signer types.Signer) *txExecResult {
	header := block.Header()
	evmContext := NewEVMBlockContext(header, p.chain, nil)
	evm := vm.NewEVM(evmContext, db, p.chainConfig(), p.execVMConfig())

	msg, err := TransactionToMessage(tx, signer, header.BaseFee)
	if err != nil {
		return &txExecResult{err: fmt.Errorf("could not apply tx %d [%v]: %w", balIdx, tx.Hash().Hex(), err)}
	}
	sender, err := signer.Sender(tx)
	if err != nil {
		return &txExecResult{err: fmt.Errorf("could not recover sender for tx at bal idx %d: %w", balIdx, err)}
	}

	gp := NewGasPool(block.GasLimit())
	// TODO: make precompiled addresses be resolvable from chain config + block
	db.Prepare(evm.GetRules(), sender, block.Coinbase(), tx.To(), vm.PrecompiledAddressesCancun, tx.AccessList())
	db.SetTxContext(tx.Hash(), balIdx-1, uint32(balIdx))

	receipt, txBAL, err := ApplyTransactionWithEVM(msg, gp, db, block.Number(), block.Hash(), evmContext.Time, tx, evm)
	if err != nil {
		return &txExecResult{err: fmt.Errorf("could not apply tx %d [%v]: %w", balIdx, tx.Hash().Hex(), err)}
	}

	return &txExecResult{
		receipt:         receipt,
		execGas:         receipt.GasUsed,
		txRegular:       gp.cumulativeRegular,
		txState:         gp.cumulativeState,
		preimages:       db.Preimages(),
		blockAccessList: txBAL,
	}
}

func (p *ParallelStateProcessor) processBlockPreTx(block *types.Block, statedb *state.StateDB, cfg vm.Config) *bal.ConstructionBlockAccessList {
	header := block.Header()
	evm := vm.NewEVM(NewEVMBlockContext(header, p.chain, nil), statedb, p.chainConfig(), cfg)
	return PreExecution(context.Background(), block.BeaconRoot(), block.ParentHash(), p.chainConfig(), evm, block.Number(), block.Time())
}

// Process performs EVM execution and state root computation for a block which is known
// to contain an access list.
func (p *ParallelStateProcessor) Process(block *types.Block, stateTransition *state.BALStateTransition, statedb *state.StateDB, cfg vm.Config) (*ProcessResultWithMetrics, error) {
	header := block.Header()
	signer := types.MakeSigner(p.chainConfig(), header.Number, header.Time)

	var (
		resCh            = make(chan *ProcessResultWithMetrics)
		rootCalcResultCh = make(chan stateRootCalculationResult)
		txResCh          = make(chan txExecResult)
	)

	// Pre-transaction processing: system-contract updates and the pre-tx BAL.
	pStart := time.Now()
	startingState := statedb.Copy()
	prepared := stateTransition.PreparedAccessList()
	preTxBAL := p.processBlockPreTx(block, statedb, cfg)
	tPreprocess := time.Since(pStart)

	// Execute transactions and the state-root calculation in parallel.
	tExecStart := time.Now()
	go p.resultHandler(block, preTxBAL, prepared, statedb, tExecStart, txResCh, rootCalcResultCh, resCh)

	// Workers execute transactions concurrently against per-tx state copies.
	// Each worker reports completion (and any block-invalidating error) on
	// txResCh, which resultHandler drains. Worker errors therefore flow through
	// the channel rather than the errgroup, so the group is used purely to bound
	// concurrency and Wait() is intentionally not called.
	var workers errgroup.Group
	workers.SetLimit(runtime.NumCPU())
	for i, tx := range block.Transactions() {
		balIdx := i + 1
		prestate := startingState.Copy()
		workers.Go(func() error {
			prestate = prestate.WithReader(state.NewReaderWithAccessList(statedb.Reader(), prepared, balIdx))
			res := p.execTx(block, tx, balIdx, prestate, signer)
			txResCh <- *res
			return nil
		})
	}

	go p.calcAndVerifyRoot(block, stateTransition, rootCalcResultCh)

	res := <-resCh
	if res.ProcessResult.Error != nil {
		return nil, res.ProcessResult.Error
	}

	stateTransition.SetPreimages(res.Preimages)
	// TODO: remove preprocess metric ?
	res.PreProcessTime = tPreprocess
	return res, nil
}
