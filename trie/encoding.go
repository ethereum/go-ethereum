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

package trie

// This file implements a codec to convert between byte arrays
// and nibble (half-byte) arrays, where the latter are used for traversing the trie.
// Hence the "compact" encoded form for a key is as the byte array, which
// is (roughly) half the length of the decoded nibble array.
// Nibble arrays for keys that represent leaf nodes have a terminator flag (numerical 16) appended to the end.
// The compact encoded form uses Hex Prefix (HP) encoding, to encode the terminator status
// and whether the key length is even or odd in the first two bytes (note the original description uses only
// one byte, but that's inconvenient)

// Encode a slice of nibbles into a HP byte array
func CompactEncode(hexSlice []byte) []byte {
	terminator := 0
	if hexSlice[len(hexSlice)-1] == 16 {
		terminator = 1
		hexSlice = hexSlice[:len(hexSlice)-1]
	}

	oddlen := len(hexSlice) % 2
	flags := byte(2*terminator + oddlen)
	if oddlen != 0 {
		hexSlice = append([]byte{flags}, hexSlice...)
	} else {
		hexSlice = append([]byte{flags, 0}, hexSlice...)
	}

	l := len(hexSlice) / 2
	var buf = make([]byte, l)
	for i := 0; i < l; i++ {
		buf[i] = 16*hexSlice[2*i] + hexSlice[2*i+1]
	}
	return buf
}

// Decode a HP encoded byte array into a nibble array
// with terminator flag if applicable
func CompactDecode(key []byte) []byte {
	base := CompactHexDecode(key) // appends the terminator flag by default
	if base[0] < 2 {
		// remove the terminator flag if its not in the HP
		base = base[:len(base)-1]
	}

	// HP tells us if key length is even or odd
	if base[0]%2 == 1 {
		base = base[1:]
	} else {
		base = base[2:]
	}

	return base
}

// Decode a byte array into a nibble array.
// Assumes the key coressponds to a terminator node (ie appends 16)
// CompactHexDecode is called immediately by the Get/Update/Remove functions.
func CompactHexDecode(key []byte) []byte {
	l := len(key)*2 + 1
	var nibbles = make([]byte, l)
	for i, b := range key {
		nibbles[i*2] = b / 16
		nibbles[i*2+1] = b % 16
	}
	nibbles[l-1] = 16
	return nibbles
}

// This is really a compact encoding of nibbles
// without hex-prefix
func DecodeCompact(nibbles []byte) []byte {
	l := len(nibbles) / 2
	var res = make([]byte, l)
	for i := 0; i < l; i++ {
		res[i] = 16*nibbles[2*i] + nibbles[2*i+1]
	}
	return res
}
