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

package types

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/codehash"
	"github.com/ethereum/go-ethereum/log"
)

var (
	// EmptyZkTrieRootHash is the known root hash of an empty zktrie.
	EmptyZkTrieRootHash = common.Hash{}

	// EmptyLegacyTrieRootHash is the known root hash of an empty legacy trie.
	EmptyLegacyTrieRootHash = common.HexToHash("56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")

	// EmptyRootHash is the known root hash of an empty trie.
	EmptyRootHash = EmptyZkTrieRootHash

	// EmptyUncleHash is the known hash of the empty uncle set.
	EmptyUncleHash = rlpHash([]*Header(nil)) // 1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347

	// // EmptyCodeHash is the known hash of the empty EVM bytecode.
	// EmptyCodeHash = crypto.Keccak256Hash(nil) // c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470

	// EmptyKeccakCodeHash is the known Keccak hash of the empty EVM bytecode.
	EmptyKeccakCodeHash = codehash.EmptyKeccakCodeHash

	// EmptyKeccakCodeHash is the known Poseidon hash of the empty EVM bytecode.
	EmptyPoseidonCodeHash = codehash.EmptyPoseidonCodeHash

	// EmptyTxsHash is the known hash of the empty transaction set.
	EmptyTxsHash = EmptyLegacyTrieRootHash

	// EmptyReceiptsHash is the known hash of the empty receipt set.
	EmptyReceiptsHash = EmptyLegacyTrieRootHash

	// EmptyWithdrawalsHash is the known hash of the empty withdrawal set.
	EmptyWithdrawalsHash = EmptyLegacyTrieRootHash
)

// TrieRootHash returns the hash itself if it's non-empty or the predefined
// emptyHash one instead.
func TrieRootHash(hash common.Hash) common.Hash {
	if hash == (common.Hash{}) {
		log.Error("Zero trie root hash!")
		return EmptyRootHash
	}
	return hash
}
