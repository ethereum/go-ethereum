package nomtdb

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/nomt/db"
	"github.com/ethereum/go-ethereum/triedb/database"
)

// Database is the NOMT triedb backend. It manages the NOMT trie engine for
// page-based merkle storage and delegates flat state to geth's ethdb.
type Database struct {
	diskdb ethdb.Database // geth's existing PebbleDB for flat state + pages
	nomt   *db.DB         // NOMT trie engine
	config *Config
}

// New creates a new NOMT backend. The diskdb is used for flat state storage,
// NOMT page storage, and metadata. Pass nil config for defaults.
func New(diskdb ethdb.Database, config *Config) *Database {
	if config == nil {
		config = &Config{}
	}
	nomtDB, err := db.New(diskdb, db.Config{
		NumWorkers: config.NumWorkers,
	})
	if err != nil {
		log.Crit("Failed to create NOMT database", "err", err)
	}
	return &Database{
		diskdb: diskdb,
		nomt:   nomtDB,
		config: config,
	}
}

// NomtDB returns the underlying NOMT trie engine.
func (d *Database) NomtDB() *db.DB {
	return d.nomt
}

// DiskDB returns the underlying ethdb for flat state access.
func (d *Database) DiskDB() ethdb.Database {
	return d.diskdb
}

// NodeReader returns a reader for accessing trie nodes within the specified state.
func (d *Database) NodeReader(root common.Hash) (database.NodeReader, error) {
	return &nodeReader{nomt: d.nomt}, nil
}

// StateReader returns a reader for accessing flat states within the specified state.
func (d *Database) StateReader(root common.Hash) (database.StateReader, error) {
	return &stateReader{diskdb: d.diskdb}, nil
}

// Size returns the current storage size of the NOMT database.
// First return is diff layer size (always 0 for NOMT), second is disk size.
func (d *Database) Size() (common.StorageSize, common.StorageSize) {
	return 0, 0
}

// Commit is a no-op for NOMT — pages are synced during trie Hash()/Commit().
func (d *Database) Commit(root common.Hash, report bool) error {
	return nil
}

// Close closes the NOMT database backend.
func (d *Database) Close() error {
	return d.nomt.Close()
}

// Update writes flat state changes to ethdb. The trie pages have already been
// persisted by the NomtTrie during Hash()/Commit().
func (d *Database) Update(accounts map[common.Hash][]byte, storages map[common.Hash]map[common.Hash][]byte) error {
	batch := d.diskdb.NewBatch()

	for accountHash, data := range accounts {
		key := NomtAccountKey(accountHash)
		if len(data) == 0 {
			if err := batch.Delete(key); err != nil {
				return err
			}
		} else {
			if err := batch.Put(key, data); err != nil {
				return err
			}
		}
	}

	for accountHash, slots := range storages {
		for slotHash, value := range slots {
			key := NomtStorageKey(accountHash, slotHash)
			if len(value) == 0 {
				if err := batch.Delete(key); err != nil {
					return err
				}
			} else {
				if err := batch.Put(key, value); err != nil {
					return err
				}
			}
		}
	}

	return batch.Write()
}
