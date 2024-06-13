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
	"errors"
	"sync"
)

// requestMap maps pending requests onto a keyspace in order to avoid multiple
// processes requesting the same thing simultaneously. If a new item is requested
// by a process, it is added to the map and subsequest requesters of the same
// item will join the same request and receive the same result. An item stays in
// the map until it is explicitly removed or released by every requester.
type requestMap[K comparable, V any] struct {
	lock      sync.Mutex
	closed    bool
	requestFn func(context.Context, K) (V, error)
	requests  map[K]*mappedRequest[K, V]
}

// mappedRequest represents a request that multiple processes can join.
type mappedRequest[K comparable, V any] struct {
	lock                       sync.Mutex
	rm                         *requestMap[K, V] // in the map if !closed && !removed && refCount != 0
	key                        K
	refCount                   int
	closed, delivered, removed bool
	deliveredCh                chan struct{}
	cancelFn                   func() // called when closed || delivered || refCount == 0 becomes true
	result                     V
	err                        error
}

// newRequestMap creates a new requestMap with the specified key and value types.
// If requestFn is specified then it is called by requestMap every time a new
// request is added to the map and the return value or error is automatically
// delivered.
// If requestFn == nil then the caller is expected to keep track of open requests
// and eventually deliver a result for them.
func newRequestMap[K comparable, V any](requestFn func(context.Context, K) (V, error)) *requestMap[K, V] {
	return &requestMap[K, V]{
		requestFn: requestFn,
		requests:  make(map[K]*mappedRequest[K, V]),
	}
}

// close cancels all pending requests. Subsequent request attempts create dummy
// requests that are not sent or added to the map and instantly return an error
// result. This simplifies shutdown check on the caller side because there has
// to be an error check mechanism for request results anyway and there is no need
// to also check for errors at request creation.
func (rm *requestMap[K, V]) close() {
	rm.lock.Lock()
	defer rm.lock.Unlock()

	if rm.closed {
		return
	}
	for _, r := range rm.requests {
		r.lock.Lock()
		if !r.delivered && r.refCount != 0 {
			r.cancelFn()
		}
		r.closed = true
		r.removed = true
		r.lock.Unlock()
	}
	rm.requests = nil
	rm.closed = true
}

// request either creates or returns an existing mappedRequest for the given key.
func (rm *requestMap[K, V]) request(key K) *mappedRequest[K, V] {
	rm.lock.Lock()
	defer rm.lock.Unlock()

	if rm.closed {
		// return a dummy removed request for simplicity
		r := &mappedRequest[K, V]{
			rm:          rm,
			key:         key,
			closed:      true,
			removed:     true,
			refCount:    1,
			deliveredCh: make(chan struct{}),
		}
		var null V
		r.deliver(null, errors.New("request map is closed"))
		return r
	}
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

// remove removes the given key from the request map, ensuring that the next
// request attempt for the same key will start a new request and potentially
// yield a different result. The old mappedRequest stays accessible for those
// who had a direct reference to it. Remove does not cancel the old request if
// it is still pending.
func (rm *requestMap[K, V]) remove(key K) {
	rm.lock.Lock()
	defer rm.lock.Unlock()

	if r, ok := rm.requests[key]; ok {
		r.lock.Lock()
		r.removed = true
		r.lock.Unlock()
		delete(rm.requests, key)
	}
}

// has returns true if the request map has a request for the given key.
func (rm *requestMap[K, V]) has(key K) bool {
	rm.lock.Lock()
	defer rm.lock.Unlock()

	_, ok := rm.requests[key]
	return ok
}

// allKeys returns the list of keys that the map has requests associated to.
func (rm *requestMap[K, V]) allKeys() []K {
	rm.lock.Lock()
	defer rm.lock.Unlock()

	keys := make([]K, 0, len(rm.requests))
	for key := range rm.requests {
		keys = append(keys, key)
	}
	return keys
}

// tryDeliver delivers the given result for the request associated with the given
// key if such a request exists in the map. This function should only be called
// with a validated result.
func (rm *requestMap[K, V]) tryDeliver(key K, result V) {
	rm.lock.Lock()
	defer rm.lock.Unlock()

	if r, ok := rm.requests[key]; ok {
		r.deliver(result, nil)
	}
}

// deliver delivers either a result or an error for the request.
func (r *mappedRequest[K, V]) deliver(result V, err error) {
	r.lock.Lock()
	if !r.delivered {
		r.result, r.err = result, err
		r.delivered = true
		close(r.deliveredCh)
		if !r.closed && r.refCount != 0 {
			r.cancelFn()
		}
	}
	r.lock.Unlock()
}

// waitForResult waits for a result or an error to be delivered until the context
// is cancelled.
func (r *mappedRequest[K, V]) waitForResult(ctx context.Context) (V, error) {
	select {
	case <-r.deliveredCh:
		// not changed after deliveredCh is closed
		return r.result, r.err
	case <-ctx.Done():
		var null V
		return null, ctx.Err()
	}
}

// release decreases the reference counter that has been previously increased by
// request(key) and cancels and removes the request if no one is interested in it
// any more.
// Note that waitForResult does not automatically release the request so that the
// caller can store the result in the local cache before making it unavailable in
// the request map.
func (r *mappedRequest[K, V]) release() {
	r.rm.lock.Lock()
	r.lock.Lock()
	r.refCount--
	if r.refCount == 0 && !r.closed {
		if !r.removed {
			delete(r.rm.requests, r.key)
		}
		if !r.delivered {
			r.cancelFn()
		}
	}
	r.lock.Unlock()
	r.rm.lock.Unlock()
}
