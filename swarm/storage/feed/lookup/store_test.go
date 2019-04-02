package lookup_test

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/swarm/log"
	"github.com/ethereum/go-ethereum/swarm/storage/feed/lookup"
)

type Data struct {
	Payload uint64
	Time    uint64
}

func (d *Data) String() string {
	return fmt.Sprintf("%d-%d", d.Payload, d.Time)
}

type DataMap map[lookup.EpochID]*Data

type StoreConfig struct {
	CacheReadTime      time.Duration
	FailedReadTime     time.Duration
	SuccessfulReadTime time.Duration
}

type StoreCounters struct {
	reads     int
	cacheHits int
	failed    int
	sucessful int
	canceled int
}

type Store struct {
	StoreConfig
	StoreCounters
	data  DataMap
	cache DataMap
	lock  sync.RWMutex
}

func NewStore(config *StoreConfig) *Store {
	store := &Store{
		StoreConfig: *config,
		data:        make(DataMap),
	}

	store.Reset()
	return store
}

func (s *Store) Reset() {
	s.cache = make(DataMap)
	s.StoreCounters = StoreCounters{}
}

func (s *Store) Put(epoch lookup.Epoch, value *Data) {
	log.Debug("Write: %d-%d, value='%d'\n", epoch.Base(), epoch.Level, value.Payload)
	s.data[epoch.ID()] = value
}

func (s *Store) Update(last lookup.Epoch, now uint64, value *Data) lookup.Epoch {
	epoch := lookup.GetNextEpoch(last, now)
	s.Put(epoch, value)
	return epoch
}

func (s *Store) Get(ctx context.Context, epoch lookup.Epoch, now uint64) (value interface{}, err error) {
	epochID := epoch.ID()
	var operationTime time.Duration
	s.reads++

	defer func() { // simulate a delay according to what has actually happened
		select {
		case <-lookup.TimeAfter(operationTime):
		case <-ctx.Done():
			s.lock.Lock()
			s.canceled++
			s.lock.Unlock()
			value = nil
			err = ctx.Err()
		}
	}()

	s.lock.Lock()
	defer s.lock.Unlock()

	// 1.- Simulate a cache read
	item := s.cache[epochID]
	operationTime += s.CacheReadTime

	if item != nil {
		s.cacheHits++
		if item.Time <= now {
			s.sucessful++
			return item, nil
		}
		return nil, nil
	}

	// 2.- simulate a full read

	item = s.data[epochID]
	if item != nil {
		operationTime += s.SuccessfulReadTime
		s.sucessful++
		s.cache[epochID] = item
		if item.Time <= now {
			return item, nil
		}
	} else {
		operationTime += s.FailedReadTime
		s.failed++
	}
	return nil, nil
}

func (s *Store) MakeReadFunc() lookup.ReadFunc {
	return func(ctx context.Context, epoch lookup.Epoch, now uint64) (interface{}, error) {
		return s.Get(ctx, epoch, now)
	}
}
