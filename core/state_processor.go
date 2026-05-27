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
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/internal/telemetry"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/holiman/uint256"
)

// StateProcessor is a basic Processor, which takes care of transitioning
// state from one point to another.
//
// StateProcessor implements Processor.
type StateProcessor struct {
	chain ChainContext // Chain context interface
}

// NewStateProcessor initialises a new StateProcessor.
func NewStateProcessor(chain ChainContext) *StateProcessor {
	return &StateProcessor{
		chain: chain,
	}
}

// chainConfig returns the chain configuration.
func (p *StateProcessor) chainConfig() *params.ChainConfig {
	return p.chain.Config()
}

// Process processes the state changes according to the Ethereum rules by running
// the transaction messages using the statedb and applying any rewards to both
// the processor (coinbase) and any included uncles.
//
// Process returns the receipts and logs accumulated during the process and
// returns the amount of gas that was used in the process. If any of the
// transactions failed to execute due to insufficient gas it will return an error.
func (p *StateProcessor) Process(ctx context.Context, block *types.Block, statedb *state.StateDB, jumpDestCache vm.JumpDestCache, cfg vm.Config) (*ProcessResult, error) {
	var (
		config      = p.chainConfig()
		receipts    = make(types.Receipts, 0, len(block.Transactions()))
		header      = block.Header()
		blockHash   = block.Hash()
		blockNumber = block.Number()
		allLogs     []*types.Log
		gp          = NewGasPool(block.GasLimit())
	)
	var tracingStateDB = vm.StateDB(statedb)
	if hooks := cfg.Tracer; hooks != nil {
		tracingStateDB = state.NewHookedState(statedb, hooks)
	}
	// Mutate the block and state according to any hard-fork specs
	if config.DAOForkSupport && config.DAOForkBlock != nil && config.DAOForkBlock.Cmp(block.Number()) == 0 {
		misc.ApplyDAOHardFork(tracingStateDB)
	}
	var (
		context         = NewEVMBlockContext(header, p.chain, nil)
		signer          = types.MakeSigner(config, header.Number, header.Time)
		evm             = vm.NewEVM(context, tracingStateDB, config, cfg)
		blockAccessList = bal.NewConstructionBlockAccessList()
	)
	defer evm.Release()
	if jumpDestCache != nil {
		evm.SetJumpDestCache(jumpDestCache)
	}

	// Run the pre-execution system calls
	blockAccessList.Merge(PreExecution(ctx, block.BeaconRoot(), block.ParentHash(), config, evm, block.Number(), block.Time()))

	// Iterate over and process the individual transactions
	for i, tx := range block.Transactions() {
		msg, err := TransactionToMessage(tx, signer, header.BaseFee)
		if err != nil {
			return nil, fmt.Errorf("could not apply tx %d [%v]: %w", i, tx.Hash().Hex(), err)
		}
		statedb.SetTxContext(tx.Hash(), i, uint32(i+1))
		_, _, spanEnd := telemetry.StartSpan(ctx, "core.ApplyTransactionWithEVM",
			telemetry.StringAttribute("tx.hash", tx.Hash().Hex()),
			telemetry.Int64Attribute("tx.index", int64(i)),
		)
		receipt, bal, err := ApplyTransactionWithEVM(msg, gp, statedb, blockNumber, blockHash, context.Time, tx, evm)
		if err != nil {
			spanEnd(&err)
			return nil, fmt.Errorf("could not apply tx %d [%v]: %w", i, tx.Hash().Hex(), err)
		}
		receipts = append(receipts, receipt)
		allLogs = append(allLogs, receipt.Logs...)
		blockAccessList.Merge(bal)
		spanEnd(nil)
	}
	requests, bal, err := PostExecution(ctx, config, block.Number(), block.Time(), allLogs, evm, uint32(len(block.Transactions())+1))
	if err != nil {
		return nil, err
	}
	blockAccessList.Merge(bal)

	// Finalize the block, applying any consensus engine specific extras
	// (e.g. block rewards).
	//
	// TODO(rjl493456442) integrate it into the PostExecution.
	p.chain.Engine().Finalize(p.chain, header, tracingStateDB, block.Body(), uint32(len(block.Transactions())+1), blockAccessList)

	return &ProcessResult{
		Receipts: receipts,
		Requests: requests,
		Logs:     allLogs,
		GasUsed:  gp.Used(),
		Bal:      blockAccessList,
	}, nil
}

