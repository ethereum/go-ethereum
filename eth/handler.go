// Copyright 2015 The go-ethereum Authors
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
	"cmp"
	crand "crypto/rand"
	"errors"
	"maps"
	"math"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dchest/siphash"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/eth/fetcher"
	"github.com/ethereum/go-ethereum/eth/protocols/eth"
	"github.com/ethereum/go-ethereum/eth/protocols/snap"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

const (
	// txChanSize is the size of channel listening to NewTxsEvent.
	// The number is referenced from the size of tx pool.
	txChanSize = 4096

	// chainHeadChanSize is the size of channel listening to ChainHeadEvent.
	chainHeadChanSize = 128

	// txMaxBroadcastSize is the max size of a transaction that will be broadcasted.
	// All transactions with a higher size will be announced and need to be fetched
	// by the peer.
	txMaxBroadcastSize = 4096
)

var syncChallengeTimeout = 15 * time.Second // Time allowance for a node to reply to the sync progress challenge

// txPool defines the methods needed from a transaction pool implementation to
// support all the operations needed by the Ethereum chain protocols.
type txPool interface {
	// Has returns an indicator whether txpool has a transaction
	// cached with the given hash.
	Has(hash common.Hash) bool

	// Get retrieves the transaction from local txpool with given
	// tx hash.
	Get(hash common.Hash) *types.Transaction

	// GetRLP retrieves the RLP-encoded transaction from local txpool
	// with given tx hash.
	GetRLP(hash common.Hash) []byte

	// GetMetadata returns the transaction type and transaction size with the
	// given transaction hash.
	GetMetadata(hash common.Hash) *txpool.TxMetadata

	// Add should add the given transactions to the pool.
	Add(txs []*types.Transaction, sync bool) []error

	// Pending should return pending transactions.
	// The slice should be modifiable by the caller.
	Pending(filter txpool.PendingFilter) map[common.Address][]*txpool.LazyTransaction

	// SubscribeTransactions subscribes to new transaction events. The subscriber
	// can decide whether to receive notifications only for newly seen transactions
	// or also for reorged out ones.
	SubscribeTransactions(ch chan<- core.NewTxsEvent, reorgs bool) event.Subscription
}

// handlerConfig is the collection of initialization parameters to create a full
// node network handler.
type handlerConfig struct {
	NodeID         enode.ID               // P2P node ID used for tx propagation topology
	Database       ethdb.Database         // Database for direct sync insertions
	Chain          *core.BlockChain       // Blockchain to serve data from
	TxPool         txPool                 // Transaction pool to propagate from
	Network        uint64                 // Network identifier to advertise
	Sync           ethconfig.SyncMode     // Whether to snap or full sync
	BloomCache     uint64                 // Megabytes to alloc for snap sync bloom
	EventMux       *event.TypeMux         // Legacy event mux, deprecate for `feed`
	RequiredBlocks map[uint64]common.Hash // Hard coded map of required block hashes for sync challenges
}

type handler struct {
	nodeID    enode.ID
	networkID uint64

	snapSync atomic.Bool // Flag whether snap sync is enabled (gets disabled if we already have blocks)
	synced   atomic.Bool // Flag whether we're considered synchronised (enables transaction processing)

	database ethdb.Database
	txpool   txPool
	chain    *core.BlockChain
	maxPeers int

	downloader     *downloader.Downloader
	txFetcher      *fetcher.TxFetcher
	peers          *peerSet
	txBroadcastKey [16]byte

	eventMux   *event.TypeMux
	txsCh      chan core.NewTxsEvent
	txsSub     event.Subscription
	blockRange *blockRangeState

	requiredBlocks map[uint64]common.Hash

	// channels for fetcher, syncer, txsyncLoop
	quitSync chan struct{}

	wg sync.WaitGroup

	handlerStartCh chan struct{}
	handlerDoneCh  chan struct{}
}

