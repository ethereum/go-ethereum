// Copyright 2021 The go-ethereum Authors
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

func TestJournal(t *testing.T) {
	//log.Root().SetHandler(log.LvlFilterHandler(log.LvlTrace, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))
	var (
		env       = fillDB()
		dl        = env.db.disklayer()
		diskIndex int
	)
	defer env.teardown()

	if err := env.db.Journal(env.roots[len(env.roots)-1]); err != nil {
		t.Error("Failed to journal triedb", "err", err)
	}
	newdb := NewDatabase(env.db.diskdb, env.db.config)

	for diskIndex = 0; diskIndex < len(env.roots); diskIndex++ {
		if env.roots[diskIndex] == dl.root {
			break
		}
	}
	for i := diskIndex; i < len(env.numbers); i++ {
		keys, vals := env.keys[i], env.vals[i]
		for j := 0; j < len(keys); j++ {
			if vals[j] == nil {
				continue
			}
			layer := newdb.Snapshot(env.roots[i])
			blob, err := layer.NodeBlob([]byte(keys[j]), crypto.Keccak256Hash(vals[j]))
			if err != nil {
				t.Error("Failed to retrieve state", "err", err)
			}
			if !bytes.Equal(blob, vals[j]) {
				t.Error("Unexpected state", "key", []byte(keys[j]), "want", vals[j], "got", blob)
			}
		}
	}
}
