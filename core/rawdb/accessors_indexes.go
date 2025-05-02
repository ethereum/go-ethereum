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
	"bytes"
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

// DecodeTxLookupEntry decodes the supplied tx lookup data.
func DecodeTxLookupEntry(data []byte, db ethdb.Reader) *uint64 {
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
		log.Error("Invalid transaction lookup entry RLP", "blob", data, "err", err)
		return nil
	}
	return &entry.BlockIndex
}

// ReadTxLookupEntry retrieves the positional metadata associated with a transaction
// hash to allow retrieving the transaction or receipt by hash.
func ReadTxLookupEntry(db ethdb.Reader, hash common.Hash) *uint64 {
	data, _ := db.Get(txLookupKey(hash))
	if len(data) == 0 {
		return nil
	}
	return DecodeTxLookupEntry(data, db)
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

// DeleteAllTxLookupEntries purges all the transaction indexes in the database.
// If condition is specified, only the entry with condition as True will be
// removed; If condition is not specified, the entry is deleted.
func DeleteAllTxLookupEntries(db ethdb.KeyValueStore, condition func(common.Hash, []byte) bool) {
	iter := NewKeyLengthIterator(db.NewIterator(txLookupPrefix, nil), common.HashLength+len(txLookupPrefix))
	defer iter.Release()

	batch := db.NewBatch()
	for iter.Next() {
		txhash := common.Hash(iter.Key()[1:])
		if condition == nil || condition(txhash, iter.Value()) {
			batch.Delete(iter.Key())
		}
		if batch.ValueSize() >= ethdb.IdealBatchSize {
			if err := batch.Write(); err != nil {
				log.Crit("Failed to delete transaction lookup entries", "err", err)
			}
			batch.Reset()
		}
	}
	if batch.ValueSize() > 0 {
		if err := batch.Write(); err != nil {
			log.Crit("Failed to delete transaction lookup entries", "err", err)
		}
		batch.Reset()
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

// ReadFilterMapRow retrieves a filter map row at the given mapRowIndex
// (see filtermaps.mapRowIndex for the storage index encoding).
// Note that zero length rows are not stored in the database and therefore all
// non-existent entries are interpreted as empty rows and return no error.
// Also note that the mapRowIndex indexing scheme is the same as the one
// proposed in EIP-7745 for tree-hashing the filter map structure and for the
// same data proximity reasons it is also suitable for database representation.
// See also:
// https://eips.ethereum.org/EIPS/eip-7745#hash-tree-structure
func ReadFilterMapExtRow(db ethdb.KeyValueReader, mapRowIndex uint64, bitLength uint) ([]uint32, error) {
	byteLength := int(bitLength) / 8
	if int(bitLength) != byteLength*8 {
		panic("invalid bit length")
	}
	key := filterMapRowKey(mapRowIndex, false)
	has, err := db.Has(key)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, nil
	}
	encRow, err := db.Get(key)
	if err != nil {
		return nil, err
	}
	if len(encRow)%byteLength != 0 {
		return nil, errors.New("Invalid encoded extended filter row length")
	}
	row := make([]uint32, len(encRow)/byteLength)
	var b [4]byte
	for i := range row {
		copy(b[:byteLength], encRow[i*byteLength:(i+1)*byteLength])
		row[i] = binary.LittleEndian.Uint32(b[:])
	}
	return row, nil
}

func ReadFilterMapBaseRows(db ethdb.KeyValueReader, mapRowIndex uint64, rowCount uint32, bitLength uint) ([][]uint32, error) {
	byteLength := int(bitLength) / 8
	if int(bitLength) != byteLength*8 {
		panic("invalid bit length")
	}
	key := filterMapRowKey(mapRowIndex, true)
	has, err := db.Has(key)
	if err != nil {
		return nil, err
	}
	rows := make([][]uint32, rowCount)
	if !has {
		return rows, nil
	}
	encRows, err := db.Get(key)
	if err != nil {
		return nil, err
	}
	encLen := len(encRows)
	var (
		entryCount, entriesInRow, rowIndex, headerLen, headerBits int
		headerByte                                                byte
	)
	for headerLen+byteLength*entryCount < encLen {
		if headerBits == 0 {
			headerByte = encRows[headerLen]
			headerLen++
			headerBits = 8
		}
		if headerByte&1 > 0 {
			entriesInRow++
			entryCount++
		} else {
			if entriesInRow > 0 {
				rows[rowIndex] = make([]uint32, entriesInRow)
				entriesInRow = 0
			}
			rowIndex++
		}
		headerByte >>= 1
		headerBits--
	}
	if headerLen+byteLength*entryCount > encLen {
		return nil, errors.New("Invalid encoded base filter rows length")
	}
	if entriesInRow > 0 {
		rows[rowIndex] = make([]uint32, entriesInRow)
	}
	nextEntry := headerLen
	for _, row := range rows {
		for i := range row {
			var b [4]byte
			copy(b[:byteLength], encRows[nextEntry:nextEntry+byteLength])
			row[i] = binary.LittleEndian.Uint32(b[:])
			nextEntry += byteLength
		}
	}
	return rows, nil
}

// WriteFilterMapRow stores a filter map row at the given mapRowIndex or deletes
// any existing entry if the row is empty.
func WriteFilterMapExtRow(db ethdb.KeyValueWriter, mapRowIndex uint64, row []uint32, bitLength uint) {
	byteLength := int(bitLength) / 8
	if int(bitLength) != byteLength*8 {
		panic("invalid bit length")
	}
	var err error
	if len(row) > 0 {
		encRow := make([]byte, len(row)*byteLength)
		for i, c := range row {
			var b [4]byte
			binary.LittleEndian.PutUint32(b[:], c)
			copy(encRow[i*byteLength:(i+1)*byteLength], b[:byteLength])
		}
		err = db.Put(filterMapRowKey(mapRowIndex, false), encRow)
	} else {
		err = db.Delete(filterMapRowKey(mapRowIndex, false))
	}
	if err != nil {
		log.Crit("Failed to store extended filter map row", "err", err)
	}
}

func WriteFilterMapBaseRows(db ethdb.KeyValueWriter, mapRowIndex uint64, rows [][]uint32, bitLength uint) {
	byteLength := int(bitLength) / 8
	if int(bitLength) != byteLength*8 {
		panic("invalid bit length")
	}
	var entryCount, zeroBits int
	for i, row := range rows {
		if len(row) > 0 {
			entryCount += len(row)
			zeroBits = i
		}
	}
	var err error
	if entryCount > 0 {
		headerLen := (zeroBits + entryCount + 7) / 8
		encRows := make([]byte, headerLen+entryCount*byteLength)
		nextEntry := headerLen

		headerPtr, headerByte := 0, byte(1)
		addHeaderBit := func(bit bool) {
			if bit {
				encRows[headerPtr] += headerByte
			}
			if headerByte += headerByte; headerByte == 0 {
				headerPtr++
				headerByte = 1
			}
		}

		for _, row := range rows {
			for _, entry := range row {
				var b [4]byte
				binary.LittleEndian.PutUint32(b[:], entry)
				copy(encRows[nextEntry:nextEntry+byteLength], b[:byteLength])
				nextEntry += byteLength
				addHeaderBit(true)
			}
			if zeroBits == 0 {
				break
			}
			addHeaderBit(false)
			zeroBits--
		}
		err = db.Put(filterMapRowKey(mapRowIndex, true), encRows)
	} else {
		err = db.Delete(filterMapRowKey(mapRowIndex, true))
	}
	if err != nil {
		log.Crit("Failed to store base filter map rows", "err", err)
	}
}

func DeleteFilterMapRows(db ethdb.KeyValueStore, mapRows common.Range[uint64], hashScheme bool, stopCallback func(bool) bool) error {
	return SafeDeleteRange(db, filterMapRowKey(mapRows.First(), false), filterMapRowKey(mapRows.AfterLast(), false), hashScheme, stopCallback)
}

// ReadFilterMapLastBlock retrieves the number of the block that generated the
// last log value entry of the given map.
func ReadFilterMapLastBlock(db ethdb.KeyValueReader, mapIndex uint32) (uint64, common.Hash, error) {
	enc, err := db.Get(filterMapLastBlockKey(mapIndex))
	if err != nil {
		return 0, common.Hash{}, err
	}
	if len(enc) != 40 {
		return 0, common.Hash{}, errors.New("invalid block number and id encoding")
	}
	var id common.Hash
	copy(id[:], enc[8:])
	return binary.BigEndian.Uint64(enc[:8]), id, nil
}

// WriteFilterMapLastBlock stores the number of the block that generated the
// last log value entry of the given map.
func WriteFilterMapLastBlock(db ethdb.KeyValueWriter, mapIndex uint32, blockNumber uint64, id common.Hash) {
	var enc [40]byte
	binary.BigEndian.PutUint64(enc[:8], blockNumber)
	copy(enc[8:], id[:])
	if err := db.Put(filterMapLastBlockKey(mapIndex), enc[:]); err != nil {
		log.Crit("Failed to store filter map last block pointer", "err", err)
	}
}

// DeleteFilterMapLastBlock deletes the number of the block that generated the
// last log value entry of the given map.
func DeleteFilterMapLastBlock(db ethdb.KeyValueWriter, mapIndex uint32) {
	if err := db.Delete(filterMapLastBlockKey(mapIndex)); err != nil {
		log.Crit("Failed to delete filter map last block pointer", "err", err)
	}
}

func DeleteFilterMapLastBlocks(db ethdb.KeyValueStore, maps common.Range[uint32], hashScheme bool, stopCallback func(bool) bool) error {
	return SafeDeleteRange(db, filterMapLastBlockKey(maps.First()), filterMapLastBlockKey(maps.AfterLast()), hashScheme, stopCallback)
}

// ReadBlockLvPointer retrieves the starting log value index where the log values
// generated by the given block are located.
func ReadBlockLvPointer(db ethdb.KeyValueReader, blockNumber uint64) (uint64, error) {
	encPtr, err := db.Get(filterMapBlockLVKey(blockNumber))
	if err != nil {
		return 0, err
	}
	if len(encPtr) != 8 {
		return 0, errors.New("invalid log value pointer encoding")
	}
	return binary.BigEndian.Uint64(encPtr), nil
}

// WriteBlockLvPointer stores the starting log value index where the log values
// generated by the given block are located.
func WriteBlockLvPointer(db ethdb.KeyValueWriter, blockNumber, lvPointer uint64) {
	var encPtr [8]byte
	binary.BigEndian.PutUint64(encPtr[:], lvPointer)
	if err := db.Put(filterMapBlockLVKey(blockNumber), encPtr[:]); err != nil {
		log.Crit("Failed to store block log value pointer", "err", err)
	}
}

// DeleteBlockLvPointer deletes the starting log value index where the log values
// generated by the given block are located.
func DeleteBlockLvPointer(db ethdb.KeyValueWriter, blockNumber uint64) {
	if err := db.Delete(filterMapBlockLVKey(blockNumber)); err != nil {
		log.Crit("Failed to delete block log value pointer", "err", err)
	}
}

func DeleteBlockLvPointers(db ethdb.KeyValueStore, blocks common.Range[uint64], hashScheme bool, stopCallback func(bool) bool) error {
	return SafeDeleteRange(db, filterMapBlockLVKey(blocks.First()), filterMapBlockLVKey(blocks.AfterLast()), hashScheme, stopCallback)
}

// FilterMapsRange is a storage representation of the block range covered by the
// filter maps structure and the corresponting log value index range.
type FilterMapsRange struct {
	Version                      uint32
	HeadIndexed                  bool
	HeadDelimiter                uint64
	BlocksFirst, BlocksAfterLast uint64
	MapsFirst, MapsAfterLast     uint32
	TailPartialEpoch             uint32
}

// ReadFilterMapsRange retrieves the filter maps range data. Note that if the
// database entry is not present, that is interpreted as a valid non-initialized
// state and returns a blank range structure and no error.
func ReadFilterMapsRange(db ethdb.KeyValueReader) (FilterMapsRange, bool, error) {
	if has, err := db.Has(filterMapsRangeKey); !has || err != nil {
		return FilterMapsRange{}, false, err
	}
	encRange, err := db.Get(filterMapsRangeKey)
	if err != nil {
		return FilterMapsRange{}, false, err
	}
	var fmRange FilterMapsRange
	if err := rlp.DecodeBytes(encRange, &fmRange); err != nil {
		return FilterMapsRange{}, false, err
	}
	return fmRange, true, err
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

// deletePrefixRange deletes everything with the given prefix from the database.
func deletePrefixRange(db ethdb.KeyValueStore, prefix []byte, hashScheme bool, stopCallback func(bool) bool) error {
	end := bytes.Clone(prefix)
	end[len(end)-1]++
	return SafeDeleteRange(db, prefix, end, hashScheme, stopCallback)
}

// DeleteFilterMapsDb removes the entire filter maps database
func DeleteFilterMapsDb(db ethdb.KeyValueStore, hashScheme bool, stopCallback func(bool) bool) error {
	return deletePrefixRange(db, []byte(filterMapsPrefix), hashScheme, stopCallback)
}

// DeleteBloomBitsDb removes the old bloombits database and the associated
// chain indexer database.
func DeleteBloomBitsDb(db ethdb.KeyValueStore, hashScheme bool, stopCallback func(bool) bool) error {
	if err := deletePrefixRange(db, bloomBitsPrefix, hashScheme, stopCallback); err != nil {
		return err
	}
	return deletePrefixRange(db, bloomBitsMetaPrefix, hashScheme, stopCallback)
}
