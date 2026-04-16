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

package state

import "github.com/ethereum/go-ethereum/common"

type mutationType int

const (
	update mutationType = iota
	deletion
)

type mutation struct {
	typ     mutationType
	applied bool

	// precedingDelete indicates that a previously unapplied deletion was
	// overwritten by an update (account deleted then re-created within
	// the same block). IntermediateRoot uses this to notify the hasher
	// of the deletion before the update so that any cached storage trie
	// is evicted and the re-created account starts with a fresh trie.
	precedingDelete bool
}

func (m *mutation) copy() *mutation {
	return &mutation{
		typ:             m.typ,
		applied:         m.applied,
		precedingDelete: m.precedingDelete,
	}
}

func (m *mutation) isDelete() bool {
	return m.typ == deletion
}

// markDelete is invoked when an account is deleted but the deletion is
// not yet committed. The pending mutation is cached and will be applied
// all together.
func (s *StateDB) markDelete(addr common.Address) {
	if _, ok := s.mutations[addr]; !ok {
		s.mutations[addr] = &mutation{}
	}
	s.mutations[addr].applied = false
	s.mutations[addr].typ = deletion
	s.mutations[addr].precedingDelete = false
}

func (s *StateDB) markUpdate(addr common.Address) {
	m, ok := s.mutations[addr]
	if !ok {
		s.mutations[addr] = &mutation{}
		m = s.mutations[addr]
	}
	// If this update overwrites a pending (unapplied) deletion, record it
	// so that IntermediateRoot can notify the hasher of the deletion first.
	// Do not reset precedingDelete otherwise: a subsequent markUpdate must
	// preserve the flag set by an earlier markDelete→markUpdate sequence.
	if !m.applied && m.typ == deletion {
		m.precedingDelete = true
	}
	m.applied = false
	m.typ = update
}
