// Copyright 2015 The go-ethereum Authors
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

// Package les implements the Light Ethereum Subprotocol.
package les

import (
"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
)
		
type lightFetcher struct{
	pm *ProtocolManager
	odr *LesOdr
	chain BlockChain
	reqMu sync.RWMutex
	requested map[uint64]chan *types.Header
	syncPoolMu sync.Mutex
	syncPool map[*peer]struct{}
	syncPoolNotify chan struct{}
	syncPoolNotified uint32
}	
	
func newLightFetcher(pm *ProtocolManager) *lightFetcher {
	f := &lightFetcher{
		pm: pm,
		chain: pm.blockchain,
		odr: pm.odr,
		requested: make(map[uint64]chan *types.Header),
		syncPool: make(map[*peer]struct{}),
		syncPoolNotify: make(chan struct{}),
	}
	go f.syncLoop()
	return f
}

func (f *lightFetcher) requestedID(reqID uint64) bool {
	f.reqMu.RLock()
	_, ok := f.requested[reqID]
	f.reqMu.RUnlock()
	return ok
}

func (f *lightFetcher) deliverHeaders(reqID uint64, headers []*types.Header) {
	f.reqMu.Lock()
	chn := f.requested[reqID]	
	if len(headers) == 1 {
		chn <- headers[0]
	} else {
		chn <- nil
	}
	close(chn)
	delete(f.requested, reqID)
	f.reqMu.Unlock()
}

func (f *lightFetcher) notify(p *peer, block blockInfo) {
	p.lock.Lock()
	if block.Td.Cmp(p.headInfo.Td) <= 0 {
		p.lock.Unlock()
		return
	}
	p.headInfo = block
	p.lock.Unlock()

	head := f.pm.blockchain.CurrentHeader()
	currentTd := core.GetTd(f.pm.chainDb, head.Hash(), head.Number.Uint64())
	if block.Td.Cmp(currentTd) > 0 {
		f.syncPoolMu.Lock()
		f.syncPool[p] = struct{}{}
		f.syncPoolMu.Unlock()
		if atomic.SwapUint32(&f.syncPoolNotified, 1) == 0 {
			f.syncPoolNotify <- struct{}{}
		}
	}
}

func (f *lightFetcher) fetchBestFromPool() *peer {
	head := f.pm.blockchain.CurrentHeader()
	currentTd := core.GetTd(f.pm.chainDb, head.Hash(), head.Number.Uint64())

	f.syncPoolMu.Lock()
	var best *peer
	for p, _ := range f.syncPool {
		td := p.Td()
		if td.Cmp(currentTd) <= 0 {
			delete(f.syncPool, p)
		} else {
			if best == nil || td.Cmp(best.Td()) > 0 {
				best = p
			}
		}
	}
	if best != nil {
		delete(f.syncPool, best)
	}
	f.syncPoolMu.Unlock()
	return best	
}

func (f *lightFetcher) syncLoop() {
	f.pm.wg.Add(1)
	defer f.pm.wg.Done()
	
	for {
		select {	
		case <-f.pm.quitSync:
			return
		case <-f.syncPoolNotify:
			atomic.StoreUint32(&f.syncPoolNotified, 0)
			chn := f.pm.getSyncLock(false)
			if chn != nil {
				if atomic.SwapUint32(&f.syncPoolNotified, 1) == 0 {
					go func() {
						<-chn
						f.syncPoolNotify <- struct{}{}
					}()
				}
			} else {
				if p := f.fetchBestFromPool(); p != nil {
					go f.syncWithPeer(p)
					if atomic.SwapUint32(&f.syncPoolNotified, 1) == 0 {
						go func() {
							time.Sleep(softRequestTimeout)
							f.syncPoolNotify <- struct{}{}
						}()
					}
				}
			}
		}
	}
}

func (f *lightFetcher) syncWithPeer(p *peer) bool {
	f.pm.wg.Add(1)
	defer f.pm.wg.Done()
	
	headNum := f.chain.CurrentHeader().Number.Uint64()
	peerHead := p.headBlockInfo()

	if !f.pm.needToSync(peerHead) {
		return true
	}

	if peerHead.Number <= headNum+1 {
		var header *types.Header
		reqID, chn := f.request(p, peerHead)
		select {
		case header = <-chn:
			if header == nil || header.Hash() != peerHead.Hash ||
			   header.Number.Uint64() != peerHead.Number {
				// missing or wrong header returned
fmt.Println("removePeer 1")
				f.pm.removePeer(p.id)
				return false
			}
			
		case <-time.After(hardRequestTimeout):
			if !disableClientRemovePeer {
fmt.Println("removePeer 2")
				f.pm.removePeer(p.id)
			}
			f.reqMu.Lock()
			close(f.requested[reqID])
			delete(f.requested, reqID)
			f.reqMu.Unlock()
			return false
		}
		
		// got the header, try to insert
		f.chain.InsertHeaderChain([]*types.Header{header}, 1)

		defer func() {
			// check header td at the end of syncing, drop peer if it was fake
			headerTd := core.GetTd(f.pm.chainDb, header.Hash(), header.Number.Uint64())
			if headerTd != nil && headerTd.Cmp(peerHead.Td) != 0 {
fmt.Println("removePeer 3")
				f.pm.removePeer(p.id)
			}
		}()
		if !f.pm.needToSync(peerHead) {
			return true
		}
	}

	f.pm.waitSyncLock()
	if !f.pm.needToSync(peerHead) {
		// synced up by the one we've been waiting to end
		f.pm.releaseSyncLock()
		return true
	}
	f.pm.syncWithLockAcquired(p)
	return !f.pm.needToSync(peerHead)
}

func (f *lightFetcher) request(p *peer, block blockInfo) (uint64, chan *types.Header) {
	reqID := f.odr.getNextReqID()
	f.reqMu.Lock()
	chn := make(chan *types.Header, 1)
	f.requested[reqID] = chn
	f.reqMu.Unlock()
	cost := p.GetRequestCost(GetBlockHeadersMsg, 1)
	p.fcServer.SendRequest(reqID, cost)
	p.RequestHeadersByHash(reqID, cost, block.Hash, 1, 0, false)
	return reqID, chn
}
