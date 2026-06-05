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
	hasBlobsMode cacheMode = iota
	topKMode
)

const (
	topKTimeout     = 4 * time.Second
	hasBlobsTimeout = 1 * time.Second
)

var (
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

type Cache struct {
	mu sync.Mutex

	entries  map[common.Hash]*cachedBlob // vhash -> blob, commitment, proofs
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
		entries:  make(map[common.Hash]*cachedBlob, cap),
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

	for _, vhash := range vhashes {
		if _, ok := filled[vhash]; ok {
			continue
		}
		filled[vhash] = struct{}{}

		c.mu.Lock()
		cached := c.entries[vhash]
		c.mu.Unlock()

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
			want := c.selectKTxs()
			c.update(want)
			c.resetTimer(topKTimer, topKTrigger, topKTimeout)

		case <-hbTrigger:
			c.switchMode(topKMode)
			want := c.selectKTxs()
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
			want := c.selectKTxs()
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
				if _, ok := c.entries[v]; ok {
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

func (c *Cache) selectKTxs() []common.Hash {
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

	var (
		vhashes []common.Hash
		blobs   uint
	)
	for blobs < c.capacity {
		tx, _ := order.Peek()
		if tx == nil {
			break
		}
		vh := vhashesOf[tx.Hash]
		vhashes = append(vhashes, vh...)
		blobs += uint(len(vh))
		order.Shift()
	}
	return vhashes
}

func (c *Cache) resetTimer(timer *mclock.Timer, trigger chan struct{}, interval time.Duration) {
	if *timer != nil {
		(*timer).Stop()
	}
	*timer = c.clock.AfterFunc(interval, func() {
		trigger <- struct{}{}
	})
}
