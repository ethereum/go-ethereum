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

// Package state provides a caching layer atop the Ethereum state trie.
package state

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state/snapshot"
)

func (s *StateDB) Snaps() *snapshot.Tree {
	return s.snaps
}

func (s *StateDB) Witness() *AccessWitness {
	if s.witness == nil {
		s.witness = NewAccessWitness(s)
	}
	return s.witness
}

func (s *StateDB) SetWitness(aw *AccessWitness) {
	s.witness = aw
}

// GetTrie returns the account trie.
func (s *StateDB) GetTrie() Trie {
	return s.trie
}

// XXX check this is still necessary
func (s *StateDB) Cap(root common.Hash) error {
	if s.snaps != nil {
		return s.snaps.Cap(root, 0)
	}
	// pre-verkle path: noop if s.snaps hasn't been
	// initialized.
	return nil
}
