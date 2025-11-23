// Copyright 2024 The go-ethereum Authors
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

package miner

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

// BundleSimulator provides functionality to simulate bundle execution.
type BundleSimulator struct {
	chain  *core.BlockChain
	config *params.ChainConfig
}

// NewBundleSimulator creates a new bundle simulator.
func NewBundleSimulator(chain *core.BlockChain) *BundleSimulator {
	return &BundleSimulator{
		chain:  chain,
		config: chain.Config(),
	}
}

// SimulateBundle simulates the execution of a bundle on top of the given state.
func (s *BundleSimulator) SimulateBundle(
	bundle *Bundle,
	header *types.Header,
	stateDB *state.StateDB,
	coinbase common.Address,
) (*BundleSimulationResult, error) {
	// Validate bundle timestamp
	if err := bundle.ValidateTimestamp(header.Time); err != nil {
		return nil, err
	}

	result := &BundleSimulationResult{
		Success:      true,
		GasUsed:      0,
		Profit:       big.NewInt(0),
		StateChanges: make(map[common.Address]*AccountChange),
		FailedTxIndex: -1,
		TxResults:    make([]*TxSimulationResult, 0, len(bundle.Txs)),
	}

	// Record initial coinbase balance
	result.CoinbaseBalance = new(big.Int).Set(stateDB.GetBalance(coinbase).ToBig())

	// Create EVM context
	vmContext := core.NewEVMBlockContext(header, s.chain, &coinbase)
	vmConfig := vm.Config{}
	evm := vm.NewEVM(vmContext, stateDB, s.config, vmConfig)

	// Simulate each transaction in the bundle
	gasPool := new(core.GasPool).AddGas(header.GasLimit)

	for i, tx := range bundle.Txs {
		// Take snapshot before transaction
		snapshot := stateDB.Snapshot()

		// Record pre-execution state
		from, err := types.Sender(types.LatestSigner(s.config), tx)
		if err != nil {
			result.Success = false
			result.FailedTxIndex = i
			result.FailedTxError = err
			return result, nil
		}

		// Execute transaction
		txResult := s.simulateTransaction(tx, evm, stateDB, gasPool, header, from)
		result.TxResults = append(result.TxResults, txResult)
		result.GasUsed += txResult.GasUsed

		// Check if transaction failed and is not allowed to revert
		if !txResult.Success && !bundle.CanRevert(i) {
			// Revert state
			stateDB.RevertToSnapshot(snapshot)
			result.Success = false
			result.FailedTxIndex = i
			result.FailedTxError = txResult.Error
			return result, nil
		}

		// Calculate profit from this transaction
		if txResult.Success {
			minerFee, _ := tx.EffectiveGasTip(header.BaseFee)
			profit := new(big.Int).Mul(minerFee, new(big.Int).SetUint64(txResult.GasUsed))
			result.Profit.Add(result.Profit, profit)
		}
	}

	// Record final coinbase balance
	finalBalance := stateDB.GetBalance(coinbase).ToBig()
	actualProfit := new(big.Int).Sub(finalBalance, result.CoinbaseBalance)
	result.Profit = actualProfit
	result.CoinbaseBalance = finalBalance

	return result, nil
}

// simulateTransaction simulates a single transaction.
func (s *BundleSimulator) simulateTransaction(
	tx *types.Transaction,
	evm *vm.EVM,
	stateDB *state.StateDB,
	gasPool *core.GasPool,
	header *types.Header,
	from common.Address,
) *TxSimulationResult {
	result := &TxSimulationResult{
		Success: true,
		Logs:    make([]*types.Log, 0),
	}

	// Set tx context
	stateDB.SetTxContext(tx.Hash(), stateDB.TxIndex())

	// Convert transaction to message
	msg, err := core.TransactionToMessage(tx, types.MakeSigner(s.config, header.Number, header.Time), header.BaseFee)
	if err != nil {
		result.Success = false
		result.Error = err
		return result
	}

	// Apply the transaction
	execResult, err := core.ApplyMessage(evm, msg, gasPool)
	if err != nil {
		result.Success = false
		result.Error = err
		result.GasUsed = tx.Gas()
		return result
	}

	// Check execution result
	if execResult.Failed() {
		result.Success = false
		result.Error = execResult.Err
	}

	result.GasUsed = execResult.UsedGas
	result.ReturnValue = execResult.ReturnData

	// Collect logs
	result.Logs = stateDB.GetLogs(tx.Hash(), header.Number.Uint64(), header.Hash(), header.Time)

	return result
}

// SimulateBundleAtPosition simulates a bundle inserted at a specific position
// in the existing block template.
func (s *BundleSimulator) SimulateBundleAtPosition(
	bundle *Bundle,
	baseBlock *types.Block,
	position int,
) (*BundleSimulationResult, error) {
	// Get state at parent block
	parent := s.chain.GetBlock(baseBlock.ParentHash(), baseBlock.NumberU64()-1)
	if parent == nil {
		return nil, errors.New("parent block not found")
	}

	stateDB, err := s.chain.StateAt(parent.Root())
	if err != nil {
		return nil, err
	}

	// Copy the state for simulation
	simState := stateDB.Copy()

	// Execute transactions before the insertion point
	if position > 0 {
		header := baseBlock.Header()
		vmContext := core.NewEVMBlockContext(header, s.chain, &header.Coinbase)
		vmConfig := vm.Config{}
		evm := vm.NewEVM(vmContext, simState, s.config, vmConfig)
		gasPool := new(core.GasPool).AddGas(header.GasLimit)

		for i, tx := range baseBlock.Transactions() {
			if i >= position {
				break
			}
			msg, err := core.TransactionToMessage(tx, types.MakeSigner(s.config, header.Number, header.Time), header.BaseFee)
			if err != nil {
				log.Warn("Failed to convert transaction to message", "err", err)
				continue
			}
			simState.SetTxContext(tx.Hash(), i)
			_, err = core.ApplyMessage(evm, msg, gasPool)
			if err != nil {
				log.Warn("Transaction execution failed in pre-bundle simulation", "err", err)
			}
		}
	}

	// Now simulate the bundle
	return s.SimulateBundle(bundle, baseBlock.Header(), simState, baseBlock.Coinbase())
}

