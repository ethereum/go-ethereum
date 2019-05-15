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
	if err := frdb.KeyValueStore.Close(); err != nil {
		errs = append(errs, err)
	}
	if err := frdb.AncientStore.Close(); err != nil {
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
	frdb, err := newFreezer(freezer, namespace)
	if err != nil {
		return nil, err
	}
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
	// Make sure we always use the same ancient store.
	//
	//                 | stored == nil    | stored != nil
	// ----------------+------------------+----------------------
	// freezer == nil  | non-freezer mode | ancient store missing
	// freezer != nil  | initialize       | ensure consistency
	stored := ReadAncientPath(kvdb)
	if stored == "" && freezer != "" {
		WriteAncientPath(kvdb, freezer)
	} else if stored != freezer {
		log.Warn("Ancient path mismatch", "stored", stored, "given", freezer)
		log.Crit("Please use a consistent ancient path or migrate it via the command line tool `geth migrate-ancient`")
	}
	return frdb, nil
}

// InspectDatabase traverses the entire database and checks the size
// of all different categories of data.
func InspectDatabase(db ethdb.Database) error {
	it := db.NewIterator()
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
			for _, meta := range [][]byte{databaseVerisionKey, headHeaderKey, headBlockKey, headFastBlockKey, fastTrieProgressKey, ancientKey} {
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
