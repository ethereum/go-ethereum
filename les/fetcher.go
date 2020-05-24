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
	"math/rand"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/fetcher"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/les/utils"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

const (
	blockDelayTimeout = 10 * time.Second       // Timeout for retrieving the headers from the peer
	gatherSlack       = 100 * time.Millisecond // Interval used to collate almost-expired requests

	outOrderThreshold = 3         // The maximum limit the server is allowed to send out_of_order announces
	timeoutThreshold  = 5         // The maximum limit the server is allowed for timeout requests
	expirationRate    = time.Hour // The linear expiration rate of the _bad_ behaviors statistic
)

// announce represents an new block announcement from the les server.
type announce struct {
	data   *announceData
	trust  bool
	peerid enode.ID
}

// request represents a record when the header request is sent.
type request struct {
	reqid  uint64
	peerid enode.ID
	sendAt time.Time
}

// response represents a response packet from network as well as a channel
// to return all un-requested data.
type response struct {
	reqid   uint64
	headers []*types.Header
	peerid  enode.ID
	remain  chan []*types.Header
}

// fetcherPeer holds the fetcher-specific information for each active peer
type fetcherPeer struct {
	latest   *announceData            // The latest announcement sent from the peer
	timeout  utils.LinearExpiredValue // The counter of all timeout requests made
	outOrder utils.LinearExpiredValue // The counter of all out-of-oreder announces
}

// lightFetcher implements retrieval of newly announced headers. It reuses
// the eth.BlockFetcher as the underlying fetcher but adding more additional
// rules: e.g. evict "timeout" peers.
type lightFetcher struct {
	// Various handlers
	clock   mclock.Clock
	ulc     *ulc
	chaindb ethdb.Database
	reqDist *requestDistributor
	peerset *serverPeerSet        // The global peerset of light client which shared by all components
	chain   *light.LightChain     // The local light chain which maintains the canonical header chain.
	fetcher *fetcher.BlockFetcher // The underlying fetcher which takes care block header retrieval.

	// Peerset maintained by fetcher
	plock sync.RWMutex
	peers map[enode.ID]*fetcherPeer

	// Various channels
	announceCh chan *announce
	requestCh  chan *request
	deliverCh  chan *response
	syncDone   chan *types.Header

	closeCh chan struct{}
	wg      sync.WaitGroup

	// Callback
	synchronise func(peer *serverPeer)

	// Test fields or hooks
	noAnnounce  bool
	newHeadHook func(*types.Header)
}

// newLightFetcher creates a light fetcher instance.
func newLightFetcher(clock mclock.Clock, chain *light.LightChain, engine consensus.Engine, peers *serverPeerSet, ulc *ulc, chaindb ethdb.Database, reqDist *requestDistributor, syncFn func(p *serverPeer)) *lightFetcher {
	// Construct the fetcher by offering all necessary APIs
	validator := func(header *types.Header) error {
		// Disable seal verification explicitly if we are running in ulc mode.
		return engine.VerifyHeader(chain, header, ulc == nil)
	}
	heighter := func() uint64 { return chain.CurrentHeader().Number.Uint64() }
	dropper := func(id string) { peers.unregister(id) }
	inserter := func(headers []*types.Header) (int, error) {
		// Disable PoW checking explicitly if we are running in ulc mode.
		checkFreq := 1
		if ulc != nil {
			checkFreq = 0
		}
		return chain.InsertHeaderChain(headers, checkFreq)
	}
	f := &lightFetcher{
		clock:       clock,
		ulc:         ulc,
		peerset:     peers,
		chaindb:     chaindb,
		chain:       chain,
		reqDist:     reqDist,
		fetcher:     fetcher.NewBlockFetcher(true, chain.GetHeaderByHash, nil, validator, nil, heighter, inserter, nil, dropper),
		peers:       make(map[enode.ID]*fetcherPeer),
		synchronise: syncFn,
		announceCh:  make(chan *announce),
		requestCh:   make(chan *request),
		deliverCh:   make(chan *response),
		syncDone:    make(chan *types.Header),
		closeCh:     make(chan struct{}),
	}
	peers.subscribe(f)
	return f
}

