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

// Package les implements the Light Ethereum Subprotocol.
package les

import (
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

const (
	blockDelayTimeout = time.Second * 10 // timeout for a peer to announce a head that has already been confirmed by others
	maxNodeCount      = 20               // maximum number of fetcherTreeNode entries remembered for each peer
)

// lightFetcher
type lightFetcher struct {
	pm    *ProtocolManager
	odr   *LesOdr
	chain *light.LightChain

	maxConfirmedTd  *big.Int
	peers           map[*peer]*fetcherPeerInfo
	lastUpdateStats *updateStatsEntry

	lock       sync.Mutex // qwerqwerqwe
	deliverChn chan fetchResponse
	reqMu      sync.RWMutex
	requested  map[uint64]fetchRequest
	timeoutChn chan uint64
	requestChn chan bool // true if initiated from outside
	syncing    bool
	syncDone   chan *peer
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
	peer    *peer
	sent    mclock.AbsTime
	timeout bool
}

// fetchResponse represents a header download response
type fetchResponse struct {
	reqID   uint64
	headers []*types.Header
	peer    *peer
}

// newLightFetcher creates a new light fetcher
func newLightFetcher(pm *ProtocolManager) *lightFetcher {
	f := &lightFetcher{
		pm:             pm,
		chain:          pm.blockchain.(*light.LightChain),
		odr:            pm.odr,
		peers:          make(map[*peer]*fetcherPeerInfo),
		deliverChn:     make(chan fetchResponse, 100),
		requested:      make(map[uint64]fetchRequest),
		timeoutChn:     make(chan uint64),
		requestChn:     make(chan bool, 100),
		syncDone:       make(chan *peer),
		maxConfirmedTd: big.NewInt(0),
	}
	go f.syncLoop()
	return f
}

// syncLoop is the main event loop of the light fetcher
func (f *lightFetcher) syncLoop() {
	f.pm.wg.Add(1)
	defer f.pm.wg.Done()

	requesting := false
	for {
		select {
		case <-f.pm.quitSync:
			return
		// when a new announce is received, request loop keeps running until
		// no further requests are necessary or possible
		case newAnnounce := <-f.requestChn:
			f.lock.Lock()
			s := requesting
			requesting = false
			if !f.syncing && !(newAnnounce && s) {
				reqID := getNextReqID()
				if peer, node, amount, retry := f.nextRequest(reqID); node != nil {
					requesting = true
					if reqID, ok := f.request(peer, reqID, node, amount); ok {
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
							f.requestChn <- false
						}()
					}
				} else {
					if retry {
						requesting = true
						go func() {
							time.Sleep(time.Millisecond * 100)
							f.requestChn <- false
						}()
					}
				}
			}
			f.lock.Unlock()
		case reqID := <-f.timeoutChn:
			f.reqMu.Lock()
			req, ok := f.requested[reqID]
			if ok {
				delete(f.requested, reqID)
			}
			f.reqMu.Unlock()
			if ok {
				f.pm.serverPool.adjustResponseTime(req.peer.poolEntry, time.Duration(mclock.Now()-req.sent), true)
				glog.V(logger.Debug).Infof("hard timeout by peer %v", req.peer.id)
				go f.pm.removePeer(req.peer.id)
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
				f.pm.serverPool.adjustResponseTime(req.peer.poolEntry, time.Duration(mclock.Now()-req.sent), req.timeout)
			}
			f.lock.Lock()
			if !ok || !(f.syncing || f.processResponse(req, resp)) {
				glog.V(logger.Debug).Infof("failed processing response by peer %v", resp.peer.id)
				go f.pm.removePeer(resp.peer.id)
			}
			f.lock.Unlock()
		case p := <-f.syncDone:
			f.lock.Lock()
			glog.V(logger.Debug).Infof("done synchronising with peer %v", p.id)
			f.checkSyncedHeaders(p)
			f.syncing = false
			f.lock.Unlock()
		}
	}
}

// addPeer adds a new peer to the fetcher's peer set
func (f *lightFetcher) addPeer(p *peer) {
	p.lock.Lock()
	p.hasBlock = func(hash common.Hash, number uint64) bool {
		return f.peerHasBlock(p, hash, number)
	}
	p.lock.Unlock()

	f.lock.Lock()
	defer f.lock.Unlock()

	f.peers[p] = &fetcherPeerInfo{nodeByHash: make(map[common.Hash]*fetcherTreeNode)}
}

