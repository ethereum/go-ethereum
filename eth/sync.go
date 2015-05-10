package eth

import (
	"math"
	"time"

	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

// Sync contains all synchronisation code for the eth protocol

func (pm *ProtocolManager) update() {
	forceSync := time.Tick(forceSyncCycle)
	blockProc := time.Tick(blockProcCycle)

	for {
		select {
		case <-pm.newPeerCh:
			// Meet the `minDesiredPeerCount` before we select our best peer
			if len(pm.peers) < minDesiredPeerCount {
				break
			}
			// Find the best peer and synchronise with it
			peer := getBestPeer(pm.peers)
			if peer == nil {
				glog.V(logger.Debug).Infoln("Sync attempt canceled. No peers available")
			}
			go pm.synchronise(peer)

		case <-forceSync:
			// Force a sync even if not enough peers are present
			if peer := getBestPeer(pm.peers); peer != nil {
				go pm.synchronise(peer)
			}
		case <-blockProc:
			// Try to pull some blocks from the downloaded
			go pm.processBlocks()

		case <-pm.quitSync:
			return
		}
	}
}

// processBlocks will attempt to reconstruct a chain by checking the first item and check if it's
// a known parent. The first block in the chain may be unknown during downloading. When the
// downloader isn't downloading blocks will be dropped with an unknown parent until either it
// has depleted the list or found a known parent.
func (pm *ProtocolManager) processBlocks() error {
	pm.wg.Add(1)
	defer pm.wg.Done()

	// Take a batch of blocks (will return nil if a previous batch has not reached the chain yet)
	blocks := pm.downloader.TakeBlocks()
	if len(blocks) == 0 {
		return nil
	}
	glog.V(logger.Debug).Infof("Inserting chain with %d blocks (#%v - #%v)\n", len(blocks), blocks[0].Number(), blocks[len(blocks)-1].Number())

	for len(blocks) != 0 && !pm.quit {
		max := int(math.Min(float64(len(blocks)), float64(blockProcAmount)))
		_, err := pm.chainman.InsertChain(blocks[:max])
		if err != nil {
			// cancel download process
			pm.downloader.Cancel()

			return err
		}
		blocks = blocks[max:]
	}
	return nil
}

func (pm *ProtocolManager) synchronise(peer *peer) {
	// Make sure the peer's TD is higher than our own. If not drop.
	if peer.td.Cmp(pm.chainman.Td()) <= 0 {
		return
	}
	// FIXME if we have the hash in our chain and the TD of the peer is
	// much higher than ours, something is wrong with us or the peer.
	// Check if the hash is on our own chain
	if pm.chainman.HasBlock(peer.recentHash) {
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

	case downloader.ErrTimeout:
		glog.V(logger.Debug).Infof("Removing peer %v due to sync timeout", peer.id)
		pm.removePeer(peer)

	default:
		glog.V(logger.Warn).Infof("Synchronisation failed: %v", err)
	}
}