// newHandler returns a handler for all Ethereum chain management protocol.
func newHandler(config *handlerConfig) (*handler, error) {
	// Create the protocol manager with the base fields
	if config.EventMux == nil {
		config.EventMux = new(event.TypeMux) // Nicety initialization for tests
	}
	h := &handler{
		nodeID:         config.NodeID,
		networkID:      config.Network,
		eventMux:       config.EventMux,
		database:       config.Database,
		txpool:         config.TxPool,
		chain:          config.Chain,
		peers:          newPeerSet(),
		txBroadcastKey: newBroadcastChoiceKey(),
		requiredBlocks: config.RequiredBlocks,
		quitSync:       make(chan struct{}),
		handlerDoneCh:  make(chan struct{}),
		handlerStartCh: make(chan struct{}),
	}
	if config.Sync == ethconfig.FullSync {
		// The database seems empty as the current block is the genesis. Yet the snap
		// block is ahead, so snap sync was enabled for this node at a certain point.
		// The scenarios where this can happen is
		// * if the user manually (or via a bad block) rolled back a snap sync node
		//   below the sync point.
		// * the last snap sync is not finished while user specifies a full sync this
		//   time. But we don't have any recent state for full sync.
		// In these cases however it's safe to reenable snap sync.
		fullBlock, snapBlock := h.chain.CurrentBlock(), h.chain.CurrentSnapBlock()
		if fullBlock.Number.Uint64() == 0 && snapBlock.Number.Uint64() > 0 {
			h.snapSync.Store(true)
			log.Warn("Switch sync mode from full sync to snap sync", "reason", "snap sync incomplete")
		} else if !h.chain.HasState(fullBlock.Root) {
			h.snapSync.Store(true)
			log.Warn("Switch sync mode from full sync to snap sync", "reason", "head state missing")
		}
	} else {
		head := h.chain.CurrentBlock()
		if head.Number.Uint64() > 0 && h.chain.HasState(head.Root) {
			log.Info("Switch sync mode from snap sync to full sync", "reason", "snap sync complete")
		} else {
			// If snap sync was requested and our database is empty, grant it
			h.snapSync.Store(true)
			log.Info("Enabled snap sync", "head", head.Number, "hash", head.Hash())
		}
	}
	// If snap sync is requested but snapshots are disabled, fail loudly
	if h.snapSync.Load() && (config.Chain.Snapshots() == nil && config.Chain.TrieDB().Scheme() == rawdb.HashScheme) {
		return nil, errors.New("snap sync not supported with snapshots disabled")
	}
	// Construct the downloader (long sync)
	h.downloader = downloader.New(config.Database, h.eventMux, h.chain, h.removePeer, h.enableSyncedFeatures)

	fetchTx := func(peer string, hashes []common.Hash) error {
		p := h.peers.peer(peer)
		if p == nil {
			return errors.New("unknown peer")
		}
		return p.RequestTxs(hashes)
	}
	addTxs := func(txs []*types.Transaction) []error {
		return h.txpool.Add(txs, false)
	}
	h.txFetcher = fetcher.NewTxFetcher(h.txpool.Has, addTxs, fetchTx, h.removePeer)
	return h, nil
}

// protoTracker tracks the number of active protocol handlers.
func (h *handler) protoTracker() {
	defer h.wg.Done()
	var active int
	for {
		select {
		case <-h.handlerStartCh:
			active++
		case <-h.handlerDoneCh:
			active--
		case <-h.quitSync:
			// Wait for all active handlers to finish.
			for ; active > 0; active-- {
				<-h.handlerDoneCh
			}
			return
		}
	}
}

// incHandlers signals to increment the number of active handlers if not
// quitting.
func (h *handler) incHandlers() bool {
	select {
	case h.handlerStartCh <- struct{}{}:
		return true
	case <-h.quitSync:
		return false
	}
}

// decHandlers signals to decrement the number of active handlers.
func (h *handler) decHandlers() {
	h.handlerDoneCh <- struct{}{}
}

