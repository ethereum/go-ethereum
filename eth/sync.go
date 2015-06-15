package eth

import (
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

const (
	forceSyncCycle      = 10 * time.Second       // Time interval to force syncs, even if few peers are available
	notifyCheckCycle    = 100 * time.Millisecond // Time interval to allow hash notifies to fulfill before hard fetching
	notifyArriveTimeout = 500 * time.Millisecond // Time allowance before an announced block is explicitly requested
	notifyFetchTimeout  = 5 * time.Second        // Maximum alloted time to return an explicitly requested block
	minDesiredPeerCount = 5                      // Amount of peers desired to start syncing

	// This is the target size for the packs of transactions sent by txsyncLoop.
	// A pack can get larger than this if a single transactions exceeds this size.
	txsyncPackSize = 100 * 1024
)

// blockAnnounce is the hash notification of the availability of a new block in
// the network.
type blockAnnounce struct {
	hash common.Hash
	peer *peer
	time time.Time
}

type txsync struct {
	p   *peer
	txs []*types.Transaction
}

// syncTransactions starts sending all currently pending transactions to the given peer.
func (pm *ProtocolManager) syncTransactions(p *peer) {
	txs := pm.txpool.GetTransactions()
	if len(txs) == 0 {
		return
	}
	select {
	case pm.txsyncCh <- &txsync{p, txs}:
	case <-pm.quitSync:
	}
}

// txsyncLoop takes care of the initial transaction sync for each new
// connection. When a new peer appears, we relay all currently pending
// transactions. In order to minimise egress bandwidth usage, we send
// the transactions in small packs to one peer at a time.
func (pm *ProtocolManager) txsyncLoop() {
	var (
		pending = make(map[discover.NodeID]*txsync)
		sending = false               // whether a send is active
		pack    = new(txsync)         // the pack that is being sent
		done    = make(chan error, 1) // result of the send
	)

	// send starts a sending a pack of transactions from the sync.
	send := func(s *txsync) {
		// Fill pack with transactions up to the target size.
		size := common.StorageSize(0)
		pack.p = s.p
		pack.txs = pack.txs[:0]
		for i := 0; i < len(s.txs) && size < txsyncPackSize; i++ {
			pack.txs = append(pack.txs, s.txs[i])
			size += s.txs[i].Size()
		}
		// Remove the transactions that will be sent.
		s.txs = s.txs[:copy(s.txs, s.txs[len(pack.txs):])]
		if len(s.txs) == 0 {
			delete(pending, s.p.ID())
		}
		// Send the pack in the background.
		glog.V(logger.Detail).Infof("%v: sending %d transactions (%v)", s.p.Peer, len(pack.txs), size)
		sending = true
		go func() { done <- pack.p.sendTransactions(pack.txs) }()
	}

	// pick chooses the next pending sync.
	pick := func() *txsync {
		if len(pending) == 0 {
			return nil
		}
		n := rand.Intn(len(pending)) + 1
		for _, s := range pending {
			if n--; n == 0 {
				return s
			}
		}
		return nil
	}

	for {
		select {
		case s := <-pm.txsyncCh:
			pending[s.p.ID()] = s
			if !sending {
				send(s)
			}
		case err := <-done:
			sending = false
			// Stop tracking peers that cause send failures.
			if err != nil {
				glog.V(logger.Debug).Infof("%v: tx send failed: %v", pack.p.Peer, err)
				delete(pending, pack.p.ID())
			}
			// Schedule the next send.
			if s := pick(); s != nil {
				send(s)
			}
		case <-pm.quitSync:
			return
		}
	}
}

// fetcher is responsible for collecting hash notifications, and periodically
// checking all unknown ones and individually fetching them.
func (pm *ProtocolManager) fetcher() {
	announces := make(map[common.Hash][]*blockAnnounce)
	request := make(map[*peer][]common.Hash)
	pending := make(map[common.Hash]*blockAnnounce)
	cycle := time.Tick(notifyCheckCycle)
	done := make(chan common.Hash)

	// Iterate the block fetching until a quit is requested
	for {
		select {
		case notifications := <-pm.newHashCh:
			// A batch of hashes the notified, schedule them for retrieval
			glog.V(logger.Debug).Infof("Scheduling %d hash announcements from %s", len(notifications), notifications[0].peer.id)
			for _, announce := range notifications {
				// Skip if it's already pending fetch
				if _, ok := pending[announce.hash]; ok {
					continue
				}
				// Otherwise queue up the peer as a potential source
				announces[announce.hash] = append(announces[announce.hash], announce)
			}

		case hash := <-done:
			// A pending import finished, remove all traces
			delete(pending, hash)

		case <-cycle:
			// Clean up any expired block fetches
			for hash, announce := range pending {
				if time.Since(announce.time) > notifyFetchTimeout {
					delete(pending, hash)
				}
			}
			// Check if any notified blocks failed to arrive
			for hash, all := range announces {
				if time.Since(all[0].time) > notifyArriveTimeout {
					announce := all[rand.Intn(len(all))]
					if !pm.chainman.HasBlock(hash) {
						request[announce.peer] = append(request[announce.peer], hash)
						pending[hash] = announce
					}
					delete(announces, hash)
				}
			}
			if len(request) == 0 {
				break
			}
			// Send out all block requests
			for peer, hashes := range request {
				glog.V(logger.Debug).Infof("Explicitly fetching %d blocks from %s", len(hashes), peer.id)
				go peer.requestBlocks(hashes)
			}
			request = make(map[*peer][]common.Hash)

		case filter := <-pm.newBlockCh:
			// Blocks arrived, extract any explicit fetches, return all else
			var blocks types.Blocks
			select {
			case blocks = <-filter:
			case <-pm.quitSync:
				return
			}

			explicit, download := []*types.Block{}, []*types.Block{}
			for _, block := range blocks {
				hash := block.Hash()

				// Filter explicitly requested blocks from hash announcements
				if _, ok := pending[hash]; ok {
					// Discard if already imported by other means
					if !pm.chainman.HasBlock(hash) {
						explicit = append(explicit, block)
					} else {
						delete(pending, hash)
					}
				} else {
					download = append(download, block)
				}
			}

			select {
			case filter <- download:
			case <-pm.quitSync:
				return
			}
			// Create a closure with the retrieved blocks and origin peers
			peers := make([]*peer, 0, len(explicit))
			blocks = make([]*types.Block, 0, len(explicit))
			for _, block := range explicit {
				hash := block.Hash()
				if announce := pending[hash]; announce != nil {
					// Drop the block if it surely cannot fit
					if pm.chainman.HasBlock(hash) || !pm.chainman.HasBlock(block.ParentHash()) {
						// delete(pending, hash) // if we drop, it will re-fetch it, wait for timeout?
						continue
					}
					// Otherwise accumulate for import
					peers = append(peers, announce.peer)
					blocks = append(blocks, block)
				}
			}
			// If any explicit fetches were replied to, import them
			if count := len(blocks); count > 0 {
				glog.V(logger.Debug).Infof("Importing %d explicitly fetched blocks", len(blocks))
				go func() {
					// Make sure all hashes are cleaned up
					for _, block := range blocks {
						hash := block.Hash()
						defer func() { done <- hash }()
					}
					// Try and actually import the blocks
					for i := 0; i < len(blocks); i++ {
						if err := pm.importBlock(peers[i], blocks[i], nil); err != nil {
							glog.V(logger.Detail).Infof("Failed to import explicitly fetched block: %v", err)
							return
						}
					}
				}()
			}

		case <-pm.quitSync:
			return
		}
	}
}

// syncer is responsible for periodically synchronising with the network, both
// downloading hashes and blocks as well as retrieving cached ones.
func (pm *ProtocolManager) syncer() {
	// Abort any pending syncs if we terminate
	defer pm.downloader.Cancel()

	forceSync := time.Tick(forceSyncCycle)
	for {
		select {
		case <-pm.newPeerCh:
			// Make sure we have peers to select from, then sync
			if pm.peers.Len() < minDesiredPeerCount {
				break
			}
			go pm.synchronise(pm.peers.BestPeer())

		case <-forceSync:
			// Force a sync even if not enough peers are present
			go pm.synchronise(pm.peers.BestPeer())

		case <-pm.quitSync:
			return
		}
	}
}

// synchronise tries to sync up our local block chain with a remote peer, both
// adding various sanity checks as well as wrapping it with various log entries.
func (pm *ProtocolManager) synchronise(peer *peer) {
	// Short circuit if no peers are available
	if peer == nil {
		return
	}
	// Make sure the peer's TD is higher than our own. If not drop.
	if peer.Td().Cmp(pm.chainman.Td()) <= 0 {
		return
	}
	// Otherwise try to sync with the downloader
	pm.downloader.Synchronise(peer.id, peer.Head())
}