func (f *lightFetcher) start() {
	f.wg.Add(1)
	f.fetcher.Start()
	go f.mainloop()
}

func (f *lightFetcher) stop() {
	close(f.closeCh)
	f.fetcher.Stop()
	f.wg.Wait()
}

// registerPeer adds an new peer to the fetcher's peer set
func (f *lightFetcher) registerPeer(p *serverPeer) {
	f.plock.Lock()
	defer f.plock.Unlock()

	f.peers[p.ID()] = &fetcherPeer{
		timeout:  utils.LinearExpiredValue{Offset: uint64(f.clock.Now() / mclock.AbsTime(expirationRate))},
		outOrder: utils.LinearExpiredValue{Offset: uint64(f.clock.Now() / mclock.AbsTime(expirationRate))},
	}
}

// unregisterPeer removes the specified peer from the fetcher's peer set
func (f *lightFetcher) unregisterPeer(p *serverPeer) {
	f.plock.Lock()
	defer f.plock.Unlock()

	delete(f.peers, p.ID())
}

// peer returns the peer from the fetcher peerset.
func (f *lightFetcher) peer(id enode.ID) *fetcherPeer {
	f.plock.RLock()
	defer f.plock.RUnlock()

	return f.peers[id]
}

// mainloop is the main event loop of the light fetcher, which is responsible for
// - announcement maintenance(ulc)
//   If we are running in ultra light client mode, then all announcements from
//   the trusted servers are maintained. If the same announcements from trusted
//   servers reach the threshold, then the relevant header is requested for retrieval.
//
// - block header retrieval
//   Whenever we receive announce with higher td compared with local chain, the
//   request will be made for header retrieval.
//
// - re-sync trigger
//   If the local chain lags too much, then the fetcher will enter "synnchronise"
//   mode to retrieve missing headers in batch.
func (f *lightFetcher) mainloop() {
	defer f.wg.Done()

	var (
		syncInterval = uint64(1) // Interval used to trigger a light resync.
		syncing      bool        // Indicator whether the client is syncing

		ulc           = f.ulc != nil
		headCh        = make(chan core.ChainHeadEvent, 100)
		trusted       = make(map[common.Hash][]enode.ID)
		trustedNumber = make(map[common.Hash]uint64)
		fetching      = make(map[uint64]*request)
		requestTimer  = time.NewTimer(0)

		// Local status
		localHead = f.chain.CurrentHeader()
		localTd   = f.chain.GetTd(localHead.Hash(), localHead.Number.Uint64())
	)
	sub := f.chain.SubscribeChainHeadEvent(headCh)
	defer sub.Unsubscribe()

	// reset updates the local status with given header.
	reset := func(header *types.Header) {
		localHead = header
		localTd = f.chain.GetTd(header.Hash(), header.Number.Uint64())
	}
	// trustedHeader returns an indicator whether the header is regarded as
	// trusted. If we are running in the ulc mode, only when we receive enough
	// same announcement from trusted server, the header will be trusted.
	trustedHeader := func(hash common.Hash) bool {
		return 100*len(trusted[hash])/len(f.ulc.keys) >= f.ulc.fraction
	}
	for {
		select {
		case anno := <-f.announceCh:
			peerid, data := anno.peerid, anno.data
			log.Debug("Received new announce", "peer", peerid, "number", data.Number, "hash", data.Hash, "reorg", data.ReorgDepth)

			peer := f.peer(peerid)
			if peer == nil {
				log.Debug("Receive announce from unknown peer", "peer", peerid)
				continue
			}
			// Announced tds should be strictly monotonic, drop the peer if
			// there are too many out-of-order announces accumulated.
			if peer.latest != nil && data.Td.Cmp(peer.latest.Td) <= 0 {
				log.Debug("Non-monotonic td", "peer", peerid, "current", data.Td, "previous", peer.latest.Td)

				if count := peer.outOrder.Add(1, uint64(f.clock.Now()/mclock.AbsTime(expirationRate))); count > outOrderThreshold {
					f.peerset.unregister(peerid.String())
				}
				continue
			}
			peer.latest = data

			// Filter out any stale announce, the local chain is ahead of announce
			if localTd != nil && data.Td.Cmp(localTd) <= 0 {
				continue
			}
			// If we are not syncing, try to trigger a single retrieval or re-sync
			if !ulc && !syncing {
				// Two scenarios lead to re-sync:
				// - reorg happens
				// - local chain lags
				// We can't retrieve the parent of the announce by single retrieval
				// in both cases, so resync is necessary.
				if data.Number > localHead.Number.Uint64()+syncInterval || data.ReorgDepth > 0 {
					syncing = true
					if !f.requestResync(peerid) {
						syncing = false
					}
					log.Debug("Trigger light sync", "peer", peerid, "local", localHead.Number, "localhash", localHead.Hash(), "remote", data.Number, "remotehash", data.Hash)
					continue
				}
				f.fetcher.Notify(peerid.String(), data.Hash, data.Number, time.Now(), f.requestHeaderByHash(peerid, data.Hash), nil)
				log.Debug("Trigger header retrieval", "peer", peerid, "number", data.Number, "hash", data.Hash)
			}
			// Keep collecting announces from trusted server even we are syncing.
			if ulc && anno.trust {
				number, hash := data.Number, data.Hash
				trusted[hash], trustedNumber[hash] = append(trusted[hash], peerid), data.Number

				// Notify underlying fetcher to retrieve header or trigger a resync if
				// we have receive enough announcements from trusted server.
				if trustedHeader(hash) && !syncing {
					if number > localHead.Number.Uint64()+syncInterval || data.ReorgDepth > 0 {
						syncing = true
						if !f.requestResync(peerid) {
							syncing = false
						}
						log.Debug("Trigger trusted light sync", "local", localHead.Number, "localhash", localHead.Hash(), "remote", data.Number, "remotehash", data.Hash)
						continue
					}
					p := trusted[hash][rand.Intn(len(trusted[hash]))]
					f.fetcher.Notify(p.String(), hash, number, time.Now(), f.requestHeaderByHash(p, hash), nil)
					log.Debug("Trigger trusted header retrieval", "number", data.Number, "hash", data.Hash)
				}
			}

		case req := <-f.requestCh:
			fetching[req.reqid] = req // Tracking all in-flight requests for response latency statistic.
			if len(fetching) == 1 {
				f.rescheduleTimer(fetching, requestTimer)
			}

		case <-requestTimer.C:
			for reqid, request := range fetching {
				if time.Since(request.sendAt) > blockDelayTimeout-gatherSlack {
					delete(fetching, reqid)
					log.Debug("request timeout", "peer", request.peerid, "reqid", reqid)

					peer := f.peer(request.peerid)
					if peer == nil {
						continue
					}
					if count := peer.timeout.Add(1, uint64(f.clock.Now()/mclock.AbsTime(expirationRate))); count > timeoutThreshold {
						f.peerset.unregister(request.peerid.String())
					}
				}
			}
			f.rescheduleTimer(fetching, requestTimer)

		case resp := <-f.deliverCh:
			if req := fetching[resp.reqid]; req != nil {
				delete(fetching, resp.reqid)

				resp.remain <- f.fetcher.FilterHeaders(resp.peerid.String(), resp.headers, time.Now())
			} else {
				// Discard the entire packet no matter it's a timeout response or unexpected one.
				resp.remain <- resp.headers
			}

		case ev := <-headCh:
			// Short circuit if we are still syncing.
			if syncing {
				continue
			}
			reset(ev.Block.Header())
			number := localHead.Number.Uint64()

			// Clean stale announcements from trusted server.
			if ulc {
				for h, n := range trustedNumber {
					if n <= number {
						delete(trustedNumber, h)
						delete(trusted, h)
					}
				}
			}
			if f.newHeadHook != nil {
				f.newHeadHook(localHead)
			}

		case origin := <-f.syncDone:
			syncing = false // Reset the status

			// Rewind all untrusted headers for ulc mode.
			if ulc {
				head := f.chain.CurrentHeader()
				ancestor := rawdb.FindCommonAncestor(f.chaindb, origin, head)
				var untrusted []common.Hash
				for head.Number.Cmp(ancestor.Number) > 0 {
					hash := head.Hash()
					if trustedHeader(hash) {
						break
					}
					untrusted = append(untrusted, hash)
					head = f.chain.GetHeader(head.ParentHash, head.Number.Uint64()-1)
				}
				if len(untrusted) > 0 {
					for i, j := 0, len(untrusted)-1; i < j; i, j = i+1, j-1 {
						untrusted[i], untrusted[j] = untrusted[j], untrusted[i]
					}
					f.chain.Rollback(untrusted)
				}
			}
			// Reset local status.
			reset(f.chain.CurrentHeader())
			if f.newHeadHook != nil {
				f.newHeadHook(localHead)
			}
			log.Debug("light sync finished", "number", localHead.Number, "hash", localHead.Hash())

		case <-f.closeCh:
			return
		}
	}
}

