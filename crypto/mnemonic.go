// Copyright 2014 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.

package crypto

import (
	"fmt"
	"strconv"
)

// TODO: See if we can refactor this into a shared util lib if we need it multiple times
func IndexOf(slice []string, value string) int64 {
	for p, v := range slice {
		if v == value {
			return int64(p)
		}
	}
	return -1
}

func MnemonicEncode(message string) []string {
	var out []string
	n := int64(len(MnemonicWords))

	for i := 0; i < len(message); i += (len(message) / 8) {
		x := message[i : i+8]
		bit, _ := strconv.ParseInt(x, 16, 64)
		w1 := (bit % n)
		w2 := ((bit / n) + w1) % n
		w3 := ((bit / n / n) + w2) % n
		out = append(out, MnemonicWords[w1], MnemonicWords[w2], MnemonicWords[w3])
	}
	return out
}

func MnemonicDecode(wordsar []string) string {
	var out string
	n := int64(len(MnemonicWords))

	for i := 0; i < len(wordsar); i += 3 {
		word1 := wordsar[i]
		word2 := wordsar[i+1]
		word3 := wordsar[i+2]
		w1 := IndexOf(MnemonicWords, word1)
		w2 := IndexOf(MnemonicWords, word2)
		w3 := IndexOf(MnemonicWords, word3)

		y := (w2 - w1) % n
		z := (w3 - w2) % n

		// Golang handles modulo with negative numbers different then most languages
		// The modulo can be negative, we don't want that.
		if z < 0 {
			z += n
		}
		if y < 0 {
			y += n
		}
		x := w1 + n*(y) + n*n*(z)
		out += fmt.Sprintf("%08x", x)
	}
	return out
}
