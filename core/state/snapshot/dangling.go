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
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

// CheckDanglingStorage iterates the snap storage data, and verifies that all
// storage also has corresponding account data.
func CheckDanglingStorage(chaindb ethdb.KeyValueStore) error {
	if err := checkDanglingDiskStorage(chaindb); err != nil {
		return err
	}
	return checkDanglingMemStorage(chaindb)
}

// checkDanglingDiskStorage checks if there is any 'dangling' storage data in the
// disk-backed snapshot layer.
func checkDanglingDiskStorage(chaindb ethdb.KeyValueStore) error {
	var (
		lastReport = time.Now()
		start      = time.Now()
		lastKey    []byte
		it         = rawdb.NewKeyLengthIterator(chaindb.NewIterator(rawdb.SnapshotStoragePrefix, nil), 1+2*common.HashLength)
	)
	log.Info("Checking dangling snapshot disk storage")

	defer it.Release()
	for it.Next() {
		k := it.Key()
		accKey := k[1:33]
		if bytes.Equal(accKey, lastKey) {
			// No need to look up for every slot
			continue
		}
		lastKey = common.CopyBytes(accKey)
		if time.Since(lastReport) > time.Second*8 {
			log.Info("Iterating snap storage", "at", fmt.Sprintf("%#x", accKey), "elapsed", common.PrettyDuration(time.Since(start)))
			lastReport = time.Now()
		}
		if data := rawdb.ReadAccountSnapshot(chaindb, common.BytesToHash(accKey)); len(data) == 0 {
			log.Warn("Dangling storage - missing account", "account", fmt.Sprintf("%#x", accKey), "storagekey", fmt.Sprintf("%#x", k))
			return fmt.Errorf("dangling snapshot storage account %#x", accKey)
		}
	}
	log.Info("Verified the snapshot disk storage", "time", common.PrettyDuration(time.Since(start)), "err", it.Error())
	return nil
}

// checkDanglingMemStorage checks if there is any 'dangling' storage in the journalled
// snapshot difflayers.
func checkDanglingMemStorage(db ethdb.KeyValueStore) error {
	var (
		start   = time.Now()
		journal = rawdb.ReadSnapshotJournal(db)
	)
	if len(journal) == 0 {
		log.Warn("Loaded snapshot journal", "diffs", "missing")
		return nil
	}
	r := rlp.NewStream(bytes.NewReader(journal), 0)
	// Firstly, resolve the first element as the journal version
	version, err := r.Uint()
	if err != nil {
		log.Warn("Failed to resolve the journal version", "error", err)
		return nil
	}
	if version != journalVersion {
		log.Warn("Discarded the snapshot journal with wrong version", "required", journalVersion, "got", version)
		return nil
	}
	// Secondly, resolve the disk layer root, ensure it's continuous
	// with disk layer. Note now we can ensure it's the snapshot journal
	// correct version, so we expect everything can be resolved properly.
	var root common.Hash
	if err := r.Decode(&root); err != nil {
		return errors.New("missing disk layer root")
	}
	// The diff journal is not matched with disk, discard them.
	// It can happen that Geth crashes without persisting the latest
	// diff journal.
	// Load all the snapshot diffs from the journal
	if err := checkDanglingJournalStorage(r); err != nil {
		return err
	}
	log.Info("Verified the snapshot journalled storage", "time", common.PrettyDuration(time.Since(start)))
	return nil
}

// loadDiffLayer reads the next sections of a snapshot journal, reconstructing a new
// diff and verifying that it can be linked to the requested parent.
func checkDanglingJournalStorage(r *rlp.Stream) error {
	for {
		// Read the next diff journal entry
		var root common.Hash
		if err := r.Decode(&root); err != nil {
			// The first read may fail with EOF, marking the end of the journal
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("load diff root: %v", err)
		}
		var destructs []journalDestruct
		if err := r.Decode(&destructs); err != nil {
			return fmt.Errorf("load diff destructs: %v", err)
		}
		var accounts []journalAccount
		if err := r.Decode(&accounts); err != nil {
			return fmt.Errorf("load diff accounts: %v", err)
		}
		accountData := make(map[common.Hash][]byte)
		for _, entry := range accounts {
			if len(entry.Blob) > 0 { // RLP loses nil-ness, but `[]byte{}` is not a valid item, so reinterpret that
				accountData[entry.Hash] = entry.Blob
			} else {
				accountData[entry.Hash] = nil
			}
		}
		var storage []journalStorage
		if err := r.Decode(&storage); err != nil {
			return fmt.Errorf("load diff storage: %v", err)
		}
		for _, entry := range storage {
			if _, ok := accountData[entry.Hash]; !ok {
				log.Error("Dangling storage - missing account", "account", fmt.Sprintf("%#x", entry.Hash), "root", root)
				return fmt.Errorf("dangling journal snapshot storage account %#x", entry.Hash)
			}
		}
	}
}
