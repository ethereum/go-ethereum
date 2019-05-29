// Copyright 2014 The go-ethereum Authors
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

/*
Package rlp implements the RLP serialization format.

The purpose of RLP (Recursive Linear Prefix) is to encode arbitrarily nested arrays of
binary data, and RLP is the main encoding method used to serialize objects in Ethereum.
The only purpose of RLP is to encode structure; encoding specific atomic data types (eg.
strings, ints, floats) is left up to higher-order protocols. In Ethereum integers must be
represented in big endian binary form with no leading zeroes (thus making the integer
value zero equivalent to the empty string).

RLP values are distinguished by a type tag. The type tag precedes the value in the input
stream and defines the size and kind of the bytes that follow.


Encoding Rules

Package rlp uses reflection and encodes RLP based on the Go type of the value.

If the type implements the Encoder interface, Encode calls EncodeRLP. It does not
call EncodeRLP on nil pointer values.

To encode a pointer, the value being pointed to is encoded. A nil pointer to a struct
type, slice or array always encodes as an empty RLP list unless the slice or array has
elememt type byte. A nil pointer to any other value encodes as the empty string.

Struct values are encoded as an RLP list of all their encoded public fields. Recursive
struct types are supported.

To encode slices and arrays, the elements are encoded as an RLP list of the value's
elements. Note that arrays and slices with element type uint8 or byte are always encoded
as an RLP string.

A Go string is encoded as an RLP string.

An unsigned integer value is encoded as an RLP string. Zero always encodes as an empty RLP
string. big.Int values are treated as integers. Signed integers (int, int8, int16, ...)
are not supported and will return an error when encoding.

Boolean values are encoded as the unsigned integers zero (false) and one (true).

An interface value encodes as the value contained in the interface.

Floating point numbers, maps, channels and functions are not supported.


Decoding Rules

Decoding uses the following type-dependent rules:

If the type implements the Decoder interface, DecodeRLP is called.

To decode into a pointer, the value will be decoded as the element type of the pointer. If
the pointer is nil, a new value of the pointer's element type is allocated. If the pointer
is non-nil, the existing value will be reused. Note that package rlp never leaves a
pointer-type struct field as nil unless one of the "nil" struct tags is present.

To decode into a struct, decoding expects the input to be an RLP list. The decoded
elements of the list are assigned to each public field in the order given by the struct's
definition. The input list must contain an element for each decoded field. Decoding
returns an error if there are too few or too many elements for the struct.

To decode into a slice, the input must be a list and the resulting slice will contain the
input elements in order. For byte slices, the input must be an RLP string. Array types
decode similarly, with the additional restriction that the number of input elements (or
bytes) must match the array's defined length.

To decode into a Go string, the input must be an RLP string. The input bytes are taken
as-is and will not necessarily be valid UTF-8.

To decode into an unsigned integer type, the input must also be an RLP string. The bytes
are interpreted as a big endian representation of the integer. If the RLP string is larger
than the bit size of the type, decoding will return an error. Decode also supports
*big.Int. There is no size limit for big integers.

To decode into a boolean, the input must contain an unsigned integer of value zero (false)
or one (true).

To decode into an interface value, one of these types is stored in the value:

	  []interface{}, for RLP lists
	  []byte, for RLP strings

Non-empty interface types are not supported when decoding.
Signed integers, floating point numbers, maps, channels and functions cannot be decoded into.


Struct Tags

Package rlp honours certain struct tags: "-", "tail", "nil", "nilList" and "nilString".

The "-" tag ignores fields.

The "tail" tag, which may only be used on the last exported struct field, allows slurping
up any excess list elements into a slice. See examples for more details.

The "nil" tag applies to pointer-typed fields and changes the decoding rules for the field
such that input values of size zero decode as a nil pointer. This tag can be useful when
decoding recursive types.

    type StructWithOptionalFoo struct {
        Foo *[20]byte `rlp:"nil"`
    }

RLP supports two kinds of empty values: empty lists and empty strings. When using the
"nil" tag, the kind of empty value allowed for a type is chosen automatically. A struct
field whose Go type is a pointer to an unsigned integer, string, boolean or byte
array/slice expects an empty RLP string. Any other pointer field type encodes/decodes as
an empty RLP list.

The choice of null value can be made explicit with the "nilList" and "nilString" struct
tags. Using these tags encodes/decodes a Go nil pointer value as the kind of empty
RLP value defined by the tag.
*/
package rlp
