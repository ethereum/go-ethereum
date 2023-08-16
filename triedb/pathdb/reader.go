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
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>

package pathdb

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/triedb/database"
)

// The types of locations where the node is found.
const (
	locDirtyCache = "dirty" // dirty cache
	locCleanCache = "clean" // clean cache
	locDiskLayer  = "disk"  // persistent state
	locDiffLayer  = "diff"  // diff layers
)

// nodeLoc is a helpful structure that contains the location where the node
// is found, as it's useful for debugging purposes.
type nodeLoc struct {
	loc   string
	depth int
}

// string returns the string representation of node location.
func (loc *nodeLoc) string() string {
	return fmt.Sprintf("loc: %s, depth: %d", loc.loc, loc.depth)
}

// reader implements the Reader interface, providing the functionalities to
// retrieve trie nodes by wrapping the internal state layer.
type reader struct {
	layer layer
	state crypto.KeccakState
}

// Node implements trie.Reader interface, retrieving the node with specified
// node info. Don't modify the returned byte slice since it's not deep-copied
// and still be referenced by database.
func (r *reader) Node(owner common.Hash, path []byte, hash common.Hash) ([]byte, error) {
	blob, loc, err := r.layer.node(owner, path, 0)
	if err != nil {
		return nil, err
	}
	if got := crypto.HashData(r.state, blob); got != hash {
		// Location is always available even if the node
		// is not found.
		switch loc.loc {
		case locCleanCache:
			cleanFalseMeter.Mark(1)
		case locDirtyCache:
			dirtyFalseMeter.Mark(1)
		case locDiffLayer:
			diffFalseMeter.Mark(1)
		case locDiskLayer:
			diskFalseMeter.Mark(1)
		}
		log.Error("Unexpected trie node", "location", loc.loc, "owner", owner, "path", path, "expect", hash, "got", got)
		return nil, fmt.Errorf("unexpected node: (%x %v), %x!=%x, %s", owner, path, hash, got, loc.string())
	}
	return blob, nil
}

// Reader retrieves a layer belonging to the given state root.
func (db *Database) Reader(root common.Hash) (database.Reader, error) {
	layer := db.tree.get(root)
	if layer == nil {
		return nil, fmt.Errorf("state %#x is not available", root)
	}
	return &reader{layer: layer, state: crypto.NewKeccakState()}, nil
}
