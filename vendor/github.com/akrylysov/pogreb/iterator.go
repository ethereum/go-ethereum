package pogreb

import (
	"errors"
	"sync"
)

// ErrIterationDone is returned by ItemIterator.Next calls when there are no more items to return.
var ErrIterationDone = errors.New("no more items in iterator")

type item struct {
	key   []byte
	value []byte
}

// ItemIterator is an iterator over DB key/value pairs. It iterates the items in an unspecified order.
type ItemIterator struct {
	db            *DB
	nextBucketIdx uint32
	queue         []item
	mu            sync.Mutex
}

// Next returns the next key/value pair if available, otherwise it returns ErrIterationDone error.
func (it *ItemIterator) Next() ([]byte, []byte, error) {
	it.mu.Lock()
	defer it.mu.Unlock()

	it.db.mu.RLock()
	defer it.db.mu.RUnlock()

	if len(it.queue) == 0 {
		for it.nextBucketIdx < it.db.nBuckets {
			err := it.db.forEachBucket(it.nextBucketIdx, func(b bucketHandle) (bool, error) {
				for i := 0; i < slotsPerBucket; i++ {
					sl := b.slots[i]
					if sl.kvOffset == 0 {
						return true, nil
					}
					key, value, err := it.db.data.readKeyValue(sl)
					if err != nil {
						return true, err
					}
					it.queue = append(it.queue, item{key: key, value: value})
				}
				return false, nil
			})
			if err != nil {
				return nil, nil, err
			}
			it.nextBucketIdx++
			if len(it.queue) > 0 {
				break
			}
		}
	}

	if len(it.queue) > 0 {
		item := it.queue[0]
		it.queue = it.queue[1:]
		return item.key, item.value, nil
	}

	return nil, nil, ErrIterationDone
}
