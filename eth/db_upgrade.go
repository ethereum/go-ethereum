// Copyright 2016 The go-ethereum Authors
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

// Package eth implements the Ethereum protocol.
package eth

import (
	"bytes"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

var deduplicateData = []byte("dbUpgrade_20170714deduplicateData")

// upgradeDeduplicateData checks the chain database version and
// starts a background process to make upgrades if necessary.
// Returns a stop function that blocks until the process has
// been safely stopped.
func upgradeDeduplicateData(db ethdb.Database) func() error {
	// If the database is already converted or empty, bail out
	data, _ := db.Get(deduplicateData)
	if len(data) > 0 && data[0] == 42 {
		return nil
	}
	if data, _ := db.Get([]byte("LastHeader")); len(data) == 0 {
		db.Put(deduplicateData, []byte{42})
		return nil
	}
	// Start the deduplication upgrade on a new goroutine
	log.Warn("Upgrading database to use lookup entries")
	stop := make(chan chan error)

	go func() {
		// Create an iterator to read the entire database and covert old lookup entires
		it := db.(*ethdb.LDBDatabase).NewIterator()
		defer func() {
			if it != nil {
				it.Release()
			}
		}()

		var (
			converted uint64
			failed    error
		)
		for failed == nil && it.Next() {
			// Skip any entries that don't look like old transaction meta entires (<hash>0x01)
			key := it.Key()
			if len(key) != common.HashLength+1 || key[common.HashLength] != 0x01 {
				continue
			}
			// Skip any entries that don't contain metadata (name clash between <hash>0x01 and <some-prefix><hash>)
			var meta struct {
				BlockHash  common.Hash
				BlockIndex uint64
				Index      uint64
			}
			if err := rlp.DecodeBytes(it.Value(), &meta); err != nil {
				continue
			}
			// Skip any already upgraded entries (clash due to <hash> ending with 0x01 (old suffix))
			hash := key[:common.HashLength]

			if hash[0] == byte('l') {
				// Potential clash, the "old" `hash` must point to a live transaction.
				if tx, _, _, _ := core.GetTransaction(db, common.BytesToHash(hash)); tx == nil || !bytes.Equal(tx.Hash().Bytes(), hash) {
					continue
				}
			}
			// Convert the old metadata to a new lookup entry, delete duplicate data
			if failed = db.Put(append([]byte("l"), hash...), it.Value()); failed == nil { // Write the new looku entry
				if failed = db.Delete(hash); failed == nil { // Delete the duplicate transaction data
					if failed = db.Delete(append([]byte("receipts-"), hash...)); failed == nil { // Delete the duplicate receipt data
						if failed = db.Delete(key); failed != nil { // Delete the old transaction metadata
							break
						}
					}
				}
			}
			// Bump the conversion counter, and recreate the iterator occasionally to
			// avoid too high memory consumption.
			converted++
			if converted%100000 == 0 {
				it.Release()
				it = db.(*ethdb.LDBDatabase).NewIterator()
				it.Seek(key)

				log.Info("Deduplicating database entries", "deduped", converted)
			}
			// Check for termination, or continue after a bit of a timeout
			select {
			case errc := <-stop:
				errc <- nil
				return
			case <-time.After(time.Microsecond * 100):
			}
		}
		// Upgrade finished, mark a such and terminate
		if failed == nil {
			log.Info("Database deduplication successful", "deduped", converted)
			db.Put(deduplicateData, []byte{42})
		} else {
			log.Error("Database deduplication failed", "deduped", converted, "err", failed)
		}
		it.Release()
		it = nil

		errc := <-stop
		errc <- failed
	}()
	// Assembly the cancellation callback
	return func() error {
		errc := make(chan error)
		stop <- errc
		return <-errc
	}
}

func addMipmapBloomBins(db ethdb.Database) (err error) {
	const mipmapVersion uint = 2

	// check if the version is set. We ignore data for now since there's
	// only one version so we can easily ignore it for now
	var data []byte
	data, _ = db.Get([]byte("setting-mipmap-version"))
	if len(data) > 0 {
		var version uint
		if err := rlp.DecodeBytes(data, &version); err == nil && version == mipmapVersion {
			return nil
		}
	}

	defer func() {
		if err == nil {
			var val []byte
			val, err = rlp.EncodeToBytes(mipmapVersion)
			if err == nil {
				err = db.Put([]byte("setting-mipmap-version"), val)
			}
			return
		}
	}()
	latestHash := core.GetHeadBlockHash(db)
	latestBlock := core.GetBlock(db, latestHash, core.GetBlockNumber(db, latestHash))
	if latestBlock == nil { // clean database
		return
	}

	tstart := time.Now()
	log.Warn("Upgrading db log bloom bins")
	for i := uint64(0); i <= latestBlock.NumberU64(); i++ {
		hash := core.GetCanonicalHash(db, i)
		if (hash == common.Hash{}) {
			return fmt.Errorf("chain db corrupted. Could not find block %d.", i)
		}
		core.WriteMipmapBloom(db, i, core.GetBlockReceipts(db, hash, i))
	}
	log.Info("Bloom-bin upgrade completed", "elapsed", common.PrettyDuration(time.Since(tstart)))
	return nil
}
