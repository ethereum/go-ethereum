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
	"errors"
	"math/big"
	"math/rand"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/fetcher"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/log"
)

const (
	blockDelayTimeout = time.Second * 10 // timeout for a peer to announce a head that has already been confirmed by others
)

var (
	errUnknownPeer         = errors.New("announcement from unknown peer")
	errInvalidAnnouncement = errors.New("received announcement is not strictly monotonic")
	errUselessAnnouncement = errors.New("received announcement is useless")
)

// announce represents a new block announcement from server.
type announce struct {
	data  *announceData
	peer  *serverPeer
	errCh chan error
}

// request represent a record when the request is sent.
type request struct {
	reqID  uint64
	peer   *serverPeer
	sendAt time.Time
}

// response represents a response packet from network as well as a channel
// to return all un-requested data.
type response struct {
	reqID   uint64
	headers []*types.Header
	peer    *serverPeer
	remain  chan []*types.Header
}

// query represents an operation to query whether a peer has announced
// a specified hash.
type query struct {
	hash  common.Hash
	peer  *serverPeer
	resCh chan bool
}

// confirm items form a linked list that is expanded with a new item every time
// a new head with a higher Td than the previous one has been downloaded and
// validated.
//
// The list contains a series of maximum confirmed Td values and the time these
// values have been confirmed, both increasing monotonically.
//
// A maximum confirmed Td is calculated both globally for all peers and also for
// each individual peer (meaning that the given peer has announced the head and
// it has also been downloaded from any peer, either before or after the given
// announcement).
//
// The linked list has a global tail where new confirmed Td entries are added and a
// separate head for each peer, pointing to the next Td entry that is higher than
// the peer's max confirmed Td (nil if it has already confirmed the current global head).
type confirm struct {
	time mclock.AbsTime
	td   *big.Int
	next *confirm
}

// lightFetcher implements retrieval of newly announced headers. It also provides a peerHasBlock function for the
// ODR system to ensure that we only request data related to a certain block from peers who have already processed
// and announced that block.
type lightFetcher struct {
	handler *clientHandler
	chain   *light.LightChain // The local light chain which maintains the canonical header chain.
	fetcher *fetcher.Fetcher  // The underlying fetcher which takes care block header retrieval.

	// Channels
	addPeer    chan *serverPeer
	delPeer    chan *serverPeer
	announceCh chan *announce
	requestCh  chan *request
	timeoutCh  chan uint64
	deliverCh  chan *response
	queryCh    chan *query
	syncCh     chan struct{}
	syncDone   chan *types.Header

	closeCh chan struct{}
	wg      sync.WaitGroup

	// Test fields or hooks
	ignoreAnnounce bool
	announceHook   func(*serverPeer, *announceData)
	newHeadHook    func(*types.Header)
	syncingHook    func()
	addDelayHook   func(p *serverPeer, delay time.Duration)
}

// fetcherPeer holds fetcher-specific information about each active peer
type fetcherPeer struct {
	latest      *announceData                 // The latest announcement packet.
	confirmedTd *big.Int                      // The maximum total difficulty which confirmed by local chain and announced by peer.
	confirms    *confirm                      // The confirms list which shared by all peers and fetcher itself.
	announces   map[common.Hash]*announceData // All announcement data.
}

