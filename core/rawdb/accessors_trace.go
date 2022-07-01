// Copyright 2021 The go-ethereum Authors
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

// ReadTxTrace retrieves the result of tx by evm-tracing which stores in db.
func ReadTxTrace(db ethdb.KeyValueReader, hash common.Hash) []byte {
	data, err := db.Get(codeKey(hash))
	if err != nil {
		log.Error("Failed to read tx trace result", "err", err)
	}
	return data
}

// WriteTxTrace write the result of tx tracing by evm-tracing to db.
func WriteTxTrace(db ethdb.KeyValueWriter, hash common.Hash, data []byte) error {
	if err := db.Put(codeKey(hash), data); err != nil {
		log.Crit("Failed to write tx trace result", "err", err)
		return err
	}
	return nil
}
