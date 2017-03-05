// Copyright 2017 The go-ethereum Authors
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

// Package core implements the Ethereum consensus protocol.
package core

import (
	"encoding/binary"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
)

func TestChainIndexerSingle(t *testing.T) {
	// run multiple tests with randomized parameters
	for i := 0; i < 10; i++ {
		testChainIndexer(t, 1)
	}
}

func TestChainIndexerWithChildren(t *testing.T) {
	// run multiple tests with randomized parameters and different number of
	// chained indexers
	for i := 2; i < 8; i++ {
		testChainIndexer(t, i)
	}
}

// testChainIndexer runs a test with either a single ChainIndexer or a chain of multiple indexers
// sectionSize and confirmReq parameters are randomized
func testChainIndexer(t *testing.T, tciCount int) {
	db, _ := ethdb.NewMemDatabase()
	stop := make(chan struct{})
	tciList := make([]*testChainIndex, tciCount)
	var lastIndexer *ChainIndexer
	for i, _ := range tciList {
		tci := &testChainIndex{t: t, sectionSize: uint64(rand.Intn(100) + 1), confirmReq: uint64(rand.Intn(10)), processCh: make(chan uint64)}
		tciList[i] = tci
		tci.indexer = NewChainIndexer(db, ethdb.NewTable(db, string([]byte{byte(i)})), tci, tci.sectionSize, tci.confirmReq, 0, stop)
		if cs := tci.indexer.CanonicalSections(); cs != 0 {
			t.Errorf("Expected 0 canonical sections, got %d", cs)
		}
		if lastIndexer != nil {
			lastIndexer.AddChildIndexer(tci.indexer)
		}
		lastIndexer = tci.indexer
	}

	// expectCs expects a certain number of available canonical sections
	expectCs := func(indexer *ChainIndexer, expCs uint64) {
		cnt := 0
		for {
			cs := indexer.CanonicalSections()
			if cs == expCs {
				return
			}
			// keep trying for 10 seconds if it does not match
			cnt++
			if cnt == 10000 {
				t.Fatalf("Expected %d canonical sections, got %d", expCs, cs)
			}
			time.Sleep(time.Millisecond)
		}
	}

	// notify the indexer about a new head or rollback, then expect processed blocks if a section is processable
	notify := func(headNum, expFailAfter uint64, rollback bool) {
		tciList[0].indexer.newHead(headNum, rollback)
		if rollback {
			for _, tci := range tciList {
				headNum = tci.rollback(headNum)
				expectCs(tci.indexer, tci.stored)
			}
		} else {
			for _, tci := range tciList {
				var more bool
				headNum, more = tci.newBlocks(headNum, expFailAfter)
				if !more {
					break
				}
				expectCs(tci.indexer, tci.stored)
			}
		}
	}

	for i := uint64(0); i <= 100; i++ {
		testCanonicalHeader(db, i)
	}
	// start indexer with an already existing chain
	notify(100, 100, false)
	// add new blocks one by one
	for i := uint64(101); i <= 1000; i++ {
		testCanonicalHeader(db, i)
		notify(i, i, false)
	}
	// do a rollback
	notify(500, 500, true)
	// create new fork
	for i := uint64(501); i <= 1000; i++ {
		testCanonicalHeader(db, i)
		notify(i, i, false)
	}

	for i := uint64(1001); i <= 1500; i++ {
		testCanonicalHeader(db, i)
	}
	// create a failed processing scenario where less blocks are available at processing time than notified
	notify(2000, 1500, false)
	// notify about a rollback (which could have caused the missing blocks if happened during processing)
	notify(1500, 1500, true)

	// create new fork
	for i := uint64(1501); i <= 2000; i++ {
		testCanonicalHeader(db, i)
		notify(i, i, false)
	}
	close(stop)
	db.Close()
}

func testCanonicalHeader(db ethdb.Database, idx uint64) {
	var rnd [8]byte
	binary.BigEndian.PutUint64(rnd[:], uint64(rand.Int63()))
	header := &types.Header{Number: big.NewInt(int64(idx)), Extra: rnd[:]}
	if idx > 0 {
		header.ParentHash = GetCanonicalHash(db, idx-1)
	}
	WriteHeader(db, header)
	WriteCanonicalHash(db, header.Hash(), idx)
}

// testChainIndex implements ChainIndexerBackend
type testChainIndex struct {
	t                          *testing.T
	sectionSize, confirmReq    uint64
	section, headerCnt, stored uint64
	indexer                    *ChainIndexer
	processCh                  chan uint64
}

// newBlocks expects process calls after new blocks have arrived. If expFailAfter < headNum then
// we are simulating a scenario where a rollback has happened after the processing has started and
// the processing of a section fails.
func (t *testChainIndex) newBlocks(headNum, expFailAfter uint64) (uint64, bool) {
	var newCount uint64
	if headNum >= t.confirmReq {
		newCount = (headNum + 1 - t.confirmReq) / t.sectionSize
		if newCount > t.stored {
			// expect processed blocks
			for exp := t.stored * t.sectionSize; exp < newCount*t.sectionSize; exp++ {
				if exp > expFailAfter {
					// rolled back after processing started, no more process calls expected
					// wait until updating is done to make sure that processing actually fails
					for {
						t.indexer.lock.Lock()
						u := t.indexer.updating
						t.indexer.lock.Unlock()
						if !u {
							break
						}
						time.Sleep(time.Millisecond)
					}

					newCount = exp / t.sectionSize
					break
				}
				select {
				case <-time.After(10 * time.Second):
					t.t.Fatalf("Expected processed block #%d, got nothing", exp)
				case proc := <-t.processCh:
					if proc != exp {
						t.t.Errorf("Expected processed block #%d, got #%d", exp, proc)
					}
				}
			}
			t.stored = newCount
		}
	}
	if t.stored == 0 {
		return 0, false
	}
	return t.stored*t.sectionSize - 1, true
}

func (t *testChainIndex) rollback(headNum uint64) uint64 {
	firstChanged := headNum / t.sectionSize
	if firstChanged < t.stored {
		t.stored = firstChanged
	}
	return t.stored * t.sectionSize
}

func (t *testChainIndex) Reset(section uint64) {
	t.section = section
	t.headerCnt = 0
}

func (t *testChainIndex) Process(header *types.Header) {
	t.headerCnt++
	if t.headerCnt > t.sectionSize {
		t.t.Error("Processing too many headers")
	}
	//t.processCh <- header.Number.Uint64()
	select {
	case <-time.After(10 * time.Second):
		t.t.Fatal("Unexpected call to Process")
	case t.processCh <- header.Number.Uint64():
	}
}

func (t *testChainIndex) Commit(db ethdb.Database) error {
	if t.headerCnt != t.sectionSize {
		t.t.Error("Not enough headers processed")
	}
	return nil
}

func (t *testChainIndex) UpdateMsg(done, all uint64) {}
