package redisdb

import (
	"context"
	"sync"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/redis/go-redis/v9"
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

	ctx := context.Background()
	if i.scanIterator == nil {
		i.scanIterator = i.db.client.Scan(ctx, 0, i.match, -1).Iterator()

		if i.start != "" {
			return nextUntilStart(i.start, i.scanIterator)
		}
	}
	return i.scanIterator.Next(ctx)
}

func nextUntilStart(start string, iter *redis.ScanIterator) bool {
	ctx := context.Background()
	for next := iter.Next(ctx); next; next = iter.Next(ctx) {
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

	ctx := context.Background()
	key := i.scanIterator.Val()
	value, err := i.db.client.Get(ctx, key).Bytes()
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
