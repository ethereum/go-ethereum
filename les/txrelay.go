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

package les

import (
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type ltrInfo struct {
	tx     *types.Transaction
	sentTo map[*peer]struct{}
}

type LesTxRelay struct {
	txSent       map[common.Hash]*ltrInfo
	txPending    map[common.Hash]struct{}
	ps           *peerSet
	peerList     []*peer
	peerStartPos int
	lock         sync.RWMutex

	reqDist *requestDistributor
}

func NewLesTxRelay(ps *peerSet, reqDist *requestDistributor) *LesTxRelay {
	r := &LesTxRelay{
		txSent:    make(map[common.Hash]*ltrInfo),
		txPending: make(map[common.Hash]struct{}),
		ps:        ps,
		reqDist:   reqDist,
	}
	ps.notify(r)
	return r
}

func (relay *LesTxRelay) registerPeer(p *peer) {
	relay.lock.Lock()
	defer relay.lock.Unlock()

	relay.peerList = relay.ps.AllPeers()
}

func (relay *LesTxRelay) unregisterPeer(p *peer) {
	relay.lock.Lock()
	defer relay.lock.Unlock()

	relay.peerList = relay.ps.AllPeers()
}

// send sends a list of transactions to at most a given number of peers at
// once, never resending any particular transaction to the same peer twice
func (relay *LesTxRelay) send(txs types.Transactions, count int) {
	sendTo := make(map[*peer]types.Transactions)

	relay.peerStartPos++ // rotate the starting position of the peer list
	if relay.peerStartPos >= len(relay.peerList) {
		relay.peerStartPos = 0
	}

	for _, tx := range txs {
		hash := tx.Hash()
		ltr, ok := relay.txSent[hash]
		if !ok {
			ltr = &ltrInfo{
				tx:     tx,
				sentTo: make(map[*peer]struct{}),
			}
			relay.txSent[hash] = ltr
			relay.txPending[hash] = struct{}{}
		}

		if len(relay.peerList) > 0 {
			cnt := count
			pos := relay.peerStartPos
			for {
				peer := relay.peerList[pos]
				if _, ok := ltr.sentTo[peer]; !ok {
					sendTo[peer] = append(sendTo[peer], tx)
					ltr.sentTo[peer] = struct{}{}
					cnt--
				}
				if cnt == 0 {
					break // sent it to the desired number of peers
				}
				pos++
				if pos == len(relay.peerList) {
					pos = 0
				}
				if pos == relay.peerStartPos {
					break // tried all available peers
				}
			}
		}
	}

	for p, list := range sendTo {
		pp := p
		ll := list

		reqID := genReqID()
		rq := &distReq{
			getCost: func(dp distPeer) uint64 {
				peer := dp.(*peer)
				return peer.GetRequestCost(SendTxMsg, len(ll))
			},
			canSend: func(dp distPeer) bool {
				return dp.(*peer) == pp
			},
			request: func(dp distPeer) func() {
				peer := dp.(*peer)
				cost := peer.GetRequestCost(SendTxMsg, len(ll))
				peer.fcServer.QueueRequest(reqID, cost)
				return func() { peer.SendTxs(reqID, cost, ll) }
			},
		}
		relay.reqDist.queue(rq)
	}
}

func (relay *LesTxRelay) Send(txs types.Transactions) {
	relay.lock.Lock()
	defer relay.lock.Unlock()

	relay.send(txs, 3)
}

func (relay *LesTxRelay) NewHead(head common.Hash, mined []common.Hash, rollback []common.Hash) {
	relay.lock.Lock()
	defer relay.lock.Unlock()

	for _, hash := range mined {
		delete(relay.txPending, hash)
	}

	for _, hash := range rollback {
		relay.txPending[hash] = struct{}{}
	}

	if len(relay.txPending) > 0 {
		txs := make(types.Transactions, len(relay.txPending))
		i := 0
		for hash := range relay.txPending {
			txs[i] = relay.txSent[hash].tx
			i++
		}
		relay.send(txs, 1)
	}
}

func (relay *LesTxRelay) Discard(hashes []common.Hash) {
	relay.lock.Lock()
	defer relay.lock.Unlock()

	for _, hash := range hashes {
		delete(relay.txSent, hash)
		delete(relay.txPending, hash)
	}
}
