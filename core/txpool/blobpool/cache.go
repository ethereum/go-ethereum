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
// Every `topKTimeout`, the cache selects the blobs of the top K most profitable
// transactions, and preloads them into the cache.
//
// For HasBlobs requests, it also causes the blobs requested by the CL to be loaded.
// (Note: the cache is not guaranteed to always hold such blobs, since the blobpool might
// drop the transaction in the window between the engine API response and the cache
// update.)
type Cache struct {
	blobpool *BlobPool
	clock    mclock.Clock

	mu      sync.Mutex
	entries map[common.Hash]*cachedBlob

	// channels into loop
	quit        chan struct{}
	topkRequest chan struct{}
	topkTimer   mclock.Timer
	hasBlobsCh  chan []common.Hash // list of tx hashes that should be pinned

	step func() // test hook fired after each loop iteration

	cancelInflights context.CancelFunc // cancels the conversion/decode goroutines
	inflight        sync.WaitGroup     // tracks all in-flight conversion/decode goroutines
	wg              sync.WaitGroup     // tracks the loop goroutine
}

// NewCache creates a blob cache backed by the given blobpool.
func NewCache(p *BlobPool) *Cache {
	return newCache(p, mclock.System{}, nil)
}

// newCache creates a blob cache for testing purposes.
// It allows injecting a clock and a step hook.
func newCache(p *BlobPool, clock mclock.Clock, step func()) *Cache {
	c := &Cache{
		entries:     make(map[common.Hash]*cachedBlob),
		blobpool:    p,
		hasBlobsCh:  make(chan []common.Hash, 1),
		clock:       clock,
		step:        step,
		quit:        make(chan struct{}),
		topkRequest: make(chan struct{}, 1),
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
	case c.hasBlobsCh <- needPin:
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
		misses  []common.Hash

		cacheHits int
		cacheMiss int
	)
	for i, h := range vhashes {
		indices[h] = append(indices[h], i)
	}

	c.mu.Lock()
	for vhash, idxs := range indices {
		n := len(idxs)

		cached := c.entries[vhash]
		if cached == nil || cached.version != version {
			cacheMiss += n
			misses = append(misses, vhash)
			continue
		}
		cacheHits += n
		for _, index := range idxs {
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

	return blobs, commitments, proofs, nil
}

func (c *Cache) loop() {
	defer c.wg.Done()

	c.triggerTopK()
	for {
		select {
		case want := <-c.hasBlobsCh:
			// HasBlobs request was received.
			// Update the cache once with the requested blobs, then reschedule topK.
			c.update(want)
			c.triggerTopKAfter(hasBlobsTimeout)

		case <-c.topkRequest:
			want := c.selectTopTxs()
			c.update(want)
			c.triggerTopKAfter(topKTimeout)

		case <-c.quit:
			c.cancelUpdate()
			if c.topkTimer != nil {
				c.topkTimer.Stop()
			}
			c.inflight.Wait()
			return
		}

		if c.step != nil {
			c.step()
		}
	}
}

// cancelUpdate stops the current update.
func (c *Cache) cancelUpdate() {
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

	// Cancel the current updates.
	c.cancelUpdate()
	ctx, cancel := context.WithCancel(context.Background())
	c.cancelInflights = cancel

	c.mu.Lock()
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
				if _, exists := c.entries[v]; exists {
					continue // recompute only new entries
				}
				cellProofs, err := sidecar.CellProofsAt(i)
				if err != nil {
					log.Error("Failed to get cell proofs", "txhash", ptx.Tx.Hash(), "err", err)
					continue
				}
				c.entries[v] = &cachedBlob{
					blob:       &sidecar.Blobs[i],
					commitment: sidecar.Commitments[i],
					proofs:     cellProofs,
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

// triggerTopKAfter makes a topK selection happen after the given interval.
func (c *Cache) triggerTopKAfter(interval time.Duration) {
	if c.topkTimer != nil {
		c.topkTimer.Stop()
	}
	// drain current request to avoid triggering before the interval
	select {
	case <-c.topkRequest:
	default:
	}
	c.topkTimer = c.clock.AfterFunc(interval, c.triggerTopK)
}

// triggerTopK causes another topK selection to happen.
// Note this is safe to call from anywhere, even outside of the loop goroutine.
func (c *Cache) triggerTopK() {
	select {
	case c.topkRequest <- struct{}{}:
	default:
	}
}
