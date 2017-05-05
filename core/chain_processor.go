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
	"sync"
	"time"
)

// ChainProcessor is an external process that creates auxiliary data structures
// using the canonical chain as an input. Its callback function NewHead is called every
// time a new head is added to the chain or it gets rolled back. The callback function
// should never block because it is called under the chain's mutex lock so that the
// structures can stay consistent with the canonical chain.
// The same interface can be used for chaining processors (one using the output of the
// other).
type ChainProcessor interface {
	NewHead(headNum uint64, rollBack bool)
}

// ChainSectionProcessor is a ChainProcessor that does a post-processing job for
// equally sized sections of the canonical chain (like BlooomBits and CHT structures).
// Further child ChainProcessors can be added which use the output of this section
// processor. These child processors receive NewHead calls only after an entire section
// has been finished or in case of rollbacks that might affect already finished sections.
type ChainSectionProcessor struct {
	backend                        ChainSectionProcessorBackend
	sectionSize, confirmReq        uint64
	stop                           chan struct{}
	updateChn                      chan uint64
	lock                           sync.Mutex
	calcValid                      bool
	procWait                       time.Duration
	stored, calcIdx, lastForwarded uint64
	childProcessors                []ChainProcessor
}

// ChainSectionProcessorBackend does the actual post-processing job.
// GetStored and SetStored are called under the canonical chain lock and should
// not block. Process is called without a chain lock and may take longer to
// finish. If the chain gets rolled back while Process is running, the results
// are considered invalid and SetStored is not called.
type ChainSectionProcessorBackend interface {
	Process(idx uint64) bool
	GetStored() uint64
	SetStored(count uint64)
	UpdateMsg(done, all uint64)
}

// NewChainSectionProcessor creates a new  ChainSectionProcessor
func NewChainSectionProcessor(backend ChainSectionProcessorBackend, sectionSize, confirmReq uint64, procWait time.Duration, stop chan struct{}) *ChainSectionProcessor {
	csp := &ChainSectionProcessor{
		backend:     backend,
		sectionSize: sectionSize,
		confirmReq:  confirmReq,
		stop:        stop,
		procWait:    procWait,
		updateChn:   make(chan uint64, 100),
		stored:      backend.GetStored(),
	}
	go csp.updateLoop()
	return csp
}

func (csp *ChainSectionProcessor) updateLoop() {
	tryUpdate := make(chan struct{}, 1)
	updating := false
	var targetCnt uint64
	updateMsg := false

	for {
		select {
		case <-csp.stop:
			return
		case targetCnt = <-csp.updateChn:
			if !updating {
				updating = true
				tryUpdate <- struct{}{}
			}
		case <-tryUpdate:
			csp.lock.Lock()
			if targetCnt > csp.stored {
				if !updateMsg && targetCnt > csp.stored+1 {
					updateMsg = true
					csp.backend.UpdateMsg(csp.stored, targetCnt)
				}
				csp.calcValid = true
				csp.calcIdx = csp.stored

				csp.lock.Unlock()
				ok := csp.backend.Process(csp.calcIdx)
				csp.lock.Lock()

				if ok && csp.calcValid {
					csp.stored = csp.calcIdx + 1
					csp.backend.SetStored(csp.stored)
					if updateMsg {
						csp.backend.UpdateMsg(csp.stored, targetCnt)
						if csp.stored >= targetCnt {
							updateMsg = false
						}
					}
					csp.lastForwarded = csp.stored*csp.sectionSize - 1
					for _, cp := range csp.childProcessors {
						cp.NewHead(csp.lastForwarded, false)
					}
				}
				csp.calcValid = false
			}
			stored := csp.stored
			csp.lock.Unlock()

			if targetCnt > stored {
				go func() {
					time.Sleep(csp.procWait)
					tryUpdate <- struct{}{}
				}()
			} else {
				updating = false
			}
		}
	}
}

// NewHead implements the ChainProcessor interface
func (csp *ChainSectionProcessor) NewHead(headNum uint64, rollback bool) {
	csp.lock.Lock()
	defer csp.lock.Unlock()

	if rollback {
		firstChanged := headNum / csp.sectionSize
		if firstChanged <= csp.calcIdx {
			csp.calcValid = false
		}
		if firstChanged < csp.stored {
			csp.stored = firstChanged
			csp.backend.SetStored(csp.stored)
			select {
			case <-csp.stop:
			case csp.updateChn <- firstChanged:
			}
		}

		if headNum < csp.lastForwarded {
			csp.lastForwarded = headNum
			for _, cp := range csp.childProcessors {
				cp.NewHead(csp.lastForwarded, true)
			}
		}

	} else {
		var newCount uint64
		if headNum >= csp.confirmReq {
			newCount = (headNum + 1 - csp.confirmReq) / csp.sectionSize
			if newCount > csp.stored {
				go func() {
					select {
					case <-csp.stop:
					case csp.updateChn <- newCount:
					}
				}()
			}
		}
	}
}

// AddChildProcessor adds a child ChainProcessor that can use the output of this
// section processor.
func (csp *ChainSectionProcessor) AddChildProcessor(cp ChainProcessor) {
	csp.childProcessors = append(csp.childProcessors, cp)
}
