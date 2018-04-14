package shyftdb

import (
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/core/types"
)

func WriteBlock(db ethdb.Putter, block *types.Block) error {
	return nil
}