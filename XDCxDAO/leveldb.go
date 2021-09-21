package XDCxDAO

import (
	"bytes"
	"encoding/hex"
	"errors"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/core/rawdb"
	"sync"

	"github.com/XinFinOrg/XDPoSChain/ethdb"
	"github.com/XinFinOrg/XDPoSChain/log"
	lru "github.com/hashicorp/golang-lru"
)

type BatchItem struct {
	Value interface{}
}

type BatchDatabase struct {
	db         ethdb.Database
	emptyKey   []byte
	cacheItems *lru.Cache // Cache for reading
	lock       sync.RWMutex
	cacheLimit int
	Debug      bool
}

// NewBatchDatabase use rlp as encoding
func NewBatchDatabase(datadir string, cacheLimit int) *BatchDatabase {
	return NewBatchDatabaseWithEncode(datadir, cacheLimit)
}

// batchdatabase is a fast cache db to retrieve in-mem object
func NewBatchDatabaseWithEncode(datadir string, cacheLimit int) *BatchDatabase {
	db, err := rawdb.NewLevelDBDatabase(datadir, 128, 1024, "")
	if err != nil {
		log.Error("Can't create new DB", "error", err)
		return nil
	}
	itemCacheLimit := defaultCacheLimit
	if cacheLimit > 0 {
		itemCacheLimit = cacheLimit
	}

	cacheItems, _ := lru.New(itemCacheLimit)

	batchDB := &BatchDatabase{
		db:         db,
		cacheItems: cacheItems,
		emptyKey:   EmptyKey(), // pre alloc for comparison
		cacheLimit: itemCacheLimit,
	}

	return batchDB

}

func (db *BatchDatabase) IsEmptyKey(key []byte) bool {
	return key == nil || len(key) == 0 || bytes.Equal(key, db.emptyKey)
}

func (db *BatchDatabase) getCacheKey(key []byte) string {
	return hex.EncodeToString(key)
}

func (db *BatchDatabase) HasObject(hash common.Hash, val interface{}) (bool, error) {
	// for mongodb only
	return false, nil
}

func (db *BatchDatabase) GetObject(hash common.Hash, val interface{}) (interface{}, error) {
	// for mongodb only
	return nil, nil
}

func (db *BatchDatabase) PutObject(hash common.Hash, val interface{}) error {
	// for mongodb only
	return nil
}

func (db *BatchDatabase) DeleteObject(hash common.Hash, val interface{}) error {
	// for mongodb only
	return nil
}

func (db *BatchDatabase) Put(key []byte, val []byte) error {
	return db.db.Put(key, val)
}

func (db *BatchDatabase) Delete(key []byte) error {
	return db.db.Delete(key)
}

func (db *BatchDatabase) Has(key []byte) (bool, error) {
	return db.db.Has(key)
}

func (db *BatchDatabase) Get(key []byte) ([]byte, error) {
	return db.db.Get(key)
}

func (db *BatchDatabase) Close() error {
	return db.db.Close()
}

func (db *BatchDatabase) NewBatch() ethdb.Batch {
	return db.db.NewBatch()
}

func (db *BatchDatabase) DeleteItemByTxHash(txhash common.Hash, val interface{}) {
}

func (db *BatchDatabase) GetListItemByTxHash(txhash common.Hash, val interface{}) interface{} {
	return []interface{}{}
}

func (db *BatchDatabase) GetListItemByHashes(hashes []string, val interface{}) interface{} {
	return []interface{}{}
}

func (db *BatchDatabase) InitBulk() {
}

func (db *BatchDatabase) CommitBulk() error {
	return nil
}

func (db *BatchDatabase) InitLendingBulk() {
}

func (db *BatchDatabase) CommitLendingBulk() error {
	return nil
}

var errNotSupported = errors.New("this operation is not supported")

// HasAncient returns an error as we don't have a backing chain freezer.
func (db *BatchDatabase) HasAncient(kind string, number uint64) (bool, error) {
	return false, errNotSupported
}

// Ancient returns an error as we don't have a backing chain freezer.
func (db *BatchDatabase) Ancient(kind string, number uint64) ([]byte, error) {
	return nil, errNotSupported
}

// Ancients returns an error as we don't have a backing chain freezer.
func (db *BatchDatabase) Ancients() (uint64, error) {
	return 0, errNotSupported
}

// AncientSize returns an error as we don't have a backing chain freezer.
func (db *BatchDatabase) AncientSize(kind string) (uint64, error) {
	return 0, errNotSupported
}

// AppendAncient returns an error as we don't have a backing chain freezer.
func (db *BatchDatabase) AppendAncient(number uint64, hash, header, body, receipts, td []byte) error {
	return errNotSupported
}

// TruncateAncients returns an error as we don't have a backing chain freezer.
func (db *BatchDatabase) TruncateAncients(items uint64) error {
	return errNotSupported
}

// Sync returns an error as we don't have a backing chain freezer.
func (db *BatchDatabase) Sync() error {
	return errNotSupported
}

func (db *BatchDatabase) NewIterator(prefix []byte, start []byte) ethdb.Iterator {
	return db.NewIterator(prefix, start)
}

func (db *BatchDatabase) Stat(property string) (string, error) {
	return db.Stat(property)
}

func (db *BatchDatabase) Compact(start []byte, limit []byte) error {
	return db.Compact(start, limit)
}