// runEthPeer registers an eth peer into the joint eth/snap peerset, adds it to
// various subsystems and starts handling messages.
func (h *handler) runEthPeer(peer *eth.Peer, handler eth.Handler) error {
	if !h.incHandlers() {
		return p2p.DiscQuitting
	}
	defer h.decHandlers()

	// If the peer has a `snap` extension, wait for it to connect so we can have
	// a uniform initialization/teardown mechanism
	snap, err := h.peers.waitSnapExtension(peer)
	if err != nil {
		peer.Log().Error("Snapshot extension barrier failed", "err", err)
		return err
	}

	// Execute the Ethereum handshake
	if err := peer.Handshake(h.networkID, h.chain, h.blockRange.currentRange()); err != nil {
		peer.Log().Debug("Ethereum handshake failed", "err", err)
		return err
	}
	reject := false // reserved peer slots
	if h.snapSync.Load() {
		if snap == nil {
			// If we are running snap-sync, we want to reserve roughly half the peer
			// slots for peers supporting the snap protocol.
			// The logic here is; we only allow up to 5 more non-snap peers than snap-peers.
			if all, snp := h.peers.len(), h.peers.snapLen(); all-snp > snp+5 {
				reject = true
			}
		}
	}
	// Ignore maxPeers if this is a trusted peer
	if !peer.Peer.Info().Network.Trusted {
		if reject || h.peers.len() >= h.maxPeers {
			return p2p.DiscTooManyPeers
		}
	}
	peer.Log().Debug("Ethereum peer connected", "name", peer.Name())

	// Register the peer locally
	if err := h.peers.registerPeer(peer, snap); err != nil {
		peer.Log().Error("Ethereum peer registration failed", "err", err)
		return err
	}
	defer h.unregisterPeer(peer.ID())

	p := h.peers.peer(peer.ID())
	if p == nil {
		return errors.New("peer dropped during handling")
	}
	// Register the peer in the downloader. If the downloader considers it banned, we disconnect
	if err := h.downloader.RegisterPeer(peer.ID(), peer.Version(), peer); err != nil {
		peer.Log().Error("Failed to register peer in eth syncer", "err", err)
		return err
	}
	if snap != nil {
		if err := h.downloader.SnapSyncer.Register(snap); err != nil {
			peer.Log().Error("Failed to register peer in snap syncer", "err", err)
			return err
		}
	}
	// Propagate existing transactions. new transactions appearing
	// after this will be sent via broadcasts.
	h.syncTransactions(peer)

	// Create a notification channel for pending requests if the peer goes down
	dead := make(chan struct{})
	defer close(dead)

	// If we have any explicit peer required block hashes, request them
	for number, hash := range h.requiredBlocks {
		resCh := make(chan *eth.Response)

		req, err := peer.RequestHeadersByNumber(number, 1, 0, false, resCh)
		if err != nil {
			return err
		}
		go func(number uint64, hash common.Hash, req *eth.Request) {
			// Ensure the request gets cancelled in case of error/drop
			defer req.Close()

			timeout := time.NewTimer(syncChallengeTimeout)
			defer timeout.Stop()

			select {
			case res := <-resCh:
				headers := ([]*types.Header)(*res.Res.(*eth.BlockHeadersRequest))
				if len(headers) == 0 {
					// Required blocks are allowed to be missing if the remote
					// node is not yet synced
					res.Done <- nil
					return
				}
				// Validate the header and either drop the peer or continue
				if len(headers) > 1 {
					res.Done <- errors.New("too many headers in required block response")
					return
				}
				if headers[0].Number.Uint64() != number || headers[0].Hash() != hash {
					peer.Log().Info("Required block mismatch, dropping peer", "number", number, "hash", headers[0].Hash(), "want", hash)
					res.Done <- errors.New("required block mismatch")
					return
				}
				peer.Log().Debug("Peer required block verified", "number", number, "hash", hash)
				res.Done <- nil
			case <-timeout.C:
				peer.Log().Warn("Required block challenge timed out, dropping", "addr", peer.RemoteAddr(), "type", peer.Name())
				h.removePeer(peer.ID())
			case <-dead:
				// Peer handler terminated, abort all goroutines
			}
		}(number, hash, req)
	}
	// Handle incoming messages until the connection is torn down
	return handler(peer)
}

