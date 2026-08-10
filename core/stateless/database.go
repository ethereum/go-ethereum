// Copyright 2024 The go-ethereum Authors
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

package stateless

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
)

// MakeHashDB imports tries, codes and block hashes from a witness into a new
// hash-based memory db. We could eventually rewrite this into a pathdb, but
// simple is better for now.
//
// Note, this hashdb approach is quite strictly self-validating:
//   - Headers are persisted keyed by hash, so blockhash will error on junk
//   - Codes are persisted keyed by hash, so bytecode lookup will error on junk
//   - Trie nodes are persisted keyed by hash, so trie expansion will error on junk
//
// Acceleration structures built would need to explicitly validate the witness.
func (w *Witness) MakeHashDB() ethdb.Database {
	var (
		memdb  = rawdb.NewMemoryDatabase()
		hasher = crypto.NewKeccakState()
		hash   = make([]byte, 32)
	)
	// Inject all the "block hashes" (i.e. headers) into the ephemeral database
	for _, header := range w.Headers {
		rawdb.WriteHeader(memdb, header)
	}
	// Inject all the bytecodes into the ephemeral database
	for code := range w.Codes {
		blob := []byte(code)

		hasher.Reset()
		hasher.Write(blob)
		hasher.Read(hash)

		rawdb.WriteCode(memdb, common.BytesToHash(hash), blob)
	}
	// Inject all the MPT trie nodes into the ephemeral database
	for node := range w.State {
		blob := []byte(node)

		hasher.Reset()
		hasher.Write(blob)
		hasher.Read(hash)

		rawdb.WriteLegacyTrieNode(memdb, common.BytesToHash(hash), blob)
	}
	return memdb
}

// MakePathDB imports nodes, codes and block hashes from a binary-tree witness
// into a new path-based memory db.
//
// The binary tree cannot use MakeHashDB. A merkle node is named by the hash of
// its own bytes, so a bag of blobs is a complete description; a binary group
// record folds at a depth stored inside it, so its hash depends on where it
// sits and the blob alone cannot be re-keyed. Nodes are therefore addressed by
// path here, which is what pathdb wants anyway.
//
// It is self-validating on the same terms as MakeHashDB. Headers and codes are
// still keyed by hash. Nodes are checked on read rather than on write: pathdb
// verifies each one against the hash its parent points at, using the binary
// hasher that folds group records at their stored depth, so a blob planted at
// the wrong path or altered in place fails to resolve.
func (w *Witness) MakePathDB() ethdb.Database {
	var (
		memdb  = rawdb.NewMemoryDatabase()
		hasher = crypto.NewKeccakState()
		hash   = make([]byte, 32)
	)
	for _, header := range w.Headers {
		rawdb.WriteHeader(memdb, header)
	}
	// Code is still content-addressed by keccak in the binary tree; only the
	// state nodes change shape.
	for code := range w.Codes {
		blob := []byte(code)

		hasher.Reset()
		hasher.Write(blob)
		hasher.Read(hash)

		rawdb.WriteCode(memdb, common.BytesToHash(hash), blob)
	}
	// Binary-tree data lives under its own namespace, the same one pathdb
	// rebinds itself to when opened in binary mode.
	tbl := rawdb.NewTable(memdb, string(rawdb.PBTPrefix))
	for path, blob := range w.Nodes {
		rawdb.WriteAccountTrieNode(tbl, []byte(path), blob)
	}
	// pathdb resolves the disk layer's root from this marker when it opens in
	// binary mode; without it the layer would come up empty and every lookup
	// would miss.
	rawdb.WriteSnapshotRoot(tbl, w.Root())
	return memdb
}
