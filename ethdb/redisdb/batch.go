package redisdb

import (
	"context"

	"github.com/ethereum/go-ethereum/ethdb"
)

type batchOperation struct {
	values []string
	delete bool
}

// redisBatch is a write-only memory redisBatch that commits changes to its host
// database when Write is called. A redisBatch cannot be used concurrently.
type redisBatch struct {
	db         *Database
	initSize   int
	operations []*batchOperation
	size       int
}

func (b *redisBatch) getBatchOperation(delete bool) *batchOperation {
	l := len(b.operations)
	if l > 0 && len(b.operations[l-1].values) < b.initSize && b.operations[l-1].delete == delete {
		return b.operations[l-1]
	}

	op := &batchOperation{
		values: make([]string, 0, b.initSize),
		delete: delete,
	}
	b.operations = append(b.operations, op)
	return op
}

// Put inserts the given value into the batch for later committing.
func (b *redisBatch) Put(key, value []byte) error {
	if len(key) == 0 {
		return nil
	}
	op := b.getBatchOperation(false)
	op.values = append(op.values, string(key), string(value))
	b.size += len(key) + len(value)
	return nil
}

// Delete inserts
func (b *redisBatch) Delete(key []byte) error {
	if len(key) == 0 {
		return nil
	}
	op := b.getBatchOperation(true)
	op.values = append(op.values, string(key))
	b.size += len(key)
	return nil
}

// ValueSize retrieves the amount of data queued up for writing.
func (b *redisBatch) ValueSize() int {
	return b.size
}

// Write flushes any accumulated data to the memory database.
func (b *redisBatch) Write() error {
	ctx := context.Background()
	for _, op := range b.operations {
		if op.delete {
			err := b.db.client.Del(ctx, op.values...).Err()
			if err != nil {
				return err
			}
		} else {
			err := b.db.client.MSet(ctx, op.values).Err()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Replay replays the batch contents.
func (b *redisBatch) Replay(w ethdb.KeyValueWriter) error {
	for _, op := range b.operations {
		if op.delete {
			for _, key := range op.values {
				if err := w.Delete([]byte(key)); err != nil {
					return err
				}
			}
		} else {
			for i := 0; i < len(op.values); i += 2 {
				if err := w.Put([]byte(op.values[i]), []byte(op.values[i+1])); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// Reset resets the batch for reuse.
func (b *redisBatch) Reset() {
	b.operations = b.operations[:0]
	b.size = 0
}

func newBatch(db *Database, size int) ethdb.Batch {
	if size <= 0 {
		size = 20
	}
	return &redisBatch{
		db:       db,
		initSize: size,
	}
}
