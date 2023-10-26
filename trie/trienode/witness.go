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
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/

package trienode

import (
	"encoding/hex"
	"encoding/json"

	"github.com/ethereum/go-ethereum/common"
)

// Witness is the set of nodes retrieved from the database for executing a state
// transition. It can be considered as the previous state for the state transition,
// effectively serving as the witness for that transition.
type Witness struct {
	Owner common.Hash
	Nodes map[string][]byte
}

func (u *Witness) MarshalJSON() ([]byte, error) {
	type Node struct {
		Key   string
		Value string
	}
	nodes := []Node{}
	for key, node := range u.Nodes {
		nodes = append(nodes, Node{
			Key:   hex.EncodeToString([]byte(key)),
			Value: hex.EncodeToString(node),
		})
	}
	return json.Marshal(&struct {
		Owner common.Hash
		Nodes []Node
	}{
		Owner: u.Owner,
		Nodes: nodes,
	})
}

// NewWitness constructs a witness structure.
func NewWitness(owner common.Hash) *Witness {
	return &Witness{Owner: owner, Nodes: make(map[string][]byte)}
}

// Add tracks the node resolved from database. Don't change the blob outside of
// function since it's not deep-copied.
func (w *Witness) Add(path string, blob []byte) {
	w.Nodes[path] = common.CopyBytes(blob)
}

// Has returns the indicator whether the specified node is in witness.
func (w *Witness) Has(path string) bool {
	_, ok := w.Nodes[path]
	return ok
}

// Len returns the number of nodes resolved in the witness.
func (w *Witness) Len() int {
	return len(w.Nodes)
}

// Copy returns a deep copied witness structure.
func (w *Witness) Copy() *Witness {
	cpy := &Witness{
		Owner: w.Owner,
		Nodes: make(map[string][]byte),
	}
	for p, n := range w.Nodes {
		cpy.Nodes[p] = n // it's not deep-copied
	}
	return cpy
}

// Witnesses represents a set of witness for a group of tries.
type Witnesses struct {
	witness map[common.Hash]*Witness
}

// NewWitnesses initializes an empty witness set.
func NewWitnesses() *Witnesses {
	return &Witnesses{witness: make(map[common.Hash]*Witness)}
}

// Merge merges the provided dirty nodes of a trie into the set. The assumption
// is held that no duplicated set belonging to the same trie will be merged twice.
func (set *Witnesses) Merge(other *Witness) error {
	_, present := set.witness[other.Owner]
	if present {
		return nil
		//return subset.Merge(other.Owner, other.Nodes)
	}
	set.witness[other.Owner] = other
	return nil
}

func (set *Witnesses) Witnesses() map[common.Hash]*Witness {
	return set.witness
}
