// Copyright 2019 The go-ethereum Authors
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

package trie

import (
	"encoding/binary"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	bloomfilter "github.com/holiman/bloomfilter/v2"
)

var (
	bloomAddMeter   = metrics.NewRegisteredMeter("trie/bloom/add", nil)
	bloomLoadMeter  = metrics.NewRegisteredMeter("trie/bloom/load", nil)
	bloomTestMeter  = metrics.NewRegisteredMeter("trie/bloom/test", nil)
	bloomMissMeter  = metrics.NewRegisteredMeter("trie/bloom/miss", nil)
	bloomFaultMeter = metrics.NewRegisteredMeter("trie/bloom/fault", nil)
	bloomErrorGauge = metrics.NewRegisteredGauge("trie/bloom/error", nil)
)

// SyncBloom is a bloom filter used during fast sync to quickly decide if a trie
// node or contract code already exists on disk or not. It self populates from the
// provided disk database on creation in a background thread and will only start
// returning live results once that's finished.
type SyncBloom struct {
	bloom   *bloomfilter.Filter
	inited  uint32
	closer  sync.Once
	closed  uint32
	pend    sync.WaitGroup
	closeCh chan struct{}
}

// NewSyncBloom creates a new bloom filter of the given size (in megabytes) and
// initializes it from the database. The bloom is hard coded to use 3 filters.
func NewSyncBloom(memory uint64, database ethdb.Iteratee) *SyncBloom {
	// Create the bloom filter to track known trie nodes
	bloom, err := bloomfilter.New(memory*1024*1024*8, 4)
	if err != nil {
		panic(fmt.Sprintf("failed to create bloom: %v", err))
	}
	log.Info("Allocated fast sync bloom", "size", common.StorageSize(memory*1024*1024))

	// Assemble the fast sync bloom and init it from previous sessions
	b := &SyncBloom{
		bloom:   bloom,
		closeCh: make(chan struct{}),
	}
	b.pend.Add(2)
	go func() {
		defer b.pend.Done()
		b.init(database)
	}()
	go func() {
		defer b.pend.Done()
		b.meter()
	}()
	return b
}

// init iterates over the database, pushing every trie hash into the bloom filter.
func (b *SyncBloom) init(database ethdb.Iteratee) {
	// Iterate over the database, but restart every now and again to avoid holding
	// a persistent snapshot since fast sync can push a ton of data concurrently,
	// bloating the disk.
	//
	// Note, this is fine, because everything inserted into leveldb by fast sync is
	// also pushed into the bloom directly, so we're not missing anything when the
	// iterator is swapped out for a new one.
	it := database.NewIterator(nil, nil)

	var (
		start = time.Now()
		swap  = time.Now()
	)
	for it.Next() && atomic.LoadUint32(&b.closed) == 0 {
		// If the database entry is a trie node, add it to the bloom
		key := it.Key()
		if len(key) == common.HashLength {
			b.bloom.AddHash(binary.BigEndian.Uint64(key))
			bloomLoadMeter.Mark(1)
		} else if ok, hash := rawdb.IsCodeKey(key); ok {
			// If the database entry is a contract code, add it to the bloom
			b.bloom.AddHash(binary.BigEndian.Uint64(hash))
			bloomLoadMeter.Mark(1)
		}
		// If enough time elapsed since the last iterator swap, restart
		if time.Since(swap) > 8*time.Second {
			key := common.CopyBytes(it.Key())

			it.Release()
			it = database.NewIterator(nil, key)

			log.Info("Initializing state bloom", "items", b.bloom.N(), "errorrate", b.bloom.FalsePosititveProbability(), "elapsed", common.PrettyDuration(time.Since(start)))
			swap = time.Now()
		}
	}
	it.Release()

	// Mark the bloom filter inited and return
	log.Info("Initialized state bloom", "items", b.bloom.N(), "errorrate", b.bloom.FalsePosititveProbability(), "elapsed", common.PrettyDuration(time.Since(start)))
	atomic.StoreUint32(&b.inited, 1)
}

// meter periodically recalculates the false positive error rate of the bloom
// filter and reports it in a metric.
func (b *SyncBloom) meter() {
	// check every second
	tick := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-tick.C:
			// Report the current error ration. No floats, lame, scale it up.
			bloomErrorGauge.Update(int64(b.bloom.FalsePosititveProbability() * 100000))
		case <-b.closeCh:
			return
		}
	}
}

// Close terminates any background initializer still running and releases all the
// memory allocated for the bloom.
func (b *SyncBloom) Close() error {
	b.closer.Do(func() {
		// Ensure the initializer is stopped
		atomic.StoreUint32(&b.closed, 1)
		close(b.closeCh)
		b.pend.Wait()

		// Wipe the bloom, but mark it "uninited" just in case someone attempts an access
		log.Info("Deallocated state bloom", "items", b.bloom.N(), "errorrate", b.bloom.FalsePosititveProbability())

		atomic.StoreUint32(&b.inited, 0)
		b.bloom = nil
	})
	return nil
}

// Add inserts a new trie node hash into the bloom filter.
func (b *SyncBloom) Add(hash []byte) {
	if atomic.LoadUint32(&b.closed) == 1 {
		return
	}
	b.bloom.AddHash(binary.BigEndian.Uint64(hash))
	bloomAddMeter.Mark(1)
}

// Contains tests if the bloom filter contains the given hash:
//   - false: the bloom definitely does not contain hash
//   - true:  the bloom maybe contains hash
//
// While the bloom is being initialized, any query will return true.
func (b *SyncBloom) Contains(hash []byte) bool {
	bloomTestMeter.Mark(1)
	if atomic.LoadUint32(&b.inited) == 0 {
		// We didn't load all the trie nodes from the previous run of Geth yet. As
		// such, we can't say for sure if a hash is not present for anything. Until
		// the init is done, we're faking "possible presence" for everything.
		return true
	}
	// Bloom initialized, check the real one and report any successful misses
	maybe := b.bloom.ContainsHash(binary.BigEndian.Uint64(hash))
	if !maybe {
		bloomMissMeter.Mark(1)
	}
	return maybe
}
