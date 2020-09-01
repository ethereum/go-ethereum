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

package snapshot

import (
	"bytes"
	"encoding/binary"
	"math/big"
	"time"

	"github.com/VictoriaMetrics/fastcache"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

var (
	// emptyRoot is the known root hash of an empty trie.
	emptyRoot = common.HexToHash("56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")

	// emptyCode is the known hash of the empty EVM bytecode.
	emptyCode = crypto.Keccak256Hash(nil)
)

// generatorStats is a collection of statistics gathered by the snapshot generator
// for logging purposes.
type generatorStats struct {
	wiping   chan struct{}      // Notification channel if wiping is in progress
	origin   uint64             // Origin prefix where generation started
	start    time.Time          // Timestamp when generation started
	accounts uint64             // Number of accounts indexed
	slots    uint64             // Number of storage slots indexed
	storage  common.StorageSize // Account and storage slot size
}

// Log creates an contextual log with the given message and the context pulled
// from the internally maintained statistics.
func (gs *generatorStats) Log(msg string, root common.Hash, marker []byte) {
	var ctx []interface{}
	if root != (common.Hash{}) {
		ctx = append(ctx, []interface{}{"root", root}...)
	}
	// Figure out whether we're after or within an account
	switch len(marker) {
	case common.HashLength:
		ctx = append(ctx, []interface{}{"at", common.BytesToHash(marker)}...)
	case 2 * common.HashLength:
		ctx = append(ctx, []interface{}{
			"in", common.BytesToHash(marker[:common.HashLength]),
			"at", common.BytesToHash(marker[common.HashLength:]),
		}...)
	}
	// Add the usual measurements
	ctx = append(ctx, []interface{}{
		"accounts", gs.accounts,
		"slots", gs.slots,
		"storage", gs.storage,
		"elapsed", common.PrettyDuration(time.Since(gs.start)),
	}...)
	// Calculate the estimated indexing time based on current stats
	if len(marker) > 0 {
		if done := binary.BigEndian.Uint64(marker[:8]) - gs.origin; done > 0 {
			left := math.MaxUint64 - binary.BigEndian.Uint64(marker[:8])

			speed := done/uint64(time.Since(gs.start)/time.Millisecond+1) + 1 // +1s to avoid division by zero
			ctx = append(ctx, []interface{}{
				"eta", common.PrettyDuration(time.Duration(left/speed) * time.Millisecond),
			}...)
		}
	}
	log.Info(msg, ctx...)
}

// generateSnapshot regenerates a brand new snapshot based on an existing state
// database and head block asynchronously. The snapshot is returned immediately
// and generation is continued in the background until done.
func generateSnapshot(diskdb ethdb.KeyValueStore, triedb *trie.Database, cache int, root common.Hash, wiper chan struct{}) *diskLayer {
	// Wipe any previously existing snapshot from the database if no wiper is
	// currently in progress.
	if wiper == nil {
		wiper = wipeSnapshot(diskdb, true)
	}
	// Create a new disk layer with an initialized state marker at zero
	rawdb.WriteSnapshotRoot(diskdb, root)

	base := &diskLayer{
		diskdb:     diskdb,
		triedb:     triedb,
		root:       root,
		cache:      fastcache.New(cache * 1024 * 1024),
		genMarker:  []byte{}, // Initialized but empty!
		genPending: make(chan struct{}),
		genAbort:   make(chan chan *generatorStats),
	}
	go base.generate(&generatorStats{wiping: wiper, start: time.Now()})
	return base
}

// generate is a background thread that iterates over the state and storage tries,
// constructing the state snapshot. All the arguments are purely for statistics
// gethering and logging, since the method surfs the blocks as they arrive, often
// being restarted.
func (dl *diskLayer) generate(stats *generatorStats) {
	// If a database wipe is in operation, wait until it's done
	if stats.wiping != nil {
		stats.Log("Wiper running, state snapshotting paused", common.Hash{}, dl.genMarker)
		select {
		// If wiper is done, resume normal mode of operation
		case <-stats.wiping:
			stats.wiping = nil
			stats.start = time.Now()

		// If generator was aboted during wipe, return
		case abort := <-dl.genAbort:
			abort <- stats
			return
		}
	}
	// Create an account and state iterator pointing to the current generator marker
	accTrie, err := trie.NewSecure(dl.root, dl.triedb)
	if err != nil {
		// The account trie is missing (GC), surf the chain until one becomes available
		stats.Log("Trie missing, state snapshotting paused", dl.root, dl.genMarker)

		abort := <-dl.genAbort
		abort <- stats
		return
	}
	stats.Log("Resuming state snapshot generation", dl.root, dl.genMarker)

	var accMarker []byte
	if len(dl.genMarker) > 0 { // []byte{} is the start, use nil for that
		accMarker = dl.genMarker[:common.HashLength]
	}
	accIt := trie.NewIterator(accTrie.NodeIterator(accMarker))
	batch := dl.diskdb.NewBatch()

	// Iterate from the previous marker and continue generating the state snapshot
	logged := time.Now()
	for accIt.Next() {
		// Retrieve the current account and flatten it into the internal format
		accountHash := common.BytesToHash(accIt.Key)

		var acc struct {
			Nonce    uint64
			Balance  *big.Int
			Root     common.Hash
			CodeHash []byte
		}
		if err := rlp.DecodeBytes(accIt.Value, &acc); err != nil {
			log.Crit("Invalid account encountered during snapshot creation", "err", err)
		}
		data := SlimAccountRLP(acc.Nonce, acc.Balance, acc.Root, acc.CodeHash)

		// If the account is not yet in-progress, write it out
		if accMarker == nil || !bytes.Equal(accountHash[:], accMarker) {
			rawdb.WriteAccountSnapshot(batch, accountHash, data)
			stats.storage += common.StorageSize(1 + common.HashLength + len(data))
			stats.accounts++
		}
		// If we've exceeded our batch allowance or termination was requested, flush to disk
		var abort chan *generatorStats
		select {
		case abort = <-dl.genAbort:
		default:
		}
		if batch.ValueSize() > ethdb.IdealBatchSize || abort != nil {
			// Only write and set the marker if we actually did something useful
			if batch.ValueSize() > 0 {
				batch.Write()
				batch.Reset()

				dl.lock.Lock()
				dl.genMarker = accountHash[:]
				dl.lock.Unlock()
			}
			if abort != nil {
				stats.Log("Aborting state snapshot generation", dl.root, accountHash[:])
				abort <- stats
				return
			}
		}
		// If the account is in-progress, continue where we left off (otherwise iterate all)
		if acc.Root != emptyRoot {
			storeTrie, err := trie.NewSecure(acc.Root, dl.triedb)
			if err != nil {
				log.Crit("Storage trie inaccessible for snapshot generation", "err", err)
			}
			var storeMarker []byte
			if accMarker != nil && bytes.Equal(accountHash[:], accMarker) && len(dl.genMarker) > common.HashLength {
				storeMarker = dl.genMarker[common.HashLength:]
			}
			storeIt := trie.NewIterator(storeTrie.NodeIterator(storeMarker))
			for storeIt.Next() {
				rawdb.WriteStorageSnapshot(batch, accountHash, common.BytesToHash(storeIt.Key), storeIt.Value)
				stats.storage += common.StorageSize(1 + 2*common.HashLength + len(storeIt.Value))
				stats.slots++

				// If we've exceeded our batch allowance or termination was requested, flush to disk
				var abort chan *generatorStats
				select {
				case abort = <-dl.genAbort:
				default:
				}
				if batch.ValueSize() > ethdb.IdealBatchSize || abort != nil {
					// Only write and set the marker if we actually did something useful
					if batch.ValueSize() > 0 {
						batch.Write()
						batch.Reset()

						dl.lock.Lock()
						dl.genMarker = append(accountHash[:], storeIt.Key...)
						dl.lock.Unlock()
					}
					if abort != nil {
						stats.Log("Aborting state snapshot generation", dl.root, append(accountHash[:], storeIt.Key...))
						abort <- stats
						return
					}
				}
			}
		}
		if time.Since(logged) > 8*time.Second {
			stats.Log("Generating state snapshot", dl.root, accIt.Key)
			logged = time.Now()
		}
		// Some account processed, unmark the marker
		accMarker = nil
	}
	// Snapshot fully generated, set the marker to nil
	if batch.ValueSize() > 0 {
		batch.Write()
	}
	log.Info("Generated state snapshot", "accounts", stats.accounts, "slots", stats.slots,
		"storage", stats.storage, "elapsed", common.PrettyDuration(time.Since(stats.start)))

	dl.lock.Lock()
	dl.genMarker = nil
	close(dl.genPending)
	dl.lock.Unlock()

	// Someone will be looking for us, wait it out
	abort := <-dl.genAbort
	abort <- nil
}
