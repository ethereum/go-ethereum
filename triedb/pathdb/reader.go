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
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>

package pathdb

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
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

// reader implements the database.NodeReader interface, providing the functionalities to
// retrieve trie nodes by wrapping the internal state layer.
type reader struct {
	layer       layer
	noHashCheck bool
}

// Node implements database.NodeReader interface, retrieving the node with specified
// node info. Don't modify the returned byte slice since it's not deep-copied
// and still be referenced by database.
func (r *reader) Node(owner common.Hash, path []byte, hash common.Hash) ([]byte, error) {
	blob, got, loc, err := r.layer.node(owner, path, 0)
	if err != nil {
		return nil, err
	}
	// Error out if the local one is inconsistent with the target.
	if !r.noHashCheck && got != hash {
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
		blobHex := "nil"
		if len(blob) > 0 {
			blobHex = hexutil.Encode(blob)
		}
		log.Error("Unexpected trie node", "location", loc.loc, "owner", owner.Hex(), "path", path, "expect", hash.Hex(), "got", got.Hex(), "blob", blobHex)
		return nil, fmt.Errorf("unexpected node: (%x %v), %x!=%x, %s, blob: %s", owner, path, hash, got, loc.string(), blobHex)
	}
	return blob, nil
}

// NodeReader retrieves a layer belonging to the given state root.
func (db *Database) NodeReader(root common.Hash) (database.NodeReader, error) {
	layer := db.tree.get(root)
	if layer == nil {
		return nil, fmt.Errorf("state %#x is not available", root)
	}
	return &reader{layer: layer, noHashCheck: db.isVerkle}, nil
}
