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
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/tracers/logger"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// StateProcessor is a basic Processor, which takes care of transitioning
// state from one point to another.
//
// StateProcessor implements Processor.
type StateProcessor struct {
	config *params.ChainConfig // Chain configuration options
	bc     *BlockChain         // Canonical block chain
	engine consensus.Engine    // Consensus engine used for block rewards
}

// NewStateProcessor initialises a new StateProcessor.
func NewStateProcessor(config *params.ChainConfig, bc *BlockChain, engine consensus.Engine) *StateProcessor {
	return &StateProcessor{
		config: config,
		bc:     bc,
		engine: engine,
	}
}

// Process processes the state changes according to the Ethereum rules by running
// the transaction messages using the statedb and applying any rewards to both
// the processor (coinbase) and any included uncles.
//
// Process returns the receipts and logs accumulated during the process and
// returns the amount of gas that was used in the process. If any of the
// transactions failed to execute due to insufficient gas it will return an error.
func (p *StateProcessor) Process(block *types.Block, statedb *state.StateDB, cfg vm.Config) (types.Receipts, []*types.Log, uint64, error) {
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
	blockContext := NewEVMBlockContext(header, p.bc, nil)
	vmenv := vm.NewEVM(blockContext, vm.TxContext{}, statedb, p.config, cfg)
	// Iterate over and process the individual transactions
	for i, tx := range block.Transactions() {
		msg, err := tx.AsMessage(types.MakeSigner(p.config, header.Number), header.BaseFee)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("could not apply tx %d [%v]: %w", i, tx.Hash().Hex(), err)
		}
		statedb.Prepare(tx.Hash(), i)
		receipt, err := applyTransaction(msg, p.config, p.bc, nil, gp, statedb, blockNumber, blockHash, tx, usedGas, vmenv)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("could not apply tx %d [%v]: %w", i, tx.Hash().Hex(), err)
		}
		receipts = append(receipts, receipt)
		allLogs = append(allLogs, receipt.Logs...)
	}
	// Finalize the block, applying any consensus engine specific extras (e.g. block rewards)
	p.engine.Finalize(p.bc, header, statedb, block.Transactions(), block.Uncles())

	return receipts, allLogs, *usedGas, nil
}

func applyTransaction(msg types.Message, config *params.ChainConfig, bc ChainContext, author *common.Address, gp *GasPool, statedb *state.StateDB, blockNumber *big.Int, blockHash common.Hash, tx *types.Transaction, usedGas *uint64, evm *vm.EVM) (*types.Receipt, error) {
	// Create a new context to be used in the EVM environment.
	txContext := NewEVMTxContext(msg)
	evm.Reset(txContext, statedb)

	// Apply the transaction to the current state (included in the env).
	result, err := ApplyMessage(evm, msg, gp)
	if err != nil {
		return nil, err
	}

	if revert := result.Revert(); revert != nil {
		CacheRevertReason(tx.Hash(), blockHash, revert)
	}

	// Update the state with pending changes.
	var root []byte
	if config.IsByzantium(blockNumber) {
		statedb.Finalise(true)
	} else {
		root = statedb.IntermediateRoot(config.IsEIP158(blockNumber)).Bytes()
	}
	*usedGas += result.UsedGas

	// Create a new receipt for the transaction, storing the intermediate root and gas used
	// by the tx.
	receipt := &types.Receipt{Type: tx.Type(), PostState: root, CumulativeGasUsed: *usedGas}
	if result.Failed() {
		receipt.Status = types.ReceiptStatusFailed
	} else {
		receipt.Status = types.ReceiptStatusSuccessful
	}
	receipt.TxHash = tx.Hash()
	receipt.GasUsed = result.UsedGas

	// If the transaction created a contract, store the creation address in the receipt.
	if msg.To() == nil {
		receipt.ContractAddress = crypto.CreateAddress(evm.TxContext.Origin, tx.Nonce())
	}

	// Set the receipt logs and create the bloom filter.
	receipt.Logs = statedb.GetLogs(tx.Hash(), blockHash)
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})
	receipt.BlockHash = blockHash
	receipt.BlockNumber = blockNumber
	receipt.TransactionIndex = uint(statedb.TxIndex())
	return receipt, err
}

