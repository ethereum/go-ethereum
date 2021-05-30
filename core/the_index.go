// Copyright 2021 orbs-network
// No license

package core

import (
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

const theIndexVerbose = true

func (bc *BlockChain) TheIndex_Hook_WriteBlockHeader(block *types.Block) {
	if _, err := os.Stat("./the-index/"); os.IsNotExist(err) {
		return
	}

	if theIndexVerbose {
		log.Info("THE-INDEX:blocks", "num", block.Header().Number)
	}

	// write block header to blob
	file, err := os.OpenFile("./the-index/blocks.rlp", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0755)
	if err != nil {
		log.Crit("THE-INDEX", "error", err)
	}
	defer file.Close()

	err = rlp.Encode(file, rlp.TheIndex_rlpBlock{
		BlockNumber: block.Header().Number,
		Time:        block.Header().Time,
		Hash:        block.Hash(),
		Coinbase:    block.Header().Coinbase,
		Difficulty:  block.Header().Difficulty,
		GasLimit:    block.Header().GasLimit,
	})
	if err != nil {
		log.Crit("THE-INDEX", "error", err)
	}
}

func (bc *BlockChain) TheIndex_Hook_WriteContractsStorage(block *types.Block, logs []*types.Log, state *state.StateDB) {
	if _, err := os.Stat("./the-index/"); os.IsNotExist(err) {
		return
	}

	// we need to aggregate by contract before we actually write
	contracts := map[common.Address]*rlp.TheIndex_rlpContract{}
	TheIndex_indexContractsLogs(logs, block, contracts)
	state.TheIndex_indexContractsState(block, contracts)

	// write all contracts to separate blobs
	for address, contract := range contracts {
		if theIndexVerbose {
			log.Info("THE-INDEX:contract", "addr", address.Hex(), "logs", len(contract.Logs), "code", len(contract.Code), "state", len(contract.States))
		}

		// write contract to blob
		file, err := os.OpenFile("./the-index/contract-"+address.Hex()+".rlp", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0755)
		if err != nil {
			log.Crit("THE-INDEX", "error", err)
		}
		defer file.Close()

		err = rlp.Encode(file, contract)
		if err != nil {
			log.Crit("THE-INDEX", "error", err)
		}
	}
}

func (bc *BlockChain) TheIndex_Hook_WriteAccountChanges(block *types.Block, state *state.StateDB) {
	if _, err := os.Stat("./the-index/"); os.IsNotExist(err) {
		return
	}

	// we need to aggregate before we actually write
	accounts := make([]rlp.TheIndex_rplAccount, 0)
	state.TheIndex_indexAccountChanges(block, &accounts)

	if len(accounts) == 0 {
		return
	}

	if theIndexVerbose {
		log.Info("THE-INDEX:accounts", "changes", len(accounts))
	}

	// write combined accounts (all users) to blob
	file, err := os.OpenFile("./the-index/accounts.rlp", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0755)
	if err != nil {
		log.Crit("THE-INDEX", "error", err)
	}
	defer file.Close()

	err = rlp.Encode(file, rlp.TheIndex_rplAccountChanges{
		BlockNumber: block.Header().Number,
		Accounts:    accounts,
	})
	if err != nil {
		log.Crit("THE-INDEX", "error", err)
	}

	// write contract accounts to separate blobs
	for _, account := range accounts {
		if account.CodeHash == nil {
			continue
		}

		// write contract account to blob
		file, err := os.OpenFile("./the-index/account-"+account.Address.Hex()+".rlp", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0755)
		if err != nil {
			log.Crit("THE-INDEX", "error", err)
		}
		defer file.Close()

		err = rlp.Encode(file, rlp.TheIndex_rplContractAccountChange{
			BlockNumber: block.Header().Number,
			Balance:     account.Balance,
		})
		if err != nil {
			log.Crit("THE-INDEX", "error", err)
		}
	}
}

func TheIndex_indexContractsLogs(logs []*types.Log, block *types.Block, contracts map[common.Address]*rlp.TheIndex_rlpContract) {
	for _, log := range logs {
		// add the contract
		var ok bool
		var contract *rlp.TheIndex_rlpContract
		// new contract address, add it to the map
		if contract, ok = contracts[log.Address]; !ok {
			contract = &rlp.TheIndex_rlpContract{BlockNumber: block.Header().Number}
			contracts[log.Address] = contract
		}
		// add the log to the contract
		contract.Logs = append(contract.Logs, rlp.TheIndex_rlpLog{Topics: log.Topics, Data: log.Data})
	}
}
