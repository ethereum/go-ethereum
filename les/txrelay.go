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
	"github.com/ethereum/go-ethereum/rlp"
)

type ltrInfo struct {
	tx     *types.Transaction
	sentTo map[*peer]struct{}
}

type lesTxRelay struct {
	txSent       map[common.Hash]*ltrInfo
	txPending    map[common.Hash]struct{}
	ps           *peerSet
	peerList     []*peer
	peerStartPos int
	lock         sync.RWMutex
	stop         chan struct{}

	retriever *retrieveManager
}

func newLesTxRelay(ps *peerSet, retriever *retrieveManager) *lesTxRelay {
	r := &lesTxRelay{
		txSent:    make(map[common.Hash]*ltrInfo),
		txPending: make(map[common.Hash]struct{}),
		ps:        ps,
		retriever: retriever,
		stop:      make(chan struct{}),
	}
	ps.notify(r)
	return r
}

func (ltrx *lesTxRelay) Stop() {
	close(ltrx.stop)
}

func (ltrx *lesTxRelay) registerPeer(p *peer) {
	ltrx.lock.Lock()
	defer ltrx.lock.Unlock()

	ltrx.peerList = ltrx.ps.AllPeers()
}

func (ltrx *lesTxRelay) unregisterPeer(p *peer) {
	ltrx.lock.Lock()
	defer ltrx.lock.Unlock()

	ltrx.peerList = ltrx.ps.AllPeers()
}

// send sends a list of transactions to at most a given number of peers at
// once, never resending any particular transaction to the same peer twice
func (ltrx *lesTxRelay) send(txs types.Transactions, count int) {
	sendTo := make(map[*peer]types.Transactions)

	ltrx.peerStartPos++ // rotate the starting position of the peer list
	if ltrx.peerStartPos >= len(ltrx.peerList) {
		ltrx.peerStartPos = 0
	}

	for _, tx := range txs {
		hash := tx.Hash()
		ltr, ok := ltrx.txSent[hash]
		if !ok {
			ltr = &ltrInfo{
				tx:     tx,
				sentTo: make(map[*peer]struct{}),
			}
			ltrx.txSent[hash] = ltr
			ltrx.txPending[hash] = struct{}{}
		}

		if len(ltrx.peerList) > 0 {
			cnt := count
			pos := ltrx.peerStartPos
			for {
				peer := ltrx.peerList[pos]
				if _, ok := ltr.sentTo[peer]; !ok {
					sendTo[peer] = append(sendTo[peer], tx)
					ltr.sentTo[peer] = struct{}{}
					cnt--
				}
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
				peer := dp.(*peer)
				return peer.GetTxRelayCost(len(ll), len(enc))
			},
			canSend: func(dp distPeer) bool {
				return !dp.(*peer).onlyAnnounce && dp.(*peer) == pp
			},
			request: func(dp distPeer) func() {
				peer := dp.(*peer)
				cost := peer.GetTxRelayCost(len(ll), len(enc))
				peer.fcServer.QueuedRequest(reqID, cost)
				return func() { peer.SendTxs(reqID, cost, enc) }
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
			txs[i] = ltrx.txSent[hash].tx
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