// runSnapExtension registers a `snap` peer into the joint eth/snap peerset and
// starts handling inbound messages. As `snap` is only a satellite protocol to
// `eth`, all subsystem registrations and lifecycle management will be done by
// the main `eth` handler to prevent strange races.
func (h *handler) runSnapExtension(peer *snap.Peer, handler snap.Handler) error {
	if !h.incHandlers() {
		return p2p.DiscQuitting
	}
	defer h.decHandlers()

	if err := h.peers.registerSnapExtension(peer); err != nil {
		if metrics.Enabled() {
			if peer.Inbound() {
				snap.IngressRegistrationErrorMeter.Mark(1)
			} else {
				snap.EgressRegistrationErrorMeter.Mark(1)
			}
		}
		peer.Log().Debug("Snapshot extension registration failed", "err", err)
		return err
	}
	return handler(peer)
}

// removePeer requests disconnection of a peer.
func (h *handler) removePeer(id string) {
	peer := h.peers.peer(id)
	if peer != nil {
		peer.Peer.Disconnect(p2p.DiscUselessPeer)
	}
}

// unregisterPeer removes a peer from the downloader, fetchers and main peer set.
func (h *handler) unregisterPeer(id string) {
	// Create a custom logger to avoid printing the entire id
	var logger log.Logger
	if len(id) < 16 {
		// Tests use short IDs, don't choke on them
		logger = log.New("peer", id)
	} else {
		logger = log.New("peer", id[:8])
	}
	// Abort if the peer does not exist
	peer := h.peers.peer(id)
	if peer == nil {
		logger.Warn("Ethereum peer removal failed", "err", errPeerNotRegistered)
		return
	}
	// Remove the `eth` peer if it exists
	logger.Debug("Removing Ethereum peer", "snap", peer.snapExt != nil)

	// Remove the `snap` extension if it exists
	if peer.snapExt != nil {
		h.downloader.SnapSyncer.Unregister(id)
	}
	h.downloader.UnregisterPeer(id)
	h.txFetcher.Drop(id)

	if err := h.peers.unregisterPeer(id); err != nil {
		logger.Error("Ethereum peer removal failed", "err", err)
	}
}

func (h *handler) Start(maxPeers int) {
	h.maxPeers = maxPeers

	// broadcast and announce transactions (only new ones, not resurrected ones)
	h.wg.Add(1)
	h.txsCh = make(chan core.NewTxsEvent, txChanSize)
	h.txsSub = h.txpool.SubscribeTransactions(h.txsCh, false)
	go h.txBroadcastLoop()

	// broadcast block range
	h.wg.Add(1)
	h.blockRange = newBlockRangeState(h.chain, h.eventMux)
	go h.blockRangeLoop(h.blockRange)

	// start sync handlers
	h.txFetcher.Start()

	// start peer handler tracker
	h.wg.Add(1)
	go h.protoTracker()
}

func (h *handler) Stop() {
	h.txsSub.Unsubscribe() // quits txBroadcastLoop
	h.blockRange.stop()
	h.txFetcher.Stop()
	h.downloader.Terminate()

	// Quit chainSync and txsync64.
	// After this is done, no new peers will be accepted.
	close(h.quitSync)

	// Disconnect existing sessions.
	// This also closes the gate for any new registrations on the peer set.
	// sessions which are already established but not added to h.peers yet
	// will exit when they try to register.
	h.peers.close()
	h.wg.Wait()

	log.Info("Ethereum protocol stopped")
}

