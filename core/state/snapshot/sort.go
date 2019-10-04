// Copyright 2019 The go-ethereum Authors
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

package snapshot

import (
	"bytes"

	"github.com/ethereum/go-ethereum/common"
)

// hashes is a helper to implement sort.Interface.
type hashes []common.Hash

// Len is the number of elements in the collection.
func (hs hashes) Len() int { return len(hs) }

// Less reports whether the element with index i should sort before the element
// with index j.
func (hs hashes) Less(i, j int) bool { return bytes.Compare(hs[i][:], hs[j][:]) < 0 }

// Swap swaps the elements with indexes i and j.
func (hs hashes) Swap(i, j int) { hs[i], hs[j] = hs[j], hs[i] }

// merge combines two sorted lists of hashes into a combo sorted one.
func merge(a, b []common.Hash) []common.Hash {
	result := make([]common.Hash, len(a)+len(b))

	i := 0
	for len(a) > 0 && len(b) > 0 {
		if bytes.Compare(a[0][:], b[0][:]) < 0 {
			result[i] = a[0]
			a = a[1:]
		} else {
			result[i] = b[0]
			b = b[1:]
		}
		i++
	}
	for j := 0; j < len(a); j++ {
		result[i] = a[j]
		i++
	}
	for j := 0; j < len(b); j++ {
		result[i] = b[j]
		i++
	}
	return result
}

// dedupMerge combines two sorted lists of hashes into a combo sorted one,
// and removes duplicates in the process
func dedupMerge(a, b []common.Hash) []common.Hash {
	result := make([]common.Hash, len(a)+len(b))
	i := 0
	for len(a) > 0 && len(b) > 0 {
		if diff := bytes.Compare(a[0][:], b[0][:]); diff < 0 {
			result[i] = a[0]
			a = a[1:]
		} else {
			result[i] = b[0]
			b = b[1:]
			// If they were equal, progress a too
			if diff == 0 {
				a = a[1:]
			}
		}
		i++
	}
	for j := 0; j < len(a); j++ {
		result[i] = a[j]
		i++
	}
	for j := 0; j < len(b); j++ {
		result[i] = b[j]
		i++
	}
	return result[:i]
}
