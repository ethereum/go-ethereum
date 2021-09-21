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
	"runtime"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
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
type CalculatedBlock struct {
	block *types.Block
	stop  bool
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
func (p *StateProcessor) Process(block *types.Block, statedb *state.StateDB, cfg vm.Config, balanceFee map[common.Address]*big.Int) (types.Receipts, []*types.Log, uint64, error) {
	var (
		receipts types.Receipts
		usedGas  = new(uint64)
		header   = block.Header()
		allLogs  []*types.Log
		gp       = new(GasPool).AddGas(block.GasLimit())
	)
	// Mutate the the block and state according to any hard-fork specs
	if p.config.DAOForkSupport && p.config.DAOForkBlock != nil && p.config.DAOForkBlock.Cmp(block.Number()) == 0 {
		misc.ApplyDAOHardFork(statedb)
	}
	if common.TIPSigning.Cmp(header.Number) == 0 {
		statedb.DeleteAddress(common.HexToAddress(common.BlockSigners))
	}
	InitSignerInTransactions(p.config, header, block.Transactions())
	balanceUpdated := map[common.Address]*big.Int{}
	totalFeeUsed := big.NewInt(0)
	for i, tx := range block.Transactions() {
		// check black-list txs after hf
		if (block.Number().Uint64() >= common.BlackListHFNumber) && !common.IsTestnet {
			// check if sender is in black list
			if tx.From() != nil && common.Blacklist[*tx.From()] {
				return nil, nil, 0, fmt.Errorf("Block contains transaction with sender in black-list: %v", tx.From().Hex())
			}
			// check if receiver is in black list
			if tx.To() != nil && common.Blacklist[*tx.To()] {
				return nil, nil, 0, fmt.Errorf("Block contains transaction with receiver in black-list: %v", tx.To().Hex())
			}
		}
		statedb.Prepare(tx.Hash(), block.Hash(), i)
		receipt, gas, err, tokenFeeUsed := ApplyTransaction(p.config, balanceFee, p.bc, nil, gp, statedb, header, tx, usedGas, cfg)
		if err != nil {
			return nil, nil, 0, err
		}
		receipts = append(receipts, receipt)
		allLogs = append(allLogs, receipt.Logs...)
		if tokenFeeUsed {
			fee := new(big.Int).SetUint64(gas)
			if block.Header().Number.Cmp(common.TIPTRC21Fee) > 0 {
				fee = fee.Mul(fee, common.TRC21GasPrice)
			}
			balanceFee[*tx.To()] = new(big.Int).Sub(balanceFee[*tx.To()], fee)
			balanceUpdated[*tx.To()] = balanceFee[*tx.To()]
			totalFeeUsed = totalFeeUsed.Add(totalFeeUsed, fee)
		}
	}
	state.UpdateTRC21Fee(statedb, balanceUpdated, totalFeeUsed)
	// Finalize the block, applying any consensus engine specific extras (e.g. block rewards)
	p.engine.Finalize(p.bc, header, statedb, block.Transactions(), block.Uncles(), receipts)
	return receipts, allLogs, *usedGas, nil
}

func (p *StateProcessor) ProcessBlockNoValidator(cBlock *CalculatedBlock, statedb *state.StateDB, cfg vm.Config, balanceFee map[common.Address]*big.Int) (types.Receipts, []*types.Log, uint64, error) {
	block := cBlock.block
	var (
		receipts types.Receipts
		usedGas  = new(uint64)
		header   = block.Header()
		allLogs  []*types.Log
		gp       = new(GasPool).AddGas(block.GasLimit())
	)
	// Mutate the the block and state according to any hard-fork specs
	if p.config.DAOForkSupport && p.config.DAOForkBlock != nil && p.config.DAOForkBlock.Cmp(block.Number()) == 0 {
		misc.ApplyDAOHardFork(statedb)
	}
	if common.TIPSigning.Cmp(header.Number) == 0 {
		statedb.DeleteAddress(common.HexToAddress(common.BlockSigners))
	}
	if cBlock.stop {
		return nil, nil, 0, ErrStopPreparingBlock
	}
	InitSignerInTransactions(p.config, header, block.Transactions())
	balanceUpdated := map[common.Address]*big.Int{}
	totalFeeUsed := big.NewInt(0)

	if cBlock.stop {
		return nil, nil, 0, ErrStopPreparingBlock
	}
	// Iterate over and process the individual transactions
	receipts = make([]*types.Receipt, block.Transactions().Len())
	for i, tx := range block.Transactions() {
		// check black-list txs after hf
		if (block.Number().Uint64() >= common.BlackListHFNumber) && !common.IsTestnet {
			// check if sender is in black list
			if tx.From() != nil && common.Blacklist[*tx.From()] {
				return nil, nil, 0, fmt.Errorf("Block contains transaction with sender in black-list: %v", tx.From().Hex())
			}
			// check if receiver is in black list
			if tx.To() != nil && common.Blacklist[*tx.To()] {
				return nil, nil, 0, fmt.Errorf("Block contains transaction with receiver in black-list: %v", tx.To().Hex())
			}
		}
		statedb.Prepare(tx.Hash(), block.Hash(), i)
		receipt, gas, err, tokenFeeUsed := ApplyTransaction(p.config, balanceFee, p.bc, nil, gp, statedb, header, tx, usedGas, cfg)
		if err != nil {
			return nil, nil, 0, err
		}
		if cBlock.stop {
			return nil, nil, 0, ErrStopPreparingBlock
		}
		receipts[i] = receipt
		allLogs = append(allLogs, receipt.Logs...)
		if tokenFeeUsed {
			fee := new(big.Int).SetUint64(gas)
			if block.Header().Number.Cmp(common.TIPTRC21Fee) > 0 {
				fee = fee.Mul(fee, common.TRC21GasPrice)
			}
			balanceFee[*tx.To()] = new(big.Int).Sub(balanceFee[*tx.To()], fee)
			balanceUpdated[*tx.To()] = balanceFee[*tx.To()]
			totalFeeUsed = totalFeeUsed.Add(totalFeeUsed, fee)
		}
	}
	state.UpdateTRC21Fee(statedb, balanceUpdated, totalFeeUsed)
	// Finalize the block, applying any consensus engine specific extras (e.g. block rewards)
	p.engine.Finalize(p.bc, header, statedb, block.Transactions(), block.Uncles(), receipts)
	return receipts, allLogs, *usedGas, nil
}

// ApplyTransaction attempts to apply a transaction to the given state database
// and uses the input parameters for its environment. It returns the receipt
// for the transaction, gas used and an error if the transaction failed,
// indicating the block was invalid.
func ApplyTransaction(config *params.ChainConfig, tokensFee map[common.Address]*big.Int, bc *BlockChain, author *common.Address, gp *GasPool, statedb *state.StateDB, header *types.Header, tx *types.Transaction, usedGas *uint64, cfg vm.Config) (*types.Receipt, uint64, error, bool) {
	if tx.To() != nil && tx.To().String() == common.BlockSigners && config.IsTIPSigning(header.Number) {
		return ApplySignTransaction(config, statedb, header, tx, usedGas)
	}
	var balanceFee *big.Int
	if tx.To() != nil {
		if value, ok := tokensFee[*tx.To()]; ok {
			balanceFee = value
		}
	}
	msg, err := tx.AsMessage(types.MakeSigner(config, header.Number), balanceFee,header.Number)
	if err != nil {
		return nil, 0, err, false
	}
	// Create a new context to be used in the EVM environment
	context := NewEVMContext(msg, header, bc, author)
	// Create a new environment which holds all relevant information
	// about the transaction and calling mechanisms.
	vmenv := vm.NewEVM(context, statedb, config, cfg)

	// If we don't have an explicit author (i.e. not mining), extract from the header
	var beneficiary common.Address
	if author == nil {
		beneficiary, _ = bc.Engine().Author(header) // Ignore error, we're past header validation
	} else {
		beneficiary = *author
	}

	coinbaseOwner := statedb.GetOwner(beneficiary)

	// Apply the transaction to the current state (included in the env)
	_, gas, failed, err := ApplyMessage(vmenv, msg, gp, coinbaseOwner)
	if err != nil {
		return nil, 0, err, false
	}
	// Update the state with pending changes
	var root []byte
	if config.IsByzantium(header.Number) {
		statedb.Finalise(true)
	} else {
		root = statedb.IntermediateRoot(config.IsEIP158(header.Number)).Bytes()
	}
	*usedGas += gas

	// Create a new receipt for the transaction, storing the intermediate root and gas used by the tx
	// based on the eip phase, we're passing wether the root touch-delete accounts.
	receipt := types.NewReceipt(root, failed, *usedGas)
	receipt.TxHash = tx.Hash()
	receipt.GasUsed = gas
	// if the transaction created a contract, store the creation address in the receipt.
	if msg.To() == nil {
		receipt.ContractAddress = crypto.CreateAddress(vmenv.Context.Origin, tx.Nonce())
	}
	// Set the receipt logs and create a bloom for filtering
	receipt.Logs = statedb.GetLogs(tx.Hash())
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})
	if balanceFee != nil && failed {
		state.PayFeeWithTRC21TxFail(statedb, msg.From(), *tx.To())
	}
	return receipt, gas, err, balanceFee != nil
}

