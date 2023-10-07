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

import "github.com/ethereum/go-ethereum/common"

// Witness is the set of nodes retrieved from the database for executing a state
// transition. It can be considered as the previous state for the state transition,
// effectively serving as the witness for that transition.
type Witness struct {
	Owner common.Hash
	Nodes map[string][]byte
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
