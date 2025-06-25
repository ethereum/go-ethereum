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
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/consensus"
	"github.com/scroll-tech/go-ethereum/consensus/misc"
	"github.com/scroll-tech/go-ethereum/core/state"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/core/vm"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/metrics"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/scroll-tech/go-ethereum/rollup/fees"
)

var (
	processorBlockTransactionGauge = metrics.NewRegisteredGauge("processor/block/transactions", nil)
	processBlockTimer              = metrics.NewRegisteredTimer("processor/block/process", nil)
	finalizeBlockTimer             = metrics.NewRegisteredTimer("processor/block/finalize", nil)
	applyTransactionTimer          = metrics.NewRegisteredTimer("processor/tx/apply", nil)
	applyMessageTimer              = metrics.NewRegisteredTimer("processor/tx/msg/apply", nil)
	updateStatedbTimer             = metrics.NewRegisteredTimer("processor/tx/statedb/update", nil)
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
	defer func(t0 time.Time) {
		processBlockTimer.Update(time.Since(t0))
	}(time.Now())

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
	// Apply Curie hard fork
	if p.config.CurieBlock != nil && p.config.CurieBlock.Cmp(block.Number()) == 0 {
		misc.ApplyCurieHardFork(statedb)
	}
	// Apply Feynman hard fork
	parent := p.bc.GetHeaderByHash(block.ParentHash())
	if p.config.IsFeynmanTransitionBlock(block.Time(), parent.Time) {
		misc.ApplyFeynmanHardFork(statedb)
	}
	blockContext := NewEVMBlockContext(header, p.bc, p.config, nil)
	vmenv := vm.NewEVM(blockContext, vm.TxContext{}, statedb, p.config, cfg)
	processorBlockTransactionGauge.Update(int64(block.Transactions().Len()))
	// Apply EIP-2935
	if p.config.IsFeynman(block.Time()) {
		ProcessParentBlockHash(block.ParentHash(), vmenv, statedb)
	}
	// Iterate over and process the individual transactions
	for i, tx := range block.Transactions() {
		msg, err := tx.AsMessage(types.MakeSigner(p.config, header.Number, header.Time), header.BaseFee)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("could not apply tx %d [%v]: %w", i, tx.Hash().Hex(), err)
		}
		statedb.SetTxContext(tx.Hash(), i)
		receipt, err := applyTransaction(msg, p.config, p.bc, nil, gp, statedb, blockNumber, blockHash, tx, usedGas, vmenv)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("could not apply tx %d [%v]: %w", i, tx.Hash().Hex(), err)
		}
		receipts = append(receipts, receipt)
		allLogs = append(allLogs, receipt.Logs...)
	}
	// Finalize the block, applying any consensus engine specific extras (e.g. block rewards)
	finalizeBlockStartTime := time.Now()
	p.engine.Finalize(p.bc, header, statedb, block.Transactions(), block.Uncles())
	finalizeBlockTimer.Update(time.Since(finalizeBlockStartTime))

	return receipts, allLogs, *usedGas, nil
}

func applyTransaction(msg types.Message, config *params.ChainConfig, bc ChainContext, author *common.Address, gp *GasPool, statedb *state.StateDB, blockNumber *big.Int, blockHash common.Hash, tx *types.Transaction, usedGas *uint64, evm *vm.EVM) (*types.Receipt, error) {
	defer func(t0 time.Time) {
		applyTransactionTimer.Update(time.Since(t0))
	}(time.Now())

	// Create a new context to be used in the EVM environment.
	txContext := NewEVMTxContext(msg)
	evm.Reset(txContext, statedb)

	l1DataFee, err := fees.CalculateL1DataFee(tx, statedb, config, blockNumber)
	if err != nil {
		return nil, err
	}

	// Apply the transaction to the current state (included in the env).
	applyMessageStartTime := time.Now()
	result, err := ApplyMessage(evm, msg, gp, l1DataFee)
	if evm.Config.Debug {
		if erroringTracer, ok := evm.Config.Tracer.(interface{ Error() error }); ok {
			err = errors.Join(err, erroringTracer.Error())
		}
	}
	applyMessageTimer.Update(time.Since(applyMessageStartTime))
	if err != nil {
		return nil, err
	}

	// Update the state with pending changes.
	var root []byte
	updateStatedbStartTime := time.Now()
	if config.IsByzantium(blockNumber) {
		statedb.Finalise(true)
	} else {
		root = statedb.IntermediateRoot(config.IsEIP158(blockNumber)).Bytes()
	}
	updateStatedbTimer.Update(time.Since(updateStatedbStartTime))
	*usedGas += result.UsedGas

	// If the result contains a revert reason, return it.
	returnVal := result.Return()
	if len(result.Revert()) > 0 {
		returnVal = result.Revert()
	}
	// Create a new receipt for the transaction, storing the intermediate root and gas used
	// by the tx.
	receipt := &types.Receipt{Type: tx.Type(), PostState: root, CumulativeGasUsed: *usedGas, ReturnValue: returnVal}
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
	receipt.L1Fee = result.L1DataFee
	return receipt, err
}

// ApplyTransaction attempts to apply a transaction to the given state database
// and uses the input parameters for its environment. It returns the receipt
// for the transaction, gas used and an error if the transaction failed,
// indicating the block was invalid.
func ApplyTransaction(config *params.ChainConfig, bc ChainContext, author *common.Address, gp *GasPool, statedb *state.StateDB, header *types.Header, tx *types.Transaction, usedGas *uint64, cfg vm.Config) (*types.Receipt, error) {
	msg, err := tx.AsMessage(types.MakeSigner(config, header.Number, header.Time), header.BaseFee)
	if err != nil {
		return nil, err
	}
	// Create a new context to be used in the EVM environment
	blockContext := NewEVMBlockContext(header, bc, config, author)
	vmenv := vm.NewEVM(blockContext, vm.TxContext{}, statedb, config, cfg)
	return applyTransaction(msg, config, bc, author, gp, statedb, header.Number, header.Hash(), tx, usedGas, vmenv)
}

// ProcessParentBlockHash stores the parent block hash in the history storage contract
// as per EIP-2935.
func ProcessParentBlockHash(prevHash common.Hash, evm *vm.EVM, statedb *state.StateDB) {
	msg := types.NewMessage(
		params.SystemAddress,          // from
		&params.HistoryStorageAddress, // to
		0,                             // nonce
		common.Big0,                   // amount
		30_000_000,                    // gasLimit
		common.Big0,                   // gasPrice
		common.Big0,                   // gasFeeCap
		common.Big0,                   // gasTipCap
		prevHash.Bytes(),              // data
		nil,                           // accessList
		false,                         // isFake
		nil,                           // setCodeAuthorizations
	)

	evm.Reset(NewEVMTxContext(msg), statedb)
	statedb.AddAddressToAccessList(params.HistoryStorageAddress)
	_, _, err := evm.Call(vm.AccountRef(msg.From()), *msg.To(), msg.Data(), 30_000_000, common.Big0, nil)
	if err != nil {
		panic(err)
	}
	statedb.Finalise(true)
}
