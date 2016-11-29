// Copyright 2016 The go-ethereum Authors
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
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
)

type lightFetcher struct {
	pm    *ProtocolManager
	odr   *LesOdr
	chain BlockChain

	headAnnouncedMu sync.Mutex
	headAnnouncedBy map[common.Hash][]*peer
	currentTd       *big.Int
	deliverChn      chan fetchResponse
	reqMu           sync.RWMutex
	requested       map[uint64]fetchRequest
	timeoutChn      chan uint64
	notifyChn       chan bool // true if initiated from outside
	syncing         bool
	syncDone        chan struct{}
}

type fetchRequest struct {
	hash   common.Hash
	amount uint64
	peer   *peer
}

type fetchResponse struct {
	reqID   uint64
	headers []*types.Header
	peer    *peer
}

func newLightFetcher(pm *ProtocolManager) *lightFetcher {
	f := &lightFetcher{
		pm:              pm,
		chain:           pm.blockchain,
		odr:             pm.odr,
		headAnnouncedBy: make(map[common.Hash][]*peer),
		deliverChn:      make(chan fetchResponse, 100),
		requested:       make(map[uint64]fetchRequest),
		timeoutChn:      make(chan uint64),
		notifyChn:       make(chan bool, 100),
		syncDone:        make(chan struct{}),
		currentTd:       big.NewInt(0),
	}
	go f.syncLoop()
	return f
}

func (f *lightFetcher) notify(p *peer, head *announceData) {
	var headHash common.Hash
	if head == nil {
		// initial notify
		headHash = p.Head()
	} else {
		if core.GetTd(f.pm.chainDb, head.Hash, head.Number) != nil {
			head.haveHeaders = head.Number
		}
		//fmt.Println("notify", p.id, head.Number, head.ReorgDepth, head.haveHeaders)
		if !p.addNotify(head) {
			//fmt.Println("addNotify fail")
			f.pm.removePeer(p.id)
		}
		headHash = head.Hash
	}
	f.headAnnouncedMu.Lock()
	f.headAnnouncedBy[headHash] = append(f.headAnnouncedBy[headHash], p)
	f.headAnnouncedMu.Unlock()
	f.notifyChn <- true
}

func (f *lightFetcher) gotHeader(header *types.Header) {
	f.headAnnouncedMu.Lock()
	defer f.headAnnouncedMu.Unlock()

	hash := header.Hash()
	peerList := f.headAnnouncedBy[hash]
	if peerList == nil {
		return
	}
	number := header.Number.Uint64()
	td := core.GetTd(f.pm.chainDb, hash, number)
	for _, peer := range peerList {
		peer.lock.Lock()
		ok := peer.gotHeader(hash, number, td)
		peer.lock.Unlock()
		if !ok {
			//fmt.Println("gotHeader fail")
			f.pm.removePeer(peer.id)
		}
	}
	delete(f.headAnnouncedBy, hash)
}

func (f *lightFetcher) nextRequest() (*peer, *announceData) {
	var bestPeer *peer
	bestTd := f.currentTd
	for _, peer := range f.pm.peers.AllPeers() {
		peer.lock.RLock()
		if !peer.headInfo.requested && (peer.headInfo.Td.Cmp(bestTd) > 0 ||
			(bestPeer != nil && peer.headInfo.Td.Cmp(bestTd) == 0 && peer.headInfo.haveHeaders > bestPeer.headInfo.haveHeaders)) {
			bestPeer = peer
			bestTd = peer.headInfo.Td
		}
		peer.lock.RUnlock()
	}
	if bestPeer == nil {
		return nil, nil
	}
	bestPeer.lock.Lock()
	res := bestPeer.headInfo
	res.requested = true
	bestPeer.lock.Unlock()
	for _, peer := range f.pm.peers.AllPeers() {
		if peer != bestPeer {
			peer.lock.Lock()
			if peer.headInfo.Hash == bestPeer.headInfo.Hash && peer.headInfo.haveHeaders == bestPeer.headInfo.haveHeaders {
				peer.headInfo.requested = true
			}
			peer.lock.Unlock()
		}
	}
	return bestPeer, res
}

func (f *lightFetcher) deliverHeaders(peer *peer, reqID uint64, headers []*types.Header) {
	f.deliverChn <- fetchResponse{reqID: reqID, headers: headers, peer: peer}
}

