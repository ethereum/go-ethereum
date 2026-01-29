// Copyright 2021 XDC Network
// This file is part of the XDC library.

package rawdb

import (
	"encoding/binary"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
)

// XDC-specific database key prefixes
var (
	validatorSetPrefix = []byte("xdc-validators-")
	epochDataPrefix    = []byte("xdc-epoch-")
	penaltyPrefix      = []byte("xdc-penalty-")
	checkpointPrefix   = []byte("xdc-checkpoint-")
	snapshotPrefix     = []byte("xdc-snapshot-")
	tradingStatePrefix = []byte("xdcx-trading-")
	lendingStatePrefix = []byte("xdcx-lending-")
)

// encodeBlockNumber encodes a block number as big endian uint64
func encodeBlockNumber(number uint64) []byte {
	enc := make([]byte, 8)
	binary.BigEndian.PutUint64(enc, number)
	return enc
}

// validatorSetKey returns the key for validator set at block number
func validatorSetKey(number uint64) []byte {
	return append(validatorSetPrefix, encodeBlockNumber(number)...)
}

// epochDataKey returns the key for epoch data
func epochDataKey(epoch uint64) []byte {
	return append(epochDataPrefix, encodeBlockNumber(epoch)...)
}

// penaltyKey returns the key for a penalty record
func penaltyKey(validator common.Address, block uint64) []byte {
	key := append(penaltyPrefix, validator.Bytes()...)
	return append(key, encodeBlockNumber(block)...)
}

// checkpointKey returns the key for a checkpoint
func checkpointKey(number uint64) []byte {
	return append(checkpointPrefix, encodeBlockNumber(number)...)
}

// snapshotKey returns the key for a snapshot
func snapshotKey(hash common.Hash) []byte {
	return append(snapshotPrefix, hash.Bytes()...)
}

// WriteValidatorSet writes the validator set for a block
func WriteValidatorSet(db ethdb.KeyValueWriter, number uint64, validators []common.Address) error {
	data := make([]byte, len(validators)*common.AddressLength)
	for i, v := range validators {
		copy(data[i*common.AddressLength:], v.Bytes())
	}
	return db.Put(validatorSetKey(number), data)
}

// ReadValidatorSet reads the validator set for a block
func ReadValidatorSet(db ethdb.KeyValueReader, number uint64) []common.Address {
	data, err := db.Get(validatorSetKey(number))
	if err != nil || len(data) == 0 {
		return nil
	}

	count := len(data) / common.AddressLength
	validators := make([]common.Address, count)
	for i := 0; i < count; i++ {
		validators[i] = common.BytesToAddress(data[i*common.AddressLength : (i+1)*common.AddressLength])
	}
	return validators
}

// DeleteValidatorSet deletes the validator set for a block
func DeleteValidatorSet(db ethdb.KeyValueWriter, number uint64) error {
	return db.Delete(validatorSetKey(number))
}

// WriteEpochData writes epoch data
func WriteEpochData(db ethdb.KeyValueWriter, epoch uint64, data []byte) error {
	return db.Put(epochDataKey(epoch), data)
}

// ReadEpochData reads epoch data
func ReadEpochData(db ethdb.KeyValueReader, epoch uint64) []byte {
	data, err := db.Get(epochDataKey(epoch))
	if err != nil {
		return nil
	}
	return data
}

// DeleteEpochData deletes epoch data
func DeleteEpochData(db ethdb.KeyValueWriter, epoch uint64) error {
	return db.Delete(epochDataKey(epoch))
}

// WritePenalty writes a penalty record
func WritePenalty(db ethdb.KeyValueWriter, validator common.Address, block uint64, amount uint64) error {
	data := encodeBlockNumber(amount)
	return db.Put(penaltyKey(validator, block), data)
}

// ReadPenalty reads a penalty record
func ReadPenalty(db ethdb.KeyValueReader, validator common.Address, block uint64) uint64 {
	data, err := db.Get(penaltyKey(validator, block))
	if err != nil || len(data) != 8 {
		return 0
	}
	return binary.BigEndian.Uint64(data)
}

// DeletePenalty deletes a penalty record
func DeletePenalty(db ethdb.KeyValueWriter, validator common.Address, block uint64) error {
	return db.Delete(penaltyKey(validator, block))
}

// WriteCheckpoint writes a checkpoint
func WriteCheckpoint(db ethdb.KeyValueWriter, number uint64, hash common.Hash) error {
	return db.Put(checkpointKey(number), hash.Bytes())
}

// ReadCheckpoint reads a checkpoint
func ReadCheckpoint(db ethdb.KeyValueReader, number uint64) common.Hash {
	data, err := db.Get(checkpointKey(number))
	if err != nil || len(data) != common.HashLength {
		return common.Hash{}
	}
	return common.BytesToHash(data)
}

// DeleteCheckpoint deletes a checkpoint
func DeleteCheckpoint(db ethdb.KeyValueWriter, number uint64) error {
	return db.Delete(checkpointKey(number))
}

// WriteXDPoSSnapshot writes an XDPoS snapshot
func WriteXDPoSSnapshot(db ethdb.KeyValueWriter, hash common.Hash, data []byte) error {
	return db.Put(snapshotKey(hash), data)
}

// ReadXDPoSSnapshot reads an XDPoS snapshot
func ReadXDPoSSnapshot(db ethdb.KeyValueReader, hash common.Hash) []byte {
	data, err := db.Get(snapshotKey(hash))
	if err != nil {
		return nil
	}
	return data
}

// DeleteXDPoSSnapshot deletes an XDPoS snapshot
func DeleteXDPoSSnapshot(db ethdb.KeyValueWriter, hash common.Hash) error {
	return db.Delete(snapshotKey(hash))
}

// HasXDPoSSnapshot checks if a snapshot exists
func HasXDPoSSnapshot(db ethdb.KeyValueReader, hash common.Hash) bool {
	has, _ := db.Has(snapshotKey(hash))
	return has
}

// XDCx Trading State

// tradingStateKey returns the key for trading state
func tradingStateKey(root common.Hash) []byte {
	return append(tradingStatePrefix, root.Bytes()...)
}

// WriteTradingStateRoot writes the trading state root for a block
func WriteTradingStateRoot(db ethdb.KeyValueWriter, blockHash, tradingRoot common.Hash) error {
	return db.Put(tradingStateKey(blockHash), tradingRoot.Bytes())
}

// ReadTradingStateRoot reads the trading state root for a block
func ReadTradingStateRoot(db ethdb.KeyValueReader, blockHash common.Hash) common.Hash {
	data, err := db.Get(tradingStateKey(blockHash))
	if err != nil || len(data) != common.HashLength {
		return common.Hash{}
	}
	return common.BytesToHash(data)
}

// XDCx Lending State

// lendingStateKey returns the key for lending state
func lendingStateKey(root common.Hash) []byte {
	return append(lendingStatePrefix, root.Bytes()...)
}

// WriteLendingStateRoot writes the lending state root for a block
func WriteLendingStateRoot(db ethdb.KeyValueWriter, blockHash, lendingRoot common.Hash) error {
	return db.Put(lendingStateKey(blockHash), lendingRoot.Bytes())
}

// ReadLendingStateRoot reads the lending state root for a block
func ReadLendingStateRoot(db ethdb.KeyValueReader, blockHash common.Hash) common.Hash {
	data, err := db.Get(lendingStateKey(blockHash))
	if err != nil || len(data) != common.HashLength {
		return common.Hash{}
	}
	return common.BytesToHash(data)
}
