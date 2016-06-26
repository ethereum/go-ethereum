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
package les

import (
	"sync"
)

const dropTimeoutRatio = 20

type odrPeerInfo struct {
	reqTimeSum, reqTimeCnt, reqCnt, timeoutCnt uint64
}

// odrPeerSet represents the collection of active peer participating in the block
// download procedure.
type odrPeerSet struct {
	peers map[*peer]*odrPeerInfo
	lock  sync.RWMutex
}

// newPeerSet creates a new peer set top track the active download sources.
func newOdrPeerSet() *odrPeerSet {
	return &odrPeerSet{
		peers: make(map[*peer]*odrPeerInfo),
	}
}

// Register injects a new peer into the working set, or returns an error if the
// peer is already known.
func (ps *odrPeerSet) register(p *peer) error {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	if _, ok := ps.peers[p]; ok {
		return errAlreadyRegistered
	}
	ps.peers[p] = &odrPeerInfo{}
	return nil
}

// Unregister removes a remote peer from the active set, disabling any further
// actions to/from that particular entity.
func (ps *odrPeerSet) unregister(p *peer) error {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	if _, ok := ps.peers[p]; !ok {
		return errNotRegistered
	}
	delete(ps.peers, p)
	return nil
}

func (ps *odrPeerSet) peerPriority(p *peer, info *odrPeerInfo, req LesOdrRequest) uint64 {
	tm := p.fcServer.CanSend(req.GetCost(p))
	if info.reqTimeCnt > 0 {
		tm += info.reqTimeSum / info.reqTimeCnt
	}
	return tm
}

func (ps *odrPeerSet) bestPeer(req LesOdrRequest, exclude map[*peer]struct{}) *peer {
	var best *peer
	var bpv uint64
	ps.lock.Lock()
	defer ps.lock.Unlock()

	for p, info := range ps.peers {
		if _, ok := exclude[p]; !ok {
			pv := ps.peerPriority(p, info, req)
			if best == nil || pv < bpv {
				best = p
				bpv = pv
			}
		}
	}
	return best
}

func (ps *odrPeerSet) updateTimeout(p *peer, timeout bool) (drop bool) {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	if info, ok := ps.peers[p]; ok {
		info.reqCnt++
		if timeout {
			// check ratio before increase to allow an extra timeout
			if info.timeoutCnt*dropTimeoutRatio >= info.reqCnt {
				return true
			}
			info.timeoutCnt++
		}
	}
	return false
}

func (ps *odrPeerSet) updateServTime(p *peer, servTime uint64) {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	if info, ok := ps.peers[p]; ok {
		info.reqTimeSum += servTime
		info.reqTimeCnt++
	}
}
