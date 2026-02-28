// Copyright 2023 The go-ethereum Authors
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

package triedb

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/triedb/pathdb"
)

// AccountHistory inspects the account history within the specified range.
//
// Start: State ID of the first history object for the query. 0 implies the first
// available object is selected as the starting point.
//
// End: State ID of the last history for the query. 0 implies the last available
// object is selected as the starting point. Note end is included for query.
//
// This function is only supported by path mode database.
func (db *Database) AccountHistory(address common.Address, start, end uint64) (*pathdb.HistoryStats, error) {
	pdb, ok := db.backend.(*pathdb.Database)
	if !ok {
		return nil, errors.New("not supported")
	}
	return pdb.AccountHistory(address, start, end)
}

// StorageHistory inspects the storage history within the specified range.
//
// Start: State ID of the first history object for the query. 0 implies the first
// available object is selected as the starting point.
//
// End: State ID of the last history for the query. 0 implies the last available
// object is selected as the starting point. Note end is included for query.
//
// Note, slot refers to the hash of the raw slot key.
//
// This function is only supported by path mode database.
func (db *Database) StorageHistory(address common.Address, slot common.Hash, start uint64, end uint64) (*pathdb.HistoryStats, error) {
	pdb, ok := db.backend.(*pathdb.Database)
	if !ok {
		return nil, errors.New("not supported")
	}
	return pdb.StorageHistory(address, slot, start, end)
}

// HistoryRange returns the block numbers associated with earliest and latest
// state history in the local store.
//
// This function is only supported by path mode database.
func (db *Database) HistoryRange() (uint64, uint64, error) {
	pdb, ok := db.backend.(*pathdb.Database)
	if !ok {
		return 0, 0, errors.New("not supported")
	}
	return pdb.HistoryRange()
}
