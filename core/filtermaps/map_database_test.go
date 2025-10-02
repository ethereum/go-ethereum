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
	"math"
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

func testStop() bool { return false }

func TestMapDatabase(t *testing.T) {
	params := testParams
	params.sanitize()
	db := memorydb.New()
	mapDb := newMapDatabase(&params, db, false)
	reader := mapReader{
		getFilterMapRows:  mapDb.getFilterMapRows,
		getFilterMap:      mapDb.getFilterMap,
		getBlockLvPointer: mapDb.getBlockLvPointer,
		getLastBlockOfMap: mapDb.getLastBlockOfMap,
	}
	// initialize database with checkpoints
	maps := generateTestMaps(&params, nil, 0x200)
	cpList := generateTestCheckpoints(&params, maps)
	mapDb.storeCheckpointList(0, cpList)
	// add new maps to the head
	maps = generateTestMaps(&params, maps, 0x50)
	mapDb.writeMapRows(common.NewRange[uint32](0x200, 0x50), common.Range[uint32]{}, common.NewRange[uint32](0x250, math.MaxUint32-0x250), maps[0x200:], testStop)
	mapDb.writePointers(common.NewRange[uint32](0x200, 0x50), maps[0x200:], testStop)
	testMapReader(t, "mapDatabase test #1", &params, reader, cpList, maps[0x200:])
	// backfill previous epoch with a single write
	mapDb.writeMapRows(common.NewRange[uint32](0x1c0, 0x40), common.Range[uint32]{}, common.Range[uint32]{}, maps[0x1c0:0x200], testStop)
	mapDb.writePointers(common.NewRange[uint32](0x1c0, 0x40), maps[0x1c0:0x200], testStop)
	testMapReader(t, "mapDatabase test #2", &params, reader, cpList[:7], maps[0x1c0:])
	// backfill previous epoch in two steps
	mapDb.writeMapRows(common.NewRange[uint32](0x180, 0x10), common.Range[uint32]{}, common.NewRange[uint32](0x190, 0x30), maps[0x180:0x190], testStop)
	mapDb.writePointers(common.NewRange[uint32](0x180, 0x10), maps[0x180:0x190], testStop)
	mapDb.writeMapRows(common.NewRange[uint32](0x190, 0x30), common.Range[uint32]{}, common.Range[uint32]{}, maps[0x190:0x1c0], testStop)
	mapDb.writePointers(common.NewRange[uint32](0x190, 0x30), maps[0x190:0x1c0], testStop)
	testMapReader(t, "mapDatabase test #3", &params, reader, cpList[:6], maps[0x180:])
	// add new maps while reorging some existing ones
	maps = generateTestMaps(&params, maps[:0x230], 0x30)
	mapDb.writeMapRows(common.NewRange[uint32](0x230, 0x30), common.NewRange[uint32](0x230, 0x30), common.NewRange[uint32](0x260, math.MaxUint32-0x260), maps[0x230:], testStop)
	mapDb.deletePointers(common.NewRange[uint32](0x230, math.MaxUint32-0x230), testStop)
	mapDb.writePointers(common.NewRange[uint32](0x230, 0x30), maps[0x230:], testStop)
	testMapReader(t, "mapDatabase test #4", &params, reader, cpList[:6], maps[0x180:])
	// unindex tail epoch
	mapDb.writeMapRows(common.Range[uint32]{}, common.NewRange[uint32](0x180, 0x40), common.Range[uint32]{}, nil, testStop)
	mapDb.deletePointers(common.NewRange[uint32](0x180, 0x40), testStop)
	testMapReader(t, "mapDatabase test #5", &params, reader, cpList[:7], maps[0x1c0:])
}