// BroadcastTransactions will propagate a batch of transactions
// - To a square root of all peers for non-blob transactions
// - And, separately, as announcements to all peers which are not known to
// already have the given transaction.
func (h *handler) BroadcastTransactions(txs types.Transactions) {
	var (
		blobTxs  int // Number of blob transactions to announce only
		largeTxs int // Number of large transactions to announce only

		directCount int // Number of transactions sent directly to peers (duplicates included)
		annCount    int // Number of transactions announced across all peers (duplicates included)

		txset = make(map[*ethPeer][]common.Hash) // Set peer->hash to transfer directly
		annos = make(map[*ethPeer][]common.Hash) // Set peer->hash to announce

		signer = types.LatestSigner(h.chain.Config())
		choice = newBroadcastChoice(h.nodeID, h.txBroadcastKey)
		peers  = h.peers.all()
	)

	for _, tx := range txs {
		var directSet map[*ethPeer]struct{}
		switch {
		case tx.Type() == types.BlobTxType:
			blobTxs++
		case tx.Size() > txMaxBroadcastSize:
			largeTxs++
		default:
			// Get transaction sender address. Here we can ignore any error
			// since we're just interested in any value.
			txSender, _ := types.Sender(signer, tx)
			directSet = choice.choosePeers(peers, txSender)
		}

		for _, peer := range peers {
			if peer.KnownTransaction(tx.Hash()) {
				continue
			}
			if _, ok := directSet[peer]; ok {
				// Send direct.
				txset[peer] = append(txset[peer], tx.Hash())
			} else {
				// Send announcement.
				annos[peer] = append(annos[peer], tx.Hash())
			}
		}
	}

	for peer, hashes := range txset {
		directCount += len(hashes)
		peer.AsyncSendTransactions(hashes)
	}
	for peer, hashes := range annos {
		annCount += len(hashes)
		peer.AsyncSendPooledTransactionHashes(hashes)
	}
	log.Debug("Distributed transactions", "plaintxs", len(txs)-blobTxs-largeTxs, "blobtxs", blobTxs, "largetxs", largeTxs,
		"bcastpeers", len(txset), "bcastcount", directCount, "annpeers", len(annos), "anncount", annCount)
}

// txBroadcastLoop announces new transactions to connected peers.
func (h *handler) txBroadcastLoop() {
	defer h.wg.Done()
	for {
		select {
		case event := <-h.txsCh:
			h.BroadcastTransactions(event.Txs)
		case <-h.txsSub.Err():
			return
		}
	}
}

// enableSyncedFeatures enables the post-sync functionalities when the initial
// sync is finished.
func (h *handler) enableSyncedFeatures() {
	// Mark the local node as synced.
	h.synced.Store(true)

	// If we were running snap sync and it finished, disable doing another
	// round on next sync cycle
	if h.snapSync.Load() {
		log.Info("Snap sync complete, auto disabling")
		h.snapSync.Store(false)
	}
}

// blockRangeState holds the state of the block range update broadcasting mechanism.
type blockRangeState struct {
	prev    eth.BlockRangeUpdatePacket
	next    atomic.Pointer[eth.BlockRangeUpdatePacket]
	headCh  chan core.ChainHeadEvent
	headSub event.Subscription
	syncSub *event.TypeMuxSubscription
}

func newBlockRangeState(chain *core.BlockChain, typeMux *event.TypeMux) *blockRangeState {
	headCh := make(chan core.ChainHeadEvent, chainHeadChanSize)
	headSub := chain.SubscribeChainHeadEvent(headCh)
	syncSub := typeMux.Subscribe(downloader.StartEvent{}, downloader.DoneEvent{}, downloader.FailedEvent{})
	st := &blockRangeState{
		headCh:  headCh,
		headSub: headSub,
		syncSub: syncSub,
	}
	st.update(chain, chain.CurrentBlock())
	st.prev = *st.next.Load()
	return st
}

// blockRangeLoop announces changes in locally-available block range to peers.
// The range to announce is the range that is available in the store, so it's not just
// about imported blocks.
func (h *handler) blockRangeLoop(st *blockRangeState) {
	defer h.wg.Done()

	for {
		select {
		case ev := <-st.syncSub.Chan():
			if ev == nil {
				continue
			}
			if _, ok := ev.Data.(downloader.StartEvent); ok && h.snapSync.Load() {
				h.blockRangeWhileSnapSyncing(st)
			}
		case <-st.headCh:
			st.update(h.chain, h.chain.CurrentBlock())
			if st.shouldSend() {
				h.broadcastBlockRange(st)
			}
		case <-st.headSub.Err():
			return
		}
	}
}

// blockRangeWhileSnapSyncing announces block range updates during snap sync.
// Here we poll the CurrentSnapBlock on a timer and announce updates to it.
func (h *handler) blockRangeWhileSnapSyncing(st *blockRangeState) {
	tick := time.NewTicker(1 * time.Minute)
	defer tick.Stop()

	for {
		select {
		case <-tick.C:
			st.update(h.chain, h.chain.CurrentSnapBlock())
			if st.shouldSend() {
				h.broadcastBlockRange(st)
			}
		// back to processing head block updates when sync is done
		case ev := <-st.syncSub.Chan():
			if ev == nil {
				continue
			}
			switch ev.Data.(type) {
			case downloader.FailedEvent, downloader.DoneEvent:
				return
			}
		// ignore head updates, but exit when the subscription ends
		case <-st.headCh:
		case <-st.headSub.Err():
			return
		}
	}
}

