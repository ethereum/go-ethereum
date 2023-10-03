package bloom

import (
	"hash"
	"sync"
	"time"

	bloomfilter "github.com/holiman/bloomfilter/v2"
)

const (
	k = 4
)

type ExpiringBloom struct {
	currentBloom int
	union        *bloomfilter.Filter
	blooms       []*bloomfilter.Filter
	size         uint64

	timer *time.Ticker
	// Mutex lock the currentBloom and union variables
	mu      sync.RWMutex
	closeCh chan struct{}
}

func NewExpiringBloom(n, m uint64, timeout time.Duration) (*ExpiringBloom, error) {
	union, err := bloomfilter.New(m*8, k)
	if err != nil {
		return nil, err
	}

	blooms := make([]*bloomfilter.Filter, 0, n)
	for i := 0; i < int(n); i++ {
		filter, err := union.NewCompatible()
		if err != nil {
			return nil, err
		}
		blooms = append(blooms, filter)
	}

	filter := ExpiringBloom{
		currentBloom: 0,
		blooms:       blooms,
		union:        union,
		size:         m * 8,
		timer:        time.NewTicker(timeout),
		closeCh:      make(chan struct{}),
	}
	go filter.loop()

	return &filter, nil
}

func (e *ExpiringBloom) loop() {
	for {
		select {
		case <-e.timer.C:
			e.tick()
		case <-e.closeCh:
			return
		}
	}
}

func (e *ExpiringBloom) tick() {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Advance the current bloom
	e.currentBloom++
	if e.currentBloom == len(e.blooms) {
		e.currentBloom = 0
	}
	// Clear the filter
	e.blooms[e.currentBloom], _ = e.blooms[e.currentBloom].NewCompatible()
	// Recreate the union filter
	e.union, _ = e.union.NewCompatible()
	for _, bloom := range e.blooms {
		e.union.UnionInPlace(bloom)
	}
}

func (e *ExpiringBloom) Stop() {
	close(e.closeCh)
}

func (e *ExpiringBloom) Add(key hash.Hash64) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	e.blooms[e.currentBloom].Add(key)
	e.union.Add(key)
}

func (e *ExpiringBloom) Contains(key hash.Hash64) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.union.Contains(key)
}
