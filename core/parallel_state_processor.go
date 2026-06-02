package core

import (
	"cmp"
	"context"
	"fmt"
	"runtime"
	"slices"
	"time"

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
func (p *ParallelStateProcessor) prepareExecResult(block *types.Block, tExecStart time.Time, preTxBal *bal.ConstructionBlockAccessList, prepared *bal.PreparedAccessList, statedb *state.StateDB, results []txExecResult) *ProcessResultWithMetrics {
	tExec := time.Since(tExecStart)
	tPostprocessStart := time.Now()
	header := block.Header()

	vmContext := NewEVMBlockContext(header, p.chain, nil)
	lastBALIdx := len(block.Transactions()) + 1
	postTxState := statedb.WithReader(state.NewReaderWithPreparedAccessList(statedb.Reader(), prepared, lastBALIdx))

	cfg := vm.Config{
		NoBaseFee:               p.vmCfg.NoBaseFee,
		EnablePreimageRecording: p.vmCfg.EnablePreimageRecording,
		ExtraEips:               slices.Clone(p.vmCfg.ExtraEips),
	}
	evm := vm.NewEVM(vmContext, postTxState, p.chainConfig(), cfg)

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
	)

	var allLogs []*types.Log
	var allReceipts []*types.Receipt
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
		return &ProcessResultWithMetrics{
			ProcessResult: &ProcessResult{Error: fmt.Errorf("gas limit exceeded")},
		}
	}

	requests, postBal, err := PostExecution(context.Background(), p.chainConfig(), block.Number(), block.Time(), allLogs, evm, uint32(len(block.Transactions())+1))
	if err != nil {
		return &ProcessResultWithMetrics{
			ProcessResult: &ProcessResult{Error: err},
		}
	}

	p.chain.Engine().Finalize(p.chain, block.Header(), evm.StateDB, block.Body(), uint32(len(block.Transactions()))+1, postBal)

	blockAccessList := bal.NewConstructionBlockAccessList()
	blockAccessList.Merge(preTxBal)
	blockAccessList.Merge(postBal)

	for _, res := range results {
		blockAccessList.Merge(res.blockAccessList)
	}

	// TODO: do we move validation to ValidateState?
	if block.AccessList().Hash() != blockAccessList.ToEncodingObj().Hash() {
		// TODO: expose json string method on encoding block access list and log it here
		return &ProcessResultWithMetrics{
			ProcessResult: &ProcessResult{Error: fmt.Errorf("invalid block access list: mismatch between local and remote block access list")},
		}
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
	}
}

type txExecResult struct {
	idx      int // transaction index
	receipt  *types.Receipt
	err      error // non-EVM error which would render the block invalid
	blockGas uint64
	execGas  uint64

	// Per-tx dimensional gas for Amsterdam 2D gas accounting (EIP-8037).
	txRegular uint64
	txState   uint64

	blockAccessList *bal.ConstructionBlockAccessList
}

