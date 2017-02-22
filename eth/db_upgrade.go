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
	"encoding/binary"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

var useSequentialKeys = []byte("dbUpgrade_20160530sequentialKeys")

// upgradeSequentialKeys checks the chain database version and
// starts a background process to make upgrades if necessary.
// Returns a stop function that blocks until the process has
// been safely stopped.
func upgradeSequentialKeys(db ethdb.Database) (stopFn func()) {
	data, _ := db.Get(useSequentialKeys)
	if len(data) > 0 && data[0] == 42 {
		return nil // already converted
	}

	if data, _ := db.Get([]byte("LastHeader")); len(data) == 0 {
		db.Put(useSequentialKeys, []byte{42})
		return nil // empty database, nothing to do
	}

	log.Info(fmt.Sprintf("Upgrading chain database to use sequential keys"))

	stopChn := make(chan struct{})
	stoppedChn := make(chan struct{})

	go func() {
		stopFn := func() bool {
			select {
			case <-time.After(time.Microsecond * 100): // make sure other processes don't get starved
			case <-stopChn:
				return true
			}
			return false
		}

		err, stopped := upgradeSequentialCanonicalNumbers(db, stopFn)
		if err == nil && !stopped {
			err, stopped = upgradeSequentialBlocks(db, stopFn)
		}
		if err == nil && !stopped {
			err, stopped = upgradeSequentialOrphanedReceipts(db, stopFn)
		}
		if err == nil && !stopped {
			log.Info(fmt.Sprintf("Database conversion successful"))
			db.Put(useSequentialKeys, []byte{42})
		}
		if err != nil {
			log.Error(fmt.Sprintf("Database conversion failed: %v", err))
		}
		close(stoppedChn)
	}()

	return func() {
		close(stopChn)
		<-stoppedChn
	}
}

// upgradeSequentialCanonicalNumbers reads all old format canonical numbers from
// the database, writes them in new format and deletes the old ones if successful.
func upgradeSequentialCanonicalNumbers(db ethdb.Database, stopFn func() bool) (error, bool) {
	prefix := []byte("block-num-")
	it := db.(*ethdb.LDBDatabase).NewIterator()
	defer func() {
		it.Release()
	}()
	it.Seek(prefix)
	cnt := 0
	for bytes.HasPrefix(it.Key(), prefix) {
		keyPtr := it.Key()
		if len(keyPtr) < 20 {
			cnt++
			if cnt%100000 == 0 {
				it.Release()
				it = db.(*ethdb.LDBDatabase).NewIterator()
				it.Seek(keyPtr)
				log.Info(fmt.Sprintf("converting %d canonical numbers...", cnt))
			}
			number := big.NewInt(0).SetBytes(keyPtr[10:]).Uint64()
			newKey := []byte("h12345678n")
			binary.BigEndian.PutUint64(newKey[1:9], number)
			if err := db.Put(newKey, it.Value()); err != nil {
				return err, false
			}
			if err := db.Delete(keyPtr); err != nil {
				return err, false
			}
		}

		if stopFn() {
			return nil, true
		}
		it.Next()
	}
	if cnt > 0 {
		log.Info(fmt.Sprintf("converted %d canonical numbers...", cnt))
	}
	return nil, false
}

// upgradeSequentialBlocks reads all old format block headers, bodies, TDs and block
// receipts from the database, writes them in new format and deletes the old ones
// if successful.
func upgradeSequentialBlocks(db ethdb.Database, stopFn func() bool) (error, bool) {
	prefix := []byte("block-")
	it := db.(*ethdb.LDBDatabase).NewIterator()
	defer func() {
		it.Release()
	}()
	it.Seek(prefix)
	cnt := 0
	for bytes.HasPrefix(it.Key(), prefix) {
		keyPtr := it.Key()
		if len(keyPtr) >= 38 {
			cnt++
			if cnt%10000 == 0 {
				it.Release()
				it = db.(*ethdb.LDBDatabase).NewIterator()
				it.Seek(keyPtr)
				log.Info(fmt.Sprintf("converting %d blocks...", cnt))
			}
			// convert header, body, td and block receipts
			var keyPrefix [38]byte
			copy(keyPrefix[:], keyPtr[0:38])
			hash := keyPrefix[6:38]
			if err := upgradeSequentialBlockData(db, hash); err != nil {
				return err, false
			}
			// delete old db entries belonging to this hash
			for bytes.HasPrefix(it.Key(), keyPrefix[:]) {
				if err := db.Delete(it.Key()); err != nil {
					return err, false
				}
				it.Next()
			}
			if err := db.Delete(append([]byte("receipts-block-"), hash...)); err != nil {
				return err, false
			}
		} else {
			it.Next()
		}

		if stopFn() {
			return nil, true
		}
	}
	if cnt > 0 {
		log.Info(fmt.Sprintf("converted %d blocks...", cnt))
	}
	return nil, false
}

