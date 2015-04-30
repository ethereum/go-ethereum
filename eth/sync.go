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
	// itimer is used to determine when to start ignoring `minDesiredPeerCount`
	itimer := time.NewTimer(peerCountTimeout)
	// btimer is used for picking of blocks from the downloader
	btimer := time.NewTicker(blockProcTimer)
out:
	for {
		select {
		case <-pm.newPeerCh:
			// Meet the `minDesiredPeerCount` before we select our best peer
			if len(pm.peers) < minDesiredPeerCount {
				break
			}

			// Find the best peer
			peer := getBestPeer(pm.peers)
			if peer == nil {
				glog.V(logger.Debug).Infoln("Sync attempt cancelled. No peers available")
			}

			itimer.Stop()
			go pm.synchronise(peer)
		case <-itimer.C:
			// The timer will make sure that the downloader keeps an active state
			// in which it attempts to always check the network for highest td peers
			// Either select the peer or restart the timer if no peers could
			// be selected.
			if peer := getBestPeer(pm.peers); peer != nil {
				go pm.synchronise(peer)
			} else {
				itimer.Reset(5 * time.Second)
			}
		case <-btimer.C:
			go pm.processBlocks()
		case <-pm.quitSync:
			break out
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

	blocks := pm.downloader.TakeBlocks()
	if len(blocks) == 0 {
		return nil
	}
	defer pm.downloader.Done()

	glog.V(logger.Debug).Infof("Inserting chain with %d blocks (#%v - #%v)\n", len(blocks), blocks[0].Number(), blocks[len(blocks)-1].Number())

	for len(blocks) != 0 && !pm.quit {
		max := int(math.Min(float64(len(blocks)), float64(blockProcAmount)))
		_, err := pm.chainman.InsertChain(blocks[:max])
		if err != nil {
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
	// Check downloader if it's busy so it doesn't show the sync message
	// for every attempty
	if pm.downloader.IsBusy() {
		return
	}

	// Get the hashes from the peer (synchronously)
	err := pm.downloader.Synchronise(peer.id, peer.recentHash)
	if err != nil && err == downloader.ErrBadPeer {
		glog.V(logger.Debug).Infoln("removed peer from peer set due to bad action")
		pm.removePeer(peer)
	} else if err != nil {
		// handle error
		glog.V(logger.Detail).Infoln("error downloading:", err)
	}
}
