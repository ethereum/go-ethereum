// Copyright 2026 The go-ethereum Authors
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

package eth

import (
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/eth/protocols/eth"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

// balFetchTimeout bounds one access-list request to a single peer.
const balFetchTimeout = 5 * time.Second

// balFetcher resolves the migration's missing access lists from eth/71+
// peers, one fetch at a time.
type balFetcher struct {
	db    ethdb.Database
	chain *core.BlockChain
	peers *peerSet
	drop  func(id string)
	kick  func()

	reqCh  chan []core.BALRequest
	term   chan struct{}
	closed chan struct{}
}

func newBALFetcher(db ethdb.Database, chain *core.BlockChain, peers *peerSet, drop func(string), kick func()) *balFetcher {
	f := &balFetcher{
		db:     db,
		chain:  chain,
		peers:  peers,
		drop:   drop,
		kick:   kick,
		reqCh:  make(chan []core.BALRequest, 1),
		term:   make(chan struct{}),
		closed: make(chan struct{}),
	}
	go f.loop()
	return f
}

func (f *balFetcher) stop() {
	close(f.term)
	<-f.closed
}

// request enqueues the follower's ask without blocking; with a fetch already
// queued the next stall retry re-asks.
func (f *balFetcher) request(reqs []core.BALRequest) {
	select {
	case f.reqCh <- reqs:
	default:
	}
}

func (f *balFetcher) loop() {
	defer close(f.closed)
	for {
		select {
		case reqs := <-f.reqCh:
			if f.fetch(reqs) {
				f.kick()
			}
		case <-f.term:
			return
		}
	}
}

// fetch tries the connected eth/71 peers in turn until the lists land or the
// peers run out, reporting whether anything was stored.
func (f *balFetcher) fetch(pending []core.BALRequest) bool {
	stored := false
	for _, peer := range f.peers.all() {
		if len(pending) == 0 {
			break
		}
		if peer.Version() < eth.ETH71 {
			continue
		}
		select {
		case <-f.term:
			return stored
		default:
		}
		n, rest := f.fetchFrom(peer.Peer, pending)
		stored = stored || n > 0
		pending = rest
	}
	if len(pending) > 0 {
		log.Debug("Missing access lists unresolved", "count", len(pending))
	}
	return stored
}

// fetchFrom asks one peer, stores what verifies and returns what is still missing.
func (f *balFetcher) fetchFrom(peer *eth.Peer, pending []core.BALRequest) (int, []core.BALRequest) {
	hashes := make([]common.Hash, len(pending))
	for i, r := range pending {
		hashes[i] = r.Hash
	}
	resCh := make(chan *eth.Response)
	req, err := peer.RequestBALs(hashes, resCh)
	if err != nil {
		return 0, pending
	}
	defer req.Close()

	timeout := time.NewTimer(balFetchTimeout)
	defer timeout.Stop()
	select {
	case res := <-resCh:
		// The dispatcher rejects oversized replies before they reach the
		// sink, so at most one item per request arrives here.
		items := ([]rlp.RawValue)(*res.Res.(*eth.BlockAccessListResponse))
		metas := res.Meta.([]common.Hash)
		// Verify everything before storing anything: one forged item voids
		// the whole response and the peer.
		var verified []int
		for i, meta := range metas {
			if meta == (common.Hash{}) {
				continue // not served
			}
			header := f.chain.GetHeader(pending[i].Hash, pending[i].Number)
			if header == nil || header.BlockAccessListHash == nil {
				continue
			}
			if meta != *header.BlockAccessListHash {
				res.Done <- errors.New("access list mismatch")
				log.Warn("Dropping peer forging access lists", "peer", peer.ID(), "block", pending[i].Number)
				f.drop(peer.ID())
				return 0, pending
			}
			verified = append(verified, i)
		}
		served := make(map[int]bool, len(verified))
		for _, i := range verified {
			rawdb.WriteAccessListRLP(f.db, pending[i].Hash, pending[i].Number, items[i])
			served[i] = true
		}
		res.Done <- nil
		var rest []core.BALRequest
		for i, r := range pending {
			if !served[i] {
				rest = append(rest, r)
			}
		}
		return len(verified), rest
	case <-timeout.C:
		return 0, pending
	case <-f.term:
		return 0, pending
	}
}