// resultHandler polls until all transactions have finished executing and the
// state root calculation is complete. The result is emitted on resCh.
func (p *ParallelStateProcessor) resultHandler(block *types.Block, preTxBAL *bal.ConstructionBlockAccessList, prepared *bal.PreparedAccessList, statedb *state.StateDB, tExecStart time.Time, txResCh <-chan txExecResult, stateRootCalcResCh <-chan stateRootCalculationResult, resCh chan *ProcessResultWithMetrics) {
	// 1. if the block has transactions, receive the execution results from all of them and return an error on resCh if any txs err'd
	// 2. once all txs are executed, compute the post-tx state transition and produce the ProcessResult sending it on resCh (or an error if the post-tx state didn't match what is reported in the BAL)
	var results []txExecResult
	var cumulativeStateGas, cumulativeRegularGas uint64
	var execErr error
	var numTxComplete int

	if len(block.Transactions()) > 0 {
	loop:
		for {
			select {
			case res := <-txResCh:
				numTxComplete++
				if execErr == nil {
					// short-circuit if invalid block was detected
					if res.err != nil {
						execErr = res.err
					} else if bottleneck := max(cumulativeRegularGas+res.txRegular, cumulativeStateGas+res.txState); bottleneck > block.GasLimit() {
						execErr = fmt.Errorf("block used too much gas in bottleneck dimension: %d. block gas limit is %d", bottleneck, block.GasLimit())
					} else {
						cumulativeStateGas += res.txState
						results = append(results, res)
					}
				}
				if numTxComplete == len(block.Transactions()) {
					break loop
				}
			}
		}

		if execErr != nil {
			// Drain stateRootCalcResCh so calcAndVerifyRoot goroutine can exit.
			<-stateRootCalcResCh
			resCh <- &ProcessResultWithMetrics{ProcessResult: &ProcessResult{Error: execErr}}
			return
		}
	}

	execResults := p.prepareExecResult(block, tExecStart, preTxBAL, prepared, statedb, results)
	rootCalcRes := <-stateRootCalcResCh

	if execResults.ProcessResult.Error != nil {
		resCh <- execResults
	} else if rootCalcRes.err != nil {
		resCh <- &ProcessResultWithMetrics{ProcessResult: &ProcessResult{Error: rootCalcRes.err}}
	} else {
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

// execTx executes single transaction returning a result which includes state accessed/modified
func (p *ParallelStateProcessor) execTx(block *types.Block, tx *types.Transaction, balIdx int, db *state.StateDB, signer types.Signer) *txExecResult {
	header := block.Header()
	context := NewEVMBlockContext(header, p.chain, nil)

	cfg := vm.Config{
		NoBaseFee:               p.vmCfg.NoBaseFee,
		EnablePreimageRecording: p.vmCfg.EnablePreimageRecording,
		ExtraEips:               slices.Clone(p.vmCfg.ExtraEips),
	}
	evm := vm.NewEVM(context, db, p.chainConfig(), cfg)

	msg, err := TransactionToMessage(tx, signer, header.BaseFee)
	if err != nil {
		err = fmt.Errorf("could not apply tx %d [%v]: %w", balIdx, tx.Hash().Hex(), err)
		return &txExecResult{err: err}
	}
	gp := NewGasPool(block.GasLimit())
	sender, err := signer.Sender(tx)
	if err != nil {
		// TODO: can this even happen at this stage?
		err = fmt.Errorf("could not recover sender for tx at bal idx %d: %v\n", balIdx, err)
	}
	// TODO: make precompiled addresses be resolvable from chain config + block
	db.Prepare(evm.GetRules(), sender, block.Coinbase(), tx.To(), vm.PrecompiledAddressesCancun, tx.AccessList())

	db.SetTxContext(tx.Hash(), balIdx-1, uint32(balIdx))

	receipt, txBAL, err := ApplyTransactionWithEVM(msg, gp, db, block.Number(), block.Hash(), context.Time, tx, evm)
	if err != nil {
		err := fmt.Errorf("could not apply tx %d [%v]: %w", balIdx, tx.Hash().Hex(), err)
		return &txExecResult{err: err}
	}

	return &txExecResult{
		idx:             balIdx,
		receipt:         receipt,
		execGas:         receipt.GasUsed,
		blockGas:        gp.Used(),
		txRegular:       gp.cumulativeRegular,
		txState:         gp.cumulativeState,
		blockAccessList: txBAL,
	}
}

func (p *ParallelStateProcessor) processBlockPreTx(block *types.Block, statedb *state.StateDB, cfg vm.Config) (*bal.ConstructionBlockAccessList, error) {
	var (
		header = block.Header()
	)
	vmContext := NewEVMBlockContext(header, p.chain, nil)
	evm := vm.NewEVM(vmContext, statedb, p.chainConfig(), cfg)

	accessList := PreExecution(context.Background(), block.BeaconRoot(), block.ParentHash(), p.chainConfig(), evm, block.Number(), block.Time())
	return accessList, nil
}

// Process performs EVM execution and state root computation for a block which is known
// to contain an access list.
func (p *ParallelStateProcessor) Process(block *types.Block, stateTransition *state.BALStateTransition, statedb *state.StateDB, cfg vm.Config) (*ProcessResultWithMetrics, error) {
	var (
		header           = block.Header()
		resCh            = make(chan *ProcessResultWithMetrics)
		signer           = types.MakeSigner(p.chainConfig(), header.Number, header.Time)
		rootCalcResultCh = make(chan stateRootCalculationResult)
		txResCh          = make(chan txExecResult)

		pStart      = time.Now()
		tExecStart  time.Time
		tPreprocess time.Duration // time to create a set of prestates for parallel transaction execution
	)

	startingState := statedb.Copy()
	prepared := stateTransition.PreparedAccessList()
	preTxBal, err := p.processBlockPreTx(block, statedb, cfg)
	if err != nil {
		return nil, err
	}

	// compute the reads/mutations at the last bal index
	tPreprocess = time.Since(pStart)

	// execute transactions and state root calculation in parallel
	tExecStart = time.Now()
	go p.resultHandler(block, preTxBal, prepared, statedb, tExecStart, txResCh, rootCalcResultCh, resCh)
	var workers errgroup.Group
	workers.SetLimit(runtime.NumCPU())
	for i, t := range block.Transactions() {
		tx := t
		idx := i
		sdb := startingState.Copy()
		workers.Go(func() error {
			startingState := sdb.WithReader(state.NewReaderWithPreparedAccessList(statedb.Reader(), prepared, idx+1))
			res := p.execTx(block, tx, idx+1, startingState, signer)
			txResCh <- *res
			return nil
		})
	}

	go p.calcAndVerifyRoot(block, stateTransition, rootCalcResultCh)

	res := <-resCh
	if res.ProcessResult.Error != nil {
		return nil, res.ProcessResult.Error
	}
	// TODO: remove preprocess metric ?
	res.PreProcessTime = tPreprocess
	return res, nil
}