// removePeer removes a new peer from the fetcher's peer set
func (f *lightFetcher) removePeer(p *peer) {
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
func (f *lightFetcher) announce(p *peer, head *announceData) {
	f.lock.Lock()
	defer f.lock.Unlock()
	glog.V(logger.Debug).Infof("received announce from peer %v  #%d  %016x  reorg: %d", p.id, head.Number, head.Hash[:8], head.ReorgDepth)

	fp := f.peers[p]
	if fp == nil {
		glog.V(logger.Debug).Infof("announce: unknown peer")
		return
	}

	if fp.lastAnnounced != nil && head.Td.Cmp(fp.lastAnnounced.td) <= 0 {
		// announced tds should be strictly monotonic
		glog.V(logger.Debug).Infof("non-monotonic Td from peer %v", p.id)
		go f.pm.removePeer(p.id)
		return
	}

	n := fp.lastAnnounced
	for i := uint64(0); i < head.ReorgDepth; i++ {
		if n == nil {
			break
		}
		n = n.parent
	}
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
				if core.GetCanonicalHash(f.pm.chainDb, nn.number) == nn.hash {
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
	p.headInfo = head
	fp.lastAnnounced = n
	p.lock.Unlock()
	f.checkUpdateStats(p, nil)
	f.requestChn <- true
}

// peerHasBlock returns true if we can assume the peer knows the given block
// based on its announcements
func (f *lightFetcher) peerHasBlock(p *peer, hash common.Hash, number uint64) bool {
	f.lock.Lock()
	defer f.lock.Unlock()

	fp := f.peers[p]
	if fp == nil || fp.root == nil {
		return false
	}

	if number >= fp.root.number {
		// it is recent enough that if it is known, is should be in the peer's block tree
		return fp.nodeByHash[hash] != nil
	}
	f.chain.LockChain()
	defer f.chain.UnlockChain()
	// if it's older than the peer's block tree root but it's in the same canonical chain
	// than the root, we can still be sure the peer knows it
	return core.GetCanonicalHash(f.pm.chainDb, fp.root.number) == fp.root.hash && core.GetCanonicalHash(f.pm.chainDb, number) == hash
}

// request initiates a header download request from a certain peer
func (f *lightFetcher) request(p *peer, reqID uint64, n *fetcherTreeNode, amount uint64) (uint64, bool) {
	fp := f.peers[p]
	if fp == nil {
		glog.V(logger.Debug).Infof("request: unknown peer")
		p.fcServer.DeassignRequest(reqID)
		return 0, false
	}
	if fp.bestConfirmed == nil || fp.root == nil || !f.checkKnownNode(p, fp.root) {
		f.syncing = true
		go func() {
			glog.V(logger.Debug).Infof("synchronising with peer %v", p.id)
			f.pm.synchronise(p)
			f.syncDone <- p
		}()
		p.fcServer.DeassignRequest(reqID)
		return 0, false
	}

	n.requested = true
	cost := p.GetRequestCost(GetBlockHeadersMsg, int(amount))
	p.fcServer.SendRequest(reqID, cost)
	f.reqMu.Lock()
	f.requested[reqID] = fetchRequest{hash: n.hash, amount: amount, peer: p, sent: mclock.Now()}
	f.reqMu.Unlock()
	go p.RequestHeadersByHash(reqID, cost, n.hash, int(amount), 0, true)
	go func() {
		time.Sleep(hardRequestTimeout)
		f.timeoutChn <- reqID
	}()
	return reqID, true
}

// requestAmount calculates the amount of headers to be downloaded starting
// from a certain head backwards
func (f *lightFetcher) requestAmount(p *peer, n *fetcherTreeNode) uint64 {
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
func (f *lightFetcher) nextRequest(reqID uint64) (*peer, *fetcherTreeNode, uint64, bool) {
	var (
		bestHash   common.Hash
		bestAmount uint64
	)
	bestTd := f.maxConfirmedTd

	for p, fp := range f.peers {
		for hash, n := range fp.nodeByHash {
			if !f.checkKnownNode(p, n) && !n.requested && (bestTd == nil || n.td.Cmp(bestTd) >= 0) {
				amount := f.requestAmount(p, n)
				if bestTd == nil || n.td.Cmp(bestTd) > 0 || amount < bestAmount {
					bestHash = hash
					bestAmount = amount
					bestTd = n.td
				}
			}
		}
	}
	if bestTd == f.maxConfirmedTd {
		return nil, nil, 0, false
	}

	peer, _, locked := f.pm.serverPool.selectPeer(reqID, func(p *peer) (bool, time.Duration) {
		fp := f.peers[p]
		if fp == nil || fp.nodeByHash[bestHash] == nil {
			return false, 0
		}
		return true, p.fcServer.CanSend(p.GetRequestCost(GetBlockHeadersMsg, int(bestAmount)))
	})
	if !locked {
		return nil, nil, 0, true
	}
	var node *fetcherTreeNode
	if peer != nil {
		node = f.peers[peer].nodeByHash[bestHash]
	}
	return peer, node, bestAmount, false
}

// deliverHeaders delivers header download request responses for processing
func (f *lightFetcher) deliverHeaders(peer *peer, reqID uint64, headers []*types.Header) {
	f.deliverChn <- fetchResponse{reqID: reqID, headers: headers, peer: peer}
}

// processResponse processes header download request responses, returns true if successful
func (f *lightFetcher) processResponse(req fetchRequest, resp fetchResponse) bool {
	if uint64(len(resp.headers)) != req.amount || resp.headers[0].Hash() != req.hash {
		glog.V(logger.Debug).Infof("response mismatch %v %016x != %v %016x", len(resp.headers), resp.headers[0].Hash().Bytes()[:8], req.amount, req.hash[:8])
		return false
	}
	headers := make([]*types.Header, req.amount)
	for i, header := range resp.headers {
		headers[int(req.amount)-1-i] = header
	}
	if _, err := f.chain.InsertHeaderChain(headers, 1); err != nil {
		if err == core.BlockFutureErr {
			return true
		}
		glog.V(logger.Debug).Infof("InsertHeaderChain error: %v", err)
		return false
	}
	tds := make([]*big.Int, len(headers))
	for i, header := range headers {
		td := f.chain.GetTd(header.Hash(), header.Number.Uint64())
		if td == nil {
			glog.V(logger.Debug).Infof("TD not found for header %v of %v", i+1, len(headers))
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
			glog.V(logger.Debug).Infof("announce inconsistency by peer %v", p.id)
			go f.pm.removePeer(p.id)
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
			td = f.chain.GetTd(header.ParentHash, header.Number.Uint64()-1)
			header = f.chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)
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
func (f *lightFetcher) checkSyncedHeaders(p *peer) {
	fp := f.peers[p]
	if fp == nil {
		glog.V(logger.Debug).Infof("checkSyncedHeaders: unknown peer")
		return
	}
	n := fp.lastAnnounced
	var td *big.Int
	for n != nil {
		if td = f.chain.GetTd(n.hash, n.number); td != nil {
			break
		}
		n = n.parent
	}
	// now n is the latest downloaded header after syncing
	if n == nil {
		glog.V(logger.Debug).Infof("synchronisation failed with peer %v", p.id)
		go f.pm.removePeer(p.id)
	} else {
		header := f.chain.GetHeader(n.hash, n.number)
		f.newHeaders([]*types.Header{header}, []*big.Int{td})
	}
}

// checkKnownNode checks if a block tree node is known (downloaded and validated)
// If it was not known previously but found in the database, sets its known flag
func (f *lightFetcher) checkKnownNode(p *peer, n *fetcherTreeNode) bool {
	if n.known {
		return true
	}
	td := f.chain.GetTd(n.hash, n.number)
	if td == nil {
		return false
	}

	fp := f.peers[p]
	if fp == nil {
		glog.V(logger.Debug).Infof("checkKnownNode: unknown peer")
		return false
	}
	header := f.chain.GetHeader(n.hash, n.number)
	if !f.checkAnnouncedHeaders(fp, []*types.Header{header}, []*big.Int{td}) {
		glog.V(logger.Debug).Infof("announce inconsistency by peer %v", p.id)
		go f.pm.removePeer(p.id)
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
		for p, _ := range f.peers {
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
func (f *lightFetcher) checkUpdateStats(p *peer, newEntry *updateStatsEntry) {
	now := mclock.Now()
	fp := f.peers[p]
	if fp == nil {
		glog.V(logger.Debug).Infof("checkUpdateStats: unknown peer")
		return
	}
	if newEntry != nil && fp.firstUpdateStats == nil {
		fp.firstUpdateStats = newEntry
	}
	for fp.firstUpdateStats != nil && fp.firstUpdateStats.time <= now-mclock.AbsTime(blockDelayTimeout) {
		f.pm.serverPool.adjustBlockDelay(p.poolEntry, blockDelayTimeout)
		fp.firstUpdateStats = fp.firstUpdateStats.next
	}
	if fp.confirmedTd != nil {
		for fp.firstUpdateStats != nil && fp.firstUpdateStats.td.Cmp(fp.confirmedTd) <= 0 {
			f.pm.serverPool.adjustBlockDelay(p.poolEntry, time.Duration(now-fp.firstUpdateStats.time))
			fp.firstUpdateStats = fp.firstUpdateStats.next
		}
	}
}
