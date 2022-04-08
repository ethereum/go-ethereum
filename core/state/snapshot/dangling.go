// Copyright 2022 The go-ethereum Authors
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
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

// danglingRange describes the range for detecting dangling storages.
type danglingRange struct {
	db    ethdb.KeyValueStore // The database stores the snapshot data
	start []byte              // The start of the key range
	limit []byte              // The last of the key range

	result   []common.Hash // The list of account hashes which have the dangling storages
	duration time.Duration // Total time spent on the iteration
}

// newDanglingRange initializes a dangling storage scanner and detects all the
// dangling accounts out.
func newDanglingRange(db ethdb.KeyValueStore, start, limit []byte, report bool) *danglingRange {
	r := &danglingRange{
		db:    db,
		start: start,
		limit: limit,
	}
	r.result, r.duration = r.detect(report)

	// Update metrics
	snapDanglingStoragesCounter.Inc(int64(len(r.result)))
	snapDanglingStoragesTimer.Update(r.duration)

	if len(r.result) > 0 {
		log.Warn("Detected dangling storages", "number", len(r.result), "start", hexutil.Encode(start), "limit", hexutil.Encode(limit), "elapsed", common.PrettyDuration(r.duration))
	} else {
		logger := log.Debug
		if report {
			logger = log.Info
		}
		logger("Verified snapshot storages", "start", hexutil.Encode(start), "limit", hexutil.Encode(limit), "elapsed", common.PrettyDuration(r.duration))
	}
	return r
}

// detect iterates the storage snapshot in the specified key range and
// returns a list of account hash of the dangling storages. Note both
// start and limit are included for iteration.
func (r *danglingRange) detect(report bool) ([]common.Hash, time.Duration) {
	var (
		checked    []byte
		result     []common.Hash
		start      = time.Now()
		lastReport = time.Now()
	)
	iter := rawdb.NewKeyLengthIterator(r.db.NewIterator(rawdb.SnapshotStoragePrefix, r.start), len(rawdb.SnapshotStoragePrefix)+2*common.HashLength)
	defer iter.Release()

	for iter.Next() {
		account := iter.Key()[len(rawdb.SnapshotStoragePrefix) : len(rawdb.SnapshotStoragePrefix)+common.HashLength]
		if r.limit != nil && bytes.Compare(account, r.limit) > 0 {
			break
		}
		// Skip unnecessary checks for checked storage.
		if bytes.Equal(account, checked) {
			continue
		}
		checked = common.CopyBytes(account)

		// Check the presence of the corresponding account.
		accountHash := common.BytesToHash(account)
		data := rawdb.ReadAccountSnapshot(r.db, accountHash)
		if len(data) != 0 {
			continue
		}
		result = append(result, accountHash)

		if time.Since(lastReport) > time.Second*8 && report {
			log.Info("Detecting dangling storage", "at", fmt.Sprintf("%#x", accountHash), "elapsed", common.PrettyDuration(time.Since(start)))
			lastReport = time.Now()
		}
	}
	return result, time.Since(start)
}

// cleanup wipes the dangling storages which fall within the range before the given key.
func (r *danglingRange) cleanup(limit []byte) error {
	var (
		err   error
		wiped int
	)
	for _, accountHash := range r.result {
		if limit != nil && bytes.Compare(accountHash.Bytes(), limit) >= 0 {
			break
		}
		prefix := append(rawdb.SnapshotStoragePrefix, accountHash.Bytes()...)
		keylen := len(rawdb.SnapshotStoragePrefix) + 2*common.HashLength
		if err = wipeKeyRange(r.db, "storage", prefix, nil, nil, keylen, snapWipedStorageMeter, false); err != nil {
			break
		}
		wiped += 1
	}
	r.result = r.result[wiped:]
	return err
}
