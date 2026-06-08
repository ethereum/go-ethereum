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
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/txpool/txorder"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/internal/telemetry"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/holiman/uint256"
)

type cacheMode int

const (
	// In hasBlobsMode, the cache keeps the blobs that it reported
	// as available in a HasBlobs response.
	hasBlobsMode cacheMode = iota
	// In topKMode, the cache fills itself with the blobs of the top K
	// most profitable transactions.
	topKMode
)

const (
	topKTimeout     = 4 * time.Second
	hasBlobsTimeout = 1 * time.Second
)

var (
	// Cache tracks 3 metrics: cache hit, cache miss, and the number of blobs
	// it contains. Note that cache miss includes the blobs that we are actually
	// missing on the lower level (in this case, the blobpool). The amount that
	// we failed to predict) can be calculated with the telemetry span
	// (blobs.filled - cache.hit).
	cacheHitMeter   = metrics.NewRegisteredMeter("blobpool/cache/hit", nil)
	cacheMissMeter  = metrics.NewRegisteredMeter("blobpool/cache/miss", nil)
	cacheBlobsGauge = metrics.NewRegisteredGauge("blobpool/cache/blobs", nil)
)

type cachedBlob struct {
	blob       *kzg4844.Blob
	commitment kzg4844.Commitment
	proofs     []kzg4844.Proof
	version    byte
}

// Cache holds the blobs that are likely to be requested by the GetBlobs engine API.
//
// Currently it operates in two modes
//   - HasBlobsMode is triggered by the HasBlobs engine API. This stage ends
//     when `hasBlobsTimeout` elapses or the GetBlobs engine API consumes
//     the result.
//     (Note: the cache is not guaranteed to always hold such blobs, since the
//     blobpool might drop the transaction in the window between the engine API
//     response and the cache update.)
//   - TopKMode is the cache's default mode. Every `topKTimeout`, the cache selects
//     the blobs of the top K most profitable transactions, unless it is in the other mode.
//     Whenever HasBlobsMode is canceled, it falls back to TopKMode.
//
// Whenever the mode is changed, the goroutines (for cache update) started by
// each mode should be canceled to prevent redundant computation.
type Cache struct {
	mu sync.Mutex

	entries map[common.Hash]*cachedBlob // Mapping from vhash to cachedBlob

	blobpool *BlobPool

	hasBlobs chan []common.Hash // List of tx hashes that need to be pinned
	getBlobs chan struct{}

	mode cacheMode

	cancelInflights context.CancelFunc // Cancel the in-flight conversion/decode goroutines
	inflight        sync.WaitGroup     // Tracks the in-flight conversion/decode goroutines
	wg              sync.WaitGroup     // Tracks the loop goroutine

	clock mclock.Clock

	step func() // test hook fired after each loop iteration

	quit chan struct{}
}

// NewCache creates a blob cache backed by the given blobpool.
func NewCache(p *BlobPool) *Cache {
	return NewCacheForTest(p, mclock.System{}, nil)
}

// NewCacheForTest creates a blob cache for test.
// It allows injecting a clock and a step hook.
func NewCacheForTest(p *BlobPool, clock mclock.Clock, step func()) *Cache {
	c := &Cache{
		entries: make(map[common.Hash]*cachedBlob),
		blobpool: p,
		hasBlobs: make(chan []common.Hash, 1),
		getBlobs: make(chan struct{}, 1),

		mode:  topKMode,
		clock: clock,

		step: step,

		quit: make(chan struct{}),
	}

	c.wg.Add(1)
	go c.loop()
	return c
}

// Stop terminates the cache loop and blocks until it and any in-flight work
// have stopped.
func (c *Cache) Stop() {
	close(c.quit)
	c.wg.Wait()
}

