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

package rawdb

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

// ReadTxTrace retrieves the transaction trace for the given hash from the database.
func ReadTxTrace(db ethdb.KeyValueReader, hash common.Hash) ([]byte, error) {
	return db.Get(txTraceKey(hash))
}

// WriteTxTrace stores the transaction trace for the given hash to the database.
func WriteTxTrace(db ethdb.KeyValueWriter, hash common.Hash, trace []byte) {
	if err := db.Put(txTraceKey(hash), trace); err != nil {
		log.Crit("Failed to store transaction trace", "err", err)
	}
}

// DeleteTxTrace deletes the transaction trace for the given hash from the database.
func DeleteTxTrace(db ethdb.KeyValueWriter, hash common.Hash) {
	if err := db.Delete(txTraceKey(hash)); err != nil {
		log.Crit("Failed to delete transaction trace", "err", err)
	}
}
