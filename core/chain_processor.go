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
)

// ChainProcessor is an external process that creates auxiliary data structures
// using the canonical chain as an input. Its callback function NewHead is called every
// time a new head is added to the chain or it gets rolled back. The callback function
// should never block because it is called under the chain's mutex lock so that the
// structures can stay consistent with the canonical chain. It should also not call
// any chain functions that use this mutex because that would cause a deadlock.
// The same interface can be used for chaining processors (one using the output of the
// other).
type ChainProcessor interface {
	NewHead(headNum uint64, rollBack bool)
}

// ChainIndexer is a ChainProcessor that does a post-processing job for
// equally sized sections of the canonical chain (like BlooomBits and CHT structures).
// Further child ChainProcessors can be added which use the output of this section
// processor. These child processors receive NewHead calls only after an entire section
// has been finished or in case of rollbacks that might affect already finished sections.
type ChainIndexer struct {
	db                             ethdb.Database
	validSectionsKey               []byte
	backend                        ChainIndex
	sectionSize, confirmReq        uint64
	stop                           chan struct{}
	updateCh                       chan uint64
	lock                           sync.Mutex
	calcValid                      bool
	procWait                       time.Duration
	stored, calcIdx, lastForwarded uint64
	childProcessors                []ChainProcessor
}

type ChainIndex interface {
	Reset(section uint64)
	Process(header *types.Header)
	Commit(db ethdb.Database) error
	UpdateMsg(done, all uint64) // print a progress update message if necessary (only called when multiple sections need to be processed)
}

// NewChainIndexer creates a new  ChainIndexer
//  backend:		an implementation of ChainIndex
//  sectionSize:	the size of processable sections
//  confirmReq:		required number of confirmation blocks before a new section is being processed
//  procWait:		waiting time between processing sections (simple way of limiting the resource usage of a db upgrade)
//  stop:		    quit channel
func NewChainIndexer(db ethdb.Database, validSectionsKey []byte, backend ChainIndex, sectionSize, confirmReq uint64, procWait time.Duration, stop chan struct{}) *ChainIndexer {
	c := &ChainIndexer{
		db:               db,
		validSectionsKey: validSectionsKey,
		backend:          backend,
		sectionSize:      sectionSize,
		confirmReq:       confirmReq,
		stop:             stop,
		procWait:         procWait,
		updateCh:         make(chan uint64, 100),
	}
	c.stored = c.ValidSections()
	go c.updateLoop()
	return c
}

func (c *ChainIndexer) updateLoop() {
	tryUpdate := make(chan struct{}, 1)
	updating := false
	var targetCount uint64
	updateMsg := false

	for {
		select {
		case <-c.stop:
			return
		case targetCount = <-c.updateCh:
			if !updating {
				updating = true
				tryUpdate <- struct{}{}
			}
		case <-tryUpdate:
			c.lock.Lock()
			if targetCount > c.stored {
				if !updateMsg && targetCount > c.stored+1 {
					updateMsg = true
					c.backend.UpdateMsg(c.stored, targetCount)
				}
				c.calcValid = true
				c.calcIdx = c.stored

				c.lock.Unlock()
				ok := c.processSection(c.calcIdx)
				c.lock.Lock()

				if ok && c.calcValid {
					c.stored = c.calcIdx + 1
					c.setValidSections(c.stored)
					if updateMsg {
						c.backend.UpdateMsg(c.stored, targetCount)
						if c.stored >= targetCount {
							updateMsg = false
						}
					}
					c.lastForwarded = c.stored*c.sectionSize - 1
					for _, cp := range c.childProcessors {
						cp.NewHead(c.lastForwarded, false)
					}
				}
				c.calcValid = false
			}
			stored := c.stored
			c.lock.Unlock()

			if targetCount > stored {
				go func() {
					time.Sleep(c.procWait)
					tryUpdate <- struct{}{}
				}()
			} else {
				updating = false
			}
		}
	}
}

func (c *ChainIndexer) processSection(section uint64) bool {
	c.backend.Reset(section)
	for i := section * c.sectionSize; i < (section+1)*c.sectionSize; i++ {
		hash := GetCanonicalHash(c.db, i)
		if hash == (common.Hash{}) {
			return false
		}
		header := GetHeader(c.db, hash, i)
		if header == nil {
			return false
		}
		c.backend.Process(header)
	}
	return c.backend.Commit(c.db) == nil
}

func (c *ChainIndexer) ValidSections() uint64 {
	data, _ := c.db.Get(c.validSectionsKey)
	if len(data) == 8 {
		return binary.BigEndian.Uint64(data[:])
	}
	return 0
}

func (c *ChainIndexer) setValidSections(cnt uint64) {
	var data [8]byte
	binary.BigEndian.PutUint64(data[:], cnt)
	c.db.Put(c.validSectionsKey, data[:])
}

// NewHead implements the ChainProcessor interface
func (c *ChainIndexer) NewHead(headNum uint64, rollback bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if rollback {
		firstChanged := headNum / c.sectionSize
		if firstChanged <= c.calcIdx {
			c.calcValid = false
		}
		if firstChanged < c.stored {
			c.stored = firstChanged
			c.setValidSections(c.stored)
			select {
			case <-c.stop:
			case c.updateCh <- firstChanged:
			}
		}

		if headNum < c.lastForwarded {
			c.lastForwarded = headNum
			for _, cp := range c.childProcessors {
				cp.NewHead(c.lastForwarded, true)
			}
		}

	} else {
		var newCount uint64
		if headNum >= c.confirmReq {
			newCount = (headNum + 1 - c.confirmReq) / c.sectionSize
			if newCount > c.stored {
				go func() {
					select {
					case <-c.stop:
					case c.updateCh <- newCount:
					}
				}()
			}
		}
	}
}

// AddChildProcessor adds a child ChainProcessor that can use the output of this
// section processor.
func (c *ChainIndexer) AddChildProcessor(cp ChainProcessor) {
	c.childProcessors = append(c.childProcessors, cp)
}
