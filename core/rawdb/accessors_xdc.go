// Copyright 2023 The XDC Network Authors
// This file is part of the XDC Network library.
//
// The XDC Network library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package rawdb

import (
	"encoding/binary"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

// XDPoS-specific database key prefixes
var (
	// Masternode data
	masternodePrefix      = []byte("masternode-")
	masternodeListPrefix  = []byte("masternodelist-")
	
	// Epoch data
	epochPrefix           = []byte("epoch-")
	epochSnapshotPrefix   = []byte("epochsnapshot-")
	
	// Penalty data
	penaltyPrefix         = []byte("penalty-")
	penalizedListPrefix   = []byte("penalizedlist-")
	
	// Reward data
	rewardPrefix          = []byte("reward-")
	rewardEpochPrefix     = []byte("rewardepoch-")
	
	// Block signer data
	blockSignerPrefix     = []byte("blocksigner-")
	
	// Vote data
	votePrefix            = []byte("vote-")
	
	// Trading state root prefix
	tradingStatePrefix    = []byte("tradingstate-")
	
	// Lending state root prefix
	lendingStatePrefix    = []byte("lendingstate-")
	
	// QC (Quorum Certificate) data
	qcPrefix              = []byte("qc-")
	
	// Timeout pool data
	timeoutPoolPrefix     = []byte("timeoutpool-")
)

// encodeBlockNumber encodes a block number as big endian uint64
func encodeXDCBlockNumber(number uint64) []byte {
	enc := make([]byte, 8)
	binary.BigEndian.PutUint64(enc, number)
	return enc
}

// MasternodeKey returns the database key for a masternode entry
func MasternodeKey(address common.Address) []byte {
	return append(masternodePrefix, address.Bytes()...)
}

// MasternodeListKey returns the database key for masternode list at epoch
func MasternodeListKey(epoch uint64) []byte {
	return append(masternodeListPrefix, encodeBlockNumber(epoch)...)
}

// EpochKey returns the database key for epoch data
func EpochKey(epoch uint64) []byte {
	return append(epochPrefix, encodeBlockNumber(epoch)...)
}

// EpochSnapshotKey returns the database key for epoch snapshot
func EpochSnapshotKey(epoch uint64) []byte {
	return append(epochSnapshotPrefix, encodeBlockNumber(epoch)...)
}

// PenaltyKey returns the database key for penalty data at epoch
func PenaltyKey(epoch uint64) []byte {
	return append(penaltyPrefix, encodeBlockNumber(epoch)...)
}

// PenalizedListKey returns the database key for penalized validators list
func PenalizedListKey(epoch uint64) []byte {
	return append(penalizedListPrefix, encodeBlockNumber(epoch)...)
}

// RewardKey returns the database key for reward at block number
func RewardKey(number uint64) []byte {
	return append(rewardPrefix, encodeXDCBlockNumber(number)...)
}

// RewardEpochKey returns the database key for epoch reward
func RewardEpochKey(epoch uint64) []byte {
	return append(rewardEpochPrefix, encodeBlockNumber(epoch)...)
}

// BlockSignerKey returns the database key for block signer at block number
func BlockSignerKey(number uint64) []byte {
	return append(blockSignerPrefix, encodeXDCBlockNumber(number)...)
}

// VoteKey returns the database key for vote data
func VoteKey(hash common.Hash) []byte {
	return append(votePrefix, hash.Bytes()...)
}

// TradingStateKey returns the database key for trading state root
func TradingStateKey(hash common.Hash) []byte {
	return append(tradingStatePrefix, hash.Bytes()...)
}

// LendingStateKey returns the database key for lending state root
func LendingStateKey(hash common.Hash) []byte {
	return append(lendingStatePrefix, hash.Bytes()...)
}

// QCKey returns the database key for quorum certificate
func QCKey(hash common.Hash) []byte {
	return append(qcPrefix, hash.Bytes()...)
}

// TimeoutPoolKey returns the database key for timeout pool at round
func TimeoutPoolKey(round uint64) []byte {
	return append(timeoutPoolPrefix, encodeBlockNumber(round)...)
}

// ReadMasternodeList reads the masternode list for an epoch
func ReadMasternodeList(db ethdb.Reader, epoch uint64) []common.Address {
	data, err := db.Get(MasternodeListKey(epoch))
	if err != nil || len(data) == 0 {
		return nil
	}
	
	var masternodes []common.Address
	if err := rlp.DecodeBytes(data, &masternodes); err != nil {
		log.Error("Invalid masternode list RLP", "epoch", epoch, "err", err)
		return nil
	}
	return masternodes
}

// WriteMasternodeList writes the masternode list for an epoch
func WriteMasternodeList(db ethdb.KeyValueWriter, epoch uint64, masternodes []common.Address) {
	data, err := rlp.EncodeToBytes(masternodes)
	if err != nil {
		log.Crit("Failed to RLP encode masternode list", "err", err)
	}
	if err := db.Put(MasternodeListKey(epoch), data); err != nil {
		log.Crit("Failed to store masternode list", "err", err)
	}
}

// ReadPenalizedList reads the penalized validators list for an epoch
func ReadPenalizedList(db ethdb.Reader, epoch uint64) []common.Address {
	data, err := db.Get(PenalizedListKey(epoch))
	if err != nil || len(data) == 0 {
		return nil
	}
	
	var penalized []common.Address
	if err := rlp.DecodeBytes(data, &penalized); err != nil {
		log.Error("Invalid penalized list RLP", "epoch", epoch, "err", err)
		return nil
	}
	return penalized
}

// WritePenalizedList writes the penalized validators list for an epoch
func WritePenalizedList(db ethdb.KeyValueWriter, epoch uint64, penalized []common.Address) {
	data, err := rlp.EncodeToBytes(penalized)
	if err != nil {
		log.Crit("Failed to RLP encode penalized list", "err", err)
	}
	if err := db.Put(PenalizedListKey(epoch), data); err != nil {
		log.Crit("Failed to store penalized list", "err", err)
	}
}

// ReadBlockSigner reads the block signer for a block number
func ReadBlockSigner(db ethdb.Reader, number uint64) common.Address {
	data, err := db.Get(BlockSignerKey(number))
	if err != nil || len(data) == 0 {
		return common.Address{}
	}
	return common.BytesToAddress(data)
}

// WriteBlockSigner writes the block signer for a block number
func WriteBlockSigner(db ethdb.KeyValueWriter, number uint64, signer common.Address) {
	if err := db.Put(BlockSignerKey(number), signer.Bytes()); err != nil {
		log.Crit("Failed to store block signer", "err", err)
	}
}

// ReadTradingStateRoot reads the trading state root for a block
func ReadTradingStateRoot(db ethdb.Reader, blockHash common.Hash) common.Hash {
	data, err := db.Get(TradingStateKey(blockHash))
	if err != nil || len(data) == 0 {
		return common.Hash{}
	}
	return common.BytesToHash(data)
}

// WriteTradingStateRoot writes the trading state root for a block
func WriteTradingStateRoot(db ethdb.KeyValueWriter, blockHash common.Hash, root common.Hash) {
	if err := db.Put(TradingStateKey(blockHash), root.Bytes()); err != nil {
		log.Crit("Failed to store trading state root", "err", err)
	}
}

// ReadLendingStateRoot reads the lending state root for a block
func ReadLendingStateRoot(db ethdb.Reader, blockHash common.Hash) common.Hash {
	data, err := db.Get(LendingStateKey(blockHash))
	if err != nil || len(data) == 0 {
		return common.Hash{}
	}
	return common.BytesToHash(data)
}

// WriteLendingStateRoot writes the lending state root for a block
func WriteLendingStateRoot(db ethdb.KeyValueWriter, blockHash common.Hash, root common.Hash) {
	if err := db.Put(LendingStateKey(blockHash), root.Bytes()); err != nil {
		log.Crit("Failed to store lending state root", "err", err)
	}
}

// EpochData stores epoch-specific information
type EpochData struct {
	Epoch       uint64           `json:"epoch"`
	StartBlock  uint64           `json:"startBlock"`
	EndBlock    uint64           `json:"endBlock"`
	Masternodes []common.Address `json:"masternodes"`
	Penalties   []common.Address `json:"penalties"`
}

// ReadEpochData reads epoch data from database
func ReadEpochData(db ethdb.Reader, epoch uint64) *EpochData {
	data, err := db.Get(EpochKey(epoch))
	if err != nil || len(data) == 0 {
		return nil
	}
	
	var epochData EpochData
	if err := rlp.DecodeBytes(data, &epochData); err != nil {
		log.Error("Invalid epoch data RLP", "epoch", epoch, "err", err)
		return nil
	}
	return &epochData
}

// WriteEpochData writes epoch data to database
func WriteEpochData(db ethdb.KeyValueWriter, epoch uint64, epochData *EpochData) {
	data, err := rlp.EncodeToBytes(epochData)
	if err != nil {
		log.Crit("Failed to RLP encode epoch data", "err", err)
	}
	if err := db.Put(EpochKey(epoch), data); err != nil {
		log.Crit("Failed to store epoch data", "err", err)
	}
}

// HasEpochData checks if epoch data exists
func HasEpochData(db ethdb.Reader, epoch uint64) bool {
	has, _ := db.Has(EpochKey(epoch))
	return has
}

// DeleteEpochData deletes epoch data
func DeleteEpochData(db ethdb.KeyValueWriter, epoch uint64) {
	if err := db.Delete(EpochKey(epoch)); err != nil {
		log.Crit("Failed to delete epoch data", "err", err)
	}
}
