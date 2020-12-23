package bigcache

import "sync"

type iteratorError string

func (e iteratorError) Error() string {
	return string(e)
}

// ErrInvalidIteratorState is reported when iterator is in invalid state
const ErrInvalidIteratorState = iteratorError("Iterator is in invalid state. Use SetNext() to move to next position")

// ErrCannotRetrieveEntry is reported when entry cannot be retrieved from underlying
const ErrCannotRetrieveEntry = iteratorError("Could not retrieve entry from cache")

var emptyEntryInfo = EntryInfo{}

// EntryInfo holds informations about entry in the cache
type EntryInfo struct {
	timestamp uint64
	hash      uint64
	key       string
	value     []byte
}

// Key returns entry's underlying key
func (e EntryInfo) Key() string {
	return e.key
}

// Hash returns entry's hash value
func (e EntryInfo) Hash() uint64 {
	return e.hash
}

// Timestamp returns entry's timestamp (time of insertion)
func (e EntryInfo) Timestamp() uint64 {
	return e.timestamp
}

// Value returns entry's underlying value
func (e EntryInfo) Value() []byte {
	return e.value
}

// EntryInfoIterator allows to iterate over entries in the cache
type EntryInfoIterator struct {
	mutex         sync.Mutex
	cache         *BigCache
	currentShard  int
	currentIndex  int
	elements      []uint32
	elementsCount int
	valid         bool
}

// SetNext moves to next element and returns true if it exists.
func (it *EntryInfoIterator) SetNext() bool {
	it.mutex.Lock()

	it.valid = false
	it.currentIndex++

	if it.elementsCount > it.currentIndex {
		it.valid = true
		it.mutex.Unlock()
		return true
	}

	for i := it.currentShard + 1; i < it.cache.config.Shards; i++ {
		it.elements, it.elementsCount = it.cache.shards[i].copyKeys()

		// Non empty shard - stick with it
		if it.elementsCount > 0 {
			it.currentIndex = 0
			it.currentShard = i
			it.valid = true
			it.mutex.Unlock()
			return true
		}
	}
	it.mutex.Unlock()
	return false
}

func newIterator(cache *BigCache) *EntryInfoIterator {
	elements, count := cache.shards[0].copyKeys()

	return &EntryInfoIterator{
		cache:         cache,
		currentShard:  0,
		currentIndex:  -1,
		elements:      elements,
		elementsCount: count,
	}
}

// Value returns current value from the iterator
func (it *EntryInfoIterator) Value() (EntryInfo, error) {
	it.mutex.Lock()

	if !it.valid {
		it.mutex.Unlock()
		return emptyEntryInfo, ErrInvalidIteratorState
	}

	entry, err := it.cache.shards[it.currentShard].getEntry(int(it.elements[it.currentIndex]))

	if err != nil {
		it.mutex.Unlock()
		return emptyEntryInfo, ErrCannotRetrieveEntry
	}
	it.mutex.Unlock()

	return EntryInfo{
		timestamp: readTimestampFromEntry(entry),
		hash:      readHashFromEntry(entry),
		key:       readKeyFromEntry(entry),
		value:     readEntry(entry),
	}, nil
}