// HasBlobs reports whether the blob is available (in the cache or the
// blobpool) and asks the loop to pin the ones it found.
func (c *Cache) HasBlobs(ctx context.Context, vhashes []common.Hash) []bool {
	var (
		missIdx     []int
		missVhashes []common.Hash
		needPin     []common.Hash // available vhashes
	)
	available := make([]bool, len(vhashes))

	// First check cache and pass missing ones to blobpool.
	c.mu.Lock()
	for i, vhash := range vhashes {
		if _, ok := c.entries[vhash]; ok {
			available[i] = true
			needPin = append(needPin, vhash)
		} else {
			missIdx = append(missIdx, i)
			missVhashes = append(missVhashes, vhash)
		}
	}
	c.mu.Unlock()

	if len(missVhashes) > 0 {
		pooled := c.blobpool.availableBlobs(missVhashes)
		// Merge two results
		for j, ok := range pooled {
			if ok {
				available[missIdx[j]] = true
				needPin = append(needPin, missVhashes[j])
			}
		}
	}

	select {
	case c.hasBlobs <- needPin:
		// Note that we also send the ones we already have in cache,
		// since it can be dropped from the cache before this signal is processed.
		return available
	case <-c.quit:
		return nil
	}
}

// GetBlobs returns the blobs and proofs for the given versioned hashes, serving
// them from the cache when possible and falling back to the blobpool for misses.
// Responses are placed in the order given in the request, using null for any
// missing blob.
//
// For instance, if the request is [A_versioned_hash, B_versioned_hash,
// C_versioned_hash] and blobpool has data for blobs A and C, but doesn't have
// data for B, the response MUST be [A, null, C].
//
// This is a utility method for the engine API, enabling consensus clients to
// retrieve blobs from the pools directly instead of the network.
//
// The version argument specifies the type of proofs to return, either the
// blob proofs (version 0) or the cell proofs (version 1). Proofs conversion is
// CPU intensive and prohibited explicitly.
func (c *Cache) GetBlobs(ctx context.Context, vhashes []common.Hash, version byte) (_ []*kzg4844.Blob, _ []kzg4844.Commitment, _ [][]kzg4844.Proof, err error) {
	_, span, spanEnd := telemetry.StartSpan(ctx, "blobpool.GetBlobs")
	defer spanEnd(&err)
	var (
		blobs       = make([]*kzg4844.Blob, len(vhashes))
		commitments = make([]kzg4844.Commitment, len(vhashes))
		proofs      = make([][]kzg4844.Proof, len(vhashes))

		indices = make(map[common.Hash][]int)
		filled  = make(map[common.Hash]struct{})
		misses  []common.Hash

		cacheHits int
		cacheMiss int
	)
	for i, h := range vhashes {
		indices[h] = append(indices[h], i)
	}

	c.mu.Lock()
	for _, vhash := range vhashes {
		if _, ok := filled[vhash]; ok {
			continue
		}
		filled[vhash] = struct{}{}

		cached := c.entries[vhash]

		if cached == nil {
			cacheMiss++
			misses = append(misses, vhash)
			continue
		}
		cacheHits++
		if cached.version != version {
			continue
		}
		for _, index := range indices[vhash] {
			blobs[index] = cached.blob
			commitments[index] = cached.commitment
			proofs[index] = cached.proofs
		}
	}
	c.mu.Unlock()

	if len(misses) > 0 {
		mb, mc, mp, err := c.blobpool.getBlobs(misses, version)
		if err != nil {
			return nil, nil, nil, err
		}
		for j, vhash := range misses {
			if mb[j] == nil {
				continue
			}
			for _, index := range indices[vhash] {
				blobs[index] = mb[j]
				commitments[index] = mc[j]
				proofs[index] = mp[j]
			}
		}
	}
	cacheHitMeter.Mark(int64(cacheHits))
	cacheMissMeter.Mark(int64(cacheMiss))
	span.SetAttributes(
		telemetry.IntAttribute("cache.hit", cacheHits),
		telemetry.IntAttribute("cache.miss", cacheMiss),
	)

	select {
	case c.getBlobs <- struct{}{}:
		return blobs, commitments, proofs, nil
	case <-c.quit:
		return nil, nil, nil, nil
	}
}

func (c *Cache) loop() {
	defer c.wg.Done()
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
			// switch to hasBlobs
			c.switchMode(hasBlobsMode)
			c.update(want)
			c.resetTimer(hbTimer, hbTrigger, hasBlobsTimeout)

		case <-topKTrigger:
			// switch to topK
			if c.mode != topKMode {
				continue
			}
			want := c.selectTopTxs()
			c.update(want)
			c.resetTimer(topKTimer, topKTrigger, topKTimeout)

		case <-hbTrigger:
			// hasBlobs mode is over - switch to topK
			c.switchMode(topKMode)
			want := c.selectTopTxs()
			c.update(want)
			c.resetTimer(topKTimer, topKTrigger, topKTimeout)

		case <-c.getBlobs:
			// hasBlobs mode is over -  switch to topK
			if c.mode != hasBlobsMode {
				continue
			}
			c.switchMode(topKMode)
			want := c.selectTopTxs()
			c.update(want)
			c.resetTimer(topKTimer, topKTrigger, topKTimeout)
		case <-c.quit:
			c.mu.Lock()
			if c.cancelInflights != nil {
				c.cancelInflights()
			}
			c.mu.Unlock()
			c.inflight.Wait()
			return
		}

		if c.step != nil {
			c.step()
		}
	}
}

