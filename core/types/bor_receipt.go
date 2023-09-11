package types

import (
	"encoding/binary"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// TenToTheFive - To be used while sorting bor logs
//
// Sorted using ( blockNumber * (10 ** 5) + logIndex )
const TenToTheFive uint64 = 100000

var (
	borReceiptPrefix = []byte("matic-bor-receipt-") // borReceiptPrefix + number + block hash -> bor block receipt

	// SystemAddress address for system sender
	SystemAddress = common.HexToAddress("0xffffFFFfFFffffffffffffffFfFFFfffFFFfFFfE")
)

// BorReceiptKey = borReceiptPrefix + num (uint64 big endian) + hash
func BorReceiptKey(number uint64, hash common.Hash) []byte {
	enc := make([]byte, 8)
	binary.BigEndian.PutUint64(enc, number)
	return append(append(borReceiptPrefix, enc...), hash.Bytes()...)
}

// GetDerivedBorTxHash get derived tx hash from receipt key
func GetDerivedBorTxHash(receiptKey []byte) common.Hash {
	return common.BytesToHash(crypto.Keccak256(receiptKey))
}

// NewBorTransaction create new bor transaction for bor receipt
func NewBorTransaction() *Transaction {
	return NewTransaction(0, common.Address{}, big.NewInt(0), 0, big.NewInt(0), make([]byte, 0))
}

// DeriveFieldsForBorReceipt fills the receipts with their computed fields based on consensus
// data and contextual infos like containing block and transactions.
func DeriveFieldsForBorReceipt(receipt *Receipt, hash common.Hash, number uint64, receipts Receipts) error {
	// get derived tx hash
	txHash := GetDerivedBorTxHash(BorReceiptKey(number, hash))
	txIndex := uint(len(receipts))

	// set tx hash and tx index
	receipt.TxHash = txHash
	receipt.TransactionIndex = txIndex
	receipt.BlockHash = hash
	receipt.BlockNumber = big.NewInt(0).SetUint64(number)

	logIndex := 0
	for i := 0; i < len(receipts); i++ {
		logIndex += len(receipts[i].Logs)
	}

	// The derived log fields can simply be set from the block and transaction
	for j := 0; j < len(receipt.Logs); j++ {
		receipt.Logs[j].BlockNumber = number
		receipt.Logs[j].BlockHash = hash
		receipt.Logs[j].TxHash = txHash
		receipt.Logs[j].TxIndex = txIndex
		receipt.Logs[j].Index = uint(logIndex)
		logIndex++
	}
	return nil
}

// DeriveFieldsForBorLogs fills the receipts with their computed fields based on consensus
// data and contextual infos like containing block and transactions.
func DeriveFieldsForBorLogs(logs []*Log, hash common.Hash, number uint64, txIndex uint, logIndex uint) {
	// get derived tx hash
	txHash := GetDerivedBorTxHash(BorReceiptKey(number, hash))

	// the derived log fields can simply be set from the block and transaction
	for j := 0; j < len(logs); j++ {
		logs[j].BlockNumber = number
		logs[j].BlockHash = hash
		logs[j].TxHash = txHash
		logs[j].TxIndex = txIndex
		logs[j].Index = logIndex
		logIndex++
	}
}

// MergeBorLogs merges receipt logs and block receipt logs
func MergeBorLogs(logs []*Log, borLogs []*Log) []*Log {
	result := append(logs, borLogs...)

	sort.SliceStable(result, func(i int, j int) bool {
		return (result[i].BlockNumber*TenToTheFive + uint64(result[i].Index)) < (result[j].BlockNumber*TenToTheFive + uint64(result[j].Index))
	})

	return result
}
