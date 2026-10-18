// Copyright 2026 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"context"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/metrics"
)

// Writing the namespace runs the node database's compaction debt into
// gigabytes, and geth's pebble answers with up to a compaction per core. Once
// it does, the import's stretch of compactions rewrites 2-3× the bytes two
// compactions would, mostly in the bottom level, and block processing's disk
// reads queue behind them. An online import therefore caps the database at
// importCompactions. With fewer compactions level 0 could outgrow pebble's
// stop-writes threshold (12 sublevels), which would stall the node's writes
// too: past paceMaxSublevels the import's own writes wait instead.
const (
	importCompactions = 2
	paceMaxSublevels  = 6
	paceInterval      = 100 * time.Millisecond
)

// The pace's counters: writes held back by a deep level 0, and the
// nanoseconds they waited in all.
var (
	paceWaitsCounter  = metrics.NewRegisteredCounter("pbt/import/pace/waits", nil)
	paceWaitedCounter = metrics.NewRegisteredCounter("pbt/import/pace/waited", nil)
)

// compactionPace holds writes back while a database's level 0 is deep.
type compactionPace struct {
	db   ethdb.CompactionController
	mu   sync.Mutex
	next time.Time // until then, the last check stands
}

// wait returns once the database's level 0 is shallow enough, or with ctx's
// error once it is cancelled.
func (p *compactionPace) wait(ctx context.Context) error {
	start := time.Now()
	defer func() {
		if waited := time.Since(start); waited >= paceInterval {
			paceWaitsCounter.Inc(1)
			paceWaitedCounter.Inc(int64(waited))
		}
	}()
	p.mu.Lock()
	defer p.mu.Unlock()
	for ctx.Err() == nil && time.Now().After(p.next) {
		if p.db.Level0Sublevels() <= paceMaxSublevels {
			p.next = time.Now().Add(paceInterval)
			break
		}
		select {
		case <-time.After(paceInterval):
		case <-ctx.Done():
		}
	}
	return ctx.Err()
}

// pacedDB is a database whose batches wait on pace before they write.
type pacedDB struct {
	ethdb.Database
	ctx  context.Context
	pace *compactionPace
}

func (d *pacedDB) NewBatch() ethdb.Batch {
	return &pacedBatch{d.Database.NewBatch(), d}
}

func (d *pacedDB) NewBatchWithSize(size int) ethdb.Batch {
	return &pacedBatch{d.Database.NewBatchWithSize(size), d}
}

type pacedBatch struct {
	ethdb.Batch
	db *pacedDB
}

func (b *pacedBatch) Write() error {
	if err := b.db.pace.wait(b.db.ctx); err != nil {
		return err
	}
	return b.Batch.Write()
}
