// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"cmp"
	context2 "context"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"math/big"
	"slices"
	"time"
)

// StateProcessor is a basic Processor, which takes care of transitioning
// state from one point to another.
//
// StateProcessor implements Processor.
type StateProcessor struct {
	config *params.ChainConfig // Chain configuration options
	chain  *HeaderChain        // Canonical header chain
}

// NewStateProcessor initialises a new StateProcessor.
func NewStateProcessor(config *params.ChainConfig, chain *HeaderChain) *StateProcessor {
	return &StateProcessor{
		config: config,
		chain:  chain,
	}
}

type ProcessResultWithMetrics struct {
	ProcessResult   *ProcessResult
	PreProcessTime  time.Duration
	PostProcessTime time.Duration
	RootCalcTime    time.Duration
	ExecTime        time.Duration
	Error           error
}

// Process processes the state changes according to the Ethereum rules by running
// the transaction messages using the statedb and applying any rewards to both
// the processor (coinbase) and any included uncles.
//
// Process returns the receipts and logs accumulated during the process and
// returns the amount of gas that was used in the process. If any of the
// transactions failed to execute due to insufficient gas it will return an error.
func (p *StateProcessor) Process(block *types.Block, statedb *state.StateDB, cfg vm.Config) (*ProcessResult, error) {
	var (
		receipts    types.Receipts
		usedGas     = new(uint64)
		header      = block.Header()
		blockHash   = block.Hash()
		blockNumber = block.Number()
		allLogs     []*types.Log
		gp          = new(GasPool).AddGas(block.GasLimit())
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

	// Iterate over and process the individual transactions
	for i, tx := range block.Transactions() {
		msg, err := TransactionToMessage(tx, signer, header.BaseFee)
		if err != nil {
			return nil, fmt.Errorf("could not apply tx %d [%v]: %w", i, tx.Hash().Hex(), err)
		}

		sender, _ := types.Sender(signer, tx)
		statedb.SetTxSender(sender)
		statedb.SetTxContext(tx.Hash(), i)

		_, receipt, err := ApplyTransactionWithEVM(msg, gp, statedb, blockNumber, blockHash, context.Time, tx, usedGas, evm, nil)
		if err != nil {
			return nil, fmt.Errorf("could not apply tx %d [%v]: %w", i, tx.Hash().Hex(), err)
		}
		receipts = append(receipts, receipt)
		allLogs = append(allLogs, receipt.Logs...)
	}

	if statedb.BlockAccessList() != nil {
		statedb.SetAccessListIndex(len(block.Transactions()) + 1)
	}

	// Read requests if Prague is enabled.
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
	p.chain.engine.Finalize(p.chain, header, tracingStateDB, block.Body())

	return &ProcessResult{
		Receipts: receipts,
		Requests: requests,
		Logs:     allLogs,
		GasUsed:  *usedGas,
	}, nil
}

func (p *StateProcessor) calcStateDiffs(evm *vm.EVM, block *types.Block, txPrestate *state.StateDB) (totalDiff *bal.StateDiff, txDiffs []*bal.StateDiff) {
	prestateDiff := txPrestate.GetStateDiff()
	// create a number of diffs (one for each worker goroutine)
	txDiffIt := bal.NewIterator(block.Body().AccessList, len(block.Transactions()))
	return txDiffIt.BuildStateDiffs(prestateDiff, uint16(len(block.Transactions()))+1)
}

func (p *StateProcessor) ProcessWithAccessList(block *types.Block, statedb *state.StateDB, cfg vm.Config) (chan *ProcessResultWithMetrics, error) {
	var (
		header      = block.Header()
		blockHash   = block.Hash()
		blockNumber = block.Number()
		resCh       = make(chan *ProcessResultWithMetrics)
		requests    [][]byte
		signer      = types.MakeSigner(p.config, header.Number, header.Time)
		ctx, cancel = context2.WithCancel(context2.Background())
	)

	type txExecResult struct {
		idx        int
		netGasUsed uint64 // accounts for the net gas used (refunds accounted for)
		receipt    *types.Receipt
		err        error
	}

	txResCh := make(chan txExecResult)
	rootCalcErrCh := make(chan error) // used for communicating if the state root calculation doesn't match the reported root
	pStart := time.Now()
	var (
		tPreprocess  time.Duration
		tVerifyStart time.Time
		tVerify      time.Duration
		tExecStart   time.Time
		tExec        time.Duration
		tPostprocess time.Duration
	)

	// called by resultHandler when all transactions have successfully executed.
	// performs post-tx state transition (system contracts and withdrawals)
	// and calculates the ProcessResult, returning it to be sent on resCh
	// by resultHandler
	prepareExecResult := func(postTxState *state.StateDB, expectedStateDiff *bal.StateDiff, receipts types.Receipts) *ProcessResult {
		tExec = time.Since(tExecStart)
		tPostprocessStart := time.Now()
		var tracingStateDB = vm.StateDB(postTxState)
		if hooks := cfg.Tracer; hooks != nil {
			tracingStateDB = state.NewHookedState(postTxState, hooks)
		}
		context := NewEVMBlockContext(header, p.chain, nil)
		evm := vm.NewEVM(context, tracingStateDB, p.config, cfg)

		// 1. order the receipts by tx index
		// 2. correctly calculate the cumulative gas used per receipt, returning bad block error if it goes over the allowed
		slices.SortFunc(receipts, func(a, b *types.Receipt) int {
			return cmp.Compare(a.TransactionIndex, b.TransactionIndex)
		})

		var cumGasUsed uint64
		var allLogs []*types.Log
		for _, receipt := range receipts {
			receipt.CumulativeGasUsed = cumGasUsed + receipt.GasUsed
			cumGasUsed += receipt.GasUsed
			if receipt.CumulativeGasUsed > header.GasLimit {
				return &ProcessResult{Error: fmt.Errorf("gas limit exceeded")}
			}
			allLogs = append(allLogs, receipt.Logs...)
		}

		// Read requests if Prague is enabled.
		if p.config.IsPrague(block.Number(), block.Time()) {
			requests = [][]byte{}
			// EIP-6110
			if err := ParseDepositLogs(&requests, allLogs, p.config); err != nil {
				return &ProcessResult{
					Error: err,
				}
			}

			// EIP-7002
			if err := ProcessWithdrawalQueue(&requests, evm); err != nil {
				return &ProcessResult{
					Error: err,
				}
			}
			// EIP-7251
			if err := ProcessConsolidationQueue(&requests, evm); err != nil {
				return &ProcessResult{
					Error: err,
				}
			}
		}
		// Finalize the block, applying any consensus engine specific extras (e.g. block rewards)
		p.chain.engine.Finalize(p.chain, header, tracingStateDB, block.Body())
		// invoke Finalise so that withdrawals are accounted for in the state diff
		postTxState.Finalise(true)

		if err := bal.ValidateStateDiff(expectedStateDiff, postTxState.GetStateDiff()); err != nil {
			return &ProcessResult{
				Error: fmt.Errorf("post-transaction-execution state transition produced a different diff that what was reported in the BAL"),
			}
		}

		tPostprocess = time.Since(tPostprocessStart)

		return &ProcessResult{
			Receipts: receipts,
			Requests: requests,
			Logs:     allLogs,
			GasUsed:  cumGasUsed,
		}
	}
	resultHandler := func(expectedDiff *bal.StateDiff, postTxState *state.StateDB) {
		defer cancel()
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
								cancel()
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

		execResults := prepareExecResult(postTxState, expectedDiff, receipts)
		err := <-rootCalcErrCh
		if err != nil {
			resCh <- &ProcessResultWithMetrics{ProcessResult: &ProcessResult{Error: err}}
		} else {
			resCh <- &ProcessResultWithMetrics{
				ProcessResult:   execResults,
				PreProcessTime:  tPreprocess,
				PostProcessTime: tPostprocess,
				ExecTime:        tExec,
				RootCalcTime:    tVerify,
			}
		}
	}

	calcAndVerifyRoot := func(postState *state.StateDB, block *types.Block, resCh chan<- error) {
		tVerifyStart = time.Now()
		root := postState.IntermediateRoot(false)
		tVerify = time.Since(tVerifyStart)

		if root != block.Root() {
			resCh <- fmt.Errorf("state root mismatch. local: %x. remote: %x", root, block.Root())
		} else {
			resCh <- nil
		}
	}

	// executes single transaction, validating the computed diff against the BAL
	// and forwarding the txExecResult to be consumed by resultHandler
	execTx := func(ctx context2.Context, tx *types.Transaction, idx int, db *state.StateDB, expectedDiff *bal.StateDiff) {
		// if an error with another transaction rendered the block invalid, don't proceed with executing this one
		// TODO: also interrupt any currently-executing transactions if one failed.
		select {
		case <-ctx.Done():
			txResCh <- txExecResult{err: ctx.Err()}
			return
		default:
		}
		var tracingStateDB = vm.StateDB(db)
		if hooks := cfg.Tracer; hooks != nil {
			tracingStateDB = state.NewHookedState(db, hooks)
		}
		context := NewEVMBlockContext(header, p.chain, nil)
		evm := vm.NewEVM(context, tracingStateDB, p.config, cfg)

		msg, err := TransactionToMessage(tx, signer, header.BaseFee)
		if err != nil {
			err = fmt.Errorf("could not apply tx %d [%v]: %w", idx, tx.Hash().Hex(), err)
			txResCh <- txExecResult{err: err}
			return
		}
		sender, _ := types.Sender(signer, tx)
		db.SetTxSender(sender)
		db.SetTxContext(tx.Hash(), idx)

		evm.StateDB = db // TODO: unsure if need to set this here since the evm should maintain a reference to the db but I recall that adding this fixed some broken tests
		gp := new(GasPool)
		gp.SetGas(block.GasLimit())
		var gasUsed uint64
		computedDiff, receipt, err := ApplyTransactionWithEVM(msg, gp, db, blockNumber, blockHash, context.Time, tx, &gasUsed, evm, nil)
		if err != nil {
			err := fmt.Errorf("could not apply tx %d [%v]: %w", idx, tx.Hash().Hex(), err)
			txResCh <- txExecResult{err: err}
			return
		}

		if err := bal.ValidateStateDiff(expectedDiff, computedDiff); err != nil {
			txResCh <- txExecResult{err: err}
			return
		}

		txResCh <- txExecResult{
			idx:        idx,
			receipt:    receipt,
			netGasUsed: gp.Gas(),
		}
		return
	}

	// Mutate the block and state according to any hard-fork specs
	if p.config.DAOForkSupport && p.config.DAOForkBlock != nil && p.config.DAOForkBlock.Cmp(block.Number()) == 0 {
		misc.ApplyDAOHardFork(statedb)
	}
	var (
		context vm.BlockContext
	)

	// Apply pre-execution system calls.
	var tracingStateDB = vm.StateDB(statedb)
	if hooks := cfg.Tracer; hooks != nil {
		tracingStateDB = state.NewHookedState(statedb, hooks)
	}
	context = NewEVMBlockContext(header, p.chain, nil)
	evm := vm.NewEVM(context, tracingStateDB, p.config, cfg)

	// process beacon-root and parent block system contracts.
	// do not include the storage writes in the BAL:
	// * beacon root will be provided as a standalone field in the BAL
	// * parent block hash is already in the header field of the block

	blockStateDiff, stateDiffs := p.calcStateDiffs(evm, block, statedb)
	preTxDiff := stateDiffs[0]

	if beaconRoot := block.BeaconRoot(); beaconRoot != nil {
		ProcessBeaconBlockRoot(*beaconRoot, evm)
	}
	if p.config.IsPrague(block.Number(), block.Time()) || p.config.IsVerkle(block.Number(), block.Time()) {
		ProcessParentBlockHash(block.ParentHash(), evm)
	}

	computedDiff := statedb.GetStateDiff()
	if err := bal.ValidateStateDiff(preTxDiff, computedDiff); err != nil {
		return nil, err
	}
	statedb.Finalise(true)

	// Iterate over and process the individual transactions
	for i := range block.Transactions() {
		statedb.ApplyDiff(stateDiffs[i+1])
		statedb.Finalise(true)
	}
	tPreprocess = time.Since(pStart)

	tExecStart = time.Now()
	for i, tx := range block.Transactions() {
		go execTx(ctx, tx, i, statedb.Copy(), stateDiffs[i+1])
	}

	go resultHandler(blockStateDiff, statedb.Copy())

	// it's possible that there isn't a post-tx-execution state diff
	// if there are no withdrawals or consolidations
	if len(stateDiffs) == len(block.Transactions())+2 {
		statedb.ApplyDiff(stateDiffs[len(block.Transactions())+1])
		statedb.Finalise(true)
	}
	go calcAndVerifyRoot(statedb, block, rootCalcErrCh)

	return resCh, nil
}

// ApplyTransactionWithEVM attempts to apply a transaction to the given state database
// and uses the input parameters for its environment similar to ApplyTransaction. However,
// this method takes an already created EVM instance as input.
func ApplyTransactionWithEVM(msg *Message, gp *GasPool, statedb *state.StateDB, blockNumber *big.Int, blockHash common.Hash, blockTime uint64, tx *types.Transaction, usedGas *uint64, evm *vm.EVM, balDiff *bal.StateDiff) (diff *bal.StateDiff, receipt *types.Receipt, err error) {
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

	// Update the state with pending changes.
	var root []byte
	if evm.ChainConfig().IsByzantium(blockNumber) {
		diff = evm.StateDB.Finalise(true)
	} else {
		root = statedb.IntermediateRoot(evm.ChainConfig().IsEIP158(blockNumber)).Bytes()
	}
	*usedGas += result.UsedGas

	// Merge the tx-local access event into the "block-local" one, in order to collect
	// all values, so that the witness can be built.
	if statedb.Database().TrieDB().IsVerkle() {
		statedb.AccessEvents().Merge(evm.AccessEvents)
	}
	return diff, MakeReceipt(evm, result, statedb, blockNumber, blockHash, blockTime, tx, *usedGas, root), nil
}

// MakeReceipt generates the receipt object for a transaction given its execution result.
func MakeReceipt(evm *vm.EVM, result *ExecutionResult, statedb *state.StateDB, blockNumber *big.Int, blockHash common.Hash, blockTime uint64, tx *types.Transaction, usedGas uint64, root []byte) *types.Receipt {
	// Create a new receipt for the transaction, storing the intermediate root and gas used
	// by the tx.
	receipt := &types.Receipt{Type: tx.Type(), PostState: root, CumulativeGasUsed: usedGas}
	if result.Failed() {
		receipt.Status = types.ReceiptStatusFailed
	} else {
		receipt.Status = types.ReceiptStatusSuccessful
	}
	receipt.TxHash = tx.Hash()
	receipt.GasUsed = result.UsedGas

	if tx.Type() == types.BlobTxType {
		receipt.BlobGasUsed = uint64(len(tx.BlobHashes()) * params.BlobTxBlobGasPerBlob)
		receipt.BlobGasPrice = evm.Context.BlobBaseFee
	}

	// If the transaction created a contract, store the creation address in the receipt.
	if tx.To() == nil {
		receipt.ContractAddress = crypto.CreateAddress(evm.TxContext.Origin, tx.Nonce())
	}

	// Set the receipt logs and create the bloom filter.
	receipt.Logs = statedb.GetLogs(tx.Hash(), blockNumber.Uint64(), blockHash, blockTime)
	receipt.Bloom = types.CreateBloom(receipt)
	receipt.BlockHash = blockHash
	receipt.BlockNumber = blockNumber
	receipt.TransactionIndex = uint(statedb.TxIndex())
	return receipt
}

// ApplyTransaction attempts to apply a transaction to the given state database
// and uses the input parameters for its environment. It returns the receipt
// for the transaction, gas used and an error if the transaction failed,
// indicating the block was invalid.
func ApplyTransaction(evm *vm.EVM, gp *GasPool, statedb *state.StateDB, header *types.Header, tx *types.Transaction, usedGas *uint64) (*types.Receipt, error) {
	msg, err := TransactionToMessage(tx, types.MakeSigner(evm.ChainConfig(), header.Number, header.Time), header.BaseFee)
	if err != nil {
		return nil, err
	}
	// Create a new context to be used in the EVM environment
	_, receipts, err := ApplyTransactionWithEVM(msg, gp, statedb, header.Number, header.Hash(), header.Time, tx, usedGas, evm, nil)
	return receipts, err
}

// ProcessBeaconBlockRoot applies the EIP-4788 system call to the beacon block root
// contract. This method is exported to be used in tests.
func ProcessBeaconBlockRoot(beaconRoot common.Hash, evm *vm.EVM) {
	if tracer := evm.Config.Tracer; tracer != nil {
		onSystemCallStart(tracer, evm.GetVMContext())
		if tracer.OnSystemCallEnd != nil {
			defer tracer.OnSystemCallEnd()
		}
	}
	msg := &Message{
		From:      params.SystemAddress,
		GasLimit:  30_000_000,
		GasPrice:  common.Big0,
		GasFeeCap: common.Big0,
		GasTipCap: common.Big0,
		To:        &params.BeaconRootsAddress,
		Data:      beaconRoot[:],
	}
	evm.SetTxContext(NewEVMTxContext(msg))
	evm.StateDB.AddAddressToAccessList(params.BeaconRootsAddress)
	_, _, _ = evm.Call(msg.From, *msg.To, msg.Data, 30_000_000, common.U2560)
	evm.StateDB.Finalise(true)
}

// ProcessParentBlockHash stores the parent block hash in the history storage contract
// as per EIP-2935/7709.
func ProcessParentBlockHash(prevHash common.Hash, evm *vm.EVM) {
	if tracer := evm.Config.Tracer; tracer != nil {
		onSystemCallStart(tracer, evm.GetVMContext())
		if tracer.OnSystemCallEnd != nil {
			defer tracer.OnSystemCallEnd()
		}
	}
	msg := &Message{
		From:      params.SystemAddress,
		GasLimit:  30_000_000,
		GasPrice:  common.Big0,
		GasFeeCap: common.Big0,
		GasTipCap: common.Big0,
		To:        &params.HistoryStorageAddress,
		Data:      prevHash.Bytes(),
	}
	evm.SetTxContext(NewEVMTxContext(msg))
	evm.StateDB.AddAddressToAccessList(params.HistoryStorageAddress)
	_, _, err := evm.Call(msg.From, *msg.To, msg.Data, 30_000_000, common.U2560)
	if err != nil {
		panic(err)
	}
	if evm.StateDB.AccessEvents() != nil {
		evm.StateDB.AccessEvents().Merge(evm.AccessEvents)
	}
	evm.StateDB.Finalise(true)
}

// ProcessWithdrawalQueue calls the EIP-7002 withdrawal queue contract.
// It returns the opaque request data returned by the contract.
func ProcessWithdrawalQueue(requests *[][]byte, evm *vm.EVM) error {
	return processRequestsSystemCall(requests, evm, 0x01, params.WithdrawalQueueAddress)
}

// ProcessConsolidationQueue calls the EIP-7251 consolidation queue contract.
// It returns the opaque request data returned by the contract.
func ProcessConsolidationQueue(requests *[][]byte, evm *vm.EVM) error {
	return processRequestsSystemCall(requests, evm, 0x02, params.ConsolidationQueueAddress)
}

func processRequestsSystemCall(requests *[][]byte, evm *vm.EVM, requestType byte, addr common.Address) error {
	if tracer := evm.Config.Tracer; tracer != nil {
		onSystemCallStart(tracer, evm.GetVMContext())
		if tracer.OnSystemCallEnd != nil {
			defer tracer.OnSystemCallEnd()
		}
	}
	msg := &Message{
		From:      params.SystemAddress,
		GasLimit:  30_000_000,
		GasPrice:  common.Big0,
		GasFeeCap: common.Big0,
		GasTipCap: common.Big0,
		To:        &addr,
	}
	evm.SetTxContext(NewEVMTxContext(msg))
	evm.StateDB.AddAddressToAccessList(addr)
	ret, _, err := evm.Call(msg.From, *msg.To, msg.Data, 30_000_000, common.U2560)
	evm.StateDB.Finalise(true)
	if err != nil {
		return fmt.Errorf("system call failed to execute: %v", err)
	}
	if len(ret) == 0 {
		return nil // skip empty output
	}
	// Append prefixed requestsData to the requests list.
	requestsData := make([]byte, len(ret)+1)
	requestsData[0] = requestType
	copy(requestsData[1:], ret)
	*requests = append(*requests, requestsData)
	return nil
}

var depositTopic = common.HexToHash("0x649bbc62d0e31342afea4e5cd82d4049e7e1ee912fc0889aa790803be39038c5")

// ParseDepositLogs extracts the EIP-6110 deposit values from logs emitted by
// BeaconDepositContract.
func ParseDepositLogs(requests *[][]byte, logs []*types.Log, config *params.ChainConfig) error {
	deposits := make([]byte, 1) // note: first byte is 0x00 (== deposit request type)
	for _, log := range logs {
		if log.Address == config.DepositContractAddress && len(log.Topics) > 0 && log.Topics[0] == depositTopic {
			request, err := types.DepositLogToRequest(log.Data)
			if err != nil {
				return fmt.Errorf("unable to parse deposit data: %v", err)
			}
			deposits = append(deposits, request...)
		}
	}
	if len(deposits) > 1 {
		*requests = append(*requests, deposits)
	}
	return nil
}

func onSystemCallStart(tracer *tracing.Hooks, ctx *tracing.VMContext) {
	if tracer.OnSystemCallStartV2 != nil {
		tracer.OnSystemCallStartV2(ctx)
	} else if tracer.OnSystemCallStart != nil {
		tracer.OnSystemCallStart()
	}
}