// upgradeSequentialOrphanedReceipts removes any old format block receipts from the
// database that did not have a corresponding block
func upgradeSequentialOrphanedReceipts(db ethdb.Database, stopFn func() bool) (error, bool) {
	prefix := []byte("receipts-block-")
	it := db.(*ethdb.LDBDatabase).NewIterator()
	defer it.Release()
	it.Seek(prefix)
	cnt := 0
	for bytes.HasPrefix(it.Key(), prefix) {
		// phase 2 already converted receipts belonging to existing
		// blocks, just remove if there's anything left
		cnt++
		if err := db.Delete(it.Key()); err != nil {
			return err, false
		}

		if stopFn() {
			return nil, true
		}
		it.Next()
	}
	if cnt > 0 {
		log.Info(fmt.Sprintf("removed %d orphaned block receipts...", cnt))
	}
	return nil, false
}

// upgradeSequentialBlockData upgrades the header, body, td and block receipts
// database entries belonging to a single hash (doesn't delete old data).
func upgradeSequentialBlockData(db ethdb.Database, hash []byte) error {
	// get old chain data and block number
	headerRLP, _ := db.Get(append(append([]byte("block-"), hash...), []byte("-header")...))
	if len(headerRLP) == 0 {
		return nil
	}
	header := new(types.Header)
	if err := rlp.Decode(bytes.NewReader(headerRLP), header); err != nil {
		return err
	}
	number := header.Number.Uint64()
	bodyRLP, _ := db.Get(append(append([]byte("block-"), hash...), []byte("-body")...))
	tdRLP, _ := db.Get(append(append([]byte("block-"), hash...), []byte("-td")...))
	receiptsRLP, _ := db.Get(append([]byte("receipts-block-"), hash...))
	// store new hash -> number association
	encNum := make([]byte, 8)
	binary.BigEndian.PutUint64(encNum, number)
	if err := db.Put(append([]byte("H"), hash...), encNum); err != nil {
		return err
	}
	// store new chain data
	if err := db.Put(append(append([]byte("h"), encNum...), hash...), headerRLP); err != nil {
		return err
	}
	if len(tdRLP) != 0 {
		if err := db.Put(append(append(append([]byte("h"), encNum...), hash...), []byte("t")...), tdRLP); err != nil {
			return err
		}
	}
	if len(bodyRLP) != 0 {
		if err := db.Put(append(append([]byte("b"), encNum...), hash...), bodyRLP); err != nil {
			return err
		}
	}
	if len(receiptsRLP) != 0 {
		if err := db.Put(append(append([]byte("r"), encNum...), hash...), receiptsRLP); err != nil {
			return err
		}
	}
	return nil
}

// upgradeChainDatabase ensures that the chain database stores block split into
// separate header and body entries.
func upgradeChainDatabase(db ethdb.Database) error {
	// Short circuit if the head block is stored already as separate header and body
	data, err := db.Get([]byte("LastBlock"))
	if err != nil {
		return nil
	}
	head := common.BytesToHash(data)

	if block := core.GetBlockByHashOld(db, head); block == nil {
		return nil
	}
	// At least some of the database is still the old format, upgrade (skip the head block!)
	log.Info(fmt.Sprint("Old database detected, upgrading..."))

	if db, ok := db.(*ethdb.LDBDatabase); ok {
		blockPrefix := []byte("block-hash-")
		for it := db.NewIterator(); it.Next(); {
			// Skip anything other than a combined block
			if !bytes.HasPrefix(it.Key(), blockPrefix) {
				continue
			}
			// Skip the head block (merge last to signal upgrade completion)
			if bytes.HasSuffix(it.Key(), head.Bytes()) {
				continue
			}
			// Load the block, split and serialize (order!)
			block := core.GetBlockByHashOld(db, common.BytesToHash(bytes.TrimPrefix(it.Key(), blockPrefix)))

			if err := core.WriteTd(db, block.Hash(), block.NumberU64(), block.DeprecatedTd()); err != nil {
				return err
			}
			if err := core.WriteBody(db, block.Hash(), block.NumberU64(), block.Body()); err != nil {
				return err
			}
			if err := core.WriteHeader(db, block.Header()); err != nil {
				return err
			}
			if err := db.Delete(it.Key()); err != nil {
				return err
			}
		}
		// Lastly, upgrade the head block, disabling the upgrade mechanism
		current := core.GetBlockByHashOld(db, head)

		if err := core.WriteTd(db, current.Hash(), current.NumberU64(), current.DeprecatedTd()); err != nil {
			return err
		}
		if err := core.WriteBody(db, current.Hash(), current.NumberU64(), current.Body()); err != nil {
			return err
		}
		if err := core.WriteHeader(db, current.Header()); err != nil {
			return err
		}
	}
	return nil
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
	log.Info(fmt.Sprint("upgrading db log bloom bins"))
	for i := uint64(0); i <= latestBlock.NumberU64(); i++ {
		hash := core.GetCanonicalHash(db, i)
		if (hash == common.Hash{}) {
			return fmt.Errorf("chain db corrupted. Could not find block %d.", i)
		}
		core.WriteMipmapBloom(db, i, core.GetBlockReceipts(db, hash, i))
	}
	log.Info(fmt.Sprint("upgrade completed in", time.Since(tstart)))
	return nil
}
