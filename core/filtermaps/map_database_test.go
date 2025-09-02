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
	"slices"
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
	cpList := generateTestCheckpoints(&testParams, maps)[:4]
	for epoch, cp := range cpList {
		mapDb.storeEpochCheckpoint(uint32(epoch), cp)
	}
	reader := mapReader{
		getFilterMapRows:  mapDb.getFilterMapRows,
		getFilterMap:      mapDb.getFilterMap,
		getBlockLvPointer: mapDb.getBlockLvPointer,
		getLastBlockOfMap: mapDb.getLastBlockOfMap,
	}
	mapDb.writeMaps(common.NewRange[uint32](256, 768), common.Range[uint32]{}, common.Range[uint32]{}, maps[256:], func() bool { return false })
	testMapReader(t, "mapDatabase test 1", &testParams, reader, cpList, maps[256:])
	mapDb.writeMaps(common.NewRange[uint32](0, 256), common.Range[uint32]{}, common.Range[uint32]{}, maps[:256], func() bool { return false })
	testMapReader(t, "mapDatabase test 2", &testParams, reader, nil, maps)
	maps2 := generateTestMaps(&testParams, slices.Clone(maps[:512]), 1024)
	mapDb.writeMaps(common.NewRange[uint32](512, 1024), common.NewRange[uint32](512, 512), common.Range[uint32]{}, maps2[512:], func() bool { return false })
	testMapReader(t, "mapDatabase test 3", &testParams, reader, nil, maps2)
}
