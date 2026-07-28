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

package bintrie

import "errors"

var (
	// errInvalidSerializedLength is returned when a node blob is shorter than
	// its declared structure requires.
	errInvalidSerializedLength = errors.New("bintrie: invalid serialized node length")

	// errNonCanonicalPrefix is returned when a branch prefix carries non-zero
	// padding bits; accepting it would let two different trees share a root.
	errNonCanonicalPrefix = errors.New("bintrie: non-canonical prefix padding")

	// errInvalidNodeTag is returned when a node blob starts with an unknown
	// type tag.
	errInvalidNodeTag = errors.New("bintrie: invalid node tag")

	// ErrNonConformantKey is returned when a key does not match its zone's
	// required length (or names an unassigned zone). The engine deliberately
	// restricts the EIP-8297 key space to the embedding's fixed per-zone
	// lengths, which is what keeps stems non-nested.
	ErrNonConformantKey = errors.New("bintrie: key does not conform to zone layout")
)
