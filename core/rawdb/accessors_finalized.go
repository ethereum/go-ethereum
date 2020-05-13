// Copyright 2020 The go-ethereum Authors
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

// ReadFinalizedBlockHash retrieves the hash of the last finalized block.
func ReadFinalizedBlockHash(db ethdb.KeyValueReader) common.Hash {
	data, _ := db.Get(finalizedBlockKey)
	if len(data) != common.HashLength {
		return common.Hash{}
	}
	return common.BytesToHash(data)
}

// WriteFinalizedBlockHash stores the hash of the last finalized block.
func WriteFinalizedBlockHash(db ethdb.KeyValueWriter, h common.Hash) error {
	if err := db.Put(finalizedBlockKey, h[:]); err != nil {
		log.Crit("Failed to store the finalized block hash", "err", err)
		return err
	}
	return nil
}