// announce processes a new announcement message received from a peer.
func (f *lightFetcher) announce(p *serverPeer, head *announceData) {
	if f.noAnnounce {
		return
	}
	select {
	case f.announceCh <- &announce{peerid: p.ID(), trust: p.trusted, data: head}:
	case <-f.closeCh:
		return
	}
}

// trackRequest sends a reqID to main loop for in-flight request tracking.
func (f *lightFetcher) trackRequest(peerid enode.ID, reqid uint64) {
	select {
	case f.requestCh <- &request{reqid: reqid, peerid: peerid, sendAt: time.Now()}:
	case <-f.closeCh:
	}
}

// requestHeaderByHash constructs a header retrieval request and sends it to
// local request distributor.
//
// Note, we rely on the underlying eth/fetcher to retrieve and validate the
// response, so that we have to obey the rule of eth/fetcher which only accepts
// the response from given peer.
func (f *lightFetcher) requestHeaderByHash(peerid enode.ID, hash common.Hash) func(common.Hash) error {
	return func(hash common.Hash) error {
		req := &distReq{
			getCost: func(dp distPeer) uint64 { return dp.(*serverPeer).getRequestCost(GetBlockHeadersMsg, 1) },
			canSend: func(dp distPeer) bool { return dp.(*serverPeer).ID() == peerid },
			request: func(dp distPeer) func() {
				peer, id := dp.(*serverPeer), genReqID()
				cost := peer.getRequestCost(GetBlockHeadersMsg, 1)
				peer.fcServer.QueuedRequest(id, cost)

				f.trackRequest(peer.ID(), id)
				return func() { peer.requestHeadersByHash(id, hash, 1, 0, false) }
			},
		}
		f.reqDist.queue(req)
		return nil
	}
}

