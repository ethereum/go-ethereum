package redisdb

import (
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/go-redis/redis"
	"sync"
)

type iter struct {
	db    *Database
	match string
	start string

	lock         sync.Mutex
	scanIterator *redis.ScanIterator
}

func (i *iter) Next() bool {
	i.lock.Lock()
	defer i.lock.Unlock()

	if i.scanIterator == nil {
		i.scanIterator = i.db.client.Scan(0, i.match, -1).Iterator()

		if i.start != "" {
			return nextUntilStart(i.start, i.scanIterator)
		}
	}
	return i.scanIterator.Next()
}

func nextUntilStart(start string, iter *redis.ScanIterator) bool {
	for next := iter.Next(); next; next = iter.Next() {
		if iter.Val() > start {
			return true
		}
	}
	return false
}

func (i *iter) Error() error {
	i.lock.Lock()
	defer i.lock.Unlock()
	if i.scanIterator == nil {
		return nil
	}
	return i.scanIterator.Err()
}

func (i *iter) Key() []byte {
	i.lock.Lock()
	defer i.lock.Unlock()
	if i.scanIterator == nil || i.scanIterator.Err() != nil {
		return nil
	}
	return []byte(i.scanIterator.Val())
}

func (i *iter) Value() []byte {
	i.lock.Lock()
	defer i.lock.Unlock()
	if i.scanIterator == nil || i.scanIterator.Err() != nil {
		return nil
	}

	key := i.scanIterator.Val()
	value, err := i.db.client.Get(key).Bytes()
	if err != nil {
		return nil
	}
	return value
}

func (i *iter) Release() {
	i.scanIterator = nil
}

func newIterator(db *Database, prefix []byte, start []byte) ethdb.Iterator {
	var match string
	if len(prefix) > 0 {
		match = string(prefix) + "*"
	}
	return &iter{
		db:    db,
		match: match,
		start: string(append(prefix, start...)),
	}
}
