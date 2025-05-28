package core

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
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
	fmt.Println("ParallelStateProcessor.Process called")
	var (
		header = block.Header()
		gp     = new(GasPool).AddGas(block.GasLimit())
	)

	// Mutate the block and state according to any hard-fork specs
	if p.config.DAOForkSupport && p.config.DAOForkBlock != nil && p.config.DAOForkBlock.Cmp(block.Number()) == 0 {
		misc.ApplyDAOHardFork(statedb)
	}
	var (
		context vm.BlockContext
		signer  = types.MakeSigner(p.config, header.Number, header.Time)
	)

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

	return p.executeParallel(block, statedb, &context, cfg, gp, signer)
}

func (p *ParallelStateProcessor) executeParallel(block *types.Block, statedb *state.StateDB, blockContext *vm.BlockContext, cfg vm.Config, gp *GasPool, signer types.Signer) (*ProcessResult, error) {
	var (
		receipts      = make(types.Receipts, len(block.Transactions()))
		header        = block.Header()
		blockHash     = block.Hash()
		blockNumber   = block.Number()
		allLogs       []*types.Log
		wg            sync.WaitGroup
		preStatedb    = statedb.Copy()
		sequentialEvm = vm.NewEVM(*blockContext, preStatedb, p.config, cfg)
		seqUsedGas    = new(uint64)
	)
	// Fetch prestate for each tx

	// Parallel executing the transaction
	postStates := make([]*state.StateDB, len(block.Transactions()))
	postEntries := make([][]state.JournalEntry, len(block.Transactions()))
	for i, tx := range block.Transactions() {
		postStates[i] = preStatedb.Copy() // Copy the state for each transaction
		cleanStatedb := postStates[i]
		i := i

		wg.Add(1)
		go func() {
			defer wg.Done()

			usedGas := new(uint64)
			msg, err := TransactionToMessage(tx, signer, header.BaseFee)
			// todo: handle error: break all routines and return error
			if err != nil {
				fmt.Printf("could not apply tx %d [%v]: %v", i, tx.Hash().Hex(), err)
			}
			cleanStatedb.SetTxContext(tx.Hash(), i)

			evm := vm.NewEVM(*blockContext, cleanStatedb, p.config, cfg)

			receipt, entries, err := ApplyTransactionWithParallelEVM(msg, gp, cleanStatedb, blockNumber, blockHash, tx, usedGas, evm)
			if err != nil {
				fmt.Printf("could not apply parallel tx %d [%v]: %v", i, tx.Hash().Hex(), err)
			}
			receipts[i] = receipt
			postEntries[i] = entries
		}()

		// execute the transaction again to simulate the state changes
		msg, err := TransactionToMessage(tx, signer, header.BaseFee)
		if err != nil {
			return nil, fmt.Errorf("could not apply tx to msg %d [%v]: %w", i, tx.Hash().Hex(), err)
		}
		preStatedb.SetTxContext(tx.Hash(), i)

		_, err = ApplyTransactionWithEVM(msg, gp, preStatedb, blockNumber, blockHash, tx, seqUsedGas, sequentialEvm)
		if err != nil {
			return nil, fmt.Errorf("could not apply tx %d [%v]: %w", i, tx.Hash().Hex(), err)
		}
	}

	wg.Wait()
	// Merge state changes
	// - Append receipts
	// - Sum usedGas
	// - Collect state state changes: simple overwrite
	// - Ommit preimages for now
	usedGas := uint64(0)
	for i, receipt := range receipts {
		if receipt == nil {
			continue // Skip nil receipts
		}
		receipt.CumulativeGasUsed = usedGas + receipt.GasUsed
		usedGas += receipt.GasUsed
		allLogs = append(allLogs, receipt.Logs...)
		statedb.MergeState(postEntries[i])
	}

	// Read requests if Prague is enabled.
	evm := vm.NewEVM(*blockContext, statedb, p.config, cfg)
	var requests [][]byte
	if p.config.IsPrague(block.Number(), block.Time()) {
		requests = [][]byte{}
		// EIP-6110
		if err := ParseDepositLogs(&requests, allLogs, p.config); err != nil {
			return nil, err
		}
		// EIP-7002
		if err := ProcessWithdrawalQueue(&requests, evm); err != nil {
			return nil, err
		}
		// EIP-7251
		if err := ProcessConsolidationQueue(&requests, evm); err != nil {
			return nil, err
		}
	}

	// Finalize the block, applying any consensus engine specific extras (e.g. block rewards)
	p.chain.engine.Finalize(p.chain, header, statedb, block.Body())

	return &ProcessResult{
		Receipts: receipts,
		Requests: requests,
		Logs:     allLogs,
		GasUsed:  usedGas,
	}, nil
}

func ApplyTransactionWithParallelEVM(msg *Message, gp *GasPool, statedb *state.StateDB, blockNumber *big.Int, blockHash common.Hash, tx *types.Transaction, usedGas *uint64, evm *vm.EVM) (receipt *types.Receipt, entries []state.JournalEntry, err error) {
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
	if statedb.GetTrie().IsVerkle() {
		statedb.AccessEvents().Merge(evm.AccessEvents)
	}

	return MakeReceipt(evm, result, statedb, blockNumber, blockHash, tx, *usedGas, root), entries, nil
}
