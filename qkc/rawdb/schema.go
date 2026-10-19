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

// Package rawdb contains the low level database accessors of the QuarkChain
// data kept next to a shard chain, such as the root blocks.
package rawdb

import "github.com/ethereum/go-ethereum/common"

// The fields below define the QuarkChain specific database schema. All keys
// start with qkc_, keeping them apart from the geth ones.
var (
	// rootHeadKey tracks the latest known root block, the root tip.
	rootHeadKey = []byte("qkc_LastRoot")

	rootBlockPrefix      = []byte("qkc_rb")     // rootBlockPrefix + hash -> root block
	lastConfirmedPrefix  = []byte("qkc_rLM")    // lastConfirmedPrefix + root hash -> last confirmed shard block hash
	xShardDepositsPrefix = []byte("qkc_xShard") // xShardDepositsPrefix + source block hash -> deposits sent to this shard by the block
)

// rootBlockKey = rootBlockPrefix + hash
func rootBlockKey(hash common.Hash) []byte {
	return append(common.CopyBytes(rootBlockPrefix), hash.Bytes()...)
}

// lastConfirmedKey = lastConfirmedPrefix + root hash
func lastConfirmedKey(rootHash common.Hash) []byte {
	return append(common.CopyBytes(lastConfirmedPrefix), rootHash.Bytes()...)
}

// xShardDepositsKey = xShardDepositsPrefix + source block hash
func xShardDepositsKey(hash common.Hash) []byte {
	return append(common.CopyBytes(xShardDepositsPrefix), hash.Bytes()...)
}
