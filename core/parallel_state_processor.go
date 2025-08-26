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
	"github.com/ethereum/go-ethereum/params"
	"golang.org/x/sync/errgroup"
	"slices"
	"time"
)

type ProcessResultWithMetrics struct {
	ProcessResult      *ProcessResult
	PreProcessTime     time.Duration
	PreProcessLoadTime time.Duration
	PostProcessTime    time.Duration
	RootCalcTime       time.Duration
	ExecTime           time.Duration

	StateDiffCalcTime time.Duration // time it took to convert BAL into a set of state diffs
}

type ParallelStateProcessor struct {
	*StateProcessor
	vmCfg *vm.Config
}

func NewParallelStateProcessor(config *params.ChainConfig, chain *HeaderChain, cfg *vm.Config) ParallelStateProcessor {
	res := NewStateProcessor(config, chain)
	return ParallelStateProcessor{
		res,
		cfg,
	}
}

// called by resultHandler when all transactions have successfully executed.
// performs post-tx state transition (system contracts and withdrawals)
// and calculates the ProcessResult, returning it to be sent on resCh
// by resultHandler
func (p *ParallelStateProcessor) prepareExecResult(block *types.Block, tExecStart time.Time, postTxState *state.StateDB, receipts types.Receipts) *ProcessResultWithMetrics {
	tExec := time.Since(tExecStart)
	var requests [][]byte
	tPostprocessStart := time.Now()
	header := block.Header()

	postTxState.SetAccessListIndex(len(block.Transactions()))
	var tracingStateDB = vm.StateDB(postTxState)
	if hooks := p.vmCfg.Tracer; hooks != nil {
		tracingStateDB = state.NewHookedState(postTxState, hooks)
	}
	context := NewEVMBlockContext(header, p.chain, nil)
	evm := vm.NewEVM(context, tracingStateDB, p.config, *p.vmCfg)

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

	computedDiff := &bal.StateDiff{make(map[common.Address]*bal.AccountState)}
	// Read requests if Prague is enabled.
	if p.config.IsPrague(block.Number(), block.Time()) {
		requests = [][]byte{}
		// EIP-6110
		if err := ParseDepositLogs(&requests, allLogs, p.config); err != nil {
			return &ProcessResultWithMetrics{
				ProcessResult: &ProcessResult{Error: err},
			}
		}

		// EIP-7002
		diff, err := ProcessWithdrawalQueue(&requests, evm)
		if err != nil {
			return &ProcessResultWithMetrics{
				ProcessResult: &ProcessResult{Error: err},
			}
		}
		computedDiff = diff
		// EIP-7251
		diff, err = ProcessConsolidationQueue(&requests, evm)
		if err != nil {
			return &ProcessResultWithMetrics{
				ProcessResult: &ProcessResult{Error: err},
			}
		}
		computedDiff.Merge(diff)
	}
	// Finalize the block, applying any consensus engine specific extras (e.g. block rewards)
	p.chain.engine.Finalize(p.chain, header, tracingStateDB, block.Body())
	// invoke Finalise so that withdrawals are accounted for in the state diff
	finalDiff := postTxState.Finalise(true)
	computedDiff.Merge(finalDiff)

	if err := postTxState.BlockAccessList().ValidateStateDiff(len(block.Transactions())+1, computedDiff); err != nil {
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
	idx     int
	receipt *types.Receipt
	err     error
}

func (p *ParallelStateProcessor) resultHandler(block *types.Block, postTxState *state.StateDB, tExecStart time.Time, txResCh <-chan txExecResult, stateRootCalcResCh <-chan stateRootCalculationResult, resCh chan *ProcessResultWithMetrics) {
	// 1. if the block has transactions, receive the execution results from all of them and return an error on resCh if any txs err'd
	// 2. once all txs are executed, compute the post-tx state transition and produce the ProcessResult sending it on resCh (or an error if the post-tx state didn't match what is reported in the BAL)
	var receipts []*types.Receipt
	gp := new(GasPool)
	gp.SetGas(block.GasLimit())
	var execErr error
	var numTxComplete int

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

	execResults := p.prepareExecResult(block, tExecStart, postTxState, receipts)
	rootCalcRes := <-stateRootCalcResCh
	if rootCalcRes.err != nil {
		resCh <- &ProcessResultWithMetrics{ProcessResult: &ProcessResult{Error: rootCalcRes.err}}
	} else {
		execResults.StateDiffCalcTime = rootCalcRes.duration
		resCh <- execResults
	}
}

type stateRootCalculationResult struct {
	err      error
	duration time.Duration
}

func (p *ParallelStateProcessor) calcAndVerifyRoot(postState *state.StateDB, block *types.Block, resCh chan stateRootCalculationResult) {
	// calculate and apply the block state modifications
	postStateDiff := &bal.StateDiff{make(map[common.Address]*bal.AccountState)}
	postState.BlockAccessList().Iterate(len(block.Transactions())+2, func(addr common.Address, state *bal.AccountState) bool {
		postStateDiff.Mutations[addr] = state
		return true
	})
	postState.ApplyStateDiff(postStateDiff)

	tVerifyStart := time.Now()
	root := postState.IntermediateRoot(true)
	tVerify := time.Since(tVerifyStart)

	var res stateRootCalculationResult
	res.duration = tVerify

	if root != block.Root() {
		res.err = fmt.Errorf("state root mismatch. local: %x. remote: %x", root, block.Root())
	}
	resCh <- res
}

// executes single transaction, validating the computed diff against the BAL
// and forwarding the txExecResult to be consumed by resultHandler
func (p *ParallelStateProcessor) execTx(block *types.Block, tx *types.Transaction, idx int, db *state.StateDB, signer types.Signer) *txExecResult {
	// TODO: also interrupt any currently-executing transactions if one failed.
	header := block.Header()
	var tracingStateDB = vm.StateDB(db)
	if hooks := p.vmCfg.Tracer; hooks != nil {
		tracingStateDB = state.NewHookedState(db, hooks)
	}
	context := NewEVMBlockContext(header, p.chain, nil)
	evm := vm.NewEVM(context, tracingStateDB, p.config, *p.vmCfg)

	msg, err := TransactionToMessage(tx, signer, header.BaseFee)
	if err != nil {
		err = fmt.Errorf("could not apply tx %d [%v]: %w", idx, tx.Hash().Hex(), err)
		return &txExecResult{err: err}
	}
	sender, _ := types.Sender(signer, tx)
	db.SetTxSender(sender)
	db.SetTxContext(tx.Hash(), idx)
	db.SetAccessListIndex(idx)

	evm.StateDB = db
	gp := new(GasPool)
	gp.SetGas(block.GasLimit())
	var gasUsed uint64
	computedDiff, receipt, err := ApplyTransactionWithEVM(msg, gp, db, block.Number(), block.Hash(), context.Time, tx, &gasUsed, evm, nil)
	if err != nil {
		err := fmt.Errorf("could not apply tx %d [%v]: %w", idx, tx.Hash().Hex(), err)
		return &txExecResult{err: err}
	}

	if err := db.BlockAccessList().ValidateStateDiff(idx+1, computedDiff); err != nil {
		return &txExecResult{err: err}
	}

	return &txExecResult{
		idx:     idx,
		receipt: receipt,
	}
}

// ProcessWithAccessList performs EVM execution and state root computation for a block which is known
// to contain an access list.
func (p *ParallelStateProcessor) Process(block *types.Block, statedb *state.StateDB, cfg vm.Config) (*ProcessResultWithMetrics, error) {
	var (
		header = block.Header()
		resCh  = make(chan *ProcessResultWithMetrics)
		signer = types.MakeSigner(p.config, header.Number, header.Time)
	)

	txResCh := make(chan txExecResult)
	pStart := time.Now()
	var (
		tPreprocess      time.Duration // time to create a set of prestates for parallel transaction execution
		tExecStart       time.Time
		modifiedPrestate = make(map[common.Address]*types.StateAccount)
		rootCalcResultCh = make(chan stateRootCalculationResult)
	)

	// Mutate the block and state according to any hard-fork specs
	if p.config.DAOForkSupport && p.config.DAOForkBlock != nil && p.config.DAOForkBlock.Cmp(block.Number()) == 0 {
		misc.ApplyDAOHardFork(statedb)
	}
	var (
		context vm.BlockContext
	)
	alReader := bal.NewReader(block.Body().AccessList)
	statedb.SetBlockAccessList(&alReader)
	// instantiate a set of StateDBs to be used for executing each transaction in parallel
	tPreprocessLoadStart := time.Now()
	modifiedPrestate = statedb.LoadModifiedPrestate(alReader.Accounts())
	tPreprocessLoad := time.Since(tPreprocessLoadStart)

	statedb.SetPrestate(modifiedPrestate)

	// Apply pre-execution system calls.
	var tracingStateDB = vm.StateDB(statedb)
	if hooks := cfg.Tracer; hooks != nil {
		tracingStateDB = state.NewHookedState(statedb, hooks)
	}
	context = NewEVMBlockContext(header, p.chain, nil)
	evm := vm.NewEVM(context, tracingStateDB, p.config, cfg)

	// validate the correctness of pre-transaction execution state changes
	computedPreTxDiff := &bal.StateDiff{make(map[common.Address]*bal.AccountState)}
	if beaconRoot := block.BeaconRoot(); beaconRoot != nil {
		computedPreTxDiff.Merge(ProcessBeaconBlockRoot(*beaconRoot, evm))
	}
	if p.config.IsPrague(block.Number(), block.Time()) || p.config.IsVerkle(block.Number(), block.Time()) {
		computedPreTxDiff.Merge(ProcessParentBlockHash(block.ParentHash(), evm))
	}

	if err := statedb.BlockAccessList().ValidateStateDiff(0, computedPreTxDiff); err != nil {
		return nil, err
	}

	// compute the post-tx state prestate (before applying final block system calls and eip-4895 withdrawals)
	// the post-tx state transition is verified by resultHandler
	postTxState := statedb.Copy()

	tPreprocess = time.Since(pStart)

	// execute transactions and state root calculation in parallel

	tExecStart = time.Now()
	go p.resultHandler(block, postTxState, tExecStart, txResCh, rootCalcResultCh, resCh)
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
	res.PreProcessLoadTime = tPreprocessLoad
	return res, nil
}
