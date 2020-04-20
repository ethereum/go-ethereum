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
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/log"
)

const (
	blockDelayTimeout    = time.Second * 10 // timeout for a peer to announce a head that has already been confirmed by others
	maxNodeCount         = 20               // maximum number of fetcherTreeNode entries remembered for each peer
	serverStateAvailable = 100              // number of recent blocks where state availability is assumed
)

// lightFetcher implements retrieval of newly announced headers. It also provides a peerHasBlock function for the
// ODR system to ensure that we only request data related to a certain block from peers who have already processed
// and announced that block.
type lightFetcher struct {
	handler *clientHandler
	chain   *light.LightChain

	lock            sync.Mutex // lock protects access to the fetcher's internal state variables except sent requests
	maxConfirmedTd  *big.Int
	peers           map[*serverPeer]*fetcherPeerInfo
	lastUpdateStats *updateStatsEntry
	syncing         bool
	syncDone        chan *serverPeer

	reqMu             sync.RWMutex // reqMu protects access to sent header fetch requests
	requested         map[uint64]fetchRequest
	deliverChn        chan fetchResponse
	timeoutChn        chan uint64
	requestTriggered  bool
	requestTrigger    chan struct{}
	lastTrustedHeader *types.Header

	closeCh chan struct{}
	wg      sync.WaitGroup
}

// fetcherPeerInfo holds fetcher-specific information about each active peer
type fetcherPeerInfo struct {
	root, lastAnnounced *fetcherTreeNode
	nodeCnt             int
	confirmedTd         *big.Int
	bestConfirmed       *fetcherTreeNode
	nodeByHash          map[common.Hash]*fetcherTreeNode
	firstUpdateStats    *updateStatsEntry
}

// fetcherTreeNode is a node of a tree that holds information about blocks recently
// announced and confirmed by a certain peer. Each new announce message from a peer
// adds nodes to the tree, based on the previous announced head and the reorg depth.
// There are three possible states for a tree node:
// - announced: not downloaded (known) yet, but we know its head, number and td
// - intermediate: not known, hash and td are empty, they are filled out when it becomes known
// - known: both announced by this peer and downloaded (from any peer).
// This structure makes it possible to always know which peer has a certain block,
// which is necessary for selecting a suitable peer for ODR requests and also for
// canonizing new heads. It also helps to always download the minimum necessary
// amount of headers with a single request.
type fetcherTreeNode struct {
	hash             common.Hash
	number           uint64
	td               *big.Int
	known, requested bool
	parent           *fetcherTreeNode
	children         []*fetcherTreeNode
}

// fetchRequest represents a header download request
type fetchRequest struct {
	hash    common.Hash
	amount  uint64
	peer    *serverPeer
	sent    mclock.AbsTime
	timeout bool
}

// fetchResponse represents a header download response
type fetchResponse struct {
	reqID   uint64
	headers []*types.Header
	peer    *serverPeer
}

// newLightFetcher creates a new light fetcher
func newLightFetcher(h *clientHandler) *lightFetcher {
	f := &lightFetcher{
		handler:        h,
		chain:          h.backend.blockchain,
		peers:          make(map[*serverPeer]*fetcherPeerInfo),
		deliverChn:     make(chan fetchResponse, 100),
		requested:      make(map[uint64]fetchRequest),
		timeoutChn:     make(chan uint64),
		requestTrigger: make(chan struct{}, 1),
		syncDone:       make(chan *serverPeer),
		closeCh:        make(chan struct{}),
		maxConfirmedTd: big.NewInt(0),
	}
	h.backend.peers.subscribe(f)

	f.wg.Add(1)
	go f.syncLoop()
	return f
}

func (f *lightFetcher) close() {
	close(f.closeCh)
	f.wg.Wait()
}

