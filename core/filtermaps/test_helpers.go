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
	"math/rand"
	"reflect"
	"slices"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

const (
	testFilterMapRowsCount = 1000
	testFilterMapsCount    = 100
	testPointersCount      = 1000
	testMaxBlocksPerMap    = 4
)

type mapReader struct {
	getFilterMapRows  func(mapIndices []uint32, rowIndex, layerIndex uint32) ([]FilterRow, error)
	getFilterMap      func(mapIndex uint32) (*finishedMap, error)
	getBlockLvPointer func(blockNumber uint64) (uint64, error)
	getLastBlockOfMap func(mapIndex uint32) (uint64, common.Hash, error)
}

func equalRows(a, b []FilterRow) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !slices.Equal(a[i], b[i]) {
			return false
		}
	}
	return true
}

func testMapReader(t *testing.T, testCase string, params *Params, reader mapReader, knownEpochs checkpointList, maps []*finishedMap) {
	mapRange := common.NewRange[uint32](uint32(len(knownEpochs))*params.mapsPerEpoch, uint32(len(maps)))
	for range testFilterMapRowsCount {
		rowIndex := uint32(rand.Intn(int(params.mapHeight)))
		lastDbLayer := uint32(rand.Intn(len(params.maxRowLength)))
		maxLen := params.maxRowLength[lastDbLayer]
		mid := mapRange.First() + uint32(rand.Intn(int(mapRange.Count())))
		count := uint32(rand.Intn(int(params.mapsPerEpoch)))
		first := max(mid, count/2) - count/2
		testRange := common.NewRange[uint32](first, count)
		mapIndices := make([]uint32, 0, testRange.Count())
		expResults := make([]FilterRow, 0, testRange.Count())
		for mapIndex := range testRange.Iter() {
			if rand.Intn(2) == 1 {
				mapIndices = append(mapIndices, mapIndex)
				if mapRange.Includes(mapIndex) {
					expResults = append(expResults, maps[mapIndex-mapRange.First()].getRow(rowIndex, maxLen))
				} else {
					expResults = append(expResults, nil)
				}
			}
		}
		rows, err := reader.getFilterMapRows(mapIndices, rowIndex, lastDbLayer)
		if err != nil {
			t.Fatalf("Test case %s: error reading %d map indices in range %d <= i < %d: %v", testCase, len(mapIndices), testRange.First(), testRange.AfterLast(), err)
		} else if !equalRows(rows, expResults) {
			t.Fatalf("Test case %s: incorrect results when reading %d map indices in range %d <= i < %d", testCase, len(mapIndices), testRange.First(), testRange.AfterLast())
		}
	}
	for range testFilterMapsCount {
		mapIndex := mapRange.First() + uint32(rand.Intn(int(mapRange.Count())))
		expResults := maps[mapIndex-mapRange.First()]
		fm, err := reader.getFilterMap(mapIndex)
		if err != nil {
			t.Fatalf("Test case %s: error reading map %d: %v", testCase, mapIndex, err)
		} else if !reflect.DeepEqual(fm, expResults) {
			t.Fatalf("Test case %s: incorrect results when reading map %d", testCase, mapIndex)
		}
	}

	testPointers := func(mapIndex uint32, expLastBlock lastBlockOfMap, blockNumber, expBlockPtr uint64) {
		lastBlock, lbHash, err := reader.getLastBlockOfMap(mapIndex)
		if err != nil {
			t.Fatalf("Test case %s: error reading last block of map %d: %v", testCase, mapIndex, err)
		} else if (lastBlockOfMap{number: lastBlock, hash: lbHash}) != expLastBlock {
			t.Fatalf("Test case %s: incorrect results when reading last block of map %d (expected %v, got %v)", testCase, mapIndex, expLastBlock, lastBlockOfMap{number: lastBlock, hash: lbHash})
		}
		if blockNumber != math.MaxUint64 {
			blockPtr, err := reader.getBlockLvPointer(blockNumber)
			if err != nil {
				t.Fatalf("Test case %s: error reading lv pointer of block %d: %v", testCase, blockNumber, err)
			} else if blockPtr != expBlockPtr {
				t.Fatalf("Test case %s: incorrect results when reading lv pointer of block %d (expected %v, got %v)", testCase, blockNumber, expBlockPtr, blockPtr)
			}
		}
	}
	testNoPointer := func(mapIndex uint32) {
		if _, _, err := reader.getLastBlockOfMap(mapIndex); err == nil {
			t.Fatalf("Test case %s: unexpected last block of map pointer at map %d", testCase, mapIndex)
		}
	}
	testKnownEpoch := func(epoch uint32) {
		testNoPointer(params.firstEpochMap(epoch))
		testPointers(params.lastEpochMap(epoch), lastBlockOfMap{number: knownEpochs[epoch].BlockNumber, hash: knownEpochs[epoch].BlockHash}, knownEpochs[epoch].BlockNumber, knownEpochs[epoch].FirstIndex)
	}
	if len(knownEpochs) > 0 {
		for range testPointersCount {
			epoch := uint32(rand.Intn(len(knownEpochs)))
			testKnownEpoch(epoch)
		}
		testKnownEpoch(uint32(len(knownEpochs) - 1))
	}
	testMapPointers := func(mapIndex uint32, blockPos int) { // 0: first 1: random 2: last
		fm := maps[mapIndex-mapRange.First()]
		if len(fm.blockPtrs) == 0 {
			testPointers(mapIndex, fm.lastBlock, math.MaxUint64, 0)
		} else {
			var subIndex int
			switch blockPos {
			case 1:
				subIndex = rand.Intn(len(fm.blockPtrs))
			case 2:
				subIndex = len(fm.blockPtrs) - 1
			}
			testPointers(mapIndex, fm.lastBlock, fm.firstBlock()+uint64(subIndex), fm.blockPtrs[subIndex])
		}
	}
	if !mapRange.IsEmpty() {
		testMapPointers(mapRange.First(), 0)
		for range testPointersCount {
			testMapPointers(mapRange.First()+uint32(rand.Intn(int(mapRange.Count()))), 1)
		}
		testMapPointers(mapRange.Last(), 2)
	}
	testNoPointer(mapRange.AfterLast())
	testNoPointer(params.lastEpochMap(params.mapEpoch(mapRange.AfterLast())))
	testNoPointer(params.firstEpochMap(params.mapEpoch(mapRange.AfterLast()) + 1))
}

