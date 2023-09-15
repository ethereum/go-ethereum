package bloom

import (
	"hash"
	"sync"
	"time"

	bloomfilter "github.com/holiman/bloomfilter/v2"
)

type ExpiringBloom struct {
	currentBloom int
	union        *bloomfilter.Filter
	blooms       []*bloomfilter.Filter
	filterM      uint64
	filterK      uint64

	timer   *time.Ticker
	mu      sync.RWMutex // Mutex only locks the currentBloom variable
	closeCh chan struct{}
}

func NewExpiringBloom(n, m, k uint64, timeout time.Duration) *ExpiringBloom {
	blooms := make([]*bloomfilter.Filter, 0, n)
	for i := 0; i < int(n); i++ {
		filter, err := bloomfilter.New(m, k)
		if err != nil {
			panic(err)
		}
		blooms = append(blooms, filter)
	}
	union, err := bloomfilter.New(m, k)
	if err != nil {
		panic(err)
	}
	filter := ExpiringBloom{
		currentBloom: 0,
		blooms:       blooms,
		union:        union,
		filterM:      m,
		filterK:      k,
		timer:        time.NewTicker(timeout),
		closeCh:      make(chan struct{}),
	}
	go filter.loop()
	return &filter
}

func (e *ExpiringBloom) loop() {
	for {
		select {
		case <-e.timer.C:
			// Reset the filters on every tick
			e.mu.Lock()
			var err error
			e.blooms[e.currentBloom], err = bloomfilter.New(e.filterM, e.filterK)
			if err != nil {
				panic(err)
			}
			e.currentBloom++
			if e.currentBloom == len(e.blooms)-1 {
				e.currentBloom = 0
			}
			// Recreate the union filter
			e.union, err = bloomfilter.New(e.filterM, e.filterK)
			if err != nil {
				panic(err)
			}
			for _, bloom := range e.blooms {
				e.union.UnionInPlace(bloom)
			}
			e.mu.Unlock()
		case <-e.closeCh:
			break
		}
	}
}

func (e *ExpiringBloom) Stop() {
	close(e.closeCh)
}

func (e *ExpiringBloom) Put(key hash.Hash64) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	e.blooms[e.currentBloom].Add(key)
	e.union.Add(key)
}

func (e *ExpiringBloom) Contain(key hash.Hash64) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.union.Contains(key)
}
