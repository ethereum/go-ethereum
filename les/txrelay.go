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
	"context"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/nodestate"
	"github.com/ethereum/go-ethereum/rlp"
)

type lesTxRelay struct {
	txSent       map[common.Hash]*types.Transaction
	txPending    map[common.Hash]struct{}
	peerList     []*serverPeer
	peerStartPos int
	lock         sync.Mutex
	stop         chan struct{}

	retriever *retrieveManager
}

func newLesTxRelay(ns *nodestate.NodeStateMachine, retriever *retrieveManager) *lesTxRelay {
	r := &lesTxRelay{
		txSent:    make(map[common.Hash]*types.Transaction),
		txPending: make(map[common.Hash]struct{}),
		retriever: retriever,
		stop:      make(chan struct{}),
	}
	ns.SubscribeField(serverPeerField, func(node *enode.Node, state nodestate.Flags, oldValue, newValue interface{}) {
		r.lock.Lock()
		defer r.lock.Unlock()

		if newValue != nil {
			p := newValue.(*serverPeer)
			// Short circuit if the peer is announce only.
			if p.onlyAnnounce {
				return
			}
			r.peerList = append(r.peerList, p)
		} else {
			p := oldValue.(*serverPeer)
			for i, peer := range r.peerList {
				if peer == p {
					// Remove from the peer list
					r.peerList = append(r.peerList[:i], r.peerList[i+1:]...)
					return
				}
			}
		}
	})
	return r
}

func (ltrx *lesTxRelay) Stop() {
	close(ltrx.stop)
}

// send sends a list of transactions to at most a given number of peers.
func (ltrx *lesTxRelay) send(txs types.Transactions, count int) {
	sendTo := make(map[*serverPeer]types.Transactions)

	ltrx.peerStartPos++ // rotate the starting position of the peer list
	if ltrx.peerStartPos >= len(ltrx.peerList) {
		ltrx.peerStartPos = 0
	}

	for _, tx := range txs {
		hash := tx.Hash()
		_, ok := ltrx.txSent[hash]
		if !ok {
			ltrx.txSent[hash] = tx
			ltrx.txPending[hash] = struct{}{}
		}
		if len(ltrx.peerList) > 0 {
			cnt := count
			pos := ltrx.peerStartPos
			for {
				peer := ltrx.peerList[pos]
				sendTo[peer] = append(sendTo[peer], tx)
				cnt--
				if cnt == 0 {
					break // sent it to the desired number of peers
				}
				pos++
				if pos == len(ltrx.peerList) {
					pos = 0
				}
				if pos == ltrx.peerStartPos {
					break // tried all available peers
				}
			}
		}
	}

	for p, list := range sendTo {
		pp := p
		ll := list
		enc, _ := rlp.EncodeToBytes(ll)

		reqID := genReqID()
		rq := &distReq{
			getCost: func(dp distPeer) uint64 {
				peer := dp.(*serverPeer)
				return peer.getTxRelayCost(len(ll), len(enc))
			},
			canSend: func(dp distPeer) bool {
				return !dp.(*serverPeer).onlyAnnounce && dp.(*serverPeer) == pp
			},
			request: func(dp distPeer) func() {
				peer := dp.(*serverPeer)
				cost := peer.getTxRelayCost(len(ll), len(enc))
				peer.fcServer.QueuedRequest(reqID, cost)
				return func() { peer.sendTxs(reqID, len(ll), enc) }
			},
		}
		go ltrx.retriever.retrieve(context.Background(), reqID, rq, func(p distPeer, msg *Msg) error { return nil }, ltrx.stop)
	}
}

func (ltrx *lesTxRelay) Send(txs types.Transactions) {
	ltrx.lock.Lock()
	defer ltrx.lock.Unlock()

	ltrx.send(txs, 3)
}

func (ltrx *lesTxRelay) NewHead(head common.Hash, mined []common.Hash, rollback []common.Hash) {
	ltrx.lock.Lock()
	defer ltrx.lock.Unlock()

	for _, hash := range mined {
		delete(ltrx.txPending, hash)
	}

	for _, hash := range rollback {
		ltrx.txPending[hash] = struct{}{}
	}

	if len(ltrx.txPending) > 0 {
		txs := make(types.Transactions, len(ltrx.txPending))
		i := 0
		for hash := range ltrx.txPending {
			txs[i] = ltrx.txSent[hash]
			i++
		}
		ltrx.send(txs, 1)
	}
}

func (ltrx *lesTxRelay) Discard(hashes []common.Hash) {
	ltrx.lock.Lock()
	defer ltrx.lock.Unlock()

	for _, hash := range hashes {
		delete(ltrx.txSent, hash)
		delete(ltrx.txPending, hash)
	}
}
