package core

import (
	"os"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

const theIndexVerbose = true

func (bc *BlockChain) theIndex_Hook_WriteBlockHeader(block *types.Block) {
	if theIndexVerbose {
		log.Info("THE-INDEX", "theIndex_Hook_WriteBlockHeader", block.Header().Number)
	}

	if _, err := os.Stat("./the-index/"); os.IsNotExist(err) {
		return
	}

	file, err := os.OpenFile("./the-index/blocks.rlp", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0755)
	if err != nil {
		log.Crit("THE-INDEX", "error", err)
	}
	defer file.Close()

	err = rlp.Encode(file, block.Header().Number)
	if err != nil {
		log.Crit("THE-INDEX", "error", err)
	}

	err = rlp.Encode(file, block.Header().Time)
	if err != nil {
		log.Crit("THE-INDEX", "error", err)
	}

}
