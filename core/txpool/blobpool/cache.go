// Copyright 2026 The go-ethereum Authors
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

package blobpool

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
)

type cacheMode int

const (
	hasBlobsMode cacheMode = iota
	topKMode
)

const (
	topKTimeout     = 1 * time.Second
	hasBlobsTimeout = 1 * time.Second
)

var (
	cacheHitMeter   = metrics.NewRegisteredMeter("blobpool/cache/hit", nil)
	cacheMissMeter  = metrics.NewRegisteredMeter("blobpool/cache/miss", nil)
	cacheBlobsGauge = metrics.NewRegisteredGauge("blobpool/cache/blobs", nil)
)

type Cache struct {
	mu sync.Mutex

	txHashOf map[common.Hash]common.Hash          // vhash -> txhash
	entries  map[common.Hash]*types.BlobTxSidecar // txhash -> sidecar
	capacity uint

	blobpool *BlobPool

	hasBlobs chan []common.Hash // list of tx hashes that needs to be cached
	getBlobs chan struct{}

	mode cacheMode

	cancelInflights context.CancelFunc
	inflight        sync.WaitGroup

	clock mclock.Clock

	step func() // test hook fired after each loop iteration

	quit chan struct{}
}

func NewCache(p *BlobPool, cap uint) *Cache {
	return NewCacheForTest(p, cap, mclock.System{}, nil)
}

func NewCacheForTest(p *BlobPool, cap uint, clock mclock.Clock, step func()) *Cache {
	c := &Cache{
		entries:  make(map[common.Hash]*types.BlobTxSidecar, cap),
		txHashOf: make(map[common.Hash]common.Hash),
		capacity: cap,
		blobpool: p,
		hasBlobs: make(chan []common.Hash, 1),
		getBlobs: make(chan struct{}, 1),

		mode:  topKMode,
		clock: clock,

		step: step,

		quit: make(chan struct{}),
	}

	go c.loop()
	return c
}

func (c *Cache) Stop() {
	c.mu.Lock()
	if c.cancelInflights != nil {
		c.cancelInflights()
	}
	c.mu.Unlock()
	close(c.quit)
}

func (c *Cache) HasBlobs(ctx context.Context, vhashes []common.Hash) []bool {
	var (
		missIdx     []int
		missVhashes []common.Hash
		needPin     []common.Hash
	)
	available := make([]bool, len(vhashes))

	c.mu.Lock()
	for i, vhash := range vhashes {
		if _, ok := c.txHashOf[vhash]; ok {
			available[i] = true
			needPin = append(needPin, vhash)
		} else {
			missIdx = append(missIdx, i)
			missVhashes = append(missVhashes, vhash)
		}
	}
	c.mu.Unlock()

	if len(missVhashes) > 0 {
		pooled := c.blobpool.AvailableBlobs(missVhashes)
		for j, ok := range pooled {
			if ok {
				available[missIdx[j]] = true
				needPin = append(needPin, missVhashes[j])
			}
		}
	}

	select {
	case c.hasBlobs <- needPin:
		return available
	case <-c.quit:
		return nil
	}
}

func (c *Cache) GetBlobs(ctx context.Context, vhashes []common.Hash, version byte) (_ []*kzg4844.Blob, _ []kzg4844.Commitment, _ [][]kzg4844.Proof, err error) {
	var (
		blobs       = make([]*kzg4844.Blob, len(vhashes))
		commitments = make([]kzg4844.Commitment, len(vhashes))
		proofs      = make([][]kzg4844.Proof, len(vhashes))

		indices = make(map[common.Hash][]int)
		filled  = make(map[common.Hash]struct{})

		cacheHits int64
		cacheMiss int64
	)
	for i, h := range vhashes {
		indices[h] = append(indices[h], i)
	}

	for _, vhash := range vhashes {
		if _, ok := filled[vhash]; ok {
			// Skip vhash that was already resolved in a previous iteration
			continue
		}
		c.mu.Lock()
		txhash := c.txHashOf[vhash]
		sidecar := c.entries[txhash]
		c.mu.Unlock()

		if sidecar != nil {
			cacheHits++
		} else {
			cacheMiss++
			if ptx := c.blobpool.getByVhash(vhash); ptx != nil {
				sidecar = ptx.Sidecar()
			}
		}
		if sidecar == nil {
			continue
		}

		for i, hash := range sidecar.BlobHashes() {
			list, ok := indices[hash]
			if !ok {
				continue
			}
			// Mark hash as seen.
			filled[hash] = struct{}{}
			if sidecar.Version != version {
				// Skip blobs with incompatible version. Note we still track the blob hash
				// in `filled` here, ensuring that we do not resolve this tx another time.
				continue
			}
			// Get or convert the proof.
			var pf []kzg4844.Proof
			switch version {
			case types.BlobSidecarVersion0:
				pf = []kzg4844.Proof{sidecar.Proofs[i]}
			case types.BlobSidecarVersion1:
				cellProofs, err := sidecar.CellProofsAt(i)
				if err != nil {
					log.Error("Failed to get cell proofs", "txhash", txhash, "err", err)
					continue
				}
				pf = cellProofs
			}
			for _, index := range list {
				blobs[index] = &sidecar.Blobs[i]
				commitments[index] = sidecar.Commitments[i]
				proofs[index] = pf
			}
		}
	}
	cacheHitMeter.Mark(cacheHits)
	cacheMissMeter.Mark(cacheMiss)

	select {
	case c.getBlobs <- struct{}{}:
		return blobs, commitments, proofs, nil
	case <-c.quit:
		return nil, nil, nil, nil
	}
}

