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
	"math/big"
	"math/rand"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/les/fetcher"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

const (
	blockDelayTimeout    = 10 * time.Second       // Timeout for retrieving the headers from the peer
	gatherSlack          = 100 * time.Millisecond // Interval used to collate almost-expired requests
	cachedAnnosThreshold = 64                     // The maximum queued announcements
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
	hash   common.Hash
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
	latest *announceData // The latest announcement sent from the peer

	// These following two fields can track the latest announces
	// from the peer with limited size for caching. We hold the
	// assumption that all enqueued announces are td-monotonic.
	announces map[common.Hash]*announce // Announcement map
	fifo      []common.Hash             // FIFO announces list
}

// addAnno enqueues an new trusted announcement. If the queued announces overflow,
// evict from the oldest.
func (fp *fetcherPeer) addAnno(anno *announce) {
	// Short circuit if the anno already exists. In normal case it should
	// never happen since only monotonic anno is accepted. But the adversary
	// may feed us fake announces with higher td but same hash. In this case,
	// ignore the anno anyway.
	hash := anno.data.Hash
	if _, exist := fp.announces[hash]; exist {
		return
	}
	fp.announces[hash] = anno
	fp.fifo = append(fp.fifo, hash)

	// Evict oldest if the announces are oversized.
	if len(fp.fifo)-cachedAnnosThreshold > 0 {
		for i := 0; i < len(fp.fifo)-cachedAnnosThreshold; i++ {
			delete(fp.announces, fp.fifo[i])
		}
		copy(fp.fifo, fp.fifo[len(fp.fifo)-cachedAnnosThreshold:])
		fp.fifo = fp.fifo[:cachedAnnosThreshold]
	}
}

// forwardAnno removes all announces from the map with a number lower than
// the provided threshold.
func (fp *fetcherPeer) forwardAnno(td *big.Int) []*announce {
	var (
		cutset  int
		evicted []*announce
	)
	for ; cutset < len(fp.fifo); cutset++ {
		anno := fp.announces[fp.fifo[cutset]]
		if anno == nil {
			continue // In theory it should never ever happen
		}
		if anno.data.Td.Cmp(td) > 0 {
			break
		}
		evicted = append(evicted, anno)
		delete(fp.announces, anno.data.Hash)
	}
	if cutset > 0 {
		copy(fp.fifo, fp.fifo[cutset:])
		fp.fifo = fp.fifo[:len(fp.fifo)-cutset]
	}
	return evicted
}

// lightFetcher implements retrieval of newly announced headers. It reuses
// the eth.BlockFetcher as the underlying fetcher but adding more additional
// rules: e.g. evict "timeout" peers.
type lightFetcher struct {
	// Various handlers
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
	newHeadHook func(*types.Header)
}

