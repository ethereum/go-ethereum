// Copyright 2020 The go-ethereum Authors
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

package lespay

import (
	"errors"

	"github.com/ethereum/go-ethereum/rlp"
)

var ErrFieldNotFound = errors.New("meta info field not found")

type (
	// MetaInfo contains encoded meta channel values belonging to a single message
	MetaInfo []rlp.RawValue

	// MetaMapping is a static list of meta channel fields the peer can send/understand
	MetaMapping struct {
		Send, Receive []string
	}

	// MatchedMapping is a mapped list of fields understood by both peers (fields appear
	// in the same order in the MetaInfo encoding)
	MatchedMapping struct {
		Send, Receive *MatchedHalfMapping
	}

	// MatchedHalfMapping is a mapped list of sent or received fields
	MatchedHalfMapping struct {
		list                             []string
		staticToMatched, matchedToStatic []int // index mapping
	}
)

func match(send, receive []string, remoteMatch bool) *MatchedHalfMapping {
	m := &MatchedHalfMapping{}
	rm := make(map[string]int)
	for i, s := range receive {
		rm[s] = i
	}
	if remoteMatch {
		m.staticToMatched = make([]int, len(receive))
	} else {
		m.staticToMatched = make([]int, len(send))
	}
	for i := range m.staticToMatched {
		m.staticToMatched[i] = -1
	}
	for i, s := range send {
		if j, ok := rm[s]; ok {
			m.list = append(m.list, s)
			var mi int
			if remoteMatch {
				mi = j
			} else {
				mi = i
			}
			m.staticToMatched[mi] = len(m.matchedToStatic)
			m.matchedToStatic = append(m.matchedToStatic, mi)
		}
	}
	return m
}

// Match creates a matched mapping out of the local and remote static mapping. The matched
// list contains mutually supported fields in the same order as they appear on the sender side.
func (local MetaMapping) Match(remote MetaMapping) MatchedMapping {
	return MatchedMapping{Send: match(local.Send, remote.Receive, false), Receive: match(remote.Send, local.Receive, true)}
}

// Has returns true if the mapping includes the field specified by the static index
func (m *MatchedHalfMapping) Has(staticIndex int) bool {
	return m != nil && len(m.staticToMatched) > staticIndex && m.staticToMatched[staticIndex] != -1
}

// Get decodes the given field of the encoded meta info if present
func (m *MetaInfo) Get(mapping *MatchedHalfMapping, staticIndex int, value interface{}) error {
	if mapping == nil || len(mapping.staticToMatched) <= staticIndex {
		return ErrFieldNotFound
	}
	if i := mapping.staticToMatched[staticIndex]; i == -1 || i >= len(*m) || len((*m)[i]) == 0 {
		return ErrFieldNotFound
	} else {

		return rlp.DecodeBytes((*m)[i], value)
	}
}

// Set encodes and sets the given field of the meta info if present
func (m *MetaInfo) Set(mapping *MatchedHalfMapping, staticIndex int, value interface{}) error {
	if mapping == nil || len(mapping.staticToMatched) <= staticIndex {
		return ErrFieldNotFound
	}
	if i := mapping.staticToMatched[staticIndex]; i == -1 {
		return ErrFieldNotFound
	} else {
		enc, err := rlp.EncodeToBytes(value)
		if err == nil {
			for len(*m) <= i {
				*m = append(*m, nil)
			}
			(*m)[i] = enc
		}
		return err
	}
}
