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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
)

var testParams = Params{
	logMapHeight:        8,
	logMapWidth:         24,
	logMapsPerEpoch:     6,
	logValuesPerMap:     8,
	logMappingFrequency: []uint{6, 4, 2, 0},
	maxRowLength:        []uint32{4, 16, 64, 256},
	rowGroupSize:        []uint32{16, 4, 1, 1},
}

func TestMapDatabase(t *testing.T) {
	testParams.sanitize()
	db := memorydb.New()
	mapDb := newMapDatabase(&testParams, db, false)
	maps := addTestMaps(&testParams, nil, 1000)
	writeMaps := make(map[uint32]*finishedMap)
	for i, fm := range maps {
		writeMaps[uint32(i)] = fm
	}
	mapDb.writeMaps(common.NewRange[uint32](0, 1000), common.Range[uint32]{}, common.Range[uint32]{}, writeMaps, func() bool { return false })
	reader := mapReader{
		getFilterMapRows:  mapDb.getFilterMapRows,
		getFilterMap:      mapDb.getFilterMap,
		getBlockLvPointer: mapDb.getBlockLvPointer,
		getLastBlockOfMap: mapDb.getLastBlockOfMap,
	}
	testMapReader(t, "mapDatabase test", &testParams, reader, nil, maps[:900])
}
