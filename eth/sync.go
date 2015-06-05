package eth

import (
	"math"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

// update periodically tries to synchronise with the network, both downloading
// hashes and blocks as well as retrieving cached ones.
func (pm *ProtocolManager) update() {
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
	if peer.td.Cmp(pm.chainman.Td()) <= 0 {
		return
	}
	// FIXME if we have the hash in our chain and the TD of the peer is
	// much higher than ours, something is wrong with us or the peer.
	// Check if the hash is on our own chain
	if pm.chainman.HasBlock(peer.recentHash) {
		glog.V(logger.Debug).Infoln("Synchronisation canceled: head already known")
		return
	}
	// Get the hashes from the peer (synchronously)
	glog.V(logger.Debug).Infof("Attempting synchronisation: %v, 0x%x", peer.id, peer.recentHash)

	err := pm.downloader.Synchronise(peer.id, peer.recentHash)
	switch err {
	case nil:
		glog.V(logger.Debug).Infof("Synchronisation completed")

	case downloader.ErrBusy:
		glog.V(logger.Debug).Infof("Synchronisation already in progress")

	case downloader.ErrTimeout, downloader.ErrBadPeer, downloader.ErrEmptyHashSet, downloader.ErrInvalidChain, downloader.ErrCrossCheckFailed:
		glog.V(logger.Debug).Infof("Removing peer %v: %v", peer.id, err)
		pm.removePeer(peer.id)

	case downloader.ErrPendingQueue:
		glog.V(logger.Debug).Infoln("Synchronisation aborted:", err)

	default:
		glog.V(logger.Warn).Infof("Synchronisation failed: %v", err)
	}
}
