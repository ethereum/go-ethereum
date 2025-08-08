package core

import (
	"fmt"
	"math/big"
	"runtime"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"golang.org/x/sync/errgroup"
)

var (
	ParallelExeTime      = time.Duration(0)
	PrefetchMergeBALTime = time.Duration(0)
	EmptyStatedb         = state.NewEmptyDB()
)

type ParallelStateProcessor struct {
	config *params.ChainConfig // Chain configuration options
	chain  *HeaderChain        // Canonical header chain
}

func NewParallelStateProcessor(config *params.ChainConfig, chain *HeaderChain) *ParallelStateProcessor {
	return &ParallelStateProcessor{
		config: config,
		chain:  chain,
	}
}

func (p *ParallelStateProcessor) Process(block *types.Block, statedb *state.StateDB, cfg vm.Config) (*ProcessResult, error) {
	fmt.Println("ParallelStateProcessor.Process called:", block.NumberU64())
	var (
		header    = block.Header()
		context   vm.BlockContext
		gp        = new(GasPool).AddGas(block.GasLimit())
		signer    = types.MakeSigner(p.config, header.Number, header.Time)
		initialdb = EmptyStatedb

		result *ProcessResult
		err    error
	)

	// Mutate the block and state according to any hard-fork specs
	if p.config.DAOForkSupport && p.config.DAOForkBlock != nil && p.config.DAOForkBlock.Cmp(block.Number()) == 0 {
		misc.ApplyDAOHardFork(statedb)
	}

	preCalPostStart := time.Now()
	statedb.PreComputePostState(block.NumberU64(), runtime.NumCPU()/2)
	PrefetchMergeBALTime += time.Since(preCalPostStart)

	// Apply pre-execution system calls.
	var tracingStateDB = vm.StateDB(statedb)
	if hooks := cfg.Tracer; hooks != nil {
		tracingStateDB = state.NewHookedState(statedb, hooks)
	}
	context = NewEVMBlockContext(header, p.chain, nil)
	evm := vm.NewEVM(context, tracingStateDB, p.config, cfg)

	if beaconRoot := block.BeaconRoot(); beaconRoot != nil {
		ProcessBeaconBlockRoot(*beaconRoot, evm)
	}
	if p.config.IsPrague(block.Number(), block.Time()) || p.config.IsVerkle(block.Number(), block.Time()) {
		ProcessParentBlockHash(block.ParentHash(), evm)
	}

	exeStart := time.Now()
	result, err = p.executeParallel(block, statedb, cfg, gp, signer, context, initialdb)
	ParallelExeTime += time.Since(exeStart)
	return result, err
}