func generateTestMaps(params *Params, maps []*finishedMap, amount uint32) []*finishedMap {
	var lastBlock lastBlockOfMap
	genesis := len(maps) == 0
	if !genesis {
		lastBlock = maps[len(maps)-1].lastBlock
	}
	for range amount {
		blockCount := rand.Intn(testMaxBlocksPerMap + 1)
		if genesis && blockCount == 0 {
			blockCount = 1
		}
		fm := &finishedMap{
			rowPtrs:   make([]uint16, params.mapHeight),
			rowData:   make([]uint32, params.valuesPerMap),
			blockPtrs: make([]uint64, blockCount),
		}
		addValues := params.valuesPerMap
		for addValues > 0 {
			addToRow := uint64(rand.Intn(int(min(addValues+1, params.valuesPerMap/2))))
			addValues -= addToRow
			var rowIndex uint32
			for {
				rowIndex = uint32(rand.Intn(int(params.mapHeight)))
				if uint64(fm.rowPtrs[rowIndex])+addToRow <= params.valuesPerMap/2 {
					break
				}
			}
			fm.rowPtrs[rowIndex] += uint16(addToRow)
		}
		var ptr uint16
		for i, r := range fm.rowPtrs {
			ptr += r
			fm.rowPtrs[i] = ptr
		}
		for i := range fm.rowData {
			fm.rowData[i] = uint32(rand.Intn(1 << params.logMapWidth))
		}
		if blockCount > 0 {
			startPtr := uint64(len(maps)) * params.valuesPerMap
			for i := range fm.blockPtrs {
				if genesis {
					genesis = false
				} else {
					lastBlock.number++
					fm.blockPtrs[i] = startPtr + uint64(i)*params.valuesPerMap/uint64(blockCount) + uint64(rand.Intn(int(params.valuesPerMap)/blockCount/2))
				}
			}
			rand.Read(lastBlock.hash[:])
		}
		fm.lastBlock = lastBlock
		maps = append(maps, fm)
	}
	return maps
}

func generateEpochCheckpoint(mapIndex uint32, maps []*finishedMap) epochCheckpoint {
	for len(maps[mapIndex].blockPtrs) == 0 {
		mapIndex--
	}
	return epochCheckpoint{
		BlockNumber: maps[mapIndex].lastBlock.number,
		BlockHash:   maps[mapIndex].lastBlock.hash,
		FirstIndex:  maps[mapIndex].blockPtrs[len(maps[mapIndex].blockPtrs)-1],
	}
}

func generateTestCheckpoints(params *Params, maps []*finishedMap) checkpointList {
	cpList := make(checkpointList, uint32(len(maps))/params.mapsPerEpoch)
	for i := range cpList {
		cpList[i] = generateEpochCheckpoint(params.lastEpochMap(uint32(i)), maps)
	}
	return cpList
}
