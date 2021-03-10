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
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/VictoriaMetrics/fastcache"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

var (
	// emptyRoot is the known root hash of an empty trie.
	emptyRoot = common.HexToHash("56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")

	// emptyCode is the known hash of the empty EVM bytecode.
	emptyCode = crypto.Keccak256Hash(nil)

	// accountCheckRange is the upper limit of the number of accounts involved in
	// each range check. This is a value estimated based on experience. If this
	// value is too large, the failure rate of range prove will increase. Otherwise
	// the the value is too small, the efficiency of the state recovery will decrease.
	accountCheckRange = 100

	// storageCheckRange is the upper limit of the number of storage slots involved
	// in each range check. This is a value estimated based on experience. If this
	// value is too large, the failure rate of range prove will increase. Otherwise
	// the the value is too small, the efficiency of the state recovery will decrease.
	storageCheckRange = 1024
)

// Metrics in generation
var (
	snapGeneratedAccountMeter     = metrics.NewRegisteredMeter("state/snapshot/generation/account/generated", nil)
	snapRecoveredAccountMeter     = metrics.NewRegisteredMeter("state/snapshot/generation/account/recovered", nil)
	snapWipedAccountMeter         = metrics.NewRegisteredMeter("state/snapshot/generation/account/wiped", nil)
	snapGeneratedStorageMeter     = metrics.NewRegisteredMeter("state/snapshot/generation/storage/generated", nil)
	snapRecoveredStorageMeter     = metrics.NewRegisteredMeter("state/snapshot/generation/storage/recovered", nil)
	snapWipedStorageMeter         = metrics.NewRegisteredMeter("state/snapshot/generation/storage/wiped", nil)
	snapSuccessfulRangeProofMeter = metrics.NewRegisteredMeter("state/snapshot/generation/proof/success", nil)
	snapFailedRangeProofMeter     = metrics.NewRegisteredMeter("state/snapshot/generation/proof/failure", nil)
)

