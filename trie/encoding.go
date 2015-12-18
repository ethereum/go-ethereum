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

func compactEncode(hexSlice []byte) []byte {
	terminator := byte(0)
	if hexSlice[len(hexSlice)-1] == 16 {
		terminator = 1
		hexSlice = hexSlice[:len(hexSlice)-1]
	}
	var (
		odd    = byte(len(hexSlice) % 2)
		buflen = len(hexSlice)/2 + 1
		bi, hi = 0, 0    // indices
		hs     = byte(0) // shift: flips between 0 and 4
	)
	if odd == 0 {
		bi = 1
		hs = 4
	}
	buf := make([]byte, buflen)
	buf[0] = terminator<<5 | byte(odd)<<4
	for bi < len(buf) && hi < len(hexSlice) {
		buf[bi] |= hexSlice[hi] << hs
		if hs == 0 {
			bi++
		}
		hi, hs = hi+1, hs^(1<<2)
	}
	return buf
}

func compactDecode(str []byte) []byte {
	base := compactHexDecode(str)
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

func compactHexDecode(str []byte) []byte {
	l := len(str)*2 + 1
	var nibbles = make([]byte, l)
	for i, b := range str {
		nibbles[i*2] = b / 16
		nibbles[i*2+1] = b % 16
	}
	nibbles[l-1] = 16
	return nibbles
}

// compactHexEncode encodes a series of nibbles into a byte array
func compactHexEncode(nibbles []byte) []byte {
	nl := len(nibbles)
	if nl == 0 {
		return nil
	}
	if nibbles[nl-1] == 16 {
		nl--
	}
	l := (nl + 1) / 2
	var str = make([]byte, l)
	for i, _ := range str {
		b := nibbles[i*2] * 16
		if nl > i*2 {
			b += nibbles[i*2+1]
		}
		str[i] = b
	}
	return str
}

func decodeCompact(key []byte) []byte {
	l := len(key) / 2
	var res = make([]byte, l)
	for i := 0; i < l; i++ {
		v1, v0 := key[2*i], key[2*i+1]
		res[i] = v1*16 + v0
	}
	return res
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

func hasTerm(s []byte) bool {
	return s[len(s)-1] == 16
}

func remTerm(s []byte) []byte {
	if hasTerm(s) {
		b := make([]byte, len(s)-1)
		copy(b, s)
		return b
	}
	return s
}
