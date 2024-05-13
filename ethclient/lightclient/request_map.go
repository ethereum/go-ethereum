// Copyright 2024 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package lightclient

import (
	"context"
	"sync"
)

type requestMap[K comparable, V any] struct {
	lock      sync.Mutex
	requestFn func(context.Context, K) (V, error)
	requests  map[K]*mappedRequest[K, V]
}

func newRequestMap[K comparable, V any](requestFn func(context.Context, K) (V, error)) *requestMap[K, V] {
	return &requestMap[K, V]{
		requestFn: requestFn,
		requests:  make(map[K]*mappedRequest[K, V]),
	}
}

func (rm *requestMap[K, V]) request(key K) *mappedRequest[K, V] {
	rm.lock.Lock()
	defer rm.lock.Unlock()

	if r, ok := rm.requests[key]; ok {
		r.lock.Lock()
		r.refCount++
		r.lock.Unlock()
		return r
	}
	ctx, cancelFn := context.WithCancel(context.Background())
	r := &mappedRequest[K, V]{
		rm:          rm,
		key:         key,
		refCount:    1,
		deliveredCh: make(chan struct{}),
		cancelFn:    cancelFn,
	}
	rm.requests[key] = r
	if rm.requestFn != nil {
		go func() {
			result, err := rm.requestFn(ctx, key)
			r.deliver(result, err)
		}()
	}
	return r
}

func (rm *requestMap[K, V]) has(key K) bool {
	rm.lock.Lock()
	defer rm.lock.Unlock()

	_, ok := rm.requests[key]
	return ok
}

func (rm *requestMap[K, V]) allKeys() []K {
	rm.lock.Lock()
	defer rm.lock.Unlock()

	keys := make([]K, 0, len(rm.requests))
	for key := range rm.requests {
		keys = append(keys, key)
	}
	return keys
}

// should only be called with validated results of successful requests
func (rm *requestMap[K, V]) tryDeliver(key K, result V) {
	rm.lock.Lock()
	defer rm.lock.Unlock()

	if r, ok := rm.requests[key]; ok {
		r.deliver(result, nil)
	}
}

type mappedRequest[K comparable, V any] struct {
	lock        sync.Mutex
	rm          *requestMap[K, V]
	key         K
	refCount    int
	delivered   bool
	deliveredCh chan struct{}
	cancelFn    func() // called when delivered || refCount == 0 becomes true
	result      V
	err         error
}

func (r *mappedRequest[K, V]) deliver(result V, err error) {
	r.lock.Lock()
	if !r.delivered {
		r.result, r.err = result, err
		r.delivered = true
		close(r.deliveredCh)
		if r.refCount != 0 {
			r.cancelFn()
		}
	}
	r.lock.Unlock()
}

func (r *mappedRequest[K, V]) getResult(ctx context.Context) (V, error) {
	select {
	case <-r.deliveredCh:
		// not changed after deliveredCh is closed
		return r.result, r.err
	case <-ctx.Done():
		var null V
		return null, ctx.Err()
	}
}

func (r *mappedRequest[K, V]) release() {
	r.rm.lock.Lock()
	r.lock.Lock()
	r.refCount--
	if r.refCount == 0 {
		delete(r.rm.requests, r.key)
		if !r.delivered {
			r.cancelFn()
		}
	}
	r.lock.Unlock()
	r.rm.lock.Unlock()
}