// broadcastBlockRange sends a range update when one is due.
func (h *handler) broadcastBlockRange(state *blockRangeState) {
	h.peers.lock.Lock()
	peerlist := slices.Collect(maps.Values(h.peers.peers))
	h.peers.lock.Unlock()
	if len(peerlist) == 0 {
		return
	}
	msg := state.currentRange()
	log.Debug("Sending BlockRangeUpdate", "peers", len(peerlist), "earliest", msg.EarliestBlock, "latest", msg.LatestBlock)
	for _, p := range peerlist {
		p.SendBlockRangeUpdate(msg)
	}
	state.prev = *state.next.Load()
}

// update assigns the values of the next block range update from the chain.
func (st *blockRangeState) update(chain *core.BlockChain, latest *types.Header) {
	earliest, _ := chain.HistoryPruningCutoff()
	st.next.Store(&eth.BlockRangeUpdatePacket{
		EarliestBlock:   min(latest.Number.Uint64(), earliest),
		LatestBlock:     latest.Number.Uint64(),
		LatestBlockHash: latest.Hash(),
	})
}

// shouldSend decides whether it is time to send a block range update. We don't want to
// send these updates constantly, so they will usually only be sent every 32 blocks.
// However, there is a special case: if the range would move back, i.e. due to SetHead, we
// want to send it immediately.
func (st *blockRangeState) shouldSend() bool {
	next := st.next.Load()
	return next.LatestBlock < st.prev.LatestBlock ||
		next.LatestBlock-st.prev.LatestBlock >= 32
}

func (st *blockRangeState) stop() {
	st.syncSub.Unsubscribe()
	st.headSub.Unsubscribe()
}

// currentRange returns the current block range.
// This is safe to call from any goroutine.
func (st *blockRangeState) currentRange() eth.BlockRangeUpdatePacket {
	return *st.next.Load()
}

// broadcastChoice implements a deterministic random choice of peers. This is designed
// specifically for choosing which peer receives a direct broadcast of a transaction.
//
// The choice is made based on the involved p2p node IDs and the transaction sender,
// ensuring that the flow of transactions is grouped by account to (try and) avoid nonce
// gaps.
type broadcastChoice struct {
	self   enode.ID
	key    [16]byte
	buffer map[*ethPeer]struct{}
	tmp    []broadcastPeer
}

type broadcastPeer struct {
	p     *ethPeer
	score uint64
}

func newBroadcastChoiceKey() (k [16]byte) {
	crand.Read(k[:])
	return k
}

func newBroadcastChoice(self enode.ID, key [16]byte) *broadcastChoice {
	return &broadcastChoice{
		self:   self,
		key:    key,
		buffer: make(map[*ethPeer]struct{}),
	}
}

// choosePeers selects the peers that will receive a direct transaction broadcast message.
// Note the return value will only stay valid until the next call to choosePeers.
func (bc *broadcastChoice) choosePeers(peers []*ethPeer, txSender common.Address) map[*ethPeer]struct{} {
	// Compute randomized scores.
	bc.tmp = slices.Grow(bc.tmp[:0], len(peers))[:len(peers)]
	hash := siphash.New(bc.key[:])
	for i, peer := range peers {
		hash.Reset()
		hash.Write(bc.self[:])
		hash.Write(peer.Peer.Peer.ID().Bytes())
		hash.Write(txSender[:])
		bc.tmp[i] = broadcastPeer{peer, hash.Sum64()}
	}

	// Sort by score.
	slices.SortFunc(bc.tmp, func(a, b broadcastPeer) int {
		return cmp.Compare(a.score, b.score)
	})

	// Take top n.
	clear(bc.buffer)
	n := int(math.Ceil(math.Sqrt(float64(len(bc.tmp)))))
	for i := range n {
		bc.buffer[bc.tmp[i].p] = struct{}{}
	}
	return bc.buffer
}
