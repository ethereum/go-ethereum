// Copyright 2018 The go-ethereum Authors
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

package rawdb

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethdb/leveldb"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/olekukonko/tablewriter"
)

// freezerdb is a database wrapper that enabled freezer data retrievals.
type freezerdb struct {
	ethdb.KeyValueStore
	ethdb.AncientStore
}

// Close implements io.Closer, closing both the fast key-value store as well as
// the slow ancient tables.
func (frdb *freezerdb) Close() error {
	var errs []error
	if err := frdb.AncientStore.Close(); err != nil {
		errs = append(errs, err)
	}
	if err := frdb.KeyValueStore.Close(); err != nil {
		errs = append(errs, err)
	}
	if len(errs) != 0 {
		return fmt.Errorf("%v", errs)
	}
	return nil
}

// nofreezedb is a database wrapper that disables freezer data retrievals.
type nofreezedb struct {
	ethdb.KeyValueStore
}

// HasAncient returns an error as we don't have a backing chain freezer.
func (db *nofreezedb) HasAncient(kind string, number uint64) (bool, error) {
	return false, errNotSupported
}

// Ancient returns an error as we don't have a backing chain freezer.
func (db *nofreezedb) Ancient(kind string, number uint64) ([]byte, error) {
	return nil, errNotSupported
}

// Ancients returns an error as we don't have a backing chain freezer.
func (db *nofreezedb) Ancients() (uint64, error) {
	return 0, errNotSupported
}

// AncientSize returns an error as we don't have a backing chain freezer.
func (db *nofreezedb) AncientSize(kind string) (uint64, error) {
	return 0, errNotSupported
}

// AppendAncient returns an error as we don't have a backing chain freezer.
func (db *nofreezedb) AppendAncient(number uint64, hash, header, body, receipts, td []byte) error {
	return errNotSupported
}

// TruncateAncients returns an error as we don't have a backing chain freezer.
func (db *nofreezedb) TruncateAncients(items uint64) error {
	return errNotSupported
}

// Sync returns an error as we don't have a backing chain freezer.
func (db *nofreezedb) Sync() error {
	return errNotSupported
}

// NewDatabase creates a high level database on top of a given key-value data
// store without a freezer moving immutable chain segments into cold storage.
func NewDatabase(db ethdb.KeyValueStore) ethdb.Database {
	return &nofreezedb{
		KeyValueStore: db,
	}
}

// NewDatabaseWithFreezer creates a high level database on top of a given key-
// value data store with a freezer moving immutable chain segments into cold
// storage.
func NewDatabaseWithFreezer(db ethdb.KeyValueStore, freezer string, namespace string) (ethdb.Database, error) {
	// Create the idle freezer instance
	frdb, err := newFreezer(freezer, namespace)
	if err != nil {
		return nil, err
	}
	// Since the freezer can be stored separately from the user's key-value database,
	// there's a fairly high probability that the user requests invalid combinations
	// of the freezer and database. Ensure that we don't shoot ourselves in the foot
	// by serving up conflicting data, leading to both datastores getting corrupted.
	//
	//   - If both the freezer and key-value store is empty (no genesis), we just
	//     initialized a new empty freezer, so everything's fine.
	//   - If the key-value store is empty, but the freezer is not, we need to make
	//     sure the user's genesis matches the freezer. That will be checked in the
	//     blockchain, since we don't have the genesis block here (nor should we at
	//     this point care, the key-value/freezer combo is valid).
	//   - If neither the key-value store nor the freezer is empty, cross validate
	//     the genesis hashes to make sure they are compatible. If they are, also
	//     ensure that there's no gap between the freezer and sunsequently leveldb.
	//   - If the key-value store is not empty, but the freezer is we might just be
	//     upgrading to the freezer release, or we might have had a small chain and
	//     not frozen anything yet. Ensure that no blocks are missing yet from the
	//     key-value store, since that would mean we already had an old freezer.

	// If the genesis hash is empty, we have a new key-value store, so nothing to
	// validate in this method. If, however, the genesis hash is not nil, compare
	// it to the freezer content.
	if kvgenesis, _ := db.Get(headerHashKey(0)); len(kvgenesis) > 0 {
		if frozen, _ := frdb.Ancients(); frozen > 0 {
			// If the freezer already contains something, ensure that the genesis blocks
			// match, otherwise we might mix up freezers across chains and destroy both
			// the freezer and the key-value store.
			if frgenesis, _ := frdb.Ancient(freezerHashTable, 0); !bytes.Equal(kvgenesis, frgenesis) {
				return nil, fmt.Errorf("genesis mismatch: %#x (leveldb) != %#x (ancients)", kvgenesis, frgenesis)
			}
			// Key-value store and freezer belong to the same network. Ensure that they
			// are contiguous, otherwise we might end up with a non-functional freezer.
			if kvhash, _ := db.Get(headerHashKey(frozen)); len(kvhash) == 0 {
				// Subsequent header after the freezer limit is missing from the database.
				// Reject startup is the database has a more recent head.
				if *ReadHeaderNumber(db, ReadHeadHeaderHash(db)) > frozen-1 {
					return nil, fmt.Errorf("gap (#%d) in the chain between ancients and leveldb", frozen)
				}
				// Database contains only older data than the freezer, this happens if the
				// state was wiped and reinited from an existing freezer.
			}
			// Otherwise, key-value store continues where the freezer left off, all is fine.
			// We might have duplicate blocks (crash after freezer write but before key-value
			// store deletion, but that's fine).
		} else {
			// If the freezer is empty, ensure nothing was moved yet from the key-value
			// store, otherwise we'll end up missing data. We check block #1 to decide
			// if we froze anything previously or not, but do take care of databases with
			// only the genesis block.
			if ReadHeadHeaderHash(db) != common.BytesToHash(kvgenesis) {
				// Key-value store contains more data than the genesis block, make sure we
				// didn't freeze anything yet.
				if kvblob, _ := db.Get(headerHashKey(1)); len(kvblob) == 0 {
					return nil, errors.New("ancient chain segments already extracted, please set --datadir.ancient to the correct path")
				}
				// Block #1 is still in the database, we're allowed to init a new feezer
			}
			// Otherwise, the head header is still the genesis, we're allowed to init a new
			// feezer.
		}
	}
	// Freezer is consistent with the key-value database, permit combining the two
	go frdb.freeze(db)

	return &freezerdb{
		KeyValueStore: db,
		AncientStore:  frdb,
	}, nil
}

