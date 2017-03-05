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
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
)

// ChainIndexer does a post-processing job for equally sized sections of the canonical
// chain (like BlooomBits and CHT structures). A ChainIndexer is connected to the blockchain
// through the event system by starting a ChainEventLoop in a goroutine.
// Further child ChainIndexers can be added which use the output of the parent section
// indexer. These child indexers receive new head notifications only after an entire section
// has been finished or in case of rollbacks that might affect already finished sections.
type ChainIndexer struct {
	chainDb, indexDb                            ethdb.Database
	backend                                     ChainIndexerBackend
	sectionSize, confirmReq                     uint64
	stop                                        chan struct{}
	lock                                        sync.Mutex
	procWait                                    time.Duration
	tryUpdate                                   chan struct{}
	stored, targetCount, calcIdx, lastForwarded uint64
	updating                                    bool
	children                                    []*ChainIndexer
}

// ChainIndexerBackend interface is a backend for the indexer doing the actual post-processing job
type ChainIndexerBackend interface {
	Reset(section uint64)           // start processing a new section
	Process(header *types.Header)   // process a single block (called for each block in the section)
	Commit(db ethdb.Database) error // do some more processing if necessary and store the results in the database
	UpdateMsg(done, all uint64)     // print a progress update message if necessary (only called when multiple sections need to be processed)
}

// NewChainIndexer creates a new  ChainIndexer
//  db:				database where the index of available processed sections is stored (the index is stored by the
//                  indexer, the actual processed chain data is stored by the backend)
//  dbKey:			key prefix where the index is stored
//  backend:		an implementation of ChainIndexerBackend
//  sectionSize:	the size of processable sections
//  confirmReq:		required number of confirmation blocks before a new section is being processed
//  procWait:		waiting time between processing sections (simple way of limiting the resource usage of a db upgrade)
//  stop:		    quit channel
func NewChainIndexer(chainDb, indexDb ethdb.Database, backend ChainIndexerBackend, sectionSize, confirmReq uint64, procWait time.Duration, stop chan struct{}) *ChainIndexer {
	c := &ChainIndexer{
		chainDb:     chainDb,
		indexDb:     indexDb,
		backend:     backend,
		sectionSize: sectionSize,
		confirmReq:  confirmReq,
		tryUpdate:   make(chan struct{}, 1),
		stop:        stop,
		procWait:    procWait,
	}
	c.stored = c.getValidSections()
	go c.updateLoop()
	return c
}

// updateLoop is the main event loop of the indexer
func (c *ChainIndexer) updateLoop() {
	updateMsg := false

	for {
		select {
		case <-c.stop:
			return
		case <-c.tryUpdate:
			c.lock.Lock()
			if c.targetCount > c.stored {
				if !updateMsg && c.targetCount > c.stored+1 {
					updateMsg = true
					c.backend.UpdateMsg(c.stored, c.targetCount)
				}
				c.calcIdx = c.stored

				var lastSectionHead common.Hash
				if c.calcIdx > 0 {
					lastSectionHead = c.getSectionHead(c.calcIdx - 1)
				}

				c.lock.Unlock()
				sectionHead, ok := c.processSection(c.calcIdx, lastSectionHead)
				c.lock.Lock()

				if ok && lastSectionHead == c.getSectionHead(c.calcIdx-1) {
					c.stored = c.calcIdx + 1
					c.setSectionHead(c.calcIdx, sectionHead)
					c.setValidSections(c.stored)
					if updateMsg {
						c.backend.UpdateMsg(c.stored, c.targetCount)
						if c.stored >= c.targetCount {
							updateMsg = false
						}
					}
					c.lastForwarded = c.stored*c.sectionSize - 1
					for _, cp := range c.children {
						cp.newHead(c.lastForwarded, false)
					}
				} else {
					// if processing has failed, do not retry until further notification
					c.targetCount = c.stored
				}
			}

			if c.targetCount > c.stored {
				go func() {
					time.Sleep(c.procWait)
					c.tryUpdate <- struct{}{}
				}()
			} else {
				c.updating = false
			}
			c.lock.Unlock()
		}
	}
}

// ChainEventLoop runs in a goroutine and feeds blockchain events to the indexer by calling newHead
// (not needed for child indexers where the parent calls newHead)
func (c *ChainIndexer) ChainEventLoop(currentHeader *types.Header, eventMux *event.TypeMux) {
	sub := eventMux.Subscribe(ChainEvent{})
	c.newHead(currentHeader.Number.Uint64(), false)
	lastHead := currentHeader.Hash()
	for {
		select {
		case <-c.stop:
			return
		case ev := <-sub.Chan():
			header := ev.Data.(ChainEvent).Block.Header()
			c.newHead(header.Number.Uint64(), header.ParentHash != lastHead)
			lastHead = header.Hash()
		}
	}
}

