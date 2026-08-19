// Copyright 2026 The go-ethereum Authors
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

// ReadShadowStateRoot returns the recorded shadow root of the given block -
// the root of the tree its header does not commit - or zero if the block was
// never replayed.
func ReadShadowStateRoot(db ethdb.KeyValueReader, number uint64, hash common.Hash) common.Hash {
	data, _ := db.Get(shadowRootKey(number, hash))
	if len(data) != common.HashLength {
		return common.Hash{}
	}
	return common.BytesToHash(data)
}

// WriteShadowStateRoot records the shadow root of the given block.
func WriteShadowStateRoot(db ethdb.KeyValueWriter, number uint64, hash common.Hash, root common.Hash) {
	if err := db.Put(shadowRootKey(number, hash), root.Bytes()); err != nil {
		log.Crit("Failed to store shadow state root", "err", err)
	}
}

// DeleteShadowStateRoot removes the recorded shadow root of the given block.
func DeleteShadowStateRoot(db ethdb.KeyValueWriter, number uint64, hash common.Hash) {
	if err := db.Delete(shadowRootKey(number, hash)); err != nil {
		log.Crit("Failed to delete shadow state root", "err", err)
	}
}

// ReadPBTMigrationCursor returns the shadow follower's last replayed block and
// shadow root, or ok=false if no cursor was written. The cursor is a resume
// hint; the shadow database itself is the truth.
func ReadPBTMigrationCursor(db ethdb.KeyValueReader) (uint64, common.Hash, common.Hash, bool) {
	data, _ := db.Get(pbtMigrationCursorKey)
	if len(data) != 8+2*common.HashLength {
		return 0, common.Hash{}, common.Hash{}, false
	}
	number := binary.BigEndian.Uint64(data)
	hash := common.BytesToHash(data[8 : 8+common.HashLength])
	root := common.BytesToHash(data[8+common.HashLength:])
	return number, hash, root, true
}

// WritePBTMigrationCursor records the shadow follower's replay position.
func WritePBTMigrationCursor(db ethdb.KeyValueWriter, number uint64, hash common.Hash, root common.Hash) {
	data := append(append(encodeBlockNumber(number), hash.Bytes()...), root.Bytes()...)
	if err := db.Put(pbtMigrationCursorKey, data); err != nil {
		log.Crit("Failed to store migration cursor", "err", err)
	}
}

// ReadPBTMigrationDone reports whether the migration finished: the fork
// finalized and the retired tree stopped being maintained.
func ReadPBTMigrationDone(db ethdb.KeyValueReader) bool {
	data, _ := db.Get(pbtMigrationDoneKey)
	return len(data) == 1 && data[0] == 1
}

// WritePBTMigrationDone marks the migration finished.
func WritePBTMigrationDone(db ethdb.KeyValueWriter) {
	if err := db.Put(pbtMigrationDoneKey, []byte{1}); err != nil {
		log.Crit("Failed to store migration done marker", "err", err)
	}
}