// generatorStats is a collection of statistics gathered by the snapshot generator
// for logging purposes.
type generatorStats struct {
	origin   uint64             // Origin prefix where generation started
	start    time.Time          // Timestamp when generation started
	accounts uint64             // Number of accounts indexed(generated or recovered)
	slots    uint64             // Number of storage slots indexed(generated or recovered)
	storage  common.StorageSize // Total account and storage slot size(generation or recovery)
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
func generateSnapshot(diskdb ethdb.KeyValueStore, triedb *trie.Database, cache int, root common.Hash) *diskLayer {
	// Create a new disk layer with an initialized state marker at zero
	var (
		stats     = &generatorStats{start: time.Now()}
		batch     = diskdb.NewBatch()
		genMarker = []byte{} // Initialized but empty!
	)
	rawdb.WriteSnapshotRoot(batch, root)
	journalProgress(batch, genMarker, stats)
	if err := batch.Write(); err != nil {
		log.Crit("Failed to write initialized state marker", "error", err)
	}
	base := &diskLayer{
		diskdb:     diskdb,
		triedb:     triedb,
		root:       root,
		cache:      fastcache.New(cache * 1024 * 1024),
		genMarker:  genMarker,
		genPending: make(chan struct{}),
		genAbort:   make(chan chan *generatorStats),
	}
	go base.generate(stats)
	log.Debug("Start snapshot generation", "root", root)
	return base
}

// journalProgress persists the generator stats into the database to resume later.
func journalProgress(db ethdb.KeyValueWriter, marker []byte, stats *generatorStats) {
	// Write out the generator marker. Note it's a standalone disk layer generator
	// which is not mixed with journal. It's ok if the generator is persisted while
	// journal is not.
	entry := journalGenerator{
		Done:   marker == nil,
		Marker: marker,
	}
	if stats != nil {
		entry.Accounts = stats.accounts
		entry.Slots = stats.slots
		entry.Storage = uint64(stats.storage)
	}
	blob, err := rlp.EncodeToBytes(entry)
	if err != nil {
		panic(err) // Cannot happen, here to catch dev errors
	}
	var logstr string
	switch {
	case marker == nil:
		logstr = "done"
	case bytes.Equal(marker, []byte{}):
		logstr = "empty"
	case len(marker) == common.HashLength:
		logstr = fmt.Sprintf("%#x", marker)
	default:
		logstr = fmt.Sprintf("%#x:%#x", marker[:common.HashLength], marker[common.HashLength:])
	}
	log.Debug("Journalled generator progress", "progress", logstr)
	rawdb.WriteSnapshotGenerator(db, blob)
}

// proveRange proves the state segment with particular prefix is "valid".
// The iteration start point will be assigned if the iterator is restored from
// the last interruption. Max will be assigned in order to limit the maximum
// amount of data involved in each iteration.
func (dl *diskLayer) proveRange(root common.Hash, tr *trie.SecureTrie, prefix []byte, kind string, origin []byte, max int, onValue func([]byte) ([]byte, error)) ([][]byte, [][]byte, []byte, bool, error) {
	var (
		keys  [][]byte
		vals  [][]byte
		count int
		last  []byte
		proof = rawdb.NewMemoryDatabase()
		iter  = dl.diskdb.NewIterator(prefix, origin)
	)
	for iter.Next() && count < max {
		key := iter.Key()
		if len(key) != len(prefix)+common.HashLength {
			continue
		}
		if !bytes.HasPrefix(key, prefix) {
			continue
		}
		last = common.CopyBytes(key[len(prefix):])
		keys = append(keys, common.CopyBytes(key[len(prefix):]))

		if onValue == nil {
			vals = append(vals, common.CopyBytes(iter.Value()))
		} else {
			converted, err := onValue(common.CopyBytes(iter.Value()))
			if err != nil {
				log.Debug("Failed to convert the flat state", "kind", kind, "key", common.BytesToHash(key[len(prefix):]), "error", err)
				return nil, nil, last, false, err
			}
			vals = append(vals, converted)
		}
		count += 1
	}
	// Generate the Merkle proofs for the first and last element
	if origin == nil {
		origin = common.Hash{}.Bytes()
	}
	if err := tr.Prove(origin, 0, proof); err != nil {
		log.Debug("Failed to prove range", "kind", kind, "origin", origin, "err", err)
		return nil, nil, last, false, err
	}
	if last != nil {
		if err := tr.Prove(last, 0, proof); err != nil {
			log.Debug("Failed to prove range", "kind", kind, "last", last, "err", err)
			return nil, nil, last, false, err
		}
	}
	// Verify the state segment with range prover, ensure that all flat states
	// in this range correspond to merkle trie.
	_, _, _, cont, err := trie.VerifyRangeProof(root, origin, last, keys, vals, proof)
	if err != nil {
		return nil, nil, last, false, err
	}
	// Range prover says the trie still has some elements on the right side but
	// the database is exhausted, then data loss is detected.
	if cont && count < max {
		return nil, nil, last, false, errors.New("data loss in the state range")
	}
	return keys, vals, last, !cont, nil
}

// genRange generates the state segment with particular prefix. Generation can
// either verify the correctness of existing state through rangeproof and skip
// generation, or iterate trie to regenerate state on demand.
func (dl *diskLayer) genRange(root common.Hash, prefix []byte, kind string, origin []byte, max int, stats *generatorStats, onState func(key []byte, val []byte, regen bool) error, onValue func([]byte) ([]byte, error)) (bool, []byte, error) {
	tr, err := trie.NewSecure(root, dl.triedb)
	if err != nil {
		// The account trie is missing (GC), surf the chain until one becomes available
		stats.Log("Trie missing, state snapshotting paused", root, dl.genMarker)

		abort := <-dl.genAbort
		abort <- stats
		return false, nil, errors.New("trie is missing")
	}
	// Use range prover to check the validity of the flat state in the range
	keys, vals, last, exhausted, err := dl.proveRange(root, tr, prefix, kind, origin, max, onValue)
	if err == nil {
		snapSuccessfulRangeProofMeter.Mark(1)
		log.Debug("Proved state range", "kind", kind, "prefix", prefix, "origin", origin, "last", last)

		// The verification is passed, process each state with the given
		// callback function. If this state represents a contract, the
		// corresponding storage check will be performed in the callback
		for i := 0; i < len(keys); i++ {
			if err := onState(keys[i], vals[i], false); err != nil {
				return false, nil, err
			}
		}
		log.Debug("Recovered state range", "kind", kind, "prefix", prefix, "origin", origin, "last", last)
		return exhausted, last, nil
	}
	snapFailedRangeProofMeter.Mark(1)
	log.Debug("Detected outdated state range", "kind", kind, "prefix", prefix, "origin", origin, "last", last)

	// The verifcation is failed, the flat state in this range cannot match the
	// merkle trie. Alternatively, use the fallback generation mechanism to regenerate
	// the correct flat state by iterating trie. But wiping the existent outdated flat
	// data in this range first.
	if last != nil {
		// Note if the returned last is nil(no more flat state can be found in the database),
		// the wiping can be skipped.
		wipedMeter := snapWipedAccountMeter
		if kind == "storage" {
			wipedMeter = snapWipedStorageMeter
		}
		limit := increseKey(common.CopyBytes(last))
		if err := wipeKeyRange(dl.diskdb, kind, prefix, origin, limit, len(prefix)+common.HashLength, wipedMeter, false); err != nil {
			return false, nil, err
		}
		log.Debug("Wiped currupted state range", "kind", kind, "prefix", prefix, "origin", origin, "limit", limit)
	}
	iter := trie.NewIterator(tr.NodeIterator(origin))
	for iter.Next() {
		if last != nil && bytes.Compare(iter.Key, last) > 0 {
			log.Debug("Regenerated state range", "kind", kind, "prefix", prefix, "origin", origin, "last", last)
			return false, last, nil // Apparently the trie is not exhausted
		}
		if err := onState(iter.Key, iter.Value, true); err != nil {
			return false, nil, err
		}
	}
	if iter.Err != nil {
		return false, nil, iter.Err
	}
	log.Debug("Regenerated state range", "kind", kind, "prefix", prefix, "origin", origin, "last", last)
	return true, nil, nil // The entire trie is exhausted
}

// generate is a background thread that iterates over the state and storage tries,
// constructing the state snapshot. All the arguments are purely for statistics
// gathering and logging, since the method surfs the blocks as they arrive, often
// being restarted.
func (dl *diskLayer) generate(stats *generatorStats) {
	var accMarker []byte
	if len(dl.genMarker) > 0 { // []byte{} is the start, use nil for that
		accMarker = dl.genMarker[:common.HashLength]
	}
	var (
		batch     = dl.diskdb.NewBatch()
		logged    = time.Now()
		accOrigin = common.CopyBytes(accMarker)
	)
	stats.Log("Resuming state snapshot generation", dl.root, dl.genMarker)

	for {
		exhausted, last, err := dl.genRange(dl.root, rawdb.SnapshotAccountPrefix, "account", accOrigin, accountCheckRange, stats, func(key []byte, val []byte, regen bool) error {
			// Retrieve the current account and flatten it into the internal format
			accountHash := common.BytesToHash(key)

			var acc struct {
				Nonce    uint64
				Balance  *big.Int
				Root     common.Hash
				CodeHash []byte
			}
			if err := rlp.DecodeBytes(val, &acc); err != nil {
				log.Crit("Invalid account encountered during snapshot creation", "err", err)
			}
			data := SlimAccountRLP(acc.Nonce, acc.Balance, acc.Root, acc.CodeHash)

			// If the account is not yet in-progress, write it out
			if accMarker == nil || !bytes.Equal(accountHash[:], accMarker) {
				if regen {
					rawdb.WriteAccountSnapshot(batch, accountHash, data)
					snapGeneratedAccountMeter.Mark(1)
				} else {
					snapRecoveredAccountMeter.Mark(1)
				}
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
				// Flush out the batch anyway no matter it's empty or not.
				// It's possible that all the states are recovered and the
				// generation indeed makes progress.
				marker := accountHash[:]
				journalProgress(batch, marker, stats)

				batch.Write()
				batch.Reset()

				dl.lock.Lock()
				dl.genMarker = marker
				dl.lock.Unlock()

				if abort != nil {
					stats.Log("Aborting state snapshot generation", dl.root, accountHash[:])
					abort <- stats
					return errors.New("aborted")
				}
			}
			// If the iterated account is the contract, create a further loop to
			// verify or regenerate the contract storage.
			if acc.Root != emptyRoot {
				var storeMarker []byte
				if accMarker != nil && bytes.Equal(accountHash[:], accMarker) && len(dl.genMarker) > common.HashLength {
					storeMarker = dl.genMarker[common.HashLength:]
				}
				var storeOrigin = common.CopyBytes(storeMarker)
				for {
					exhausted, last, err := dl.genRange(acc.Root, append(rawdb.SnapshotStoragePrefix, accountHash.Bytes()...), "storage", storeOrigin, storageCheckRange, stats, func(key []byte, val []byte, regen bool) error {
						if regen {
							rawdb.WriteStorageSnapshot(batch, accountHash, common.BytesToHash(key), val)
							snapGeneratedStorageMeter.Mark(1)
						} else {
							snapRecoveredStorageMeter.Mark(1)
						}
						stats.storage += common.StorageSize(1 + 2*common.HashLength + len(val))
						stats.slots++

						// If we've exceeded our batch allowance or termination was requested, flush to disk
						var abort chan *generatorStats
						select {
						case abort = <-dl.genAbort:
						default:
						}
						if batch.ValueSize() > ethdb.IdealBatchSize || abort != nil {
							// Flush out the batch anyway no matter it's empty or not.
							// It's possible that all the states are recovered and the
							// generation indeed makes progress.
							marker := append(accountHash[:], key...)
							journalProgress(batch, marker, stats)

							batch.Write()
							batch.Reset()

							dl.lock.Lock()
							dl.genMarker = marker
							dl.lock.Unlock()

							if abort != nil {
								stats.Log("Aborting state snapshot generation", dl.root, append(accountHash[:], key...))
								abort <- stats
								return errors.New("aborted")
							}
							if time.Since(logged) > 8*time.Second {
								stats.Log("Generating state snapshot", dl.root, append(accountHash[:], key...))
								logged = time.Now()
							}
						}
						return nil
					}, nil)
					if err != nil {
						return err
					}
					if exhausted {
						return nil
					}
					storeOrigin = increseKey(last)
					if storeOrigin == nil {
						return nil // special case, the last is 0xffffffff...fff
					}
				}
			}
			if time.Since(logged) > 8*time.Second {
				stats.Log("Generating state snapshot", dl.root, key)
				logged = time.Now()
			}
			// Some account processed, unmark the marker
			accMarker = nil
			return nil
		}, FullAccountRLP)
		if err != nil {
			abort := <-dl.genAbort
			abort <- nil
			return
		}
		if exhausted {
			break
		}
		accOrigin = increseKey(last)
		if accOrigin == nil {
			break // special case, the last is 0xffffffff...fff
		}
	}
	// Snapshot fully generated, set the marker to nil.
	// Note even there is nothing to commit, persist the
	// generator anyway to mark the snapshot is complete.
	journalProgress(batch, nil, stats)
	batch.Write()
	batch.Reset()

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

// increseKey increase the input key by one bit. Return nil if the entire
// addition operation overflows,
func increseKey(key []byte) []byte {
	for i := len(key) - 1; i >= 0; i-- {
		key[i]++
		if key[i] != 0x0 {
			return key
		}
	}
	return nil
}
