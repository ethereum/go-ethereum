package redisdb

import (
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/go-redis/redis"
	"time"
)

// Database is a key-value lookup for redis.
type Database struct {
	client *redis.Client
}

func (db *Database) key(key []byte) string {
	return string(key)
}

func (db *Database) Has(key []byte) (bool, error) {
	err := db.client.Get(db.key(key)).Err()
	if err == nil {
		return true, nil
	}
	if err == redis.Nil {
		return false, nil
	}
	return false, err
}

func (db *Database) Get(key []byte) ([]byte, error) {
	v, err := db.client.Get(db.key(key)).Result()
	if err != nil {
		if err == redis.Nil {
			//TODO return nil or empty slice?
			return nil, nil
		}
		return nil, err
	}

	return []byte(v), nil
}

func (db *Database) Put(key []byte, value []byte) error {
	return db.client.Set(db.key(key), value, time.Duration(0)).Err()
}

func (db *Database) Delete(key []byte) error {
	return db.client.Del(db.key(key)).Err()
}

func (db *Database) Close() error {
	return db.client.Close()
}

func (db *Database) Stat(property string) (string, error) {
	config := db.client.ConfigGet(property)
	if err := config.Err(); err != nil {
		return "", err
	}
	return config.String(), nil
}

func (db *Database) NewBatch() ethdb.Batch {
	return newBatch(db, 0)
}

func (db *Database) NewBatchWithSize(size int) ethdb.Batch {
	return newBatch(db, size)
}

func (db *Database) NewIterator(prefix []byte, start []byte) ethdb.Iterator {
	return newIterator(db, prefix, start)
}

func (db *Database) Compact(start []byte, limit []byte) error {
	//Does nothing
	return nil
}

func (db *Database) NewSnapshot() (ethdb.Snapshot, error) {
	return newSnapshot(db)
}

func New(endpoint string) (*Database, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     endpoint,
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	return &Database{rdb}, nil
}