// PreExecution processes pre-execution system calls.
func PreExecution(ctx context.Context, beaconRoot *common.Hash, parent common.Hash, config *params.ChainConfig, evm *vm.EVM, number *big.Int, time uint64) *bal.ConstructionBlockAccessList {
	_, _, spanEnd := telemetry.StartSpan(ctx, "core.preExecution")
	defer spanEnd(nil)

	var blockAccessList *bal.ConstructionBlockAccessList
	if config.IsAmsterdam(number, time) {
		blockAccessList = bal.NewConstructionBlockAccessList()
	}
	// EIP-4788
	if beaconRoot != nil {
		ProcessBeaconBlockRoot(*beaconRoot, evm, blockAccessList)
	}
	// EIP-2935
	if config.IsPrague(number, time) || config.IsUBT(number, time) {
		ProcessParentBlockHash(parent, evm, blockAccessList)
	}
	return blockAccessList
}

// PostExecution processes post-execution system calls when Prague is enabled.
// If Prague is not activated, it returns null requests to differentiate from
// empty requests.
func PostExecution(ctx context.Context, config *params.ChainConfig, number *big.Int, time uint64, allLogs []*types.Log, evm *vm.EVM, blockAccessIndex uint32) (requests [][]byte, blockAccessList *bal.ConstructionBlockAccessList, err error) {
	_, _, spanEnd := telemetry.StartSpan(ctx, "core.postExecution")
	defer spanEnd(&err)

	if config.IsAmsterdam(number, time) {
		blockAccessList = bal.NewConstructionBlockAccessList()
	}
	// Read requests if Prague is enabled.
	if config.IsPrague(number, time) {
		rules := config.Rules(number, true, time) // IsMerge is always true

		requests = [][]byte{}
		// EIP-6110
		if err := ParseDepositLogs(&requests, allLogs, config); err != nil {
			return nil, nil, fmt.Errorf("failed to parse deposit logs: %w", err)
		}
		// EIP-7002
		if err := ProcessWithdrawalQueue(&requests, rules, evm, blockAccessIndex, blockAccessList); err != nil {
			return nil, nil, fmt.Errorf("failed to process withdrawal queue: %w", err)
		}
		// EIP-7251
		if err := ProcessConsolidationQueue(&requests, rules, evm, blockAccessIndex, blockAccessList); err != nil {
			return nil, nil, fmt.Errorf("failed to process consolidation queue: %w", err)
		}
	}
	return requests, blockAccessList, nil
}

// ApplyTransactionWithEVM attempts to apply a transaction to the given state database
// and uses the input parameters for its environment similar to ApplyTransaction. However,
// this method takes an already created EVM instance as input.
func ApplyTransactionWithEVM(msg *Message, gp *GasPool, statedb *state.StateDB, blockNumber *big.Int, blockHash common.Hash, blockTime uint64, tx *types.Transaction, evm *vm.EVM) (receipt *types.Receipt, bal *bal.ConstructionBlockAccessList, err error) {
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
		bal = evm.StateDB.Finalise(true)
	} else {
		root = statedb.IntermediateRoot(evm.ChainConfig().IsEIP158(blockNumber)).Bytes()
	}
	// Merge the tx-local access event into the "block-local" one, in order to collect
	// all values, so that the witness can be built.
	if statedb.Database().Type().Is(state.TypeUBT) {
		statedb.AccessEvents().Merge(evm.AccessEvents)
	}
	return MakeReceipt(evm, result, statedb, blockNumber, blockHash, blockTime, tx, gp.CumulativeUsed(), root), bal, nil
}

