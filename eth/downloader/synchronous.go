package downloader

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

// THIS IS PENDING AND TO DO CHANGES FOR MAKING THE DOWNLOADER SYNCHRONOUS

// SynchroniseWithPeer will select the peer and use it for synchronising. If an empty string is given
// it will use the best peer possible and synchronise if it's TD is higher than our own. If any of the
// checks fail an error will be returned. This method is synchronous
func (d *Downloader) SynchroniseWithPeer(id string) (types.Blocks, error) {
	// Check if we're busy
	if d.isBusy() {
		return nil, errBusy
	}

	// Attempt to select a peer. This can either be nothing, which returns, best peer
	// or selected peer. If no peer could be found an error will be returned
	var p *peer
	if len(id) == 0 {
		p = d.peers[id]
		if p == nil {
			return nil, errUnknownPeer
		}
	} else {
		p = d.peers.bestPeer()
	}

	// Make sure our td is lower than the peer's td
	if p.td.Cmp(d.currentTd()) <= 0 || d.hasBlock(p.recentHash) {
		return nil, errLowTd
	}

	// Get the hash from the peer and initiate the downloading progress.
	err := d.getFromPeer(p, p.recentHash, false)
	if err != nil {
		return nil, err
	}

	return d.queue.blocks, nil
}

// Synchronise will synchronise using the best peer.
func (d *Downloader) Synchronise() (types.Blocks, error) {
	return d.SynchroniseWithPeer("")
}

func (d *Downloader) getFromPeer(p *peer, hash common.Hash, ignoreInitial bool) error {
	glog.V(logger.Detail).Infoln("Synchronising with the network using:", p.id)
	// Start the fetcher. This will block the update entirely
	// interupts need to be send to the appropriate channels
	// respectively.
	if err := d.startFetchingHashes(p, hash, ignoreInitial); err != nil {
		// handle error
		glog.V(logger.Debug).Infoln("Error fetching hashes:", err)
		// XXX Reset
		return err
	}

	// Start fetching blocks in paralel. The strategy is simple
	// take any available peers, seserve a chunk for each peer available,
	// let the peer deliver the chunkn and periodically check if a peer
	// has timedout. When done downloading, process blocks.
	if err := d.startFetchingBlocks(p); err != nil {
		glog.V(logger.Debug).Infoln("Error downloading blocks:", err)
		// XXX reset
		return err
	}

	glog.V(logger.Detail).Infoln("Sync completed")

	return nil
}
