// Copyright 2020 The go-ethereum Authors
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

package snap

import "slices"

// enrEntry is the ENR entry which advertises `snap` protocol on the discovery.
// It lists the served protocol versions in ascending order. An empty list means
// snap/1 only.
type enrEntry struct {
	Versions []uint `rlp:"tail"`
}

// newENREntry creates the entry for the given served versions.
func newENREntry(versions []uint) *enrEntry {
	if len(versions) == 1 && versions[0] == SNAP1 {
		return &enrEntry{}
	}
	return &enrEntry{Versions: slices.Sorted(slices.Values(versions))}
}

// ENRKey implements enr.Entry.
func (e enrEntry) ENRKey() string {
	return "snap"
}
