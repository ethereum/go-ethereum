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

func CompactEncode(hexSlice []byte) []byte {
	terminator := 0
	if hexSlice[len(hexSlice)-1] == 16 {
		terminator = 1
	}

	if terminator == 1 {
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

func CompactDecode(str []byte) []byte {
	base := CompactHexDecode(str)
	base = base[:len(base)-1]
	if base[0] >= 2 {
		base = append(base, 16)
	}
	if base[0]%2 == 1 {
		base = base[1:]
	} else {
		base = base[2:]
	}

	return base
}

func CompactHexDecode(str []byte) []byte {
	l := len(str)*2 + 1
	var nibbles = make([]byte, l)
	for i, b := range str {
		nibbles[i*2] = b / 16
		nibbles[i*2+1] = b % 16
	}
	nibbles[l-1] = 16
	return nibbles
}

func DecodeCompact(key []byte) []byte {
	l := len(key) / 2
	var res = make([]byte, l)
	for i := 0; i < l; i++ {
		v1, v0 := key[2*i], key[2*i+1]
		res[i] = v1*16 + v0
	}
	return res
}
