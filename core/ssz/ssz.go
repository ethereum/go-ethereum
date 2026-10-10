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

// Package ssz implements the Simple Serialize (SSZ) specification of the
// Ethereum consensus layer, following consensus-specs ssz/simple-serialize.md.
//
// The package covers serialization, strict deserialization and merkleization
// of the classic SSZ types. Types implement the Object interface with
// hand-written methods built from the primitives here: the Append* encoders,
// the Decoder cursor, the list and bitfield helpers, and Pack and Merkleize.
// There is no reflection and no code generation.
package ssz

import "fmt"

// Constants from the SSZ specification.
const (
	// BytesPerLengthOffset is the size of a serialized variable-size field offset.
	BytesPerLengthOffset = 4
	// MaxSize is the exclusive upper bound of any SSZ serialization
	// (sum of all parts must be < 2**32).
	MaxSize = 1 << 32
)

// Object is the interface implemented by all SSZ-serializable types.
//
// Methods are hand-written per type: fixed-size fields serialize in place,
// variable-size fields via the offset mechanism (see Decoder and the Append*
// helpers). HashTreeRoot cannot fail on a well-formed value; validity is
// established at construction or decode time.
type Object interface {
	// SizeSSZ returns the serialized size in bytes of the current value.
	SizeSSZ() int
	// MarshalSSZTo appends the SSZ serialization to dst and returns the
	// extended slice.
	MarshalSSZTo(dst []byte) ([]byte, error)
	// UnmarshalSSZ decodes the value from data. Decoding is strict: all of
	// data must be consumed and every malformed input rejected with an error
	// wrapping one of the named sentinel errors of this package.
	UnmarshalSSZ(data []byte) error
	// HashTreeRoot returns the SSZ Merkle root of the value.
	HashTreeRoot() [32]byte
}

// Encode serializes obj into a freshly allocated buffer.
func Encode(obj Object) ([]byte, error) {
	size := obj.SizeSSZ()
	if uint64(size) >= MaxSize { // widened: MaxSize overflows int on 32-bit platforms
		return nil, fmt.Errorf("ssz: encoding %T: size %d: %w", obj, size, ErrTooBig)
	}
	return obj.MarshalSSZTo(make([]byte, 0, size))
}

// Decode strictly deserializes data into obj.
func Decode(data []byte, obj Object) error {
	if uint64(len(data)) >= MaxSize {
		return fmt.Errorf("ssz: decoding %T: input %d bytes: %w", obj, len(data), ErrTooBig)
	}
	return obj.UnmarshalSSZ(data)
}

// HashTreeRoot computes the SSZ Merkle root of obj.
func HashTreeRoot(obj Object) [32]byte {
	return obj.HashTreeRoot()
}
