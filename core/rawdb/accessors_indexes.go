// Copyright 2018 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package rawdb

import (
	"encoding/binary"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
)

// ReadTxLookupEntry retrieves the positional metadata associated with a transaction
// hash to allow retrieving the transaction or receipt by hash.
func ReadTxLookupEntry(db ethdb.Reader, hash common.Hash) *uint64 {
	data, _ := db.Get(txLookupKey(hash))
	if len(data) == 0 {
		return nil
	}
	// Database v6 tx lookup just stores the block number
	if len(data) < common.HashLength {
		number := new(big.Int).SetBytes(data).Uint64()
		return &number
	}
	// Database v4-v5 tx lookup format just stores the hash
	if len(data) == common.HashLength {
		return ReadHeaderNumber(db, common.BytesToHash(data))
	}
	// Finally try database v3 tx lookup format
	var entry LegacyTxLookupEntry
	if err := rlp.DecodeBytes(data, &entry); err != nil {
		log.Error("Invalid transaction lookup entry RLP", "hash", hash, "blob", data, "err", err)
		return nil
	}
	return &entry.BlockIndex
}

// writeTxLookupEntry stores a positional metadata for a transaction,
// enabling hash based transaction and receipt lookups.
func writeTxLookupEntry(db ethdb.KeyValueWriter, hash common.Hash, numberBytes []byte) {
	if err := db.Put(txLookupKey(hash), numberBytes); err != nil {
		log.Crit("Failed to store transaction lookup entry", "err", err)
	}
}

// WriteTxLookupEntries is identical to WriteTxLookupEntry, but it works on
// a list of hashes
func WriteTxLookupEntries(db ethdb.KeyValueWriter, number uint64, hashes []common.Hash) {
	numberBytes := new(big.Int).SetUint64(number).Bytes()
	for _, hash := range hashes {
		writeTxLookupEntry(db, hash, numberBytes)
	}
}

// WriteTxLookupEntriesByBlock stores a positional metadata for every transaction from
// a block, enabling hash based transaction and receipt lookups.
func WriteTxLookupEntriesByBlock(db ethdb.KeyValueWriter, block *types.Block) {
	numberBytes := block.Number().Bytes()
	for _, tx := range block.Transactions() {
		writeTxLookupEntry(db, tx.Hash(), numberBytes)
	}
}

// DeleteTxLookupEntry removes all transaction data associated with a hash.
func DeleteTxLookupEntry(db ethdb.KeyValueWriter, hash common.Hash) {
	if err := db.Delete(txLookupKey(hash)); err != nil {
		log.Crit("Failed to delete transaction lookup entry", "err", err)
	}
}

// DeleteTxLookupEntries removes all transaction lookups for a given block.
func DeleteTxLookupEntries(db ethdb.KeyValueWriter, hashes []common.Hash) {
	for _, hash := range hashes {
		DeleteTxLookupEntry(db, hash)
	}
}

// ReadTransaction retrieves a specific transaction from the database, along with
// its added positional metadata.
func ReadTransaction(db ethdb.Reader, hash common.Hash) (*types.Transaction, common.Hash, uint64, uint64) {
	blockNumber := ReadTxLookupEntry(db, hash)
	if blockNumber == nil {
		return nil, common.Hash{}, 0, 0
	}
	blockHash := ReadCanonicalHash(db, *blockNumber)
	if blockHash == (common.Hash{}) {
		return nil, common.Hash{}, 0, 0
	}
	body := ReadBody(db, blockHash, *blockNumber)
	if body == nil {
		log.Error("Transaction referenced missing", "number", *blockNumber, "hash", blockHash)
		return nil, common.Hash{}, 0, 0
	}
	for txIndex, tx := range body.Transactions {
		if tx.Hash() == hash {
			return tx, blockHash, *blockNumber, uint64(txIndex)
		}
	}
	log.Error("Transaction not found", "number", *blockNumber, "hash", blockHash, "txhash", hash)
	return nil, common.Hash{}, 0, 0
}