func ApplySignTransaction(config *params.ChainConfig, statedb *state.StateDB, header *types.Header, tx *types.Transaction, usedGas *uint64) (*types.Receipt, uint64, error, bool) {
	// Update the state with pending changes
	var root []byte
	if config.IsByzantium(header.Number) {
		statedb.Finalise(true)
	} else {
		root = statedb.IntermediateRoot(config.IsEIP158(header.Number)).Bytes()
	}
	from, err := types.Sender(types.MakeSigner(config, header.Number), tx)
	if err != nil {
		return nil, 0, err, false
	}
	nonce := statedb.GetNonce(from)
	if nonce < tx.Nonce() {
		return nil, 0, ErrNonceTooHigh, false
	} else if nonce > tx.Nonce() {
		return nil, 0, ErrNonceTooLow, false
	}
	statedb.SetNonce(from, nonce+1)
	// Create a new receipt for the transaction, storing the intermediate root and gas used by the tx
	// based on the eip phase, we're passing wether the root touch-delete accounts.
	receipt := types.NewReceipt(root, false, *usedGas)
	receipt.TxHash = tx.Hash()
	receipt.GasUsed = 0
	// if the transaction created a contract, store the creation address in the receipt.
	// Set the receipt logs and create a bloom for filtering
	log := &types.Log{}
	log.Address = common.HexToAddress(common.BlockSigners)
	log.BlockNumber = header.Number.Uint64()
	statedb.AddLog(log)
	receipt.Logs = statedb.GetLogs(tx.Hash())
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})
	return receipt, 0, nil, false
}

func InitSignerInTransactions(config *params.ChainConfig, header *types.Header, txs types.Transactions) {
	nWorker := runtime.NumCPU()
	signer := types.MakeSigner(config, header.Number)
	chunkSize := txs.Len() / nWorker
	if txs.Len()%nWorker != 0 {
		chunkSize++
	}
	wg := sync.WaitGroup{}
	wg.Add(nWorker)
	for i := 0; i < nWorker; i++ {
		from := i * chunkSize
		to := from + chunkSize
		if to > txs.Len() {
			to = txs.Len()
		}
		go func(from int, to int) {
			for j := from; j < to; j++ {
				types.CacheSigner(signer, txs[j])
				txs[j].CacheHash()
			}
			wg.Done()
		}(from, to)
	}
	wg.Wait()
}