func applyTransactionWithResult(msg types.Message, config *params.ChainConfig, bc ChainContext, author *common.Address, gp *GasPool, statedb *state.StateDB, header *types.Header, msgTx types.Message, usedGas *uint64, evm *vm.EVM, tracer TracerResult) (*types.Receipt, *ExecutionResult, interface{}, error) {
	// Create a new context to be used in the EVM environment.
	txContext := NewEVMTxContext(msg)
	evm.Reset(txContext, statedb)

	// Apply the transaction to the current state (included in the env).
	result, err := ApplyMessage(evm, msg, gp)
	if err != nil {
		return nil, nil, nil, err
	}

	traceResult, err := tracer.GetResult()

	// if err != nil {
	// 	return nil, nil, nil, err
	// }
	// Update the state with pending changes.
	var root []byte
	if config.IsByzantium(header.Number) {
		statedb.Finalise(true)
	} else {
		root = statedb.IntermediateRoot(config.IsEIP158(header.Number)).Bytes()
	}
	*usedGas += result.UsedGas

	// Create a new receipt for the transaction, storing the intermediate root and gas used
	// by the tx.
	receipt := &types.Receipt{Type: 0, PostState: root, CumulativeGasUsed: *usedGas}
	if result.Failed() {
		receipt.Status = types.ReceiptStatusFailed
	} else {
		receipt.Status = types.ReceiptStatusSuccessful
	}
	// receipt.TxHash = tx.Hash()
	receipt.GasUsed = result.UsedGas

	// If the transaction created a contract, store the creation address in the receipt.
	// if msg.To() == nil {
	// 	receipt.ContractAddress = crypto.CreateAddress(evm.TxContext.Origin, tx.Nonce())
	// }

	// Set the receipt logs and create the bloom filter.
	receipt.BlockHash = header.Hash()
	receipt.BlockNumber = header.Number
	receipt.TransactionIndex = uint(statedb.TxIndex())
	return receipt, result, traceResult, err
}

// ApplyTransaction attempts to apply a transaction to the given state database
// and uses the input parameters for its environment. It returns the receipt
// for the transaction, gas used and an error if the transaction failed,
// indicating the block was invalid.
func ApplyTransaction(config *params.ChainConfig, bc ChainContext, author *common.Address, gp *GasPool, statedb *state.StateDB, header *types.Header, tx *types.Transaction, usedGas *uint64, cfg vm.Config) (*types.Receipt, error) {
	msg, err := tx.AsMessage(types.MakeSigner(config, header.Number), header.BaseFee)
	if err != nil {
		return nil, err
	}
	// Create a new context to be used in the EVM environment
	blockContext := NewEVMBlockContext(header, bc, author)
	vmenv := vm.NewEVM(blockContext, vm.TxContext{}, statedb, config, cfg)
	return applyTransaction(msg, config, bc, author, gp, statedb, header.Number, header.Hash(), tx, usedGas, vmenv)
}

func ApplyUnsignedTransactionWithResult(config *params.ChainConfig, bc ChainContext, author *common.Address, gp *GasPool, statedb *state.StateDB, header *types.Header, msg types.Message, usedGas *uint64, cfg vm.Config) (*types.Receipt, *ExecutionResult, interface{}, error) {
	// Create struct logger to get JSON stack traces
	// tracer := logger.NewStructLogger(nil)
	tracer := NewCallTracer(statedb)
	// Create a new context to be used in the EVM environment
	blockContext := NewEVMBlockContext(header, bc, author)
	vmenv := vm.NewEVM(blockContext, vm.TxContext{}, statedb, config, vm.Config{Debug: true, Tracer: tracer, NoBaseFee: true})
	return applyTransactionWithResult(msg, config, bc, author, gp, statedb, header, msg, usedGas, vmenv, tracer)
}

// StructLogRes stores a structured log emitted by the EVM while replaying a
// transaction in debug mode
type StructLogRes struct {
	Pc      uint64             `json:"pc"`
	Op      string             `json:"op"`
	Gas     uint64             `json:"gas"`
	GasCost uint64             `json:"gasCost"`
	Depth   int                `json:"depth"`
	Error   string             `json:"error,omitempty"`
	Stack   *[]string          `json:"stack,omitempty"`
	Memory  *[]string          `json:"memory,omitempty"`
	Storage *map[string]string `json:"storage,omitempty"`
}

