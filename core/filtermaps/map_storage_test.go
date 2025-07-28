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

func TestMapStorage(t *testing.T) {
	params := testParams
	params.sanitize()
	db := memorydb.New()
	mapDb := newMapDatabase(&params, db, false)
	ms := newMapStorage(&params, mapDb, make(chan bool))
	<-ms.testHookCh
	defer ms.stop()

	reader := mapReader{
		getFilterMapRows:  ms.getFilterMapRows,
		getFilterMap:      ms.getFilterMap,
		getBlockLvPointer: ms.getBlockLvPointer,
		getLastBlockOfMap: ms.getLastBlockOfMap,
	}
	waitCycle := func() bool {
		ms.testHookCh <- true
		return !<-ms.testHookCh
	}
	expKnownEpochs := func(testCase int, exp uint32) {
		if ms.knownEpochs != exp {
			t.Fatalf("Invalid known epochs number in test case #%d (expected %d, got %d)", testCase, exp, ms.knownEpochs)
		}
	}
	expTailEpoch := func(testCase int, exp uint32) {
		if ms.tailEpoch() != exp {
			t.Fatalf("Invalid tail epoch in test case #%d (expected %d, got %d)", testCase, exp, ms.tailEpoch())
		}
	}
	expReadRows := func(testCase int, exp bool) {
		if mapDb.testReadRows != exp {
			t.Fatalf("Invalid read rows flag in test case #%d (expected %v, got %v)", testCase, exp, mapDb.testReadRows)
		}
		mapDb.testReadRows = false
	}
	// initialize database with checkpoints
	maps := generateTestMaps(&params, nil, 0x200)
	cpList := generateTestCheckpoints(&params, maps)
	ms.addKnownEpochs(cpList)
	expTailEpoch(0, 8)
	expKnownEpochs(0, 8)
	// add new maps to the head
	maps = generateTestMaps(&params, maps, 0x50)
	for m := uint32(0x200); m < 0x250; m++ {
		ms.addMap(m, maps[m], true)
	}
	for waitCycle() {
		testMapReader(t, "mapStorage test #1", &params, reader, cpList, maps[0x200:])
	}
	expReadRows(1, false)
	expTailEpoch(1, 8)
	expKnownEpochs(1, 9)
	// backfill previous epoch with a single write
	for m := uint32(0x1c0); m < 0x200; m++ {
		ms.addMap(m, maps[m], false)
	}
	for waitCycle() {
		testMapReader(t, "mapDatabase test #2", &params, reader, cpList[:7], maps[0x1c0:])
	}
	expReadRows(2, false)
	expTailEpoch(2, 7)
	expKnownEpochs(2, 9)
	// backfill previous epoch in two steps
	for m := uint32(0x180); m < 0x192; m++ {
		ms.addMap(m, maps[m], true)
	}
	for waitCycle() {
	}
	expReadRows(3, false)
	expTailEpoch(3, 7)
	for m := uint32(0x192); m < 0x1c0; m++ {
		ms.addMap(m, maps[m], false)
	}
	for waitCycle() {
		testMapReader(t, "mapDatabase test #3", &params, reader, cpList[:6], maps[0x180:])
	}
	expReadRows(3, true)
	expTailEpoch(3, 6)
	expKnownEpochs(3, 9)
	// add new maps while reorging some existing ones
	maps = generateTestMaps(&params, maps[:0x234], 0x30)
	ms.deleteMaps(common.NewRange[uint32](0x234, math.MaxUint32-0x234))
	expKnownEpochs(4, 8)
	for m := uint32(0x234); m < 0x264; m++ {
		ms.addMap(m, maps[m], true)
	}
	for waitCycle() {
		testMapReader(t, "mapDatabase test #4", &params, reader, cpList[:6], maps[0x180:])
	}
	expReadRows(4, true)
	expTailEpoch(4, 6)
	expKnownEpochs(4, 9)
	// unindex tail epoch
	ms.deleteMaps(common.NewRange[uint32](0x180, 0x40))
	expTailEpoch(5, 7)
	for waitCycle() {
		testMapReader(t, "mapDatabase test #5", &params, reader, cpList[:7], maps[0x1c0:])
	}
	expReadRows(5, false)
	expTailEpoch(5, 7)
	expKnownEpochs(5, 9)
	// remove head maps
	maps = maps[:0x253]
	ms.deleteMaps(common.NewRange[uint32](0x253, math.MaxUint32-0x253))
	for waitCycle() {
		testMapReader(t, "mapDatabase test #6", &params, reader, cpList[:7], maps[0x1c0:])
	}
	expReadRows(6, true)
	expTailEpoch(6, 7)
	expKnownEpochs(6, 9)
	// add maps until epoch boundary and check known epochs increase
	maps = generateTestMaps(&params, maps, 0x30)
	for m := uint32(0x253); m < 0x283; m++ {
		ms.addMap(m, maps[m], true)
	}
	expTailEpoch(7, 7)
	expKnownEpochs(7, 9)
	for waitCycle() {
		testMapReader(t, "mapDatabase test #7", &params, reader, cpList[:7], maps[0x1c0:])
	}
	expReadRows(7, true)
	expTailEpoch(7, 7)
	expKnownEpochs(7, 10)
}
