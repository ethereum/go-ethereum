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
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

// journalAccount is an account entry in a diffLayer's disk journal.
type journalAccount struct {
	Hash common.Hash
	Blob []byte
}

// journalStorage is an account's storage map in a diffLayer's disk journal.
type journalStorage struct {
	Hash common.Hash
	Keys []common.Hash
	Vals [][]byte
}

// loadDiffLayer reads the next sections of a snapshot journal, reconstructing a new
// diff and verifying that it can be linked to the requested parent.
func loadDiffLayer(parent snapshot, r *rlp.Stream) (snapshot, error) {
	// Read the next diff journal entry
	var (
		number uint64
		root   common.Hash
	)
	if err := r.Decode(&number); err != nil {
		// The first read may fail with EOF, marking the end of the journal
		if err == io.EOF {
			return parent, nil
		}
		return nil, fmt.Errorf("load diff number: %v", err)
	}
	if err := r.Decode(&root); err != nil {
		return nil, fmt.Errorf("load diff root: %v", err)
	}
	var accounts []journalAccount
	if err := r.Decode(&accounts); err != nil {
		return nil, fmt.Errorf("load diff accounts: %v", err)
	}
	accountData := make(map[common.Hash][]byte)
	for _, entry := range accounts {
		accountData[entry.Hash] = entry.Blob
	}
	var storage []journalStorage
	if err := r.Decode(&storage); err != nil {
		return nil, fmt.Errorf("load diff storage: %v", err)
	}
	storageData := make(map[common.Hash]map[common.Hash][]byte)
	for _, entry := range storage {
		slots := make(map[common.Hash][]byte)
		for i, key := range entry.Keys {
			slots[key] = entry.Vals[i]
		}
		storageData[entry.Hash] = slots
	}
	// Validate the block number to avoid state corruption
	if parent, ok := parent.(*diffLayer); ok {
		if number != parent.number+1 {
			return nil, fmt.Errorf("snapshot chain broken: block #%d after #%d", number, parent.number)
		}
	}
	return loadDiffLayer(newDiffLayer(parent, number, root, accountData, storageData), r)
}

// journal is the internal version of Journal that also returns the journal file
// so subsequent layers know where to write to.
func (dl *diffLayer) journal() (io.WriteCloser, error) {
	// If we've reached the bottom, open the journal
	var writer io.WriteCloser
	if parent, ok := dl.parent.(*diskLayer); ok {
		file, err := os.Create(parent.journal)
		if err != nil {
			return nil, err
		}
		writer = file
	}
	// If we haven't reached the bottom yet, journal the parent first
	if writer == nil {
		file, err := dl.parent.(*diffLayer).journal()
		if err != nil {
			return nil, err
		}
		writer = file
	}
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	if dl.stale {
		writer.Close()
		return nil, ErrSnapshotStale
	}
	buf := bufio.NewWriter(writer)
	// Everything below was journalled, persist this layer too
	if err := rlp.Encode(buf, dl.number); err != nil {
		buf.Flush()
		writer.Close()
		return nil, err
	}
	if err := rlp.Encode(buf, dl.root); err != nil {
		buf.Flush()
		writer.Close()
		return nil, err
	}
	accounts := make([]journalAccount, 0, len(dl.accountData))
	for hash, blob := range dl.accountData {
		accounts = append(accounts, journalAccount{Hash: hash, Blob: blob})
	}
	if err := rlp.Encode(buf, accounts); err != nil {
		buf.Flush()
		writer.Close()
		return nil, err
	}
	storage := make([]journalStorage, 0, len(dl.storageData))
	for hash, slots := range dl.storageData {
		keys := make([]common.Hash, 0, len(slots))
		vals := make([][]byte, 0, len(slots))
		for key, val := range slots {
			keys = append(keys, key)
			vals = append(vals, val)
		}
		storage = append(storage, journalStorage{Hash: hash, Keys: keys, Vals: vals})
	}
	if err := rlp.Encode(buf, storage); err != nil {
		buf.Flush()
		writer.Close()
		return nil, err
	}
	buf.Flush()
	return writer, nil
}