// FormatLogs formats EVM returned structured logs for json output
func FormatLogs(logs []logger.StructLog) []StructLogRes {
	formatted := make([]StructLogRes, len(logs))
	for index, trace := range logs {
		formatted[index] = StructLogRes{
			Pc:      trace.Pc,
			Op:      trace.Op.String(),
			Gas:     trace.Gas,
			GasCost: trace.GasCost,
			Depth:   trace.Depth,
			Error:   trace.ErrorString(),
		}
		if trace.Stack != nil {
			stack := make([]string, len(trace.Stack))
			for i, stackValue := range trace.Stack {
				stack[i] = stackValue.Hex()
			}
			formatted[index].Stack = &stack
		}
		if trace.Memory != nil {
			memory := make([]string, 0, (len(trace.Memory)+31)/32)
			for i := 0; i+32 <= len(trace.Memory); i += 32 {
				memory = append(memory, fmt.Sprintf("%x", trace.Memory[i:i+32]))
			}
			formatted[index].Memory = &memory
		}
		if trace.Storage != nil {
			storage := make(map[string]string)
			for i, storageValue := range trace.Storage {
				storage[fmt.Sprintf("%x", i)] = fmt.Sprintf("%x", storageValue)
			}
			formatted[index].Storage = &storage
		}
	}
	return formatted
}

type call struct {
	Type      string         `json:"type"`
	From      common.Address `json:"from"`
	To        common.Address `json:"to"`
	Value     *hexutil.Big   `json:"value,omitempty"`
	Gas       hexutil.Uint64 `json:"gas"`
	GasUsed   hexutil.Uint64 `json:"gasUsed"`
	Input     hexutil.Bytes  `json:"input"`
	Output    hexutil.Bytes  `json:"output"`
	Time      string         `json:"time,omitempty"`
	Calls     []*call        `json:"calls,omitempty"`
	Error     string         `json:"error,omitempty"`
	startTime time.Time
	outOff    uint64
	outLen    uint64
	gasIn     uint64
	gasCost   uint64
}

type TracerResult interface {
	vm.EVMLogger
	GetResult() (interface{}, error)
}

type CallTracer struct {
	callStack []*call
	descended bool
	statedb   *state.StateDB
}

func NewCallTracer(statedb *state.StateDB) TracerResult {
	return &CallTracer{
		callStack: []*call{},
		descended: false,
		statedb:   statedb,
	}
}

func (tracer *CallTracer) i() int {
	return len(tracer.callStack) - 1
}

func (tracer *CallTracer) GetResult() (interface{}, error) {
	return tracer.callStack[0], nil
}

func (tracer *CallTracer) CaptureStart(evm *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	hvalue := hexutil.Big(*value)
	tracer.callStack = []*call{&call{
		From:  from,
		To:    to,
		Value: &hvalue,
		Gas:   hexutil.Uint64(gas),
		Input: hexutil.Bytes(input),
		Calls: []*call{},
	}}
}
func (tracer *CallTracer) CaptureEnd(output []byte, gasUsed uint64, t time.Duration, err error) {
	tracer.callStack[tracer.i()].GasUsed = hexutil.Uint64(gasUsed)
	tracer.callStack[tracer.i()].Time = fmt.Sprintf("%v", t)
	tracer.callStack[tracer.i()].Output = hexutil.Bytes(output)
}

func (tracer *CallTracer) descend(newCall *call) {
	tracer.callStack[tracer.i()].Calls = append(tracer.callStack[tracer.i()].Calls, newCall)
	tracer.callStack = append(tracer.callStack, newCall)
	tracer.descended = true
}

func toAddress(value *uint256.Int) common.Address {
	return common.BytesToAddress(value.Bytes())
}