func (c *Cache) loop() {
	var (
		topKTimer   = new(mclock.Timer)
		topKTrigger = make(chan struct{}, 1)
		hbTimer     = new(mclock.Timer)
		hbTrigger   = make(chan struct{}, 1)
	)
	c.resetTimer(topKTimer, topKTrigger, topKTimeout)

	for {
		select {
		case want := <-c.hasBlobs:
			c.switchMode(hasBlobsMode)
			c.update(want)
			c.resetTimer(hbTimer, hbTrigger, hasBlobsTimeout)

		case <-topKTrigger:
			if c.mode != topKMode {
				if c.step != nil {
					c.step()
				}
				continue
			}
			want := selectTxs(c.blobpool.indexSnapshot(), c.capacity)
			c.update(want)
			c.resetTimer(topKTimer, topKTrigger, topKTimeout)

		case <-hbTrigger:
			c.switchMode(topKMode)
			want := selectTxs(c.blobpool.indexSnapshot(), c.capacity)
			c.update(want)
			c.resetTimer(topKTimer, topKTrigger, topKTimeout)

		case <-c.getBlobs:
			if c.mode != hasBlobsMode {
				if c.step != nil {
					c.step()
				}
				continue
			}
			c.switchMode(topKMode)
			want := selectTxs(c.blobpool.indexSnapshot(), c.capacity)
			c.update(want)
			c.resetTimer(topKTimer, topKTrigger, topKTimeout)
		case <-c.quit:
			return
		}

		if c.step != nil {
			c.step()
		}
	}
}

func (c *Cache) switchMode(mode cacheMode) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.mode = mode

	if c.cancelInflights != nil {
		c.cancelInflights()
		c.cancelInflights = nil
	}
}

func (c *Cache) update(want []common.Hash) {
	wantSet := make(map[common.Hash]struct{}, len(want))
	for _, vh := range want {
		wantSet[vh] = struct{}{}
	}

	c.mu.Lock()
	if c.cancelInflights != nil {
		c.cancelInflights()
	}
	ctx, cancel := context.WithCancel(context.Background())
	c.cancelInflights = cancel

	var missing []common.Hash
	overlap := make(map[common.Hash]struct{})
	for vh := range wantSet {
		if txhash, ok := c.txHashOf[vh]; ok {
			overlap[txhash] = struct{}{}
		} else {
			missing = append(missing, vh)
		}
	}

	// Delete non-overlapping entries
	for txhash, sc := range c.entries {
		if _, ok := overlap[txhash]; ok {
			continue
		}
		delete(c.entries, txhash)
		if sc != nil {
			for _, vh := range sc.BlobHashes() {
				delete(c.txHashOf, vh)
			}
			cacheBlobsGauge.Dec(int64(len(sc.Commitments)))
		}
	}
	c.mu.Unlock()

	c.inflight.Add(1)
	go func() {
		defer c.inflight.Done()
		for _, vh := range missing {
			select {
			case <-ctx.Done():
				return
			default:
			}
			c.mu.Lock()
			_, loaded := c.txHashOf[vh]
			c.mu.Unlock()
			if loaded {
				continue
			}
			ptx := c.blobpool.getByVhash(vh)
			if ptx == nil {
				continue
			}
			sidecar := ptx.Sidecar()
			if sidecar == nil {
				continue
			}
			txhash := ptx.Tx.Hash()
			c.mu.Lock()
			c.entries[txhash] = sidecar
			for _, v := range sidecar.BlobHashes() {
				c.txHashOf[v] = txhash
			}
			cacheBlobsGauge.Inc(int64(len(sidecar.Commitments)))
			c.mu.Unlock()
		}
	}()
}

func (c *Cache) resetTimer(timer *mclock.Timer, trigger chan struct{}, interval time.Duration) {
	if *timer != nil {
		(*timer).Stop()
	}
	*timer = c.clock.AfterFunc(interval, func() {
		trigger <- struct{}{}
	})
}

// selectTxs returns the versioned hashes of the k blob transactions most
// likely to be included in upcoming blocks, sorted by execution tip cap and
// flattened across each picked tx's blobs.
func selectTxs(snapshot []txDigest, k uint) []common.Hash {
	if k <= 0 {
		return nil
	}
	sort.Slice(snapshot, func(i, j int) bool {
		return snapshot[i].tip.Gt(snapshot[j].tip)
	})
	if len(snapshot) > int(k) {
		snapshot = snapshot[:k]
	}
	var vhashes []common.Hash
	for _, d := range snapshot {
		vhashes = append(vhashes, d.vhashes...)
	}
	return vhashes
}