// NewMemoryDatabase creates an ephemeral in-memory key-value database without a
// freezer moving immutable chain segments into cold storage.
func NewMemoryDatabase() ethdb.Database {
	return NewDatabase(memorydb.New())
}

// NewMemoryDatabaseWithCap creates an ephemeral in-memory key-value database
// with an initial starting capacity, but without a freezer moving immutable
// chain segments into cold storage.
func NewMemoryDatabaseWithCap(size int) ethdb.Database {
	return NewDatabase(memorydb.NewWithCap(size))
}

// NewLevelDBDatabase creates a persistent key-value database without a freezer
// moving immutable chain segments into cold storage.
func NewLevelDBDatabase(file string, cache int, handles int, namespace string) (ethdb.Database, error) {
	db, err := leveldb.New(file, cache, handles, namespace)
	if err != nil {
		return nil, err
	}
	return NewDatabase(db), nil
}

// NewLevelDBDatabaseWithFreezer creates a persistent key-value database with a
// freezer moving immutable chain segments into cold storage.
func NewLevelDBDatabaseWithFreezer(file string, cache int, handles int, freezer string, namespace string) (ethdb.Database, error) {
	kvdb, err := leveldb.New(file, cache, handles, namespace)
	if err != nil {
		return nil, err
	}
	frdb, err := NewDatabaseWithFreezer(kvdb, freezer, namespace)
	if err != nil {
		kvdb.Close()
		return nil, err
	}
	return frdb, nil
}

