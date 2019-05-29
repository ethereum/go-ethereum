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
	"math"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/prque"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

// relayTracer includes the raw transaction and relative
// relay information.
type relayTracer struct {
	tx    *types.Transaction
	queue *prque.Prque  // Priority queue of the peers to relay the transactions to.
	index map[*peer]int // Peer indexes in the priority queue used to remove element.
}

// newRelayTracer creates a relayTracer and initializes the priority queue.
func newRelayTracer(tx *types.Transaction, peers []*peer) *relayTracer {
	info := &relayTracer{
		tx:    tx,
		index: make(map[*peer]int),
	}
	info.queue = prque.New(info.setIndex)
	for _, peer := range peers {
		info.queue.Push(peer, 0)
	}
	return info
}

// setIndex saves the index in queue of element into the map.
func (r *relayTracer) setIndex(a interface{}, i int) {
	r.index[a.(*peer)] = i
}

type LesTxRelay struct {
	peerList  []*peer
	retriever *retrieveManager
	pending   map[common.Hash]*relayTracer // Transactions which has been sent but not finalized.
	stop      chan struct{}
	lock      sync.RWMutex
}

func NewLesTxRelay(ps *peerSet, retriever *retrieveManager) *LesTxRelay {
	r := &LesTxRelay{
		pending:   make(map[common.Hash]*relayTracer),
		retriever: retriever,
		stop:      make(chan struct{}),
	}
	for _, peer := range ps.AllPeers() {
		if !peer.isOnlyAnnounce {
			r.peerList = append(r.peerList, peer)
		}
	}
	ps.notify(r)
	return r
}

func (l *LesTxRelay) Stop() {
	close(l.stop)
}

func (l *LesTxRelay) registerPeer(p *peer) {
	l.lock.Lock()
	defer l.lock.Unlock()

	// Short circuit if the peer is announce only.
	if p.isOnlyAnnounce {
		return
	}
	l.peerList = append(l.peerList, p)

	// Register new peer to all relay tracers.
	for _, tx := range l.pending {
		tx.queue.Push(p, 0)
	}
}

func (l *LesTxRelay) unregisterPeer(p *peer) {
	l.lock.Lock()
	defer l.lock.Unlock()

	for i, peer := range l.peerList {
		if peer == p {
			// Remove from the peer list
			l.peerList = append(l.peerList[:i], l.peerList[i+1:]...)

			// Update all relay tracers as well.
			for _, tx := range l.pending {
				if _, exist := tx.index[p]; exist {
					tx.queue.Remove(tx.index[p])
					delete(tx.index, p)
				}
			}
		}
	}
}

// send relays a list of transactions to at most a given number of peers at
// once, never resending any particular transaction to the same peer twice.
func (l *LesTxRelay) send(txs types.Transactions) {
	var (
		resend = int(math.Sqrt(float64(len(l.peerList))))
		sendTo = make(map[*peer]types.Transactions)
	)
	for _, tx := range txs {
		hash := tx.Hash()
		t, exist := l.pending[hash]
		if !exist {
			t = newRelayTracer(tx, l.peerList)
			l.pending[hash] = t
		}
		// If this is a new transaction, broadcast to all sendable peers.
		// Otherwise(e.g. resend reverted transaction), only send to a part
		// of them.
		cnt := len(l.peerList)
		if exist {
			cnt = resend
		}
		for i := 0; i < cnt; i++ {
			item, priority := t.queue.Pop()
			peer, ok := item.(*peer)
			if !ok {
				log.Warn("Unexpected item in priority queue")
				continue
			}
			sendTo[peer] = append(sendTo[peer], tx)
			t.queue.Push(item, priority-1)
		}
	}

	for p, txs := range sendTo {
		var (
			pp     = p
			ll     = txs
			enc, _ = rlp.EncodeToBytes(txs)
			reqID  = genReqID()
		)
		rq := &distReq{
			getCost: func(dp distPeer) uint64 {
				peer := dp.(*peer)
				return peer.GetTxRelayCost(len(ll), len(enc))
			},
			canSend: func(dp distPeer) bool {
				return !dp.(*peer).isOnlyAnnounce && dp.(*peer) == pp
			},
			request: func(dp distPeer) func() {
				peer := dp.(*peer)
				cost := peer.GetTxRelayCost(len(ll), len(enc))
				peer.fcServer.QueuedRequest(reqID, cost)
				return func() { peer.SendTxs(reqID, cost, enc) }
			},
		}
		go l.retriever.retrieve(context.Background(), reqID, rq, func(p distPeer, msg *Msg) error { return nil }, l.stop)
	}
}

// Send relays a batch of transaction into the network and returns all unsend
// transactions.
func (l *LesTxRelay) Send(txs types.Transactions) error {
	l.lock.Lock()
	defer l.lock.Unlock()

	if len(l.peerList) == 0 {
		return light.ErrNoPeers
	}
	l.send(txs)
	return nil
}

// Discard marks a batch of transaction are finalized and won't be reverted.
func (l *LesTxRelay) Discard(hashes []common.Hash) {
	l.lock.Lock()
	defer l.lock.Unlock()

	for _, hash := range hashes {
		delete(l.pending, hash)
	}
}
