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
	"github.com/ethereum/go-ethereum/common/hexutil"
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
	accountCheckRange = 128

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

// proofResult is returned by the range provers
type proofResult struct {
	// These fields are set if the verification failed.
	staleKeys   [][]byte // The stale keys to be deleted or overwritten (nil if verification passed).
	staleValues [][]byte // The stale values to be deleted or overwritten (nil if verification passed).

	// These fields are used/set if the verification passed
	continued bool   // true if there are more elements in this trie.
	last      []byte // the last key (if check was successfull).
	count     int    // number of elements (set if verification passed)

	tr *trie.Trie // trie for the given root, which can be used by the caller afterwards (may be nil)
}

// proveStorageRange checks whether the snapshot storage range is already complete, and if not,
// recovers the data using the trie.
func (dl *diskLayer) proveStorageRange(root common.Hash, accountHash common.Hash, origin []byte, max int) (*proofResult, error) {
	var (
		keys    = make([][]byte, 0) // Init to non-nil, to differentiate against nil returnvalue which means OK
		vals    = make([][]byte, 0)
		prefix  = append(rawdb.SnapshotStoragePrefix, accountHash.Bytes()...)
		iter    = dl.diskdb.NewIterator(prefix, origin)
		aborted bool
		last    []byte
	)
	for iter.Next() {
		key := iter.Key()
		if len(key) != len(prefix)+common.HashLength {
			continue // random state with same prefix
		}
		if len(keys) == max {
			// There exists more data in the snapshot, but we've reached the
			// max elems per iteration
			aborted = true
			break
		}
		keys = append(keys, common.CopyBytes(key[len(prefix):]))
		vals = append(vals, common.CopyBytes(iter.Value()))
	}
	iter.Release()
	if len(keys) > 0 {
		last = keys[len(keys)-1]
	}
	// If the elements cover the entire storage range, then we don't need to use the prover
	// to check validity. We can just feed the key/vals into a stacktrie and obtain the
	// root hash
	if len(origin) == 0 && !aborted {
		stackTr := trie.NewStackTrie(nil)
		for i, key := range keys {
			stackTr.TryUpdate(key, vals[i])
		}
		if gotRoot := stackTr.Hash(); gotRoot != root {
			// This trie needs to be regenerated
			log.Debug("Failed to prove storage range", "origin", hexutil.Encode(origin), "root", root, "got", gotRoot)
			return &proofResult{
				staleKeys:   keys,
				staleValues: vals,
			}, nil
		}
		return &proofResult{
			last:  last,
			count: len(keys),
		}, nil
	}
	// The trie was too large to handle in one iteration. Thus, we need to do
	// use the prover
	if origin == nil {
		origin = common.Hash{}.Bytes()
	}
	tr, err := trie.New(root, dl.triedb)
	if err != nil { // This error is special, it needs to abort the whole procedure
		return nil, fmt.Errorf("trie %#x is missing", root)
	}
	// Generate the Merkle proofs for the first and last element
	proof := rawdb.NewMemoryDatabase()
	if err := tr.Prove(origin, 0, proof); err != nil {
		log.Debug("Failed to prove storage range", "origin", hexutil.Encode(origin))
		return &proofResult{
			staleKeys:   keys,
			staleValues: vals,
			tr:          tr,
		}, nil
	}
	if last != nil {
		if err := tr.Prove(last, 0, proof); err != nil {
			log.Debug("Failed to prove storage range", "last", hexutil.Encode(last), "err", err)
			return &proofResult{
				staleKeys:   keys,
				staleValues: vals,
				tr:          tr,
			}, nil
		}
	}
	// Verify the segment with range prover, ensure that all flat states
	// in this range correspond to merkle trie.
	_, _, _, cont, err := trie.VerifyRangeProof(root, origin, last, keys, vals, proof)
	if err != nil {
		log.Debug("Failed to verify storage range proof", "last", hexutil.Encode(last))
		return &proofResult{
			staleKeys:   keys,
			staleValues: vals,
			tr:          tr,
		}, nil
	}
	// Range prover says the trie still has some elements on the right side but
	// the snap database is exhausted, then data loss is detected.
	if cont && !aborted {
		// TODO (@holiman or @rlj) investigate:
		// This is an odd case. How can this happen? It means our snap data
		// is correct, but not compelete? Maybe we can just continue here without
		// deleting the existing data.
		// We definitely need a testcase for this path
		return &proofResult{
			continued: true,
			tr:        tr,
			last:      last,
			count:     len(keys),
		}, nil
		//return nil, false, errors.New("data loss in the state range")
	}
	// All ok
	return &proofResult{
		continued: cont,
		tr:        tr,
		last:      last,
		count:     len(keys),
	}, nil
}