func (tracer *CallTracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
	// for depth < len(tracer.callStack) {
	//   c := tracer.callStack[tracer.i()]
	//   c.GasUsed = c.Gas - gas
	//   tracer.callStack = tracer.callStack[:tracer.i()]
	// }
	defer func() {
		if r := recover(); r != nil {
			tracer.callStack[tracer.i()].Error = "internal failure"
			log.Warn("Panic during trace. Recovered.", "err", r)
		}
	}()
	if op == vm.CREATE || op == vm.CREATE2 {
		inOff := scope.Stack.Back(1).Uint64()
		inLen := scope.Stack.Back(2).Uint64()
		hvalue := hexutil.Big(*scope.Contract.Value())
		tracer.descend(&call{
			Type:      op.String(),
			From:      scope.Contract.Caller(),
			Input:     scope.Memory.GetCopy(int64(inOff), int64(inLen)),
			gasIn:     gas,
			gasCost:   cost,
			Value:     &hvalue,
			startTime: time.Now(),
		})
		return
	}
	if op == vm.SELFDESTRUCT {
		hvalue := hexutil.Big(*tracer.statedb.GetBalance(scope.Contract.Caller()))
		tracer.descend(&call{
			Type: op.String(),
			From: scope.Contract.Caller(),
			To:   toAddress(scope.Stack.Back(0)),
			// TODO: Is this input correct?
			Input:     scope.Contract.Input,
			Value:     &hvalue,
			gasIn:     gas,
			gasCost:   cost,
			startTime: time.Now(),
		})
		return
	}
	if op == vm.CALL || op == vm.CALLCODE || op == vm.DELEGATECALL || op == vm.STATICCALL {
		toAddress := toAddress(scope.Stack.Back(1))
		if _, isPrecompile := vm.PrecompiledContractsIstanbul[toAddress]; isPrecompile {
			return
		}
		off := 1
		if op == vm.DELEGATECALL || op == vm.STATICCALL {
			off = 0
		}
		inOff := scope.Stack.Back(2 + off).Uint64()
		inLength := scope.Stack.Back(3 + off).Uint64()
		newCall := &call{
			Type:      op.String(),
			From:      scope.Contract.Address(),
			To:        toAddress,
			Input:     scope.Memory.GetCopy(int64(inOff), int64(inLength)),
			gasIn:     gas,
			gasCost:   cost,
			outOff:    scope.Stack.Back(4 + off).Uint64(),
			outLen:    scope.Stack.Back(5 + off).Uint64(),
			startTime: time.Now(),
		}
		if off == 1 {
			value := hexutil.Big(*new(big.Int).SetBytes(scope.Stack.Back(2).Bytes()))
			newCall.Value = &value
		}
		tracer.descend(newCall)
		return
	}
	if tracer.descended {
		if depth >= len(tracer.callStack) {
			tracer.callStack[tracer.i()].Gas = hexutil.Uint64(gas)
		}
		tracer.descended = false
	}
	if op == vm.REVERT {
		tracer.callStack[tracer.i()].Error = "execution reverted"
		return
	}
	if depth == len(tracer.callStack)-1 {
		c := tracer.callStack[tracer.i()]
		// c.Time = fmt.Sprintf("%v", time.Since(c.startTime))
		tracer.callStack = tracer.callStack[:len(tracer.callStack)-1]
		if vm.StringToOp(c.Type) == vm.CREATE || vm.StringToOp(c.Type) == vm.CREATE2 {
			c.GasUsed = hexutil.Uint64(c.gasIn - c.gasCost - gas)
			ret := scope.Stack.Back(0)
			if ret.Uint64() != 0 {
				c.To = common.BytesToAddress(ret.Bytes())
				c.Output = tracer.statedb.GetCode(c.To)
			} else if c.Error == "" {
				c.Error = "internal failure"
			}
		} else {
			c.GasUsed = hexutil.Uint64(c.gasIn - c.gasCost + uint64(c.Gas) - gas)
			ret := scope.Stack.Back(0)
			if ret.Uint64() != 0 {
				c.Output = hexutil.Bytes(scope.Memory.GetCopy(int64(c.outOff), int64(c.outLen)))
			} else if c.Error == "" {
				c.Error = "internal failure"
			}
		}
	}
	return
}
func (tracer *CallTracer) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, memory *vm.ScopeContext, depth int, err error) {
}
func (tracer *CallTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
}
func (tracer *CallTracer) CaptureExit(output []byte, gasUsed uint64, err error) {}
