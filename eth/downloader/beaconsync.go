// Copyright 2021 The go-ethereum Authors
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

package downloader

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// beaconBackfiller is the chain and state backfilling that can be commenced once
// the skeleton syncer has successfully reverse downloaded all the headers up to
// the genesis block or an existing header in the database. Its operation is fully
// directed by the skeleton sync's head/tail events.
type beaconBackfiller struct {
	downloader *Downloader // Downloader to direct via this callback implementation
	syncMode   SyncMode    // Sync mode to use for backfilling the skeleton chains
	success    func()      // Callback to run on successful sync cycle completion
	filling    bool        // Flag whether the downloader is backfilling or not
	lock       sync.Mutex  // Mutex protecting the sync lock
}

// newBeaconBackfiller is a helper method to create the backfiller.
func newBeaconBackfiller(dl *Downloader, success func()) backfiller {
	return &beaconBackfiller{
		downloader: dl,
		success:    success,
	}
}

// suspend cancels any background downloader threads.
func (b *beaconBackfiller) suspend() {
	b.downloader.Cancel()
}

// resume starts the downloader threads for backfilling state and chain data.
func (b *beaconBackfiller) resume() {
	b.lock.Lock()
	if b.filling {
		// If a previous filling cycle is still running, just ignore this start
		// request. // TODO(karalabe): We should make this channel driven
		b.lock.Unlock()
		return
	}
	b.filling = true
	mode := b.syncMode
	b.lock.Unlock()

	// Start the backfilling on its own thread since the downloader does not have
	// its own lifecycle runloop.
	go func() {
		// Set the backfiller to non-filling when download completes
		defer func() {
			b.lock.Lock()
			b.filling = false
			b.lock.Unlock()
		}()
		// If the downloader fails, report an error as in beacon chain mode there
		// should be no errors as long as the chain we're syncing to is valid.
		if err := b.downloader.synchronise("", common.Hash{}, nil, mode, true); err != nil {
			log.Error("Beacon backfilling failed", "err", err)
			return
		}
		// Synchronization succeeded. Since this happens async, notify the outer
		// context to disable snap syncing and enable transaction propagation.
		if b.success != nil {
			b.success()
		}
	}()
}

// setMode updates the sync mode from the current one to the requested one. If
// there's an active sync in progress, it will be cancelled and restarted.
func (b *beaconBackfiller) setMode(mode SyncMode) {
	// Update the old sync mode and track if it was changed
	b.lock.Lock()
	updated := b.syncMode != mode
	filling := b.filling
	b.syncMode = mode
	b.lock.Unlock()

	// If the sync mode was changed mid-sync, restart. This should never ever
	// really happen, we just handle it to detect programming errors.
	if !updated || !filling {
		return
	}
	log.Error("Downloader sync mode changed mid-run", "old", mode.String(), "new", mode.String())
	b.suspend()
	b.resume()
}

// BeaconSync is the Ethereum 2 version of the chain synchronization, where the
// chain is not downloaded from genesis onward, rather from trusted head announces
// backwards.
//
// Internally backfilling and state sync is done the same way, but the header
// retrieval and scheduling is replaced.
func (d *Downloader) BeaconSync(mode SyncMode, head *types.Header) error {
	// When the downloader starts a sync cycle, it needs to be aware of the sync
	// mode to use (full, snap). To keep the skeleton chain oblivious, inject the
	// mode into the backfiller directly.
	//
	// Super crazy dangerous type cast. Should be fine (TM), we're only using a
	// different backfiller implementation for skeleton tests.
	d.skeleton.filler.(*beaconBackfiller).setMode(mode)

	// Signal the skeleton sync to switch to a new head, however it wants
	if err := d.skeleton.Sync(head); err != nil {
		return err
	}
	return nil
}

// findBeaconAncestor tries to locate the common ancestor link of the local chain
// and the beacon chain just requested. In the general case when our node was in
// sync and on the correct chain, checking the top N links should already get us
// a match. In the rare scenario when we ended up on a long reorganisation (i.e.
// none of the head links match), we do a binary search to find the ancestor.
func (d *Downloader) findBeaconAncestor() uint64 {
	// Figure out the current local head position
	var head *types.Header

	switch d.getMode() {
	case FullSync:
		head = d.blockchain.CurrentBlock().Header()
	case SnapSync:
		head = d.blockchain.CurrentFastBlock().Header()
	default:
		head = d.lightchain.CurrentHeader()
	}
	number := head.Number.Uint64()

	// If the head is present in the skeleton chain, return that
	if head.Hash() == d.skeleton.Header(number).Hash() {
		return number
	}
	// Head header not present, binary search to find the ancestor
	start, end := uint64(0), number
	for start+1 < end {
		// Split our chain interval in two, and request the hash to cross check
		check := (start + end) / 2

		h := d.skeleton.Header(check)
		n := h.Number.Uint64()

		var known bool
		switch d.getMode() {
		case FullSync:
			known = d.blockchain.HasBlock(h.Hash(), n)
		case SnapSync:
			known = d.blockchain.HasFastBlock(h.Hash(), n)
		default:
			known = d.lightchain.HasHeader(h.Hash(), n)
		}
		if !known {
			end = check
			continue
		}
		start = check
	}
	return start
}

// fetchBeaconHeaders feeds skeleton headers to the downloader queue for scheduling
// until sync errors or is finished.
func (d *Downloader) fetchBeaconHeaders(from uint64) error {
	head, err := d.skeleton.Head()
	if err != nil {
		return err
	}
	for {
		// Retrieve a batch of headers and feed it to the header processor
		var (
			headers = make([]*types.Header, 0, maxHeadersProcess)
			hashes  = make([]common.Hash, 0, maxHeadersProcess)
		)
		for i := 0; i < maxHeadersProcess && from <= head.Number.Uint64(); i++ {
			headers = append(headers, d.skeleton.Header(from))
			hashes = append(hashes, headers[i].Hash())
			from++
		}
		select {
		case d.headerProcCh <- &headerTask{
			headers: headers,
			hashes:  hashes,
		}:
		case <-d.cancelCh:
			return errCanceled
		}
		// If we still have headers to import, loop and keep pushing them
		if from <= head.Number.Uint64() {
			continue
		}
		// If the pivot block is committed, signal header sync termination
		if atomic.LoadInt32(&d.committed) == 1 {
			d.headerProcCh <- nil
			return nil
		}
		// State sync still going, wait a bit for new headers and retry
		select {
		case <-time.After(fsHeaderContCheck):
		case <-d.cancelCh:
			return errCanceled
		}
		head, err = d.skeleton.Head()
		if err != nil {
			return err
		}
	}
}