// InspectDatabase traverses the entire database and checks the size
// of all different categories of data.
func InspectDatabase(db ethdb.Database) error {
	it := db.NewIterator(nil, nil)
	defer it.Release()

	var (
		count  int64
		start  = time.Now()
		logged = time.Now()

		// Key-value store statistics
		total           common.StorageSize
		headerSize      common.StorageSize
		bodySize        common.StorageSize
		receiptSize     common.StorageSize
		tdSize          common.StorageSize
		numHashPairing  common.StorageSize
		hashNumPairing  common.StorageSize
		trieSize        common.StorageSize
		txlookupSize    common.StorageSize
		accountSnapSize common.StorageSize
		storageSnapSize common.StorageSize
		preimageSize    common.StorageSize
		bloomBitsSize   common.StorageSize
		cliqueSnapsSize common.StorageSize

		// Ancient store statistics
		ancientHeaders  common.StorageSize
		ancientBodies   common.StorageSize
		ancientReceipts common.StorageSize
		ancientHashes   common.StorageSize
		ancientTds      common.StorageSize

		// Les statistic
		chtTrieNodes   common.StorageSize
		bloomTrieNodes common.StorageSize

		// Meta- and unaccounted data
		metadata    common.StorageSize
		unaccounted common.StorageSize
	)
	// Inspect key-value database first.
	for it.Next() {
		var (
			key  = it.Key()
			size = common.StorageSize(len(key) + len(it.Value()))
		)
		total += size
		switch {
		case bytes.HasPrefix(key, headerPrefix) && bytes.HasSuffix(key, headerTDSuffix):
			tdSize += size
		case bytes.HasPrefix(key, headerPrefix) && bytes.HasSuffix(key, headerHashSuffix):
			numHashPairing += size
		case bytes.HasPrefix(key, headerPrefix) && len(key) == (len(headerPrefix)+8+common.HashLength):
			headerSize += size
		case bytes.HasPrefix(key, headerNumberPrefix) && len(key) == (len(headerNumberPrefix)+common.HashLength):
			hashNumPairing += size
		case bytes.HasPrefix(key, blockBodyPrefix) && len(key) == (len(blockBodyPrefix)+8+common.HashLength):
			bodySize += size
		case bytes.HasPrefix(key, blockReceiptsPrefix) && len(key) == (len(blockReceiptsPrefix)+8+common.HashLength):
			receiptSize += size
		case bytes.HasPrefix(key, txLookupPrefix) && len(key) == (len(txLookupPrefix)+common.HashLength):
			txlookupSize += size
		case bytes.HasPrefix(key, SnapshotAccountPrefix) && len(key) == (len(SnapshotAccountPrefix)+common.HashLength):
			accountSnapSize += size
		case bytes.HasPrefix(key, SnapshotStoragePrefix) && len(key) == (len(SnapshotStoragePrefix)+2*common.HashLength):
			storageSnapSize += size
		case bytes.HasPrefix(key, preimagePrefix) && len(key) == (len(preimagePrefix)+common.HashLength):
			preimageSize += size
		case bytes.HasPrefix(key, bloomBitsPrefix) && len(key) == (len(bloomBitsPrefix)+10+common.HashLength):
			bloomBitsSize += size
		case bytes.HasPrefix(key, []byte("clique-")) && len(key) == 7+common.HashLength:
			cliqueSnapsSize += size
		case bytes.HasPrefix(key, []byte("cht-")) && len(key) == 4+common.HashLength:
			chtTrieNodes += size
		case bytes.HasPrefix(key, []byte("blt-")) && len(key) == 4+common.HashLength:
			bloomTrieNodes += size
		case len(key) == common.HashLength:
			trieSize += size
		default:
			var accounted bool
			for _, meta := range [][]byte{databaseVerisionKey, headHeaderKey, headBlockKey, headFastBlockKey, fastTrieProgressKey} {
				if bytes.Equal(key, meta) {
					metadata += size
					accounted = true
					break
				}
			}
			if !accounted {
				unaccounted += size
			}
		}
		count += 1
		if count%1000 == 0 && time.Since(logged) > 8*time.Second {
			log.Info("Inspecting database", "count", count, "elapsed", common.PrettyDuration(time.Since(start)))
			logged = time.Now()
		}
	}
	// Inspect append-only file store then.
	ancients := []*common.StorageSize{&ancientHeaders, &ancientBodies, &ancientReceipts, &ancientHashes, &ancientTds}
	for i, category := range []string{freezerHeaderTable, freezerBodiesTable, freezerReceiptTable, freezerHashTable, freezerDifficultyTable} {
		if size, err := db.AncientSize(category); err == nil {
			*ancients[i] += common.StorageSize(size)
			total += common.StorageSize(size)
		}
	}
	// Display the database statistic.
	stats := [][]string{
		{"Key-Value store", "Headers", headerSize.String()},
		{"Key-Value store", "Bodies", bodySize.String()},
		{"Key-Value store", "Receipts", receiptSize.String()},
		{"Key-Value store", "Difficulties", tdSize.String()},
		{"Key-Value store", "Block number->hash", numHashPairing.String()},
		{"Key-Value store", "Block hash->number", hashNumPairing.String()},
		{"Key-Value store", "Transaction index", txlookupSize.String()},
		{"Key-Value store", "Bloombit index", bloomBitsSize.String()},
		{"Key-Value store", "Trie nodes", trieSize.String()},
		{"Key-Value store", "Trie preimages", preimageSize.String()},
		{"Key-Value store", "Account snapshot", accountSnapSize.String()},
		{"Key-Value store", "Storage snapshot", storageSnapSize.String()},
		{"Key-Value store", "Clique snapshots", cliqueSnapsSize.String()},
		{"Key-Value store", "Singleton metadata", metadata.String()},
		{"Ancient store", "Headers", ancientHeaders.String()},
		{"Ancient store", "Bodies", ancientBodies.String()},
		{"Ancient store", "Receipts", ancientReceipts.String()},
		{"Ancient store", "Difficulties", ancientTds.String()},
		{"Ancient store", "Block number->hash", ancientHashes.String()},
		{"Light client", "CHT trie nodes", chtTrieNodes.String()},
		{"Light client", "Bloom trie nodes", bloomTrieNodes.String()},
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Database", "Category", "Size"})
	table.SetFooter([]string{"", "Total", total.String()})
	table.AppendBulk(stats)
	table.Render()

	if unaccounted > 0 {
		log.Error("Database contains unaccounted data", "size", unaccounted)
	}
	return nil
}