// MakeReceipt generates the receipt object for a transaction given its execution result.
func MakeReceipt(evm *vm.EVM, result *ExecutionResult, statedb *state.StateDB, blockNumber *big.Int, blockHash common.Hash, blockTime uint64, tx *types.Transaction, cumulativeGas uint64, root []byte) *types.Receipt {
	// Create a new receipt for the transaction, storing the intermediate root
	// and gas used by the tx.
	//
	// The cumulative gas used equals the sum of gasUsed across all preceding
	// txs with refunded gas deducted.
	receipt := &types.Receipt{Type: tx.Type(), PostState: root, CumulativeGasUsed: cumulativeGas}
	if result.Failed() {
		receipt.Status = types.ReceiptStatusFailed
	} else {
		receipt.Status = types.ReceiptStatusSuccessful
	}
	receipt.TxHash = tx.Hash()

	// GasUsed = max(tx_gas_used - gas_refund, calldata_floor_gas_cost), unchanged
	// in the Amsterdam fork.
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
// for the transaction and an error if the transaction failed,
// indicating the block was invalid.
func ApplyTransaction(evm *vm.EVM, gp *GasPool, statedb *state.StateDB, header *types.Header, tx *types.Transaction) (*types.Receipt, *bal.ConstructionBlockAccessList, error) {
	msg, err := TransactionToMessage(tx, types.MakeSigner(evm.ChainConfig(), header.Number, header.Time), header.BaseFee)
	if err != nil {
		return nil, nil, err
	}
	// Create a new context to be used in the EVM environment
	return ApplyTransactionWithEVM(msg, gp, statedb, header.Number, header.Hash(), header.Time, tx, evm)
}

// ProcessBeaconBlockRoot applies the EIP-4788 system call to the beacon block root
// contract. This method is exported to be used in tests.
func ProcessBeaconBlockRoot(beaconRoot common.Hash, evm *vm.EVM, blockAccessList *bal.ConstructionBlockAccessList) {
	if tracer := evm.Config.Tracer; tracer != nil {
		onSystemCallStart(tracer, evm.GetVMContext())
		if tracer.OnSystemCallEnd != nil {
			defer tracer.OnSystemCallEnd()
		}
	}
	msg := &Message{
		From:      params.SystemAddress,
		GasLimit:  30_000_000,
		GasPrice:  uint256.NewInt(0),
		GasFeeCap: uint256.NewInt(0),
		GasTipCap: uint256.NewInt(0),
		To:        &params.BeaconRootsAddress,
		Data:      beaconRoot[:],
	}
	evm.SetTxContext(NewEVMTxContext(msg))
	evm.StateDB.Prepare(evm.GetRules(), common.Address{}, common.Address{}, nil, nil, nil)
	evm.StateDB.SetTxContext(common.Hash{}, 0, 0)
	evm.StateDB.AddAddressToAccessList(params.BeaconRootsAddress)
	_, _, _ = evm.Call(msg.From, *msg.To, msg.Data, vm.NewGasBudget(30_000_000), common.U2560)
	if evm.StateDB.AccessEvents() != nil {
		evm.StateDB.AccessEvents().Merge(evm.AccessEvents)
	}
	blockAccessList.Merge(evm.StateDB.Finalise(true))
}

// ProcessParentBlockHash stores the parent block hash in the history storage contract
// as per EIP-2935/7709.
func ProcessParentBlockHash(prevHash common.Hash, evm *vm.EVM, blockAccessList *bal.ConstructionBlockAccessList) {
	if tracer := evm.Config.Tracer; tracer != nil {
		onSystemCallStart(tracer, evm.GetVMContext())
		if tracer.OnSystemCallEnd != nil {
			defer tracer.OnSystemCallEnd()
		}
	}
	msg := &Message{
		From:      params.SystemAddress,
		GasLimit:  30_000_000,
		GasPrice:  uint256.NewInt(0),
		GasFeeCap: uint256.NewInt(0),
		GasTipCap: uint256.NewInt(0),
		To:        &params.HistoryStorageAddress,
		Data:      prevHash.Bytes(),
	}
	evm.SetTxContext(NewEVMTxContext(msg))
	evm.StateDB.Prepare(evm.GetRules(), common.Address{}, common.Address{}, nil, nil, nil)
	evm.StateDB.SetTxContext(common.Hash{}, 0, 0)
	evm.StateDB.AddAddressToAccessList(params.HistoryStorageAddress)
	_, _, err := evm.Call(msg.From, *msg.To, msg.Data, vm.NewGasBudget(30_000_000), common.U2560)
	if err != nil {
		panic(err)
	}
	if evm.StateDB.AccessEvents() != nil {
		evm.StateDB.AccessEvents().Merge(evm.AccessEvents)
	}
	blockAccessList.Merge(evm.StateDB.Finalise(true))
}

// ProcessWithdrawalQueue calls the EIP-7002 withdrawal queue contract.
// It returns the opaque request data returned by the contract.
func ProcessWithdrawalQueue(requests *[][]byte, rules params.Rules, evm *vm.EVM, blockAccessIndex uint32, blockAccessList *bal.ConstructionBlockAccessList) error {
	return processRequestsSystemCall(requests, rules, evm, 0x01, params.WithdrawalQueueAddress, blockAccessIndex, blockAccessList)
}

// ProcessConsolidationQueue calls the EIP-7251 consolidation queue contract.
// It returns the opaque request data returned by the contract.
func ProcessConsolidationQueue(requests *[][]byte, rules params.Rules, evm *vm.EVM, blockAccessIndex uint32, blockAccessList *bal.ConstructionBlockAccessList) error {
	return processRequestsSystemCall(requests, rules, evm, 0x02, params.ConsolidationQueueAddress, blockAccessIndex, blockAccessList)
}

func processRequestsSystemCall(requests *[][]byte, rules params.Rules, evm *vm.EVM, requestType byte, addr common.Address, blockAccessIndex uint32, blockAccessList *bal.ConstructionBlockAccessList) error {
	if tracer := evm.Config.Tracer; tracer != nil {
		onSystemCallStart(tracer, evm.GetVMContext())
		if tracer.OnSystemCallEnd != nil {
			defer tracer.OnSystemCallEnd()
		}
	}
	msg := &Message{
		From:      params.SystemAddress,
		GasLimit:  30_000_000,
		GasPrice:  uint256.NewInt(0),
		GasFeeCap: uint256.NewInt(0),
		GasTipCap: uint256.NewInt(0),
		To:        &addr,
	}
	evm.SetTxContext(NewEVMTxContext(msg))
	evm.StateDB.Prepare(rules, common.Address{}, common.Address{}, nil, nil, nil)
	evm.StateDB.SetTxContext(common.Hash{}, 0, blockAccessIndex)
	evm.StateDB.AddAddressToAccessList(addr)
	ret, _, err := evm.Call(msg.From, *msg.To, msg.Data, vm.NewGasBudget(30_000_000), common.U2560)
	if evm.StateDB.AccessEvents() != nil {
		evm.StateDB.AccessEvents().Merge(evm.AccessEvents)
	}
	bal := evm.StateDB.Finalise(true)
	if err != nil {
		return fmt.Errorf("system call failed to execute: %v", err)
	}
	blockAccessList.Merge(bal)

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

// AssembleBlock finalizes the state and assembles the block with provided
// body and receipts.
func AssembleBlock(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, body *types.Body, receipts []*types.Receipt, blockAccessList *bal.ConstructionBlockAccessList) *types.Block {
	// Assign the post-transition state root
	header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))

	if !chain.Config().IsAmsterdam(header.Number, header.Time) {
		return types.NewBlock(header, body, receipts, trie.NewStackTrie(nil))
	}
	// Assign the BlockAccessListHash if Amsterdam has been enabled
	bal := blockAccessList.ToEncodingObj()
	balHash := bal.Hash()
	header.BlockAccessListHash = &balHash
	return types.NewBlock(header, body, receipts, trie.NewStackTrie(nil)).WithAccessListUnsafe(bal)
}
