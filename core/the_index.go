package core

import (
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

const theIndexVerbose = true

type rlpBlock struct {
	BlockNumber *big.Int
	Time        uint64
}

func (bc *BlockChain) theIndex_Hook_WriteBlockHeader(block *types.Block) {
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

	err = rlp.Encode(file, rlpBlock{BlockNumber: block.Header().Number, Time: block.Header().Time})
	if err != nil {
		log.Crit("THE-INDEX", "error", err)
	}
}

type rlpLog struct {
	Topics []common.Hash
	Data   []byte
}

type rlpContract struct {
	BlockNumber *big.Int
	Logs        []rlpLog
}

func (bc *BlockChain) theIndex_Hook_WriteContractsStorage(block *types.Block, logs []*types.Log) {
	if _, err := os.Stat("./the-index/"); os.IsNotExist(err) {
		return
	}

	// index all the contracts that need storage in this block
	contracts := map[common.Address]*rlpContract{}

	// index logs from all contracts
	for _, log := range logs {
		var ok bool
		var contract *rlpContract
		if contract, ok = contracts[log.Address]; !ok {
			contract = &rlpContract{BlockNumber: block.Header().Number}
			contracts[log.Address] = contract
		}
		contract.Logs = append(contract.Logs, rlpLog{Topics: log.Topics, Data: log.Data})
	}

	// write to blob
	for address, contract := range contracts {
		if theIndexVerbose {
			log.Info("THE-INDEX:contract", "addr", address.Hex(), "logs", len(contract.Logs))
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
