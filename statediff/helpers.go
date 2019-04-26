// Copyright 2015 The go-ethereum Authors
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

// Contains a batch of utility type declarations used by the tests. As the node
// operates on unique types, a lot of them are needed to check various features.

package statediff

import (
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/trie"
)

func sortKeys(data AccountsMap) []string {
	var keys []string
	for key := range data {
		keys = append(keys, key.Hex())
	}
	sort.Strings(keys)

	return keys
}

// BytesToNiblePath
func bytesToNiblePath(path []byte) string {
	if hasTerm(path) {
		path = path[:len(path)-1]
	}
	nibblePath := ""
	for i, v := range common.ToHex(path) {
		if i%2 == 0 && i > 1 {
			continue
		}
		nibblePath = nibblePath + string(v)
	}

	return nibblePath
}

func findIntersection(a, b []string) []string {
	lenA := len(a)
	lenB := len(b)
	iOfA, iOfB := 0, 0
	updates := make([]string, 0)
	if iOfA >= lenA || iOfB >= lenB {
		return updates
	}
	for {
		switch strings.Compare(a[iOfA], b[iOfB]) {
		// a[iOfA] < b[iOfB]
		case -1:
			iOfA++
			if iOfA >= lenA {
				return updates
			}
			// a[iOfA] == b[iOfB]
		case 0:
			updates = append(updates, a[iOfA])
			iOfA++
			iOfB++
			if iOfA >= lenA || iOfB >= lenB {
				return updates
			}
			// a[iOfA] > b[iOfB]
		case 1:
			iOfB++
			if iOfB >= lenB {
				return updates
			}
		}
	}

}

func pathToStr(it trie.NodeIterator) string {
	return bytesToNiblePath(it.Path())
}

// Duplicated from trie/encoding.go
func hexToKeyBytes(hex []byte) []byte {
	if hasTerm(hex) {
		hex = hex[:len(hex)-1]
	}
	if len(hex)&1 != 0 {
		panic("can't convert hex key of odd length")
	}
	key := make([]byte, (len(hex)+1)/2)
	decodeNibbles(hex, key)

	return key
}

func decodeNibbles(nibbles []byte, bytes []byte) {
	for bi, ni := 0, 0; ni < len(nibbles); bi, ni = bi+1, ni+2 {
		bytes[bi] = nibbles[ni]<<4 | nibbles[ni+1]
	}
}

// hasTerm returns whether a hex key has the terminator flag.
func hasTerm(s []byte) bool {
	return len(s) > 0 && s[len(s)-1] == 16
}
