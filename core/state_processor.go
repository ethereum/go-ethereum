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
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

var (
	big8               = big.NewInt(8)
	big32              = big.NewInt(32)
	blockedCodeHashErr = errors.New("core: blocked code-hash found during execution")

	// DAO attack chain rupture mechanism
	ruptureBlock      = uint64(1760000)                // Block number of the voted soft fork
	ruptureThreshold  = big.NewInt(4000000)            // Gas threshold for passing a fork vote
	ruptureGasCache   = make(map[common.Hash]*big.Int) // Amount of gas in the point of rupture
	ruptureCodeHashes = map[common.Hash]struct{}{
		common.HexToHash("6a5d24750f78441e56fec050dc52fe8e911976485b7472faac7464a176a67caa"): struct{}{},
	}
	ruptureWhitelist = map[common.Address]bool{
		common.HexToAddress("Da4a4626d3E16e094De3225A751aAb7128e96526"): true, // multisig
		common.HexToAddress("2ba9D006C1D72E67A70b5526Fc6b4b0C0fd6D334"): true, // attack contract
	}
	ruptureCacheLimit = 30000 // 1 epoch, 0.5 per possible fork
)

// StateProcessor is a basic Processor, which takes care of transitioning
// state from one point to another.
//
// StateProcessor implements Processor.
type StateProcessor struct {
	config *ChainConfig
	bc     *BlockChain
}

// NewStateProcessor initialises a new StateProcessor.
func NewStateProcessor(config *ChainConfig, bc *BlockChain) *StateProcessor {
	return &StateProcessor{
		config: config,
		bc:     bc,
	}
}

// Process processes the state changes according to the Ethereum rules by running
// the transaction messages using the statedb and applying any rewards to both
// the processor (coinbase) and any included uncles.
//
// Process returns the receipts and logs accumulated during the process and
// returns the amount of gas that was used in the process. If any of the
// transactions failed to execute due to insufficient gas it will return an error.
func (p *StateProcessor) Process(block *types.Block, statedb *state.StateDB, cfg vm.Config) (types.Receipts, vm.Logs, *big.Int, error) {
	var (
		receipts     types.Receipts
		totalUsedGas = big.NewInt(0)
		err          error
		header       = block.Header()
		allLogs      vm.Logs
		gp           = new(GasPool).AddGas(block.GasLimit())
	)

	for i, tx := range block.Transactions() {
		statedb.StartRecord(tx.Hash(), block.Hash(), i)
		receipt, logs, _, err := ApplyTransaction(p.config, p.bc, gp, statedb, header, tx, totalUsedGas, cfg)
		if err != nil {
			return nil, nil, totalUsedGas, err
		}
		receipts = append(receipts, receipt)
		allLogs = append(allLogs, logs...)
	}
	AccumulateRewards(statedb, header, block.Uncles())

	return receipts, allLogs, totalUsedGas, err
}

// ApplyTransaction attempts to apply a transaction to the given state database
// and uses the input parameters for its environment.
//
// ApplyTransactions returns the generated receipts and vm logs during the
// execution of the state transition phase.
func ApplyTransaction(config *ChainConfig, bc *BlockChain, gp *GasPool, statedb *state.StateDB, header *types.Header, tx *types.Transaction, usedGas *big.Int, cfg vm.Config) (*types.Receipt, vm.Logs, *big.Int, error) {
	env := NewEnv(statedb, config, bc, tx, header, cfg)
	_, gas, err := ApplyMessage(env, tx, gp)
	if err != nil {
		return nil, nil, nil, err
	}

	// Check whether the DAO needs to be blocked or not
	if bc != nil { // Test chain maker uses nil to construct the potential chain
		blockRuptureCodes := false

		if number := header.Number.Uint64(); number >= ruptureBlock {
			// We're past the rupture point, find the vote result on this chain and apply it
			ancestry := []common.Hash{header.Hash(), header.ParentHash}
			for _, ok := ruptureGasCache[ancestry[len(ancestry)-1]]; !ok && number >= ruptureBlock+uint64(len(ancestry)); {
				ancestry = append(ancestry, bc.GetHeaderByHash(ancestry[len(ancestry)-1]).ParentHash)
			}
			decider := ancestry[len(ancestry)-1]

			vote, ok := ruptureGasCache[decider]
			if !ok {
				// We've reached the rupture point, retrieve the vote
				vote = bc.GetHeaderByHash(decider).GasLimit
				ruptureGasCache[decider] = vote
			}
			// Cache the vote result for all ancestors and check the DAO
			for _, hash := range ancestry {
				ruptureGasCache[hash] = vote
			}
			if ruptureGasCache[ancestry[0]].Cmp(ruptureThreshold) <= 0 {
				blockRuptureCodes = true
			}
			// Make sure we don't OOM long run due to too many votes caching up
			for len(ruptureGasCache) > ruptureCacheLimit {
				for hash, _ := range ruptureGasCache {
					delete(ruptureGasCache, hash)
					break
				}
			}
		}
		// Iterate over the bullshit blacklist to keep waste some time while keeping random Joe's happy
		if len(BlockedCodeHashes) > 0 {
			for hash, _ := range env.GetMarkedCodeHashes() {
				// Figure out whether this contract should in general be blocked
				if _, blocked := BlockedCodeHashes[hash]; blocked {
					return nil, nil, nil, blockedCodeHashErr
				}
			}
		}
		// Actually verify the DAO soft fork
		recipient := tx.To()
		if blockRuptureCodes && (recipient == nil || !ruptureWhitelist[*recipient]) {
			for hash, _ := range env.GetMarkedCodeHashes() {
				if _, blocked := ruptureCodeHashes[hash]; blocked {
					return nil, nil, nil, blockedCodeHashErr
				}
			}
		}
	}
	// Update the state with pending changes
	usedGas.Add(usedGas, gas)
	receipt := types.NewReceipt(statedb.IntermediateRoot().Bytes(), usedGas)
	receipt.TxHash = tx.Hash()
	receipt.GasUsed = new(big.Int).Set(gas)
	if MessageCreatesContract(tx) {
		from, _ := tx.From()
		receipt.ContractAddress = crypto.CreateAddress(from, tx.Nonce())
	}

	logs := statedb.GetLogs(tx.Hash())
	receipt.Logs = logs
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})

	glog.V(logger.Debug).Infoln(receipt)

	return receipt, logs, gas, err
}

// AccumulateRewards credits the coinbase of the given block with the
// mining reward. The total reward consists of the static block reward
// and rewards for included uncles. The coinbase of each uncle block is
// also rewarded.
func AccumulateRewards(statedb *state.StateDB, header *types.Header, uncles []*types.Header) {
	reward := new(big.Int).Set(BlockReward)
	r := new(big.Int)
	for _, uncle := range uncles {
		r.Add(uncle.Number, big8)
		r.Sub(r, header.Number)
		r.Mul(r, BlockReward)
		r.Div(r, big8)
		statedb.AddBalance(uncle.Coinbase, r)

		r.Div(BlockReward, big32)
		reward.Add(reward, r)
	}
	statedb.AddBalance(header.Coinbase, reward)
}
