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
// start with qkcPrefix, keeping them apart from the geth ones.
var (
	qkcPrefix = []byte("qkc_")

	// rootHeadKey tracks the latest known root block, the root tip.
	rootHeadKey = qkcKey([]byte("qkc_LastRoot"))

	rootBlockPrefix     = []byte("rb")  // qkcPrefix + rootBlockPrefix + hash -> root block
	lastConfirmedPrefix = []byte("rLM") // qkcPrefix + lastConfirmedPrefix + root hash -> last confirmed shard block hash
)

// qkcKey prepends qkcPrefix to the concatenation of the given key parts.
func qkcKey(parts ...[]byte) []byte {
	key := common.CopyBytes(qkcPrefix)
	for _, part := range parts {
		key = append(key, part...)
	}
	return key
}

// rootBlockKey = qkcPrefix + rootBlockPrefix + hash
func rootBlockKey(hash common.Hash) []byte {
	return qkcKey(rootBlockPrefix, hash.Bytes())
}

// lastConfirmedKey = qkcPrefix + lastConfirmedPrefix + root hash
func lastConfirmedKey(rootHash common.Hash) []byte {
	return qkcKey(lastConfirmedPrefix, rootHash.Bytes())
}