// requestResync constructs a re-sync request based on a given block hash.
func (f *lightFetcher) requestResync(peerid enode.ID) bool {
	req := &distReq{
		getCost: func(dp distPeer) uint64 { return 0 },
		canSend: func(dp distPeer) bool {
			p := dp.(*serverPeer)
			if p.onlyAnnounce {
				return false
			}
			return p.ID() == peerid
		},
		request: func(dp distPeer) func() {
			go func() {
				defer func(header *types.Header) {
					f.syncDone <- header
				}(f.chain.CurrentHeader())

				f.synchronise(dp.(*serverPeer))
			}()
			return nil
		},
	}
	_, ok := <-f.reqDist.queue(req)
	return ok
}

// deliverHeaders delivers header download request responses for processing
func (f *lightFetcher) deliverHeaders(peer *serverPeer, reqid uint64, headers []*types.Header) []*types.Header {
	remain := make(chan []*types.Header, 1)
	select {
	case f.deliverCh <- &response{reqid: reqid, headers: headers, peerid: peer.ID(), remain: remain}:
	case <-f.closeCh:
		return nil
	}
	return <-remain
}

// rescheduleTimer resets the specified timeout timer to the next request timeout.
func (f *lightFetcher) rescheduleTimer(requests map[uint64]*request, timer *time.Timer) {
	// Short circuit if no inflight requests
	if len(requests) == 0 {
		return
	}
	// Otherwise find the earliest expiring request
	earliest := time.Now()
	for _, req := range requests {
		if earliest.After(req.sendAt) {
			earliest = req.sendAt
		}
	}
	timer.Reset(blockDelayTimeout - time.Since(earliest))
}
