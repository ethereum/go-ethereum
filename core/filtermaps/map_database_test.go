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
	maps := generateTestMaps(&testParams, nil, 1024)
	writeMaps := make(map[uint32]*finishedMap)
	for i, fm := range maps {
		writeMaps[uint32(i)] = fm
	}
	cpList := generateTestCheckpoints(&testParams, maps)[:4]
	for epoch, cp := range cpList {
		mapDb.storeEpochCheckpoint(uint32(epoch), cp)
	}
	mapDb.writeMaps(common.NewRange[uint32](256, 768), common.Range[uint32]{}, common.Range[uint32]{}, writeMaps, func() bool { return false })
	reader := mapReader{
		getFilterMapRows:  mapDb.getFilterMapRows,
		getFilterMap:      mapDb.getFilterMap,
		getBlockLvPointer: mapDb.getBlockLvPointer,
		getLastBlockOfMap: mapDb.getLastBlockOfMap,
	}
	testMapReader(t, "mapDatabase test", &testParams, reader, cpList, maps[256:])
}