// newLightFetcher creates a new light fetcher
func newLightFetcher(h *clientHandler) *lightFetcher {
	chain := h.backend.blockchain

	// Construct the fetcher by offering all necessary callbacks
	validator := func(header *types.Header) error {
		// Disable seal verification explicitly if we are running in ulc mode.
		return h.backend.engine.VerifyHeader(chain, header, !h.isULCEnabled())
	}
	heighter := func() uint64 { return chain.CurrentHeader().Number.Uint64() }
	dropper := func(id string) { h.backend.peers.unregister(id) }
	inserter := func(headers []*types.Header) (int, error) {
		// Disable PoW checking explicitly if we are running ulc mode.
		checkFreq := 1
		if h.isULCEnabled() {
			checkFreq = 0
		}
		return chain.InsertHeaderChain(headers, checkFreq)
	}
	f := &lightFetcher{
		handler:    h,
		fetcher:    fetcher.New(true, chain.GetHeaderByHash, nil, validator, nil, heighter, inserter, nil, dropper),
		chain:      h.backend.blockchain,
		addPeer:    make(chan *serverPeer),
		delPeer:    make(chan *serverPeer),
		announceCh: make(chan *announce),
		requestCh:  make(chan *request),
		timeoutCh:  make(chan uint64),
		deliverCh:  make(chan *response),
		queryCh:    make(chan *query),
		syncCh:     make(chan struct{}),
		syncDone:   make(chan *types.Header),
		closeCh:    make(chan struct{}),
	}
	h.backend.peers.subscribe(f)
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

// syncLoop is the main event loop of the light fetcher, which is responsible for
// * announcement maintenance(ulc)
// * block header retrieval
// * re-sync trigger
// * response delay and announcement delay statistic
func (f *lightFetcher) mainloop() {
	defer f.wg.Done()

	var (
		syncDist = uint64(1) // Interval used to trigger a light resync.
		syncing  bool        // Indicator whether the client is syncing

		ulc           = f.handler.isULCEnabled()
		headCh        = make(chan core.ChainHeadEvent, 100)
		peers         = make(map[*serverPeer]*fetcherPeer)
		trusted       = make(map[common.Hash][]*serverPeer)
		trustedNumber = make(map[common.Hash]uint64)
		fetching      = make(map[uint64]*request)

		// Local status
		localConfirm *confirm
		localHead    = f.chain.CurrentHeader()
		localTd      = f.chain.GetTd(localHead.Hash(), localHead.Number.Uint64())
	)
	sub := f.chain.SubscribeChainHeadEvent(headCh)
	defer sub.Unsubscribe()

	// resetLocal updates the local status with given header.
	resetLocal := func(header *types.Header) {
		localHead = header
		localTd = f.chain.GetTd(header.Hash(), header.Number.Uint64())

		// All confirm records will be linked together and shared by all peers.
		// In this way, we can judge whether the announcement from peer has
		// hysteresis according to the maximum difficulty of each peer being confirmed.
		if localConfirm == nil || localConfirm.td.Cmp(localTd) < 0 {
			newConfirm := &confirm{time: mclock.Now(), td: localTd}
			if localConfirm != nil {
				localConfirm.next = newConfirm
			}
			localConfirm = newConfirm
		}
	}
	// trustedHeader returns an indicator whether the header is regarded as
	// trusted. If we are running in the ulc mode, only when we receive enough
	// same announcement from trusted server, the header will be trusted.
	trustedHeader := func(hash common.Hash) bool {
		agreed := len(trusted[hash])
		return 100*agreed/len(f.handler.ulc.trustedKeys) >= f.handler.ulc.minTrustedFraction
	}
	// updateQoS drops stale confirm items and updates announcement delay statistic.
	updateQoS := func(p *serverPeer, fp *fetcherPeer) {
		now := mclock.Now()
		// Track global confirm list if haven't.
		if fp.confirms == nil {
			fp.confirms = localConfirm
		}
		// Discard stale confirm items and feed server pool a delay statistic.
		if fp.confirmedTd != nil {
			for fp.confirms != nil && fp.confirms.td.Cmp(fp.confirmedTd) <= 0 {
				if f.addDelayHook != nil {
					f.addDelayHook(p, time.Duration(now-fp.confirms.time))
				}
				f.handler.backend.serverPool.adjustBlockDelay(p.poolEntry, time.Duration(now-fp.confirms.time))
				p.Log().Debug("Add announcement delay", "time", common.PrettyDuration(time.Duration(now-fp.confirms.time)))
				fp.confirms = fp.confirms.next
			}
		}
		// Drop expired confirm items and feed server pool a "timeout" statistic.
		for fp.confirms != nil && fp.confirms.time <= now-mclock.AbsTime(blockDelayTimeout) {
			if f.addDelayHook != nil {
				f.addDelayHook(p, blockDelayTimeout)
				p.Log().Debug("Add announcement delay", "time", common.PrettyDuration(blockDelayTimeout))
			}
			f.handler.backend.serverPool.adjustBlockDelay(p.poolEntry, blockDelayTimeout)
			fp.confirms = fp.confirms.next
		}
	}

	for {
		select {
		case p := <-f.addPeer:
			peers[p] = &fetcherPeer{announces: make(map[common.Hash]*announceData)}
			log.Debug("Register peer", "id", p.id)

		case p := <-f.delPeer:
			delete(peers, p)
			log.Debug("Unregister peer", "id", p.id)

		case anno := <-f.announceCh:
			p, data := anno.peer, anno.data
			p.Log().Debug("Received new announcement", "number", data.Number, "hash", data.Hash, "reorg", data.ReorgDepth)

			fp, exist := peers[p]
			if !exist {
				p.Log().Debug("Announcement from unknown peer")
				anno.errCh <- errUnknownPeer
				continue
			}
			// announced tds should be strictly monotonic.
			if fp.latest != nil && data.Td.Cmp(fp.latest.Td) <= 0 {
				p.Log().Debug("Received non-monotonic td", "current", data.Td, "previous", fp.latest.Td)
				anno.errCh <- errInvalidAnnouncement
				continue
			}
			// filter out stale announcement if the td is less than local one.
			if localTd != nil && data.Td.Cmp(localTd) <= 0 {
				if f.chain.HasHeader(data.Hash, data.Number) {
					fp.latest, fp.confirmedTd = data, data.Td
					updateQoS(p, fp)
					anno.errCh <- nil
					p.Log().Debug("Received announcement is stale", "local", localTd, "received", data.Td)
				} else {
					anno.errCh <- errUselessAnnouncement
					p.Log().Debug("Received announcement is useless", "local", localTd, "received", data.Td)
				}
				continue
			}
			fp.latest, fp.announces[data.Hash] = data, data

			if !ulc && !syncing {
				if data.Number > localHead.Number.Uint64()+syncDist {
					// Trigger light sync if the new announcement is not continuous
					// with local chain.
					p.Log().Debug("Trigger light sync", "local", localHead.Number, "localhash", localHead.Hash(), "remote", data.Number, "remotehash", data.Hash)
					f.requestReSync(data.Hash)
				} else {
					p.Log().Debug("Trigger header retrieval", "number", data.Number, "hash", data.Hash)
					f.fetcher.Notify(p.id, data.Hash, data.Number, time.Now(), f.requestHeaderByHash(p, data.Hash), nil)
				}
			}
			if ulc && p.trusted {
				// Keep collecting announcement from trusted server even we are syncing.
				number, hash := data.Number, data.Hash
				trusted[hash], trustedNumber[hash] = append(trusted[hash], p), data.Number

				// Notify underlying fetcher to retrieve header or trigger a resync if
				// we have receive enough announcements from trusted server.
				if trustedHeader(hash) && !syncing {
					if number > localHead.Number.Uint64()+syncDist {
						p.Log().Debug("Trigger trusted light sync", "local", localHead.Number, "localhash", localHead.Hash(), "remote", data.Number, "remotehash", data.Hash)
						f.requestReSync(data.Hash)
					} else {
						p := trusted[hash][rand.Intn(len(trusted[hash]))]
						p.Log().Debug("Trigger trusted header retrieval", "number", data.Number, "hash", data.Hash)
						f.fetcher.Notify(p.id, hash, number, time.Now(), f.requestHeaderByHash(p, hash), nil)
					}
				}
			}
			anno.errCh <- nil

		case req := <-f.requestCh:
			fetching[req.reqID] = req // Tracking all in-flight requests for response latency statistic.

		case id := <-f.timeoutCh:
			if req, exist := fetching[id]; exist {
				log.Debug("request timeout", "peer", req.peer.id, "reqid", id)
				delete(fetching, id)
				f.handler.backend.serverPool.adjustResponseTime(req.peer.poolEntry, time.Since(req.sendAt), true)
				go f.handler.backend.peers.unregister(req.peer.id)
			}

		case resp := <-f.deliverCh:
			if req := fetching[resp.reqID]; req != nil {
				// Feed response delay statistic for server pool.
				delete(fetching, resp.reqID)
				f.handler.backend.serverPool.adjustResponseTime(req.peer.poolEntry, time.Since(req.sendAt), false)

				resp.remain <- f.fetcher.FilterHeaders(resp.peer.id, resp.headers, time.Now())
			} else {
				// Discard the entire packet no matter it's a timeout response or unexpected one.
				resp.remain <- resp.headers
			}

		case q := <-f.queryCh:
			fp := peers[q.peer]
			q.resCh <- fp != nil && fp.announces[q.hash] != nil

		case ev := <-headCh:
			// Short circuit if we are still syncing.
			if syncing {
				continue
			}
			resetLocal(ev.Block.Header())
			number, hash := localHead.Number.Uint64(), localHead.Hash()

			// Clean stale announcements from trusted server.
			if ulc {
				for h, n := range trustedNumber {
					if n <= number {
						delete(trustedNumber, h)
						delete(trusted, h)
					}
				}
			}
			for p, fp := range peers {
				// Update the maximum confirmed td of peer if it already announced it.
				if _, exist := fp.announces[hash]; exist {
					fp.confirmedTd = localTd
				}
				updateQoS(p, fp)
				// Delete all stale announcements.
				for h, anno := range fp.announces {
					if h == hash || anno.Number < number {
						delete(fp.announces, h)
					}
				}
			}
			if f.newHeadHook != nil {
				f.newHeadHook(localHead)
			}
			log.Debug("receive new head", "number", number, "hash", hash)

		case <-f.syncCh:
			syncing = true // Mark the syncing as true only if we truly start syncing.

		case origin := <-f.syncDone:
			syncing = false // Reset the status

			// Rewind all untrusted headers for ulc mode.
			if ulc {
				head := f.chain.CurrentHeader()
				ancestor := rawdb.FindCommonAncestor(f.handler.backend.chainDb, origin, head)
				if ancestor == nil {
					// todo how should we handle this
				}
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
			resetLocal(f.chain.CurrentHeader())
			log.Debug("light sync finished", "number", localHead.Number, "hash", localHead.Hash())

		case <-f.closeCh:
			return
		}
	}
}

// registerPeer adds a new peer to the fetcher's peer set
func (f *lightFetcher) registerPeer(p *serverPeer) {
	select {
	case f.addPeer <- p:
	case <-f.closeCh:
	}
}

// unregisterPeer removes a new peer from the fetcher's peer set
func (f *lightFetcher) unregisterPeer(p *serverPeer) {
	select {
	case f.delPeer <- p:
	case <-f.closeCh:
	}
}

// announce processes a new announcement message received from a peer.
func (f *lightFetcher) announce(p *serverPeer, head *announceData) {
	if f.ignoreAnnounce {
		return
	}
	if f.announceHook != nil {
		f.announceHook(p, head)
	}
	errCh := make(chan error, 1)
	select {
	case f.announceCh <- &announce{peer: p, data: head, errCh: errCh}:
	case <-f.closeCh:
		return
	}
	err := <-errCh
	switch err {
	case errInvalidAnnouncement, errUselessAnnouncement:
		f.handler.backend.peers.unregister(p.id)
	default:
	}
}

// queryAnnounced checks whether the specified peer has announced a given
// hash announcement.
func (f *lightFetcher) queryAnnounced(peer *serverPeer, hash common.Hash) bool {
	resCh := make(chan bool, 1)
	select {
	case f.queryCh <- &query{peer: peer, hash: hash, resCh: resCh}:
		return <-resCh
	case <-f.closeCh:
		return false
	}
}

// trackRequest sends a reqID to main loop for in-flight request tracking.
func (f *lightFetcher) trackRequest(peer *serverPeer, id uint64) {
	select {
	case f.requestCh <- &request{reqID: id, peer: peer, sendAt: time.Now()}:
	case <-f.closeCh:
	}
}

// requestHeaderByHash constructs a header retrieval request and sends it to
// local request distributor.
// Note, we rely on the underlying eth/fetcher to retrieve and validate the response,
// so that we have to obey the rule of eth/fetcher which only accepts the response
// from given peer.
func (f *lightFetcher) requestHeaderByHash(peer *serverPeer, hash common.Hash) func(common.Hash) error {
	return func(hash common.Hash) error {
		req := &distReq{
			getCost: func(dp distPeer) uint64 { return dp.(*serverPeer).getRequestCost(GetBlockHeadersMsg, 1) },
			canSend: func(dp distPeer) bool { return dp.(*serverPeer) == peer },
			request: func(dp distPeer) func() {
				id := genReqID()
				cost := peer.getRequestCost(GetBlockHeadersMsg, 1)
				peer.fcServer.QueuedRequest(id, cost)

				f.trackRequest(peer, id)
				go func() {
					time.Sleep(hardRequestTimeout)
					f.timeoutCh <- id
				}()
				return func() { peer.requestHeadersByHash(id, hash, 1, 0, false) }
			},
		}
		// We have to spawn a go routine here is we need to send query
		// to main loop, otherwise it will stuck the whole loop.
		go func() {
			<-f.handler.backend.reqDist.queue(req)
		}()
		return nil
	}
}

// requestReSync constructs a re-sync request based on a given block hash.
func (f *lightFetcher) requestReSync(headHash common.Hash) {
	req := &distReq{
		getCost: func(dp distPeer) uint64 { return 0 },
		canSend: func(dp distPeer) bool {
			p := dp.(*serverPeer)
			if p.announceOnly {
				return false
			}
			return f.queryAnnounced(p, headHash)
		},
		request: func(dp distPeer) func() {
			go func() {
				p := dp.(*serverPeer)
				origin := f.chain.CurrentHeader()

				f.syncCh <- struct{}{} // Mark the status of fetcher as syncing.
				defer func() {
					f.syncDone <- origin
				}()
				if f.syncingHook != nil {
					f.syncingHook()
				}
				f.handler.synchronise(p)
			}()
			return nil
		},
	}
	// We have to spawn a go routine here is we need to send query
	// to main loop, otherwise it will stuck the whole loop.
	go func() {
		<-f.handler.backend.reqDist.queue(req)
	}()
}

// deliverHeaders delivers header download request responses for processing
func (f *lightFetcher) deliverHeaders(peer *serverPeer, reqID uint64, headers []*types.Header) []*types.Header {
	remain := make(chan []*types.Header, 1)
	select {
	case f.deliverCh <- &response{reqID: reqID, headers: headers, peer: peer, remain: remain}:
	case <-f.closeCh:
		return nil
	}
	return <-remain
}