// syncLoop is the main event loop of the light fetcher
func (f *lightFetcher) syncLoop() {
	defer f.wg.Done()
	for {
		select {
		case <-f.closeCh:
			return
		// request loop keeps running until no further requests are necessary or possible
		case <-f.requestTrigger:
			f.lock.Lock()
			var (
				rq      *distReq
				reqID   uint64
				syncing bool
			)
			if !f.syncing {
				rq, reqID, syncing = f.nextRequest()
			}
			f.requestTriggered = rq != nil
			f.lock.Unlock()

			if rq != nil {
				if _, ok := <-f.handler.backend.reqDist.queue(rq); ok {
					if syncing {
						f.lock.Lock()
						f.syncing = true
						f.lock.Unlock()
					} else {
						go func() {
							time.Sleep(softRequestTimeout)
							f.reqMu.Lock()
							req, ok := f.requested[reqID]
							if ok {
								req.timeout = true
								f.requested[reqID] = req
							}
							f.reqMu.Unlock()
							// keep starting new requests while possible
							f.requestTrigger <- struct{}{}
						}()
					}
				} else {
					f.requestTrigger <- struct{}{}
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
				f.handler.backend.serverPool.adjustResponseTime(req.peer.poolEntry, time.Duration(mclock.Now()-req.sent), true)
				req.peer.Log().Debug("Fetching data timed out hard")
				go f.handler.removePeer(req.peer.id)
			}
		case resp := <-f.deliverChn:
			f.reqMu.Lock()
			req, ok := f.requested[resp.reqID]
			if ok && req.peer != resp.peer {
				ok = false
			}
			if ok {
				delete(f.requested, resp.reqID)
			}
			f.reqMu.Unlock()
			if ok {
				f.handler.backend.serverPool.adjustResponseTime(req.peer.poolEntry, time.Duration(mclock.Now()-req.sent), req.timeout)
			}
			f.lock.Lock()
			if !ok || !(f.syncing || f.processResponse(req, resp)) {
				resp.peer.Log().Debug("Failed processing response")
				go f.handler.removePeer(resp.peer.id)
			}
			f.lock.Unlock()
		case p := <-f.syncDone:
			f.lock.Lock()
			p.Log().Debug("Done synchronising with peer")
			f.checkSyncedHeaders(p)
			f.syncing = false
			f.lock.Unlock()
			f.requestTrigger <- struct{}{} // f.requestTriggered is always true here
		}
	}
}

// registerPeer adds a new peer to the fetcher's peer set
func (f *lightFetcher) registerPeer(p *serverPeer) {
	p.lock.Lock()
	p.hasBlock = func(hash common.Hash, number uint64, hasState bool) bool {
		return f.peerHasBlock(p, hash, number, hasState)
	}
	p.lock.Unlock()

	f.lock.Lock()
	defer f.lock.Unlock()
	f.peers[p] = &fetcherPeerInfo{nodeByHash: make(map[common.Hash]*fetcherTreeNode)}
}

// unregisterPeer removes a new peer from the fetcher's peer set
func (f *lightFetcher) unregisterPeer(p *serverPeer) {
	p.lock.Lock()
	p.hasBlock = nil
	p.lock.Unlock()

	f.lock.Lock()
	defer f.lock.Unlock()

	// check for potential timed out block delay statistics
	f.checkUpdateStats(p, nil)
	delete(f.peers, p)
}

// announce processes a new announcement message received from a peer, adding new
// nodes to the peer's block tree and removing old nodes if necessary
func (f *lightFetcher) announce(p *serverPeer, head *announceData) {
	f.lock.Lock()
	defer f.lock.Unlock()
	p.Log().Debug("Received new announcement", "number", head.Number, "hash", head.Hash, "reorg", head.ReorgDepth)

	fp := f.peers[p]
	if fp == nil {
		p.Log().Debug("Announcement from unknown peer")
		return
	}

	if fp.lastAnnounced != nil && head.Td.Cmp(fp.lastAnnounced.td) <= 0 {
		// announced tds should be strictly monotonic
		p.Log().Debug("Received non-monotonic td", "current", head.Td, "previous", fp.lastAnnounced.td)
		go f.handler.removePeer(p.id)
		return
	}

	n := fp.lastAnnounced
	for i := uint64(0); i < head.ReorgDepth; i++ {
		if n == nil {
			break
		}
		n = n.parent
	}
	// n is now the reorg common ancestor, add a new branch of nodes
	if n != nil && (head.Number >= n.number+maxNodeCount || head.Number <= n.number) {
		// if announced head block height is lower or same as n or too far from it to add
		// intermediate nodes then discard previous announcement info and trigger a resync
		n = nil
		fp.nodeCnt = 0
		fp.nodeByHash = make(map[common.Hash]*fetcherTreeNode)
	}
	// check if the node count is too high to add new nodes, discard oldest ones if necessary
	if n != nil {
		// n is now the reorg common ancestor, add a new branch of nodes
		// check if the node count is too high to add new nodes
		locked := false
		for uint64(fp.nodeCnt)+head.Number-n.number > maxNodeCount && fp.root != nil {
			if !locked {
				f.chain.LockChain()
				defer f.chain.UnlockChain()
				locked = true
			}
			// if one of root's children is canonical, keep it, delete other branches and root itself
			var newRoot *fetcherTreeNode
			for i, nn := range fp.root.children {
				if rawdb.ReadCanonicalHash(f.handler.backend.chainDb, nn.number) == nn.hash {
					fp.root.children = append(fp.root.children[:i], fp.root.children[i+1:]...)
					nn.parent = nil
					newRoot = nn
					break
				}
			}
			fp.deleteNode(fp.root)
			if n == fp.root {
				n = newRoot
			}
			fp.root = newRoot
			if newRoot == nil || !f.checkKnownNode(p, newRoot) {
				fp.bestConfirmed = nil
				fp.confirmedTd = nil
			}

			if n == nil {
				break
			}
		}
		if n != nil {
			for n.number < head.Number {
				nn := &fetcherTreeNode{number: n.number + 1, parent: n}
				n.children = append(n.children, nn)
				n = nn
				fp.nodeCnt++
			}
			n.hash = head.Hash
			n.td = head.Td
			fp.nodeByHash[n.hash] = n
		}
	}

	if n == nil {
		// could not find reorg common ancestor or had to delete entire tree, a new root and a resync is needed
		if fp.root != nil {
			fp.deleteNode(fp.root)
		}
		n = &fetcherTreeNode{hash: head.Hash, number: head.Number, td: head.Td}
		fp.root = n
		fp.nodeCnt++
		fp.nodeByHash[n.hash] = n
		fp.bestConfirmed = nil
		fp.confirmedTd = nil
	}

	f.checkKnownNode(p, n)
	p.lock.Lock()
	p.headInfo = blockInfo{Number: head.Number, Hash: head.Hash, Td: head.Td}
	fp.lastAnnounced = n
	p.lock.Unlock()
	f.checkUpdateStats(p, nil)
	if !f.requestTriggered {
		f.requestTriggered = true
		f.requestTrigger <- struct{}{}
	}
}

// peerHasBlock returns true if we can assume the peer knows the given block
// based on its announcements
func (f *lightFetcher) peerHasBlock(p *serverPeer, hash common.Hash, number uint64, hasState bool) bool {
	f.lock.Lock()
	defer f.lock.Unlock()

	fp := f.peers[p]
	if fp == nil || fp.root == nil {
		return false
	}

	if hasState {
		if fp.lastAnnounced == nil || fp.lastAnnounced.number > number+serverStateAvailable {
			return false
		}
	}

	if f.syncing {
		// always return true when syncing
		// false positives are acceptable, a more sophisticated condition can be implemented later
		return true
	}

	if number >= fp.root.number {
		// it is recent enough that if it is known, is should be in the peer's block tree
		return fp.nodeByHash[hash] != nil
	}
	f.chain.LockChain()
	defer f.chain.UnlockChain()
	// if it's older than the peer's block tree root but it's in the same canonical chain
	// as the root, we can still be sure the peer knows it
	//
	// when syncing, just check if it is part of the known chain, there is nothing better we
	// can do since we do not know the most recent block hash yet
	return rawdb.ReadCanonicalHash(f.handler.backend.chainDb, fp.root.number) == fp.root.hash && rawdb.ReadCanonicalHash(f.handler.backend.chainDb, number) == hash
}

// requestAmount calculates the amount of headers to be downloaded starting
// from a certain head backwards
func (f *lightFetcher) requestAmount(p *serverPeer, n *fetcherTreeNode) uint64 {
	amount := uint64(0)
	nn := n
	for nn != nil && !f.checkKnownNode(p, nn) {
		nn = nn.parent
		amount++
	}
	if nn == nil {
		amount = n.number
	}
	return amount
}

// requestedID tells if a certain reqID has been requested by the fetcher
func (f *lightFetcher) requestedID(reqID uint64) bool {
	f.reqMu.RLock()
	_, ok := f.requested[reqID]
	f.reqMu.RUnlock()
	return ok
}

// nextRequest selects the peer and announced head to be requested next, amount
// to be downloaded starting from the head backwards is also returned
func (f *lightFetcher) nextRequest() (*distReq, uint64, bool) {
	var (
		bestHash    common.Hash
		bestAmount  uint64
		bestTd      *big.Int
		bestSyncing bool
	)
	bestHash, bestAmount, bestTd, bestSyncing = f.findBestRequest()

	if bestTd == f.maxConfirmedTd {
		return nil, 0, false
	}

	var rq *distReq
	reqID := genReqID()
	if bestSyncing {
		rq = f.newFetcherDistReqForSync(bestHash)
	} else {
		rq = f.newFetcherDistReq(bestHash, reqID, bestAmount)
	}
	return rq, reqID, bestSyncing
}

// findBestRequest finds the best head to request that has been announced by but not yet requested from a known peer.
// It also returns the announced Td (which should be verified after fetching the head),
// the necessary amount to request and whether a downloader sync is necessary instead of a normal header request.
func (f *lightFetcher) findBestRequest() (bestHash common.Hash, bestAmount uint64, bestTd *big.Int, bestSyncing bool) {
	bestTd = f.maxConfirmedTd
	bestSyncing = false

	for p, fp := range f.peers {
		for hash, n := range fp.nodeByHash {
			if f.checkKnownNode(p, n) || n.requested {
				continue
			}
			// if ulc mode is disabled, isTrustedHash returns true
			amount := f.requestAmount(p, n)
			if (bestTd == nil || n.td.Cmp(bestTd) > 0 || amount < bestAmount) && (f.isTrustedHash(hash) || f.maxConfirmedTd.Int64() == 0) {
				bestHash = hash
				bestTd = n.td
				bestAmount = amount
				bestSyncing = fp.bestConfirmed == nil || fp.root == nil || !f.checkKnownNode(p, fp.root)
			}
		}
	}
	return
}

// isTrustedHash checks if the block can be trusted by the minimum trusted fraction.
func (f *lightFetcher) isTrustedHash(hash common.Hash) bool {
	// If ultra light cliet mode is disabled, trust all hashes
	if f.handler.ulc == nil {
		return true
	}
	// Ultra light enabled, only trust after enough confirmations
	var agreed int
	for peer, info := range f.peers {
		if peer.trusted && info.nodeByHash[hash] != nil {
			agreed++
		}
	}
	return 100*agreed/len(f.handler.ulc.keys) >= f.handler.ulc.fraction
}

func (f *lightFetcher) newFetcherDistReqForSync(bestHash common.Hash) *distReq {
	return &distReq{
		getCost: func(dp distPeer) uint64 {
			return 0
		},
		canSend: func(dp distPeer) bool {
			p := dp.(*serverPeer)
			f.lock.Lock()
			defer f.lock.Unlock()

			if p.onlyAnnounce {
				return false
			}
			fp := f.peers[p]
			return fp != nil && fp.nodeByHash[bestHash] != nil
		},
		request: func(dp distPeer) func() {
			if f.handler.ulc != nil {
				// Keep last trusted header before sync
				f.setLastTrustedHeader(f.chain.CurrentHeader())
			}
			go func() {
				p := dp.(*serverPeer)
				p.Log().Debug("Synchronisation started")
				f.handler.synchronise(p)
				f.syncDone <- p
			}()
			return nil
		},
	}
}

// newFetcherDistReq creates a new request for the distributor.
func (f *lightFetcher) newFetcherDistReq(bestHash common.Hash, reqID uint64, bestAmount uint64) *distReq {
	return &distReq{
		getCost: func(dp distPeer) uint64 {
			p := dp.(*serverPeer)
			return p.getRequestCost(GetBlockHeadersMsg, int(bestAmount))
		},
		canSend: func(dp distPeer) bool {
			p := dp.(*serverPeer)
			f.lock.Lock()
			defer f.lock.Unlock()

			if p.onlyAnnounce {
				return false
			}
			fp := f.peers[p]
			if fp == nil {
				return false
			}
			n := fp.nodeByHash[bestHash]
			return n != nil && !n.requested
		},
		request: func(dp distPeer) func() {
			p := dp.(*serverPeer)
			f.lock.Lock()
			fp := f.peers[p]
			if fp != nil {
				n := fp.nodeByHash[bestHash]
				if n != nil {
					n.requested = true
				}
			}
			f.lock.Unlock()

			cost := p.getRequestCost(GetBlockHeadersMsg, int(bestAmount))
			p.fcServer.QueuedRequest(reqID, cost)
			f.reqMu.Lock()
			f.requested[reqID] = fetchRequest{hash: bestHash, amount: bestAmount, peer: p, sent: mclock.Now()}
			f.reqMu.Unlock()
			go func() {
				time.Sleep(hardRequestTimeout)
				f.timeoutChn <- reqID
			}()
			return func() { p.requestHeadersByHash(reqID, bestHash, int(bestAmount), 0, true) }
		},
	}
}

// deliverHeaders delivers header download request responses for processing
func (f *lightFetcher) deliverHeaders(peer *serverPeer, reqID uint64, headers []*types.Header) {
	f.deliverChn <- fetchResponse{reqID: reqID, headers: headers, peer: peer}
}

// processResponse processes header download request responses, returns true if successful
func (f *lightFetcher) processResponse(req fetchRequest, resp fetchResponse) bool {
	if uint64(len(resp.headers)) != req.amount || resp.headers[0].Hash() != req.hash {
		req.peer.Log().Debug("Response content mismatch", "requested", len(resp.headers), "reqfrom", resp.headers[0], "delivered", req.amount, "delfrom", req.hash)
		return false
	}
	headers := make([]*types.Header, req.amount)
	for i, header := range resp.headers {
		headers[int(req.amount)-1-i] = header
	}

	if _, err := f.chain.InsertHeaderChain(headers, 1); err != nil {
		if err == consensus.ErrFutureBlock {
			return true
		}
		log.Debug("Failed to insert header chain", "err", err)
		return false
	}
	tds := make([]*big.Int, len(headers))
	for i, header := range headers {
		td := f.chain.GetTd(header.Hash(), header.Number.Uint64())
		if td == nil {
			log.Debug("Total difficulty not found for header", "index", i+1, "number", header.Number, "hash", header.Hash())
			return false
		}
		tds[i] = td
	}
	f.newHeaders(headers, tds)
	return true
}

// newHeaders updates the block trees of all active peers according to a newly
// downloaded and validated batch or headers
func (f *lightFetcher) newHeaders(headers []*types.Header, tds []*big.Int) {
	var maxTd *big.Int

	for p, fp := range f.peers {
		if !f.checkAnnouncedHeaders(fp, headers, tds) {
			p.Log().Debug("Inconsistent announcement")
			go f.handler.removePeer(p.id)
		}
		if fp.confirmedTd != nil && (maxTd == nil || maxTd.Cmp(fp.confirmedTd) > 0) {
			maxTd = fp.confirmedTd
		}
	}

	if maxTd != nil {
		f.updateMaxConfirmedTd(maxTd)
	}
}

// checkAnnouncedHeaders updates peer's block tree if necessary after validating
// a batch of headers. It searches for the latest header in the batch that has a
// matching tree node (if any), and if it has not been marked as known already,
// sets it and its parents to known (even those which are older than the currently
// validated ones). Return value shows if all hashes, numbers and Tds matched
// correctly to the announced values (otherwise the peer should be dropped).
func (f *lightFetcher) checkAnnouncedHeaders(fp *fetcherPeerInfo, headers []*types.Header, tds []*big.Int) bool {
	var (
		n      *fetcherTreeNode
		header *types.Header
		td     *big.Int
	)

	for i := len(headers) - 1; ; i-- {
		if i < 0 {
			if n == nil {
				// no more headers and nothing to match
				return true
			}
			// we ran out of recently delivered headers but have not reached a node known by this peer yet, continue matching
			hash, number := header.ParentHash, header.Number.Uint64()-1
			td = f.chain.GetTd(hash, number)
			header = f.chain.GetHeader(hash, number)
			if header == nil || td == nil {
				log.Error("Missing parent of validated header", "hash", hash, "number", number)
				return false
			}
		} else {
			header = headers[i]
			td = tds[i]
		}
		hash := header.Hash()
		number := header.Number.Uint64()
		if n == nil {
			n = fp.nodeByHash[hash]
		}
		if n != nil {
			if n.td == nil {
				// node was unannounced
				if nn := fp.nodeByHash[hash]; nn != nil {
					// if there was already a node with the same hash, continue there and drop this one
					nn.children = append(nn.children, n.children...)
					n.children = nil
					fp.deleteNode(n)
					n = nn
				} else {
					n.hash = hash
					n.td = td
					fp.nodeByHash[hash] = n
				}
			}
			// check if it matches the header
			if n.hash != hash || n.number != number || n.td.Cmp(td) != 0 {
				// peer has previously made an invalid announcement
				return false
			}
			if n.known {
				// we reached a known node that matched our expectations, return with success
				return true
			}
			n.known = true
			if fp.confirmedTd == nil || td.Cmp(fp.confirmedTd) > 0 {
				fp.confirmedTd = td
				fp.bestConfirmed = n
			}
			n = n.parent
			if n == nil {
				return true
			}
		}
	}
}

// checkSyncedHeaders updates peer's block tree after synchronisation by marking
// downloaded headers as known. If none of the announced headers are found after
// syncing, the peer is dropped.
func (f *lightFetcher) checkSyncedHeaders(p *serverPeer) {
	fp := f.peers[p]
	if fp == nil {
		p.Log().Debug("Unknown peer to check sync headers")
		return
	}
	var (
		node = fp.lastAnnounced
		td   *big.Int
	)
	if f.handler.ulc != nil {
		// Roll back untrusted blocks
		h, unapproved := f.lastTrustedTreeNode(p)
		f.chain.Rollback(unapproved)
		node = fp.nodeByHash[h.Hash()]
	}
	// Find last valid block
	for node != nil {
		if td = f.chain.GetTd(node.hash, node.number); td != nil {
			break
		}
		node = node.parent
	}
	// Now node is the latest downloaded/approved header after syncing
	if node == nil {
		p.Log().Debug("Synchronisation failed")
		go f.handler.removePeer(p.id)
		return
	}
	header := f.chain.GetHeader(node.hash, node.number)
	f.newHeaders([]*types.Header{header}, []*big.Int{td})
}

// lastTrustedTreeNode return last approved treeNode and a list of unapproved hashes
func (f *lightFetcher) lastTrustedTreeNode(p *serverPeer) (*types.Header, []common.Hash) {
	unapprovedHashes := make([]common.Hash, 0)
	current := f.chain.CurrentHeader()

	if f.lastTrustedHeader == nil {
		return current, unapprovedHashes
	}

	canonical := f.chain.CurrentHeader()
	if canonical.Number.Uint64() > f.lastTrustedHeader.Number.Uint64() {
		canonical = f.chain.GetHeaderByNumber(f.lastTrustedHeader.Number.Uint64())
	}
	commonAncestor := rawdb.FindCommonAncestor(f.handler.backend.chainDb, canonical, f.lastTrustedHeader)
	if commonAncestor == nil {
		log.Error("Common ancestor of last trusted header and canonical header is nil", "canonical hash", canonical.Hash(), "trusted hash", f.lastTrustedHeader.Hash())
		return current, unapprovedHashes
	}

	for current.Hash() == commonAncestor.Hash() {
		if f.isTrustedHash(current.Hash()) {
			break
		}
		unapprovedHashes = append(unapprovedHashes, current.Hash())
		current = f.chain.GetHeader(current.ParentHash, current.Number.Uint64()-1)
	}
	return current, unapprovedHashes
}

func (f *lightFetcher) setLastTrustedHeader(h *types.Header) {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.lastTrustedHeader = h
}

// checkKnownNode checks if a block tree node is known (downloaded and validated)
// If it was not known previously but found in the database, sets its known flag
func (f *lightFetcher) checkKnownNode(p *serverPeer, n *fetcherTreeNode) bool {
	if n.known {
		return true
	}
	td := f.chain.GetTd(n.hash, n.number)
	if td == nil {
		return false
	}
	header := f.chain.GetHeader(n.hash, n.number)
	// check the availability of both header and td because reads are not protected by chain db mutex
	// Note: returning false is always safe here
	if header == nil {
		return false
	}

	fp := f.peers[p]
	if fp == nil {
		p.Log().Debug("Unknown peer to check known nodes")
		return false
	}
	if !f.checkAnnouncedHeaders(fp, []*types.Header{header}, []*big.Int{td}) {
		p.Log().Debug("Inconsistent announcement")
		go f.handler.removePeer(p.id)
	}
	if fp.confirmedTd != nil {
		f.updateMaxConfirmedTd(fp.confirmedTd)
	}
	return n.known
}

// deleteNode deletes a node and its child subtrees from a peer's block tree
func (fp *fetcherPeerInfo) deleteNode(n *fetcherTreeNode) {
	if n.parent != nil {
		for i, nn := range n.parent.children {
			if nn == n {
				n.parent.children = append(n.parent.children[:i], n.parent.children[i+1:]...)
				break
			}
		}
	}
	for {
		if n.td != nil {
			delete(fp.nodeByHash, n.hash)
		}
		fp.nodeCnt--
		if len(n.children) == 0 {
			return
		}
		for i, nn := range n.children {
			if i == 0 {
				n = nn
			} else {
				fp.deleteNode(nn)
			}
		}
	}
}

// updateStatsEntry items form a linked list that is expanded with a new item every time a new head with a higher Td
// than the previous one has been downloaded and validated. The list contains a series of maximum confirmed Td values
// and the time these values have been confirmed, both increasing monotonically. A maximum confirmed Td is calculated
// both globally for all peers and also for each individual peer (meaning that the given peer has announced the head
// and it has also been downloaded from any peer, either before or after the given announcement).
// The linked list has a global tail where new confirmed Td entries are added and a separate head for each peer,
// pointing to the next Td entry that is higher than the peer's max confirmed Td (nil if it has already confirmed
// the current global head).
type updateStatsEntry struct {
	time mclock.AbsTime
	td   *big.Int
	next *updateStatsEntry
}

// updateMaxConfirmedTd updates the block delay statistics of active peers. Whenever a new highest Td is confirmed,
// adds it to the end of a linked list together with the time it has been confirmed. Then checks which peers have
// already confirmed a head with the same or higher Td (which counts as zero block delay) and updates their statistics.
// Those who have not confirmed such a head by now will be updated by a subsequent checkUpdateStats call with a
// positive block delay value.
func (f *lightFetcher) updateMaxConfirmedTd(td *big.Int) {
	if f.maxConfirmedTd == nil || td.Cmp(f.maxConfirmedTd) > 0 {
		f.maxConfirmedTd = td
		newEntry := &updateStatsEntry{
			time: mclock.Now(),
			td:   td,
		}
		if f.lastUpdateStats != nil {
			f.lastUpdateStats.next = newEntry
		}

		f.lastUpdateStats = newEntry
		for p := range f.peers {
			f.checkUpdateStats(p, newEntry)
		}
	}
}

// checkUpdateStats checks those peers who have not confirmed a certain highest Td (or a larger one) by the time it
// has been confirmed by another peer. If they have confirmed such a head by now, their stats are updated with the
// block delay which is (this peer's confirmation time)-(first confirmation time). After blockDelayTimeout has passed,
// the stats are updated with blockDelayTimeout value. In either case, the confirmed or timed out updateStatsEntry
// items are removed from the head of the linked list.
// If a new entry has been added to the global tail, it is passed as a parameter here even though this function
// assumes that it has already been added, so that if the peer's list is empty (all heads confirmed, head is nil),
// it can set the new head to newEntry.
func (f *lightFetcher) checkUpdateStats(p *serverPeer, newEntry *updateStatsEntry) {
	now := mclock.Now()
	fp := f.peers[p]
	if fp == nil {
		p.Log().Debug("Unknown peer to check update stats")
		return
	}

	if newEntry != nil && fp.firstUpdateStats == nil {
		fp.firstUpdateStats = newEntry
	}
	for fp.firstUpdateStats != nil && fp.firstUpdateStats.time <= now-mclock.AbsTime(blockDelayTimeout) {
		f.handler.backend.serverPool.adjustBlockDelay(p.poolEntry, blockDelayTimeout)
		fp.firstUpdateStats = fp.firstUpdateStats.next
	}
	if fp.confirmedTd != nil {
		for fp.firstUpdateStats != nil && fp.firstUpdateStats.td.Cmp(fp.confirmedTd) <= 0 {
			f.handler.backend.serverPool.adjustBlockDelay(p.poolEntry, time.Duration(now-fp.firstUpdateStats.time))
			fp.firstUpdateStats = fp.firstUpdateStats.next
		}
	}
}