func (f *lightFetcher) requestedID(reqID uint64) bool {
	f.reqMu.RLock()
	_, ok := f.requested[reqID]
	f.reqMu.RUnlock()
	return ok
}

func (f *lightFetcher) request(p *peer, block *announceData) {
	//fmt.Println("request", p.id, block.Number, block.haveHeaders)
	amount := block.Number - block.haveHeaders
	if amount == 0 {
		return
	}
	if amount > 100 {
		f.syncing = true
		go func() {
			//fmt.Println("f.pm.synchronise(p)")
			f.pm.synchronise(p)
			//fmt.Println("sync done")
			f.syncDone <- struct{}{}
		}()
		return
	}

	reqID := f.odr.getNextReqID()
	f.reqMu.Lock()
	f.requested[reqID] = fetchRequest{hash: block.Hash, amount: amount, peer: p}
	f.reqMu.Unlock()
	cost := p.GetRequestCost(GetBlockHeadersMsg, int(amount))
	p.fcServer.SendRequest(reqID, cost)
	go p.RequestHeadersByHash(reqID, cost, block.Hash, int(amount), 0, true)
	go func() {
		time.Sleep(hardRequestTimeout)
		f.timeoutChn <- reqID
	}()
}

func (f *lightFetcher) processResponse(req fetchRequest, resp fetchResponse) bool {
	if uint64(len(resp.headers)) != req.amount || resp.headers[0].Hash() != req.hash {
		return false
	}
	headers := make([]*types.Header, req.amount)
	for i, header := range resp.headers {
		headers[int(req.amount)-1-i] = header
	}
	if _, err := f.chain.InsertHeaderChain(headers, 1); err != nil {
		return false
	}
	for _, header := range headers {
		td := core.GetTd(f.pm.chainDb, header.Hash(), header.Number.Uint64())
		if td == nil {
			return false
		}
		if td.Cmp(f.currentTd) > 0 {
			f.currentTd = td
		}
		f.gotHeader(header)
	}
	return true
}

func (f *lightFetcher) checkSyncedHeaders() {
	//fmt.Println("checkSyncedHeaders()")
	for _, peer := range f.pm.peers.AllPeers() {
		peer.lock.Lock()
		h := peer.firstHeadInfo
		remove := false
	loop:
		for h != nil {
			if td := core.GetTd(f.pm.chainDb, h.Hash, h.Number); td != nil {
				//fmt.Println(" found", h.Number)
				ok := peer.gotHeader(h.Hash, h.Number, td)
				if !ok {
					remove = true
					break loop
				}
				if td.Cmp(f.currentTd) > 0 {
					f.currentTd = td
				}
			}
			h = h.next
		}
		peer.lock.Unlock()
		if remove {
			//fmt.Println("checkSync fail")
			f.pm.removePeer(peer.id)
		}
	}
}

func (f *lightFetcher) syncLoop() {
	f.pm.wg.Add(1)
	defer f.pm.wg.Done()

	srtoNotify := false
	for {
		select {
		case <-f.pm.quitSync:
			return
		case ext := <-f.notifyChn:
			//fmt.Println("<-f.notifyChn", f.syncing, ext, srtoNotify)
			s := srtoNotify
			srtoNotify = false
			if !f.syncing && !(ext && s) {
				if p, r := f.nextRequest(); r != nil {
					srtoNotify = true
					go func() {
						time.Sleep(softRequestTimeout)
						f.notifyChn <- false
					}()
					f.request(p, r)
				}
			}
		case reqID := <-f.timeoutChn:
			f.reqMu.Lock()
			req, ok := f.requested[reqID]
			if ok {
				delete(f.requested, reqID)
			}
			f.reqMu.Unlock()
			if ok {
				//fmt.Println("hard timeout")
				f.pm.removePeer(req.peer.id)
			}
		case resp := <-f.deliverChn:
			//fmt.Println("<-f.deliverChn", f.syncing)
			f.reqMu.Lock()
			req, ok := f.requested[resp.reqID]
			if ok && req.peer != resp.peer {
				ok = false
			}
			if ok {
				delete(f.requested, resp.reqID)
			}
			f.reqMu.Unlock()
			if !ok || !(f.syncing || f.processResponse(req, resp)) {
				//fmt.Println("processResponse fail")
				f.pm.removePeer(resp.peer.id)
			}
		case <-f.syncDone:
			//fmt.Println("<-f.syncDone", f.syncing)
			f.checkSyncedHeaders()
			f.syncing = false
		}
	}
}
