package core

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/rlp"
)

var receiptsPre = []byte("receipts-")

// PutTransactions stores the transactions in the given database
func PutTransactions(db common.Database, block *types.Block, txs types.Transactions) {
	for i, tx := range block.Transactions() {
		rlpEnc, err := rlp.EncodeToBytes(tx)
		if err != nil {
			glog.V(logger.Debug).Infoln("Failed encoding tx", err)
			return
		}
		db.Put(tx.Hash().Bytes(), rlpEnc)

		var txExtra struct {
			BlockHash  common.Hash
			BlockIndex uint64
			Index      uint64
		}
		txExtra.BlockHash = block.Hash()
		txExtra.BlockIndex = block.NumberU64()
		txExtra.Index = uint64(i)
		rlpMeta, err := rlp.EncodeToBytes(txExtra)
		if err != nil {
			glog.V(logger.Debug).Infoln("Failed encoding tx meta data", err)
			return
		}
		db.Put(append(tx.Hash().Bytes(), 0x0001), rlpMeta)
	}
}

// PutReceipts stores the receipts in the current database
func PutReceipts(db common.Database, receipts types.Receipts) error {
	for _, receipt := range receipts {
		storageReceipt := (*types.ReceiptForStorage)(receipt)
		bytes, err := rlp.EncodeToBytes(storageReceipt)
		if err != nil {
			return err
		}
		err = db.Put(append(receiptsPre, receipt.TxHash[:]...), bytes)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetReceipt returns a receipt by hash
func GetReceipt(db common.Database, txHash common.Hash) *types.Receipt {
	data, _ := db.Get(append(receiptsPre, txHash[:]...))
	if len(data) == 0 {
		return nil
	}

	var receipt types.Receipt
	err := rlp.DecodeBytes(data, &receipt)
	if err != nil {
		glog.V(logger.Error).Infoln("GetReceipt err:", err)
	}
	return &receipt
}

// GetReceiptFromBlock returns all receipts with the given block
func GetReceiptsFromBlock(db common.Database, block *types.Block) types.Receipts {
	// at some point we want:
	//receipts := make(types.Receipts, len(block.Transactions()))
	// but since we need to support legacy, we can't (yet)
	var receipts types.Receipts
	for _, tx := range block.Transactions() {
		if receipt := GetReceipt(db, tx.Hash()); receipt != nil {
			receipts = append(receipts, receipt)
		}
	}

	return receipts
}