// proveAccountRange proves the state segment with particular prefix is "valid".
// The iteration start point will be assigned if the iterator is restored from
// the last interruption. Max will be assigned in order to limit the maximum
// amount of data involved in each iteration.
func (dl *diskLayer) proveAccountRange(root common.Hash, tr *trie.Trie, origin []byte, max int) ([][]byte, [][]byte, []byte, bool, error) {
	var (
		prefix  = rawdb.SnapshotAccountPrefix
		keys    [][]byte
		vals    [][]byte
		last    []byte
		iter    = dl.diskdb.NewIterator(prefix, origin)
		aborted bool
	)
	for iter.Next() {
		key := iter.Key()
		if len(key) != len(prefix)+common.HashLength {
			continue // random state with same prefix
		}
		if len(keys) == max {
			// There exists more data in the snapshot, but we've reached the
			// max elems per iteration
			aborted = true
			break
		}
		keys = append(keys, common.CopyBytes(key[len(prefix):]))
		if v, err := FullAccountRLP(iter.Value()); err != nil {
			// Log the full key, so we can "db get" the key and check what it was
			log.Error("Malformed account data in snapshot", "key", hexutil.Encode(key), "error", err)
			iter.Release()
			return nil, nil, last, false, err
		} else {
			vals = append(vals, v)
		}
	}
	iter.Release()
	if err := iter.Error(); err != nil {
		return nil, nil, last, false, err
	}
	if len(keys) > 0 {
		last = keys[len(keys)-1]
	}
	// Generate the Merkle proofs for the first and last element
	if origin == nil {
		origin = common.Hash{}.Bytes()
	}
	proof := rawdb.NewMemoryDatabase()

	if err := tr.Prove(origin, 0, proof); err != nil {
		log.Debug("Failed to prove account range", "origin", hexutil.Encode(origin), "err", err)
		return nil, nil, last, false, err
	}
	if last != nil {
		if err := tr.Prove(last, 0, proof); err != nil {
			log.Debug("Failed to prove account range", "last", hexutil.Encode(last), "err", err)
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
	if cont && !aborted {
		return nil, nil, last, false, errors.New("data loss in the state range")
	}
	return keys, vals, last, !cont, nil
}

func (dl *diskLayer) generateStorageRange(root, accountHash common.Hash, origin []byte, max int, stats *generatorStats, ctrl *genController) (bool, []byte, error) {
	// Configure logger
	var logCtx = []interface{}{"kind", "storage"}
	if len(origin) > 0 {
		logCtx = append(logCtx, "origin", hexutil.Encode(origin))
	}
	logger := log.New(logCtx...)
	// Use range prover to check the validity of the flat state in the range
	result, err := dl.proveStorageRange(root, accountHash, origin, max)
	if err != nil {
		stats.Log("State snapshotting paused", root, dl.genMarker)
		return false, nil, errors.New("trie is missing")
	}
	if result.staleKeys == nil { // If 'nil', then the check was ok
		ctrl.flushIfNeeded(append(accountHash[:], result.last...))
		// Update meters
		logger.Trace("Storage range ok", "items", result.count, "last", hexutil.Encode(result.last))
		snapSuccessfulRangeProofMeter.Mark(1)
		snapRecoveredStorageMeter.Mark(int64(result.count))
		stats.slots += uint64(result.count)
		// We don't know the exact size, but it's somewhere between 1 and 32
		stats.storage += common.StorageSize(1 + 2*common.HashLength + result.count*20)
		return !result.continued, result.last, nil
	}
	// Put the stale key/value pairs into a map for later use
	var staleKeyValues = make(map[string][]byte)
	for i, k := range result.staleKeys {
		staleKeyValues[string(k)] = result.staleValues[i]
	}
	result.staleKeys, result.staleValues = nil, nil
	snapFailedRangeProofMeter.Mark(1)
	logger.Debug("Detected stale state range", "last", hexutil.Encode(result.last), "error", err)
	// The verifcation failed, the flat state in this range does not match the trie data.
	//  Use the trie to regenerate the snapshot storage data.
	tr := result.tr
	if tr == nil {
		if tr, err = trie.New(root, dl.triedb); err != nil {
			return false, nil, err
		}
	}
	var (
		iter          = trie.NewIterator(tr.NodeIterator(origin))
		batch         = ctrl.batch
		lastWritten   []byte
		trieExhausted = true

		// counters
		count     = 0 // number of slots delivered by iterator
		created   = 0 // slots created from the trie
		updated   = 0 // slots updated from the trie
		deleted   = 0 // slots not in trie, but were in snapshot
		untouched = 0 // slots already correct
	)
	// We keep track of two separate 'last's here:
	// - 'last': represents the last key that was wiped from the snapshot.
	//      We cannot go beyond that, since we haven't wiped those keys.
	// - 'lastWritten: represents the last key we regenerated from the trie.
	//      We return this to the caller, so the iteration can be continued from
	//      that point (and marker properly set).
	for iter.Next() {
		// The storage trie can be very large, and overflow the storageCheckRange.
		// Therefore, we need to check and limit the iteration here
		if count == max {
			trieExhausted = false
			break
		}
		count++
		key := iter.Key
		if result.last != nil && bytes.Compare(key, result.last) > 0 {
			logger.Debug("Regenerated state range", "last", hexutil.Encode(result.last))
			return false, lastWritten, nil // Apparently the trie is not exhausted
		}
		lastWritten = common.CopyBytes(iter.Key)
		// Do we _need_ to write it?
		strKey := string(key)
		if staleVal, ok := staleKeyValues[strKey]; ok {
			// We can simply overwrite it (no need to delete)
			delete(staleKeyValues, strKey)
			if bytes.Equal(staleVal, iter.Value) {
				// Optimal case, the key was fine. No need to even overwrite
				untouched++
				continue
			}
			updated++
		} else {
			created++
		}
		rawdb.WriteStorageSnapshot(batch, accountHash, common.BytesToHash(key), iter.Value)
	}
	// Add the deletions to the batch aswell
	for k, _ := range staleKeyValues {
		batch.Delete([]byte(k))
		deleted++
	}
	if err := ctrl.flushIfNeeded(append(accountHash[:], lastWritten...)); err != nil {
		return false, nil, err
	}
	if iter.Err != nil {
		logger.Error("Iterator error during storage regeneration", "error", err)
		return false, nil, iter.Err
	}
	snapWipedStorageMeter.Mark(int64(deleted))
	snapGeneratedStorageMeter.Mark(int64(count))
	logger.Debug("Regenerated snapshot storage range", "items", count,
		"created", created, "updated", updated, "deleted", deleted, "untouched", untouched,
		"last", hexutil.Encode(result.last))
	return trieExhausted, lastWritten, nil
}

// genAccountRange generates the state segment with particular prefix. Generation can
// either verify the correctness of existing state through rangeproof and skip
// generation, or iterate trie to regenerate state on demand.
func (dl *diskLayer) genAccountRange(root common.Hash, origin []byte, max int, stats *generatorStats, ctrl *genController, onAccount func(key, val, old []byte, regen bool) error) (bool, []byte, error) {
	var kind = "account"
	var logCtx = []interface{}{"kind", kind}
	if len(origin) > 0 {
		logCtx = append(logCtx, "origin", hexutil.Encode(origin))
	}
	logger := log.New(logCtx...)

	tr, err := trie.New(root, dl.triedb)
	if err != nil {
		stats.Log("Trie missing, state snapshotting paused", root, dl.genMarker)
		return false, nil, errors.New("trie is missing")
	}
	// Use range prover to check the validity of the flat state in the range
	keys, vals, last, exhausted, err := dl.proveAccountRange(root, tr, origin, max)
	if err == nil {
		snapSuccessfulRangeProofMeter.Mark(1)
		logger.Trace("Proved state range", "last", hexutil.Encode(last))

		// The verification passed, process each state with the given
		// callback function. If this state represents a contract, the
		// corresponding storage check will be performed in the callback
		for i := 0; i < len(keys); i++ {
			if err := onAccount(keys[i], vals[i], nil, false); err != nil {
				return false, nil, err
			}
		}
		logger.Trace("Recovered state range", "last", hexutil.Encode(last))
		return exhausted, last, nil
	}
	// Verifcation failed, the snapshot account data in this range does not match the trie.
	snapFailedRangeProofMeter.Mark(1)
	logger.Debug("Detected outdated state range", "last", hexutil.Encode(last), "error", err)
	// Now we use the trie to generate the snapshot account data.
	var (
		iter          = trie.NewIterator(tr.NodeIterator(origin))
		lastWritten   []byte
		trieExhausted = true

		updated = 0
		count   = 0
		deleted = 0
	)
	for iter.Next() {
		key := iter.Key
		var oldValue []byte
		for len(keys) > 0 {
			if diff := bytes.Compare(keys[0], key); diff < 0 {
				// the snapshot key must be deleted
				ctrl.batch.Delete(keys[0])
				keys = keys[1:]
				vals = vals[1:]
				deleted++
				continue
			} else if diff == 0 {
				// the snapshot key can be overwritten
				oldValue = vals[0]
				keys = keys[1:]
				vals = vals[1:]
				updated++
			}
			// else: not there yet, leave it for now
			break
		}
		if err := onAccount(iter.Key, iter.Value, oldValue, true); err != nil {
			return false, nil, err
		}
		lastWritten = common.CopyBytes(iter.Key)
		count++
	}
	// Now delete any remaining keys
	if len(keys) > 0 {
		for _, k := range keys {
			ctrl.batch.Delete(k)
		}
		deleted += len(keys)
	}
	if iter.Err != nil {
		return false, nil, iter.Err
	}
	logger.Debug("Regenerated snapshot account range", "written", count,
		"updated", updated, "deleted", deleted, "last", hexutil.Encode(lastWritten))
	return trieExhausted, lastWritten, nil
}

type fullAccount struct {
	Nonce    uint64
	Balance  *big.Int
	Root     common.Hash
	CodeHash []byte
}

// genController is used to ensure that batches are kept in sync with the marker,
// so the marker is updated properly.
type genController struct {
	batch   ethdb.Batch
	dl      *diskLayer
	stats   *generatorStats
	logged  time.Time
	abortCh chan *generatorStats
}

// checkAbort checks if abort has been requested. If so, it flushes the batch
// and returns an error. If abort has not been requested, it flushes the batch if
// it's deemed to be needed
func (ctrl *genController) checkAbort(currentLocation []byte) error {
	select {
	case abort := <-ctrl.dl.genAbort:
		ctrl.abortCh = abort
	default:
		return ctrl.flushIfNeeded(currentLocation)
	}
	ctrl.stats.Log("Aborting state snapshot generation", ctrl.dl.root, currentLocation)
	if err := ctrl.doFlush(currentLocation); err != nil {
		return err
	}
	return errors.New("aborted")
}

func (ctrl *genController) abort() {
	if ctrl.abortCh == nil { // aborted due to some internal error, wait the signal
		ctrl.abortCh = <-ctrl.dl.genAbort
	}
	ctrl.abortCh <- ctrl.stats
	return
}

// doFlush writes the batch to db
func (ctrl *genController) doFlush(currentLocation []byte) error {
	var (
		batch = ctrl.batch
		stats = ctrl.stats
		dl    = ctrl.dl
	)
	// Flush out the batch anyway no matter it's empty or not.
	// It's possible that all the states are recovered and the
	// generation indeed makes progress.
	journalProgress(batch, currentLocation, stats)

	if err := batch.Write(); err != nil {
		return err
	}
	ctrl.batch.Reset()

	dl.lock.Lock()
	dl.genMarker = currentLocation
	dl.lock.Unlock()
	return nil
}

// flushIfNeeded write the batch to db if needed
func (ctrl *genController) flushIfNeeded(currentLocation []byte) error {
	var (
		batch = ctrl.batch
		stats = ctrl.stats
		dl    = ctrl.dl
	)
	if batch.ValueSize() > ethdb.IdealBatchSize {
		ctrl.doFlush(currentLocation)
	}
	if time.Since(ctrl.logged) > 8*time.Second {
		stats.Log("Generating state snapshot", dl.root, currentLocation)
		ctrl.logged = time.Now()
	}
	return nil
}

// generate is a background thread that iterates over the state and storage tries,
// constructing the state snapshot. All the arguments are purely for statistics
// gathering and logging, since the method surfs the blocks as they arrive, often
// being restarted.
func (dl *diskLayer) generate(stats *generatorStats) {
	var (
		accMarker    []byte
		accountRange = accountCheckRange
	)
	if len(dl.genMarker) > 0 { // []byte{} is the start, use nil for that
		// Always reset the initial account range as 1
		// whenever recover from the interruption.
		accMarker, accountRange = dl.genMarker[:common.HashLength], 1
	}
	var (
		accOrigin = common.CopyBytes(accMarker)
		ctrl      = &genController{
			batch:  dl.diskdb.NewBatch(),
			dl:     dl,
			stats:  stats,
			logged: time.Now(),
		}
	)
	stats.Log("Resuming state snapshot generation", dl.root, dl.genMarker)

	onAccount := func(key, val, oldSnapData []byte, regen bool) error {
		// Retrieve the current account and flatten it into the internal format
		accountHash := common.BytesToHash(key)
		var acc fullAccount
		if err := rlp.DecodeBytes(val, &acc); err != nil {
			log.Crit("Invalid account encountered during snapshot creation", "err", err)
		}
		// If the account is not yet in-progress, write it out
		if accMarker == nil || !bytes.Equal(accountHash[:], accMarker) {
			stats.accounts++
			if !regen {
				// Technically not always correct to use len(val) here, but approximately.
				// It saves us some RLP decoding + encoding just to figure out the size.
				stats.storage += common.StorageSize(1 + common.HashLength + len(val))
				snapRecoveredAccountMeter.Mark(1)
			} else {
				data := SlimAccountRLP(acc.Nonce, acc.Balance, acc.Root, acc.CodeHash)
				if !bytes.Equal(data, oldSnapData) {
					rawdb.WriteAccountSnapshot(ctrl.batch, accountHash, data)
				}
				//else: No need to write - the existing snap data is correct

				// Update meters
				stats.storage += common.StorageSize(1 + common.HashLength + len(data))
				snapGeneratedAccountMeter.Mark(1)
			}
		}
		// If we've exceeded our batch allowance or termination was requested, flush to disk
		if err := ctrl.checkAbort(accountHash[:]); err != nil {
			return err
		}
		// The account is in progress
		// If the iterated account is the contract, create a further loop to
		// verify or regenerate the contract storage.
		if acc.Root != emptyRoot {
			var storeMarker []byte
			if accMarker != nil && bytes.Equal(accountHash[:], accMarker) && len(dl.genMarker) > common.HashLength {
				storeMarker = dl.genMarker[common.HashLength:]
			}
			var storeOrigin = common.CopyBytes(storeMarker)
			for {
				exhausted, last, err := dl.generateStorageRange(acc.Root, accountHash, storeOrigin, storageCheckRange, stats, ctrl)
				if err != nil {
					return err
				}
				if err := ctrl.checkAbort(append(accountHash[:], last...)); err != nil {
					return err
				}
				if exhausted {
					return nil
				}
				if storeOrigin = increaseKey(last); storeOrigin == nil {
					return nil // special case, the last is 0xffffffff...fff
				}
			}
		} else {
			// If the root is empty, we still need to ensure that any previous snapshot
			// storage values are cleared
			// TODO: investigate if this can be avoided, this will be very costly since it
			// affects every single EOA account
			//  - Perhaps we can avoid if where codeHash is emptyCode
			prefix := append(rawdb.SnapshotStoragePrefix, accountHash.Bytes()...)
			keyLen := len(rawdb.SnapshotStoragePrefix) + 2*common.HashLength
			if err := wipeKeyRange(dl.diskdb, "storage", prefix, nil, nil, keyLen, snapWipedStorageMeter, true); err != nil {
				return err
			}
		}
		// Some account processed, unmark the marker
		accMarker = nil
		return nil
	}
	for {
		exhausted, last, err := dl.genAccountRange(dl.root, accOrigin, accountRange, stats, ctrl, onAccount)
		// The procedure it aborted, either by external signal or internal error
		if err != nil {
			ctrl.abort()
			return
		}
		// Abort the procedure if the entire snapshot is generated
		if exhausted {
			break
		}
		if accOrigin = increaseKey(last); accOrigin == nil {
			break // special case, the last is 0xffffffff...fff
		}
		accountRange = accountCheckRange
	}
	// Snapshot fully generated, set the marker to nil.
	// Note even there is nothing to commit, persist the
	// generator anyway to mark the snapshot is complete.
	if err := ctrl.doFlush(nil); err != nil {
		log.Error("Failed to flush batch", "error", err)
		ctrl.abort()
		return
	}

	log.Info("Generated state snapshot", "accounts", stats.accounts, "slots", stats.slots,
		"storage", stats.storage, "elapsed", common.PrettyDuration(time.Since(stats.start)))

	dl.lock.Lock()
	dl.genMarker = nil
	close(dl.genPending)
	dl.lock.Unlock()

	// Someone will be looking for us, wait it out
	ctrl.stats = nil
	ctrl.abort()
}

// increaseKey increase the input key by one bit. Return nil if the entire
// addition operation overflows,
func increaseKey(key []byte) []byte {
	for i := len(key) - 1; i >= 0; i-- {
		key[i]++
		if key[i] != 0x0 {
			return key
		}
	}
	return nil
}
