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

import "math"

// Trie keys are dealt with in three distinct encodings:
//
// KEYBYTES encoding contains the actual key and nothing else. All bits in each byte of this key
// are significant. This encoding is the input to most API functions.
//
// BINARY encoding contains one byte for each bit of the key and an optional trailing
// 'terminator' byte of value 2 which indicates whether or not the node at the key
// contains a value. The first  (most significant) 7 bits of each byte are always 0
// (except for the terminator, which has 6 zero-bits to start). Our tries use this
// encoding under the hood because it permits the trie to be binary -- allowing 2^8
// distinct key paths for each key byte instead of just 2.
//
// COMPACT encoding is a way of storing a binary-encoded key or a slice of a binary-encoded key
// in as efficient of a way as possible. This entails tightly-packing the data into bytes without
// padding (except to fill out the last byte) while still capturing all binary key metadata.
// The compact encoding takes the format [header nibble] [key] [padding bits]
// Header Nibble:
// - first bit: 1 if should be terminated / 0 if not (see 'terminator' byte above)
// - bits 2-4: the number of unused, least significant bits in the last byte of the compact key
//   - Calculated as [8 - ((4 (for header nibble) + key length without terminator) % 8)] % 8
// Body:
// - key bits are tightly packed starting at bit 5 of the first byte (after the header nibble)
// Padding:
// - If the first nibble plus the number of key bits is not an even multiple of 8, the unused bits
//   of the last byte will contain 0s
//
// Example BINARY-encoded key conversion to COMPACT encoding:
// BINARY key: 1 1 0 1 1 2(terminator)
// COMPACT first bit = 1 (terminator present)
// COMPACT bits 2-4 = [8 - ((4 (for header nibble) + key length without terminator) % 8)] % 8
//                  = [8 - ((4 + 5) % 8)] %8 = 7 unused bits in the last byte = 111
// COMPACT first nibble: 1111
// COMPACT key = 1111 1101 1[000 0000], 2 bytes total, where the last 7 bits of the last byte are unused.

// Converts the provided BINARY-encoded key into the COMPACT-encoded format detailed above.
func binaryKeyToCompactKey(binaryKey []byte) []byte {
	currentByte := uint8(0)
	keyLength := len(binaryKey)

	// Set the first bit of the first byte if terminator is present, then remove it from the key.
	if hasBinaryKeyTerminator(binaryKey) {
		binaryKey = binaryKey[:len(binaryKey)-1]
		currentByte = 1 << 7
		keyLength--
	}

	lastByteUnusedBits := uint8((8 - (4+keyLength)%8) % 8)
	currentByte += lastByteUnusedBits << 4

	returnLength := (keyLength + 4 + int(lastByteUnusedBits)) / 8
	returnBytes := make([]byte, returnLength)
	returnIndex := 0
	for i := 0; i < len(binaryKey); i++ {
		bitPosition := (4 + i) % 8
		if bitPosition == 0 {
			returnBytes[returnIndex] = currentByte
			currentByte = uint8(0)
			returnIndex++
		}

		currentByte += (1 & binaryKey[i]) << (7 - bitPosition)
	}
	returnBytes[returnIndex] = currentByte

	return returnBytes
}

// Converts the provided key from the COMPACT encoding to the BINARY key format (both specified above).
func compactKeyToBinaryKey(compactKey []byte) []byte {
	if len(compactKey) == 0 {
		// This technically is an invalid compact format
		return make([]byte, 0)
	}

	addTerminator := compactKey[0] >> 7
	lastByteUnusedBits := (compactKey[0] << 1) >> 5

	binaryKeyLength := len(compactKey)*8 - 4   // length - header nibble
	binaryKeyLength += int(addTerminator)      // terminator byte
	binaryKeyLength -= int(lastByteUnusedBits) // extra padding bits

	if binaryKeyLength < 0 {
		// Invalid key
		return make([]byte, 0)
	}

	binaryKey := make([]byte, binaryKeyLength)

	binaryKeyIndex := 0
	compactKeyByteIndex := 0
	currentBitIndex := 4
	currentByte := compactKey[compactKeyByteIndex]
	for ; binaryKeyIndex < binaryKeyLength-int(addTerminator); currentBitIndex++ {
		shift := 7 - (currentBitIndex % 8)
		if shift == 7 {
			compactKeyByteIndex++
			currentByte = compactKey[compactKeyByteIndex]
		}
		binaryKey[binaryKeyIndex] = (currentByte & (1 << shift)) >> shift
		binaryKeyIndex++
	}

	if addTerminator > 0 && binaryKeyLength > 0 {
		binaryKey[binaryKeyLength-1] = binaryKeyTerminator
	}

	return binaryKey
}

// Converts the provided key from KEYBYTES encoding to BINARY encoding (both listed above).
func keyBytesToBinaryKey(key []byte) []byte {
	length := len(key)*8 + 1
	var binaryKey = make([]byte, length)
	for i, keyByte := range key {
		for bit := 0; bit < 8; bit++ {
			shift := 7 - bit
			binaryKey[i*8+bit] = keyByte & (1 << shift) >> shift
		}
	}
	binaryKey[length-1] = binaryKeyTerminator
	return binaryKey
}

// Converts the provided key from BINARY encoding to KEYBYTES encoding (both listed above).
func binaryKeyToKeyBytes(binaryKey []byte) (keyBytes []byte) {
	if hasBinaryKeyTerminator(binaryKey) {
		binaryKey = binaryKey[:len(binaryKey)-1]
	}
	if len(binaryKey) == 0 {
		return make([]byte, 0)
	}

	keyLength := int(math.Ceil(float64(len(binaryKey)) / 8.0))
	keyBytes = make([]byte, keyLength)

	byteInt := uint8(0)
	for bit := 0; bit < len(binaryKey); bit++ {
		byteBit := bit % 8
		if byteBit == 0 && bit != 0 {
			keyBytes[(bit/8)-1] = byteInt
			byteInt = 0
		}
		byteInt += (1 << (7 - byteBit)) * binaryKey[bit]
	}

	keyBytes[keyLength-1] = byteInt

	return keyBytes
}

// prefixLen returns the length of the common prefix of a and b.
func prefixLen(a, b []byte) int {
	var i, length = 0, len(a)
	if len(b) < length {
		length = len(b)
	}
	for ; i < length; i++ {
		if a[i] != b[i] {
			break
		}
	}
	return i
}

const binaryKeyTerminator = 2

// hasBinaryKeyTerminator returns whether a BINARY encoded key has the terminator flag.
func hasBinaryKeyTerminator(binaryKey []byte) bool {
	return len(binaryKey) > 0 && binaryKey[len(binaryKey)-1] == binaryKeyTerminator
}