func (p *ParallelStateProcessor) executeParallel(block *types.Block, statedb *state.StateDB, cfg vm.Config, gp *GasPool, signer types.Signer, context vm.BlockContext, initialdb *state.StateDB) (*ProcessResult, error) {
	var (
		receipts    = make(types.Receipts, len(block.Transactions()))
		header      = block.Header()
		blockHash   = block.Hash()
		blockNumber = block.Number()
		allLogs     []*types.Log
		maxLayer    = runtime.NumCPU() / 2
		lenTx       = len(block.Transactions())

		preStateProvider PreStateProvider
		workers          errgroup.Group
	)

	// leave some cpus for prefetching

	switch preStateType {
	case BALPreState:
		{
			workers.SetLimit(runtime.NumCPU() / 2)
		}

	case SeqPreState: // must set workers limit = 1
		{
			workers.SetLimit(1)
			preStatedb := statedb.Copy()
			gpcp := *gp
			preStateProvider = &SequentialPrestateProvider{
				statedb: preStatedb,
				block:   block,
				gp:      &gpcp,
				signer:  signer,
				usedGas: new(uint64),
				evm:     vm.NewEVM(context, preStatedb, p.config, cfg),
			}
		}
	}

	// Parallel executing the transaction
	postEntries := make([][]state.JournalEntry, len(block.Transactions()))

	for i, tx := range block.Transactions() {
		i := i
		workers.Go(func() error {
			var (
				cleanStatedb *state.StateDB
				err          error
			)
			switch preStateType {
			case BALPreState:
				{
					cleanStatedb = initialdb.Copy()
					cleanStatedb.SetTxContext(tx.Hash(), i)
					postSnapshot, postBals := statedb.PostBAL()
					err = cleanStatedb.SetTxBALReader(statedb, blockNumber.Uint64(), maxLayer, lenTx, postSnapshot, postBals)
					if err != nil {
						return err
					}
				}
			case SeqPreState:
				{
					cleanStatedb, err = preStateProvider.PrestateAtIndex(i)
					if err != nil {
						return err
					}
					cleanStatedb.SetTxContext(tx.Hash(), i)
				}
			}

			evm := vm.NewEVM(context, cleanStatedb, p.config, cfg)

			usedGas := new(uint64)
			msg, err := TransactionToMessage(tx, signer, header.BaseFee)
			if err != nil {
				return err
			}
			// todo: handle gp race
			gpcp := *gp
			receipt, entries, err := ApplyTransactionWithParallelEVM(msg, &gpcp, cleanStatedb, blockNumber, blockHash, block.Time(), tx, usedGas, evm)
			if err != nil {
				return err
			}
			receipts[i] = receipt
			postEntries[i] = entries

			return nil
		})
	}

	err := workers.Wait()
	if err != nil {
		return nil, err
	}

	// Merge state changes
	// - Append receipts
	// - Sum usedGas
	// - TODO: validate state changes are the same with bal
	usedGas := uint64(0)

	// set it to avoid read bal post state, -2 is a magic tx number
	statedb.SetTxContext(common.Hash{}, -2)

	for i := range receipts {
		receipt := receipts[i]
		if receipt == nil {
			continue // Skip nil receipts
		}
		receipt.CumulativeGasUsed = usedGas + receipt.GasUsed
		usedGas += receipt.GasUsed
		allLogs = append(allLogs, receipt.Logs...)
		// statedb.MergeState(postEntries[i])
	}

	// Read requests if Prague is enabled.

	var requests [][]byte
	if p.config.IsPrague(block.Number(), block.Time()) {
		requests = [][]byte{}
		// EIP-6110
		if err := ParseDepositLogs(&requests, allLogs, p.config); err != nil {
			return nil, err
		}
		// Commented out EIP-7002, EIP-7251 and p.chain.engine.Finalize for now, since statedb might cause concurrent map writes (journal.go:211) when postState = statedb.Copy()
		if preStateType == SeqPreState {
			evm := vm.NewEVM(context, statedb, p.config, cfg)
			// // EIP-7002
			if err := ProcessWithdrawalQueue(&requests, evm); err != nil {
				return nil, err
			}
			// // EIP-7251
			if err := ProcessConsolidationQueue(&requests, evm); err != nil {
				return nil, err
			}
		}

	}

	// Finalize the block, applying any consensus engine specific extras (e.g. block rewards)
	if preStateType == SeqPreState {
		p.chain.engine.Finalize(p.chain, header, statedb, block.Body())
	}

	return &ProcessResult{
		Receipts: receipts,
		Requests: requests,
		Logs:     allLogs,
		GasUsed:  usedGas,
	}, nil
}

func ApplyTransactionWithParallelEVM(msg *Message, gp *GasPool, statedb *state.StateDB, blockNumber *big.Int, blockHash common.Hash, blockTime uint64, tx *types.Transaction, usedGas *uint64, evm *vm.EVM) (receipt *types.Receipt, entries []state.JournalEntry, err error) {
	if hooks := evm.Config.Tracer; hooks != nil {
		if hooks.OnTxStart != nil {
			hooks.OnTxStart(evm.GetVMContext(), tx, msg.From)
		}
		if hooks.OnTxEnd != nil {
			defer func() { hooks.OnTxEnd(receipt, err) }()
		}
	}
	// Apply the transaction to the current state (included in the env).
	result, err := ApplyMessage(evm, msg, gp)
	if err != nil {
		return nil, nil, err
	}
	// copy changed state
	entries = statedb.JournalEntriesCopy()
	// Update the state with pending changes.
	var root []byte
	if evm.ChainConfig().IsByzantium(blockNumber) {
		evm.StateDB.Finalise(true)
	} else {
		root = statedb.IntermediateRoot(evm.ChainConfig().IsEIP158(blockNumber)).Bytes()
	}
	*usedGas += result.UsedGas

	// Merge the tx-local access event into the "block-local" one, in order to collect
	// all values, so that the witness can be built.
	if statedb.GetTrie() != nil && statedb.GetTrie().IsVerkle() {
		statedb.AccessEvents().Merge(evm.AccessEvents)
	}

	return MakeReceipt(evm, result, statedb, blockNumber, blockHash, blockTime, tx, *usedGas, root), entries, nil
}
