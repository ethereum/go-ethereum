package shyftdb

import (
	"fmt"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/core/types"
)

func WriteBlock(db ethdb.Putter, block *types.Block) error {
	return nil
}