// AddChildIndexer adds a child ChainIndexer that can use the output of this one
func (c *ChainIndexer) AddChildIndexer(ci *ChainIndexer) {
	c.children = append(c.children, ci)
}

// newHead notifies the indexer about new chain heads or rollbacks
func (c *ChainIndexer) newHead(headNum uint64, rollback bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if rollback {
		firstChanged := headNum / c.sectionSize
		if firstChanged < c.targetCount {
			c.targetCount = firstChanged
		}
		if firstChanged < c.stored {
			c.stored = firstChanged
			c.setValidSections(c.stored)
		}
		headNum = firstChanged * c.sectionSize

		if headNum < c.lastForwarded {
			c.lastForwarded = headNum
			for _, cp := range c.children {
				cp.newHead(c.lastForwarded, true)
			}
		}

	} else {
		var newCount uint64
		if headNum >= c.confirmReq {
			newCount = (headNum + 1 - c.confirmReq) / c.sectionSize
			if newCount > c.targetCount {
				c.targetCount = newCount
				if !c.updating {
					c.updating = true
					c.tryUpdate <- struct{}{}
				}
			}
		}
	}
}

// processSection processes an entire section by calling backend functions while ensuring
// the continuity of the passed headers. Since the chain mutex is not held while processing,
// the continuity can be broken by a long reorg, in which case the function returns with ok == false.
func (c *ChainIndexer) processSection(section uint64, lastSectionHead common.Hash) (sectionHead common.Hash, ok bool) {
	c.backend.Reset(section)

	head := lastSectionHead
	for i := section * c.sectionSize; i < (section+1)*c.sectionSize; i++ {
		hash := GetCanonicalHash(c.chainDb, i)
		if hash == (common.Hash{}) {
			return common.Hash{}, false
		}
		header := GetHeader(c.chainDb, hash, i)
		if header == nil || header.ParentHash != head {
			return common.Hash{}, false
		}
		c.backend.Process(header)
		head = header.Hash()
	}
	if err := c.backend.Commit(c.chainDb); err != nil {
		return common.Hash{}, false
	}
	return head, true
}

// CanonicalSections returns the number of processed sections that are consistent with
// the current canonical chain
func (c *ChainIndexer) CanonicalSections() uint64 {
	c.lock.Lock()
	defer c.lock.Unlock()

	cnt := c.getValidSections()
	for cnt > 0 {
		if c.getSectionHead(cnt-1) == GetCanonicalHash(c.chainDb, cnt*c.sectionSize-1) {
			break
		}
		cnt--
		c.setValidSections(cnt)
	}
	return cnt
}

// getValidSections reads the number of valid sections from the index database
func (c *ChainIndexer) getValidSections() uint64 {
	data, _ := c.indexDb.Get([]byte("count"))
	if len(data) == 8 {
		return binary.BigEndian.Uint64(data[:])
	}
	return 0
}

// setValidSections writes the number of valid sections to the index database
func (c *ChainIndexer) setValidSections(cnt uint64) {
	oldCnt := c.getValidSections()
	if cnt < oldCnt {
		for i := cnt; i < oldCnt; i++ {
			c.removeSectionHead(i)
		}
	}

	var data [8]byte
	binary.BigEndian.PutUint64(data[:], cnt)
	c.indexDb.Put([]byte("count"), data[:])
}

// getSectionHead reads the last block hash of a processed section from the index database
func (c *ChainIndexer) getSectionHead(idx uint64) common.Hash {
	var data [8]byte
	binary.BigEndian.PutUint64(data[:], idx)

	hash, _ := c.indexDb.Get(append([]byte("shead"), data[:]...))
	if len(hash) == len(common.Hash{}) {
		return common.BytesToHash(hash)
	}
	return common.Hash{}
}

// setSectionHead writes the last block hash of a processed section to the index database
func (c *ChainIndexer) setSectionHead(idx uint64, shead common.Hash) {
	var data [8]byte
	binary.BigEndian.PutUint64(data[:], idx)

	c.indexDb.Put(append([]byte("shead"), data[:]...), shead.Bytes())
}

// removeSectionHead removes the reference to a processed section from the index database
func (c *ChainIndexer) removeSectionHead(idx uint64) {
	var data [8]byte
	binary.BigEndian.PutUint64(data[:], idx)

	c.indexDb.Delete(append([]byte("shead"), data[:]...))
}
