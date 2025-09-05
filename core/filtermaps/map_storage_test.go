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
	"fmt"
	"math"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
)

func TestMapStorage(t *testing.T) {
	testParams.sanitize()
	db := memorydb.New()
	mapDb := newMapDatabase(&testParams, db, false)
	ms := newMapStorage(&testParams, mapDb, make(chan bool))
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
	// initialize database with checkpoints
	maps := generateTestMaps(&testParams, nil, 0x200)
	cpList := generateTestCheckpoints(&testParams, maps)
	ms.addKnownEpochs(cpList)
	// add new maps to the head
	maps = generateTestMaps(&testParams, maps, 0x50)
	for m := uint32(0x200); m < 0x250; m++ {
		ms.addMap(m, maps[m], false)
	}
	for waitCycle() {
		fmt.Println("#1", ms.tailEpoch())
		testMapReader(t, "mapStorage test #1", &testParams, reader, cpList, maps[0x200:])
	}
	// backfill previous epoch with a single write
	for m := uint32(0x1c0); m < 0x200; m++ {
		ms.addMap(m, maps[m], false)
	}
	for waitCycle() {
		fmt.Println("#2", ms.tailEpoch())
		testMapReader(t, "mapDatabase test #2", &testParams, reader, cpList[:7], maps[0x1c0:])
	}
	// backfill previous epoch in two steps
	for m := uint32(0x180); m < 0x190; m++ {
		ms.addMap(m, maps[m], true)
	}
	for waitCycle() {
	}
	for m := uint32(0x190); m < 0x1c0; m++ {
		ms.addMap(m, maps[m], false)
	}
	for waitCycle() {
		fmt.Println("#3", ms.tailEpoch())
		testMapReader(t, "mapDatabase test #3", &testParams, reader, cpList[:6], maps[0x180:])
	}
	// add new maps while reorging some existing ones
	maps = generateTestMaps(&testParams, maps[:0x230], 0x30)
	ms.deleteMaps(common.NewRange[uint32](0x230, math.MaxUint32-0x230))
	for m := uint32(0x230); m < 0x260; m++ {
		ms.addMap(m, maps[m], false)
	}
	for waitCycle() {
		fmt.Println("#4", ms.tailEpoch())
		testMapReader(t, "mapDatabase test #4", &testParams, reader, cpList[:6], maps[0x180:])
	}
	// unindex tail epoch
	ms.deleteMaps(common.NewRange[uint32](0x180, 0x40))
	for waitCycle() {
		fmt.Println("#5", ms.tailEpoch())
		testMapReader(t, "mapDatabase test #5", &testParams, reader, cpList[:7], maps[0x1c0:])
	}
}
