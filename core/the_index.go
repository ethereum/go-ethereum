// Copyright 2021 orbs-network
// No license

package core

import (
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

const contractShards byte = 64
const maxFileSizeMb = 1024
const fileIndexFormat = "%05d"

const blocksFilePrefix = "blocks-"
const accountsFilePrefix = "accounts-"
const contractsShardFilePrefix = "contracts%02d-"
const cursorFileName = "cursor"

var theIndexVerbose = os.Getenv("THEINDEX_VERBOSE")
var indexPath = os.Getenv("THEINDEX_PATH") // eg: "./the-index/"
var maxFileIndex map[string]int = theIndex_getCurrentMaxFileIndexesOnInit()
var startIndexAfterBlock = theIndex_getCurrentCursorBlockOnInit()

func (bc *BlockChain) TheIndex_Hook_UpdateCursor(block *types.Block) {
	if !theIndex_isEnabled() || !theIndex_shouldIndexBlock(block) {
		return
	}

	file, err := theIndex_openFileReplace(cursorFileName)
	if err != nil {
		bc.theIndex_criticalError(err)
	}
	defer file.Close()

	err = rlp.Encode(file, rlp.TheIndex_rlpCursor{
		BlockNumber: block.Header().Number,
		Time:        block.Header().Time,
	})
	if err != nil {
		bc.theIndex_criticalError(err)
	}
}

func (bc *BlockChain) TheIndex_Hook_WriteBlockHeader(block *types.Block) {
	if !theIndex_isEnabled() || !theIndex_shouldIndexBlock(block) {
		return
	}

	if len(theIndexVerbose) > 0 || block.Header().Number.Int64()%5000 == 0 {
		log.Info("THE-INDEX:blocks", "num", block.Header().Number)
	}

	// write block header to blob
	file, err := theIndex_openFileAppendWithPaging(blocksFilePrefix)
	if err != nil {
		bc.theIndex_criticalError(err)
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
		bc.theIndex_criticalError(err)
	}
}

func (bc *BlockChain) TheIndex_Hook_WriteContractsAndAccounts(block *types.Block, logs []*types.Log, state *state.StateDB) {
	if !theIndex_isEnabled() || !theIndex_shouldIndexBlock(block) {
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
		if len(theIndexVerbose) > 0 {
			log.Info("THE-INDEX:contract", "addr", address.Hex(), "logs", len(contract.Logs), "code", len(contract.Code), "state", len(contract.States))
		}
	}

	// write all contracts to sharded blobs
	for shard, contractsInTheShard := range contractsPerShard {
		// write contract to blob
		file, err := theIndex_openFileAppendWithPaging(fmt.Sprintf(contractsShardFilePrefix, shard))
		if err != nil {
			bc.theIndex_criticalError(err)
		}
		defer file.Close()

		err = rlp.Encode(file, rlp.TheIndex_rlpContractsForBlock{
			BlockNumber: block.Header().Number,
			Contracts:   contractsInTheShard,
		})
		if err != nil {
			bc.theIndex_criticalError(err)
		}
	}

	// write all accounts to a blob
	if len(accounts) > 0 {
		if len(theIndexVerbose) > 0 {
			log.Info("THE-INDEX:accounts", "changes", len(accounts))
		}

		// write combined accounts (all users) to blob
		file, err := theIndex_openFileAppendWithPaging(accountsFilePrefix)
		if err != nil {
			bc.theIndex_criticalError(err)
		}
		defer file.Close()

		err = rlp.Encode(file, rlp.TheIndex_rlpAccountsForBlock{
			BlockNumber: block.Header().Number,
			Accounts:    accounts,
		})
		if err != nil {
			bc.theIndex_criticalError(err)
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
			contracts[account.Address].Balance = []*big.Int{account.Balance}
		}
	}
}

func theIndex_getCurrentMaxFileIndexesOnInit() map[string]int {
	res := map[string]int{}

	if theIndex_isEnabled() {
		fmt.Println("THE-INDEX:init", "enabled", indexPath)

		res[blocksFilePrefix] = theIndex_getCurrentMaxFileIndex(blocksFilePrefix)
		res[accountsFilePrefix] = theIndex_getCurrentMaxFileIndex(accountsFilePrefix)
		for shard := 0; shard < int(contractShards); shard++ {
			filePrefix := fmt.Sprintf(contractsShardFilePrefix, shard)
			res[filePrefix] = theIndex_getCurrentMaxFileIndex(filePrefix)
		}

	} else {
		fmt.Println("THE-INDEX:init", "disabled", indexPath)
	}
	return res
}

func theIndex_isEnabled() bool {
	if len(indexPath) == 0 {
		return false
	}
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		return false
	}
	return true
}

func theIndex_shouldIndexBlock(block *types.Block) bool {
	return (block.NumberU64() > startIndexAfterBlock)
}

func theIndex_getCurrentMaxFileIndex(prefix string) int {
	var res = 1
	re := regexp.MustCompile(`-(\d+)\.`)
	matches, err := filepath.Glob(indexPath + prefix + "*.rlp")
	if err != nil {
		fmt.Println("THE-INDEX", "error", err)
	}
	for _, filePath := range matches {
		indexStr := re.FindStringSubmatch(filePath)
		var index = 0
		if len(indexStr) >= 1 {
			index, err = strconv.Atoi(indexStr[1])
			if err != nil {
				fmt.Println("THE-INDEX", "error", err)
			}
		}
		if index > res {
			res = index
		}
	}
	return res
}

func theIndex_getCurrentCursorBlockOnInit() uint64 {
	file, err := theIndex_openFileRead(cursorFileName)
	if err != nil {
		fmt.Println("THE-INDEX:init", "cursor not found")
		return 0
	}
	defer file.Close()

	cursorData := new(rlp.TheIndex_rlpCursor)
	err = rlp.Decode(file, cursorData)
	if err != nil {
		fmt.Println("THE-INDEX:init", "cursor corrupt")
		return 0
	}

	fmt.Println("THE-INDEX:init", "cursor found", cursorData.BlockNumber.Uint64())
	return cursorData.BlockNumber.Uint64()
}

func theIndex_openFileAppendWithPaging(prefix string) (*os.File, error) {
	filePath := fmt.Sprintf(indexPath+prefix+fileIndexFormat+".rlp", maxFileIndex[prefix])
	stat, err := os.Stat(filePath)
	if err == nil && stat.Size() >= maxFileSizeMb*1024*1024 {
		// compress the file and move to the next one
		exec.Command("gzip", filePath).Run()
		maxFileIndex[prefix]++
		filePath = fmt.Sprintf(indexPath+prefix+fileIndexFormat+".rlp", maxFileIndex[prefix])
	}
	return os.OpenFile(filePath, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0755)
}

func theIndex_openFileReplace(name string) (*os.File, error) {
	filePath := indexPath + name + ".rlp"
	return os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0755)
}

func theIndex_openFileRead(name string) (*os.File, error) {
	filePath := indexPath + name + ".rlp"
	return os.OpenFile(filePath, os.O_RDONLY, 0755)
}

func (bc *BlockChain) theIndex_criticalError(err error) {
	bc.Stop()
	bc.db.Close()
	log.Crit("THE-INDEX:critical", "error", err)
}
