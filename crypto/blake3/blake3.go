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

// Package blake3 wraps the BLAKE3 hash for use as the EIP-8297 binary tree
// hash function. The tree spec keeps the hash choice open; isolating the
// implementation behind this package makes a future swap a one-file change.
package blake3

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/zeebo/blake3"
)

// Sum256 returns the 32-byte BLAKE3 digest of data.
//
// zeebo/blake3 was selected over lukechampine.com/blake3 on 2026-07-28
// benchmarks (20-35% faster on arm64 across the engine's 67-133B preimage
// sizes; both carry amd64 assembly). The test suite cross-checks both.
func Sum256(data []byte) [32]byte {
	return blake3.Sum256(data)
}

// HashData returns the BLAKE3 digest of data as a common.Hash.
func HashData(data []byte) common.Hash {
	return common.Hash(blake3.Sum256(data))
}
