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
	"github.com/allegro/bigcache"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
)

// diskLayer is a low level persistent snapshot built on top of a key-value store.
type diskLayer struct {
	journal string              // Path of the snapshot journal to use on shutdown
	db      ethdb.KeyValueStore // Key-value store containing the base snapshot
	cache   *bigcache.BigCache  // Cache to avoid hitting the disk for direct access

	number uint64      // Block number of the base snapshot
	root   common.Hash // Root hash of the base snapshot
}

// Info returns the block number and root hash for which this snapshot was made.
func (dl *diskLayer) Info() (uint64, common.Hash) {
	return dl.number, dl.root
}

// Account directly retrieves the account associated with a particular hash in
// the snapshot slim data format.
func (dl *diskLayer) Account(hash common.Hash) *Account {
	data := dl.AccountRLP(hash)
	if len(data) == 0 { // can be both nil and []byte{}
		return nil
	}
	account := new(Account)
	if err := rlp.DecodeBytes(data, account); err != nil {
		panic(err)
	}
	return account
}

// AccountRLP directly retrieves the account RLP associated with a particular
// hash in the snapshot slim data format.
func (dl *diskLayer) AccountRLP(hash common.Hash) []byte {
	key := string(hash[:])

	// Try to retrieve the account from the memory cache
	if blob, err := dl.cache.Get(key); err == nil {
		snapshotCleanHitMeter.Mark(1)
		snapshotCleanReadMeter.Mark(int64(len(blob)))
		return blob
	}
	// Cache doesn't contain account, pull from disk and cache for later
	blob := rawdb.ReadAccountSnapshot(dl.db, hash)
	dl.cache.Set(key, blob)

	snapshotCleanMissMeter.Mark(1)
	snapshotCleanWriteMeter.Mark(int64(len(blob)))

	return blob
}

// Storage directly retrieves the storage data associated with a particular hash,
// within a particular account.
func (dl *diskLayer) Storage(accountHash, storageHash common.Hash) []byte {
	key := string(append(accountHash[:], storageHash[:]...))

	// Try to retrieve the storage slot from the memory cache
	if blob, err := dl.cache.Get(key); err == nil {
		snapshotCleanHitMeter.Mark(1)
		snapshotCleanReadMeter.Mark(int64(len(blob)))
		return blob
	}
	// Cache doesn't contain storage slot, pull from disk and cache for later
	blob := rawdb.ReadStorageSnapshot(dl.db, accountHash, storageHash)
	dl.cache.Set(key, blob)

	snapshotCleanMissMeter.Mark(1)
	snapshotCleanWriteMeter.Mark(int64(len(blob)))

	return blob
}

// Update creates a new layer on top of the existing snapshot diff tree with
// the specified data items. Note, the maps are retained by the method to avoid
// copying everything.
func (dl *diskLayer) Update(blockHash common.Hash, accounts map[common.Hash][]byte, storage map[common.Hash]map[common.Hash][]byte) *diffLayer {
	return newDiffLayer(dl, dl.number+1, blockHash, accounts, storage)
}

// Cap traverses downwards the diff tree until the number of allowed layers are
// crossed. All diffs beyond the permitted number are flattened downwards.
func (dl *diskLayer) Cap(layers int, memory uint64) (uint64, uint64) {
	return dl.number, dl.number
}

// Journal commits an entire diff hierarchy to disk into a single journal file.
func (dl *diskLayer) Journal() error {
	// There's no journalling a disk layer
	return nil
}
