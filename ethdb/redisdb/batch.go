package redisdb

import (
	"github.com/ethereum/go-ethereum/ethdb"
)

// batch is a write-only memory batch that commits changes to its host
// database when Write is called. A batch cannot be used concurrently.
type batch struct {
	db      *Database
	deletes []string
	writes  []string
	size    int
}

// Put inserts the given value into the batch for later committing.
func (b *batch) Put(key, value []byte) error {
	b.writes = append(b.writes, string(key), string(value))
	b.size += len(key) + len(value)
	return nil
}

// Delete inserts the a key removal into the batch for later committing.
func (b *batch) Delete(key []byte) error {
	if len(key) == 0 {
		return nil
	}
	b.deletes = append(b.deletes, string(key))
	b.size += len(key)
	return nil
}

// ValueSize retrieves the amount of data queued up for writing.
func (b *batch) ValueSize() int {
	return b.size
}

// Write flushes any accumulated data to the memory database.
func (b *batch) Write() error {
	if len(b.deletes) > 0 {
		err := b.db.client.Del(b.deletes...).Err()
		if err != nil {
			return err
		}
	}

	if len(b.writes) > 0 {
		err := b.db.client.MSet(b.writes).Err()
		if err != nil {
			return err
		}
	}

	return nil
}

// Replay replays the batch contents.
func (b *batch) Replay(w ethdb.KeyValueWriter) error {
	for _, key := range b.deletes {
		if err := w.Delete([]byte(key)); err != nil {
			return err
		}
	}
	for i := 0; i < len(b.writes); i += 2 {
		if err := w.Put([]byte(b.writes[i]), []byte(b.writes[i+1])); err != nil {
			return err
		}
	}
	return nil
}

// Reset resets the batch for reuse.
func (b *batch) Reset() {
	b.deletes = b.deletes[:0]
	b.writes = b.writes[:0]
	b.size = 0
}

func newBatch(db *Database, size int) ethdb.Batch {
	return &batch{
		db: db,
	}
}
