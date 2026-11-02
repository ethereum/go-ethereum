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

package ssz

import (
	"errors"
	"fmt"
)

// ActiveFields is the active_fields configuration of a ProgressiveContainer
// type (EIP-7495): a bitvector of at most 256 positions in which a set bit
// marks a field and a cleared bit a gap, left by a field removed in, or
// reserved for, another version of the type. The gaps are what keep every
// field's generalized index stable across forks.
//
// The configuration belongs to the type, not the value. It never appears in
// the serialization, which is that of a Container, and enters only the hash
// tree root, as zero chunks at the gaps and as the mix-in word. Declare one
// per type at package level with MustActiveFields, next to the type it
// describes, spelled as the EIP spells it:
//
//	var afSquare = ssz.MustActiveFields(1, 0, 1) // ProgressiveContainer(active_fields=[1, 0, 1])
type ActiveFields struct {
	bits    [32]byte // pack_bits(active_fields), LSB-first: the mix-in word
	width   int      // number of positions, gaps included
	nfields int      // number of set bits: the number of fields
}

// NewActiveFields validates an active_fields configuration and returns its
// descriptor. The mask is the EIP's literal, one entry per position, each 0
// or 1. Three of the EIP's four illegal configurations are rejected here: no
// positions, more than 256 positions, and a trailing 0, which would be a real
// zero leaf that changes the root rather than a no-op. The fourth, a count of
// set bits different from the number of fields, is checked by
// HashTreeRootProgressiveContainer, the one place a type lists its fields.
func NewActiveFields(mask ...int) (ActiveFields, error) {
	var af ActiveFields
	if len(mask) == 0 {
		return af, errors.New("ssz: active_fields with no positions is illegal")
	}
	if len(mask) > 256 {
		return af, fmt.Errorf("ssz: active_fields with %d positions exceeds 256", len(mask))
	}
	if mask[len(mask)-1] != 1 {
		return af, errors.New("ssz: active_fields ending in 0 is illegal")
	}
	for i, b := range mask {
		switch b {
		case 0:
		case 1:
			af.bits[i/8] |= 1 << (i % 8)
			af.nfields++
		default:
			return af, fmt.Errorf("ssz: active_fields entry %d is %d, not 0 or 1", i, b)
		}
	}
	af.width = len(mask)
	return af, nil
}

// MustActiveFields is NewActiveFields for package-level type descriptors: an
// illegal configuration is a programming error, not input, so it panics.
func MustActiveFields(mask ...int) ActiveFields {
	af, err := NewActiveFields(mask...)
	if err != nil {
		panic(err)
	}
	return af
}

// Width returns the number of positions in the configuration, gaps included,
// which is the number of chunks the field tree is built over.
func (af ActiveFields) Width() int { return af.width }

// NumFields returns the number of set bits, the number of fields the type
// declares.
func (af ActiveFields) NumFields() int { return af.nfields }

// Active reports whether position i holds a field. Positions outside the
// configuration are gaps.
func (af ActiveFields) Active(i int) bool {
	if i < 0 || i >= af.width {
		return false
	}
	return af.bits[i/8]&(1<<(i%8)) != 0
}

// MixInActiveFields returns hash(root ‖ pack_bits(active_fields)), the
// EIP-7495 mix-in that tells a field that is absent from a field that is
// zero; without it two versions of a type could share a root. The word is
// the configuration packed LSB-first into one chunk, bit i of the
// configuration being bit i%8 of byte i/8. The root is the left input of the
// hash, generalized index 2, and the word the right, 3.
func MixInActiveFields(root [32]byte, af ActiveFields) [32]byte {
	return hashPair(root, af.bits)
}

// HashTreeRootProgressiveContainer computes the root of a ProgressiveContainer
// from the roots of its fields, given in declaration order:
//
//	mix_in_active_fields(merkleize_progressive(field_roots), active_fields)
//
// where field_roots holds the root of a field at each set position of the
// configuration and a zero chunk at each cleared one, so the chunk list is as
// wide as the configuration, not as long as the field list. EIPs PR 12419
// spells this out; compacting the roots over the gaps gives a different tree.
//
// The number of roots must equal the number of set bits. A mismatch is a bug
// in the type, either in its configuration or in a HashTreeRoot that lists
// one field too few or too many, so it panics, as Merkleize does for a limit
// below its chunk count.
func HashTreeRootProgressiveContainer(af ActiveFields, roots [][32]byte) [32]byte {
	return MixInActiveFields(MerkleizeProgressive(layoutChunks(af, roots)), af)
}

// layoutChunks lays the field roots out over the configuration, the root of a
// field at each set position and a zero chunk at each cleared one. It is the
// one place the gaps are expanded, for the root and for the proofs alike.
func layoutChunks(af ActiveFields, roots [][32]byte) [][32]byte {
	if af.width == 0 {
		panic("ssz: progressive container with an uninitialized ActiveFields")
	}
	if len(roots) != af.nfields {
		panic(fmt.Sprintf("ssz: progressive container has %d field roots for %d active fields", len(roots), af.nfields))
	}
	chunks := make([][32]byte, af.width)
	k := 0
	for i := range chunks {
		if af.Active(i) {
			chunks[i] = roots[k]
			k++
		}
	}
	return chunks
}
