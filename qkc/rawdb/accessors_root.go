// Copyright 2026 The quarkchain Authors
// This file is part of the quarkchain library.
//
// The quarkchain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The quarkchain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the quarkchain library. If not, see <http://www.gnu.org/licenses/>.

package rawdb

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/qkc/types"
)

// ReadRootHeadHash retrieves the hash of the root tip.
func ReadRootHeadHash(db ethdb.KeyValueReader) common.Hash {
	data, _ := db.Get(rootHeadKey)
	if len(data) == 0 {
		return common.Hash{}
	}
	return common.BytesToHash(data)
}

// WriteRootHeadHash stores the hash of the root tip.
func WriteRootHeadHash(db ethdb.KeyValueWriter, hash common.Hash) {
	if err := db.Put(rootHeadKey, hash.Bytes()); err != nil {
		log.Crit("Failed to store root head hash", "err", err)
	}
}

// HasRootBlock verifies the existence of a root block corresponding to the hash.
func HasRootBlock(db ethdb.KeyValueReader, hash common.Hash) bool {
	has, err := db.Has(rootBlockKey(hash))
	return has && err == nil
}

// ReadRootBlockSSZ retrieves a root block in its raw SSZ database encoding.
func ReadRootBlockSSZ(db ethdb.KeyValueReader, hash common.Hash) []byte {
	data, _ := db.Get(rootBlockKey(hash))
	return data
}

// ReadRootBlock retrieves the root block corresponding to the hash.
func ReadRootBlock(db ethdb.KeyValueReader, hash common.Hash) *types.RootBlock {
	data := ReadRootBlockSSZ(db, hash)
	if len(data) == 0 {
		return nil
	}
	block := new(types.RootBlock)
	if err := block.UnmarshalSSZ(data); err != nil {
		log.Error("Invalid root block SSZ", "hash", hash, "err", err)
		return nil
	}
	return block
}

// ReadRootBlockHeader retrieves the header of the root block corresponding to
// the hash, without decoding the shard block headers it confirms.
func ReadRootBlockHeader(db ethdb.KeyValueReader, hash common.Hash) *types.RootBlockHeader {
	data := ReadRootBlockSSZ(db, hash)
	if len(data) == 0 {
		return nil
	}
	header, err := types.DecodeRootBlockHeaderFromBlockSSZ(data)
	if err != nil {
		log.Error("Invalid root block header SSZ", "hash", hash, "err", err)
		return nil
	}
	return header
}

// WriteRootBlock stores a root block into the database.
func WriteRootBlock(db ethdb.KeyValueWriter, block *types.RootBlock) {
	data, err := block.MarshalSSZ()
	if err != nil {
		log.Crit("Failed to SSZ encode root block", "err", err)
	}
	if err := db.Put(rootBlockKey(block.Hash()), data); err != nil {
		log.Crit("Failed to store root block", "err", err)
	}
}

// ReadLastConfirmedBlockHash retrieves the hash of the latest shard block
// confirmed by the root chain up to the given root block. It is the zero hash if
// the root chain confirms no block of the shard yet.
func ReadLastConfirmedBlockHash(db ethdb.KeyValueReader, rootHash common.Hash) common.Hash {
	data, _ := db.Get(lastConfirmedKey(rootHash))
	if len(data) == 0 {
		return common.Hash{}
	}
	return common.BytesToHash(data)
}

// WriteLastConfirmedBlockHash stores the hash of the latest shard block
// confirmed by the root chain up to the given root block.
func WriteLastConfirmedBlockHash(db ethdb.KeyValueWriter, rootHash common.Hash, hash common.Hash) {
	if err := db.Put(lastConfirmedKey(rootHash), hash.Bytes()); err != nil {
		log.Crit("Failed to store last confirmed shard block hash", "err", err)
	}
}
