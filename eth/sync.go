package eth

import (
	"math"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

const (
	forceSyncCycle      = 10 * time.Second       // Time interval to force syncs, even if few peers are available
	blockProcCycle      = 500 * time.Millisecond // Time interval to check for new blocks to process
	notifyCheckCycle    = 100 * time.Millisecond // Time interval to allow hash notifies to fulfill before hard fetching
	notifyArriveTimeout = 500 * time.Millisecond // Time allowance before an announced block is explicitly requested
	notifyFetchTimeout  = 5 * time.Second        // Maximum alloted time to return an explicitly requested block
	minDesiredPeerCount = 5                      // Amount of peers desired to start syncing
	blockProcAmount     = 256
)

// blockAnnounce is the hash notification of the availability of a new block in
// the network.
type blockAnnounce struct {
	hash common.Hash
	peer *peer
	time time.Time
}

// fetcher is responsible for collecting hash notifications, and periodically
// checking all unknown ones and individually fetching them.
func (pm *ProtocolManager) fetcher() {
	announces := make(map[common.Hash]*blockAnnounce)
	request := make(map[*peer][]common.Hash)
	pending := make(map[common.Hash]*blockAnnounce)
	cycle := time.Tick(notifyCheckCycle)

	// Iterate the block fetching until a quit is requested
	for {
		select {
		case notifications := <-pm.newHashCh:
			// A batch of hashes the notified, schedule them for retrieval
			glog.V(logger.Debug).Infof("Scheduling %d hash announcements from %s", len(notifications), notifications[0].peer.id)
			for _, announce := range notifications {
				announces[announce.hash] = announce
			}

		case <-cycle:
			// Clean up any expired block fetches
			for hash, announce := range pending {
				if time.Since(announce.time) > notifyFetchTimeout {
					delete(pending, hash)
				}
			}
			// Check if any notified blocks failed to arrive
			for hash, announce := range announces {
				if time.Since(announce.time) > notifyArriveTimeout {
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
				peer.requestBlocks(hashes)
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
			// If any explicit fetches were replied to, import them
			if count := len(explicit); count > 0 {
				glog.V(logger.Debug).Infof("Importing %d explicitly fetched blocks", count)
				go func() {
					for _, block := range explicit {
						hash := block.Hash()

						// Make sure there's still something pending to import
						if announce := pending[hash]; announce != nil {
							delete(pending, hash)
							if err := pm.importBlock(announce.peer, block, nil); err != nil {
								glog.V(logger.Detail).Infof("Failed to import explicitly fetched block: %v", err)
								return
							}
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
	forceSync := time.Tick(forceSyncCycle)
	blockProc := time.Tick(blockProcCycle)
	blockProcPend := int32(0)

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

		case <-blockProc:
			// Try to pull some blocks from the downloaded
			if atomic.CompareAndSwapInt32(&blockProcPend, 0, 1) {
				go func() {
					pm.processBlocks()
					atomic.StoreInt32(&blockProcPend, 0)
				}()
			}

		case <-pm.quitSync:
			return
		}
	}
}

// processBlocks retrieves downloaded blocks from the download cache and tries
// to construct the local block chain with it. Note, since the block retrieval
// order matters, access to this function *must* be synchronized/serialized.
func (pm *ProtocolManager) processBlocks() error {
	pm.wg.Add(1)
	defer pm.wg.Done()

	// Short circuit if no blocks are available for insertion
	blocks := pm.downloader.TakeBlocks()
	if len(blocks) == 0 {
		return nil
	}
	glog.V(logger.Debug).Infof("Inserting chain with %d blocks (#%v - #%v)\n", len(blocks), blocks[0].RawBlock.Number(), blocks[len(blocks)-1].RawBlock.Number())

	for len(blocks) != 0 && !pm.quit {
		// Retrieve the first batch of blocks to insert
		max := int(math.Min(float64(len(blocks)), float64(blockProcAmount)))
		raw := make(types.Blocks, 0, max)
		for _, block := range blocks[:max] {
			raw = append(raw, block.RawBlock)
		}
		// Try to inset the blocks, drop the originating peer if there's an error
		index, err := pm.chainman.InsertChain(raw)
		if err != nil {
			glog.V(logger.Debug).Infoln("Downloaded block import failed:", err)
			pm.removePeer(blocks[index].OriginPeer)
			pm.downloader.Cancel()
			return err
		}
		blocks = blocks[max:]
	}
	return nil
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
	// FIXME if we have the hash in our chain and the TD of the peer is
	// much higher than ours, something is wrong with us or the peer.
	// Check if the hash is on our own chain
	head := peer.Head()
	if pm.chainman.HasBlock(head) {
		glog.V(logger.Debug).Infoln("Synchronisation canceled: head already known")
		return
	}
	// Get the hashes from the peer (synchronously)
	glog.V(logger.Detail).Infof("Attempting synchronisation: %v, 0x%x", peer.id, head)

	err := pm.downloader.Synchronise(peer.id, head)
	switch err {
	case nil:
		glog.V(logger.Detail).Infof("Synchronisation completed")

	case downloader.ErrBusy:
		glog.V(logger.Detail).Infof("Synchronisation already in progress")

	case downloader.ErrTimeout, downloader.ErrBadPeer, downloader.ErrEmptyHashSet, downloader.ErrInvalidChain, downloader.ErrCrossCheckFailed:
		glog.V(logger.Debug).Infof("Removing peer %v: %v", peer.id, err)
		pm.removePeer(peer.id)

	case downloader.ErrPendingQueue:
		glog.V(logger.Debug).Infoln("Synchronisation aborted:", err)

	default:
		glog.V(logger.Warn).Infof("Synchronisation failed: %v", err)
	}
}
