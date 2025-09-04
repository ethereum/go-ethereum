// Copyright 2025 The go-ethereum Authors
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

package filtermaps

import (
	"testing"

	"github.com/ethereum/go-ethereum/ethdb/memorydb"
)

func TestMapStorage(t *testing.T) {
	testParams.sanitize()
	db := memorydb.New()
	mapDb := newMapDatabase(&testParams, db, false)
	ms := newMapStorage(&testParams, mapDb)
	reader := mapReader{
		getFilterMapRows:  ms.getFilterMapRows,
		getFilterMap:      ms.getFilterMap,
		getBlockLvPointer: ms.getBlockLvPointer,
		getLastBlockOfMap: ms.getLastBlockOfMap,
	}
	// initialize database with checkpoints
	maps := generateTestMaps(&testParams, nil, 0x200)
	cpList := generateTestCheckpoints(&testParams, maps)
	ms.addKnownEpochs(cpList)
	// add new maps to the head
	maps = generateTestMaps(&testParams, maps, 0x50)
	for m := uint32(0x200); m < 0x250; m++ {
		ms.addMap(m, maps[m], false)
	}
	testMapReader(t, "mapStorage test #1", &testParams, reader, cpList, maps[0x200:])
}
