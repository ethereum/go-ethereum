// Copyright 2026-2027 The QuarkChain Authors
// This file is part of the QuarkChain library.
//
// The QuarkChain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The QuarkChain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the QuarkChain library. If not, see <http://www.gnu.org/licenses/>.

package rawdb

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

// xShardDepositsPrefix + source block hash -> deposits sent to this shard by the block
var xShardDepositsPrefix = []byte("qkc_xShard")

// xShardDepositsKey = xShardDepositsPrefix + source block hash
func xShardDepositsKey(hash common.Hash) []byte {
	return append(common.CopyBytes(xShardDepositsPrefix), hash.Bytes()...)
}

// HasXShardDeposits reports whether the deposits sent to this shard by the block
// of another shard with the given hash are stored.
func HasXShardDeposits(db ethdb.KeyValueReader, hash common.Hash) bool {
	has, _ := db.Has(xShardDepositsKey(hash))
	return has
}

// ReadXShardDeposits retrieves the deposits sent to this shard by the block of
// another shard with the given hash. It returns nil if they are not stored, and
// an empty list if the block sent this shard nothing.
func ReadXShardDeposits(db ethdb.KeyValueReader, hash common.Hash) []*types.DepositTx {
	data, _ := db.Get(xShardDepositsKey(hash))
	if len(data) == 0 {
		return nil
	}
	var deposits []*types.DepositTx
	if err := rlp.DecodeBytes(data, &deposits); err != nil {
		log.Error("Invalid cross-shard deposits RLP", "hash", hash, "err", err)
		return nil
	}
	return deposits
}

// WriteXShardDeposits stores the deposits sent to this shard by the block of
// another shard with the given hash, as the RLP encoding of the list.
func WriteXShardDeposits(db ethdb.KeyValueWriter, hash common.Hash, deposits []*types.DepositTx) {
	data, err := rlp.EncodeToBytes(deposits)
	if err != nil {
		log.Crit("Failed to RLP encode cross-shard deposits", "err", err)
	}
	if err := db.Put(xShardDepositsKey(hash), data); err != nil {
		log.Crit("Failed to store cross-shard deposits", "err", err)
	}
}
