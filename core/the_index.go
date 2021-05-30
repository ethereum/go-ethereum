// Copyright 2021 orbs-network
// No license

package core

import (
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

const theIndexVerbose = true

const contractShards byte = 64

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

func (bc *BlockChain) TheIndex_Hook_WriteContractsAndAccounts(block *types.Block, logs []*types.Log, state *state.StateDB) {
	if _, err := os.Stat("./the-index/"); os.IsNotExist(err) {
		return
	}

	// we need to aggregate by contract before we actually write
	contracts := map[common.Address]*rlp.TheIndex_rlpContract{}
	TheIndex_indexContractsLogs(logs, block, contracts)
	state.TheIndex_indexContractsState(contracts)

	// we also need to aggregate by account before we actually write
	accounts := make([]rlp.TheIndex_rlpAccount, 0)
	state.TheIndex_indexAccountChanges(block, &accounts)
	TheIndex_addAccountsToContracts(accounts, contracts)

	// shard the contracts
	contractsPerShard := map[byte][]rlp.TheIndex_rlpContract{}
	for address, contract := range contracts {
		shard := address[0] % contractShards
		// new shard, add it to the map
		if _, ok := contractsPerShard[shard]; !ok {
			contractsPerShard[shard] = make([]rlp.TheIndex_rlpContract, 0)
		}
		// add the contract to the shard
		contractsPerShard[shard] = append(contractsPerShard[shard], *contract)
		if theIndexVerbose {
			log.Info("THE-INDEX:contract", "addr", address.Hex(), "logs", len(contract.Logs), "code", len(contract.Code), "state", len(contract.States))
		}
	}

	// write all contracts to sharded blobs
	for shard, contractsInTheShard := range contractsPerShard {
		// write contract to blob
		file, err := os.OpenFile("./the-index/contracts-"+fmt.Sprintf("%02x", shard)+".rlp", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0755)
		if err != nil {
			log.Crit("THE-INDEX", "error", err)
		}
		defer file.Close()

		err = rlp.Encode(file, rlp.TheIndex_rlpContractsForBlock{
			BlockNumber: block.Header().Number,
			Contracts:   contractsInTheShard,
		})
		if err != nil {
			log.Crit("THE-INDEX", "error", err)
		}
	}

	// write all accounts to a blob
	if len(accounts) > 0 {
		if theIndexVerbose {
			log.Info("THE-INDEX:accounts", "changes", len(accounts))
		}

		// write combined accounts (all users) to blob
		file, err := os.OpenFile("./the-index/accounts.rlp", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0755)
		if err != nil {
			log.Crit("THE-INDEX", "error", err)
		}
		defer file.Close()

		err = rlp.Encode(file, rlp.TheIndex_rlpAccountsForBlock{
			BlockNumber: block.Header().Number,
			Accounts:    accounts,
		})
		if err != nil {
			log.Crit("THE-INDEX", "error", err)
		}
	}
}

func TheIndex_indexContractsLogs(logs []*types.Log, block *types.Block, contracts map[common.Address]*rlp.TheIndex_rlpContract) {
	for _, log := range logs {
		// new contract address, add it to the map
		if _, ok := contracts[log.Address]; !ok {
			contracts[log.Address] = &rlp.TheIndex_rlpContract{Address: log.Address}
		}
		// add the log to the contract
		contracts[log.Address].Logs = append(contracts[log.Address].Logs, rlp.TheIndex_rlpLog{Topics: log.Topics, Data: log.Data})
	}
}

func TheIndex_addAccountsToContracts(accounts []rlp.TheIndex_rlpAccount, contracts map[common.Address]*rlp.TheIndex_rlpContract) {
	for _, account := range accounts {
		if account.CodeHash != nil {
			// new contract address, add it to the map
			if _, ok := contracts[account.Address]; !ok {
				contracts[account.Address] = &rlp.TheIndex_rlpContract{Address: account.Address}
			}
			// add the account to the contract
			contracts[account.Address].Balance = account.Balance
		}
	}
}