// switchMode sets the cache mode and cancels any in-flight update from the
// previous mode.
func (c *Cache) switchMode(mode cacheMode) {
	c.mode = mode

	if c.cancelInflights != nil {
		c.cancelInflights()
		c.cancelInflights = nil
	}
}

// update updates the cache to hold the wanted vhashes. It evicts entries that
// are no longer wanted and loads the missing ones from the blobpool in the
// background.
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
	for vh := range wantSet {
		if _, ok := c.entries[vh]; !ok {
			missing = append(missing, vh)
		}
	}

	for vh := range c.entries {
		if _, ok := wantSet[vh]; ok {
			continue
		}
		delete(c.entries, vh)
		cacheBlobsGauge.Dec(1)
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
			_, loaded := c.entries[vh]
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

			c.mu.Lock()
			for i, v := range sidecar.BlobHashes() {
				if _, ok := wantSet[v]; !ok {
					continue
				}
				var pf []kzg4844.Proof
				switch sidecar.Version {
				case types.BlobSidecarVersion0:
					pf = []kzg4844.Proof{sidecar.Proofs[i]}
				case types.BlobSidecarVersion1:
					cellProofs, err := sidecar.CellProofsAt(i)
					if err != nil {
						log.Error("Failed to get cell proofs", "txhash", ptx.Tx.Hash(), "err", err)
						continue
					}
					pf = cellProofs
				}
				c.entries[v] = &cachedBlob{
					blob:       &sidecar.Blobs[i],
					commitment: sidecar.Commitments[i],
					proofs:     pf,
					version:    sidecar.Version,
				}
				cacheBlobsGauge.Inc(1)
			}
			c.mu.Unlock()
		}
	}()
}

// selectTopTxs returns the vhashes of the top K most profitable pending blob
// transactions, up to the active fork's maxBlobsPerBlock.
func (c *Cache) selectTopTxs() []common.Hash {
	p := c.blobpool
	head := p.head.Load()
	if head == nil {
		return nil
	}
	config := p.chain.Config()
	baseFee := eip1559.CalcBaseFee(config, head)

	filter := txpool.PendingFilter{
		BlobTxs: true,
		BaseFee: uint256.MustFromBig(baseFee),
	}
	if head.ExcessBlobGas != nil {
		filter.BlobFee = uint256.MustFromBig(eip4844.CalcBlobFee(config, head))
	}
	if config.IsOsaka(head.Number, head.Time) {
		filter.BlobVersion = types.BlobSidecarVersion1
	} else {
		filter.BlobVersion = types.BlobSidecarVersion0
	}
	pending, _ := p.Pending(filter)
	vhashesOf := p.vhashesByTx()

	order := txorder.NewTransactionsByPriceAndNonce(p.signer, pending, baseFee)

	// Bound the selection by the active fork's blob limit so the cache follows
	// BPO changes to maxBlobsPerBlock.
	target := uint(eip4844.MaxBlobsPerBlock(config, head.Time))

	var (
		vhashes []common.Hash
		blobs   uint
	)
	for blobs < target {
		tx, _ := order.Peek()
		if tx == nil {
			break
		}
		vh, ok := vhashesOf[tx.Hash]
		if ok {
			vhashes = append(vhashes, vh...)
			blobs += uint(len(vh))
		}
		order.Shift()
	}
	return vhashes
}

// resetTimer sets the given timer to fire on the trigger channel after the
// interval.
func (c *Cache) resetTimer(timer *mclock.Timer, trigger chan struct{}, interval time.Duration) {
	if *timer != nil {
		(*timer).Stop()
	}
	*timer = c.clock.AfterFunc(interval, func() {
		trigger <- struct{}{}
	})
}