// newLightFetcher creates a light fetcher instance.
func newLightFetcher(chain *light.LightChain, engine consensus.Engine, peers *serverPeerSet, ulc *ulc, chaindb ethdb.Database, reqDist *requestDistributor, syncFn func(p *serverPeer)) *lightFetcher {
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

	f.peers[p.ID()] = &fetcherPeer{announces: make(map[common.Hash]*announce)}
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

// forEachPeer iterates the fetcher peerset, abort the iteration if the
// callback returns false.
func (f *lightFetcher) forEachPeer(check func(id enode.ID, p *fetcherPeer) bool) {
	f.plock.RLock()
	defer f.plock.RUnlock()

	for id, peer := range f.peers {
		if !check(id, peer) {
			return
		}
	}
}

// mainloop is the main event loop of the light fetcher, which is responsible for
//
//   - announcement maintenance(ulc)
//
//     If we are running in ultra light client mode, then all announcements from
//     the trusted servers are maintained. If the same announcements from trusted
//     servers reach the threshold, then the relevant header is requested for retrieval.
//
//   - block header retrieval
//     Whenever we receive announce with higher td compared with local chain, the
//     request will be made for header retrieval.
//
//   - re-sync trigger
//     If the local chain lags too much, then the fetcher will enter "synchronise"
//     mode to retrieve missing headers in batch.
func (f *lightFetcher) mainloop() {
	defer f.wg.Done()

	var (
		syncInterval = uint64(1) // Interval used to trigger a light resync.
		syncing      bool        // Indicator whether the client is syncing

		ulc          = f.ulc != nil
		headCh       = make(chan core.ChainHeadEvent, 100)
		fetching     = make(map[uint64]*request)
		requestTimer = time.NewTimer(0)

		// Local status
		localHead = f.chain.CurrentHeader()
		localTd   = f.chain.GetTd(localHead.Hash(), localHead.Number.Uint64())
	)
	defer requestTimer.Stop()
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
	trustedHeader := func(hash common.Hash, number uint64) (bool, []enode.ID) {
		var (
			agreed  []enode.ID
			trusted bool
		)
		f.forEachPeer(func(id enode.ID, p *fetcherPeer) bool {
			if anno := p.announces[hash]; anno != nil && anno.trust && anno.data.Number == number {
				agreed = append(agreed, id)
				if 100*len(agreed)/len(f.ulc.keys) >= f.ulc.fraction {
					trusted = true
					return false // abort iteration
				}
			}
			return true
		})
		return trusted, agreed
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
			// the announce is out-of-order.
			if peer.latest != nil && data.Td.Cmp(peer.latest.Td) <= 0 {
				f.peerset.unregister(peerid.String())
				log.Debug("Non-monotonic td", "peer", peerid, "current", data.Td, "previous", peer.latest.Td)
				continue
			}
			peer.latest = data

			// Filter out any stale announce, the local chain is ahead of announce
			if localTd != nil && data.Td.Cmp(localTd) <= 0 {
				continue
			}
			peer.addAnno(anno)

			// If we are not syncing, try to trigger a single retrieval or re-sync
			if !ulc && !syncing {
				// Two scenarios lead to re-sync:
				// - reorg happens
				// - local chain lags
				// We can't retrieve the parent of the announce by single retrieval
				// in both cases, so resync is necessary.
				if data.Number > localHead.Number.Uint64()+syncInterval || data.ReorgDepth > 0 {
					syncing = true
					go f.startSync(peerid)
					log.Debug("Trigger light sync", "peer", peerid, "local", localHead.Number, "localhash", localHead.Hash(), "remote", data.Number, "remotehash", data.Hash)
					continue
				}
				f.fetcher.Notify(peerid.String(), data.Hash, data.Number, time.Now(), f.requestHeaderByHash(peerid), nil)
				log.Debug("Trigger header retrieval", "peer", peerid, "number", data.Number, "hash", data.Hash)
			}
			// Keep collecting announces from trusted server even we are syncing.
			if ulc && anno.trust {
				// Notify underlying fetcher to retrieve header or trigger a resync if
				// we have receive enough announcements from trusted server.
				trusted, agreed := trustedHeader(data.Hash, data.Number)
				if trusted && !syncing {
					if data.Number > localHead.Number.Uint64()+syncInterval || data.ReorgDepth > 0 {
						syncing = true
						go f.startSync(peerid)
						log.Debug("Trigger trusted light sync", "local", localHead.Number, "localhash", localHead.Hash(), "remote", data.Number, "remotehash", data.Hash)
						continue
					}
					p := agreed[rand.Intn(len(agreed))]
					f.fetcher.Notify(p.String(), data.Hash, data.Number, time.Now(), f.requestHeaderByHash(p), nil)
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
					f.peerset.unregister(request.peerid.String())
					log.Debug("Request timeout", "peer", request.peerid, "reqid", reqid)
				}
			}
			f.rescheduleTimer(fetching, requestTimer)

		case resp := <-f.deliverCh:
			if req := fetching[resp.reqid]; req != nil {
				delete(fetching, resp.reqid)
				f.rescheduleTimer(fetching, requestTimer)

				// The underlying fetcher does not check the consistency of request and response.
				// The adversary can send the fake announces with invalid hash and number but always
				// delivery some mismatched header. So it can't be punished by the underlying fetcher.
				// We have to add two more rules here to detect.
				if len(resp.headers) != 1 {
					f.peerset.unregister(req.peerid.String())
					log.Debug("Deliver more than requested", "peer", req.peerid, "reqid", req.reqid)
					continue
				}
				if resp.headers[0].Hash() != req.hash {
					f.peerset.unregister(req.peerid.String())
					log.Debug("Deliver invalid header", "peer", req.peerid, "reqid", req.reqid)
					continue
				}
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

			// Clean stale announcements from les-servers.
			var droplist []enode.ID
			f.forEachPeer(func(id enode.ID, p *fetcherPeer) bool {
				removed := p.forwardAnno(localTd)
				for _, anno := range removed {
					if header := f.chain.GetHeaderByHash(anno.data.Hash); header != nil {
						if header.Number.Uint64() != anno.data.Number {
							droplist = append(droplist, id)
							break
						}
						// In theory td should exists.
						td := f.chain.GetTd(anno.data.Hash, anno.data.Number)
						if td != nil && td.Cmp(anno.data.Td) != 0 {
							droplist = append(droplist, id)
							break
						}
					}
				}
				return true
			})
			for _, id := range droplist {
				f.peerset.unregister(id.String())
				log.Debug("Kicked out peer for invalid announcement")
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

				// Recap the ancestor with genesis header in case the ancestor
				// is not found. It can happen the original head is before the
				// checkpoint while the synced headers are after it. In this
				// case there is no ancestor between them.
				if ancestor == nil {
					ancestor = f.chain.Genesis().Header()
				}
				var untrusted []common.Hash
				for head.Number.Cmp(ancestor.Number) > 0 {
					hash, number := head.Hash(), head.Number.Uint64()
					if trusted, _ := trustedHeader(hash, number); trusted {
						break
					}
					untrusted = append(untrusted, hash)
					head = f.chain.GetHeader(head.ParentHash, number-1)
					if head == nil {
						break // all the synced headers will be dropped
					}
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
	select {
	case f.announceCh <- &announce{peerid: p.ID(), trust: p.trusted, data: head}:
	case <-f.closeCh:
		return
	}
}

// trackRequest sends a reqID to main loop for in-flight request tracking.
func (f *lightFetcher) trackRequest(peerid enode.ID, reqid uint64, hash common.Hash) {
	select {
	case f.requestCh <- &request{reqid: reqid, peerid: peerid, sendAt: time.Now(), hash: hash}:
	case <-f.closeCh:
	}
}

// requestHeaderByHash constructs a header retrieval request and sends it to
// local request distributor.
//
// Note, we rely on the underlying eth/fetcher to retrieve and validate the
// response, so that we have to obey the rule of eth/fetcher which only accepts
// the response from given peer.
func (f *lightFetcher) requestHeaderByHash(peerid enode.ID) func(common.Hash) error {
	return func(hash common.Hash) error {
		req := &distReq{
			getCost: func(dp distPeer) uint64 { return dp.(*serverPeer).getRequestCost(GetBlockHeadersMsg, 1) },
			canSend: func(dp distPeer) bool { return dp.(*serverPeer).ID() == peerid },
			request: func(dp distPeer) func() {
				peer, id := dp.(*serverPeer), rand.Uint64()
				cost := peer.getRequestCost(GetBlockHeadersMsg, 1)
				peer.fcServer.QueuedRequest(id, cost)

				return func() {
					f.trackRequest(peer.ID(), id, hash)
					peer.requestHeadersByHash(id, hash, 1, 0, false)
				}
			},
		}
		f.reqDist.queue(req)
		return nil
	}
}

// startSync invokes synchronisation callback to start syncing.
func (f *lightFetcher) startSync(id enode.ID) {
	defer func(header *types.Header) {
		f.syncDone <- header
	}(f.chain.CurrentHeader())

	peer := f.peerset.peer(id.String())
	if peer == nil || peer.onlyAnnounce {
		return
	}
	f.synchronise(peer)
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
		timer.Stop()
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
