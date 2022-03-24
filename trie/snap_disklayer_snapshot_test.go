// Copyright 2022 The go-ethereum Authors
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

package trie

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

func TestGetSnapshot(t *testing.T) {
	defer func(origin uint64) {
		cacheSizeLimit = origin
	}(cacheSizeLimit)
	cacheSizeLimit = 1024 * 256 // Lower the dirty cache size

	var (
		index  int
		env    = fillDB(t)
		snapdb = env.db.backend.(*snapDatabase)
		dl     = snapdb.tree.bottom().(*diskLayer)
	)
	for index = 0; index < len(env.roots); index++ {
		if env.roots[index] == dl.root {
			break
		}
	}
	for i := 0; i < index; i++ {
		layer, err := dl.GetSnapshot(env.roots[i], snapdb.freezer)
		if err != nil {
			t.Fatalf("Failed to retrieve snapshot %v", err)
		}
		defer layer.Release()

		keys, vals := env.keys[i], env.vals[i]
		for j, key := range keys {
			if len(vals[j]) == 0 {
				// deleted node, expect error
				blob, _ := layer.NodeBlob([]byte(key), crypto.Keccak256Hash(vals[j])) // error can occur
				if len(blob) != 0 {
					t.Error("Unexpected state", "key", []byte(key), "got", blob)
				}
			} else {
				// normal node, expect correct value
				blob, err := layer.NodeBlob([]byte(key), crypto.Keccak256Hash(vals[j]))
				if err != nil {
					t.Error("Failed to retrieve state", "err", err)
				}
				if !bytes.Equal(blob, vals[j]) {
					t.Error("Unexpected state", "key", []byte(key), "want", vals[j], "got", blob)
				}
			}
		}
	}
}