// ReadReceipt retrieves a specific transaction receipt from the database, along with
// its added positional metadata.
func ReadReceipt(db ethdb.Reader, hash common.Hash, config *params.ChainConfig) (*types.Receipt, common.Hash, uint64, uint64) {
	// Retrieve the context of the receipt based on the transaction hash
	blockNumber := ReadTxLookupEntry(db, hash)
	if blockNumber == nil {
		return nil, common.Hash{}, 0, 0
	}
	blockHash := ReadCanonicalHash(db, *blockNumber)
	if blockHash == (common.Hash{}) {
		return nil, common.Hash{}, 0, 0
	}
	blockHeader := ReadHeader(db, blockHash, *blockNumber)
	if blockHeader == nil {
		return nil, common.Hash{}, 0, 0
	}
	// Read all the receipts from the block and return the one with the matching hash
	receipts := ReadReceipts(db, blockHash, *blockNumber, blockHeader.Time, config)
	for receiptIndex, receipt := range receipts {
		if receipt.TxHash == hash {
			return receipt, blockHash, *blockNumber, uint64(receiptIndex)
		}
	}
	log.Error("Receipt not found", "number", *blockNumber, "hash", blockHash, "txhash", hash)
	return nil, common.Hash{}, 0, 0
}

var emptyRow = []uint32{}

// ReadFilterMapRow retrieves a filter map row at the given mapRowIndex
// (see filtermaps.mapRowIndex for the storage index encoding).
// Note that zero length rows are not stored in the database and therefore all
// non-existent entries are interpreted as empty rows and return no error.
// Also note that the mapRowIndex indexing scheme is the same as the one
// proposed in EIP-7745 for tree-hashing the filter map structure and for the
// same data proximity reasons it is also suitable for database representation.
// See also:
// https://eips.ethereum.org/EIPS/eip-7745#hash-tree-structure
func ReadFilterMapRow(db ethdb.KeyValueReader, mapRowIndex uint64) ([]uint32, error) {
	key := filterMapRowKey(mapRowIndex)
	has, err := db.Has(key)
	if err != nil {
		return nil, err
	}
	if !has {
		return emptyRow, nil
	}
	encRow, err := db.Get(key)
	if err != nil {
		return nil, err
	}
	if len(encRow)&3 != 0 {
		return nil, errors.New("Invalid encoded filter row length")
	}
	row := make([]uint32, len(encRow)/4)
	for i := range row {
		row[i] = binary.LittleEndian.Uint32(encRow[i*4 : (i+1)*4])
	}
	return row, nil
}

// WriteFilterMapRow stores a filter map row at the given mapRowIndex or deletes
// any existing entry if the row is empty.
func WriteFilterMapRow(db ethdb.KeyValueWriter, mapRowIndex uint64, row []uint32) {
	var err error
	if len(row) > 0 {
		encRow := make([]byte, len(row)*4)
		for i, c := range row {
			binary.LittleEndian.PutUint32(encRow[i*4:(i+1)*4], c)
		}
		err = db.Put(filterMapRowKey(mapRowIndex), encRow)
	} else {
		err = db.Delete(filterMapRowKey(mapRowIndex))
	}
	if err != nil {
		log.Crit("Failed to store filter map row", "err", err)
	}
}

// ReadFilterMapBlockPtr retrieves the number of the block that generated the
// first log value entry of the given map.
func ReadFilterMapBlockPtr(db ethdb.KeyValueReader, mapIndex uint32) (uint64, error) {
	encPtr, err := db.Get(filterMapBlockPtrKey(mapIndex))
	if err != nil {
		return 0, err
	}
	if len(encPtr) != 8 {
		return 0, errors.New("Invalid block number encoding")
	}
	return binary.BigEndian.Uint64(encPtr), nil
}

// WriteFilterMapBlockPtr stores the number of the block that generated the
// first log value entry of the given map.
func WriteFilterMapBlockPtr(db ethdb.KeyValueWriter, mapIndex uint32, blockNumber uint64) {
	var encPtr [8]byte
	binary.BigEndian.PutUint64(encPtr[:], blockNumber)
	if err := db.Put(filterMapBlockPtrKey(mapIndex), encPtr[:]); err != nil {
		log.Crit("Failed to store filter map block pointer", "err", err)
	}
}

// DeleteFilterMapBlockPtr deletes the number of the block that generated the
// first log value entry of the given map.
func DeleteFilterMapBlockPtr(db ethdb.KeyValueWriter, mapIndex uint32) {
	if err := db.Delete(filterMapBlockPtrKey(mapIndex)); err != nil {
		log.Crit("Failed to delete filter map block pointer", "err", err)
	}
}

// ReadBlockLvPointer retrieves the starting log value index where the log values
// generated by the given block are located.
func ReadBlockLvPointer(db ethdb.KeyValueReader, blockNumber uint64) (uint64, error) {
	encPtr, err := db.Get(blockLVKey(blockNumber))
	if err != nil {
		return 0, err
	}
	if len(encPtr) != 8 {
		return 0, errors.New("Invalid log value pointer encoding")
	}
	return binary.BigEndian.Uint64(encPtr), nil
}

