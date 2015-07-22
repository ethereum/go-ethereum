// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// go-ethereum is free software: you can redistribute it and/or modify
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

import (
	"bytes"
	"encoding/hex"
	"strings"
)

func CompactEncode(hexSlice []byte) string {
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

	var buff bytes.Buffer
	for i := 0; i < len(hexSlice); i += 2 {
		buff.WriteByte(byte(16*hexSlice[i] + hexSlice[i+1]))
	}

	return buff.String()
}

func CompactDecode(str string) []byte {
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

func CompactHexDecode(str string) []byte {
	base := "0123456789abcdef"
	var hexSlice []byte

	enc := hex.EncodeToString([]byte(str))
	for _, v := range enc {
		hexSlice = append(hexSlice, byte(strings.IndexByte(base, byte(v))))
	}
	hexSlice = append(hexSlice, 16)

	return hexSlice
}

func DecodeCompact(key []byte) string {
	const base = "0123456789abcdef"
	var str string

	for _, v := range key {
		if v < 16 {
			str += string(base[v])
		}
	}

	res, _ := hex.DecodeString(str)

	return string(res)
}
