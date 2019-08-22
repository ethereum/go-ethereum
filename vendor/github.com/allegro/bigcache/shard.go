package bigcache

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/allegro/bigcache/queue"
)

type onRemoveCallback func(wrappedEntry []byte, reason RemoveReason)

type cacheShard struct {
	hashmap     map[uint64]uint32
	entries     queue.BytesQueue
	lock        sync.RWMutex
	entryBuffer []byte
	onRemove    onRemoveCallback

	isVerbose  bool
	logger     Logger
	clock      clock
	lifeWindow uint64

	stats Stats
}

func (s *cacheShard) get(key string, hashedKey uint64) ([]byte, error) {
	s.lock.RLock()
	itemIndex := s.hashmap[hashedKey]

	if itemIndex == 0 {
		s.lock.RUnlock()
		s.miss()
		return nil, ErrEntryNotFound
	}

	wrappedEntry, err := s.entries.Get(int(itemIndex))
	if err != nil {
		s.lock.RUnlock()
		s.miss()
		return nil, err
	}
	if entryKey := readKeyFromEntry(wrappedEntry); key != entryKey {
		if s.isVerbose {
			s.logger.Printf("Collision detected. Both %q and %q have the same hash %x", key, entryKey, hashedKey)
		}
		s.lock.RUnlock()
		s.collision()
		return nil, ErrEntryNotFound
	}
	entry := readEntry(wrappedEntry)
	s.lock.RUnlock()
	s.hit()
	return entry, nil
}

func (s *cacheShard) set(key string, hashedKey uint64, entry []byte) error {
	currentTimestamp := uint64(s.clock.epoch())

	s.lock.Lock()

	if previousIndex := s.hashmap[hashedKey]; previousIndex != 0 {
		if previousEntry, err := s.entries.Get(int(previousIndex)); err == nil {
			resetKeyFromEntry(previousEntry)
		}
	}

	if oldestEntry, err := s.entries.Peek(); err == nil {
		s.onEvict(oldestEntry, currentTimestamp, s.removeOldestEntry)
	}

	w := wrapEntry(currentTimestamp, hashedKey, key, entry, &s.entryBuffer)

	for {
		if index, err := s.entries.Push(w); err == nil {
			s.hashmap[hashedKey] = uint32(index)
			s.lock.Unlock()
			return nil
		}
		if s.removeOldestEntry(NoSpace) != nil {
			s.lock.Unlock()
			return fmt.Errorf("entry is bigger than max shard size")
		}
	}
}

func (s *cacheShard) del(key string, hashedKey uint64) error {
	// Optimistic pre-check using only readlock
	s.lock.RLock()
	itemIndex := s.hashmap[hashedKey]

	if itemIndex == 0 {
		s.lock.RUnlock()
		s.delmiss()
		return ErrEntryNotFound
	}

	if err := s.entries.CheckGet(int(itemIndex)); err != nil {
		s.lock.RUnlock()
		s.delmiss()
		return err
	}
	s.lock.RUnlock()

	s.lock.Lock()
	{
		// After obtaining the writelock, we need to read the same again,
		// since the data delivered earlier may be stale now
		itemIndex = s.hashmap[hashedKey]

		if itemIndex == 0 {
			s.lock.Unlock()
			s.delmiss()
			return ErrEntryNotFound
		}

		wrappedEntry, err := s.entries.Get(int(itemIndex))
		if err != nil {
			s.lock.Unlock()
			s.delmiss()
			return err
		}

		delete(s.hashmap, hashedKey)
		s.onRemove(wrappedEntry, Deleted)
		resetKeyFromEntry(wrappedEntry)
	}
	s.lock.Unlock()

	s.delhit()
	return nil
}

func (s *cacheShard) onEvict(oldestEntry []byte, currentTimestamp uint64, evict func(reason RemoveReason) error) bool {
	oldestTimestamp := readTimestampFromEntry(oldestEntry)
	if currentTimestamp-oldestTimestamp > s.lifeWindow {
		evict(Expired)
		return true
	}
	return false
}

func (s *cacheShard) cleanUp(currentTimestamp uint64) {
	s.lock.Lock()
	for {
		if oldestEntry, err := s.entries.Peek(); err != nil {
			break
		} else if evicted := s.onEvict(oldestEntry, currentTimestamp, s.removeOldestEntry); !evicted {
			break
		}
	}
	s.lock.Unlock()
}

func (s *cacheShard) getOldestEntry() ([]byte, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.entries.Peek()
}

func (s *cacheShard) getEntry(index int) ([]byte, error) {
	s.lock.RLock()
	entry, err := s.entries.Get(index)
	s.lock.RUnlock()

	return entry, err
}

func (s *cacheShard) copyKeys() (keys []uint32, next int) {
	s.lock.RLock()
	keys = make([]uint32, len(s.hashmap))

	for _, index := range s.hashmap {
		keys[next] = index
		next++
	}

	s.lock.RUnlock()
	return keys, next
}

func (s *cacheShard) removeOldestEntry(reason RemoveReason) error {
	oldest, err := s.entries.Pop()
	if err == nil {
		hash := readHashFromEntry(oldest)
		delete(s.hashmap, hash)
		s.onRemove(oldest, reason)
		return nil
	}
	return err
}

func (s *cacheShard) reset(config Config) {
	s.lock.Lock()
	s.hashmap = make(map[uint64]uint32, config.initialShardSize())
	s.entryBuffer = make([]byte, config.MaxEntrySize+headersSizeInBytes)
	s.entries.Reset()
	s.lock.Unlock()
}

func (s *cacheShard) len() int {
	s.lock.RLock()
	res := len(s.hashmap)
	s.lock.RUnlock()
	return res
}

func (s *cacheShard) capacity() int {
	s.lock.RLock()
	res := s.entries.Capacity()
	s.lock.RUnlock()
	return res
}

func (s *cacheShard) getStats() Stats {
	var stats = Stats{
		Hits:       atomic.LoadInt64(&s.stats.Hits),
		Misses:     atomic.LoadInt64(&s.stats.Misses),
		DelHits:    atomic.LoadInt64(&s.stats.DelHits),
		DelMisses:  atomic.LoadInt64(&s.stats.DelMisses),
		Collisions: atomic.LoadInt64(&s.stats.Collisions),
	}
	return stats
}

func (s *cacheShard) hit() {
	atomic.AddInt64(&s.stats.Hits, 1)
}

func (s *cacheShard) miss() {
	atomic.AddInt64(&s.stats.Misses, 1)
}

func (s *cacheShard) delhit() {
	atomic.AddInt64(&s.stats.DelHits, 1)
}

func (s *cacheShard) delmiss() {
	atomic.AddInt64(&s.stats.DelMisses, 1)
}

func (s *cacheShard) collision() {
	atomic.AddInt64(&s.stats.Collisions, 1)
}

func initNewShard(config Config, callback onRemoveCallback, clock clock) *cacheShard {
	return &cacheShard{
		hashmap:     make(map[uint64]uint32, config.initialShardSize()),
		entries:     *queue.NewBytesQueue(config.initialShardSize()*config.MaxEntrySize, config.maximumShardSize(), config.Verbose),
		entryBuffer: make([]byte, config.MaxEntrySize+headersSizeInBytes),
		onRemove:    callback,

		isVerbose:  config.Verbose,
		logger:     newLogger(config.Logger),
		clock:      clock,
		lifeWindow: uint64(config.LifeWindow.Seconds()),
	}
}