// WriteBlockLvPointer stores the starting log value index where the log values
// generated by the given block are located.
func WriteBlockLvPointer(db ethdb.KeyValueWriter, blockNumber, lvPointer uint64) {
	var encPtr [8]byte
	binary.BigEndian.PutUint64(encPtr[:], lvPointer)
	if err := db.Put(blockLVKey(blockNumber), encPtr[:]); err != nil {
		log.Crit("Failed to store block log value pointer", "err", err)
	}
}

// DeleteBlockLvPointer deletes the starting log value index where the log values
// generated by the given block are located.
func DeleteBlockLvPointer(db ethdb.KeyValueWriter, blockNumber uint64) {
	if err := db.Delete(blockLVKey(blockNumber)); err != nil {
		log.Crit("Failed to delete block log value pointer", "err", err)
	}
}

// FilterMapsRange is a storage representation of the block range covered by the
// filter maps structure and the corresponting log value index range.
type FilterMapsRange struct {
	Initialized                      bool
	HeadLvPointer, TailLvPointer     uint64
	HeadBlockNumber, TailBlockNumber uint64
	HeadBlockHash, TailParentHash    common.Hash
}

// ReadFilterMapsRange retrieves the filter maps range data. Note that if the
// database entry is not present, that is interpreted as a valid non-initialized
// state and returns a blank range structure and no error.
func ReadFilterMapsRange(db ethdb.KeyValueReader) (FilterMapsRange, error) {
	if has, err := db.Has(filterMapsRangeKey); !has || err != nil {
		return FilterMapsRange{}, err
	}
	encRange, err := db.Get(filterMapsRangeKey)
	if err != nil {
		return FilterMapsRange{}, err
	}
	var fmRange FilterMapsRange
	if err := rlp.DecodeBytes(encRange, &fmRange); err != nil {
		return FilterMapsRange{}, err
	}
	return fmRange, err
}

// WriteFilterMapsRange stores the filter maps range data.
func WriteFilterMapsRange(db ethdb.KeyValueWriter, fmRange FilterMapsRange) {
	encRange, err := rlp.EncodeToBytes(&fmRange)
	if err != nil {
		log.Crit("Failed to encode filter maps range", "err", err)
	}
	if err := db.Put(filterMapsRangeKey, encRange); err != nil {
		log.Crit("Failed to store filter maps range", "err", err)
	}
}

// DeleteFilterMapsRange deletes the filter maps range data which is interpreted
// as reverting to the un-initialized state.
func DeleteFilterMapsRange(db ethdb.KeyValueWriter) {
	if err := db.Delete(filterMapsRangeKey); err != nil {
		log.Crit("Failed to delete filter maps range", "err", err)
	}
}

// RevertPoint is the storage representation of a filter maps revert point.
type RevertPoint struct {
	BlockHash common.Hash
	MapIndex  uint32
	RowLength []uint
}

// ReadRevertPoint retrieves the revert point for the given block number if
// present. Note that revert points may or may not exist for any block number
// and a non-existent entry causes no error.
func ReadRevertPoint(db ethdb.KeyValueReader, blockNumber uint64) (*RevertPoint, error) {
	key := revertPointKey(blockNumber)
	if has, err := db.Has(key); !has || err != nil {
		return nil, err
	}
	enc, err := db.Get(key)
	if err != nil {
		return nil, err
	}
	rp := new(RevertPoint)
	if err := rlp.DecodeBytes(enc, rp); err != nil {
		return nil, err
	}
	return rp, nil
}

// WriteRevertPoint stores a revert point for the given block number.
func WriteRevertPoint(db ethdb.KeyValueWriter, blockNumber uint64, rp *RevertPoint) {
	enc, err := rlp.EncodeToBytes(rp)
	if err != nil {
		log.Crit("Failed to encode revert point", "err", err)
	}
	if err := db.Put(revertPointKey(blockNumber), enc); err != nil {
		log.Crit("Failed to store revert point", "err", err)
	}
}

// DeleteRevertPoint deletes the given revert point.
func DeleteRevertPoint(db ethdb.KeyValueWriter, blockNumber uint64) {
	if err := db.Delete(revertPointKey(blockNumber)); err != nil {
		log.Crit("Failed to delete revert point", "err", err)
	}
}
