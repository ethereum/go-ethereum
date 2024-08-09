// Copyright 2024 The go-ethereum Authors
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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

// ReadStateHistoryIndexHead retrieves the number of latest indexed state history.
func ReadStateHistoryIndexHead(db ethdb.KeyValueReader) *uint64 {
	data, _ := db.Get(stateHistoryIndexHeadKey)
	if len(data) != 8 {
		return nil
	}
	number := binary.BigEndian.Uint64(data)
	return &number
}

// WriteStateHistoryIndexHead stores the number of latest indexed state history
// into database.
func WriteStateHistoryIndexHead(db ethdb.KeyValueWriter, number uint64) {
	if err := db.Put(stateHistoryIndexHeadKey, encodeBlockNumber(number)); err != nil {
		log.Crit("Failed to store the state index tail", "err", err)
	}
}

// DeleteStateHistoryIndexHead removes the number of latest indexed state history.
func DeleteStateHistoryIndexHead(db ethdb.KeyValueWriter) {
	if err := db.Delete(stateHistoryIndexHeadKey); err != nil {
		log.Crit("Failed to delete the state index tail", "err", err)
	}
}

// ReadStateIndex retrieves the state index with the provided account address
// and state hash.
func ReadStateIndex(db ethdb.KeyValueReader, address common.Address, state common.Hash) []byte {
	data, err := db.Get(stateIndexKey(address, state))
	if err != nil || len(data) == 0 {
		return nil
	}
	return data
}

// WriteStateIndex writes the provided state index into database.
func WriteStateIndex(db ethdb.KeyValueWriter, address common.Address, state common.Hash, data []byte) {
	if err := db.Put(stateIndexKey(address, state), data); err != nil {
		log.Crit("Failed to store state index", "err", err)
	}
}

// DeleteStateIndex deletes the specified state index from the database.
func DeleteStateIndex(db ethdb.KeyValueWriter, address common.Address, state common.Hash) {
	if err := db.Delete(stateIndexKey(address, state)); err != nil {
		log.Crit("Failed to delete state index", "err", err)
	}
}

// ReadStateIndexBlock retrieves the state index block with the provided state
// identifier along with the block id.
func ReadStateIndexBlock(db ethdb.KeyValueReader, address common.Address, state common.Hash, id uint32) []byte {
	data, err := db.Get(stateIndexBlockKey(address, state, id))
	if err != nil || len(data) == 0 {
		return nil
	}
	return data
}

// WriteStateIndexBlock writes the provided state index block into database.
func WriteStateIndexBlock(db ethdb.KeyValueWriter, address common.Address, state common.Hash, id uint32, data []byte) {
	if err := db.Put(stateIndexBlockKey(address, state, id), data); err != nil {
		log.Crit("Failed to store state index", "err", err)
	}
}

// DeleteStateIndexBlock deletes the specified state index block from the database.
func DeleteStateIndexBlock(db ethdb.KeyValueWriter, address common.Address, state common.Hash, id uint32) {
	if err := db.Delete(stateIndexBlockKey(address, state, id)); err != nil {
		log.Crit("Failed to delete state index", "err", err)
	}
}
