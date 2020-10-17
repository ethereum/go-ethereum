package core

import (
	"github.com/maticnetwork/bor/common"
	"github.com/maticnetwork/bor/core/rawdb"
	"github.com/maticnetwork/bor/core/types"
)

// GetBorReceiptByHash retrieves the bor block receipt in a given block.
func (bc *BlockChain) GetBorReceiptByHash(hash common.Hash) *types.BorReceipt {
	if receipt, ok := bc.borReceiptsCache.Get(hash); ok {
		return receipt.(*types.BorReceipt)
	}

	// read header from hash
	number := rawdb.ReadHeaderNumber(bc.db, hash)
	if number == nil {
		return nil
	}

	// read bor reciept by hash and number
	receipt := rawdb.ReadBorReceipt(bc.db, hash, *number)
	if receipt == nil {
		return nil
	}

	// add into bor receipt cache
	bc.borReceiptsCache.Add(hash, receipt)
	return receipt
}
