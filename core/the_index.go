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

	contracts := map[common.Address]*rlp.TheIndex_rlpContract{}
	TheIndex_indexContractsLogs(logs, block, contracts)
	state.TheIndex_indexContractsState(block, contracts)

	// write to blob
	for address, contract := range contracts {
		if theIndexVerbose {
			log.Info("THE-INDEX:contract", "addr", address.Hex(), "logs", len(contract.Logs), "code", len(contract.Code), "state", len(contract.States))
		}

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
